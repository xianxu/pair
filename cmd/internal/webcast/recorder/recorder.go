// Package recorder is couch broadcast-web's recorder (#458): it records the
// cast tab of a Chrome into the format of atlas/webcast-format.md, from CDP's
// DOM, CSS, Network and Page domains plus a small observer in the couchcast
// isolated world of every frame.
//
// It lives on the CDP side of webcast's Browser seam: the real Browser
// (webcast.Chrome) owns the sessions and forwards their events and the live
// phase here; nothing CDP crosses the Browser interface.
//
// The pure core is the mirror (mirror.go, observe.go). The Recorder is the
// thin shell around it: one goroutine owns the mirror, takes inputs from a
// FIFO queue in arrival order, makes its CDP calls in that order, and is the
// single writer of the record stream. Asset bodies and screenshots are
// fetched by two side workers whose results return as inputs.
package recorder

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/webcast/assets"
	rec "github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// ObserverJS is the observer; the Browser installs it in the couchcast world
// of every frame, with the __couchcast binding.
//
//go:embed observer.js
var ObserverJS string

// Caller makes a CDP call on a session and says where its reply fell in
// the read order (cdp.Conn.CallSeq).
type Caller interface {
	CallSeq(ctx context.Context, session, method string, params, result any) (uint64, error)
}

// FrameWriter keeps a screenshot and its frame record: a recording's
// DirWriter. Never a Live: pixels never reach viewers.
type FrameWriter interface {
	PutFrame(f *rec.Frame, png []byte) error
}

// Output opens where a recording goes, at the first go-live, so a cast never
// live leaves nothing behind. The recorder closes the sink it opened when it
// ends. frames may be nil (no screenshots).
type Output func() (sink rec.Sink, frames FrameWriter, err error)

// Options configures a Recorder.
type Options struct {
	ID     string // the head's id
	Call   Caller
	Out    Output
	Assets assets.Store // nil: no asset records
	// ShotEvery is the screenshot period while live (and one at each
	// segment); zero is 5 s, negative is none.
	ShotEvery time.Duration
	// MaxQueue bounds queued CDP events; over it they are dropped and a
	// resync re-snapshots. Zero is 100 000.
	MaxQueue int
	Now      func() time.Time
	Logf     func(format string, args ...any)
}

// Inputs. The Browser sends them in the order they happened.
type (
	// Event is a CDP event of any session, as read (Seq from cdp.Event).
	Event struct {
		Session, Method string
		Params          json.RawMessage
		Seq             uint64
	}
	// Live goes live on a tab, or switches the cast to another tab while
	// live. Session is the tab's page session.
	Live struct {
		Tab     rec.Tab
		Session string
	}
	// Pause pauses the cast.
	Pause struct{}
	// Title is the cast tab's title, as its document reported it.
	Title struct{ Title string }
	// FrameAttached: an out-of-process frame's session, under Parent's.
	FrameAttached struct{ Session, Parent, FrameID string }
	// Detached: a session went away.
	Detached struct{ Session string }
)

// side-worker results and queries, internal inputs.
type (
	assetDone struct{ r *rec.Asset }
	shotDone  struct {
		at     time.Time
		png    []byte
		w, h   int
		scale  float64
		docs   []layoutDoc
		failed error
	}
	query struct {
		fn   func()
		done chan struct{}
	}
	stamped struct {
		in any
		at time.Time
	}
)

const (
	defaultShotEvery = 5 * time.Second
	defaultMaxQueue  = 100000
	callTimeout      = 5 * time.Second
	sideQueue        = 256
	shotScale        = 0.5
)

