package webcast

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/xianxu/pair/cmd/internal/cdp"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/drive"
	"github.com/xianxu/pair/cmd/internal/webcast/notedraft"
	"github.com/xianxu/pair/cmd/internal/webcast/recorder"
	rec "github.com/xianxu/pair/cmd/internal/webcast/recording"
)

//go:embed inject.js
var controlJS string

var injectJS = buildOperatorScript()

// LaunchOptions starts the cast Chrome.
type LaunchOptions struct {
	Binary string
	// ProfileDir is the cast's own Chrome profile (--user-data-dir), kept
	// across casts so logins to the app under test survive.
	ProfileDir string
	// StartURL is the first page; empty is about:blank.
	StartURL string
	// StartTimeout bounds waiting for Chrome's first page; zero is
	// defaultStartTimeout.
	StartTimeout time.Duration
	// Headless is for tests only: the cast is a window the operator uses.
	Headless bool
	// noPagesAfter lets protocol tests fire page-loss grace after observed events; nil uses time.After.
	noPagesAfter func(time.Duration) <-chan time.Time
	// Record, when set, records the cast tab while live (#458).
	Record *RecordOptions
}

// RecordOptions configure the cast's recorder.
type RecordOptions struct {
	Sketches         *board.SketchService
	SketchAuthority  func(epoch, generation uint64) (board.SketchAuthority, error)
	DrawingAuthority func() drive.DrawingAuthority
	SketchAsset      func(string, int, int) ([]byte, int, error)
	NoteDrafts       *notedraft.Service
	Operator         *board.Board
	ID               string          // the recording's id (its head)
	Out              recorder.Output // opened at the first go-live
	// ShotEvery is the fidelity screenshots' period; zero is the
	// recorder's default.
	ShotEvery time.Duration
	Logf      func(format string, args ...any)
}

const (
	defaultStartTimeout = 20 * time.Second
	// callTimeout bounds each CDP command; a page mid-navigation can leave
	// one unanswered.
	callTimeout = 5 * time.Second
	// stderrTail is how much of Chrome's output is kept to explain a failed
	// launch. In memory only.
	stderrTail = 8 << 10
)

// closeWait is how long Close waits for Chrome to exit before killing it. A
// var so tests can shorten it.
var closeWait = 5 * time.Second

// noPagesGrace: on macOS, closing Chrome's last window leaves it running
// windowless. Once every page has been gone this long, the cast ends and
// Chrome is closed. A var so tests can shorten it.
var noPagesGrace = time.Second

// ErrProfileInUse: another Chrome already runs on the cast profile; Chrome
// hands the launch to it and exits.
var ErrProfileInUse = errors.New("webcast: the cast Chrome profile is already in use by another Chrome")

// Chrome is the real Browser: a Chrome process driven over its debugging
// pipe. One goroutine (run) owns every piece of mutable state; CDP events,
// requests from SetLive and Close, and timers are all inputs to its select,
// so their order is the order they're handled in.
type Chrome struct {
	operatorSketch  *operatorSketchRuntime
	controlRevision atomic.Uint64
	operatorRefresh chan struct{}
	operatorReplies chan operatorDelivery
	opts            LaunchOptions
	proc            process
	conn            *cdp.Conn
	output          *tail
	pub             *publisher[State]

	requests  chan func(*loopState)
	ready     chan struct{} // closed when the first page's control is set up
	readyOnce sync.Once
	done      chan struct{}
	err       error // set before done closes

	// work tracks the loop's off-loop CDP calls, so none outlives the Chrome.
	work sync.WaitGroup
	// ctx is cancelled at the end, failing any call still in flight.
	ctx    context.Context
	cancel context.CancelFunc
	// rec records the cast while live; nil when not recording. The loop
	// forwards every CDP event and the live phase to it; it is the record
	// stream's single writer (#466/#467 add their inputs to its queue).
	rec *recorder.Recorder
	// dialogs: the cast tab's JavaScript dialog, for the recorder's
	// dispatch (#467), read off the loop.
	dialogs dialogSignal
	// activate brings the Chrome app to the front (#501); nil leaves it
	// where it is. Only Launch sets it, for a windowed Chrome, so no test
	// on a headless or scripted Chrome can take the operator's focus.
	activate func() error
}

