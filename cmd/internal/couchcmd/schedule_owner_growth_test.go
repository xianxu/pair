package couchcmd

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Measure the complete persisted task rows, rather than only calls to a pure
// transition helper: every ledger publication encodes every configured row.
// A due sweep must grow with the catalogue, not multiply full-ledger work by
// the number of due tasks. This uses real discovery, the owner and native IO.
func TestScheduleOwnerDueTickLedgerWorkGrowsLinearly(t *testing.T) {
	measure := func(t *testing.T, taskCount int) int {
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
		return work
	}
	var small, large int
	t.Run("8-tasks", func(t *testing.T) { small = measure(t, 8) })
	t.Run("16-tasks", func(t *testing.T) { large = measure(t, 16) })
	// The fixed pair of bounded notice jobs may add a small constant. Doubling
	// due tasks must not quadruple the rows passed through full-ledger encoding.
	if small > 0 && large > 2*small+32 {
		t.Fatalf("due tick full-ledger work grew faster than linearly: 8 tasks=%d rows; 16 tasks=%d rows (limit %d)", small, large, 2*small+32)
	}
}
