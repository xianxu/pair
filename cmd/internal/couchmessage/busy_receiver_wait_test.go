package couchmessage

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/testwait"
)

// The receiver's busy interval is independent of wall-clock delivery time.
// An atomic clock lets the worker and receipt reads observe the same jumps.
type busyReceiverFixture struct {
	broker    *Broker
	clock     atomic.Int64
	base      time.Time
	from, to  Binding
	endpoint  *busyReceiverEndpoint
	messages  []Message
	committed bool
}

// The acknowledgement synchronizes with processing an interim receipt, not
// with a desired future status (Queued progress is unsupported before #629).
type busyReceiverEndpoint struct {
	*busyEndpoint
	progressApplied chan struct{}
}

func (e *busyReceiverEndpoint) Deliver(ctx context.Context, m Message, progress func(Receipt)) (Receipt, error) {
	return e.FakeDeliveryEndpoint.Deliver(ctx, m, func(r Receipt) {
		progress(r)
		e.progressApplied <- struct{}{}
	})
}

func newBusyReceiverFixture(t *testing.T, progress Status) *busyReceiverFixture {
	t.Helper()
	f := &busyReceiverFixture{base: time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC), from: brokerBinding("pair:1"), to: brokerBinding("ops:0"), committed: progress != ""}
	f.clock.Store(f.base.UnixNano())
	f.broker = NewBroker(context.Background(), func() time.Time { return time.Unix(0, f.clock.Load()) }, nil)
	t.Cleanup(func() { _ = f.broker.Close() })
	f.endpoint = &busyReceiverEndpoint{busyEndpoint: &busyEndpoint{FakeDeliveryEndpoint: newFakeEndpoint(f.base.Add(-time.Minute)), busy: !f.committed}, progressApplied: make(chan struct{}, 1)}
	for _, registration := range []struct {
		binding  Binding
		endpoint DeliveryEndpoint
	}{{f.from, newFakeEndpoint(f.base)}, {f.to, f.endpoint}} {
		if err := f.broker.Register(registration.binding, registration.endpoint); err != nil {
			t.Fatal(err)
		}
	}
	for i := 1; i <= 3; i++ {
		id, body := fmt.Sprintf("busy-%d", i), fmt.Sprintf("report %d", i)
		r, err := f.broker.Send(context.Background(), f.from, id, Route{Target: f.to.Slot}, body)
		if err != nil {
			t.Fatal(err)
		}
		f.messages = append(f.messages, r.Message)
		if i == 1 && f.committed {
			if m := delivered(t, f.endpoint.FakeDeliveryEndpoint); m.ID != id {
				t.Fatalf("first delivery = %s", m.ID)
			}
			f.endpoint.interim <- Receipt{Status: progress, Detail: "waiting for recognized composer"}
			select {
			case <-f.endpoint.progressApplied:
			case <-testwait.After(t):
				t.Fatal("receiver did not apply interim receipt")
			}
		}
	}
	return f
}

func assertBusyReceiverReceipt(t *testing.T, got Receipt, original Message, position int) bool {
	t.Helper()
	if got.Status.Terminal() {
		t.Errorf("%s became terminal during composer wait: %+v", original.ID, got)
		return false
	}
	if got.Status != Queued && got.Status != Held {
		t.Errorf("%s wait status = %s", original.ID, got.Status)
	}
	if got.Message.ID != original.ID || got.Message.Body != original.Body || got.Message.From != original.From || got.Message.To != original.To || !got.Message.AcceptedAt.Equal(original.AcceptedAt) {
		t.Errorf("receipt changed accepted message: got %+v, original %+v", got.Message, original)
	}
	for _, detail := range []string{"waiting for recognized composer", fmt.Sprintf("position %d", position)} {
		if !strings.Contains(got.Detail, detail) {
			t.Errorf("%s detail %q omits %q", original.ID, got.Detail, detail)
		}
	}
	return true
}