// Emit is Emitter's: it queues r for the recorder and never blocks.
func (c *Chrome) Emit(r rec.Record, doc int) error {
	if c.rec == nil {
		return ErrNotRecording
	}
	c.rec.Send(recorder.Emit{Record: r, Doc: doc})
	return nil
}

func (c *Chrome) ClearAnnotations(ctx context.Context, generation uint64) error {
	if c.rec == nil {
		return ErrNotRecording
	}
	return c.rec.ClearAnnotations(ctx, generation)
}

func (c *Chrome) ResetControl(ctx context.Context, generation uint64) error {
	if c.rec == nil {
		return ErrNotRecording
	}
	return c.rec.ResetControl(ctx, generation)
}

// endCause is why the loop is closing Chrome, once it is.
type endCause uint8

const (
	notEnding endCause = iota
	// endRequested: Close was called.
	endRequested
	// endLastWindow: every page was gone for noPagesGrace.
	endLastWindow
)

// loopState is owned by run.
type loopState struct {
	sketchScopes  map[operatorContextID]*operatorSketchScope
	sketchLoaders map[string]string
	uiGuards      map[string]string
	tracker       Tracker
	phase         Phase
	sessions      map[string]string // session -> target
	contexts      map[string]int64  // session -> the control's world in its top frame
	frames        map[string]bool   // out-of-process frame sessions
	// recTab is what the recorder was last told is cast ("" when not live)
	// and recTitle its title.
	recTab, recTitle string
	// driver is who drives the cast tab (#467), "" for nobody; dialogs
	// are the page sessions showing a JavaScript dialog.
	driver  string
	dialogs map[string]bool
	// tap sees every event first: the live tests' window on what Chrome
	// said (nil in production).
	tap     func(cdp.Event)
	hadPage bool
	noPages <-chan time.Time
	ending  endCause
	killAt  <-chan time.Time
}

// Launch starts Chrome and returns once its first page is open with its
// control. Cancelling
// ctx abandons the start and closes Chrome.
func Launch(ctx context.Context, opts LaunchOptions) (*Chrome, error) {
	if opts.StartURL == "" {
		opts.StartURL = "about:blank"
	}
	if opts.StartTimeout <= 0 {
		opts.StartTimeout = defaultStartTimeout
	}
	if err := os.MkdirAll(opts.ProfileDir, 0o700); err != nil {
		return nil, fmt.Errorf("webcast: profile dir: %w", err)
	}
	// Chrome reads commands on fd 3 and writes replies and events on fd 4.
	cmdR, cmdW, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	outR, outW, err := os.Pipe()
	if err != nil {
		cmdR.Close()
		cmdW.Close()
		return nil, err
	}
	args := []string{
		"--remote-debugging-pipe",
		"--user-data-dir=" + opts.ProfileDir,
		"--no-first-run",
		"--no-default-browser-check",
	}
	if opts.Headless {
		args = append(args, "--headless=new")
	}
	args = append(args, opts.StartURL)
	output := &tail{max: stderrTail}
	cmd := exec.Command(opts.Binary, args...)
	cmd.ExtraFiles = []*os.File{cmdR, outW}
	cmd.Stdout, cmd.Stderr = output, output
	proc, err := startProcess(cmd, func() { outR.Close() })
	cmdR.Close()
	outW.Close()
	if err != nil {
		cmdW.Close()
		outR.Close()
		return nil, fmt.Errorf("webcast: start Chrome: %w", err)
	}
	c := start(cdp.New(outR, cmdW), proc, output, opts)
	if !opts.Headless {
		c.activate = activateProcess(cmd.Process.Pid)
	}
	timer := time.NewTimer(opts.StartTimeout)
	defer timer.Stop()
	select {
	case <-c.ready:
		return c, nil
	case <-c.done:
		return nil, c.launchError()
	case <-ctx.Done():
		c.Close()
		return nil, ctx.Err()
	case <-timer.C:
		c.Close()
		return nil, fmt.Errorf("webcast: Chrome opened no page within %s%s", opts.StartTimeout, c.output.quote())
	}
}

