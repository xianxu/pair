package recorder

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/capability"
	"github.com/xianxu/pair/cmd/internal/webcast/notes"
	rec "github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func operatorHarness(t *testing.T) (*harness, *board.Board, chan OperatorReply) {
	return operatorHarnessAnnotation(t, true)
}

func operatorHarnessAnnotation(t *testing.T, allowed bool) (*harness, *board.Board, chan OperatorReply) {
	h := newHarness(t)
	b := board.New(notes.Structural{}, nil)
	b.WithAnnotationCleanup(h.r.ClearAnnotations)
	if err := b.SetAnnotationAllowed(context.Background(), allowed); err != nil {
		t.Fatal(err)
	}
	replies := make(chan OperatorReply, 64)
	if err := h.r.ask(context.Background(), func() {
		h.r.opts.Operator = b
		h.r.opts.OperatorReply = func(session string, ctx int64, reply OperatorReply) { replies <- reply }
		h.r.opts.Out = func() (Opened, error) { h.out.opened++; return Opened{Sink: rec.Tee(&h.out.sink, b)}, nil }
	}); err != nil {
		t.Fatal(err)
	}
	p := h.page("A", "http://a/", "L1")
	p.insert(p.body(), nil, p.el("p"))
	p.take()
	h.r.Send(Event{Session: "A", Method: "Runtime.executionContextCreated", Params: raw(map[string]any{"context": map[string]any{"id": 7, "name": "couchcast", "auxData": map[string]any{"frameId": p.frameID}}})})
	h.r.Send(Live{Tab: rec.Tab{ID: p.frameID}, Session: "A"})
	h.settle()
	return h, b, replies
}
func operatorEvent(request any, ctx int64) Event {
	return Event{Session: "A", Method: "Runtime.bindingCalled", Params: raw(map[string]any{"name": "__couchcast", "executionContextId": ctx, "payload": string(raw(map[string]any{"t": "operator", "request": request}))})}
}
func operatorRequest(op, id string, body any, generation ...uint64) any {
	g := uint64(1)
	if len(generation) > 0 {
		g = generation[0]
	}
	return map[string]any{"op": op, "request": id, "body": body, "generation": strconv.FormatUint(g, 10)}
}
func takeOperator(t *testing.T, ch <-chan OperatorReply) OperatorReply {
	t.Helper()
	select {
	case r := <-ch:
		return r
	case <-time.After(5 * time.Second):
		t.Fatal("no operator reply")
		return OperatorReply{}
	}
}
func prepareOperator(t *testing.T, h *harness, ch <-chan OperatorReply) OperatorReply {
	h.r.Send(operatorEvent(operatorRequest("prepare-note", "1", map[string]any{"seg": 1, "path": []int{1, 2, 0}, "box": []int{1, 2, 30, 40}, "region": []int{3, 4, 10, 10}}), 7))
	got := takeOperator(t, ch)
	if got.Outcome != board.OperatorAccepted || got.Draft == "" {
		t.Fatalf("prepare %+v", got)
	}
	return got
}
func TestOperatorBindingPrepareSaveAndReplay(t *testing.T) {
	h, b, ch := operatorHarness(t)
	draft := prepareOperator(t, h, ch)
	req := operatorRequest("save-note", "2", map[string]any{"draft": draft.Draft, "color": 0, "text": "hello"})
	h.r.Send(operatorEvent(req, 7))
	first := takeOperator(t, ch)
	if first.Outcome != board.OperatorPending {
		t.Fatalf("enqueue %+v", first)
	}
	final := takeOperator(t, ch)
	if final.Outcome != board.OperatorAccepted || final.ID == "" {
		t.Fatalf("settle %+v", final)
	}
	h.r.Send(operatorEvent(req, 7))
	if got := takeOperator(t, ch); got.Outcome != board.OperatorAccepted || got.ID != final.ID {
		t.Fatalf("replay %+v", got)
	}
	v := b.OperatorView()
	if len(v.Notes) != 1 || v.Notes[0].Author.ID != board.Operator || *v.Notes[0].Region != (rec.Box{3, 4, 10, 10}) {
		t.Fatalf("notes %+v", v.Notes)
	}
}
func TestOperatorOrderingPauseRefusesAndReleases(t *testing.T) {
	h, b, ch := operatorHarness(t)
	draft := prepareOperator(t, h, ch)
	// Queue both while the loop is held: save admits before Pause, but its
	// queued writer input follows Pause and must settle known-not-written.
	if err := h.r.ask(context.Background(), func() {
		h.r.Send(operatorEvent(operatorRequest("save-note", "2", map[string]any{"draft": draft.Draft, "color": 0, "text": "hello"}), 7))
		h.r.Send(Pause{})
	}); err != nil {
		t.Fatal(err)
	}
	if got := takeOperator(t, ch); got.Outcome != board.OperatorPending {
		t.Fatal(got)
	}
	if got := takeOperator(t, ch); got.Outcome != board.OperatorRefused || got.Error != board.ErrPaused.Error() {
		t.Fatal(got)
	}
	if len(b.OperatorView().Notes) != 0 {
		t.Fatal("write crossed pause")
	}
	// Paused bindings still receive a refusal instead of disappearing.
	h.r.Send(operatorEvent(operatorRequest("save-note", "3", map[string]any{"draft": draft.Draft, "color": 0, "text": "hello"}), 7))
	if got := takeOperator(t, ch); got.Error != board.ErrPaused.Error() {
		t.Fatal(got)
	}
}
func TestOperatorRejectsUnknownContextAndDestroyedDraft(t *testing.T) {
	h, _, ch := operatorHarness(t)
	draft := prepareOperator(t, h, ch)
	req := operatorRequest("save-note", "2", map[string]any{"draft": draft.Draft, "color": 0, "text": "hello"})
	h.r.Send(operatorEvent(req, 99))
	if got := takeOperator(t, ch); got.Outcome != board.OperatorRefused {
		t.Fatal(got)
	}
	h.r.Send(Event{Session: "A", Method: "Runtime.executionContextDestroyed", Params: raw(map[string]any{"executionContextId": 7})})
	h.r.Send(operatorEvent(req, 7))
	if got := takeOperator(t, ch); got.Outcome != board.OperatorRefused {
		t.Fatal(got)
	}
}