// Recorder records one cast. Start it, Send it inputs, Close it.
type Recorder struct {
	opts Options

	mu       sync.Mutex
	queue    []stamped
	events   int
	overflow int // events dropped since the last resync
	signal   chan struct{}
	closing  bool
	done     chan struct{}

	ctx    context.Context
	cancel context.CancelFunc
	side   sync.WaitGroup
	assetQ chan assetJob
	shotQ  chan shotJob

	// Owned by the loop.
	m        *mirror
	sink     rec.Sink
	frames   FrameWriter
	t0       time.Time
	opened   bool
	live     bool
	seg      int
	top      string // the cast page's session
	tab      rec.Tab
	parents  map[string]string // frame session -> parent session
	frameIDs map[string]string // session -> its main frame id
	loader   map[string]string // session -> its main frame's loader id
	contexts map[string]map[int64]string
	snapSeq  map[string]uint64 // session -> reply number of its last snapshot
	resync   bool
	headers  map[string]sheetHeader
	requests map[string]netReq
	shotN    int
	nextShot time.Time
	err      error
}

type sheetHeader struct {
	Session     string
	ID          string `json:"styleSheetId"`
	FrameID     string `json:"frameId"`
	OwnerNode   int    `json:"ownerNode"`
	Constructed bool   `json:"isConstructed"`
}

type netReq struct{ typ, url, mime string }

