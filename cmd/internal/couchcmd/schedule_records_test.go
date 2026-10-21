package couchcmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchschedule"
	"github.com/xianxu/pair/cmd/internal/operationalrun"
)

func retainedPassiveFixture(t *testing.T) (*manualTaskRunner, *couchschedule.FakeTaskRuntime, couchschedule.StateStore, couchschedule.TaskState) {
	t.Helper()
	runner, fake, root := manualRunnerFixture(t)
	declaration, err := manualDeclaration(t.Context(), root, "report")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	state, _ := couchschedule.Advance(couchschedule.TaskState{}, couchschedule.Event{Kind: couchschedule.CatalogChanged, At: now, Declaration: &declaration})
	state.Last = &couchschedule.RunSummary{ID: "old-passive", Revision: "old-declaration", Source: "scheduled", DueAt: now.Add(-3 * time.Minute), StartedAt: now.Add(-2 * time.Minute), EndedAt: now.Add(-2 * time.Minute), Outcome: "skipped", Notice: "hub offline, skipped"}
	store := couchschedule.StateStore{Dir: filepath.Join(runner.namespace, "schedule")}
	if err := store.Update(t.Context(), func(l *couchschedule.Ledger) error { l.Tasks[state.Key.ID()] = state; return nil }); err != nil {
		t.Fatal(err)
	}
	return runner, fake, store, state
}
func TestSchedulePassiveHistoryPrecedesManualReplacement(t *testing.T) {
	runner, fake, store, before := retainedPassiveFixture(t)
	// A valid new declaration must not supply the previous run's revision.
	if before.Revision == before.Last.Revision {
		t.Fatal("fixture needs replaced revision")
	}
	hubDir := filepath.Join(runner.namespace, "schedule", before.Key.HubID())
	if err := os.WriteFile(hubDir, []byte("temporarily unavailable history"), 0600); err != nil {
		t.Fatal(err)
	}
	fake.Enqueue(couchschedule.ExecutionResult{Exit: 0, Outcome: "quiet"})
	if _, err := runner.run(t.Context(), before.Key.Hub, before.Key.Task, nil); err == nil {
		t.Fatal("manual replaced sole facts without durable passive history")
	}
	ledger, err := store.View(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	state := ledger.Tasks[before.Key.ID()]
	if state.Active != nil || state.Last.ID != before.Last.ID || len(fake.Runs()) != 0 {
		t.Fatalf("manual work crossed missing passive history: %+v runs=%v", state, fake.Runs())
	}
	if err := os.Remove(hubDir); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.run(t.Context(), before.Key.Hub, before.Key.Task, nil); err != nil {
		t.Fatal(err)
	}
	record, _, err := (operationalrun.Store{Namespace: runner.namespace}).Read(t.Context(), scheduleRunKey(before))
	if err != nil || record.DeclarationRevision != before.Last.Revision || !record.StartedAt.Equal(before.Last.StartedAt) || !record.EndedAt.Equal(before.Last.EndedAt) {
		t.Fatalf("passive facts replaced by current declaration/clock: %+v %v", record, err)
	}
	if len(fake.Runs()) != 1 {
		t.Fatalf("manual retry duplicated execution: %+v", fake.Runs())
	}
}
func TestSchedulePassiveLegacyRevisionNeverInvented(t *testing.T) {
	runner, _, store, state := retainedPassiveFixture(t)
	state.Last.Revision = ""
	if err := store.Update(t.Context(), func(l *couchschedule.Ledger) error { l.Tasks[state.Key.ID()] = state; return nil }); err != nil {
		t.Fatal(err)
	}
	if err := store.Update(t.Context(), func(l *couchschedule.Ledger) error {
		return ensurePassiveRun(context.Background(), runner.namespace, l.Tasks[state.Key.ID()])
	}); err == nil {
		t.Fatal("invented passive revision from current declaration")
	}
}

func TestScheduleManualUnknownFactsSurviveHistoryFailure(t *testing.T) {
	c, roots, now := scheduleServiceFixture(t)
	state := serviceManualOwner(t, c, roots[0])
	c.ownerIdentity = func(int) string { return "" }
	blocked := filepath.Join(filepath.Dir(c.store.Dir), "schedule", state.Key.HubID())
	first := true
	c.store.BeforePersist = func() error {
		if first {
			first = false
			return os.WriteFile(blocked, []byte("history unavailable"), 0600)
		}
		return nil
	}
	if err := c.refresh(t.Context()); err != nil {
		t.Fatalf("one task's unavailable history blocked all hubs: %v", err)
	}
	ledger, err := c.store.View(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	retained := ledger.Tasks[state.Key.ID()]
	if retained.Active == nil || retained.Active.Stage != "uncertain" || retained.Last == nil {
		t.Fatalf("unknown facts not committed before publication: %+v", retained)
	}
	if retained.Allowed || retained.Refusal == "" {
		t.Fatal("unavailable history lost its task-local diagnostic")
	}
	ended := retained.Last.EndedAt
	if err := os.Remove(blocked); err != nil {
		t.Fatal(err)
	}
	c.store.BeforePersist = nil
	*now = now.Add(time.Second)
	if err := c.refresh(t.Context()); err != nil {
		t.Fatal(err)
	}
	record, _, err := (operationalrun.Store{Namespace: filepath.Dir(c.store.Dir)}).Read(t.Context(), scheduleRunKey(retained))
	if err != nil || record.Outcome != "unknown" || !record.EndedAt.Equal(ended) {
		t.Fatalf("unknown fact retry changed observation: %+v %v", record, err)
	}
	ledger, err = c.store.View(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if !ledger.Tasks[state.Key.ID()].Last.EndedAt.Equal(ended) {
		t.Fatal("canonical unknown facts changed after adoption")
	}
}