type operatorFailSink struct{ rec.Sink }

func (s operatorFailSink) Put(r rec.Record) error {
	if r.Kind() == "note" {
		return errors.New("partial sink failure")
	}
	return s.Sink.Put(r)
}
func TestOperatorSinkFailureIsUnknown(t *testing.T) {
	h, _, ch := operatorHarness(t)
	draft := prepareOperator(t, h, ch)
	h.r.ask(context.Background(), func() { h.r.sink = operatorFailSink{h.r.sink} })
	h.r.Send(operatorEvent(operatorRequest("save-note", "2", map[string]any{"draft": draft.Draft, "color": 0, "text": "hello"}), 7))
	if got := takeOperator(t, ch); got.Outcome != board.OperatorPending {
		t.Fatal(got)
	}
	if got := takeOperator(t, ch); got.Outcome != board.OperatorUnknown {
		t.Fatal(got)
	}
}

func TestOperatorCompletionKeepsOriginalRequest(t *testing.T) {
	h, _, ch := operatorHarness(t)
	draft := prepareOperator(t, h, ch)
	h.r.ask(context.Background(), func() {
		h.r.Send(operatorEvent(operatorRequest("save-note", "2", map[string]any{"draft": draft.Draft, "color": 0, "text": "hello"}), 7))
		h.r.Send(operatorEvent(operatorRequest("stroke", "3", map[string]any{"seg": 1, "id": "s", "color": 0, "pts": [][3]int{{1, 2, 0}}}), 7))
	})
	if got := takeOperator(t, ch); got.Request != "2" || got.Outcome != board.OperatorPending {
		t.Fatal(got)
	}
	if got := takeOperator(t, ch); got.Request != "3" || got.Outcome != board.OperatorRefused {
		t.Fatal(got)
	}
	if got := takeOperator(t, ch); got.Request != "2" || got.Outcome != board.OperatorAccepted {
		t.Fatal(got)
	}
}
func TestOperatorOldContextCannotRebindAfterNavigation(t *testing.T) {
	h, _, ch := operatorHarness(t)
	prepareOperator(t, h, ch)
	p := h.page("A", "http://a/new", "L2")
	p.insert(p.body(), nil, p.el("p"))
	p.take()
	h.r.Send(Event{Session: "A", Method: "DOM.documentUpdated", Params: raw(map[string]any{}), Seq: h.fc.next()})
	h.settle()
	h.r.Send(operatorEvent(operatorRequest("prepare-note", "3", map[string]any{"seg": 2, "path": []int{1, 2, 0}, "box": []int{1, 2, 30, 40}, "region": []int{3, 4, 10, 10}}), 7))
	if got := takeOperator(t, ch); got.Outcome != board.OperatorRefused || got.Error != board.ErrStale.Error() {
		t.Fatal(got)
	}
}

