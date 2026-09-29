package couchtty

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

// fakeActivityProbe is the stateful stand-in for threadactivity.Latest
// (pair#247): a reply table keyed by thread, a failure switch, and a call log so
// a test can see exactly which threads a pass probed.
type fakeActivityProbe struct {
	mu      sync.Mutex
	replies map[couchcore.ThreadAddress]time.Time
	failing map[couchcore.ThreadAddress]bool
	calls   []couchcore.ThreadAddress
}

func newFakeActivityProbe() *fakeActivityProbe {
	return &fakeActivityProbe{replies: map[couchcore.ThreadAddress]time.Time{}, failing: map[couchcore.ThreadAddress]bool{}}
}

func (f *fakeActivityProbe) probe(_ context.Context, row couchcore.ActionableThreadSummary) (time.Time, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, row.Address)
	if f.failing[row.Address] {
		return time.Time{}, errors.New("session store unreadable")
	}
	return f.replies[row.Address], nil
}

func (f *fakeActivityProbe) set(address couchcore.ThreadAddress, at time.Time, fail bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.replies[address], f.failing[address] = at, fail
}

func (f *fakeActivityProbe) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

func (f *fakeActivityProbe) probed() []couchcore.ThreadAddress {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]couchcore.ThreadAddress(nil), f.calls...)
}

