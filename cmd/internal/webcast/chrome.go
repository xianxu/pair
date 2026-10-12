package webcast

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/cdp"
)

//go:embed inject.js
var injectJS string

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
}

const (
	defaultStartTimeout = 20 * time.Second
	// callTimeout bounds each CDP command; a page mid-navigation can leave
	// one unanswered.
	callTimeout = 5 * time.Second
	// closeWait is how long Close waits for Chrome to exit before killing it.
	closeWait = 5 * time.Second
	// stderrTail is how much of Chrome's output is kept to explain a failed
	// launch. In memory only.
	stderrTail = 8 << 10
)

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
	opts   LaunchOptions
	cmd    *exec.Cmd
	conn   *cdp.Conn
	output *tail
	pub    *publisher[State]

	requests  chan func(*loopState)
	exited    chan struct{} // closed when the process has been reaped
	ready     chan struct{} // closed when the first page's control is set up
	readyOnce sync.Once
	done      chan struct{}
	err       error // set before done closes

	// work tracks the loop's off-loop CDP calls, so none outlives the Chrome.
	work sync.WaitGroup
	// ctx is cancelled at the end, failing any call still in flight.
	ctx    context.Context
	cancel context.CancelFunc
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
	tracker  Tracker
	phase    Phase
	sessions map[string]string // session -> target
	contexts map[string]int64  // session -> the control's world in its top frame
	hadPage  bool
	noPages  <-chan time.Time
	ending   endCause
	killAt   <-chan time.Time
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
	err = cmd.Start()
	cmdR.Close()
	outW.Close()
	if err != nil {
		cmdW.Close()
		outR.Close()
		return nil, fmt.Errorf("webcast: start Chrome: %w", err)
	}
	loopCtx, cancel := context.WithCancel(context.Background())
	c := &Chrome{
		opts: opts, cmd: cmd, conn: cdp.New(outR, cmdW), output: output,
		pub:      newPublisher(State{}),
		requests: make(chan func(*loopState)),
		exited:   make(chan struct{}),
		ready:    make(chan struct{}),
		done:     make(chan struct{}),
		ctx:      loopCtx, cancel: cancel,
	}
	go func() {
		cmd.Wait()
		outR.Close()
		close(c.exited)
	}()
	go c.run()

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

// profileInUseExit is Chrome's exit status when another Chrome holds the
// profile's lock (RESULT_CODE_PROFILE_IN_USE).
const profileInUseExit = 21

// launchError explains a Chrome that ended before its first page. A windowed
// Chrome hands a launch on a profile in use to the running one and exits; a
// headless one exits with profileInUseExit.
func (c *Chrome) launchError() error {
	out := c.output.String()
	if strings.Contains(out, "Opening in existing browser session") || c.cmd.ProcessState.ExitCode() == profileInUseExit {
		return fmt.Errorf("%w (%s)", ErrProfileInUse, c.opts.ProfileDir)
	}
	return fmt.Errorf("webcast: Chrome exited before opening a page: %v%s", c.err, c.output.quote())
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
	if !c.do(func(s *loopState) { c.setLive(s, live) }) {
		return ErrEnded
	}
	return nil
}

func (c *Chrome) Close() error {
	c.do(func(s *loopState) { c.beginClose(s, endRequested) })
	<-c.done
	return nil
}

func (c *Chrome) run() {
	s := &loopState{phase: PhaseIdle, sessions: map[string]string{}, contexts: map[string]int64{}}
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
			c.cmd.Process.Kill()
		case <-c.exited:
			break loop
		}
		c.publish(s)
	}
	c.finish(s)
}