func TestOperatorUnusedOldContextCannotAcquireNewDocument(t *testing.T) {
	h, _, ch := operatorHarness(t)
	p := h.page("A", "http://a/new", "L2")
	p.insert(p.body(), nil, p.el("p"))
	p.take()
	h.r.Send(Event{Session: "A", Method: "DOM.documentUpdated", Params: raw(map[string]any{}), Seq: h.fc.next()})
	h.settle()
	h.r.Send(operatorEvent(operatorRequest("prepare-note", "1", map[string]any{"seg": 2, "path": []int{1, 2, 0}, "box": []int{1, 2, 30, 40}, "region": []int{3, 4, 10, 10}}), 7))
	if got := takeOperator(t, ch); got.Outcome != board.OperatorRefused {
		t.Fatal(got)
	}
}

func TestOperatorQueueOverflowAndShutdownReply(t *testing.T) {
	h, _, ch := operatorHarness(t)
	req := operatorRequest("stroke", "1", map[string]any{"seg": 1, "id": "s", "color": 0, "pts": [][3]int{{1, 2, 0}}})
	h.r.ask(context.Background(), func() {
		h.r.mu.Lock()
		h.r.opts.MaxQueue = 1
		h.r.mu.Unlock()
		h.r.Send(Event{Session: "A", Method: "Page.unhandled", Params: raw(map[string]any{})})
		h.r.Send(operatorEvent(req, 7))
	})
	if got := takeOperator(t, ch); got.Outcome != board.OperatorRefused {
		t.Fatal(got)
	}
	h.settle()
	h.r.Close()
	h.r.Send(operatorEvent(req, 7))
	if got := takeOperator(t, ch); got.Outcome != board.OperatorRefused {
		t.Fatal(got)
	}
}
func TestOperatorNewContextMayPrepareAfterNavigation(t *testing.T) {
	h, _, ch := operatorHarness(t)
	prepareOperator(t, h, ch)
	p := h.page("A", "http://a/new", "L2")
	p.insert(p.body(), nil, p.el("p"))
	p.take()
	h.r.Send(Event{Session: "A", Method: "Page.frameNavigated", Params: raw(map[string]any{"frame": map[string]any{"id": p.frameID, "loaderId": "L2"}})})
	h.r.Send(Event{Session: "A", Method: "Runtime.executionContextCreated", Params: raw(map[string]any{"context": map[string]any{"id": 8, "name": "couchcast", "auxData": map[string]any{"frameId": p.frameID}}})})
	h.r.Send(Event{Session: "A", Method: "DOM.documentUpdated", Params: raw(map[string]any{}), Seq: h.fc.next()})
	h.settle()
	h.r.Send(operatorEvent(operatorRequest("prepare-note", "1", map[string]any{"seg": 2, "path": []int{1, 2, 0}, "box": []int{1, 2, 30, 40}, "region": []int{3, 4, 10, 10}}), 8))
	if got := takeOperator(t, ch); got.Outcome != board.OperatorAccepted {
		t.Fatal(got)
	}
}

