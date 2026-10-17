package couchmessage

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"
)

// Read the wire field too: acceptance time must survive the same JSON
// representation used by receipts and wrapper delivery requests.
func acceptedAtOnWire(t *testing.T, m Message) time.Time {
	t.Helper()
	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	var wire struct{ AcceptedAt time.Time }
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatal(err)
	}
	return wire.AcceptedAt
}

func TestBrokerAcceptanceTimeSurvivesQueuedPromotion(t *testing.T) {
	base := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	var elapsed atomic.Int64
	now := func() time.Time { return base.Add(time.Duration(elapsed.Load()) * time.Second) }
	b := NewBroker(context.Background(), now, nil)
	t.Cleanup(func() { _ = b.Close() })
	hub, from := brokerBinding("ops:0"), brokerBinding("pair:1")
	wrapper := newFakeEndpoint(base.Add(-time.Minute))
	if err := b.Register(hub, wrapper); err != nil {
		t.Fatal(err)
	}
	if err := b.Register(from, newFakeEndpoint(base.Add(-time.Minute))); err != nil {
		t.Fatal(err)
	}
	direct, err := b.Send(context.Background(), from, "direct", Route{Target: hub.Slot}, "first")
	if err != nil {
		t.Fatal(err)
	}
	if got := acceptedAtOnWire(t, direct.Message); !got.Equal(base) {
		t.Errorf("direct receipt accepted at %s, want %s", got, base)
	}
	first := delivered(t, wrapper)
	if got := acceptedAtOnWire(t, first); !got.Equal(base) {
		t.Errorf("direct delivery accepted at %s, want %s", got, base)
	}
	elapsed.Store(10)
	queued, err := b.Send(context.Background(), from, "queued", Route{Target: hub.Slot}, "second")
	if err != nil || queued.Status != Queued || queued.Detail != "position 2; waiting for recognized composer" {
		t.Fatalf("queued receipt: %+v, %v", queued, err)
	}
	accepted := base.Add(10 * time.Second)
	if got := acceptedAtOnWire(t, queued.Message); !got.Equal(accepted) {
		t.Errorf("queued receipt accepted at %s, want %s", got, accepted)
	}
	elapsed.Store(20)
	wrapper.outcomes <- Receipt{Status: Submitted}
	promoted := delivered(t, wrapper)
	if promoted.ID != "queued" {
		t.Fatalf("promoted %s, want queued", promoted.ID)
	}
	if got := acceptedAtOnWire(t, promoted); !got.Equal(accepted) {
		t.Errorf("promoted delivery accepted at %s, want original %s", got, accepted)
	}
	if want := base.Add(20*time.Second + PasteTimeout); !promoted.Deadline.Equal(want) {
		t.Errorf("promotion deadline %s, want %s", promoted.Deadline, want)
	}
	wrapper.outcomes <- Receipt{Status: Submitted}
	waitStatus(t, b, from, "queued", Submitted)
	terminal, err := b.Status(from, "queued")
	if err != nil {
		t.Fatal(err)
	}
	if got := acceptedAtOnWire(t, terminal.Message); !got.Equal(accepted) {
		t.Errorf("terminal receipt accepted at %s, want original %s", got, accepted)
	}
}