func (c *Chrome) handle(s *loopState, ev cdp.Event) {
	targetOf := func(session string) (string, bool) {
		t, ok := s.sessions[session]
		return t, ok
	}
	for _, in := range translate(ev.SessionID, ev.Method, ev.Params, targetOf) {
		switch in := in.(type) {
		case TargetCreated:
			s.tracker.Apply(in)
			s.hadPage, s.noPages = true, nil
		case TargetDestroyed:
			s.tracker.Apply(in)
			if s.hadPage && s.tracker.Pages() == 0 && s.noPages == nil {
				s.noPages = time.After(noPagesGrace)
			}
		case TrackerInput:
			s.tracker.Apply(in)
		case pageAttached:
			s.sessions[in.SessionID] = in.TargetID
			c.goSetup(in.SessionID)
		case pageDetached:
			delete(s.sessions, in.SessionID)
			delete(s.contexts, in.SessionID)
		case contextCreated:
			s.contexts[in.SessionID] = in.ContextID
		case contextDestroyed:
			if s.contexts[in.SessionID] == in.ContextID {
				delete(s.contexts, in.SessionID)
			}
		case contextsCleared:
			delete(s.contexts, in.SessionID)
		case toggleRequested:
			c.setLive(s, s.phase != PhaseLive)
		case helloRequested:
			c.goSetControl(in.SessionID, in.ContextID, s.phase == PhaseLive)
		}
	}
}

// goSetup prepares a newly attached page. Auto-attach holds a new page
// before its first document runs, so the control is in place for it; the
// page is released at the end whatever happened, since a page left waiting
// is a frozen tab. The binding comes before the script, so the script's
// first report has somewhere to go.
func (c *Chrome) goSetup(session string) {
	c.work.Go(func() {
		defer c.call(session, "Runtime.runIfWaitingForDebugger", nil, nil)
		for _, step := range []struct {
			method string
			params any
		}{
			{"Runtime.addBinding", map[string]any{"name": bindingName, "executionContextName": worldName}},
			{"Runtime.enable", nil},
			// Without the Page domain, the script runs in the current
			// document only, never in the next one (measured, Chrome 154).
			{"Page.enable", nil},
			{"Page.addScriptToEvaluateOnNewDocument", map[string]any{"source": injectJS, "worldName": worldName, "runImmediately": true}},
		} {
			if c.call(session, step.method, step.params, nil) != nil {
				// The page closed mid-setup; nothing else depends on it.
				return
			}
		}
		c.readyOnce.Do(func() { close(c.ready) })
	})
}

// goSetControl shows live or idle on one page's control. A failure means the
// document went away; the next one asks with hello.
func (c *Chrome) goSetControl(session string, contextID int64, live bool) {
	expr := fmt.Sprintf("globalThis.__couchcastSet && globalThis.__couchcastSet(%t)", live)
	c.goCall(session, "Runtime.evaluate", map[string]any{"expression": expr, "contextId": contextID}, nil)
}

func (c *Chrome) setLive(s *loopState, live bool) {
	if s.ending != notEnding {
		return
	}
	s.phase = PhaseIdle
	if live {
		s.phase = PhaseLive
	}
	for session, ctx := range s.contexts {
		c.goSetControl(session, ctx, live)
	}
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
	next := State{Phase: s.phase, Tab: tab, HasTab: ok}
	if next != c.pub.get() {
		c.pub.publish(next)
	}
}

// finish runs once the pipe is gone or Chrome exited: it makes sure the
// process is reaped, every call has returned, and says why it ended.
func (c *Chrome) finish(s *loopState) {
	c.conn.Close()
	select {
	case <-c.exited:
	case <-time.After(closeWait):
		c.cmd.Process.Kill()
		<-c.exited
	}
	c.cancel()
	c.work.Wait()
	switch s.ending {
	case endRequested:
		c.err = nil
	case endLastWindow:
		c.err = ErrClosedByOperator
	default:
		if code := c.cmd.ProcessState.ExitCode(); code != 0 {
			c.err = fmt.Errorf("webcast: Chrome exited with status %d%s", code, c.output.quote())
		} else {
			c.err = ErrClosedByOperator
		}
	}
	c.pub.finish(State{Phase: PhaseEnded})
	close(c.done)
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
