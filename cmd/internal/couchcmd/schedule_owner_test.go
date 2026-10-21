package couchcmd

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchschedule"
)

func TestScheduleOwnerCanonicalTerminalPrecedesCatalogueEdit(t *testing.T) {
	for _, edit := range []string{"removed", "revision", "owner-policy"} {
		t.Run(edit, func(t *testing.T) {
			a, fake, state, notices := scheduledDeliveryFixture(t)
			key := scheduleRunKey(state)
			if err := a.history().FinalizeNotice(t.Context(), key, state.Active.Revision, "submitted"); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(state.Key.Hub, "scripts/scheduled/report/schedule")
			switch edit {
			case "removed":
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			case "revision":
				if err := os.WriteFile(path, []byte(`{"version":1,"cron":"*/2 * * * *","timeout":"1m","script_run":true}`), 0600); err != nil {
					t.Fatal(err)
				}
			case "owner-policy":
				if err := os.WriteFile(path, []byte(`{"version":1,"cron":"* * * * *","timeout":"1m","script_run":true,"scope":"team","owner":"absent-owner"}`), 0600); err != nil {
					t.Fatal(err)
				}
			}
			fail := true
			a.coordinator.store.BeforePersist = func() error {
				if fail {
					return errors.New("authority unavailable")
				}
				return nil
			}
			owner := newScheduleOwner(a.coordinator)
			if err := owner.open(t.Context()); err != nil {
				t.Fatal(err)
			}
			defer owner.close()
			if err := owner.refresh(t.Context()); err == nil {
				t.Fatal("failed authority commit acknowledged")
			}
			ledger, err := a.coordinator.store.View(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			if got := ledger.Tasks[state.Key.ID()]; got.Active == nil || got.Last.Notice != "pending" {
				t.Fatalf("failed commit released prior authority: %+v", got)
			}
			fail = false
			if err := owner.refresh(t.Context()); err != nil {
				t.Fatal(err)
			}
			ledger, err = a.coordinator.store.View(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			got := ledger.Tasks[state.Key.ID()]
			if got.Active != nil || got.Last == nil || got.Last.Notice != "submitted" {
				t.Fatalf("terminal history stranded by edit: %+v", got)
			}
			record, output, err := a.history().Read(t.Context(), key)
			if err != nil || record.Notice != "submitted" || string(output) != "saved execution output" {
				t.Fatalf("canonical facts changed: %+v %q %v", record, output, err)
			}
			if len(fake.Runs()) != 0 || notices.admissions != 0 {
				t.Fatal("settlement replayed an external effect")
			}
		})
	}
}

func TestScheduleOwnerCatalogueEditBusyWhileExecuting(t *testing.T) {
	a, _, state, _ := scheduledResultFixture(t)
	if err := os.Remove(filepath.Join(state.Key.Hub, "scripts/scheduled/report/schedule")); err != nil {
		t.Fatal(err)
	}
	owner := newScheduleOwner(a.coordinator)
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
	got := ledger.Tasks[state.Key.ID()]
	if got.Active == nil || got.Active.Stage != "running" || got.Revision != state.Revision || !got.Allowed {
		t.Fatalf("edit cancelled retained execution: %+v", got)
	}
	if owner.busy[state.Key.ID()] == "" {
		t.Fatal("busy catalogue edit is invisible")
	}
	_ = couchschedule.SettlementStable
}

func TestScheduleOwnerCompletionPublishesCanonicalFactsBeforeAuthority(t *testing.T) {
	a, _, state, _ := scheduledResultFixture(t)
	fail := false
	a.coordinator.store.BeforePersist = func() error {
		if fail {
			return errors.New("ledger commit failed")
		}
		return nil
	}
	owner := newScheduleOwner(a.coordinator)
	if err := owner.open(t.Context()); err != nil {
		t.Fatal(err)
	}
	defer owner.close()
	facts := &scheduleExecutionFacts{result: couchschedule.ExecutionResult{Exit: 10, Outcome: "wake", Output: []byte("new result")}, ended: a.coordinator.now().Add(time.Second)}
	event := couchschedule.Event{Kind: couchschedule.TaskFinished, At: facts.ended, Attempt: state.Active.ID, Revision: state.Active.Revision, Exit: 10, Outcome: "wake", HubOnline: true}
	fail = true
	if _, err := owner.apply(t.Context(), state.Key, event, facts); err == nil {
		t.Fatal("failed authority commit acknowledged")
	}
	record, output, err := a.history().Read(t.Context(), scheduledResultRecord(state, facts.ended, "wake").Key)
	if err != nil || record.Notice != "pending" || string(output) != "new result" {
		t.Fatalf("completion facts not saved: %+v %q %v", record, output, err)
	}
	fail = false
	if _, err := owner.apply(t.Context(), state.Key, event, facts); err != nil {
		t.Fatal(err)
	}
	l, err := a.coordinator.store.View(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	got := l.Tasks[state.Key.ID()]
	if got.Last == nil || got.Last.Outcome != "wake" || got.Active == nil || got.Active.Stage != "notifying" {
		t.Fatalf("completion not canonical: %+v", got)
	}
}

func TestScheduleOwnerDueConsumptionCommitsBeforePassiveHistory(t *testing.T) {
	c, _, now := scheduleServiceFixture(t)
	owner := newScheduleOwner(c)
	if err := owner.open(t.Context()); err != nil {
		t.Fatal(err)
	}
	defer owner.close()
	if err := owner.refresh(t.Context()); err != nil {
		t.Fatal(err)
	}
	l, err := c.store.View(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	var state couchschedule.TaskState
	for _, state = range l.Tasks {
		break
	}
	*now = state.NextDue.Add(2 * time.Minute)
	event := couchschedule.Event{Kind: couchschedule.ClockObserved, At: *now, RunID: "missed-occurrence", HubOnline: true, Location: c.location}
	if _, err = owner.apply(t.Context(), state.Key, event, nil); err != nil {
		t.Fatal(err)
	}
	l, err = c.store.View(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	state = l.Tasks[state.Key.ID()]
	if state.Last == nil || state.Last.Outcome != "skipped" {
		t.Fatalf("due not consumed: %+v", state)
	}
	if _, _, err := c.runs.Read(t.Context(), scheduleRunKey(state)); err != nil {
		t.Fatalf("committed passive facts lack history: %v", err)
	}
}

func TestScheduleOwnerAuthorizesOnlyAfterDurableController(t *testing.T) {
	runner, _, reserved, store := reservedExecutionFixture(t)
	c := newScheduleCoordinator(runner.namespace, []string{reserved.Key.Hub}, time.UTC, runner.now)
	fail := true
	c.store.BeforePersist = func() error {
		if fail {
			return errors.New("no authority commit")
		}
		return nil
	}
	o := newScheduleOwner(c)
	if err := o.open(t.Context()); err != nil {
		t.Fatal(err)
	}
	defer o.close()
	d, err := manualDeclaration(t.Context(), reserved.Key.Hub, reserved.Key.Task)
	if err != nil {
		t.Fatal(err)
	}
	identity := couchschedule.OwnerIdentity{PID: 7474, Start: "controller"}
	if err := o.authorize(t.Context(), reserved, identity, d); err == nil {
		t.Fatal("failed commit authorized script")
	}
	l, err := store.View(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if got := l.Tasks[reserved.Key.ID()]; got.Active.Stage != "reserved" || got.Active.ControllerPID != 0 {
		t.Fatalf("failed ready changed authority: %+v", got)
	}
	fail = false
	if err := o.authorize(t.Context(), reserved, identity, d); err != nil {
		t.Fatal(err)
	}
	l, err = store.View(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if got := l.Tasks[reserved.Key.ID()]; got.Active.Stage != "running" || got.Active.ControllerPID != identity.PID {
		t.Fatalf("ready facts not durable: %+v", got)
	}
}

func TestScheduleOwnerDispatchTickAndWorkerCompletion(t *testing.T) {
	c, _, now := scheduleServiceFixture(t)
	o := newScheduleOwner(c)
	notices := &resultScheduleNotices{}
	o.configure(scheduleExecutionWorker{}, notices)
	clock := newSchedulerClockFake(*now)
	if err := o.open(t.Context()); err != nil {
		t.Fatal(err)
	}
	defer o.close()
	if err := o.refresh(t.Context()); err != nil {
		t.Fatal(err)
	}
	l, err := c.store.View(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range l.Tasks {
		*now = s.NextDue
		break
	}
	clock.advance(t, *now)
	if err := o.tick(t.Context(), clock); err != nil {
		t.Fatal(err)
	}
	if len(o.loop.notices) != 2 {
		t.Fatalf("due notices not dispatched: %d", len(o.loop.notices))
	}

	a, _, state, _ := scheduledResultFixture(t)
	executionOwner := newScheduleOwner(a.coordinator)
	if err := executionOwner.open(t.Context()); err != nil {
		t.Fatal(err)
	}
	defer executionOwner.close()
	facts := scheduleExecutionFacts{result: couchschedule.ExecutionResult{Exit: 0, Outcome: "quiet", Output: []byte("quiet output")}, ended: a.coordinator.now()}
	if err := executionOwner.complete(t.Context(), state, facts); err != nil {
		t.Fatal(err)
	}
	l, err = a.coordinator.store.View(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if got := l.Tasks[state.Key.ID()]; got.Active != nil || got.Last == nil || got.Last.Outcome != "quiet" {
		t.Fatalf("worker facts not completed: %+v", got)
	}
}

func TestScheduleOwnerNoticeCapacityDoesNotCreateUnsentDelivery(t *testing.T) {
	c, roots, now := scheduleServiceFixture(t)
	dir := filepath.Join(roots[0], "scripts/scheduled/another")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "schedule"), []byte(`{"version":1,"cron":"* * * * *","timeout":"1m"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\nfunc main(){}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	o := newScheduleOwner(c)
	o.configure(scheduleExecutionWorker{}, &resultScheduleNotices{})
	if err := o.open(t.Context()); err != nil {
		t.Fatal(err)
	}
	defer o.close()
	if err := o.refresh(t.Context()); err != nil {
		t.Fatal(err)
	}
	l, err := c.store.View(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range l.Tasks {
		*now = s.NextDue
		break
	}
	clock := newSchedulerClockFake(*now)
	if err := o.tick(t.Context(), clock); err != nil {
		t.Fatal(err)
	}
	l, err = c.store.View(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	delivery, waiting := 0, 0
	for _, s := range l.Tasks {
		if s.Active != nil {
			if s.Active.Stage == "delivery" {
				delivery++
			}
			if s.Active.Stage == "notifying" || s.Active.Stage == "reserved" {
				waiting++
			}
		}
	}
	if delivery != 2 || waiting != 1 {
		t.Fatalf("capacity fabricated unsent delivery: delivery=%d waiting=%d", delivery, waiting)
	}
}

func TestScheduleOwnerPendingEditBlocksNextOccurrenceAfterSettlement(t *testing.T) {
	a, _, state, _ := scheduledResultFixture(t)
	o := newScheduleOwner(a.coordinator)
	if err := o.open(t.Context()); err != nil {
		t.Fatal(err)
	}
	defer o.close()
	o.busy[state.Key.ID()] = "catalogue change pending: task busy"
	facts := scheduleExecutionFacts{result: couchschedule.ExecutionResult{Exit: 0, Outcome: "quiet"}, ended: a.coordinator.now()}
	if err := o.complete(t.Context(), state, facts); err != nil {
		t.Fatal(err)
	}
	if err := o.reserveDueBatch(t.Context(), newSchedulerClockFake(state.NextDue), map[string]bool{state.Key.Hub: true}); err != nil {
		t.Fatal(err)
	}
	l, err := a.coordinator.store.View(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if got := l.Tasks[state.Key.ID()]; got.Active != nil {
		t.Fatalf("pending edit admitted another old-revision occurrence: %+v", got)
	}
}
