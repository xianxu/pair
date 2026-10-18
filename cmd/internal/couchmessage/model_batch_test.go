package couchmessage

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

func actorBatchFixture(t *testing.T) (ActorState, Message, []Message, time.Time) {
	t.Helper()
	now := time.Unix(100, 0)
	b := binding("ops:0")
	s, _ := queueOf(t, registered(t, b, now), b, now, "head", "two", "three", "four", "five", "later")
	head := mustPending(t, s)
	s, _ = advance(t, s, Event{Kind: DeliveryStarted, Binding: b, Message: head, At: now})
	at := now.Add(45 * time.Minute)
	group := append([]Message{head}, s.queue[:4]...)
	for i := 1; i < len(group); i++ {
		group[i].Deadline = at.Add(PasteTimeout)
	}
	return s, head, group, at
}

func TestActorBatchOneConfirmationFinishesEveryOriginalReceipt(t *testing.T) {
	s, head, group, at := actorBatchFixture(t)
	remaining := s.RemainingAt(at)
	s, fx := advance(t, s, Event{Kind: FreezeBatch, Binding: head.To, Message: head, Messages: group, At: at})
	if s.Queued() != 1 || s.Remaining() != remaining || !reflect.DeepEqual(s.PendingMessages(), group) {
		t.Fatalf("freeze changed queue/budget/group: queued=%d budget=%d group=%v", s.Queued(), s.Remaining(), s.PendingMessages())
	}
	if len(fx) != 6 {
		t.Fatalf("freeze effects=%d, want five members and remaining queue position", len(fx))
	}
	for _, status := range []Status{Held, Queued, Delivering} {
		var progress []Effect
		s, progress = advance(t, s, Event{Kind: DeliveryProgress, Binding: head.To, Message: head, Status: status, Detail: "group evidence", At: at})
		if len(progress) != len(group) {
			t.Fatalf("progress count=%d", len(progress))
		}
		for i, e := range progress {
			if e.Receipt.Message != group[i] || e.Receipt.Status != status {
				t.Fatalf("member progress=%+v", e.Receipt)
			}
		}
	}
	s, fx = advance(t, s, Event{Kind: DeliveryFinished, Binding: head.To, Message: head, Status: Submitted, At: at})
	if len(fx) != len(group) || len(s.PendingMessages()) != 0 || s.Queued() != 1 {
		t.Fatalf("completion lost group or queue: %+v", fx)
	}
	for i, e := range fx {
		if e.Receipt.Message != group[i] || e.Receipt.Status != Submitted {
			t.Fatalf("original receipt=%+v", e.Receipt)
		}
	}
	_, late := advance(t, s, Event{Kind: DeliveryFinished, Binding: head.To, Message: head, Status: Cancelled, At: at})
	if len(late) != 0 {
		t.Fatal("late callback emitted receipts")
	}
}

func TestActorBatchRefusesChangedOrNonPrefixMembership(t *testing.T) {
	mutations := []func([]Message) []Message{
		func(g []Message) []Message { return g[1:] },
		func(g []Message) []Message { g[1], g[2] = g[2], g[1]; return g },
		func(g []Message) []Message { g[1].Body += "changed"; return g },
		func(g []Message) []Message { g[1].AcceptedAt = time.Unix(9, 0); return g },
		func(g []Message) []Message { g[1] = g[0]; return g },
		func(g []Message) []Message { g[1].Deadline = time.Unix(1, 0); return g },
	}
	for i, mutate := range mutations {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			s, head, group, at := actorBatchFixture(t)
			before := s
			got, fx, err := Advance(s, Event{Kind: FreezeBatch, Binding: head.To, Message: head, Messages: mutate(group), At: at})
			if err == nil || len(fx) != 0 || !reflect.DeepEqual(got, before) {
				t.Fatalf("invalid freeze changed actor: err=%v effects=%v", err, fx)
			}
		})
	}
}

func TestActorBatchKeepsBudgetAndPositionsAcrossGeneratedPrefixes(t *testing.T) {
	for count := 1; count <= 5; count++ {
		for _, outcome := range []Status{Submitted, Cancelled, Indeterminate, Expired} {
			s, head, group, at := actorBatchFixture(t)
			group = group[:count]
			s, _ = advance(t, s, Event{Kind: FreezeBatch, Binding: head.To, Message: head, Messages: group, At: at})
			if _, _, err := Advance(s, Event{Kind: FreezeBatch, Binding: head.To, Message: head, Messages: group, At: at}); err == nil {
				t.Fatal("membership froze twice")
			}
			before := s.Remaining()
			for _, m := range group {
				s, fx := advance(t, s, Event{Kind: Admit, Binding: head.To, Message: m, At: time.Unix(100, 0)})
				if s.Remaining() != before || len(fx) != 0 {
					t.Fatal("duplicate member admission spent allowance")
				}
			}
			q := s.queue[0]
			projected := s.waitingReceipt(Receipt{Message: q, Status: Queued}, Receipt{Message: head, Status: Held, Detail: "waiting for empty composer"})
			want := fmt.Sprintf("position %d; waiting for empty composer", count+1)
			if projected.Detail != want {
				t.Fatalf("position=%q want=%q", projected.Detail, want)
			}
			_, fx := advance(t, s, Event{Kind: DeliveryFinished, Binding: head.To, Message: head, Status: outcome, At: at})
			if len(fx) != count {
				t.Fatalf("outcome count=%d want=%d", len(fx), count)
			}
		}
	}
}

func TestActorBatchDisconnectEndsGroupAndUnjoinedQueueSeparately(t *testing.T) {
	s, head, group, at := actorBatchFixture(t)
	s, _ = advance(t, s, Event{Kind: FreezeBatch, Binding: head.To, Message: head, Messages: group, At: at})
	s, fx := advance(t, s, Event{Kind: Disconnect, Binding: head.To, At: at})
	if len(fx) != 6 || s.Connected() || len(s.PendingMessages()) != 0 || s.Queued() != 0 {
		t.Fatalf("disconnect=%+v", fx)
	}
	for i, e := range fx {
		want := Indeterminate
		if i == 5 {
			want = Cancelled
		}
		if e.Receipt.Status != want {
			t.Fatalf("member %d status=%s want=%s", i, e.Receipt.Status, want)
		}
	}
}