func TestBusyReceiverWaitSurvivesClockJumpsAndDrainsFIFO(t *testing.T) {
	for _, progress := range []Status{"", Queued, Held} {
		name := "reserve refused"
		if progress != "" {
			name = "committed " + string(progress)
		}
		for _, elapsed := range []time.Duration{91 * time.Second, 45 * time.Minute, time.Hour} {
			t.Run(name+"/"+elapsed.String(), func(t *testing.T) {
				f := newBusyReceiverFixture(t, progress)
				f.clock.Store(f.base.Add(elapsed).UnixNano())
				live := true
				for i, original := range f.messages {
					r, err := f.broker.Status(f.from, original.ID)
					if err != nil {
						t.Errorf("status: %v", err)
						live = false
					} else {
						live = assertBusyReceiverReceipt(t, r, original, i+1) && live
					}
					r, err = f.broker.Send(context.Background(), f.from, original.ID, Route{Target: f.to.Slot}, original.Body)
					if err != nil {
						t.Errorf("replay: %v", err)
						live = false
					} else {
						live = assertBusyReceiverReceipt(t, r, original, i+1) && live
					}
					r, known, _ := f.broker.ReceiptSnapshot(original.ID)
					if !known {
						t.Errorf("snapshot lost %s", original.ID)
						live = false
					} else {
						live = assertBusyReceiverReceipt(t, r, original, i+1) && live
					}
				}
				// If expiry already broke the contract, don't wait for an event
				// the existing implementation can no longer produce.
				if !live {
					return
				}
				f.endpoint.mu.Lock()
				f.endpoint.busy = false
				f.endpoint.mu.Unlock()
				for i, original := range f.messages {
					if i != 0 || !f.committed {
						m := delivered(t, f.endpoint.FakeDeliveryEndpoint)
						if m.ID != original.ID || m.Body != original.Body || !m.AcceptedAt.Equal(original.AcceptedAt) {
							t.Fatalf("FIFO delivery = %+v, want %+v", m, original)
						}
					}
					f.endpoint.outcomes <- Receipt{Status: Submitted}
					waitStatus(t, f.broker, f.from, original.ID, Submitted)
				}
				for _, original := range f.messages {
					r, err := f.broker.Send(context.Background(), f.from, original.ID, Route{Target: f.to.Slot}, original.Body)
					if err != nil || r.Status != Submitted {
						t.Fatalf("completed replay: %+v %v", r, err)
					}
				}
				select {
				case m := <-f.endpoint.delivered:
					t.Fatalf("duplicate delivery: %+v", m)
				default:
				}
			})
		}
	}
}

func TestBusyReceiverDoesNotBlockAnotherRecipient(t *testing.T) {
	f := newBusyReceiverFixture(t, "")
	other := brokerBinding("ops:1")
	ep := newFakeEndpoint(f.base)
	if err := f.broker.Register(other, ep); err != nil {
		t.Fatal(err)
	}
	if _, err := f.broker.Send(context.Background(), f.from, "independent", Route{Target: other.Slot}, "independent report"); err != nil {
		t.Fatal(err)
	}
	if m := delivered(t, ep); m.ID != "independent" {
		t.Fatal(m)
	}
	ep.outcomes <- Receipt{Status: Submitted}
	waitStatus(t, f.broker, f.from, "independent", Submitted)
	select {
	case m := <-f.endpoint.delivered:
		t.Fatalf("busy receiver accepted paste: %+v", m)
	default:
	}
}

func TestBusyReceiverDisconnectOrReplacementCancelsOldWork(t *testing.T) {
	for _, progress := range []Status{"", Queued, Held} {
		for _, replacement := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/replacement=%v", progress, replacement), func(t *testing.T) {
				f := newBusyReceiverFixture(t, progress)
				f.clock.Store(f.base.Add(2 * time.Hour).UnixNano())
				next := f.to
				next.Nonce = "replacement"
				ep := newFakeEndpoint(f.base)
				if !replacement {
					f.broker.Disconnect(f.to)
				}
				if err := f.broker.Register(next, ep); err != nil {
					t.Fatal(err)
				}
				for i, original := range f.messages {
					r, err := f.broker.Status(f.from, original.ID)
					want := Cancelled
					reason := f.to.Slot
					if i == 0 && f.committed {
						want = Indeterminate
						reason = "wrapper disconnected"
					}
					if err != nil || r.Status != want || !strings.Contains(r.Detail, reason) || !strings.Contains(r.Detail, "disconnected") {
						t.Errorf("receiver cancellation: %+v %v", r, err)
					}
					if r.Message.ID != original.ID || r.Message.Body != original.Body || r.Message.To != original.To || !r.Message.AcceptedAt.Equal(original.AcceptedAt) {
						t.Errorf("cancelled receipt changed message: %+v", r)
					}
				}
				// A fresh message is a FIFO barrier: old work must not be routed
				// into the new incarnation before or after it.
				if _, err := f.broker.Send(context.Background(), f.from, "fresh", Route{Target: next.Slot}, "new receiver"); err != nil {
					t.Fatal(err)
				}
				if m := delivered(t, ep); m.ID != "fresh" || m.To != next {
					t.Fatalf("replacement got old work: %+v", m)
				}
				ep.outcomes <- Receipt{Status: Submitted}
				waitStatus(t, f.broker, f.from, "fresh", Submitted)
				select {
				case m := <-ep.delivered:
					t.Fatalf("replacement got extra work: %+v", m)
				default:
				}
			})
		}
	}
}

func TestBusyReceiverSenderDisconnectPreservesHandoffUncertainty(t *testing.T) {
	for _, progress := range []Status{"", Queued, Held} {
		t.Run(string(progress), func(t *testing.T) {
			f := newBusyReceiverFixture(t, progress)
			f.broker.Disconnect(f.from)
			for i, m := range f.messages {
				r, known, _ := f.broker.ReceiptSnapshot(m.ID)
				want := Cancelled
				if i == 0 && f.committed {
					want = Indeterminate
				}
				if !known || r.Status != want || !strings.Contains(r.Detail, "sender disconnected") {
					t.Errorf("sender disconnect: receipt=%+v known=%v, want %s", r, known, want)
				}
			}
		})
	}
}
