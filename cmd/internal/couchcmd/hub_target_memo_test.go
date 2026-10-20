package couchcmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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

// One repository discovery serves N and 2N different slot admissions. Prefixes
// and aliases share that flight; a missing primary still uses fresh resolution.
func TestHubTargetMemoRepositoryBatch(t *testing.T) {
	for _, size := range []int{4, 8} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			var discovers, fallback atomic.Int64
			started, release := make(chan struct{}), make(chan struct{})
			m := newHubTargetMemo(t.Context(), func(_ context.Context, ref string) (couchcore.HubSlot, error) {
				fallback.Add(1)
				return couchcore.HubSlot{Repository: "/repo", Number: 0}, nil
			}, nil)
			m.names = func(context.Context) ([]couchcore.RepositoryName, error) {
				return []couchcore.RepositoryName{{Key: "/repo", Dir: "repository", Alias: "repo"}}, nil
			}
			m.discover = func(ctx context.Context, primary string) (map[int]couchcore.HubSlot, error) {
				if primary != "/repo" {
					t.Errorf("primary %s", primary)
				}
				if discovers.Add(1) == 1 {
					close(started)
				}
				select {
				case <-release:
				case <-ctx.Done():
					return nil, ctx.Err()
				}
				slots := map[int]couchcore.HubSlot{}
				for n := 1; n <= size; n++ {
					slots[n] = couchcore.HubSlot{Repository: primary, Number: n}
				}
				return slots, nil
			}
			results := make(chan error, size)
			for n := 1; n <= size; n++ {
				go func(n int) {
					got, err := m.Warm(t.Context(), fmt.Sprintf("rep:%d", n))
					if err == nil && got.Number != n {
						err = fmt.Errorf("slot %v want %d", got, n)
					}
					results <- err
				}(n)
			}
			<-started
			close(release)
			for n := 0; n < size; n++ {
				if err := <-results; err != nil {
					t.Fatal(err)
				}
			}
			for n := 1; n <= size; n++ {
				if _, err := m.Warm(t.Context(), fmt.Sprintf("repo:%d", n)); err != nil {
					t.Fatal(err)
				}
			}
			if discovers.Load() != 1 || fallback.Load() != 0 {
				t.Fatalf("discover %d fallback %d", discovers.Load(), fallback.Load())
			}
			if _, err := m.Warm(t.Context(), "repo:0"); err != nil {
				t.Fatal(err)
			}
			if discovers.Load() != 1 || fallback.Load() != 1 {
				t.Fatalf("primary fallback: discover %d fallback %d", discovers.Load(), fallback.Load())
			}
		})
	}
}

func TestHubTargetMemoPrewarmAndSharedFlightCap(t *testing.T) {
	lifetime, cancel := context.WithCancel(t.Context())
	defer cancel()
	started := make(chan string, hubTargetMaxFlights)
	m := newHubTargetMemo(lifetime, func(ctx context.Context, ref string) (couchcore.HubSlot, error) {
		started <- ref
		<-ctx.Done()
		return couchcore.HubSlot{}, ctx.Err()
	}, nil)
	m.names = func(context.Context) ([]couchcore.RepositoryName, error) { return nil, nil }
	m.discover = func(ctx context.Context, primary string) (map[int]couchcore.HubSlot, error) {
		started <- primary
		<-ctx.Done()
		return nil, ctx.Err()
	}
	primaries := []string{"/repo", "/repo"}
	for n := 1; n < hubTargetMaxFlights; n++ {
		primaries = append(primaries, fmt.Sprintf("/repo%d", n))
	}
	m.Prewarm(primaries)
	for n := 0; n < hubTargetMaxFlights; n++ {
		select {
		case <-started:
		case <-time.After(5 * time.Second):
			t.Fatal("prewarm not started")
		}
	}
	if _, err := m.Warm(t.Context(), "other:1"); !errors.Is(err, errSlotCatalogWarming) {
		t.Fatalf("shared cap: %v", err)
	}
	cancel()
	m.wait()
	select {
	case extra := <-started:
		t.Fatalf("over cap: %s", extra)
	default:
	}
}

func TestHubTargetMemoRepositoryFailureExpiryAndFallback(t *testing.T) {
	now := time.Unix(1000, 0)
	var clock sync.Mutex
	var discoveries, fallback atomic.Int64
	m := newHubTargetMemo(t.Context(), func(context.Context, string) (couchcore.HubSlot, error) {
		fallback.Add(1)
		return couchcore.HubSlot{Repository: "/fresh", Number: 1}, nil
	}, func() time.Time { clock.Lock(); defer clock.Unlock(); return now })
	m.names = func(context.Context) ([]couchcore.RepositoryName, error) {
		return []couchcore.RepositoryName{{Key: "/repo", Dir: "repo"}, {Key: "/other", Dir: "other"}}, nil
	}
	m.discover = func(context.Context, string) (map[int]couchcore.HubSlot, error) {
		discoveries.Add(1)
		return nil, errors.New("repository probe failed")
	}
	for _, ref := range []string{"repo:1", "repo:2"} {
		if _, err := m.Warm(t.Context(), ref); err == nil || !strings.Contains(err.Error(), "repository probe failed") {
			t.Fatalf("%s: %v", ref, err)
		}
	}
	if discoveries.Load() != 1 || fallback.Load() != 0 {
		t.Fatalf("failure repeated or masked: discoveries %d fallback %d", discoveries.Load(), fallback.Load())
	}
	clock.Lock()
	now = now.Add(hubTargetFailureAge)
	clock.Unlock()
	_, _ = m.Warm(t.Context(), "repo:3")
	if discoveries.Load() != 2 {
		t.Fatalf("expired failure not retried: %d", discoveries.Load())
	}
	// Unknown/ambiguous references belong to the existing resolver, which can
	// retain sibling-directory precedence and supply the precise refusal.
	if _, err := m.Warm(t.Context(), "missing:1"); err != nil {
		t.Fatal(err)
	}
	if fallback.Load() != 1 {
		t.Fatalf("fallback %d", fallback.Load())
	}
}

