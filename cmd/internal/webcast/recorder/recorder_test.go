package recorder

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	rec "github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// fakeChrome is the stateful double of the Caller: the sessions' fake pages,
// one message counter for events and replies (cdp.Conn's Seq), and the
// loader id of each session's current document.
type fakeChrome struct {
	ui     *fakeControlUI
	mu     sync.Mutex
	seq    uint64
	pages  map[string]*fakePage
	loader map[string]string
	calls  []string
	// more answers the methods a test needs beyond these (nil, false: the
	// default empty reply).
	more         func(session, method string, params map[string]any, p *fakePage) (any, error, bool)
	captureProbe func(context.Context, map[string]any) (any, error)
}

func newFakeChrome() *fakeChrome {
	return &fakeChrome{pages: map[string]*fakePage{}, loader: map[string]string{}}
}

func (f *fakeChrome) next() uint64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seq++
	return f.seq
}

func (f *fakeChrome) CallSeq(ctx context.Context, session, method string, params, result any) (uint64, error) {
	f.mu.Lock()
	f.calls = append(f.calls, session+" "+method)
	p := f.pages[session]
	loader := f.loader[session]
	ui := f.ui
	f.mu.Unlock()
	var reply any = map[string]any{}
	var pm map[string]any
	data, _ := json.Marshal(params)
	json.Unmarshal(data, &pm)
	if method == "Runtime.callFunctionOn" && strings.Contains(fmt.Sprint(pm["functionDeclaration"]), "function captureReady(") {
		reply = map[string]any{"result": map[string]any{"value": map[string]any{"viewport": []float64{0, 0, 1000, 2000}, "dpr": 1, "pending": false, "reason": ""}}}
		if f.captureProbe != nil {
			var err error
			reply, err = f.captureProbe(ctx, pm)
			if err != nil {
				return f.next(), err
			}
		}
		b, _ := json.Marshal(reply)
		return f.next(), json.Unmarshal(b, result)
	}

	switch method {
	case "DOM.getDocument":
		if p == nil {
			return 0, fmt.Errorf("no page %s", session)
		}
		reply = map[string]any{"root": p.cdp(p.doc)}
	case "Page.getFrameTree":
		if p == nil {
			return f.next(), fmt.Errorf("no page %s", session)
		}
		reply = map[string]any{"frameTree": map[string]any{"frame": map[string]any{"id": p.frameID, "loaderId": loader}}}
	case "Page.getLayoutMetrics":
		reply = map[string]any{"cssLayoutViewport": map[string]any{"clientWidth": 1000, "clientHeight": 700}}
	default:
		if f.more != nil || ui != nil {
			var pm map[string]any
			b, _ := json.Marshal(params)
			json.Unmarshal(b, &pm)
			if method == "Runtime.callFunctionOn" && pm["functionDeclaration"] == capturedPageProbe {
				if err := ctx.Err(); err != nil {
					return f.next(), err
				}
				if p == nil {
					return f.next(), fmt.Errorf("missing page")
				}
				if p.pageError != nil {
					return f.next(), p.pageError
				}
				valid := false
				if ui != nil {
					ui.mu.Lock()
					w := ui.worlds[session]
					id, _ := pm["executionContextId"].(float64)
					valid = w != nil && w.page == p && w.id == int(id)
					ui.mu.Unlock()
				}
				if !valid {
					return f.next(), fmt.Errorf("expired page context")
				}
				reply = map[string]any{"result": map[string]any{"value": map[string]any{"url": p.url, "title": p.title}}}
				break
			}
			respond := f.more
			if ui != nil {
				respond = func(session, method string, pm map[string]any, p *fakePage) (any, error, bool) {
					if v, e, ok := ui.more(session, method, pm, p); ok {
						return v, e, ok
					}
					if f.more != nil {
						return f.more(session, method, pm, p)
					}
					return nil, nil, false
				}
			}
			if r, err, ok := respond(session, method, pm, p); ok {
				if err != nil {
					return f.next(), err
				}
				reply = r
			}
		}
	}
	seq := f.next()
	if result != nil {
		b, _ := json.Marshal(reply)
		if err := json.Unmarshal(b, result); err != nil {
			return seq, err
		}
	}
	return seq, nil
}

type memOut struct {
	sink   rec.MemSink
	opened int
}

// harness drives a Recorder over a fakeChrome.
type harness struct {
	ui  *fakeControlUI
	t   *testing.T
	fc  *fakeChrome
	r   *Recorder
	out *memOut
}