func TestOperatorPreparedNodeDoesNotReresolveMovedPath(t *testing.T) {
	h, b, ch := operatorHarness(t)
	draft := prepareOperator(t, h, ch)
	var node rec.NodeID
	h.r.ask(context.Background(), func() { node = h.r.operator.draft.node })
	h.fc.mu.Lock()
	p := h.fc.pages["A"]
	h.fc.mu.Unlock()
	p.insert(p.body(), p.body().kids[0], p.el("div"))
	h.events("A", p)
	h.settle()
	h.r.Send(operatorEvent(operatorRequest("save-note", "2", map[string]any{"draft": draft.Draft, "color": 0, "text": "same node"}), 7))
	takeOperator(t, ch)
	if got := takeOperator(t, ch); got.Outcome != board.OperatorAccepted {
		t.Fatal(got)
	}
	if got := b.OperatorView(); len(got.Notes) != 1 || got.Notes[0].At.Node != node {
		t.Fatalf("note changed anchor: %+v", got.Notes)
	}
}
func TestOperatorOrderingDriverRefuses(t *testing.T) {
	h, _, ch := operatorHarness(t)
	draft := prepareOperator(t, h, ch)
	who := rec.Author{ID: "viewer", Name: "Viewer"}
	h.r.ask(context.Background(), func() {
		h.r.Send(operatorEvent(operatorRequest("save-note", "2", map[string]any{"draft": draft.Draft, "color": 0, "text": "hello"}), 7))
		h.r.Send(Emit{Record: &rec.Ctl{Op: rec.CtlGrant, Who: who, Control: rec.Control{Driver: &who}}})
	})
	if got := takeOperator(t, ch); got.Outcome != board.OperatorPending {
		t.Fatal(got)
	}
	if got := takeOperator(t, ch); got.Outcome != board.OperatorRefused || got.Error != board.ErrDriving.Error() {
		t.Fatal(got)
	}
}
func TestOperatorPrivateWriteQueueRefusal(t *testing.T) {
	h, _, ch := operatorHarness(t)
	draft := prepareOperator(t, h, ch)
	h.r.ask(context.Background(), func() {
		h.r.mu.Lock()
		h.r.operatorQueued = maxOperatorQueue
		h.r.mu.Unlock()
		h.r.handleOperator("A", 7, board.OperatorRequest{Op: "save-note", Request: "2", Generation: h.r.opts.Operator.AnnotationCapability().Generation, Save: &board.OperatorSave{Draft: draft.Draft, Color: 0, Text: "hello"}})
		h.r.mu.Lock()
		h.r.operatorQueued = 0
		h.r.mu.Unlock()
	})
	if got := takeOperator(t, ch); got.Outcome != board.OperatorRefused || got.Error != errOperatorQueue.Error() {
		t.Fatal(got)
	}
	select {
	case got := <-ch:
		t.Fatalf("duplicate completion %+v", got)
	default:
	}
}

func TestOperatorMissingSinkCannotAccept(t *testing.T) {
	h, _, ch := operatorHarness(t)
	draft := prepareOperator(t, h, ch)
	h.r.ask(context.Background(), func() { h.r.sink = nil })
	h.r.Send(operatorEvent(operatorRequest("save-note", "2", map[string]any{"draft": draft.Draft, "color": 0, "text": "hello"}), 7))
	if got := takeOperator(t, ch); got.Outcome != board.OperatorPending {
		t.Fatal(got)
	}
	if got := takeOperator(t, ch); got.Outcome != board.OperatorRefused {
		t.Fatal(got)
	}
}

func TestOperatorSameDocumentTabReturnGetsFreshDraftScope(t *testing.T) {
	h, _, ch := operatorHarness(t)
	first := prepareOperator(t, h, ch)
	p := h.page("B", "http://b/", "L9")
	h.ctxCreated("B", p.frameID, 8)
	h.r.Send(Live{Tab: rec.Tab{ID: p.frameID}, Session: "B"})
	h.settle()
	h.r.Send(Live{Tab: rec.Tab{ID: "T-A"}, Session: "A"})
	h.settle()
	// A still has Chrome context 7 and the same DOM, but recording DocSeg is 3.
	h.r.Send(operatorEvent(operatorRequest("save-note", "2", map[string]any{"draft": first.Draft, "color": 0, "text": "old handle"}), 7))
	if got := takeOperator(t, ch); got.Outcome != board.OperatorRefused {
		t.Fatalf("old draft %+v", got)
	}
	h.r.Send(operatorEvent(operatorRequest("stroke", "3", map[string]any{"seg": 1, "id": "old", "color": 0, "pts": [][3]int{{1, 2, 0}}}), 7))
	if got := takeOperator(t, ch); got.Outcome != board.OperatorRefused {
		t.Fatalf("old stroke %+v", got)
	}
	h.r.Send(operatorEvent(operatorRequest("prepare-note", "4", map[string]any{"seg": 3, "path": []int{1, 2, 0}, "box": []int{1, 2, 30, 40}, "region": []int{3, 4, 10, 10}}), 7))
	fresh := takeOperator(t, ch)
	if fresh.Outcome != board.OperatorAccepted || fresh.Doc != 3 || fresh.Draft == first.Draft || fresh.Draft == "" {
		t.Fatalf("returned document %+v", fresh)
	}
	h.r.Send(operatorEvent(operatorRequest("save-note", "5", map[string]any{"draft": fresh.Draft, "color": 0, "text": "returned page"}), 7))
	if got := takeOperator(t, ch); got.Outcome != board.OperatorPending {
		t.Fatal(got)
	}
	if got := takeOperator(t, ch); got.Outcome != board.OperatorAccepted {
		t.Fatal(got)
	}
}

