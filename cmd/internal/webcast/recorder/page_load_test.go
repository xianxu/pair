package recorder

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/notes"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	rec "github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func TestRecorderTopNavigationStartsSnapshot(t *testing.T) {
	h := newHarness(t)
	p := h.page("A", "http://same/", "L1")
	h.r.Send(Live{Tab: rec.Tab{ID: "T-A"}, Session: "A"})
	h.settle()
	h.fc.mu.Lock()
	h.fc.loader["A"] = "L2"
	h.fc.mu.Unlock()
	params, err := json.Marshal(map[string]any{"frame": map[string]any{"id": p.frameID, "loaderId": "L2", "url": "http://same/"}})
	if err != nil {
		t.Fatal(err)
	}
	h.r.Send(Event{Session: "A", Method: "Page.frameNavigated", Params: params, Seq: h.fc.next()})
	h.settle()
	segments := 0
	for _, r := range h.out.sink.Records() {
		if _, ok := r.(*rec.Seg); ok {
			segments++
		}
	}
	if segments != 2 {
		t.Fatalf("top navigation alone recorded %d segments, want 2 without waiting for a DOM edit", segments)
	}
	h.validate()
}

func TestRecorderSegmentsPersistPageLoad(t *testing.T) {
	h := newHarness(t)
	h.page("A", "http://same/", "L1")
	h.r.Send(Live{Tab: rec.Tab{ID: "T-A"}, Session: "A"})
	h.settle()
	for _, r := range h.out.sink.Records() {
		if _, ok := r.(*rec.Seg); !ok {
			continue
		}
		raw, err := rec.Encode(r)
		if err != nil {
			t.Fatal(err)
		}
		var wire struct {
			PageLoad *struct {
				Target string `json:"target"`
				Loader string `json:"loader"`
				Root   string `json:"root"`
			} `json:"page_load"`
		}
		if err := json.Unmarshal(raw, &wire); err != nil {
			t.Fatal(err)
		}
		if wire.PageLoad == nil || wire.PageLoad.Target != "T-A" || wire.PageLoad.Loader != "L1" || wire.PageLoad.Root == "" {
			t.Fatalf("accepted segment lacks exact string-root page load: %s", raw)
		}
		return
	}
	t.Fatal("no accepted segment")
}

// A gate holds a real stateful CDP reply, with cancellation and explicit join.
// Capturing the reply before waiting exposes stale snapshot evidence rather
// than accidentally returning the replacement fake page after release.
type loadSnapshotGate struct {
	*fakeChrome
	mu          sync.Mutex
	method      string
	entered     chan struct{}
	release     chan struct{}
	cancelled   chan struct{}
	faultMethod string
	fault       error
	emptyRoot   bool
}