func newHarness(t *testing.T) *harness {
	h := &harness{t: t, fc: newFakeChrome(), out: &memOut{}}
	h.r = Start(Options{ID: "t", Call: h.fc, ShotEvery: -1,
		Out: func() (Opened, error) { h.out.opened++; return Opened{Sink: &h.out.sink}, nil }})
	t.Cleanup(func() { h.r.Close() })
	return h
}

func (h *harness) page(session, url, loader string) *fakePage {
	p := newFakePage(url)
	p.frameID = "T-" + session
	h.fc.mu.Lock()
	h.fc.pages[session], h.fc.loader[session] = p, loader
	h.fc.mu.Unlock()
	return p
}

// events sends the page's pending CDP events, each with its place in the
// read order.
func (h *harness) events(session string, p *fakePage) {
	for _, e := range p.take() {
		h.r.Send(Event{Session: session, Method: e.method, Params: e.params, Seq: h.fc.next()})
	}
}

// settle waits for the recorder to handle everything sent so far.
func (h *harness) settle() {
	h.t.Helper()
	if _, err := h.r.Stats(context.Background()); err != nil {
		h.t.Fatal(err)
	}
}

func (h *harness) kinds() string {
	var ks []string
	for _, r := range h.out.sink.Records() {
		k := r.Kind()
		if s, ok := r.(*rec.Seg); ok {
			k += ":" + s.Why
		}
		if s, ok := r.(*rec.Stop); ok {
			k += ":" + s.Why
		}
		ks = append(ks, k)
	}
	return strings.Join(ks, " ")
}

func (h *harness) validate() {
	h.t.Helper()
	fs, _ := rec.Validate(files(h.t, h.out.sink.Records()))
	if errs := rec.Errors(fs); len(errs) != 0 {
		h.t.Fatalf("invalid recording: %v\n%s", errs, h.kinds())
	}
	st, err := h.r.Stats(context.Background())
	if err != nil {
		h.t.Fatal(err)
	}
	if len(st.Dropped) != 0 {
		h.t.Fatalf("dropped %v: %v", st.Dropped, st.Examples)
	}
}

// The segment table: go-live, pause, resume on the same document, a
// navigation while paused, a tab switch, end.
func TestRecorderSegments(t *testing.T) {
	h := newHarness(t)
	h.page("A", "http://a/", "L1")
	h.page("B", "http://b/", "L9")
	if h.settle(); h.out.opened != 0 {
		t.Fatal("nothing opens before go-live")
	}
	h.r.Send(Live{Tab: rec.Tab{ID: "T-A"}, Session: "A"})
	h.r.Send(Pause{})
	h.r.Send(Live{Tab: rec.Tab{ID: "T-A"}, Session: "A"}) // same document: resume
	h.r.Send(Pause{})
	h.settle()
	// A navigation while paused: a new document (a new loader; backend ids
	// may even repeat).
	h.fc.mu.Lock()
	h.fc.loader["A"] = "L2"
	h.fc.mu.Unlock()
	h.r.Send(Live{Tab: rec.Tab{ID: "T-A"}, Session: "A"})
	h.r.Send(Live{Tab: rec.Tab{ID: "T-B"}, Session: "B"}) // tab switch while live
	h.settle()
	h.r.Close()
	want := "head seg:live doc stop:pause seg:resume doc stop:pause seg:navigate doc seg:tab doc stop:end"
	if got := h.kinds(); got != want {
		t.Fatalf("records\n got %s\nwant %s", got, want)
	}
	if !h.out.sink.Closed() || h.out.opened != 1 {
		t.Fatalf("closed %v opened %d", h.out.sink.Closed(), h.out.opened)
	}
	h.validate()
	// A resume keeps ids: the second doc names the same nodes as the first.
	var docs []*rec.Doc
	for _, r := range h.out.sink.Records() {
		if d, ok := r.(*rec.Doc); ok {
			docs = append(docs, d)
		}
	}
	if docs[0].Doc != docs[1].Doc || docs[2].Doc == docs[0].Doc {
		t.Fatalf("doc ids %d %d %d", docs[0].Doc, docs[1].Doc, docs[2].Doc)
	}
}

