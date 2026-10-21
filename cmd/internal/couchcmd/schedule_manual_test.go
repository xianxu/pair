package couchcmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchschedule"
	"github.com/xianxu/pair/cmd/internal/operationalrun"
)

func manualRunnerFixture(t *testing.T) (*manualTaskRunner, *couchschedule.FakeTaskRuntime, string) {
	t.Helper()
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(base, "ops")
	dir := filepath.Join(root, "scripts", "scheduled", "report")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "schedule"), []byte(`{"version":1,"cron":"* * * * *","timeout":"1m"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\nfunc main(){}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	fake := couchschedule.NewFakeTaskRuntime()
	runner := &manualTaskRunner{namespace: filepath.Join(base, "couch"), permitsDir: filepath.Join(base, "permits"), executor: fake, now: time.Now, newID: func() (string, error) { return "manual-1", nil }}
	return runner, fake, root
}

func TestScheduleManualPersistsResultBeforeClearingAndPreservesDue(t *testing.T) {
	runner, fake, root := manualRunnerFixture(t)
	ctx := context.Background()
	decls, err := (couchschedule.Catalog{}).Discover(ctx, []string{root})
	if err != nil {
		t.Fatal(err)
	}
	store := couchschedule.StateStore{Dir: filepath.Join(runner.namespace, "schedule")}
	due := time.Now().Add(time.Hour)
	state, _ := couchschedule.Advance(couchschedule.TaskState{}, couchschedule.Event{Kind: couchschedule.CatalogChanged, At: time.Now(), Declaration: &decls[0]})
	state.NextDue = due
	if err := store.Update(ctx, func(l *couchschedule.Ledger) error { l.Tasks[state.Key.ID()] = state; return nil }); err != nil {
		t.Fatal(err)
	}
	fake.Enqueue(couchschedule.ExecutionResult{Exit: 10, Outcome: "wake", Output: []byte("retained output")})
	result, err := runner.run(ctx, root, "report", []string{"--help", "--json", ""})
	if err != nil || result.Exit != 10 {
		t.Fatalf("result=%+v error=%v", result, err)
	}
	ledger, err := store.View(ctx)
	if err != nil {
		t.Fatal(err)
	}
	state = ledger.Tasks[state.Key.ID()]
	if state.Active != nil || state.Last == nil || state.Last.Source != "manual" || !state.NextDue.Equal(due) {
		t.Fatalf("manual changed recurrence/left owner: %+v", state)
	}
	record, output, err := (operationalrun.Store{Namespace: runner.namespace}).Read(ctx, operationalrun.RunKey{Hub: state.Key.HubID(), Task: "report", ID: "manual-1"})
	if err != nil || record.Exit == nil || *record.Exit != 10 || string(output) != "retained output" || record.Notice != "manual" {
		t.Fatalf("missing durable run: %+v %q %v", record, output, err)
	}
	if runs := fake.Runs(); len(runs) != 1 || !runs[0].Authorized || len(runs[0].Request.Args) != 3 || runs[0].Request.Args[2] != "" {
		t.Fatalf("not shared literal execution: %+v", runs)
	}
}

type refusedHistory struct{}

func (refusedHistory) Publish(context.Context, operationalrun.RunKey, operationalrun.Record, []byte) error {
	return errors.New("durability refused")
}

func TestScheduleManualLostHistoryRetainsUnknownWithoutReplay(t *testing.T) {
	runner, fake, root := manualRunnerFixture(t)
	runner.history = refusedHistory{}
	fake.Enqueue(couchschedule.ExecutionResult{Exit: 0, Outcome: "quiet"})
	if _, err := runner.run(context.Background(), root, "report", nil); err == nil {
		t.Fatal("lost history reported success")
	}
	store := couchschedule.StateStore{Dir: filepath.Join(runner.namespace, "schedule")}
	ledger, err := store.View(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	state := ledger.Tasks[(couchschedule.TaskKey{Hub: root, Task: "report"}).ID()]
	if state.Allowed || state.Refusal == "" || state.Active == nil || state.Active.Stage != "uncertain" || state.Last.Outcome != "unknown" {
		t.Fatalf("lost result did not retain authority: %+v", state)
	}
	runner.newID = func() (string, error) { return "manual-2", nil }
	if _, err := runner.run(context.Background(), root, "report", nil); err == nil {
		t.Fatal("unknown run replayed")
	}
	if len(fake.Runs()) != 1 {
		t.Fatal("a second run reached execution")
	}
}

func TestScheduleManualDurableReservationFailureNeverStarts(t *testing.T) {
	runner, fake, root := manualRunnerFixture(t)
	runner.beforePersist = func() error { return errors.New("reservation refused") }
	if _, err := runner.run(context.Background(), root, "report", nil); err == nil {
		t.Fatal("reservation failure ignored")
	}
	if len(fake.Runs()) != 0 {
		t.Fatal("execution preceded durable reservation")
	}
}

func TestScheduleManualSavedResultRecoversAuthorityWithoutExecutingAgain(t *testing.T) {
	runner, fake, root := manualRunnerFixture(t)
	writes := 0
	runner.beforePersist = func() error {
		writes++
		if writes == 3 {
			return errors.New("completion write refused")
		}
		return nil
	}
	if _, err := runner.run(context.Background(), root, "report", nil); err == nil {
		t.Fatal("missing authority completion failure")
	}
	coordinator := newScheduleCoordinator(runner.namespace, []string{root}, time.Local, time.Now)
	if err := coordinator.refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	store := couchschedule.StateStore{Dir: filepath.Join(runner.namespace, "schedule")}
	ledger, err := store.View(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	state := ledger.Tasks[(couchschedule.TaskKey{Hub: root, Task: "report"}).ID()]
	if state.Active != nil || state.Last == nil || state.Last.ID != "manual-1" || state.Last.Outcome != "quiet" {
		t.Fatalf("published result was not recovered: %+v", state)
	}
	if len(fake.Runs()) != 1 {
		t.Fatal("recovery executed task again")
	}
}
