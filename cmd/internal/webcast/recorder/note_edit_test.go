package recorder

import (
	"context"
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast/board"
	rec "github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func TestOperatorEditRecordsOnceAndRechecksPause(t *testing.T) {
	h, b, ch := operatorHarness(t)
	draft := prepareOperator(t, h, ch)
	h.r.Send(operatorEvent(operatorRequest("save-note", "2", map[string]any{"draft": draft.Draft, "color": 0, "text": "original"}), 7))
	if got := takeOperator(t, ch); got.Outcome != board.OperatorPending {
		t.Fatal(got)
	}
	created := takeOperator(t, ch)
	if created.Outcome != board.OperatorAccepted {
		t.Fatal(created)
	}
	req := operatorRequest("edit-note", "3", map[string]any{"seg": 1, "id": created.ID, "text": "edited"})
	h.r.Send(operatorEvent(req, 7))
	if got := takeOperator(t, ch); got.Outcome != board.OperatorPending {
		t.Fatal(got)
	}
	if got := takeOperator(t, ch); got.Outcome != board.OperatorAccepted || got.ID != "" {
		t.Fatal(got)
	}
	h.r.Send(operatorEvent(req, 7))
	if got := takeOperator(t, ch); got.Outcome != board.OperatorAccepted {
		t.Fatal(got)
	}
	notes := b.OperatorView().Notes
	if len(notes) != 1 || notes[0].Text != "edited" || !notes[0].Edited || notes[0].Author.ID != board.Operator {
		t.Fatalf("notes %+v", notes)
	}
	if err := h.r.ask(context.Background(), func() {
		h.r.Send(operatorEvent(operatorRequest("edit-note", "4", map[string]any{"seg": 1, "id": created.ID, "text": "late"}), 7))
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
	h.settle()
	n := 0
	for _, r := range h.out.sink.Records() {
		if _, ok := r.(*rec.NoteEdit); ok {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("recorded %d edits", n)
	}
}