// Events Chrome sent before a snapshot's reply are in the snapshot: applied
// again they would insert a node twice. They are dropped by Seq.
func TestRecorderDropsEventsTheSnapshotIncludes(t *testing.T) {
	h := newHarness(t)
	p := h.page("A", "http://a/", "L1")
	// The page changes, Chrome sends the event (Seq n), then the recorder's
	// getDocument replies (Seq > n) with the change already in.
	p.insert(p.body(), nil, p.el("div"))
	early := p.take()
	seqs := []uint64{}
	for range early {
		seqs = append(seqs, h.fc.next())
	}
	h.r.Send(Live{Tab: rec.Tab{ID: "T-A"}, Session: "A"})
	for i, e := range early {
		h.r.Send(Event{Session: "A", Method: e.method, Params: e.params, Seq: seqs[i]})
	}
	h.settle()
	p.insert(p.body(), nil, p.el("span")) // after the snapshot: applied
	h.events("A", p)
	h.settle()
	h.validate()
	if got := h.kinds(); got != "head seg:live doc ins" {
		t.Fatalf("records %s", got)
	}
}

// A click whose handler navigates is reported synchronously by the
// observer: its record comes before the navigation's segment.
func TestRecorderClickBeforeNavigation(t *testing.T) {
	h := newHarness(t)
	p := h.page("A", "http://a/", "L1")
	btn := p.el("a", "href", "/next")
	p.insert(p.body(), nil, btn)
	p.take()
	h.r.Send(Event{Session: "A", Method: "Runtime.executionContextCreated", Params: raw(map[string]any{
		"context": map[string]any{"id": 7, "name": "couchcast", "auxData": map[string]any{"frameId": p.frameID}}})})
	h.r.Send(Live{Tab: rec.Tab{ID: p.frameID}, Session: "A"})
	payload, _ := json.Marshal(map[string]any{"t": "obs", "items": []any{map[string]any{"k": "click", "p": []int{1, 2, 0}, "x": 5, "y": 6}}})
	h.settle()
	h.r.Send(Event{Session: "A", Method: "Runtime.bindingCalled", Seq: h.fc.next(), Params: raw(map[string]any{
		"name": "__couchcast", "payload": string(payload), "executionContextId": 7})})
	// The navigation: a new loader, then CDP's documentUpdated. No settle:
	// the click is resolved against the mirror, whatever Chrome has now.
	h.fc.mu.Lock()
	h.fc.loader["A"] = "L2"
	h.fc.pages["A"] = newFakePage("http://a/next")
	h.fc.pages["A"].frameID = p.frameID
	h.fc.mu.Unlock()
	h.r.Send(Event{Session: "A", Method: "DOM.documentUpdated", Seq: h.fc.next(), Params: raw(map[string]any{})})
	h.settle()
	h.validate()
	if got := h.kinds(); got != "head seg:live doc click seg:navigate doc" {
		t.Fatalf("records %s", got)
	}
}

