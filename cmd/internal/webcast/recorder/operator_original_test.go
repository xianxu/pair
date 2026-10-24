package recorder

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast/board"
	rec "github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func TestOperatorOriginalsUsesCurrentRecordedIdentityWhilePaused(t *testing.T) {
	h, b, ch := operatorHarness(t)
	draft := prepareOperator(t, h, ch)
	h.r.Send(operatorEvent(operatorRequest("save-note", "2", map[string]any{"draft": draft.Draft, "color": 0, "text": "Save"}), 7))
	takeOperator(t, ch)
	saved := takeOperator(t, ch)
	if saved.Outcome != board.OperatorAccepted {
		t.Fatal(saved)
	}
	note := b.OperatorView().Notes[0]
	for _, paused := range []bool{false, true} {
		if paused {
			h.r.Send(Pause{})
			h.settle()
		}
		refs, err := h.r.OperatorOriginals(context.Background(), "A", 7, 1, []rec.NodeID{note.At.Node, 999999})
		if err != nil || len(refs) != 1 || refs[note.At.Node] == 0 {
			t.Fatalf("paused=%v refs=%v err=%v", paused, refs, err)
		}
	}
	for _, scope := range []struct {
		s string
		c int64
		d int
	}{{"other", 7, 1}, {"A", 8, 1}, {"A", 7, 2}} {
		if _, err := h.r.OperatorOriginals(context.Background(), scope.s, scope.c, scope.d, []rec.NodeID{note.At.Node}); !errors.Is(err, board.ErrStale) {
			t.Fatalf("foreign scope accepted: %+v %v", scope, err)
		}
	}
}

func TestOperatorSketchOriginalSaveIdentityWithoutNotesAndReplacement(t *testing.T) {
	h, b, _ := operatorHarness(t)
	h.fc.mu.Lock()
	p := h.fc.pages["A"]
	h.fc.mu.Unlock()
	button := p.el("button")
	p.insert(button, nil, p.txt("Save"))
	p.take()
	p.insert(p.body(), nil, button)
	h.events("A", p)
	h.settle()
	var node rec.NodeID
	if err := h.r.ask(context.Background(), func() { node = h.r.m.sessions["A"].ids[button.backend] }); err != nil {
		t.Fatal(err)
	}
	begin := mirrorSketchBegin()
	begin.Tab = "T-A"
	h.r.Send(Emit{Record: begin, Doc: 1})
	h.r.Send(Emit{Record: &rec.SketchStrokeBegin{SketchOperation: rec.SketchOperation{Operation: "stroke", Sequence: 1, Digest: strings.Repeat("b", 64)}, Sketch: "sk", ID: "p", At: rec.NoteAt{Node: node, Text: "Save", Box: &rec.Box{1, 2, 30, 40}}, Width: 3}, Doc: 1})
	h.settle()
	if len(b.OperatorView().Notes) != 0 {
		t.Fatal("fixture accidentally has note authority")
	}
	refs, err := h.r.OperatorOriginals(context.Background(), "A", 7, 1, []rec.NodeID{node})
	if err != nil || refs[node] != button.backend {
		t.Fatalf("unchanged common Save lost recorded proof: %v %v", refs, err)
	}
	if _, err = h.r.OperatorOriginals(context.Background(), "A", 7, 2, []rec.NodeID{node}); !errors.Is(err, board.ErrStale) {
		t.Fatalf("cross-document proof accepted: %v", err)
	}
	replacement := p.el("button")
	p.insert(replacement, nil, p.txt("Save"))
	p.take()
	p.remove(button)
	p.insert(p.body(), nil, replacement)
	h.events("A", p)
	h.settle()
	var replacementID rec.NodeID
	h.r.ask(context.Background(), func() { replacementID = h.r.m.sessions["A"].ids[replacement.backend] })
	refs, err = h.r.OperatorOriginals(context.Background(), "A", 7, 1, []rec.NodeID{node, replacementID})
	if err != nil || len(refs) != 0 {
		t.Fatalf("identical replacement claimed recorded identity: %v %v", refs, err)
	}
}

func TestOperatorOriginalsSameLoadReturnsAcrossActivationSegments(t *testing.T) {
	h, b, _, n := preparedFixture(t)
	h.r.Send(Emit{Record: &n.Note, Doc: n.Document})
	h.settle()
	node := n.Note.At.Node
	initial, err := h.r.OperatorOriginals(context.Background(), "A", 7, 1, []rec.NodeID{node})
	if err != nil || initial[node] == 0 {
		t.Fatal(initial, err)
	}
	h.page("B", "http://b/", "L9")
	h.r.Send(Live{Tab: rec.Tab{ID: "T-B"}, Session: "B"})
	h.settle()
	h.r.Send(Live{Tab: rec.Tab{ID: "T-A"}, Session: "A"})
	h.settle()
	returned, err := h.r.OperatorOriginals(context.Background(), "A", 7, 3, []rec.NodeID{node})
	if err != nil || returned[node] != initial[node] {
		t.Fatalf("same-load note lost trusted identity across A-B-A: before=%v after=%v err=%v", initial, returned, err)
	}
	if b.OperatorView().Notes[0].ID != n.Note.ID {
		t.Fatal("same-load saved note did not return")
	}
	h.r.Send(Pause{})
	h.settle()
	paused, err := h.r.OperatorOriginals(context.Background(), "A", 7, 3, []rec.NodeID{node})
	if err != nil || paused[node] != initial[node] {
		t.Fatalf("pause revoked same-load anchor: %v %v", paused, err)
	}
}

func TestOperatorSketchOriginalsSameLoadReturnAndPendingFence(t *testing.T) {
	h, b, _ := operatorHarness(t)
	var node rec.NodeID
	h.r.ask(context.Background(), func() { node, _, _ = h.r.m.nodeAt(h.r.m.sessions["A"], h.r.frameIDs["A"], []int{1, 2, 0}) })
	begin := mirrorSketchBegin()
	begin.Tab = "T-A"
	h.r.Send(Emit{Record: begin, Doc: 1})
	h.r.Send(Emit{Record: &rec.SketchStrokeBegin{SketchOperation: rec.SketchOperation{Operation: "stroke", Sequence: 1, Digest: strings.Repeat("b", 64)}, Sketch: "sk", ID: "p", At: rec.NoteAt{Node: node, Box: &rec.Box{1, 2, 30, 40}}, Width: 3}, Doc: 1})
	h.settle()
	before, err := h.r.OperatorOriginals(context.Background(), "A", 7, 1, []rec.NodeID{node})
	if err != nil || before[node] == 0 {
		t.Fatal(before, err)
	}
	h.page("B", "http://b/", "L9")
	h.r.Send(Live{Tab: rec.Tab{ID: "T-B"}, Session: "B"})
	h.settle()
	h.r.Send(Live{Tab: rec.Tab{ID: "T-A"}, Session: "A"})
	h.settle()
	after, err := h.r.OperatorOriginals(context.Background(), "A", 7, 3, []rec.NodeID{node})
	if err != nil || after[node] != before[node] {
		t.Fatalf("same-load sketch lost original anchor: %v %v", after, err)
	}
	b.BeginPageLoad(999)
	if _, err = h.r.OperatorOriginals(context.Background(), "A", 7, 3, []rec.NodeID{node}); !errors.Is(err, board.ErrStale) {
		t.Fatalf("pending load exposed previous backend references: %v", err)
	}
}
