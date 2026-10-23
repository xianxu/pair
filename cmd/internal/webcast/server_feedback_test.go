package webcast

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast/board"
)

func nextFeedback(t *testing.T, stream *sse) board.LiveFeedbackMask {
	t.Helper()
	for {
		kind, text := stream.next()
		if kind != "feedback" {
			continue
		}
		var mask board.LiveFeedbackMask
		if err := json.Unmarshal([]byte(text), &mask); err != nil {
			t.Fatal(err)
		}
		return mask
	}
}

func TestLiveFeedbackStreamStartsWithCurrentMaskAndReconnects(t *testing.T) {
	c := newBoardCast(t)
	viewer := openSSE(t, c.viewURL+"events")
	initial := nextFeedback(t, viewer)
	if initial.Scope == "" || len(initial.Notes) != 0 || len(initial.Sketches) != 0 {
		t.Fatalf("initial mask=%+v", initial)
	}
	if code, text := post(t, c.url+"note", "application/json", noteBody(1, c.p.ID, 0)); code != 204 {
		t.Fatalf("note=%d %s", code, text)
	}
	selection, err := c.bd.Selection()
	if err != nil {
		t.Fatal(err)
	}
	if len(selection.Notes) != 1 {
		t.Fatal(selection.Notes)
	}
	retained := c.hub.Retained()
	if err = c.bd.SetDeliveredFeedback(nil, selection.Notes); err != nil {
		t.Fatal(err)
	}
	current := nextFeedback(t, viewer)
	if current.Scope != initial.Scope || current.Revision <= initial.Revision || len(current.Notes) != 1 {
		t.Fatalf("update=%+v initial=%+v", current, initial)
	}
	reconnected := nextFeedback(t, openSSE(t, c.url+"events"))
	if !reflect.DeepEqual(current, reconnected) {
		t.Fatalf("reconnect lost current mask: %+v != %+v", reconnected, current)
	}
	// Display metadata must never enter the recorded catch-up.
	if c.hub.Retained() != retained {
		t.Fatal("live mask entered recorded catch-up")
	}
}

func TestFeedbackMaskModuleIsServedToViewers(t *testing.T) {
	c := newBoardCast(t)
	r, err := castClient.Get(c.viewURL + "feedback-mask.js")
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	if r.StatusCode != 200 {
		t.Fatalf("feedback module status=%d", r.StatusCode)
	}
}

func TestLiveFeedbackPrecedesRecordedCatchUp(t *testing.T) {
	for i := 0; i < 20; i++ {
		c := newBoardCast(t)
		stream := openSSE(t, c.viewURL+"events")
		kind, _ := stream.next()
		stream.resp.Body.Close()
		if kind != "feedback" {
			t.Fatalf("first event=%q: live catch-up could paint before the confirmed mask", kind)
		}
	}
}