func (g *loadSnapshotGate) hold(method string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.method = method
	g.entered = make(chan struct{})
	g.release = make(chan struct{})
	g.cancelled = make(chan struct{})
}
func (g *loadSnapshotGate) CallSeq(ctx context.Context, session, method string, params, result any) (uint64, error) {
	seq, err := g.fakeChrome.CallSeq(ctx, session, method, params, result)
	g.mu.Lock()
	fault, emptyRoot := error(nil), false
	if g.faultMethod == method {
		fault, emptyRoot = g.fault, g.emptyRoot
		g.faultMethod = ""
	}
	held := g.method == method
	if held {
		g.method = ""
	}
	entered, release, cancelled := g.entered, g.release, g.cancelled
	g.mu.Unlock()
	if fault != nil {
		return seq, fault
	}
	if emptyRoot {
		if decodeErr := json.Unmarshal([]byte(`{"root":null}`), result); decodeErr != nil {
			return seq, decodeErr
		}
	}
	if !held {
		return seq, err
	}
	close(entered)
	select {
	case <-release:
		return seq, err
	case <-ctx.Done():
		close(cancelled)
		return seq, ctx.Err()
	}
}
func waitLoadSignal(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(3 * time.Second):
		t.Fatal("load gate was not reached")
	}
}
func loadHarness(t *testing.T) (*harness, *board.Board, *loadSnapshotGate) {
	h := &harness{t: t, fc: newFakeChrome(), out: &memOut{}}
	b := board.New(notes.Structural{}, nil)
	g := &loadSnapshotGate{fakeChrome: h.fc}
	h.r = Start(Options{ID: "load", Call: g, ShotEvery: -1, Operator: b, Out: func() (Opened, error) { return Opened{Sink: rec.Tee(&h.out.sink, b)}, nil }})
	t.Cleanup(func() { h.r.Close() })
	h.page("A", "http://same/", "L1")
	h.r.Send(Live{Tab: rec.Tab{ID: "T-A"}, Session: "A"})
	h.settle()
	return h, b, g
}
func loadNavigate(h *harness, loader string) {
	h.fc.mu.Lock()
	h.fc.loader["A"] = loader
	h.fc.mu.Unlock()
	h.r.Send(Event{Session: "A", Method: "Page.frameNavigated", Seq: h.fc.next(), Params: raw(map[string]any{"frame": map[string]any{"id": "T-A", "loaderId": loader}})})
}
func TestRecorderPageLoadPendingBeforeSnapshotIO(t *testing.T) {
	h, b, g := loadHarness(t)
	if !b.PageLoadReady() {
		t.Fatal("accepted initial snapshot is not ready")
	}
	g.hold("DOM.getDocument")
	loadNavigate(h, "L2")
	waitLoadSignal(t, g.entered)
	if b.PageLoadReady() {
		t.Fatal("old page remains admitted while replacement snapshot IO is held")
	}
	close(g.release)
	h.settle()
	if !b.PageLoadReady() {
		t.Fatal("matching accepted replacement did not become ready")
	}
}
func TestRecorderPageLoadSupersededReplyHasNoEffects(t *testing.T) {
	h, b, g := loadHarness(t)
	g.hold("DOM.getDocument")
	loadNavigate(h, "L2")
	waitLoadSignal(t, g.entered)
	loadNavigate(h, "L3")
	waitLoadSignal(t, g.cancelled)
	h.settle()
	var loaders []string
	for _, r := range h.out.sink.Records() {
		if s, ok := r.(*rec.Seg); ok && s.PageLoad != nil {
			loaders = append(loaders, s.PageLoad.Loader)
		}
	}
	if !reflect.DeepEqual(loaders, []string{"L1", "L3"}) {
		t.Fatalf("superseded unreserved snapshot changed history: %v", loaders)
	}
	if !b.PageLoadReady() {
		t.Fatal("superseded completion prevented successor confirmation")
	}
}

type loadSinkGate struct {
	sink             rec.Sink
	mu               sync.Mutex
	hold             bool
	entered, release chan struct{}
}

func (g *loadSinkGate) Put(r rec.Record) error {
	g.mu.Lock()
	held := g.hold && r.Kind() == "doc"
	if held {
		g.hold = false
	}
	entered, release := g.entered, g.release
	g.mu.Unlock()
	if held {
		close(entered)
		<-release
	}
	return g.sink.Put(r)
}
func (g *loadSinkGate) Close() error { return g.sink.Close() }
func TestRecorderPageLoadNavigationDuringReservedSinkWrite(t *testing.T) {
	h := &harness{t: t, fc: newFakeChrome(), out: &memOut{}}
	b := board.New(notes.Structural{}, nil)
	sink := &loadSinkGate{sink: rec.Tee(&h.out.sink, b), entered: make(chan struct{}), release: make(chan struct{})}
	h.r = Start(Options{ID: "load", Call: h.fc, ShotEvery: -1, Operator: b, Out: func() (Opened, error) { return Opened{Sink: sink}, nil }})
	var released sync.Once
	t.Cleanup(func() { released.Do(func() { close(sink.release) }); h.r.Close() })
	h.page("A", "http://same/", "L1")
	h.r.Send(Live{Tab: rec.Tab{ID: "T-A"}, Session: "A"})
	h.settle()
	sink.mu.Lock()
	sink.hold = true
	sink.mu.Unlock()
	loadNavigate(h, "L2")
	waitLoadSignal(t, sink.entered)
	sent := make(chan struct{})
	go func() { loadNavigate(h, "L3"); close(sent) }()
	waitLoadSignal(t, sent)
	if b.PageLoadReady() {
		t.Fatal("held old historical commit blocked ingress or restored stale page authority")
	}
	released.Do(func() { close(sink.release) })
	h.settle()
	var loaders []string
	for _, r := range h.out.sink.Records() {
		if s, ok := r.(*rec.Seg); ok && s.PageLoad != nil {
			loaders = append(loaders, s.PageLoad.Loader)
		}
	}
	if !reflect.DeepEqual(loaders, []string{"L1", "L2", "L3"}) {
		t.Fatalf("reserved history or successor lost: %v", loaders)
	}
	if !b.PageLoadReady() {
		t.Fatal("new generation did not become ready")
	}
}

