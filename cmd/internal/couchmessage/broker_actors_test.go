package couchmessage

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// blockingEndpoint never answers Observe: a listing that probed it would hang
// until its context ended.
type blockingEndpoint struct{ *FakeDeliveryEndpoint }

func (blockingEndpoint) Observe(ctx context.Context) (Observation, error) {
	<-ctx.Done()
	return Observation{}, ctx.Err()
}

func TestActorsReadsMemoryWithoutProbing(t *testing.T) {
	base := time.Unix(1000, 0)
	var nanos atomic.Int64
	nanos.Store(base.UnixNano())
	now := func() time.Time { return time.Unix(0, nanos.Load()) }
	restingProbes := atomic.Int64{}
	b := NewBroker(context.Background(), now, func(_ context.Context, v Binding) (bool, error) {
		restingProbes.Add(1)
		return v.Slot != "pair:2", nil
	})
	defer b.Close()
	// The listing's resting state comes from an in-memory view (Couch's
	// slot-git cache), never from the git probe.
	b.SetRestingView(func(v Binding) (bool, bool) { return v.Slot != "pair:2", v.Slot != "pair:4" })
	from := brokerBinding("brain:0")
	slots := []Binding{brokerBinding("pair:1"), brokerBinding("pair:2"), brokerBinding("pair:3"), brokerBinding("pair:4")}
	if err := b.Register(from, newFakeEndpoint(base)); err != nil {
		t.Fatal(err)
	}
	for _, v := range slots {
		if err := b.Register(v, blockingEndpoint{newFakeEndpoint(base)}); err != nil {
			t.Fatal(err)
		}
	}
	// Pushed observations for pair:1, pair:2 and pair:4; pair:3 has none.
	for _, v := range []Binding{slots[0], slots[1], slots[3]} {
		if err := b.ReconcileObservation(v, Observation{LastActivity: base, Sequence: 1}); err != nil {
			t.Fatal(err)
		}
	}
	probesBefore := restingProbes.Load()
	listing := func() map[string]Candidate {
		t.Helper()
		start := time.Now()
		rows, err := b.Actors(context.Background(), from)
		if err != nil {
			t.Fatal(err)
		}
		if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
			t.Fatalf("listing probed: took %v", elapsed)
		}
		out := map[string]Candidate{}
		for _, r := range rows {
			out[r.Binding.Slot] = r
		}
		return out
	}
	rows := listing()
	if !rows["pair:1"].Known || !rows["pair:1"].Resting || !rows["pair:2"].Known || rows["pair:2"].Resting || rows["pair:3"].Known || rows["pair:4"].Known {
		t.Fatalf("rows %+v", rows)
	}
	if restingProbes.Load() != probesBefore {
		t.Fatal("listing ran resting probes")
	}
	// An idle wrapper pushes nothing; its connected observation stays current
	// (#365), where the heartbeat design aged it out after five seconds.
	nanos.Store(base.Add(time.Hour).UnixNano())
	if rows = listing(); !rows["pair:1"].Known {
		t.Fatal("idle connected actor became unknown")
	}
}
