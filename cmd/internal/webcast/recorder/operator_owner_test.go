package recorder

import (
	"context"
	"testing"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/capability"
)

func ownerPrepareRequest(generation uint64) any {
	return operatorRequest("prepare-note", "1", map[string]any{"seg": 1, "path": []int{1, 2, 0}, "box": []int{1, 2, 30, 40}, "region": []int{3, 4, 10, 10}}, generation)
}

func TestOperatorOwnerInitialOffPrepareSaveStroke(t *testing.T) {
	h, b, replies := operatorHarnessAnnotation(t, false)
	if snapshot := b.AnnotationCapability(); snapshot.Phase != capability.Off || snapshot.Generation != 0 {
		t.Fatalf("not initial OFF: %+v", snapshot)
	}
	h.r.Send(operatorEvent(ownerPrepareRequest(0), 7))
	draft := takeOperator(t, replies)
	if draft.Outcome != board.OperatorAccepted || draft.Draft == "" {
		t.Fatalf("owner prepare while viewer OFF: %+v", draft)
	}
	h.r.Send(operatorEvent(operatorRequest("save-note", "2", map[string]any{"draft": draft.Draft, "color": 0, "text": "owner note"}, 0), 7))
	if got := takeOperator(t, replies); got.Outcome != board.OperatorPending {
		t.Fatalf("owner save enqueue: %+v", got)
	}
	if got := takeOperator(t, replies); got.Outcome != board.OperatorAccepted || got.ID == "" {
		t.Fatalf("owner save: %+v", got)
	}
	h.r.Send(operatorEvent(operatorRequest("stroke", "3", map[string]any{"seg": 1, "id": "owner", "color": 0, "pts": [][3]int{{1, 2, 0}}}, 0), 7))
	if got := takeOperator(t, replies); got.Outcome != board.OperatorPending {
		t.Fatalf("owner stroke enqueue: %+v", got)
	}
	if got := takeOperator(t, replies); got.Outcome != board.OperatorAccepted {
		t.Fatalf("owner stroke: %+v", got)
	}
	if view := b.OperatorView(); len(view.Notes) != 1 || len(view.Strokes) != 1 || view.Notes[0].Owner != broadcast.OperatorNoteOwner || view.Notes[0].Author.ID != board.Operator || view.Strokes[0].Author.ID != board.Operator {
		t.Fatalf("owner effects missing: %+v", view)
	}
	if _, err := b.AdmitAnnotation(); err == nil || b.AnnotationCapability().Phase != capability.Off {
		t.Fatal("owner mutation granted viewer authority")
	}
}

func TestOperatorOwnerOffRejectsUnprovedOrigins(t *testing.T) {
	for _, origin := range []string{"unknown", "wrong-world", "iframe", "other-session", "retired"} {
		t.Run(origin, func(t *testing.T) {
			h, b, replies := operatorHarnessAnnotation(t, false)
			ctx := int64(99)
			session := "A"
			switch origin {
			case "wrong-world", "iframe", "other-session":
				world, frame := "couchcast", ""
				if err := h.r.ask(context.Background(), func() { frame = h.r.frameIDs["A"] }); err != nil {
					t.Fatal(err)
				}
				if origin == "wrong-world" {
					world = "page"
				}
				if origin == "iframe" {
					frame = "iframe"
				}
				if origin == "other-session" {
					session = "B"
				}
				h.r.Send(Event{Session: session, Method: "Runtime.executionContextCreated", Params: raw(map[string]any{"context": map[string]any{"id": ctx, "name": world, "auxData": map[string]any{"frameId": frame}}})})
			case "retired":
				ctx = 7
				h.r.Send(Event{Session: "A", Method: "Runtime.executionContextDestroyed", Params: raw(map[string]any{"executionContextId": ctx})})
			}
			event := operatorEvent(ownerPrepareRequest(0), ctx)
			event.Session = session
			h.r.Send(event)
			if got := takeOperator(t, replies); got.Outcome != board.OperatorRefused || got.Error != board.ErrStale.Error() {
				t.Fatalf("unproved origin %s: %+v", origin, got)
			}
			if view := b.OperatorView(); len(view.Notes) != 0 || len(view.Strokes) != 0 {
				t.Fatal("unproved origin wrote feedback")
			}
		})
	}
}

func TestOperatorOwnerInitialOffGenerationRetiredBeforeDispatch(t *testing.T) {
	h, b, replies := operatorHarnessAnnotation(t, false)
	if err := h.r.ask(context.Background(), func() {
		h.r.Send(operatorEvent(ownerPrepareRequest(0), 7))
		if err := b.SetAnnotationAllowed(context.Background(), true); err != nil {
			t.Error(err)
		}
	}); err != nil {
		t.Fatal(err)
	}
	if got := takeOperator(t, replies); got.Outcome != board.OperatorRefused || got.Error != board.ErrDisabled.Error() {
		t.Fatalf("queued initial OFF generation revived: %+v", got)
	}
}
