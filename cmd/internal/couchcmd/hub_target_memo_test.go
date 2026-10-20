package couchcmd

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func TestHubTargetMemoWarmFreshExpiryAndSingleFlight(t *testing.T) {
	var calls atomic.Int64
	release := make(chan struct{})
	slot := couchcore.HubSlot{Repository: "/repo", Number: 6}
	now := time.Unix(1000, 0)
	var clock sync.Mutex
	m := newHubTargetMemo(t.Context(), func(ctx context.Context, ref string) (couchcore.HubSlot, error) {
		calls.Add(1)
		select {
		case <-release:
		case <-ctx.Done():
			return couchcore.HubSlot{}, ctx.Err()
		}
		return slot, nil
	}, func() time.Time { clock.Lock(); defer clock.Unlock(); return now })

	// Cold: two concurrent admissions share one resolution and refuse as
	// warming when their budget ends first.
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()
	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i := range errs {
		wg.Add(1)
		go func(i int) { defer wg.Done(); _, errs[i] = m.Warm(ctx, "pair:6") }(i)
	}
	wg.Wait()
	for _, err := range errs {
		if !errors.Is(err, errSlotCatalogWarming) {
			t.Fatalf("cold admission: %v", err)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("resolutions = %d, want one single flight", calls.Load())
	}
	// The resolution finishes after the refusal it caused and fills the memo.
	close(release)
	if got, err := m.Warm(t.Context(), "pair:6"); err != nil || got != slot {
		t.Fatalf("after warming: %v %v", got, err)
	}
	before := calls.Load()
	if got, err := m.Warm(t.Context(), "pair:6"); err != nil || got != slot || calls.Load() != before {
		t.Fatalf("warm read resolved again: %v %v calls %d", got, err, calls.Load())
	}
	// Expiry: an old entry resolves again.
	clock.Lock()
	now = now.Add(hubTargetMemoAge)
	clock.Unlock()
	if _, err := m.Warm(t.Context(), "pair:6"); err != nil || calls.Load() != before+1 {
		t.Fatalf("expired entry: %v calls %d", err, calls.Load())
	}
	// Fresh always resolves and refreshes the entry.
	if _, err := m.Fresh(t.Context(), "pair:6"); err != nil || calls.Load() != before+2 {
		t.Fatalf("fresh: %v calls %d", err, calls.Load())
	}
}

func TestHubTargetMemoResolutionEndsWithTheLifetime(t *testing.T) {
	lifetime, cancel := context.WithCancel(context.Background())
	m := newHubTargetMemo(lifetime, func(ctx context.Context, _ string) (couchcore.HubSlot, error) {
		<-ctx.Done()
		return couchcore.HubSlot{}, ctx.Err()
	}, nil)
	short, stop := context.WithTimeout(context.Background(), time.Millisecond)
	defer stop()
	if _, err := m.Warm(short, "pair:6"); !errors.Is(err, errSlotCatalogWarming) {
		t.Fatalf("cold: %v", err)
	}
	cancel()
	done := make(chan struct{})
	go func() { m.wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("an outstanding resolution outlived the supervisor lifetime")
	}
}

// TestHubTargetMemoRemembersALateFailure (pair#726 BR-1): a cold resolution
// that fails after admission's budget answers the next admissions with its
// own error, then retries once its failure ages out.
func TestHubTargetMemoRemembersALateFailure(t *testing.T) {
	var calls atomic.Int64
	release := make(chan struct{})
	now := time.Unix(1000, 0)
	var clock sync.Mutex
	m := newHubTargetMemo(t.Context(), func(ctx context.Context, _ string) (couchcore.HubSlot, error) {
		if calls.Add(1) == 1 {
			<-release
		}
		return couchcore.HubSlot{}, errors.New("sdlc workspace: not a slot")
	}, func() time.Time { clock.Lock(); defer clock.Unlock(); return now })
	short, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
	defer cancel()
	if _, err := m.Warm(short, "pair:6"); !errors.Is(err, errSlotCatalogWarming) {
		t.Fatalf("cold: %v", err)
	}
	close(release)
	deadline := time.Now().Add(5 * time.Second)
	var err error
	for time.Now().Before(deadline) {
		if _, err = m.Warm(t.Context(), "pair:6"); err != nil && !errors.Is(err, errSlotCatalogWarming) {
			break
		}
	}
	if err == nil || !strings.Contains(err.Error(), "not a slot") || calls.Load() != 1 {
		t.Fatalf("after the late failure: %v (calls %d)", err, calls.Load())
	}
	if _, err = m.Warm(t.Context(), "pair:6"); !strings.Contains(err.Error(), "not a slot") || calls.Load() != 1 {
		t.Fatalf("remembered failure: %v (calls %d)", err, calls.Load())
	}
	clock.Lock()
	now = now.Add(hubTargetFailureAge)
	clock.Unlock()
	if _, err = m.Warm(t.Context(), "pair:6"); err == nil || calls.Load() != 2 {
		t.Fatalf("aged failure did not retry: %v (calls %d)", err, calls.Load())
	}
}

func TestHubTargetMemoKeysCanonicallyAndCapsFlights(t *testing.T) {
	block := make(chan struct{})
	defer close(block)
	var calls atomic.Int64
	m := newHubTargetMemo(t.Context(), func(ctx context.Context, ref string) (couchcore.HubSlot, error) {
		calls.Add(1)
		select {
		case <-block:
		case <-ctx.Done():
		}
		return couchcore.HubSlot{}, ctx.Err()
	}, nil)
	short, cancel := context.WithTimeout(t.Context(), 5*time.Millisecond)
	defer cancel()
	for i := 0; i < hubTargetMaxFlights+4; i++ {
		_, _ = m.Warm(short, fmt.Sprintf("pair:%d", i))
	}
	m.mu.Lock()
	flights := len(m.inflight)
	m.mu.Unlock()
	if flights != hubTargetMaxFlights {
		t.Fatalf("background resolutions in flight = %d, want the cap %d", flights, hubTargetMaxFlights)
	}
	for deadline := time.Now().Add(2 * time.Second); calls.Load() < hubTargetMaxFlights && time.Now().Before(deadline); {
		time.Sleep(time.Millisecond)
	}
	if n := calls.Load(); n != hubTargetMaxFlights {
		t.Fatalf("resolutions started = %d, want the cap %d", n, hubTargetMaxFlights)
	}
	if hubTargetKey("pair:6") != hubTargetKey("pair:6") || hubTargetKey("pair:06") == hubTargetKey("pair:6") {
		t.Fatal("key canonicalization")
	}
}