// testClock is the console's injected clock.
type testClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *testClock) Now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.now }
func (c *testClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

type activityFixture struct {
	*consoleFixture
	clock     *testClock
	probe     *fakeActivityProbe
	primary   couchcore.ActionableThreadSummary
	attached  couchcore.ActionableThreadSummary
	parked    couchcore.ActionableThreadSummary
	inventory func() []couchcore.ActionableThreadSummary
	mu        sync.Mutex
}

// activityFixtureWith runs a console whose attached child is slot :1 of pair,
// beside its live primary and one parked thread. The ticker is an hour, so
// every pass in these tests is one the console asked for itself.
func activityFixtureWith(t *testing.T, probe *fakeActivityProbe, clock *testClock) *activityFixture {
	t.Helper()
	f := &activityFixture{clock: clock, probe: probe}
	f.consoleFixture = newFixtureBeforeRun(t, 24, 100, func(c *Console) {
		c.activityInterval = time.Hour
		c.now = clock.Now
	})
	f.con.mu.Lock()
	address := f.con.panes[f.con.order[0]].thread
	f.con.mu.Unlock()
	f.primary, f.attached = groupedRow("/workspace/pair", 0, "primary"), groupedRow("/workspace/pair", 1, "one")
	f.attached.Address = address
	f.parked = groupedRow("/workspace/ducks", 0, "ducks")
	f.parked.State = couchcore.ThreadParked
	rows := []couchcore.ActionableThreadSummary{f.primary, f.attached, f.parked}
	f.inventory = func() []couchcore.ActionableThreadSummary { return rows }
	// The probe goes in BEFORE any inventory, so its own first pass sees no live
	// threads: the pass that finds them is the one the landing inventory asks
	// for, not the hour-long ticker.
	f.con.SetActivityProbe(probe.probe)
	f.con.SetActionableProvider(func(context.Context, []couchcore.LiveTTYObservation) ([]couchcore.ActionableThreadSummary, error) {
		f.mu.Lock()
		defer f.mu.Unlock()
		return f.inventory(), nil
	})
	return f
}

func (f *activityFixture) activity() map[couchcore.ThreadAddress]time.Time {
	return f.con.menuSnapshot().Activity
}

// chipIdle is the level the tab bar's model gives the attached thread.
func (f *activityFixture) chipIdle(t *testing.T) IdleLevel {
	t.Helper()
	f.con.mu.Lock()
	defer f.con.mu.Unlock()
	for _, actor := range f.con.statusModelLocked().Actors {
		if actor.Thread == f.attached.Address {
			return actor.Idle
		}
	}
	t.Fatal("the attached thread has no chip")
	return 0
}

// switcherIdle is the level the switcher draws for the attached thread, read
// back from the same state and clock the console renders with.
func (f *activityFixture) switcherIdle() IdleLevel {
	state := f.con.menuSnapshot()
	at, known := state.Activity[f.attached.Address]
	return IdleLevelFor(f.clock.Now(), at, known)
}

func TestActivityPassProbesLiveThreadsOnceAndFadesBothViews(t *testing.T) {
	clock := &testClock{now: idleNow}
	probe := newFakeActivityProbe()
	f := activityFixtureWith(t, probe, clock)
	probe.set(f.attached.Address, idleNow.Add(-30*time.Hour), false)
	probe.set(f.primary.Address, idleNow.Add(-time.Hour), false)
	probe.set(f.parked.Address, idleNow.Add(-9*24*time.Hour), false)

	waitFor(t, "a pass after the inventory lands", func() bool { return len(f.activity()) == 2 })
	if _, ok := f.activity()[f.parked.Address]; ok {
		t.Fatal("a parked thread was given live activity")
	}
	for _, address := range probe.probed() {
		if address == f.parked.Address {
			t.Fatal("the pass probed a thread that is not live")
		}
	}
	if seen := map[couchcore.ThreadAddress]int{}; func() bool {
		for _, a := range probe.probed() {
			seen[a]++
		}
		return seen[f.attached.Address] != 1 || seen[f.primary.Address] != 1
	}() {
		t.Fatalf("each live thread must be probed once per pass: %v", probe.probed())
	}
	if got := f.chipIdle(t); got != IdleDay {
		t.Fatalf("tab bar level = %d, want IdleDay for 30h of silence", got)
	}
	if got := f.switcherIdle(); got != IdleDay {
		t.Fatalf("switcher level = %d, want the tab bar's IdleDay", got)
	}

	// Time passes with no new activity: the next pass moves the fade on.
	clock.advance(48 * time.Hour)
	before := probe.callCount()
	f.con.requestActivity()
	waitFor(t, "the next pass", func() bool { return probe.callCount() >= before+2 })
	waitFor(t, "the chip crosses into the stale band", func() bool { return f.chipIdle(t) == IdleStale })
	if got := f.switcherIdle(); got != IdleStale {
		t.Fatalf("switcher level = %d, want IdleStale", got)
	}
}

// A failed probe keeps the last value: a stale thread must not flash bright.
func TestActivityPassKeepsTheLastValueWhenAProbeFails(t *testing.T) {
	clock := &testClock{now: idleNow}
	probe := newFakeActivityProbe()
	f := activityFixtureWith(t, probe, clock)
	probe.set(f.attached.Address, idleNow.Add(-4*24*time.Hour), false)
	waitFor(t, "first pass", func() bool { return f.chipIdle(t) == IdleStale })

	probe.set(f.attached.Address, time.Time{}, true)
	// The same pass observes the primary at a new time: seeing that time is the
	// proof this pass landed, so the assertion below is about its merge.
	marker := idleNow.Add(-7 * time.Minute)
	probe.set(f.primary.Address, marker, false)
	f.con.requestActivity()
	waitFor(t, "the failing pass to land", func() bool { return f.activity()[f.primary.Address].Equal(marker) })
	if got := f.chipIdle(t); got != IdleStale {
		t.Fatalf("level after a failed probe = %d, want the kept IdleStale", got)
	}
}

// A thread that stops being live drops out on the next pass (ARCH-FUNERAL):
// nothing is kept for threads the pass no longer probes.
func TestActivityPassDropsAThreadThatIsNoLongerLive(t *testing.T) {
	clock := &testClock{now: idleNow}
	probe := newFakeActivityProbe()
	f := activityFixtureWith(t, probe, clock)
	probe.set(f.primary.Address, idleNow.Add(-2*24*time.Hour), false)
	probe.set(f.attached.Address, idleNow.Add(-2*24*time.Hour), false)
	waitFor(t, "first pass", func() bool { return len(f.activity()) == 2 })

	primary := f.primary
	primary.State = couchcore.ThreadParked
	f.mu.Lock()
	f.inventory = func() []couchcore.ActionableThreadSummary {
		return []couchcore.ActionableThreadSummary{primary, f.attached, f.parked}
	}
	f.mu.Unlock()
	f.con.requestMenuRefresh()
	waitFor(t, "the primary drops out", func() bool {
		_, still := f.activity()[f.primary.Address]
		return !still && len(f.activity()) == 1
	})
}

// Restart: nothing is carried in memory. A fresh console over the same sources
// reaches the same fade from its first pass.
func TestActivitySurvivesACouchRestartThroughItsSources(t *testing.T) {
	for run := 0; run < 2; run++ {
		clock := &testClock{now: idleNow}
		probe := newFakeActivityProbe()
		f := activityFixtureWith(t, probe, clock)
		probe.set(f.attached.Address, idleNow.Add(-4*24*time.Hour), false)
		waitFor(t, "the first pass", func() bool { return len(f.activity()) >= 1 })
		if got := f.chipIdle(t); got != IdleStale {
			t.Fatalf("run %d: level = %d, want IdleStale from the sources alone", run, got)
		}
	}
}