func TestRecorderPageLoadPreparedDispatchChecksPendingScope(t *testing.T) {
	h, b, a, n := preparedFixture(t)
	b.BeginPageLoad(900)
	settled := make(chan board.OperatorOutcome, 1)
	if err := h.r.WritePrepared(context.Background(), a, n, nil, func(out board.OperatorOutcome, _ error) { settled <- out }); err != nil {
		t.Fatal(err)
	}
	if out := <-settled; out != board.OperatorRefused {
		t.Fatalf("fresh note dispatched while scope pending: %s", out)
	}
}
func TestRecorderPageLoadInitialRetryPrecedesQueuedPause(t *testing.T) {
	h := &harness{t: t, fc: newFakeChrome(), out: &memOut{}}
	g := &loadSnapshotGate{fakeChrome: h.fc}
	g.hold("DOM.getDocument")
	h.r = Start(Options{ID: "load", Call: g, ShotEvery: -1, Out: func() (Opened, error) { return Opened{Sink: &h.out.sink}, nil }})
	t.Cleanup(func() { h.r.Close() })
	h.page("A", "http://same/", "L1")
	h.r.Send(Live{Tab: rec.Tab{ID: "T-A"}, Session: "A"})
	waitLoadSignal(t, g.entered)
	h.r.Send(Pause{})
	loadNavigate(h, "L2")
	waitLoadSignal(t, g.cancelled)
	h.settle()
	if got := h.kinds(); got != "head seg:live doc stop:pause" {
		t.Fatalf("initial intent lost FIFO activation: %s", got)
	}
	for _, r := range h.out.sink.Records() {
		if seg, ok := r.(*rec.Seg); ok && seg.PageLoad.Loader != "L2" {
			t.Fatalf("cancelled old observation accepted: %+v", seg.PageLoad)
		}
	}
}

func TestFakeChromeDetachedSessionRefusesAllMethods(t *testing.T) {
	f := newFakeChrome()
	f.pages["gone"] = newFakePage("http://old/")
	f.loader["gone"] = "old"
	if err := f.detach("gone"); err != nil {
		t.Fatal(err)
	}
	f.more = func(string, string, map[string]any, *fakePage) (any, error, bool) {
		t.Fatal("detached session reached successful fallback")
		return nil, nil, false
	}
	for _, method := range []string{"DOM.getDocument", "Page.getFrameTree", "Page.enable", "Page.getLayoutMetrics", "Runtime.callFunctionOn", "CSS.getStyleSheetText", "Network.getResponseBody"} {
		if _, err := f.CallSeq(context.Background(), "gone", method, nil, &map[string]any{}); err == nil {
			t.Fatalf("detached %s incorrectly succeeded", method)
		}
	}
	if err := f.detach("gone"); err == nil {
		t.Fatal("duplicate detach was not detected")
	}
}