// start runs the loop over an open connection to a started process.
func start(conn *cdp.Conn, proc process, output *tail, opts LaunchOptions) *Chrome {
	if opts.noPagesAfter == nil {
		opts.noPagesAfter = time.After
	}
	loopCtx, cancel := context.WithCancel(context.Background())
	c := &Chrome{
		opts: opts, proc: proc, conn: conn, output: output,
		operatorRefresh: make(chan struct{}, 1), operatorReplies: make(chan operatorDelivery, 32),
		pub:      newPublisher(State{}),
		requests: make(chan func(*loopState)),
		ready:    make(chan struct{}),
		done:     make(chan struct{}),
		ctx:      loopCtx, cancel: cancel,
	}
	if opts.Record != nil {
		c.rec = recorder.Start(recorder.Options{ID: opts.Record.ID, Call: conn, Out: opts.Record.Out,
			ShotEvery: opts.Record.ShotEvery, Logf: opts.Record.Logf, Dialog: c.dialogs.signal, Operator: opts.Record.Operator, NoteDrafts: opts.Record.NoteDrafts, OperatorReply: c.operatorReply, UIGuard: c.uiGuard, SyncEvents: conn.SyncEvents})
	}
	if opts.Record != nil {
		c.startOperator(opts.Record.Operator)
	}
	go c.run()
	return c
}

// process is the Chrome process as the loop sees it.
type process interface {
	Kill()
	// Exited closes once the process has been reaped.
	Exited() <-chan struct{}
	// ExitCode is valid once Exited has closed.
	ExitCode() int
}

type osProcess struct {
	cmd    *exec.Cmd
	exited chan struct{}
}

// processWaitDelay bounds reaping once the process has exited: a writer
// other than an *os.File makes Wait also wait for every copy of the output
// pipe to close, and Chrome's helpers inherit it and can outlive Chrome.
var processWaitDelay = 2 * time.Second

// startProcess starts cmd and reaps it in the background, running reaped
// once it has been, before Exited closes.
func startProcess(cmd *exec.Cmd, reaped func()) (*osProcess, error) {
	cmd.WaitDelay = processWaitDelay
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	p := &osProcess{cmd: cmd, exited: make(chan struct{})}
	go func() {
		cmd.Wait()
		reaped()
		close(p.exited)
	}()
	return p, nil
}

func (p *osProcess) Kill()                   { p.cmd.Process.Kill() }
func (p *osProcess) Exited() <-chan struct{} { return p.exited }
func (p *osProcess) ExitCode() int           { return p.cmd.ProcessState.ExitCode() }

// launchError explains a Chrome that ended before its first page.
func (c *Chrome) launchError() error {
	if profileInUse(c.output.String()) {
		return fmt.Errorf("%w (%s)", ErrProfileInUse, c.opts.ProfileDir)
	}
	return fmt.Errorf("webcast: Chrome exited before opening a page: %v", c.err)
}

// profileInUse reads Chrome's output for another Chrome holding the profile:
// a windowed Chrome hands the launch over and says so; a headless one fails
// to take the profile's lock (measured, Chrome 154). Other failures to set
// up the lock exit with the same status, so the status alone says nothing.
func profileInUse(output string) bool {
	return strings.Contains(output, "Opening in existing browser session") ||
		strings.Contains(output, "SingletonLock: File exists")
}

func (c *Chrome) State() State                      { return c.pub.get() }
func (c *Chrome) Subscribe() (<-chan State, func()) { return c.pub.subscribe() }
func (c *Chrome) Done() <-chan struct{}             { return c.done }

func (c *Chrome) Err() error {
	select {
	case <-c.done:
		return c.err
	default:
		return nil
	}
}

