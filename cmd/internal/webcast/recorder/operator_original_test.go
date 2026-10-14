package recorder

import (
	"context"
	"errors"
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
