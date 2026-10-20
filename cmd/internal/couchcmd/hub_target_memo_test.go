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

// TestHubTargetMemoOrdersByStartAndSweeps: an older resolution that finishes
// after a newer one never replaces it, and remembered entries are swept and
// capped (pair#726 review).
func TestHubTargetMemoOrdersByStartAndSweeps(t *testing.T) {
	now := time.Unix(1000, 0)
	var clock sync.Mutex
	tick := func() time.Time { clock.Lock(); defer clock.Unlock(); now = now.Add(time.Millisecond); return now }
	oldRelease, oldStarted := make(chan struct{}), make(chan struct{})
	var first atomic.Bool
	m := newHubTargetMemo(t.Context(), func(ctx context.Context, ref string) (couchcore.HubSlot, error) {
		if ref == "pair:6" && first.CompareAndSwap(false, true) {
			close(oldStarted)
			<-oldRelease
			return couchcore.HubSlot{Repository: "/old", Number: 6}, nil
		}
		return couchcore.HubSlot{Repository: "/new", Number: 6}, nil
	}, tick)
	done := make(chan struct{})
	go func() { defer close(done); _, _ = m.Fresh(t.Context(), "pair:6") }()
	<-oldStarted
	if _, err := m.Fresh(t.Context(), "pair:6"); err != nil {
		t.Fatal(err)
	}
	close(oldRelease)
	<-done
	if got, err := m.Warm(t.Context(), "pair:6"); err != nil || got.Repository != "/new" {
		t.Fatalf("an older resolution replaced a newer one: %v %v", got, err)
	}
	for i := 0; i < hubTargetMaxEntries+10; i++ {
		_, _ = m.Fresh(t.Context(), fmt.Sprintf("pair:%d", 100+i))
	}
	m.mu.Lock()
	n := len(m.entries)
	m.mu.Unlock()
	if n > hubTargetMaxEntries {
		t.Fatalf("entries = %d over the cap %d", n, hubTargetMaxEntries)
	}
	clock.Lock()
	now = now.Add(hubTargetMemoAge)
	clock.Unlock()
	_, _ = m.Fresh(t.Context(), "pair:1")
	m.mu.Lock()
	n = len(m.entries)
	m.mu.Unlock()
	if n != 1 {
		t.Fatalf("expired entries kept: %d", n)
	}
}

// TestHubTargetMemoAgesFailuresFromTheirEnd (M1 review round 3): a failed
// resolution slower than the failure age still answers the next admission
// with its own error, counted from when it finished.
func TestHubTargetMemoAgesFailuresFromTheirEnd(t *testing.T) {
	now := time.Unix(1000, 0)
	var clock sync.Mutex
	var calls atomic.Int64
	m := newHubTargetMemo(t.Context(), func(context.Context, string) (couchcore.HubSlot, error) {
		calls.Add(1)
		clock.Lock()
		now = now.Add(2 * hubTargetFailureAge) // a hung probe outlasts the failure age
		clock.Unlock()
		return couchcore.HubSlot{}, errors.New("sdlc workspace: probe timed out")
	}, func() time.Time { clock.Lock(); defer clock.Unlock(); return now })
	_, _ = m.Fresh(t.Context(), "pair:6")
	if _, err := m.Warm(t.Context(), "pair:6"); err == nil || !strings.Contains(err.Error(), "probe timed out") || calls.Load() != 1 {
		t.Fatalf("slow failure not answered from the memo: %v (resolutions %d)", err, calls.Load())
	}
}

// TestHubTargetMemoWarmCapsItsOwnWait: a caller passing an unbounded context
// still gets an answer within the admission budget.
func TestHubTargetMemoWarmCapsItsOwnWait(t *testing.T) {
	block := make(chan struct{})
	defer close(block)
	m := newHubTargetMemo(t.Context(), func(ctx context.Context, _ string) (couchcore.HubSlot, error) {
		select {
		case <-block:
		case <-ctx.Done():
		}
		return couchcore.HubSlot{}, ctx.Err()
	}, nil)
	result := make(chan error, 1)
	go func() { _, err := m.Warm(context.Background(), "pair:6"); result <- err }()
	select {
	case err := <-result:
		if !errors.Is(err, errSlotCatalogWarming) {
			t.Fatalf("unbounded caller: %v", err)
		}
	case <-time.After(20 * slotAdmissionBudget): // safety bound only; a pass never depends on it
		t.Fatal("Warm waited past its own admission budget for an unbounded caller")
	}
}
