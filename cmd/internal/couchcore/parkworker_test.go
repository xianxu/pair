package couchcore

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/pairlifecycletest"
)

func TestParkWorkerBoundsAndCoalesces(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var mu sync.Mutex
	calls := 0
	worker := newParkWorker(1)
	address := ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-0123456789abcdef"}
	work := func(context.Context) (ParkResult, error) {
		mu.Lock()
		calls++
		mu.Unlock()
		close(started)
		<-release
		return ParkResult{}, nil
	}
	first, err := worker.Submit(context.Background(), address, "park-nonce", work)
	if err != nil {
		t.Fatal(err)
	}
	<-started
	duplicate, err := worker.Submit(context.Background(), address, "park-nonce", work)
	if err != nil || duplicate != first {
		t.Fatalf("duplicate = %p, %v; first=%p", duplicate, err, first)
	}
	other := ThreadAddress{RepoScope: address.RepoScope, Tag: "couch-fedcba9876543210"}
	if _, err := worker.Submit(context.Background(), other, "other", func(context.Context) (ParkResult, error) {
		t.Fatal("overloaded work executed")
		return ParkResult{}, nil
	}); !errors.Is(err, ErrParkWorkerOverloaded) {
		t.Fatalf("overload err = %v", err)
	}
	close(release)
	if _, err := first.Await(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := duplicate.Await(context.Background()); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if calls != 1 {
		t.Fatalf("coalesced calls = %d", calls)
	}
}

type sequenceClock struct {
	mu    sync.Mutex
	times []time.Time
	last  time.Time
}

func (c *sequenceClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.times) == 0 {
		return c.last
	}
	c.last = c.times[0]
	c.times = c.times[1:]
	return c.last
}

func TestParkWorkerBeginDeadlineHasZeroExternalEffectsAtOneSecond(t *testing.T) {
	for _, test := range []struct {
		name       string
		elapsed    time.Duration
		wantErr    bool
		wantEffect int
	}{
		{name: "999ms commits before deadline", elapsed: 999 * time.Millisecond, wantEffect: 1},
		{name: "exactly 1s refuses", elapsed: time.Second, wantErr: true, wantEffect: 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			store, _, thread := createControllerThread(t)
			start := time.Unix(100, 0).UTC()
			clock := &sequenceClock{times: []time.Time{start, start.Add(test.elapsed)}}
			lifecycle := &fakeControllerLifecycle{model: pairlifecycletest.New(start)}
			artifacts := NewFakeThreadArtifactCollisionChecker()
			artifacts.SetPairSession(thread.Address, "pair-exact", true)
			controller := PairLifecycleController{
				Threads: store, DataDir: t.TempDir(), Lifecycle: lifecycle, Sessions: artifacts,
				Clock: clock, Nonce: func() (string, error) { return "park-deadline", nil },
			}
			result, err := controller.Park(context.Background(), thread.Address)
			if (err != nil) != test.wantErr {
				t.Fatalf("Park err = %v, wantErr=%v", err, test.wantErr)
			}
			if got := len(artifacts.TriggeredQuits()); got != test.wantEffect {
				t.Fatalf("trigger effects = %d, want %d", got, test.wantEffect)
			}
			if got := len(lifecycle.trace); got != test.wantEffect*3 {
				t.Fatalf("lifecycle effects = %v", lifecycle.trace)
			}
			persisted, getErr := store.GetThread(thread.Address)
			if getErr != nil || persisted.Park == nil || len(persisted.Incarnations) != 1 {
				t.Fatalf("deadline occupancy = %+v, %v", persisted, getErr)
			}
			if test.wantErr && (!result.SoftTargetMissed || persisted.Park.Phase != ParkRequested) {
				t.Fatalf("deadline result = %+v, thread=%+v", result, persisted)
			}
		})
	}
}