// do runs fn on the loop. False once the Chrome has ended.
func (c *Chrome) do(fn func(*loopState)) bool {
	select {
	case c.requests <- fn:
		return true
	case <-c.done:
		return false
	}
}

func (c *Chrome) SetLive(live bool) error {
	var err error
	ran := make(chan struct{})
	if !c.do(func(s *loopState) { err = c.setLive(s, live); close(ran) }) {
		return ErrEnded
	}
	<-ran
	return err
}

// Focus activates the cast tab, which raises its window within Chrome, then
// brings Chrome itself in front of other apps.
func (c *Chrome) Focus() error {
	var tab string
	ran := make(chan struct{})
	if !c.do(func(s *loopState) { tab = castTab(s); close(ran) }) {
		return ErrEnded
	}
	<-ran
	if tab != "" {
		if err := c.call("", "Target.activateTarget", map[string]any{"targetId": tab}, nil); err != nil {
			return err
		}
	}
	if c.activate != nil {
		return c.activate()
	}
	return nil
}

func (c *Chrome) Close() error {
	c.do(func(s *loopState) { c.beginClose(s, endRequested) })
	<-c.done
	return nil
}

func (c *Chrome) run() {
	s := &loopState{phase: PhaseIdle, sessions: map[string]string{}, contexts: map[string]int64{}, frames: map[string]bool{}, dialogs: map[string]bool{}}
	events := c.conn.Events()
	// Every off-loop call starts on this goroutine, before finish waits for
	// them all.
	// Discovery reports every page's URL and lifecycle; auto-attach (pages
	// only) gives each page, existing or new, a session, holding new ones
	// until goSetup releases them.
	c.goCall("", "Target.setDiscoverTargets", map[string]any{"discover": true}, nil)
	c.goCall("", "Target.setAutoAttach", map[string]any{
		"autoAttach": true, "waitForDebuggerOnStart": true, "flatten": true,
		"filter": []map[string]any{{"type": "page", "exclude": false}, {"exclude": true}},
	}, nil)
loop:
	for {
		select {
		case ev, ok := <-events:
			if !ok {
				// The pipe is gone: Chrome exited, or the conn failed.
				break loop
			}
			c.handle(s, ev)
		case fn := <-c.requests:
			fn(s)
		case <-s.noPages:
			s.noPages = nil
			if s.tracker.Pages() == 0 {
				c.beginClose(s, endLastWindow)
			}
		case <-s.killAt:
			s.killAt = nil
			c.proc.Kill()
		case <-c.proc.Exited():
			break loop
		}
		c.publish(s)
	}
	c.finish(s)
}

