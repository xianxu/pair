package couchtty

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func TestTitleActivityPassSharesScheduleAndWrapperEvidence(t *testing.T) {
	clock := &testClock{now: time.Date(2026, 10, 17, 0, 0, 0, 0, time.UTC)}
	probe := newFakeActivityProbe()
	f := activityFixtureWith(t, probe, clock)
	waitFor(t, "initial activity", func() bool { return probe.callCount() >= 2 })
	before := probe.callCount()
	var calls atomic.Int32
	var sawWork atomic.Bool
	native := clock.Now().Add(-time.Hour)
	work := clock.Now()
	f.con.SetWrapperActivity(func(row couchcore.ActionableThreadSummary) time.Time {
		if row.Address == f.attached.Address {
			return work
		}
		return time.Time{}
	})
	f.con.SetActivityPass(func(ctx context.Context, rows []couchcore.ActionableThreadSummary, workSnapshot map[couchcore.ThreadAddress]time.Time) (map[couchcore.ThreadAddress]time.Time, map[couchcore.ThreadAddress]bool) {
		calls.Add(1)
		sawWork.Store(workSnapshot[f.attached.Address].Equal(work))
		// Holding Console.mu here would deadlock: a display pass must run outside it.
		f.con.mu.Lock()
		f.con.mu.Unlock()
		observed := map[couchcore.ThreadAddress]time.Time{}
		for _, r := range rows {
			observed[r.Address] = native
		}
		return observed, map[couchcore.ThreadAddress]bool{}
	})
	waitFor(t, "combined display activity", func() bool { return calls.Load() > 0 && sawWork.Load() && f.activity()[f.attached.Address].Equal(work) })
	if got := probe.callCount(); got > before+2 {
		t.Fatalf("legacy activity scan continued beside display pass: %d -> %d", before, got)
	}
}

func TestTitleActivityPassCancelsReplacedInventory(t *testing.T) {
	clock := &testClock{now: idleNow}
	probe := newFakeActivityProbe()
	f := activityFixtureWith(t, probe, clock)
	waitFor(t, "initial inventory", func() bool { return probe.callCount() >= 2 })
	entered, cancelled := make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	f.con.SetActivityPass(func(ctx context.Context, rows []couchcore.ActionableThreadSummary, workSnapshot map[couchcore.ThreadAddress]time.Time) (map[couchcore.ThreadAddress]time.Time, map[couchcore.ThreadAddress]bool) {
		if calls.Add(1) == 1 {
			close(entered)
			<-ctx.Done()
			close(cancelled)
			return map[couchcore.ThreadAddress]time.Time{f.primary.Address: idleNow}, nil
		}
		return nil, map[couchcore.ThreadAddress]bool{f.attached.Address: true}
	})
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("pass never began")
	}
	primary := f.primary
	primary.State = couchcore.ThreadParked
	f.mu.Lock()
	f.inventory = func() []couchcore.ActionableThreadSummary {
		return []couchcore.ActionableThreadSummary{primary, f.attached, f.parked}
	}
	f.mu.Unlock()
	f.con.requestMenuRefresh()
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("obsolete pass was not cancelled by inventory")
	}
	waitFor(t, "replacement activity pass", func() bool { return calls.Load() >= 2 })
	if _, ok := f.activity()[f.primary.Address]; ok {
		t.Fatal("late cancelled completion restored departed activity")
	}
}

func TestTitleTargetSnapshotDoesNotMutateOwner(t *testing.T) {
	rows := []couchcore.ActionableThreadSummary{{TitleTarget: &couchcore.TitleTarget{Session: "pair-original", Sessions: map[string]bool{"pair-original": true}}}}
	snapshot := cloneResourceRows(rows)
	snapshot[0].TitleTarget.Session = "pair-other"
	delete(snapshot[0].TitleTarget.Sessions, "pair-original")
	if rows[0].TitleTarget.Session != "pair-original" || !rows[0].TitleTarget.Sessions["pair-original"] {
		t.Fatal("snapshot mutated private display authority")
	}
}
