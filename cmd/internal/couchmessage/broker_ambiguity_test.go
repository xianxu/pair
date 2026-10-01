package couchmessage

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

type routingEffectsEndpoint struct {
	*FakeDeliveryEndpoint
	reservations atomic.Int32
	deliveries   atomic.Int32
}

func (f *routingEffectsEndpoint) Reserve(ctx context.Context, id string, sequence uint64) error {
	f.reservations.Add(1)
	return f.FakeDeliveryEndpoint.Reserve(ctx, id, sequence)
}
func (f *routingEffectsEndpoint) Deliver(ctx context.Context, m Message) (Receipt, error) {
	f.deliveries.Add(1)
	return f.FakeDeliveryEndpoint.Deliver(ctx, m)
}

func TestBrokerFamilyAmbiguityIncludesSender(t *testing.T) {
	for _, tc := range []struct {
		name       string
		other      bool
		repository string
		want       error
	}{
		{"sender-is-only-evidence-of-conflicting-repository", true, "/other/.git", ErrAmbiguous},
		{"same-repository-selects-other-slot", true, "/repo/.git", nil},
		{"sender-only-is-ineligible", false, "", ErrNoRecipient},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base := time.Unix(1000, 0)
			var nanos atomic.Int64
			nanos.Store(base.UnixNano())
			b := NewBroker(context.Background(), func() time.Time { return time.Unix(0, nanos.Load()) }, func(context.Context, Binding) (bool, error) { return true, nil })
			defer b.Close()
			from, to := brokerBinding("pair:0"), brokerBinding("pair:1")
			to.Repository = tc.repository
			sender := &routingEffectsEndpoint{FakeDeliveryEndpoint: newFakeEndpoint(base)}
			recipient := &routingEffectsEndpoint{FakeDeliveryEndpoint: newFakeEndpoint(base)}
			if err := b.Register(from, sender); err != nil {
				t.Fatal(err)
			}
			if tc.other {
				if err := b.Register(to, recipient); err != nil {
					t.Fatal(err)
				}
			}
			nanos.Store(base.Add(time.Minute).UnixNano())
			r, err := b.Send(context.Background(), from, "family-id", "pair", "work")
			if !errors.Is(err, tc.want) {
				t.Errorf("family admission: recipient=%+v err=%v want=%v", r.Message.To, err, tc.want)
			}
			if tc.want == nil {
				if r.Message.To != to {
					t.Fatalf("selected sender or changed recipient: %+v", r)
				}
				select {
				case m := <-recipient.delivered:
					if m.To != to || m.From != from {
						t.Fatalf("wrong delivery: %+v", m)
					}
				case <-time.After(time.Second):
					t.Fatal("normal family delivery did not start")
				}
			} else {
				if recipient.reservations.Load() != 0 || recipient.deliveries.Load() != 0 {
					t.Errorf("refused family performed effects: reservations=%d deliveries=%d", recipient.reservations.Load(), recipient.deliveries.Load())
				}
				b.mu.Lock()
				for _, a := range b.actors {
					if a.reservation != "" {
						t.Errorf("retained reservation %q", a.reservation)
					}
					if _, pending := a.state.Pending(); pending {
						t.Error("refused family admitted a message")
					}
				}
				b.mu.Unlock()
			}
			if sender.reservations.Load() != 0 || sender.deliveries.Load() != 0 {
				t.Errorf("sender became eligible: reservations=%d deliveries=%d", sender.reservations.Load(), sender.deliveries.Load())
			}
		})
	}
}
