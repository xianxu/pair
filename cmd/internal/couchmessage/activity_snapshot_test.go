package couchmessage

import (
	"context"
	"testing"
	"time"
)

type snapshotEndpoint struct {
	*FakeDeliveryEndpoint
	t *testing.T
}

func (e snapshotEndpoint) Observe(context.Context) (Observation, error) {
	e.t.Fatal("snapshot probed wrapper")
	return Observation{}, nil
}

func TestActivitySnapshotUsesConnectedCachedActivityWithoutIO(t *testing.T) {
	now := time.Unix(1000, 0)
	b := NewBroker(context.Background(), func() time.Time { return now }, func(context.Context, Binding) (bool, error) {
		t.Fatal("snapshot probed resting state")
		return false, nil
	})
	defer b.Close()
	b.SetFamilies(func(context.Context) (map[string]string, error) {
		t.Fatal("snapshot read families")
		return nil, nil
	})
	observed, local, disconnected := brokerBinding("pair:0"), brokerBinding("pair:1"), brokerBinding("pair:2")
	for _, binding := range []Binding{observed, local, disconnected} {
		if err := b.Register(binding, snapshotEndpoint{FakeDeliveryEndpoint: newFakeEndpoint(now), t: t}); err != nil {
			t.Fatal(err)
		}
	}
	if err := b.ReconcileObservation(observed, Observation{LastActivity: now.Add(time.Minute)}); err != nil {
		t.Fatal(err)
	}
	if err := b.ReconcileObservation(local, Observation{LastActivity: now.Add(-time.Minute)}); err != nil {
		t.Fatal(err)
	}
	b.OperatorInput(local)
	b.Disconnect(disconnected)
	got := b.ActivitySnapshot()
	if len(got) != 2 || !got[observed].Equal(now.Add(time.Minute)) || !got[local].Equal(now) {
		t.Fatalf("cached activity = %v", got)
	}
	delete(got, observed)
	if _, ok := b.ActivitySnapshot()[observed]; !ok {
		t.Fatal("snapshot exposed mutable broker state")
	}
}