func TestOperatorTabReturnRetiresAcceptedLedgerAndAllowsFreshSave(t *testing.T) {
	h, _, ch := operatorHarness(t)
	first := prepareOperator(t, h, ch)
	h.r.Send(operatorEvent(operatorRequest("save-note", "2", map[string]any{"draft": first.Draft, "color": 0, "text": "accepted before switching"}), 7))
	if got := takeOperator(t, ch); got.Outcome != board.OperatorPending {
		t.Fatal(got)
	}
	accepted := takeOperator(t, ch)
	if accepted.Outcome != board.OperatorAccepted || accepted.ID == "" {
		t.Fatalf("original save %+v", accepted)
	}
	// Treat this acceptance as lost in transit to Chrome. The recorder's ledger
	// can still reconcile it until A stops being the recording document.
	h.r.Send(operatorEvent(operatorRequest("status", "2", map[string]any{}), 7))
	if got := takeOperator(t, ch); got.Outcome != board.OperatorAccepted || got.ID != accepted.ID {
		t.Fatalf("original ledger %+v", got)
	}
	p := h.page("B", "http://b/", "L9")
	h.ctxCreated("B", p.frameID, 8)
	h.r.Send(Live{Tab: rec.Tab{ID: p.frameID}, Session: "B"})
	h.settle()
	h.r.Send(Live{Tab: rec.Tab{ID: "T-A"}, Session: "A"})
	h.settle()
	// The same Chrome context 7 now belongs to recording document 3. Asking
	// forever about the old accepted operation cannot recover its retired ledger.
	h.r.Send(operatorEvent(operatorRequest("status", "2", map[string]any{}), 7))
	if got := takeOperator(t, ch); got.Outcome != board.OperatorUnknown {
		t.Fatalf("retired ledger %+v", got)
	}
	h.r.Send(operatorEvent(operatorRequest("prepare-note", "3", map[string]any{"seg": 3, "path": []int{1, 2, 0}, "box": []int{1, 2, 30, 40}, "region": []int{3, 4, 10, 10}}), 7))
	fresh := takeOperator(t, ch)
	if fresh.Outcome != board.OperatorAccepted || fresh.Doc != 3 || fresh.Draft == "" || fresh.Draft == first.Draft {
		t.Fatalf("fresh draft %+v", fresh)
	}
	h.r.Send(operatorEvent(operatorRequest("save-note", "4", map[string]any{"draft": fresh.Draft, "color": 0, "text": "fresh annotation"}), 7))
	if got := takeOperator(t, ch); got.Outcome != board.OperatorPending {
		t.Fatal(got)
	}
	if got := takeOperator(t, ch); got.Outcome != board.OperatorAccepted || got.ID == "" || got.ID == accepted.ID {
		t.Fatalf("fresh save %+v", got)
	}
}