func (c *Chrome) handle(s *loopState, ev cdp.Event) {
	if ev.Barrier != nil {
		if c.rec != nil {
			c.rec.Send(recorder.Barrier{Done: ev.Barrier})
		} else {
			close(ev.Barrier)
		}
		return
	}
	if s.tap != nil {
		s.tap(ev)
	}
	if c.handleSketchBinding(s, ev) {
		return
	}
	if c.operatorSketch != nil && (ev.Method == "Page.frameNavigated" || ev.Method == "Page.frameDetached") {
		var p struct {
			Frame   struct{ ID, LoaderID string }
			FrameID string `json:"frameId"`
		}
		if json.Unmarshal(ev.Params, &p) == nil {
			id := p.Frame.ID
			if id == "" {
				id = p.FrameID
			}
			c.operatorSketch.RetireFrame(id)
			if id != "" && id == s.sessions[ev.SessionID] {
				c.retireOperatorSketchScopes(s, ev.SessionID)
				if s.sketchLoaders == nil {
					s.sketchLoaders = map[string]string{}
				}
				s.sketchLoaders[ev.SessionID] = p.Frame.LoaderID
			}
		}
	}
	if c.rec != nil {
		c.rec.Send(recorder.Event{Session: ev.SessionID, Method: ev.Method, Params: ev.Params, Seq: ev.Seq})
	}
	targetOf := func(session string) (string, bool) {
		t, ok := s.sessions[session]
		return t, ok
	}
	for _, in := range translate(ev.SessionID, ev.Method, ev.Params, targetOf) {
		switch in := in.(type) {
		case TargetCreated:
			if c.closeOpened(s, in) {
				continue
			}
			s.tracker.Apply(in)
			s.hadPage, s.noPages = true, nil
		case TargetDestroyed:
			s.tracker.Apply(in)
			if s.hadPage && s.tracker.Pages() == 0 && s.noPages == nil {
				s.noPages = c.opts.noPagesAfter(noPagesGrace)
			}
		case TrackerInput:
			s.tracker.Apply(in)
		case pageAttached:
			s.sessions[in.SessionID] = in.TargetID
			c.goSetup(in.SessionID, s.driver != "")
		case frameAttached:
			s.frames[in.SessionID] = true
			c.goSetupFrame(in)
		case pageDetached:
			c.retireOperatorSketchScopes(s, in.SessionID)
			delete(s.sketchLoaders, in.SessionID)
			delete(s.sessions, in.SessionID)
			delete(s.uiGuards, in.SessionID)
			delete(s.contexts, in.SessionID)
			delete(s.dialogs, in.SessionID)
			delete(s.frames, in.SessionID)
			if c.rec != nil {
				c.rec.Send(recorder.Detached{Session: in.SessionID})
			}
		case contextCreated:
			if s.contexts[in.SessionID] != in.ContextID {
				c.retireOperatorSketchScopes(s, in.SessionID)
			}
			s.contexts[in.SessionID] = in.ContextID
		case contextDestroyed:
			if s.contexts[in.SessionID] == in.ContextID {
				c.retireOperatorSketchScopes(s, in.SessionID)
				delete(s.contexts, in.SessionID)
			}
		case contextsCleared:
			c.retireOperatorSketchScopes(s, in.SessionID)
			delete(s.contexts, in.SessionID)
		case toggleRequested:
			c.setLive(s, s.phase != PhaseLive)
		case helloRequested:
			c.goSetControl(in.SessionID, in.ContextID, s.phase == PhaseLive, s.driver, s.uiGuards[in.SessionID] != "")
			c.refreshOperator()
		case dialogOpened:
			s.dialogs[in.SessionID] = true
		case dialogClosed:
			delete(s.dialogs, in.SessionID)
		}
	}
}

// goSetup prepares a newly attached page. Auto-attach holds a new page
// before its first document runs, so the control is in place for it; the
// page is released at the end whatever happened, since a page left waiting
// is a frozen tab. The binding comes before the script, so the script's
// first report has somewhere to go.
func (c *Chrome) goSetup(session string, driving bool) {
	c.work.Go(func() {
		defer c.call(session, "Runtime.runIfWaitingForDebugger", nil, nil)
		type step struct {
			method string
			params any
		}
		var extra []step
		if driving {
			// A page opened while a viewer drives shows no file chooser.
			extra = append(extra, step{"Page.setInterceptFileChooserDialog", map[string]any{"enabled": true}})
		}
		for _, step := range append([]step{
			{"Runtime.addBinding", map[string]any{"name": bindingName, "executionContextName": worldName}},
			{"Runtime.enable", nil},
			// Without the Page domain, the script runs in the current
			// document only, never in the next one (measured, Chrome 154).
			{"Page.enable", nil},
			{"Page.addScriptToEvaluateOnNewDocument", map[string]any{"source": injectJS, "worldName": worldName, "runImmediately": true}},
			{"Page.addScriptToEvaluateOnNewDocument", map[string]any{"source": recorder.ObserverJS, "worldName": worldName, "runImmediately": true}},
			// Out-of-process iframes get sessions of their own, held until
			// goSetupFrame releases them.
			{"Target.setAutoAttach", frameAutoAttach},
		}, extra...) {
			if c.call(session, step.method, step.params, nil) != nil {
				// The page closed mid-setup; nothing else depends on it.
				return
			}
		}
		c.readyOnce.Do(func() { close(c.ready) })
	})
}

