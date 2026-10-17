package couchmessage

import (
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

// queueOf admits ids to b in order, the first one becoming the delivery and
// the rest queuing behind it (pair#492).
func queueOf(t *testing.T, s ActorState, b Binding, now time.Time, ids ...string) (ActorState, []Effect) {
	t.Helper()
	var all []Effect
	for _, id := range ids {
		var fx []Effect
		s, fx = advance(t, s, admitEvent(b, id, now))
		all = append(all, fx...)
	}
	return s, all
}

func TestAdmitBehindPendingQueuesInOrder(t *testing.T) {
	now := time.Unix(100, 0)
	b := binding("ops:0")
	s, fx := queueOf(t, registered(t, b, now), b, now, "m1", "m2", "m3")
	if got := mustPending(t, s); got.ID != "m1" {
		t.Fatalf("pending %s", got.ID)
	}
	if s.Queued() != 2 || s.Remaining() != InboundAllowance-3 {
		t.Fatalf("queued %d remaining %d", s.Queued(), s.Remaining())
	}
	if len(fx) != 3 || fx[1].Receipt.Status != Queued || fx[1].Receipt.Detail != "position 1" || fx[2].Receipt.Detail != "position 2" {
		t.Fatalf("receipts: %+v", fx)
	}
	if !fx[1].Receipt.Message.Deadline.Equal(admitEvent(b, "m2", now).Message.Deadline) {
		t.Fatalf("queued deadline %v", fx[1].Receipt.Message.Deadline)
	}
	// The same message again is a no-op, queued or pending.
	again, fx := advance(t, s, admitEvent(b, "m2", now))
	if len(fx) != 0 || again.Queued() != 2 || again.Remaining() != s.Remaining() {
		t.Fatal("readmitting a queued message spent or emitted")
	}
}

func TestPromoteTakesHeadWithFreshPasteBy(t *testing.T) {
	now := time.Unix(100, 0)
	b := binding("ops:0")
	s, _ := queueOf(t, registered(t, b, now), b, now, "m1", "m2", "m3")
	if _, _, err := Advance(s, Event{Kind: Promote, Binding: b, Message: Message{ID: "m2"}, At: now}); err == nil {
		t.Fatal("promoted behind a delivery in progress")
	}
	s, _ = advance(t, s, Event{Kind: DeliveryStarted, Binding: b, Message: mustPending(t, s), At: now})
	s, _ = advance(t, s, Event{Kind: DeliveryFinished, Binding: b, Message: mustPending(t, s), Status: Submitted, At: now})
	if _, _, err := Advance(s, Event{Kind: Promote, Binding: b, Message: Message{ID: "m3"}, At: now}); err == nil {
		t.Fatal("promoted a message that is not the head")
	}
	at := now.Add(time.Minute)
	s, fx := advance(t, s, Event{Kind: Promote, Binding: b, Message: Message{ID: "m2"}, At: at})
	m := mustPending(t, s)
	if m.ID != "m2" || !m.Deadline.Equal(at.Add(PasteTimeout)) || s.Queued() != 1 {
		t.Fatalf("promoted %+v queued %d", m, s.Queued())
	}
	if len(fx) != 2 || fx[0].Receipt.Message != m || fx[0].Receipt.Status != Queued || fx[0].Receipt.Detail != "" || fx[1].Receipt.Message.ID != "m3" || fx[1].Receipt.Detail != "position 1" {
		t.Fatalf("receipts: %+v", fx)
	}
	if head, ok := s.QueueHead(); !ok || head.ID != "m3" {
		t.Fatalf("head %+v", head)
	}
}

func TestQueuedMessageSurvivesConnectedWait(t *testing.T) {
	now := time.Unix(100, 0)
	b := binding("ops:0")
	s, _ := queueOf(t, registered(t, b, now), b, now, "m1", "m2")
	s, fx := advance(t, s, Event{Kind: Tick, At: now.Add(2 * time.Hour)})
	if len(fx) != 0 || s.Queued() != 1 {
		t.Fatalf("connected queue expired: %+v queued %d", fx, s.Queued())
	}
}

func TestWaitingReceiptKeepsPositionWithinDetailBound(t *testing.T) {
	now := time.Unix(100, 0)
	b := binding("ops:0")
	s, _ := queueOf(t, registered(t, b, now), b, now, "head", "waiting")
	r := Receipt{Message: s.queue[0], Status: Queued}
	head := Receipt{Message: mustPending(t, s), Status: Held, Detail: "waiting for empty composer: " + strings.Repeat("界", MaxReceiptDetailBytes)}
	got := s.waitingReceipt(r, head)
	if got.Message != r.Message || got.Status != r.Status || len(got.Detail) > MaxReceiptDetailBytes || !utf8.ValidString(got.Detail) || !strings.HasPrefix(got.Detail, "position 2; waiting for empty composer") {
		t.Fatalf("bounded waiting receipt: %+v", got)
	}
}

func TestQueueFullRefuses(t *testing.T) {
	now := time.Unix(100, 0)
	b := binding("ops:0")
	s, _, _ := Advance(ActorState{}, Event{Kind: Register, Binding: b, Allowance: HubAllowance, At: now})
	ids := []string{"pending"}
	for i := 0; i < MaxQueued; i++ {
		ids = append(ids, "q"+strconv.Itoa(i))
	}
	s, _ = queueOf(t, s, b, now, ids...)
	if _, _, err := Advance(s, admitEvent(b, "over", now)); !errors.Is(err, ErrRecipientBusy) {
		t.Fatalf("past MaxQueued: %v", err)
	}
}

// The TL's ask: a queued message never vanishes; a recipient going away
// cancels it with words the sender can act on.
func TestRecipientLeavingCancelsQueueWithResend(t *testing.T) {
	now := time.Unix(100, 0)
	b := binding("ops:0")
	replacement := b
	replacement.Start = "new"
	for name, e := range map[string]Event{
		"disconnect": {Kind: Disconnect, Binding: b, At: now},
		"relaunch":   {Kind: Register, Binding: replacement, At: now},
	} {
		s, _ := queueOf(t, registered(t, b, now), b, now, "m1", "m2")
		s, fx := advance(t, s, e)
		var r *Receipt
		for i := range fx {
			if fx[i].Receipt.Message.ID == "m2" {
				r = &fx[i].Receipt
			}
		}
		if r == nil || r.Status != Cancelled || r.Detail != "ops:0 disconnected (relaunched or exited) before delivery; resend" || s.Queued() != 0 {
			t.Fatalf("%s: %+v", name, fx)
		}
	}
}

func TestCancelRemovesOneQueuedMessage(t *testing.T) {
	now := time.Unix(100, 0)
	b := binding("ops:0")
	s, _ := queueOf(t, registered(t, b, now), b, now, "m1", "m2", "m3")
	s, fx := advance(t, s, Event{Kind: Cancel, Binding: b, Message: Message{ID: "m2"}, Detail: "sender disconnected", At: now})
	if len(fx) != 2 || fx[0].Receipt.Status != Cancelled || fx[0].Receipt.Message.ID != "m2" || fx[1].Receipt.Detail != "position 1" || s.Queued() != 1 {
		t.Fatalf("cancel: %+v", fx)
	}
	if _, fx = advance(t, s, Event{Kind: Cancel, Binding: b, Message: Message{ID: "gone"}, At: now}); len(fx) != 0 {
		t.Fatal("cancelling an unknown message emitted")
	}
}
