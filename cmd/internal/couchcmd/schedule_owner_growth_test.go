package couchcmd

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchschedule"
	"github.com/xianxu/pair/cmd/internal/durablefile"
)

// Measure the complete persisted task rows, rather than only calls to a pure
// transition helper: every ledger publication encodes every configured row.
// A due sweep must grow with the catalogue, not multiply full-ledger work by
// the number of due tasks. This uses real discovery, the owner and native IO.
func TestScheduleOwnerDueTickLedgerWorkGrowsLinearly(t *testing.T) {
	measure := func(t *testing.T, taskCount int) ownerSweepWork {
		t.Helper()
		base, err := filepath.EvalSymlinks(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		root := filepath.Join(base, "hub")
		for i := 0; i < taskCount; i++ {
			dir := filepath.Join(root, "scripts", "scheduled", fmt.Sprintf("task-%02d", i))
			if err = os.MkdirAll(dir, 0700); err != nil {
				t.Fatal(err)
			}
			if err = os.WriteFile(filepath.Join(dir, "schedule"), []byte(`{"version":1,"cron":"* * * * *","timeout":"1m"}`), 0600); err != nil {
				t.Fatal(err)
			}
			if err = os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\nfunc main() {}\n"), 0600); err != nil {
				t.Fatal(err)
			}
		}
		now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
		coordinator := newScheduleCoordinator(filepath.Join(base, "namespace"), []string{root}, time.UTC, func() time.Time { return now })
		work, publications := 0, 0
		counting := false
		coordinator.store.BeforePersist = func() error {
			if counting {
				publications++
				work += taskCount
			}
			return nil
		}
		historySyncs := 0
		coordinator.runs.SyncDirectory = func(path string) error { historySyncs++; return durablefile.SyncDirectory(path) }
		owner := newScheduleOwner(coordinator)
		owner.configure(scheduleExecutionWorker{}, &resultScheduleNotices{})
		if err = owner.open(t.Context()); err != nil {
			t.Fatal(err)
		}
		defer owner.close()
		if err = owner.refresh(t.Context()); err != nil {
			t.Fatal(err)
		}
		before, err := coordinator.store.View(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if len(before.Tasks) != taskCount {
			t.Fatalf("discovered %d tasks, want %d", len(before.Tasks), taskCount)
		}
		for _, state := range before.Tasks {
			if state.NextDue.IsZero() {
				t.Fatal("discovery did not arm due time")
			}
			now = state.NextDue
			break
		}
		work, publications = 0, 0
		counting = true
		if err = owner.tick(t.Context(), newSchedulerClockFake(now)); err != nil {
			t.Fatal(err)
		}
		counting = false
		after, err := coordinator.store.View(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		for _, state := range after.Tasks {
			// Capacity may retain the committed reservation; only admitted
			// notice candidates require completed pending-history companions.
			if !state.NextDue.After(now) || state.Active == nil || !state.Active.DueAt.Equal(now) || (state.Active.Stage != "reserved" && (state.Last == nil || state.Last.Notice != "pending")) {
				t.Fatalf("due occurrence not consumed exactly at the slot: %+v", state)
			}
		}
		if jobs := len(owner.loop.notices); jobs != 2 {
			t.Fatalf("notice worker bound changed: got %d queued jobs, want 2", jobs)
		}
		if publications == 0 {
			t.Fatal("due tick did not persist authority")
		}
		t.Logf("tasks=%d ledger publications=%d encoded task rows=%d", taskCount, publications, work)
		return ownerSweepWork{work, historySyncs}
	}
	var small, large ownerSweepWork
	t.Run("8-tasks", func(t *testing.T) { small = measure(t, 8) })
	t.Run("16-tasks", func(t *testing.T) { large = measure(t, 16) })
	// The fixed pair of bounded notice jobs may add a small constant. Doubling
	// due tasks must not quadruple the rows passed through full-ledger encoding.
	assertOwnerSweepGrowth(t, small, large)
}

func TestScheduleOwnerReceiptSweepLedgerWorkGrowsLinearly(t *testing.T) {
	measure := func(t *testing.T, count int) ownerSweepWork {
		a, _, template, notices := scheduledDeliveryFixture(t)
		states := make([]couchschedule.TaskState, count)
		for i := range states {
			states[i] = template
			states[i].Key.Task = fmt.Sprintf("receipt-%02d", i)
			record := scheduledResultRecord(states[i], a.coordinator.now(), "wake")
			saveScheduledResult(t, a, record)
		}
		if err := updateScheduleFixture(a.coordinator.store, t.Context(), func(l *couchschedule.Ledger) error {
			l.Tasks = map[string]couchschedule.TaskState{}
			for _, state := range states {
				l.Tasks[state.Key.ID()] = state
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		publications := 0
		a.coordinator.store.BeforePersist = func() error { publications++; return nil }
		historySyncs := 0
		a.coordinator.runs.SyncDirectory = func(path string) error { historySyncs++; return durablefile.SyncDirectory(path) }
		owner := newScheduleOwner(a.coordinator)
		owner.configure(scheduleExecutionWorker{}, notices)
		owner.recovered = true
		if err := owner.open(t.Context()); err != nil {
			t.Fatal(err)
		}
		defer owner.close()
		if err := owner.tick(t.Context(), newSchedulerClockFake(a.coordinator.now())); err != nil {
			t.Fatal(err)
		}
		ledger, err := a.coordinator.store.View(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		for _, prior := range states {
			state := ledger.Tasks[prior.Key.ID()]
			if state.Active != nil || state.Last == nil || state.Last.ID != prior.Active.ID || state.Last.Notice != "submitted" || !state.NextDue.Equal(prior.NextDue) {
				t.Fatalf("receipt not settled without changing occurrence/due: %+v", state)
			}
			record, output, err := a.history().Read(t.Context(), scheduleRunKey(prior))
			if err != nil || record.Notice != "submitted" || string(output) != "saved execution output" {
				t.Fatalf("terminal history missing: %+v %q %v", record, output, err)
			}
		}
		if notices.admissions != 0 || len(owner.loop.notices) != 0 || len(owner.loop.execution) != 0 {
			t.Fatal("receipt sweep replayed effects")
		}
		if publications == 0 {
			t.Fatal("receipt sweep did not commit")
		}
		work := publications * count
		t.Logf("tasks=%d ledger publications=%d encoded task rows=%d", count, publications, work)
		return ownerSweepWork{work, historySyncs}
	}
	var small, large ownerSweepWork
	t.Run("8-tasks", func(t *testing.T) { small = measure(t, 8) })
	t.Run("16-tasks", func(t *testing.T) { large = measure(t, 16) })
	assertOwnerSweepGrowth(t, small, large)
}

// Recovery uses only committed unknown facts, with one batched adoption rather
// than one full-ledger encode per repaired diagnostic bundle.
func TestScheduleOwnerUnknownRefreshLedgerWorkGrowsLinearly(t *testing.T) {
	measure := func(t *testing.T, count int) ownerSweepWork {
		a, fake, state, notices := scheduledResultFixture(t)
		state, _ = couchschedule.Advance(state, couchschedule.Event{Kind: couchschedule.OwnerUnknown, At: a.coordinator.now(), Attempt: state.Active.ID, Revision: state.Active.Revision})
		if state.Active == nil || state.Active.Stage != "uncertain" || state.Last == nil || state.Last.Outcome != "unknown" {
			t.Fatalf("missing unknown fixture: %+v", state)
		}
		if err := updateScheduleFixture(a.coordinator.store, t.Context(), func(l *couchschedule.Ledger) error {
			l.Tasks = map[string]couchschedule.TaskState{}
			for i := 0; i < count; i++ {
				s := state
				s.Key.Task = fmt.Sprintf("unknown-%02d", i)
				l.Tasks[s.Key.ID()] = s
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		publications := 0
		a.coordinator.store.BeforePersist = func() error { publications++; return nil }
		historySyncs := 0
		a.coordinator.runs.SyncDirectory = func(path string) error { historySyncs++; return durablefile.SyncDirectory(path) }
		owner := newScheduleOwner(a.coordinator)
		owner.configure(scheduleExecutionWorker{}, notices)
		if err := owner.open(t.Context()); err != nil {
			t.Fatal(err)
		}
		defer owner.close()
		if err := owner.refresh(t.Context()); err != nil {
			t.Fatal(err)
		}
		ledger, err := a.coordinator.store.View(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		unknown := 0
		for _, got := range ledger.Tasks {
			if got.Active == nil {
				continue
			}
			unknown++
			record, output, err := a.history().Read(t.Context(), scheduleRunKey(got))
			if err != nil || record.Outcome != "unknown" || len(output) != 0 || got.Last == nil || got.Last.ID != state.Last.ID || !got.Last.EndedAt.Equal(state.Last.EndedAt) || !got.NextDue.Equal(state.NextDue) {
				t.Fatalf("committed unknown facts not repaired: %+v %+v %q %v", got, record, output, err)
			}
		}
		if unknown != count || len(fake.Runs()) != 0 || notices.admissions != 0 {
			t.Fatal("unknown recovery lost facts or replayed an effect")
		}
		if publications == 0 {
			t.Fatal("refresh did not commit")
		}
		work := publications * len(ledger.Tasks)
		t.Logf("tasks=%d ledger publications=%d encoded task rows=%d", count, publications, work)
		return ownerSweepWork{work, historySyncs}
	}
	var small, large ownerSweepWork
	t.Run("8-tasks", func(t *testing.T) { small = measure(t, 8) })
	t.Run("16-tasks", func(t *testing.T) { large = measure(t, 16) })
	assertOwnerSweepGrowth(t, small, large)
}

// Count every native history durability confirmation as well as whole-ledger
// encoding. This rejects an N-task canonical reread inside each task's sweep.
type ownerSweepWork struct{ rows, historySyncs int }

func assertOwnerSweepGrowth(t *testing.T, small, large ownerSweepWork) {
	t.Helper()
	t.Logf("8-task work=%+v; 16-task work=%+v", small, large)
	if small.rows <= 0 || large.rows > 2*small.rows+32 {
		t.Fatalf("ledger work is not linear: %+v/%+v", small, large)
	}
	if small.historySyncs <= 0 || large.historySyncs > 2*small.historySyncs+32 {
		t.Fatalf("history IO is not linear: %+v/%+v", small, large)
	}
}