// frameAutoAttach attaches a session's out-of-process iframes, flattened.
var frameAutoAttach = map[string]any{
	"autoAttach": true, "waitForDebuggerOnStart": true, "flatten": true,
	"filter": []map[string]any{{"type": "iframe", "exclude": false}, {"exclude": true}},
}

// goSetupFrame prepares an out-of-process iframe's session as goSetup does a
// page's, with the recorder's observer and no control, and tells the
// recorder. Released at the end whatever happened.
func (c *Chrome) goSetupFrame(f frameAttached) {
	c.work.Go(func() {
		defer c.call(f.SessionID, "Runtime.runIfWaitingForDebugger", nil, nil)
		if c.rec != nil {
			allowed, err := c.rec.AdmitRecordingFrame(c.ctx, f.TargetID, f.SessionID, f.Parent)
			c.rec.Send(recorder.FrameAttached{Session: f.SessionID, Parent: f.Parent, FrameID: f.TargetID})
			if err != nil || !allowed {
				c.call(f.SessionID, "Target.setAutoAttach", frameAutoAttach, nil)
				return
			}
		}
		for _, step := range []struct {
			method string
			params any
		}{
			{"Runtime.addBinding", map[string]any{"name": bindingName, "executionContextName": worldName}},
			{"Runtime.enable", nil},
			{"Page.enable", nil},
			{"Page.addScriptToEvaluateOnNewDocument", map[string]any{"source": recorder.ObserverJS, "worldName": worldName, "runImmediately": true}},
			{"Target.setAutoAttach", frameAutoAttach},
		} {
			if c.call(f.SessionID, step.method, step.params, nil) != nil {
				return
			}
		}
	})
}

// goSetControl shows live or idle on one page's control. A failure means the
// document went away; the next one asks with hello.
func (c *Chrome) goSetControl(session string, contextID int64, live bool, driver string, guard bool) {
	name, _ := json.Marshal(driver)
	expr := fmt.Sprintf("globalThis.__couchcastSet && globalThis.__couchcastSet(%t, %s, %t, %d)", live, name, guard, c.controlRevision.Add(1))
	c.goCall(session, "Runtime.evaluate", map[string]any{"expression": expr, "contextId": contextID}, nil)
}

func (c *Chrome) setLive(s *loopState, live bool) error {
	if s.ending != notEnding {
		return ErrEnded
	}
	s.phase = PhaseIdle
	if live {
		s.phase = PhaseLive
	}
	for session, ctx := range s.contexts {
		c.goSetControl(session, ctx, live, s.driver, s.uiGuards[session] != "")
	}
	return nil
}

// beginClose asks Chrome to close, and kills it if it hasn't after closeWait.
// The first cause wins.
func (c *Chrome) beginClose(s *loopState, cause endCause) {
	if s.ending != notEnding {
		return
	}
	s.ending = cause
	s.killAt = time.After(closeWait)
	c.goCall("", "Browser.close", nil, nil)
}

func (c *Chrome) goCall(session, method string, params, result any) {
	c.work.Go(func() { c.call(session, method, params, result) })
}

func (c *Chrome) call(session, method string, params, result any) error {
	ctx, cancel := context.WithTimeout(c.ctx, callTimeout)
	defer cancel()
	return c.conn.Call(ctx, session, method, params, result)
}

func (c *Chrome) publish(s *loopState) {
	if s.ending != notEnding {
		return
	}
	tab, ok := s.tracker.Active()
	c.tellRecorder(s, tab, ok)
	// The cast tab's dialog, for a driver's dispatch: recomputed after
	// every input, so a dialog, a tab switch or a closed tab all count.
	c.dialogs.set(s.dialogs[sessionOf(s, castTab(s))])
	next := State{Phase: s.phase, Tab: tab, HasTab: ok}
	for _, guard := range s.uiGuards {
		if guard == "unknown" {
			next.Recovery = "Remote input completion is uncertain. Close and reopen the cast Chrome to restore Couch UI."
			break
		}
	}
	if next != c.pub.get() {
		c.pub.publish(next)
	}
}

