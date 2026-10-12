package couchmessage

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/testwait"
)

// queueBroker is a broker with a hub (ops:0, the hub's allowance) and the
// given senders registered on fake wrappers.
func queueBroker(t *testing.T, now time.Time, senders ...string) (*Broker, Binding, *FakeDeliveryEndpoint, map[string]Binding) {
	t.Helper()
	b := NewBroker(context.Background(), func() time.Time { return now }, func(context.Context, Binding) (bool, error) { return true, nil })
	t.Cleanup(func() { _ = b.Close() })
	hub, fh := brokerBinding("ops:0"), newFakeEndpoint(now.Add(-time.Minute))
	if err := b.RegisterWithAllowance(hub, fh, HubAllowance); err != nil {
		t.Fatal(err)
	}
	bindings := map[string]Binding{}
	for _, slot := range senders {
		bindings[slot] = brokerBinding(slot)
		if err := b.Register(bindings[slot], newFakeEndpoint(now.Add(-time.Minute))); err != nil {
			t.Fatal(err)
		}
	}
	return b, hub, fh, bindings
}

func delivered(t *testing.T, f *FakeDeliveryEndpoint) Message {
	t.Helper()
	select {
	case m := <-f.delivered:
		return m
	case <-testwait.After(t):
		t.Fatal("nothing delivered")
	}
	return Message{}
}

// The 10-12 case (pair#492): the hub holds one message behind its draft while
// three slots report. Each is queued, not refused, and all three arrive in the
// order sent once the hold ends.
func TestHubQueuesSeveralSendersAndDeliversInOrder(t *testing.T) {
	now := time.Date(2026, 10, 12, 10, 47, 0, 0, time.UTC)
	b, _, fh, from := queueBroker(t, now, "pair:0", "pair:1", "pair:2", "pair:3")
	if _, err := b.Send(context.Background(), from["pair:0"], "held", Route{Target: "ops:0"}, "first"); err != nil {
		t.Fatal(err)
	}
	delivered(t, fh)
	fh.interim <- Receipt{Status: Held, Detail: "draft in the composer"}
	for i, slot := range []string{"pair:1", "pair:2", "pair:3"} {
		r, err := b.Send(context.Background(), from[slot], "report-"+slot, Route{Target: "ops:0"}, "plan from "+slot)
		if err != nil || r.Status != Queued || r.Detail != "position "+string(rune('1'+i)) {
			t.Fatalf("%s: %+v %v", slot, r, err)
		}
	}
	fh.outcomes <- Receipt{Status: Submitted}
	for _, slot := range []string{"pair:1", "pair:2", "pair:3"} {
		m := delivered(t, fh)
		if m.From != from[slot] {
			t.Fatalf("delivered %s, want %s", m.From.Slot, slot)
		}
		fh.outcomes <- Receipt{Status: Submitted}
	}
	for _, slot := range []string{"pair:1", "pair:2", "pair:3"} {
		waitStatus(t, b, from[slot], "report-"+slot, Submitted)
	}
}

// busyEndpoint refuses reservations while busy is set, as a wrapper does
// while the operator types (its sequence moves) or a delivery releases.
type busyEndpoint struct {
	*FakeDeliveryEndpoint
	mu       sync.Mutex
	busy     bool
	refusals int
}

func (e *busyEndpoint) Reserve(ctx context.Context, id string, seq uint64) error {
	e.mu.Lock()
	busy := e.busy
	if busy {
		e.refusals++
	}
	e.mu.Unlock()
	if busy {
		return ErrRecipientBusy
	}
	return e.FakeDeliveryEndpoint.Reserve(ctx, id, seq)
}

// A wrapper that refuses the reservation holds the message in the queue; it
// is delivered once the wrapper settles, never refused.
func TestPromotionHoldsWhileWrapperBusy(t *testing.T) {
	now := time.Date(2026, 10, 12, 11, 0, 0, 0, time.UTC)
	b := NewBroker(context.Background(), func() time.Time { return now }, func(context.Context, Binding) (bool, error) { return true, nil })
	defer b.Close()
	sender, hub := brokerBinding("pair:1"), brokerBinding("ops:0")
	wrapper := &busyEndpoint{FakeDeliveryEndpoint: newFakeEndpoint(now.Add(-time.Minute)), busy: true}
	if err := b.Register(hub, wrapper); err != nil {
		t.Fatal(err)
	}
	if err := b.Register(sender, newFakeEndpoint(now.Add(-time.Minute))); err != nil {
		t.Fatal(err)
	}
	r, err := b.Send(context.Background(), sender, "typed-over", Route{Target: "ops:0"}, "report")
	if err != nil || r.Status != Queued || r.Detail != "position 1" {
		t.Fatalf("send to a busy wrapper: %+v %v", r, err)
	}
	testwait.Until(t, "two refusals", func() bool { wrapper.mu.Lock(); defer wrapper.mu.Unlock(); return wrapper.refusals >= 2 })
	if got, _ := b.Status(sender, "typed-over"); got.Status != Queued {
		t.Fatalf("held message changed to %s", got.Status)
	}
	wrapper.mu.Lock()
	wrapper.busy = false
	wrapper.mu.Unlock()
	if m := delivered(t, wrapper.FakeDeliveryEndpoint); m.ID != "typed-over" {
		t.Fatal(m.ID)
	}
}

// The TL's ask: a queued message whose recipient goes away ends with a
// receipt a sender can act on; one whose sender goes away is cancelled too.
func TestQueuedMessagesEndWithActionableReceipts(t *testing.T) {
	now := time.Date(2026, 10, 12, 11, 0, 0, 0, time.UTC)
	b, hub, fh, from := queueBroker(t, now, "pair:0", "pair:1", "pair:2")
	if _, err := b.Send(context.Background(), from["pair:0"], "first", Route{Target: "ops:0"}, "first"); err != nil {
		t.Fatal(err)
	}
	delivered(t, fh)
	for _, slot := range []string{"pair:1", "pair:2"} {
		if _, err := b.Send(context.Background(), from[slot], "q-"+slot, Route{Target: "ops:0"}, "report"); err != nil {
			t.Fatal(err)
		}
	}
	b.Disconnect(from["pair:2"])
	if r, err := b.Status(hub, "q-pair:2"); err != nil || r.Status != Cancelled || r.Detail != "sender disconnected before delivery" {
		t.Fatalf("sender left: %+v %v", r, err)
	}
	b.Disconnect(hub)
	r, err := b.Status(from["pair:1"], "q-pair:1")
	if err != nil || r.Status != Cancelled || r.Detail != "ops:0 disconnected (relaunched or exited) before delivery; resend" {
		t.Fatalf("recipient left: %+v %v", r, err)
	}
}

func waitStatus(t *testing.T, b *Broker, caller Binding, id string, want Status) {
	t.Helper()
	testwait.Until(t, id+" "+string(want), func() bool { r, err := b.Status(caller, id); return err == nil && r.Status == want })
}