// Over the queue bound, events are dropped and the recorder resyncs from a
// fresh snapshot: the recording still ends as the page is.
func TestRecorderOverflowResyncs(t *testing.T) {
	h := &harness{t: t, fc: newFakeChrome(), out: &memOut{}}
	reports := &recorderReports{}
	gate := make(chan struct{})
	slow := &gatedCaller{fakeChrome: h.fc, gate: gate}
	h.r = Start(Options{ID: "t", Call: slow, ShotEvery: -1, MaxQueue: 5,
		Logf: reports.logf, Warnf: reports.warnf,
		Out: func() (Opened, error) { return Opened{Sink: &h.out.sink}, nil }})
	t.Cleanup(func() { h.r.Close() })
	p := h.page("A", "http://a/", "L1")
	close(gate)
	h.r.Send(Live{Tab: rec.Tab{ID: "T-A"}, Session: "A"})
	h.settle()
	slow.hold() // the recorder is busy in a call: events pile up
	h.r.Send(Event{Session: "A", Method: "DOM.inlineStyleInvalidated", Seq: h.fc.next(), Params: raw(map[string]any{"nodeIds": []int{p.body().nodeID}})})
	slow.waitEntered(t)
	for i := 0; i < 20; i++ {
		p.insert(p.body(), nil, p.el("li"))
	}
	h.events("A", p)
	slow.release()
	h.settle()
	h.settle()
	st, err := h.r.Stats(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if st.Lost == 0 || st.Overflow != 0 {
		t.Fatalf("want events lost over the bound, then resynced: %+v", st)
	}
	if !strings.Contains(st.Losses(), "over the queue bound") {
		t.Errorf("losses: %q", st.Losses())
	}
	h.validate()
	s, err := rec.Reduce(h.out.sink.Records(), 1<<62)
	if err != nil {
		t.Fatal(err)
	}
	if err := sameTree(p.doc, s.Top, "doc"); err != nil {
		t.Fatalf("after resync: %v\n%s", err, h.kinds())
	}
	if err := h.r.Close(); err == nil {
		t.Fatal("legacy close lost its resynced overflow report")
	}
	logs, warnings := reports.snapshot()
	if len(logs) != 1 || len(warnings) != 0 {
		t.Fatalf("recovered overflow logs = %v, warnings = %v", logs, warnings)
	}
}

// gatedCaller can hold every call until released, and says when the
// recorder is inside a held call: the injected interleaving (ARCH-ORDER).
type gatedCaller struct {
	*fakeChrome
	mu      sync.Mutex
	gate    chan struct{}
	held    bool
	entered chan struct{}
}

func (g *gatedCaller) hold() {
	g.mu.Lock()
	g.gate, g.held, g.entered = make(chan struct{}), true, make(chan struct{}, 1)
	g.mu.Unlock()
}

func (g *gatedCaller) release() {
	g.mu.Lock()
	close(g.gate)
	g.held = false
	g.mu.Unlock()
}

// waitEntered returns once the recorder is blocked in a held call.
func (g *gatedCaller) waitEntered(t *testing.T) {
	t.Helper()
	select {
	case <-g.entered:
	case <-time.After(3 * time.Second):
		t.Fatal("the recorder never made a call")
	}
}

func (g *gatedCaller) CallSeq(ctx context.Context, session, method string, params, result any) (uint64, error) {
	g.mu.Lock()
	gate, held, entered := g.gate, g.held, g.entered
	g.mu.Unlock()
	if held {
		select {
		case entered <- struct{}{}:
		default:
		}
	}
	<-gate
	return g.fakeChrome.CallSeq(ctx, session, method, params, result)
}

// IDOf and NodeOf answer from the recorder's goroutine.
func TestRecorderIDMapQueries(t *testing.T) {
	h := newHarness(t)
	p := h.page("A", "http://a/", "L1")
	div := p.el("div")
	p.insert(p.body(), nil, div)
	p.take()
	h.r.Send(Live{Tab: rec.Tab{ID: "T-A"}, Session: "A"})
	id, ok, err := h.r.IDOf(context.Background(), "A", div.backend)
	if err != nil || !ok {
		t.Fatal(ok, err)
	}
	s, b, ok, err := h.r.NodeOf(context.Background(), id)
	if err != nil || !ok || s != "A" || b != div.backend {
		t.Fatalf("NodeOf: %s %d %v %v", s, b, ok, err)
	}
	h.r.Close()
	if _, _, err := h.r.IDOf(context.Background(), "A", 1); err == nil {
		t.Fatal("a closed recorder answers nothing")
	}
}

// Records made outside the recorder enter its queue: stamped, checked by
// its reducer, written in order with the page's; refused ones are counted.
func TestRecorderEmit(t *testing.T) {
	h := newHarness(t)
	h.page("A", "http://a/", "L1")
	h.r.Send(Emit{Record: &rec.Note{ID: "early", Author: rec.Author{ID: "v"}, At: rec.NoteAt{Node: 1}}}) // before go-live: dropped
	h.r.Send(Live{Tab: rec.Tab{ID: "T-A"}, Session: "A"})
	h.settle()
	top := h.id("A", h.fc.pages["A"].doc.backend)
	h.r.Send(Emit{Record: &rec.Note{ID: "n1", Author: rec.Author{ID: "v"}, At: rec.NoteAt{Node: top}}})
	h.r.Send(Emit{Record: &rec.NoteState{ID: "nobody", State: rec.NoteResolved}}) // refused: no such note
	h.r.Send(Emit{Record: &rec.Rm{ID: top}})                                      // refused: the recorder's own kind
	h.settle()
	if got := h.kinds(); got != "head seg:live doc note" {
		t.Fatalf("records %s", got)
	}
	st, err := h.r.Stats(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if st.Dropped["note-state"] != 1 || st.Dropped["rm"] != 1 {
		t.Fatalf("stats %+v", st)
	}
}

// A board's record names the document it was made on (#470): written while
// that document is current, dropped (not counted as refused) once it isn't.
func TestRecorderEmitChecksTheDocument(t *testing.T) {
	h := newHarness(t)
	h.page("A", "http://a/", "L1")
	h.r.Send(Live{Tab: rec.Tab{ID: "T-A"}, Session: "A"})
	h.settle()
	top := h.id("A", h.fc.pages["A"].doc.backend)
	h.r.Send(Emit{Record: &rec.Note{ID: "on", Author: rec.Author{ID: "v"}, At: rec.NoteAt{Node: top}}, Doc: 1})
	h.r.Send(Emit{Record: &rec.Note{ID: "gone", Author: rec.Author{ID: "v"}, At: rec.NoteAt{Node: top}}, Doc: 2})
	h.settle()
	if got := h.kinds(); got != "head seg:live doc note" {
		t.Fatalf("records %s", got)
	}
	if n := h.last("note").(*rec.Note); n.ID != "on" {
		t.Fatalf("note %s written", n.ID)
	}
	if st, _ := h.r.Stats(context.Background()); len(st.Dropped) != 0 {
		t.Fatalf("a note for a gone document is not a refusal: %+v", st.Dropped)
	}
}

// A submission's resolution reaches the recorder queue after a viewer has
// resolved and reopened the same note. Check the revision at apply time,
// while a fresh revision can still resolve that reopened note normally.
func TestRecorderSubmissionEmitRechecksQueuedNoteRevision(t *testing.T) {
	h := newHarness(t)
	h.page("A", "http://a/", "L1")
	h.r.Send(Live{Tab: rec.Tab{ID: "T-A"}, Session: "A"})
	h.settle()
	top := h.id("A", h.fc.pages["A"].doc.backend)
	h.r.Send(Emit{Record: &rec.Note{ID: "n1", Author: rec.Author{ID: "v"}, At: rec.NoteAt{Node: top}}, Doc: 1})
	h.settle()
	selected, err := rec.Reduce(h.out.sink.Records(), 1<<62)
	if err != nil {
		t.Fatal(err)
	}
	if len(selected.Notes) != 1 {
		t.Fatalf("notes: %+v", selected.Notes)
	}
	revision := selected.Notes[0].Revision
	// All three writes enter the same queue before the test waits for reduction.
	h.r.Send(Emit{Record: &rec.NoteState{ID: "n1", State: rec.NoteResolved}, Doc: 1})
	h.r.Send(Emit{Record: &rec.NoteState{ID: "n1", State: rec.NoteOpen}, Doc: 1})
	h.r.Send(Emit{Record: &rec.NoteState{ID: "n1", State: rec.NoteResolved, IfRevision: &revision}, Doc: 1})
	h.settle()
	reopened, err := rec.Reduce(h.out.sink.Records(), 1<<62)
	if err != nil {
		t.Fatal(err)
	}
	if len(reopened.Notes) != 1 || reopened.Notes[0].State != rec.NoteOpen || reopened.Notes[0].Revision != revision+2 {
		t.Fatalf("stale queued resolution consumed reopened note: %+v", reopened.Notes)
	}
	if got := h.kinds(); got != "head seg:live doc note note-state note-state" {
		t.Fatalf("unexpected recorded writes: %s", got)
	}
	fresh := reopened.Notes[0].Revision
	h.r.Send(Emit{Record: &rec.NoteState{ID: "n1", State: rec.NoteResolved, IfRevision: &fresh}, Doc: 1})
	h.settle()
	resolved, err := rec.Reduce(h.out.sink.Records(), 1<<62)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Notes[0].State != rec.NoteResolved {
		t.Fatal("matching revision could not resolve note")
	}
}

// A board record that reaches the recorder during a pause is dropped, even on
// the current document (pair#499: viewers do nothing while paused). The board
// refuses requests while paused, but one it admitted just before the pause
// can still be queued after the stop.
func TestRecorderEmitDropsARecordDuringAPause(t *testing.T) {
	h := newHarness(t)
	h.page("A", "http://a/", "L1")
	h.r.Send(Live{Tab: rec.Tab{ID: "T-A"}, Session: "A"})
	h.settle()
	top := h.id("A", h.fc.pages["A"].doc.backend)
	h.r.Send(Pause{})
	h.r.Send(Emit{Record: &rec.Note{ID: "paused", Author: rec.Author{ID: "v"}, At: rec.NoteAt{Node: top}}, Doc: 1})
	h.r.Send(Emit{Record: &rec.Stroke{ID: "v.s", Author: rec.Author{ID: "v"}, Pts: [][3]float64{{1, 2, 0}}}, Doc: 1})
	h.r.Send(Live{Tab: rec.Tab{ID: "T-A"}, Session: "A"}) // resume, same document
	h.settle()
	if got := h.kinds(); strings.Contains(got, "note") || strings.Contains(got, "stroke") {
		t.Fatalf("records from the pause were written: %s", got)
	}
}