func TestRecorderPageLoadPendingDetachFailsCaptureAndAllowsNextTarget(t *testing.T) {
	h := &harness{t: t, fc: newFakeChrome(), out: &memOut{}}
	g := &loadSnapshotGate{fakeChrome: h.fc}
	g.hold("DOM.getDocument")
	diagnostics := make(chan string, 8)
	h.r = Start(Options{ID: "load", Call: g, ShotEvery: -1, Logf: func(format string, args ...any) { diagnostics <- fmt.Sprintf(format, args...) }, Out: func() (Opened, error) { return Opened{Sink: &h.out.sink}, nil }})
	t.Cleanup(func() { h.r.Close() })
	h.page("A", "http://old/", "L1")
	h.page("B", "http://next/", "L9")
	h.r.Send(Live{Tab: rec.Tab{ID: "T-A"}, Session: "A"})
	waitLoadSignal(t, g.entered)
	h.detach("A")
	waitLoadSignal(t, g.cancelled)
	h.r.Send(Live{Tab: rec.Tab{ID: "T-B"}, Session: "B"})
	h.settle()
	select {
	case log := <-diagnostics:
		if !strings.Contains(log, "no snapshot of the cast tab") {
			t.Fatalf("missing failed snapshot diagnostic: %s", log)
		}
	default:
		t.Fatal("detached capture disappeared without diagnostic")
	}
	if got := h.kinds(); got != "head seg:live doc" {
		t.Fatalf("detached unreserved load fabricated history: %s", got)
	}
	for _, r := range h.out.sink.Records() {
		if s, ok := r.(*rec.Seg); ok && s.Tab.ID != "T-B" {
			t.Fatalf("detached A was recorded: %+v", s)
		}
	}
	h.r.loads.mu.Lock()
	cells := len(h.r.loads.cells)
	h.r.loads.mu.Unlock()
	if cells != 1 {
		t.Fatalf("detached session cell retained: %d", cells)
	}
}
func TestRecorderPageLoadCloseDrainsOrderedIntents(t *testing.T) {
	h := &harness{t: t, fc: newFakeChrome(), out: &memOut{}}
	b := board.New(notes.Structural{}, nil)
	g := &loadSnapshotGate{fakeChrome: h.fc}
	g.hold("DOM.getDocument")
	h.r = Start(Options{ID: "load", Call: g, ShotEvery: -1, Operator: b, Out: func() (Opened, error) { return Opened{Sink: rec.Tee(&h.out.sink, b)}, nil }})
	var release sync.Once
	t.Cleanup(func() { release.Do(func() { close(g.release) }); h.r.Close() })
	h.page("A", "http://a/", "L1")
	h.page("B", "http://b/", "L2")
	h.r.Send(Live{Tab: rec.Tab{ID: "T-A"}, Session: "A"})
	waitLoadSignal(t, g.entered)
	h.r.Send(Live{Tab: rec.Tab{ID: "T-B"}, Session: "B"})
	updates, stop := b.SubscribeSubmissionCandidates()
	defer stop()
	closed := make(chan error, 1)
	go func() { closed <- h.r.Close() }()
	waitLoadSignal(t, updates)
	release.Do(func() { close(g.release) })
	if err := <-closed; err != nil {
		t.Fatal(err)
	}
	if got := h.kinds(); got != "head seg:live doc seg:tab doc stop:end" {
		t.Fatalf("close lost accepted FIFO intents: %s", got)
	}
	if b.PageLoadReady() {
		t.Fatal("historical close drain confirmed live scope")
	}
	h.r.loads.mu.Lock()
	cells := len(h.r.loads.cells)
	h.r.loads.mu.Unlock()
	if cells != 0 {
		t.Fatalf("close retained %d session cells", cells)
	}
}
func TestRecorderPageLoadDuplicateEventCostAndFrameIsolation(t *testing.T) {
	for _, n := range []int{8, 16} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			h, b, _ := loadHarness(t)
			before := h.fc.countCalls("A DOM.getDocument")
			for i := 0; i < n; i++ {
				h.r.Send(Event{Session: "A", Method: "Page.frameNavigated", Seq: h.fc.next(), Params: raw(map[string]any{"frame": map[string]any{"id": "T-A", "loaderId": "L1"}})})
			}
			h.r.Send(Event{Session: "A", Method: "Page.frameNavigated", Params: raw(map[string]any{"frame": map[string]any{"id": "child", "parentId": "T-A", "loaderId": "child-load"}})})
			h.r.Send(Event{Session: "foreign", Method: "Page.frameNavigated", Params: raw(map[string]any{"frame": map[string]any{"id": "foreign", "loaderId": "foreign-load"}})})
			h.r.Send(Event{Session: "A", Method: "Page.navigatedWithinDocument", Params: raw(map[string]any{"frameId": "T-A", "url": "http://same/#fragment"})})
			h.settle()
			if got := h.fc.countCalls("A DOM.getDocument") - before; got != 0 {
				t.Fatalf("duplicate/frame/same-document notifications started %d snapshots", got)
			}
			if !b.PageLoadReady() {
				t.Fatal("non-load events cleared confirmed scope")
			}
			for i := 0; i < n; i++ {
				h.r.Send(Event{Session: "A", Method: "DOM.documentUpdated", Seq: h.fc.next(), Params: raw(map[string]any{})})
			}
			h.settle()
			if got := h.fc.countCalls("A DOM.getDocument") - before; got > n {
				t.Fatalf("%d events started %d snapshots", n, got)
			}
			segs := 0
			for _, r := range h.out.sink.Records() {
				if r.Kind() == "seg" {
					segs++
				}
			}
			if segs != 1 {
				t.Fatalf("unchanged identity created %d activation segments", segs)
			}
		})
	}
}
func TestRecorderPageLoadPendingKeepsHistoricalResolution(t *testing.T) {
	h, b, _, n := preparedFixture(t)
	h.r.Send(Emit{Record: &n.Note, Doc: n.Document})
	h.settle()
	b.BeginPageLoad(900)
	stale := n.Note
	stale.ID = "fresh-pending"
	h.r.Send(Emit{Record: &stale, Doc: n.Document})
	rev := uint64(0)
	h.r.Send(Emit{Record: &rec.NoteState{ID: n.Note.ID, State: rec.NoteResolved, IfRevision: &rev}, Doc: 0})
	h.settle()
	var fresh, states int
	for _, r := range h.out.sink.Records() {
		if note, ok := r.(*rec.Note); ok && note.ID == stale.ID {
			fresh++
		}
		if r.Kind() == "note-state" {
			states++
		}
	}
	if fresh != 0 || states != 1 {
		t.Fatalf("pending creation or historical resolution changed: fresh=%d resolutions=%d", fresh, states)
	}
}