func TestOperatorRetiredGenerationPrepareRefused(t *testing.T) {
	h, b, replies := operatorHarness(t)
	b.WithAnnotationCleanup(h.r.ClearAnnotations)
	if err := b.SetAnnotationAllowed(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	h.r.Send(operatorEvent(operatorRequest("prepare-note", "1", map[string]any{"seg": 1, "path": []int{1, 2, 0}, "box": []int{1, 2, 30, 40}, "region": []int{3, 4, 10, 10}}), 7))
	got := takeOperator(t, replies)
	if got.Outcome != board.OperatorRefused || got.Error != board.ErrDisabled.Error() {
		t.Fatalf("disabled prepare: %+v", got)
	}
}

func TestOperatorIngressGenerationCannotCrossOffOn(t *testing.T) {
	h, b, replies := operatorHarness(t)
	// Keep the loop inside this callback until the binding's captured generation
	// has been retired. Fake cleanup isolates ingress from writer ordering.
	b.WithAnnotationCleanup(func(context.Context, uint64) error { return nil })
	if err := h.r.ask(context.Background(), func() {
		h.r.Send(operatorEvent(operatorRequest("prepare-note", "1", map[string]any{"seg": 1, "path": []int{1, 2, 0}, "box": []int{1, 2, 30, 40}, "region": []int{3, 4, 10, 10}}), 7))
		if err := b.SetAnnotationAllowed(context.Background(), false); err != nil {
			t.Error(err)
		}
		if err := b.SetAnnotationAllowed(context.Background(), true); err != nil {
			t.Error(err)
		}
	}); err != nil {
		t.Fatal(err)
	}
	if got := takeOperator(t, replies); got.Outcome != board.OperatorRefused || got.Error != board.ErrDisabled.Error() {
		t.Fatalf("queued prepare revived: %+v", got)
	}
}

func TestOperatorQueuedSaveGenerationAndHeldDraft(t *testing.T) {
	h, b, replies := operatorHarness(t)
	draft := prepareOperator(t, h, replies)
	b.WithAnnotationCleanup(func(context.Context, uint64) error { return nil })
	if err := h.r.ask(context.Background(), func() {
		h.r.handleOperator("A", 7, board.OperatorRequest{Op: "save-note", Request: "2", Generation: h.r.opts.Operator.AnnotationCapability().Generation, Save: &board.OperatorSave{Draft: draft.Draft, Color: 0, Text: "held text"}})
		if err := b.SetAnnotationAllowed(context.Background(), false); err != nil {
			t.Error(err)
		}
		if err := b.SetAnnotationAllowed(context.Background(), true); err != nil {
			t.Error(err)
		}
	}); err != nil {
		t.Fatal(err)
	}
	if got := takeOperator(t, replies); got.Outcome != board.OperatorPending {
		t.Fatal(got)
	}
	if got := takeOperator(t, replies); got.Outcome != board.OperatorRefused || got.Error != board.ErrDisabled.Error() {
		t.Fatalf("queued save revived: %+v", got)
	}
	if len(b.OperatorView().Notes) != 0 {
		t.Fatal("stale write reached sink")
	}
	// The draft is content, not authority: a fresh save may use the same handle.
	h.r.Send(operatorEvent(operatorRequest("save-note", "3", map[string]any{"draft": draft.Draft, "color": 0, "text": "held text"}, b.AnnotationCapability().Generation), 7))
	if got := takeOperator(t, replies); got.Outcome != board.OperatorPending {
		t.Fatal(got)
	}
	if got := takeOperator(t, replies); got.Outcome != board.OperatorAccepted {
		t.Fatal(got)
	}
	if len(b.OperatorView().Notes) != 1 {
		t.Fatal("held draft was lost")
	}
}

func TestOperatorQueuedBindingDoesNotDeadlockCleanup(t *testing.T) {
	h, b, replies := operatorHarness(t)
	entered, release, loopDone := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() { loopDone <- h.r.ask(context.Background(), func() { close(entered); <-release }) }()
	<-entered
	h.r.Send(operatorEvent(operatorRequest("stroke", "1", map[string]any{"seg": 1, "id": "pen", "color": 0, "pts": [][3]float64{{1, 2, 0}}}), 7))
	off := make(chan error, 1)
	go func() { off <- b.SetAnnotationAllowed(context.Background(), false) }()
	deadline := time.Now().Add(time.Second)
	for b.AnnotationCapability().Phase != capability.Disabling && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	close(release)
	if err := <-loopDone; err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-off:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("operator waited on cleanup command lock")
	}
	if got := takeOperator(t, replies); got.Outcome != board.OperatorRefused {
		t.Fatal(got)
	}
}