// Start runs a recorder; it records nothing until the first Live.
func Start(opts Options) *Recorder {
	if opts.ShotEvery == 0 {
		opts.ShotEvery = defaultShotEvery
	}
	if opts.MaxQueue <= 0 {
		opts.MaxQueue = defaultMaxQueue
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.Logf == nil {
		opts.Logf = func(string, ...any) {}
	}
	ctx, cancel := context.WithCancel(context.Background())
	r := &Recorder{
		opts: opts, signal: make(chan struct{}, 1), done: make(chan struct{}), ctx: ctx, cancel: cancel,
		assetQ: make(chan assetJob, sideQueue), shotQ: make(chan shotJob, 1),
		m: newMirror(), parents: map[string]string{}, frameIDs: map[string]string{}, loader: map[string]string{},
		contexts: map[string]map[int64]string{}, snapSeq: map[string]uint64{}, headers: map[string]sheetHeader{},
		requests: map[string]netReq{},
	}
	r.side.Add(2)
	go r.assetWorker()
	go r.shotWorker()
	go r.run()
	return r
}

// Send queues an input; it never blocks. CDP events past MaxQueue are
// dropped and counted; the recorder resyncs from a fresh snapshot, which
// includes them. Other inputs are never dropped.
func (r *Recorder) Send(in any) {
	at := r.opts.Now()
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closing {
		return
	}
	if _, ok := in.(Event); ok {
		if r.events >= r.opts.MaxQueue {
			r.overflow++
			return
		}
		r.events++
	}
	r.queue = append(r.queue, stamped{in, at})
	select {
	case r.signal <- struct{}{}:
	default:
	}
}

// Close ends the recording (a live segment gets stop end), closes the sink,
// and returns once the recorder's goroutines are done. Idempotent.
func (r *Recorder) Close() error {
	r.mu.Lock()
	if !r.closing {
		r.closing = true
		select {
		case r.signal <- struct{}{}:
		default:
		}
	}
	r.mu.Unlock()
	<-r.done
	return r.err
}

// Stats is what the recorder lost, for the operator's report.
type Stats struct {
	Dropped  map[string]int // records the reducer refused, by kind
	Desync   int            // events and reports naming nodes the mirror lacked
	Overflow int            // events dropped over MaxQueue
	Examples []string
}

// Stats is answered on the recorder's goroutine.
func (r *Recorder) Stats(ctx context.Context) (Stats, error) {
	var s Stats
	err := r.ask(ctx, func() {
		s = Stats{Dropped: map[string]int{}, Desync: r.m.Desync, Examples: append([]string(nil), r.m.Examples...)}
		for k, v := range r.m.Dropped {
			s.Dropped[k] = v
		}
		r.mu.Lock()
		s.Overflow = r.overflow
		r.mu.Unlock()
	})
	return s, err
}

// IDOf is the recording id of a session's node by backendNodeId (#467's
// injector), answered on the recorder's goroutine.
func (r *Recorder) IDOf(ctx context.Context, session string, backend int) (rec.NodeID, bool, error) {
	var id rec.NodeID
	var ok bool
	err := r.ask(ctx, func() { id, ok = r.m.IDOf(session, backend) })
	return id, ok, err
}

// NodeOf is the session and backendNodeId of a live recording id.
func (r *Recorder) NodeOf(ctx context.Context, id rec.NodeID) (session string, backend int, ok bool, err error) {
	err = r.ask(ctx, func() { session, backend, ok = r.m.NodeOf(id) })
	return
}

var errClosed = errors.New("recorder: closed")

func (r *Recorder) ask(ctx context.Context, fn func()) error {
	q := query{fn: fn, done: make(chan struct{})}
	r.mu.Lock()
	if r.closing {
		r.mu.Unlock()
		return errClosed
	}
	r.queue = append(r.queue, stamped{q, r.opts.Now()})
	select {
	case r.signal <- struct{}{}:
	default:
	}
	r.mu.Unlock()
	select {
	case <-q.done:
		return nil
	case <-r.done:
		return errClosed
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *Recorder) run() {
	defer close(r.done)
	tick := time.NewTicker(250 * time.Millisecond)
	defer tick.Stop()
	for {
		r.mu.Lock()
		if len(r.queue) == 0 {
			closing := r.closing
			r.mu.Unlock()
			if closing {
				break
			}
			select {
			case <-r.signal:
			case <-tick.C:
				r.maybeShot(r.opts.Now())
			}
			continue
		}
		in := r.queue[0]
		r.queue[0] = stamped{}
		r.queue = r.queue[1:]
		if _, ok := in.in.(Event); ok {
			r.events--
		}
		resync := r.overflow > 0 && !r.resync
		r.mu.Unlock()
		if resync {
			r.resync = true
		}
		r.handle(in)
		r.afterInput(in.at)
	}
	if r.live {
		r.stop(r.opts.Now(), rec.StopEnd)
		r.write()
	}
	r.cancel()
	close(r.assetQ)
	close(r.shotQ)
	r.side.Wait()
	// Results that came back after the end are dropped: the recording is
	// closed.
	if r.sink != nil {
		r.err = errors.Join(r.err, r.sink.Close())
	}
}

// t is ms since the recording began.
func (r *Recorder) t(at time.Time) int64 {
	if !r.opened {
		return 0
	}
	return at.Sub(r.t0).Milliseconds()
}

func (r *Recorder) call(session, method string, params, result any) (uint64, error) {
	ctx, cancel := context.WithTimeout(r.ctx, callTimeout)
	defer cancel()
	return r.opts.Call.CallSeq(ctx, session, method, params, result)
}

func (r *Recorder) handle(in stamped) {
	switch v := in.in.(type) {
	case query:
		v.fn()
		close(v.done)
	case Live:
		r.goLive(in.at, v)
	case Pause:
		if r.live {
			r.stop(in.at, rec.StopPause)
		}
	case Title:
		if r.live && v.Title != r.m.st.Title {
			r.m.Emit(r.t(in.at), &rec.Title{V: v.Title})
		}
	case FrameAttached:
		r.parents[v.Session] = v.Parent
		r.frameIDs[v.Session] = v.FrameID
		r.m.Session(v.Session, v.FrameID, false)
		if r.live && r.casts(v.Session) {
			r.enable(v.Session)
			r.frameSnapshot(in.at, v.Session)
			r.observe(v.Session, true)
		}
	case Detached:
		delete(r.parents, v.Session)
		delete(r.contexts, v.Session)
		delete(r.snapSeq, v.Session)
		if v.Session != r.top {
			r.m.Drop(v.Session)
		}
	case Event:
		r.event(in.at, v)
	case assetDone:
		if r.live {
			r.m.Emit(r.t(in.at), v.r)
		}
	case shotDone:
		r.shotDone(v)
	}
}

// afterInput fetches what the mirror asked for, resyncs when events were
// lost, and writes what the mirror emitted.
func (r *Recorder) afterInput(at time.Time) {
	if r.live && r.resync {
		r.resync = false
		r.mu.Lock()
		lost := r.overflow
		r.overflow = 0
		r.mu.Unlock()
		if lost > 0 {
			r.opts.Logf("recorder: %d events over the queue bound; resyncing", lost)
		}
		r.resnapshot(at, r.top)
		for s := range r.parents {
			if r.casts(s) {
				r.frameSnapshot(at, s)
			}
		}
	}
	for len(r.m.Expand) > 0 {
		e := r.m.Expand[0]
		r.m.Expand = r.m.Expand[1:]
		r.call(e.Session, "DOM.requestChildNodes", map[string]any{"nodeId": e.NodeID, "depth": -1, "pierce": true}, nil)
	}
	for len(r.m.Truncated) > 0 {
		f := r.m.Truncated[0]
		r.m.Truncated = r.m.Truncated[1:]
		if text, ok := r.readText(f.Session, f.Backend); ok {
			r.m.FullText(r.t(at), f.Session, f.Backend, text)
		}
	}
	r.write()
}

func (r *Recorder) write() {
	for _, x := range r.m.take() {
		if r.sink == nil {
			continue
		}
		if err := r.sink.Put(x); err != nil && r.err == nil {
			r.err = err
			r.opts.Logf("recorder: write: %v", err)
		}
	}
}

// casts reports whether a session is the cast page's or a frame under it.
func (r *Recorder) casts(session string) bool {
	for s, n := session, 0; s != "" && n < 32; s, n = r.parents[s], n+1 {
		if s == r.top {
			return true
		}
	}
	return false
}

// castSessions is the cast page's session and its frames', the page first.
func (r *Recorder) castSessions() []string {
	out := []string{r.top}
	var frames []string
	for s := range r.parents {
		if r.casts(s) {
			frames = append(frames, s)
		}
	}
	sort.Strings(frames)
	return append(out, frames...)
}

func (r *Recorder) goLive(at time.Time, l Live) {
	if !r.opened {
		sink, frames, err := r.opts.Out()
		if err != nil {
			r.err = err
			r.opts.Logf("recorder: open: %v", err)
			return
		}
		r.sink, r.frames, r.t0, r.opened = sink, frames, at, true
		r.m.Emit(0, &rec.Head{Format: rec.Format, V: rec.Version, ID: r.opts.ID, Started: at.Format(time.RFC3339Nano)})
	}
	if r.live && l.Session == r.top {
		return
	}
	why := rec.SegLive
	switch {
	case r.seg == 0:
	case l.Session != r.top:
		why = rec.SegTab
	default:
		why = rec.SegNavigate // decided below: resume when it is the same document
	}
	if r.live {
		r.quiet() // switching tabs: the old page stops reporting
	}
	r.top, r.tab = l.Session, l.Tab
	r.frameIDs[r.top] = l.Tab.ID
	r.m.Session(r.top, l.Tab.ID, true)
	r.live = true
	r.enable(r.top)
	r.snapshotTop(at, why)
	for _, s := range r.castSessions()[1:] {
		r.enable(s)
		r.frameSnapshot(at, s)
	}
	for _, s := range r.castSessions() {
		r.observe(s, true)
	}
	r.goAssets()
	r.nextShot = time.Time{}
	r.maybeShot(at)
}

// snapshotTop takes the cast page's document: a new segment (why, or resume
// when the document is the one that was paused).
func (r *Recorder) snapshotTop(at time.Time, why string) {
	root, key, err := r.document(r.top)
	if err != nil {
		r.opts.Logf("recorder: snapshot: %v", err)
		r.live = false
		return
	}
	if why == rec.SegNavigate && r.m.SameDocument(r.top, key) && !r.m.st.Live {
		why = rec.SegResume
	}
	r.seg++
	r.tab.URL = root.DocumentURL
	seg := &rec.Seg{Seg: r.seg, Why: why, Tab: r.tab, VP: r.viewport()}
	if err := r.m.Seg(r.t(at), seg, r.top, root, key); err != nil {
		r.opts.Logf("%v", err)
	}
}

// document snapshots a session's document and names it.
func (r *Recorder) document(session string) (*cnode, string, error) {
	var tree struct {
		FrameTree struct {
			Frame struct {
				ID       string `json:"id"`
				LoaderID string `json:"loaderId"`
			} `json:"frame"`
		} `json:"frameTree"`
	}
	if _, err := r.call(session, "Page.getFrameTree", nil, &tree); err == nil {
		r.loader[session] = tree.FrameTree.Frame.LoaderID
	}
	var res struct {
		Root *cnode `json:"root"`
	}
	seq, err := r.call(session, "DOM.getDocument", map[string]any{"depth": -1, "pierce": true}, &res)
	if err != nil {
		return nil, "", err
	}
	if res.Root == nil {
		return nil, "", errors.New("recorder: getDocument returned no root")
	}
	r.snapSeq[session] = seq
	return res.Root, fmt.Sprintf("%s:%d", r.loader[session], res.Root.BackendNodeID), nil
}

func (r *Recorder) resnapshot(at time.Time, session string) {
	root, key, err := r.document(session)
	if err != nil {
		return
	}
	if !r.m.SameDocument(session, key) {
		// A new document: the top one is a new segment.
		r.seg++
		r.tab.URL = root.DocumentURL
		seg := &rec.Seg{Seg: r.seg, Why: rec.SegNavigate, Tab: r.tab, VP: r.viewport()}
		if err := r.m.Seg(r.t(at), seg, session, root, key); err != nil {
			r.opts.Logf("%v", err)
		}
		r.observe(session, true)
		r.goAssets()
		r.nextShot = time.Time{}
		return
	}
	r.m.Resnapshot(r.t(at), session, root)
}

func (r *Recorder) frameSnapshot(at time.Time, session string) {
	root, key, err := r.document(session)
	if err != nil {
		return
	}
	r.m.FrameDoc(r.t(at), session, root, key)
}

func (r *Recorder) viewport() rec.Viewport {
	vp := rec.Viewport{W: 1280, H: 800, DPR: 1}
	var lm struct {
		CSS struct {
			W float64 `json:"clientWidth"`
			H float64 `json:"clientHeight"`
		} `json:"cssLayoutViewport"`
	}
	if _, err := r.call(r.top, "Page.getLayoutMetrics", nil, &lm); err == nil && lm.CSS.W > 0 {
		vp.W, vp.H = int(lm.CSS.W), int(lm.CSS.H)
	}
	if ctx, ok := r.topContext(r.top); ok {
		var res struct {
			Result struct {
				Value float64 `json:"value"`
			} `json:"result"`
		}
		if _, err := r.call(r.top, "Runtime.evaluate", map[string]any{"expression": "devicePixelRatio", "contextId": ctx, "returnByValue": true}, &res); err == nil && res.Result.Value > 0 {
			vp.DPR = res.Result.Value
		}
	}
	return vp
}

// topContext is the couchcast world of a session's main frame.
func (r *Recorder) topContext(session string) (int64, bool) {
	for id, frame := range r.contexts[session] {
		if frame == r.frameIDs[session] {
			return id, true
		}
	}
	return 0, false
}

// enable turns on the domains the recording reads.
func (r *Recorder) enable(session string) {
	r.call(session, "DOM.enable", map[string]any{"includeWhitespace": "all"}, nil)
	r.call(session, "CSS.enable", nil, nil)
	r.call(session, "Network.enable", map[string]any{"maxTotalBufferSize": 64 << 20, "maxResourceBufferSize": 16 << 20}, nil)
}

func (r *Recorder) disable(session string) {
	for _, m := range []string{"Network.disable", "CSS.disable", "DOM.disable"} {
		r.call(session, m, nil, nil)
	}
}

// observe switches the observer of every frame of a session on (with a
// resync) or off.
func (r *Recorder) observe(session string, on bool) {
	expr := fmt.Sprintf("globalThis.__couchcastRec && globalThis.__couchcastRec(%t, true)", on)
	for ctx := range r.contexts[session] {
		r.call(session, "Runtime.evaluate", map[string]any{"expression": expr, "contextId": ctx}, nil)
	}
}

// quiet stops the cast page and its frames from reporting.
func (r *Recorder) quiet() {
	for _, s := range r.castSessions() {
		r.observe(s, false)
		r.disable(s)
	}
}

func (r *Recorder) stop(at time.Time, why string) {
	r.quiet()
	r.live = false
	r.m.Emit(r.t(at), &rec.Stop{Why: why})
}

func (r *Recorder) event(at time.Time, e Event) {
	switch e.Method {
	case "Runtime.executionContextCreated":
		var p struct {
			Context struct {
				ID      int64  `json:"id"`
				Name    string `json:"name"`
				AuxData struct {
					FrameID string `json:"frameId"`
				} `json:"auxData"`
			} `json:"context"`
		}
		if json.Unmarshal(e.Params, &p) == nil && p.Context.Name == worldName {
			if r.contexts[e.Session] == nil {
				r.contexts[e.Session] = map[int64]string{}
			}
			r.contexts[e.Session][p.Context.ID] = p.Context.AuxData.FrameID
		}
		return
	case "Runtime.executionContextDestroyed":
		var p struct {
			ID int64 `json:"executionContextId"`
		}
		if json.Unmarshal(e.Params, &p) == nil {
			delete(r.contexts[e.Session], p.ID)
		}
		return
	case "Runtime.executionContextsCleared":
		delete(r.contexts, e.Session)
		return
	case "Page.frameNavigated":
		var p struct {
			Frame struct {
				ID       string `json:"id"`
				ParentID string `json:"parentId"`
				LoaderID string `json:"loaderId"`
			} `json:"frame"`
		}
		if json.Unmarshal(e.Params, &p) == nil && p.Frame.ID == r.frameIDs[e.Session] {
			r.loader[e.Session] = p.Frame.LoaderID
		}
		return
	}
	if !r.live || !r.casts(e.Session) {
		return
	}
	t := r.t(at)
	if strings.HasPrefix(e.Method, "DOM.") && e.Seq != 0 && e.Seq < r.snapSeq[e.Session] {
		return // sent before the snapshot, which includes it
	}
	switch e.Method {
	case "DOM.documentUpdated":
		if e.Session == r.top {
			r.resnapshot(at, e.Session)
		} else {
			r.frameSnapshot(at, e.Session)
		}
	case "DOM.inlineStyleInvalidated":
		var p struct {
			IDs []int `json:"nodeIds"`
		}
		json.Unmarshal(e.Params, &p)
		for _, id := range p.IDs {
			var a struct {
				Attributes []string `json:"attributes"`
			}
			if _, err := r.call(e.Session, "DOM.getAttributes", map[string]any{"nodeId": id}, &a); err != nil {
				continue
			}
			var v *string
			for i := 0; i+1 < len(a.Attributes); i += 2 {
				if a.Attributes[i] == "style" {
					v = &a.Attributes[i+1]
				}
			}
			if r.m.InlineStyle(t, e.Session, id, v) == errDesync {
				r.resync = true
			}
		}
	case "CSS.styleSheetAdded":
		var p struct {
			Header sheetHeader `json:"header"`
		}
		if json.Unmarshal(e.Params, &p) != nil {
			return
		}
		h := p.Header
		h.Session = e.Session
		r.headers[h.ID] = h
		if h.Constructed {
			r.sheetText(t, h)
		}
	case "CSS.styleSheetChanged":
		var p struct {
			ID string `json:"styleSheetId"`
		}
		json.Unmarshal(e.Params, &p)
		h, ok := r.headers[p.ID]
		switch {
		case !ok:
		case h.Constructed:
			r.sheetText(t, h)
		case h.OwnerNode != 0:
			r.ownedRules(t, h)
		}
	case "CSS.styleSheetRemoved":
		var p struct {
			ID string `json:"styleSheetId"`
		}
		json.Unmarshal(e.Params, &p)
		delete(r.headers, p.ID)
	case "CSS.fontsUpdated":
		var p struct {
			Font *struct {
				Family  string `json:"fontFamily"`
				Style   string `json:"fontStyle"`
				Weight  string `json:"fontWeight"`
				Stretch string `json:"fontStretch"`
				Range   string `json:"unicodeRange"`
				Src     string `json:"src"`
			} `json:"font"`
		}
		if json.Unmarshal(e.Params, &p) == nil && p.Font != nil && p.Font.Src != "" && p.Font.Family != "" {
			f := p.Font
			r.m.Emit(t, &rec.Font{Family: f.Family, Src: f.Src, Style: f.Style, Weight: f.Weight, Stretch: f.Stretch, Range: f.Range})
		}
	case "Network.responseReceived":
		var p struct {
			ID       string `json:"requestId"`
			Type     string `json:"type"`
			Response struct {
				URL      string `json:"url"`
				MIMEType string `json:"mimeType"`
			} `json:"response"`
		}
		if json.Unmarshal(e.Params, &p) == nil && assetType(p.Type) {
			r.requests[e.Session+"|"+p.ID] = netReq{p.Type, p.Response.URL, p.Response.MIMEType}
		}
	case "Network.loadingFinished", "Network.loadingFailed":
		var p struct {
			ID string `json:"requestId"`
		}
		json.Unmarshal(e.Params, &p)
		key := e.Session + "|" + p.ID
		if q, ok := r.requests[key]; ok && e.Method == "Network.loadingFinished" {
			r.queueAsset(assetJob{session: e.Session, requestID: p.ID, url: q.url, mime: q.mime})
		}
		delete(r.requests, key)
	case "Page.navigatedWithinDocument":
		var p struct {
			FrameID string `json:"frameId"`
			URL     string `json:"url"`
		}
		if json.Unmarshal(e.Params, &p) == nil && e.Session == r.top && p.FrameID == r.frameIDs[r.top] && p.URL != r.m.st.URL {
			r.m.Emit(t, &rec.Nav{URL: p.URL})
		}
	case "Runtime.bindingCalled":
		r.binding(at, e)
	default:
		if strings.HasPrefix(e.Method, "DOM.") {
			if r.m.Event(t, e.Session, e.Method, e.Params) == errDesync {
				r.resync = true
			}
		}
	}
}

func assetType(t string) bool { return t == "Image" || t == "Stylesheet" || t == "Font" }

func (r *Recorder) sheetText(t int64, h sheetHeader) {
	var res struct {
		Text string `json:"text"`
	}
	if _, err := r.call(h.Session, "CSS.getStyleSheetText", map[string]any{"styleSheetId": h.ID}, &res); err == nil {
		r.m.ConstructedSheet(t, h.Session, h.ID, res.Text)
	}
}

// ownedRules reads a style or link element's CSSOM rules in the couchcast
// world (its rules after insertRule; CDP's own text is the source).
func (r *Recorder) ownedRules(t int64, h sheetHeader) {
	v, ok := r.inWorld(h.Session, h.FrameID, h.OwnerNode,
		"function(){ try { return this.sheet ? Array.from(this.sheet.cssRules, r => r.cssText).join('\\n') : null } catch (e) { return null } }")
	if ok && v != nil {
		r.m.OwnedSheet(t, h.Session, h.OwnerNode, *v)
	}
}

// readText reads a text node's full value in the couchcast world.
func (r *Recorder) readText(session string, backend int) (string, bool) {
	for _, frame := range r.sortedFrames(session) {
		if v, ok := r.inWorld(session, frame, backend, "function(){ return this.data }"); ok && v != nil {
			return *v, true
		}
	}
	return "", false
}

func (r *Recorder) sortedFrames(session string) []string {
	var out []string
	main := r.frameIDs[session]
	seen := map[string]bool{}
	if main != "" {
		out, seen[main] = append(out, main), true
	}
	for _, f := range r.contexts[session] {
		if !seen[f] {
			out, seen[f] = append(out, f), true
		}
	}
	return out
}

// inWorld runs fn on a node in the couchcast world of a frame, returning a
// string result.
func (r *Recorder) inWorld(session, frameID string, backend int, fn string) (*string, bool) {
	var ctx int64
	for id, f := range r.contexts[session] {
		if f == frameID || frameID == "" {
			ctx = id
			break
		}
	}
	if ctx == 0 {
		return nil, false
	}
	var obj struct {
		Object struct {
			ID string `json:"objectId"`
		} `json:"object"`
	}
	if _, err := r.call(session, "DOM.resolveNode", map[string]any{"backendNodeId": backend, "executionContextId": ctx, "objectGroup": "couchcast-rec"}, &obj); err != nil {
		return nil, false
	}
	defer r.call(session, "Runtime.releaseObjectGroup", map[string]any{"objectGroup": "couchcast-rec"}, nil)
	var res struct {
		Result struct {
			Value *string `json:"value"`
		} `json:"result"`
	}
	if _, err := r.call(session, "Runtime.callFunctionOn", map[string]any{"objectId": obj.Object.ID, "functionDeclaration": fn, "returnByValue": true}, &res); err != nil {
		return nil, false
	}
	return res.Result.Value, true
}

// binding handles an observer report.
func (r *Recorder) binding(at time.Time, e Event) {
	var p struct {
		Name    string `json:"name"`
		Payload string `json:"payload"`
		Context int64  `json:"executionContextId"`
	}
	if json.Unmarshal(e.Params, &p) != nil || p.Name != bindingName {
		return
	}
	var msg struct {
		T     string `json:"t"`
		Items []Item `json:"items"`
	}
	if json.Unmarshal([]byte(p.Payload), &msg) != nil || msg.T != "obs" {
		return
	}
	frame := r.contexts[e.Session][p.Context]
	t := r.t(at)
	for _, it := range msg.Items {
		if it.K == "hello" {
			// A new document's observer: switch it on.
			r.call(e.Session, "Runtime.evaluate", map[string]any{"expression": "globalThis.__couchcastRec(true, true)", "contextId": p.Context}, nil)
			continue
		}
		off := r.m.TopScroll()
		if (it.K == "click" || it.K == "ptr") && !(e.Session == r.top && frame == r.frameIDs[r.top]) {
			o, ok := r.frameOffset(e.Session, frame, 0)
			if !ok {
				continue
			}
			off = o
		}
		if r.m.Observe(t, e.Session, frame, it, off) == errStale && it.K != "click" {
			// A report racing a change is normal; a click is worth a note,
			// which the mirror has made. Neither needs a resync.
			continue
		}
	}
}

// frameOffset maps a frame's viewport origin into top-page coordinates: its
// owner iframe's content box in the parent frame's viewport, plus that
// frame's offset, up to the top.
func (r *Recorder) frameOffset(session, frameID string, depth int) ([2]float64, bool) {
	if depth > 8 {
		return [2]float64{}, false
	}
	if session == r.top && frameID == r.frameIDs[r.top] {
		return r.m.TopScroll(), true
	}
	ownerSession, ownerFrame := session, r.frameIDs[session]
	if frameID == r.frameIDs[session] {
		// An out-of-process frame's document: its owner is in the parent.
		ownerSession = r.parents[session]
		ownerFrame = r.frameIDs[ownerSession]
		if ownerSession == "" {
			return [2]float64{}, false
		}
	}
	var fo struct {
		Backend int `json:"backendNodeId"`
	}
	if _, err := r.call(ownerSession, "DOM.getFrameOwner", map[string]any{"frameId": frameID}, &fo); err != nil {
		return [2]float64{}, false
	}
	var bm struct {
		Model struct {
			Content []float64 `json:"content"`
		} `json:"model"`
	}
	if _, err := r.call(ownerSession, "DOM.getBoxModel", map[string]any{"backendNodeId": fo.Backend}, &bm); err != nil || len(bm.Model.Content) < 2 {
		return [2]float64{}, false
	}
	up, ok := r.frameOffset(ownerSession, ownerFrame, depth+1)
	if !ok {
		return [2]float64{}, false
	}
	return [2]float64{bm.Model.Content[0] + up[0], bm.Model.Content[1] + up[1]}, true
}

// The isolated world the observer and the in-page control share, and their
// binding (webcast's names; the Browser installs both).
const (
	worldName   = "couchcast"
	bindingName = "__couchcast"
)
