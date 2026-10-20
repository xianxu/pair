package couchmessage

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// The hook runs after the wrapper accepted a reservation, before the broker
// validates its own observation. It makes the input/admission race explicit.
type inputInvalidationEndpoint struct {
	*FakeDeliveryEndpoint
	once         sync.Once
	after        func()
	releaseOnce  sync.Once
	afterRelease func()
	deliveries   atomic.Int32
}

func (e *inputInvalidationEndpoint) Release(ctx context.Context, id string) error {
	err := e.FakeDeliveryEndpoint.Release(ctx, id)
	if e.afterRelease != nil {
		e.releaseOnce.Do(e.afterRelease)
	}
	return err
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

func TestBrokerSenderReconnectCannotAuthorizeOldAdmission(t *testing.T) {
	for _, phase := range []string{"catalog", "reserve", "release"} {
		t.Run(phase, func(t *testing.T) {
			now := time.Unix(1000, 0)
			b := NewBroker(t.Context(), func() time.Time { return now }, nil)
			defer b.Close()
			sender, hub := brokerBinding("pair:7"), brokerBinding("ops:0")
			endpoint := &inputInvalidationEndpoint{FakeDeliveryEndpoint: newFakeEndpoint(now), after: func() { b.OperatorInput(hub) }}
			if err := b.Register(sender, newFakeEndpoint(now)); err != nil {
				t.Fatal(err)
			}
			if err := b.Register(hub, endpoint); err != nil {
				t.Fatal(err)
			}
			reconnect := func() {
				b.Disconnect(sender)
				if err := b.Register(sender, newFakeEndpoint(now)); err != nil {
					t.Fatal(err)
				}
				for _, binding := range []Binding{sender, hub} {
					if got, err := b.Caller(binding.Scope, binding.Tag, binding.Session, binding.Nonce); err != nil || got != binding {
						t.Fatalf("current Caller must survive actor replacement: %+v, %v", got, err)
					}
				}
			}
			switch phase {
			case "catalog":
				var once sync.Once
				b.SetFamilies(func(context.Context) (map[string]string, error) { once.Do(reconnect); return nil, nil })
			case "reserve":
				endpoint.after = func() { b.OperatorInput(hub); reconnect() }
			case "release":
				endpoint.afterRelease = reconnect
			}
			receipt, err := b.Send(t.Context(), sender, "old-sender", Route{Target: hub.Slot}, "report")
			if !errors.Is(err, ErrUnavailable) {
				t.Errorf("reconnected sender authorized an old admission: receipt %+v, err %v", receipt, err)
			} else if !strings.HasPrefix(err.Error(), "caller pair:7 current") {
				t.Errorf("sender lifetime refusal lost side/phase: %v", err)
			}
			if count := endpoint.deliveries.Load(); count != 0 {
				t.Errorf("stale sender delivered %d messages", count)
			}
		})
	}
}

func TestBrokerAdmissionLossNamesSideAndPhase(t *testing.T) {
	for _, phase := range []string{"entry", "catalog", "release-caller", "release-recipient"} {
		t.Run(phase, func(t *testing.T) {
			now := time.Unix(1000, 0)
			b := NewBroker(t.Context(), func() time.Time { return now }, nil)
			defer b.Close()
			sender, hub := brokerBinding("pair:7"), brokerBinding("ops:0")
			endpoint := &inputInvalidationEndpoint{FakeDeliveryEndpoint: newFakeEndpoint(now), after: func() { b.OperatorInput(hub) }}
			if err := b.Register(sender, newFakeEndpoint(now)); err != nil {
				t.Fatal(err)
			}
			if err := b.Register(hub, endpoint); err != nil {
				t.Fatal(err)
			}
			prefix := "caller pair:7 current"
			switch phase {
			case "entry":
				b.Disconnect(sender)
				prefix = "caller lookup"
			case "catalog":
				var once sync.Once
				b.SetFamilies(func(context.Context) (map[string]string, error) {
					once.Do(func() { b.Disconnect(sender) })
					return nil, nil
				})
			case "release-caller":
				endpoint.afterRelease = func() { b.Disconnect(sender) }
			case "release-recipient":
				endpoint.afterRelease = func() { b.Disconnect(hub) }
				prefix = "recipient ops:0 admission"
			}
			receipt, err := b.Send(t.Context(), sender, "lost-admission", Route{Target: hub.Slot}, "report")
			if !errors.Is(err, ErrUnavailable) || !strings.HasPrefix(err.Error(), prefix) {
				t.Errorf("loss must preserve unavailable and name %q: receipt %+v, err %v", prefix, receipt, err)
			}
			if count := endpoint.deliveries.Load(); count != 0 {
				t.Errorf("lost registration delivered %d messages", count)
			}
		})
	}
}