func TestHubTargetMemoLateRepositoryCannotReplaceFreshTarget(t *testing.T) {
	var tick atomic.Int64
	started, release := make(chan struct{}), make(chan struct{})
	m := newHubTargetMemo(t.Context(), func(context.Context, string) (couchcore.HubSlot, error) {
		return couchcore.HubSlot{Repository: "/fresh", Number: 1}, nil
	}, func() time.Time { return time.Unix(1000, tick.Add(1)) })
	m.names = func(context.Context) ([]couchcore.RepositoryName, error) {
		return []couchcore.RepositoryName{{Key: "/repo", Dir: "repo"}}, nil
	}
	m.discover = func(context.Context, string) (map[int]couchcore.HubSlot, error) {
		close(started)
		<-release
		return map[int]couchcore.HubSlot{1: {Repository: "/old", Number: 1}}, nil
	}
	m.Prewarm([]string{"/repo"})
	<-started
	if _, err := m.Fresh(t.Context(), "repo:1"); err != nil {
		t.Fatal(err)
	}
	close(release)
	m.wait()
	if got, err := m.Warm(t.Context(), "repo:1"); err != nil || got.Repository != "/fresh" {
		t.Fatalf("late repository overwrote effect proof: %v %v", got, err)
	}
}

// BR-19: the same exact sibling wins over an enrolled prefix or alias for
// cold reads and for references already filled by startup prewarm.
func TestHubTargetMemoPreservesSiblingPrecedence(t *testing.T) {
	for _, name := range []string{"parley", "par", "blog"} {
		for _, prewarm := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/prewarm=%v", name, prewarm), func(t *testing.T) {
				fleet := t.TempDir()
				enrolled := filepath.Join(fleet, "parley.nvim")
				sibling := filepath.Join(fleet, name)
				if err := os.MkdirAll(sibling, 0700); err != nil {
					t.Fatal(err)
				}
				want := couchcore.HubSlot{Repository: sibling, Number: 1}
				r := &serviceRig{t: t, world: newMessageWorld(), slotGit: map[string]couchcore.SlotGitStatus{}}
				r.decorateAuthority = func(a messageAuthority) messageAuthority {
					a.hubTarget = func(context.Context, string) (couchcore.HubSlot, error) { return want, nil }
					a.hubRepositoryFleetRoot = fleet
					a.hubRepositoryNames = func(context.Context) ([]couchcore.RepositoryName, error) {
						return []couchcore.RepositoryName{{Key: enrolled, Dir: "parley.nvim", Alias: "blog"}}, nil
					}
					a.hubRepositorySlots = func(_ context.Context, primary string) (map[int]couchcore.HubSlot, error) {
						return map[int]couchcore.HubSlot{1: {Repository: primary, Number: 1}}, nil
					}
					return a
				}
				r.slotOps = newSlotOperations(nil, func(ctx context.Context) ([]couchcore.RepositoryName, error) {
					return r.s.authority.hubRepositoryNames(ctx)
				}, time.Now)
				r.init()
				m := r.s.hubTargets
				if prewarm {
					m.Prewarm([]string{enrolled})
					m.wait()
				}
				ref := name + ":1"
				got, err := m.Warm(t.Context(), ref)
				key, queued, queueErr := r.slotOps.queueTarget(t.Context(), ref)
				if queueErr != nil || queued.Key != sibling || key != "remote\x00"+sibling+":1" {
					t.Errorf("queue target %q %v %v, want sibling %s", key, queued, queueErr, sibling)
				}
				fresh, freshErr := m.Fresh(t.Context(), ref)
				if err != nil || freshErr != nil || got != fresh || got != want {
					t.Fatalf("warm %v %v, fresh %v %v, want sibling %v", got, err, fresh, freshErr, want)
				}
			})
		}
	}
}

func TestHubTargetMemoSiblingDoesNotNeedEnrollment(t *testing.T) {
	fleet := t.TempDir()
	primary := filepath.Join(fleet, "repo")
	if err := os.MkdirAll(primary, 0700); err != nil {
		t.Fatal(err)
	}
	want := couchcore.HubSlot{Repository: primary, Number: 1}
	m := newHubTargetMemo(t.Context(), func(context.Context, string) (couchcore.HubSlot, error) { return want, nil }, nil)
	m.fleetRoot = fleet
	m.names = func(context.Context) ([]couchcore.RepositoryName, error) {
		return nil, errors.New("enrollment unavailable")
	}
	m.discover = func(context.Context, string) (map[int]couchcore.HubSlot, error) {
		return map[int]couchcore.HubSlot{1: want}, nil
	}
	got, err := m.Warm(t.Context(), "repo:1")
	if err != nil || got != want {
		t.Fatalf("exact sibling depends on enrollment: %v %v", got, err)
	}
}