// tellRecorder keeps the recorder on the cast tab: live on the active tab,
// a switch when it changes, its title, pause.
func (c *Chrome) tellRecorder(s *loopState, tab Tab, ok bool) {
	if c.rec == nil {
		return
	}
	if s.phase == PhaseLive && !ok {
		// No page for a moment: a new one becomes the cast tab (seg tab),
		// or Chrome ends after noPagesGrace (stop end). Not a pause.
		return
	}
	cast := ""
	if s.phase == PhaseLive {
		cast = tab.ID
	}
	if cast != s.recTab {
		if cast == "" {
			c.rec.Send(recorder.Pause{})
		} else if session := sessionOf(s, cast); session != "" {
			c.rec.Send(recorder.Live{Tab: rec.Tab{ID: tab.ID, URL: tab.URL, Title: tab.Title}, Session: session})
		} else {
			return // its session isn't attached yet; the next publish tells it
		}
		s.recTab, s.recTitle = cast, tab.Title
	}
	if cast != "" && tab.Title != s.recTitle {
		s.recTitle = tab.Title
		c.rec.Send(recorder.Title{Title: tab.Title})
	}
}

func sessionOf(s *loopState, target string) string {
	for session, t := range s.sessions {
		if t == target {
			return session
		}
	}
	return ""
}

// finish runs once the pipe is gone or Chrome exited: it makes sure the
// process is reaped, every call has returned, and says why it ended.
func (c *Chrome) finish(s *loopState) {
	c.retireOperatorSketchScopes(s, "")
	c.cancel() // unblock bounded bridge/guard callbacks before joining the recorder
	// Read before our own Close, which would mask it.
	connErr := c.conn.Err()
	if c.rec != nil {
		// Before the pipe closes: the recording ends with stop end.
		if err := c.rec.Close(); err != nil && c.opts.Record.Logf != nil {
			c.opts.Record.Logf("%v", err)
		}
	}
	c.conn.Close()
	select {
	case <-c.proc.Exited():
	case <-time.After(closeWait):
		c.proc.Kill()
		<-c.proc.Exited()
	}
	c.cancel()
	c.work.Wait()
	c.err = endError(s.ending, connErr, c.proc.ExitCode(), c.output.quote())
	c.pub.finish(State{Phase: PhaseEnded})
	close(c.done)
}

// endError says why a Chrome ended. A cause the loop chose wins; then a
// protocol failure (Chrome sent something unreadable); then a failing exit
// status; a clean exit nobody asked for is the operator quitting Chrome.
// Only what Chrome sent counts as a connection failure: the pipe closing,
// however it surfaces (EOF, a closed reader, a write after exit), is what
// every ending looks like, including our own teardown.
func endError(cause endCause, connErr error, exitCode int, output string) error {
	switch cause {
	case endRequested:
		return nil
	case endLastWindow:
		return ErrClosedByOperator
	}
	if errors.Is(connErr, cdp.ErrProtocol) {
		return fmt.Errorf("webcast: lost Chrome: %w", connErr)
	}
	if exitCode != 0 {
		return fmt.Errorf("webcast: Chrome exited with status %d%s", exitCode, output)
	}
	return ErrClosedByOperator
}

// tail keeps the last max bytes written to it.
type tail struct {
	mu  sync.Mutex
	max int
	buf []byte
}

func (t *tail) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.buf = append(t.buf, p...)
	if over := len(t.buf) - t.max; over > 0 {
		t.buf = append(t.buf[:0], t.buf[over:]...)
	}
	return len(p), nil
}

func (t *tail) String() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return string(t.buf)
}

// quote is the output's last lines, for an error message; empty when none.
func (t *tail) quote() string {
	out := strings.TrimSpace(t.String())
	if out == "" {
		return ""
	}
	lines := strings.Split(out, "\n")
	if len(lines) > 5 {
		lines = lines[len(lines)-5:]
	}
	return ":\n  " + strings.Join(lines, "\n  ")
}

var _ Browser = (*Chrome)(nil)
