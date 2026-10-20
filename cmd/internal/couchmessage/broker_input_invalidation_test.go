package couchmessage

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// The hook runs after the wrapper accepted a reservation, before the broker
// validates its own observation. It makes the input/admission race explicit.
type inputInvalidationEndpoint struct {
	*FakeDeliveryEndpoint
	once       sync.Once
	after      func()
	deliveries atomic.Int32
}

func (e *inputInvalidationEndpoint) Reserve(ctx context.Context, id string, sequence uint64) error {
	if err := e.FakeDeliveryEndpoint.Reserve(ctx, id, sequence); err != nil {
		return err
	}
	e.once.Do(e.after)
	return nil
}

func (e *inputInvalidationEndpoint) Deliver(ctx context.Context, message Message, progress func(Receipt)) (Receipt, error) {
	e.deliveries.Add(1)
	return e.FakeDeliveryEndpoint.Deliver(ctx, message, progress)
}

func TestBrokerInputDuringReservationQueuesLiveRecipient(t *testing.T) {
	now := time.Unix(1000, 0)
	b := NewBroker(t.Context(), func() time.Time { return now }, nil)
	defer b.Close()
	sender, hub := brokerBinding("pair:7"), brokerBinding("ops:0")
	endpoint := &inputInvalidationEndpoint{FakeDeliveryEndpoint: newFakeEndpoint(now)}
	endpoint.after = func() {
		b.OperatorInput(hub)
		for _, binding := range []Binding{sender, hub} {
			if got, err := b.Caller(binding.Scope, binding.Tag, binding.Session, binding.Nonce); err != nil || got != binding {
				t.Fatalf("input invalidated a live registration: got %+v, err %v", got, err)
			}
		}
	}
	for binding, wrapper := range map[Binding]DeliveryEndpoint{sender: newFakeEndpoint(now), hub: endpoint} {
		if err := b.Register(binding, wrapper); err != nil {
			t.Fatal(err)
		}
	}
	receipt, err := b.Send(t.Context(), sender, "input-race", Route{Target: hub.Slot}, "report")
	if err != nil || receipt.Status != Queued {
		t.Fatalf("live recipient input must queue, not lose registration: receipt %+v, err %v", receipt, err)
	}
	message := delivered(t, endpoint.FakeDeliveryEndpoint)
	if message.ID != "input-race" || message.From != sender || message.To != hub {
		t.Fatalf("wrong queued message: %+v", message)
	}
	endpoint.outcomes <- Receipt{Status: Submitted}
	waitStatus(t, b, sender, message.ID, Submitted)
	repeated, err := b.Send(t.Context(), sender, message.ID, Route{Target: hub.Slot}, "report")
	if err != nil || repeated.Status != Submitted {
		t.Fatalf("repeat did not retain submitted receipt: %+v, %v", repeated, err)
	}
	if count := endpoint.deliveries.Load(); count != 1 {
		t.Fatalf("delivered the same message %d times", count)
	}
}

func TestBrokerReservationRejectsLostRegistration(t *testing.T) {
	for _, change := range []string{"sender-disconnect", "recipient-disconnect", "same-binding-reconnect"} {
		t.Run(change, func(t *testing.T) {
			now := time.Unix(1000, 0)
			b := NewBroker(t.Context(), func() time.Time { return now }, nil)
			defer b.Close()
			sender, hub := brokerBinding("pair:7"), brokerBinding("ops:0")
			endpoint := &inputInvalidationEndpoint{FakeDeliveryEndpoint: newFakeEndpoint(now)}
			replacement := &inputInvalidationEndpoint{FakeDeliveryEndpoint: newFakeEndpoint(now), after: func() {}}
			endpoint.after = func() {
				switch change {
				case "sender-disconnect":
					b.Disconnect(sender)
				case "recipient-disconnect":
					b.Disconnect(hub)
				case "same-binding-reconnect":
					b.Disconnect(hub)
					if err := b.Register(hub, replacement); err != nil {
						t.Fatal(err)
					}
					if got, err := b.Caller(hub.Scope, hub.Tag, hub.Session, hub.Nonce); err != nil || got != hub {
						t.Fatalf("replacement was not registered: %+v, %v", got, err)
					}
				}
			}
			for binding, wrapper := range map[Binding]DeliveryEndpoint{sender: newFakeEndpoint(now), hub: endpoint} {
				if err := b.Register(binding, wrapper); err != nil {
					t.Fatal(err)
				}
			}
			if receipt, err := b.Send(t.Context(), sender, "stale-reservation", Route{Target: hub.Slot}, "report"); !errors.Is(err, ErrUnavailable) {
				t.Fatalf("stale work was not refused: receipt %+v, err %v", receipt, err)
			}
			if old, next := endpoint.deliveries.Load(), replacement.deliveries.Load(); old != 0 || next != 0 {
				t.Fatalf("stale work delivered: old %d, replacement %d", old, next)
			}
		})
	}
}