func TestOperatorQueuedWriterDoesNotWaitForCleanupCommand(t *testing.T) {
	h, b, replies := operatorHarness(t)
	draft := prepareOperator(t, h, replies)
	off := make(chan error, 1)
	if err := h.r.ask(context.Background(), func() {
		h.r.handleOperator("A", 7, board.OperatorRequest{Op: "save-note", Request: "2", Generation: h.r.opts.Operator.AnnotationCapability().Generation, Save: &board.OperatorSave{Draft: draft.Draft, Color: 0, Text: "queued"}})
		go func() { off <- b.SetAnnotationAllowed(context.Background(), false) }()
		// Cleanup now holds command and waits behind this callback and writer.
		until := time.Now().Add(time.Second)
		for b.AnnotationCapability().Phase != capability.Disabling && time.Now().Before(until) {
			time.Sleep(time.Millisecond)
		}
	}); err != nil {
		t.Fatal(err)
	}
	if got := takeOperator(t, replies); got.Outcome != board.OperatorPending {
		t.Fatal(got)
	}
	if got := takeOperator(t, replies); got.Outcome != board.OperatorRefused {
		t.Fatal(got)
	}
	select {
	case err := <-off:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("writer deadlocked behind annotation cleanup")
	}
	if len(b.OperatorView().Notes) != 0 {
		t.Fatal("write crossed clear")
	}
}

func TestOperatorOriginGenerationBeforeIngressCannotCrossOffOn(t *testing.T) {
	for _, op := range []string{"stroke", "prepare-note", "save-note", "resolve"} {
		t.Run(op, func(t *testing.T) {
			h, b, replies := operatorHarness(t)
			draft := prepareOperator(t, h, replies)
			bodies := map[string]any{
				"stroke":       map[string]any{"seg": 1, "id": "old", "color": 0, "pts": [][3]float64{{1, 2, 0}}},
				"prepare-note": map[string]any{"seg": 1, "path": []int{1, 2, 0}, "box": []int{1, 2, 30, 40}, "region": []int{3, 4, 10, 10}},
				"save-note":    map[string]any{"draft": draft.Draft, "color": 0, "text": "old save"},
				"resolve":      map[string]any{"seg": 1, "id": "old", "state": "resolved"},
			}
			// Constructed by the page before OFF, delivered to ingress only after ON.
			delayed := operatorEvent(operatorRequest(op, "2", bodies[op]), 7)
			if err := b.SetAnnotationAllowed(context.Background(), false); err != nil {
				t.Fatal(err)
			}
			if err := b.SetAnnotationAllowed(context.Background(), true); err != nil {
				t.Fatal(err)
			}
			h.r.Send(delayed)
			got := takeOperator(t, replies)
			if got.Outcome != board.OperatorRefused || got.Error != board.ErrDisabled.Error() {
				t.Fatalf("old originating generation acquired new authority: %+v", got)
			}
			if v := b.OperatorView(); len(v.Strokes) != 0 || len(v.Notes) != 0 {
				t.Fatal("old mutation reached sink")
			}
		})
	}
}

func TestOperatorResolveRefusesRevisionChangedBeforeWrite(t *testing.T) {
	h, b, ch := operatorHarness(t)
	draft := prepareOperator(t, h, ch)
	h.r.Send(operatorEvent(operatorRequest("save-note", "2", map[string]any{"draft": draft.Draft, "color": 0, "text": "original"}), 7))
	if got := takeOperator(t, ch); got.Outcome != board.OperatorPending {
		t.Fatal(got)
	}
	saved := takeOperator(t, ch)
	if saved.Outcome != board.OperatorAccepted {
		t.Fatal(saved)
	}
	if err := h.r.ask(context.Background(), func() {
		// Admission freezes the revision, then a reopen wins before its queued writer.
		event := operatorEvent(operatorRequest("resolve", "3", map[string]any{"seg": 1, "id": saved.ID, "state": rec.NoteResolved}), 7)
		h.r.operatorBinding(event)
		if !h.r.m.Emit(h.r.m.State().T, &rec.NoteState{ID: saved.ID, State: rec.NoteResolved}) || !h.r.m.Emit(h.r.m.State().T, &rec.NoteState{ID: saved.ID, State: rec.NoteOpen}) {
			t.Error("fixture transition rejected")
		}
		if err := h.r.write(); err != nil {
			t.Error(err)
		}
	}); err != nil {
		t.Fatal(err)
	}
	if got := takeOperator(t, ch); got.Outcome != board.OperatorPending {
		t.Fatal(got)
	}
	if got := takeOperator(t, ch); got.Outcome != board.OperatorRefused || got.Error != board.ErrStale.Error() {
		t.Fatalf("late resolve: %+v", got)
	}
	if got := b.OperatorView(); len(got.Notes) != 1 || got.Notes[0].ID != saved.ID {
		t.Fatal("late resolve removed reopened note")
	}
}