func TestRecorderPageLoadViewportIOStillPrecedesReservation(t *testing.T) {
	h, _, g := loadHarness(t)
	g.hold("Page.getLayoutMetrics")
	loadNavigate(h, "L2")
	waitLoadSignal(t, g.entered)
	loadNavigate(h, "L3")
	waitLoadSignal(t, g.cancelled)
	h.settle()
	var loaders []string
	for _, r := range h.out.sink.Records() {
		if s, ok := r.(*rec.Seg); ok {
			loaders = append(loaders, s.PageLoad.Loader)
		}
	}
	if !reflect.DeepEqual(loaders, []string{"L1", "L3"}) {
		t.Fatalf("viewport IO had already reserved old evidence: %v", loaders)
	}
}

func TestRecorderPageLoadFailedEvidenceStaysUnavailableUntilExplicitRetry(t *testing.T) {
	for _, fault := range []string{"frame-tree", "document", "missing-root"} {
		t.Run(fault, func(t *testing.T) {
			h, b, g := loadHarness(t)
			before := h.fc.countCalls("A DOM.getDocument")
			g.mu.Lock()
			g.faultMethod = "DOM.getDocument"
			g.fault = fmt.Errorf("injected snapshot failure")
			if fault == "frame-tree" {
				g.faultMethod = "Page.getFrameTree"
			}
			if fault == "missing-root" {
				g.fault = nil
				g.emptyRoot = true
			}
			g.mu.Unlock()
			loadNavigate(h, "L2")
			h.settle()
			if b.CurrentPageLoad() != nil || b.PageLoadReady() {
				t.Fatal("failed evidence restored earlier recorded load")
			}
			if fault == "frame-tree" && h.fc.countCalls("A DOM.getDocument") != before {
				t.Fatal("failed frame tree reused a prior loader for a document capture")
			}
			for _, r := range h.out.sink.Records() {
				if seg, ok := r.(*rec.Seg); ok && seg.PageLoad.Loader != "L1" {
					t.Fatalf("failed snapshot created load: %+v", seg.PageLoad)
				}
			}
			h.r.Send(Pause{})
			h.r.Send(Live{Tab: rec.Tab{ID: "T-A"}, Session: "A"})
			h.settle()
			if load := b.CurrentPageLoad(); load == nil || load.Loader != "L2" {
				t.Fatalf("explicit retry failed to establish current load: %+v", load)
			}
		})
	}
}
