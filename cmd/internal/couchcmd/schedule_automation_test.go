package couchcmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchschedule"
)

func TestScheduleAutomationDisabledDueIsPerHub(t *testing.T) {
	for _, direct := range []bool{false, true} {
		t.Run(fmt.Sprintf("script=%t", direct), func(t *testing.T) {
			coordinator, roots, now := scheduleServiceFixture(t)
			for _, root := range roots {
				if err := os.WriteFile(filepath.Join(root, "scripts/scheduled/report/schedule"), []byte(fmt.Sprintf(`{"version":1,"cron":"* * * * *","timeout":"1m","script_run":%t}`, direct)), 0600); err != nil {
					t.Fatal(err)
				}
			}
			service := &messageService{scheduleAutomationEnabled: func(root string) bool { return root != roots[0] }}
			owner := newScheduleOwner(coordinator)
			owner.automationEnabled = service.scheduleAutomationOn
			if err := owner.open(t.Context()); err != nil {
				t.Fatal(err)
			}
			defer owner.close()
			if err := owner.refresh(t.Context()); err != nil {
				t.Fatal(err)
			}
			*now = now.Add(time.Minute)
			if err := owner.reserveDueBatch(t.Context(), newSchedulerClockFake(*now), map[string]bool{roots[0]: true, roots[1]: true}); err != nil {
				t.Fatal(err)
			}
			ledger, err := owner.state.ViewDurable(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			off := ledger.Tasks[(couchschedule.TaskKey{Hub: roots[0], Task: "report"}).ID()]
			on := ledger.Tasks[(couchschedule.TaskKey{Hub: roots[1], Task: "report"}).ID()]
			if off.Active != nil || off.Last == nil || off.Last.Outcome != "skipped" || off.Last.Notice != "automation off, skipped" || !off.NextDue.After(*now) {
				t.Fatalf("disabled due not skipped: %+v", off)
			}
			if on.Active == nil || on.Active.Stage != "reserved" {
				t.Fatalf("enabled hub suppressed: %+v", on)
			}
			record, _, err := coordinator.runs.ReadDurable(t.Context(), scheduleRunKey(off))
			if err != nil || record.Outcome != "skipped" || record.Notice != "automation off, skipped" {
				t.Fatalf("missing durable skip: %+v %v", record, err)
			}
		})
	}
}

func TestScheduleAutomationDisabledReservedScriptRefusesButManualAuthorizes(t *testing.T) {
	for _, manual := range []bool{false, true} {
		t.Run(fmt.Sprintf("manual=%t", manual), func(t *testing.T) {
			runner, _, state, store := reservedExecutionFixture(t)
			if manual {
				state.Active.Source = "manual"
				state.Active.DueAt = time.Time{}
				if err := updateScheduleFixture(store, t.Context(), func(l *couchschedule.Ledger) error { l.Tasks[state.Key.ID()] = state; return nil }); err != nil {
					t.Fatal(err)
				}
			}
			coordinator := newScheduleCoordinator(runner.namespace, []string{state.Key.Hub}, time.UTC, runner.now)
			owner := newScheduleOwner(coordinator)
			owner.automationEnabled = func(string) bool { return false }
			if err := owner.open(t.Context()); err != nil {
				t.Fatal(err)
			}
			defer owner.close()
			declaration, err := manualDeclaration(t.Context(), state.Key.Hub, state.Key.Task)
			if err != nil {
				t.Fatal(err)
			}
			err = owner.authorize(t.Context(), state, couchschedule.OwnerIdentity{PID: 123, Start: "test-owner"}, declaration)
			if manual {
				if err != nil {
					t.Fatal(err)
				}
			} else {
				if err == nil || !strings.Contains(err.Error(), "automation off") {
					t.Fatalf("disabled script authorized: %v", err)
				}
			}
			ledger, readErr := owner.state.ViewDurable(t.Context())
			if readErr != nil {
				t.Fatal(readErr)
			}
			got := ledger.Tasks[state.Key.ID()]
			if !manual && (got.Active == nil || got.Active.Stage != "reserved" || got.Active.ControllerPID != 0) {
				t.Fatalf("off script acquired execution authority: %+v", got)
			}
		})
	}
}

func TestScheduleAutomationOffAtFinalNoticeAdmission(t *testing.T) {
	endpoint := newControlledEndpoint()
	rig, notice, _ := scheduleNoticeRig(t, endpoint)
	frozen, ok := notice.Binding("/repo1")
	if !ok {
		t.Fatal("missing binding")
	}
	// Flip after the hub gate and owner handoff, before the broker call.
	rig.s.scheduleAutomationEnabled = func(string) bool { return true }
	got, err := notice.Admit(t.Context(), "/repo1", frozen, "automation-off-notice", "due", func(_ctx context.Context, send func() (string, error)) (string, error) {
		rig.s.scheduleAutomationEnabled = func(root string) bool { return root != "/repo1" }
		return send()
	})
	if err != nil || got != "dropped: automation off" {
		t.Fatalf("disabled notice admitted: %q %v", got, err)
	}
	if endpoint.deliveries() != 0 {
		t.Fatal("disabled automation reached endpoint")
	}
	if _, known, _ := rig.s.broker.ReceiptSnapshot("automation-off-notice"); known {
		t.Fatal("disabled automation entered broker")
	}
}

func TestScheduleAutomationOffWorkerRefusalSettlesWithoutStart(t *testing.T) {
	runner, fake, state, _ := reservedExecutionFixture(t)
	coordinator := newScheduleCoordinator(runner.namespace, []string{state.Key.Hub}, time.UTC, runner.now)
	owner := newScheduleOwner(coordinator)
	owner.automationEnabled = func(string) bool { return false }
	owner.configure(scheduleExecutionWorker{}, &resultScheduleNotices{})
	if err := owner.open(t.Context()); err != nil {
		t.Fatal(err)
	}
	defer owner.close()
	worker := scheduleExecutionWorker{namespace: runner.namespace, permitsDir: runner.permitsDir, executor: fake, now: runner.now, withDeadline: runner.withDeadline}
	facts := worker.execute(t.Context(), state, nil, func(ctx context.Context, identity couchschedule.OwnerIdentity, declaration couchschedule.Declaration) error {
		return owner.authorize(ctx, state, identity, declaration)
	})
	if facts.err == nil || !strings.Contains(facts.err.Error(), "automation off") {
		t.Fatalf("off worker accepted: %+v", facts)
	}
	runs := fake.Runs()
	if len(runs) != 1 || runs[0].Authorized {
		t.Fatalf("off worker received start authorization: %+v", runs)
	}
	if err := owner.complete(t.Context(), state, facts); err != nil {
		t.Fatal(err)
	}
	ledger, err := owner.state.ViewDurable(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	got := ledger.Tasks[state.Key.ID()]
	if got.Last == nil || got.Last.Outcome != "refused" || !got.NextDue.Equal(state.NextDue) {
		t.Fatalf("off refusal did not settle exact occurrence: %+v", got)
	}
	record, _, err := coordinator.runs.ReadDurable(t.Context(), scheduleRunKey(got))
	if err != nil || record.Outcome != "refused" {
		t.Fatalf("missing refusal history: %+v %v", record, err)
	}
}

func TestScheduleAutomationOffDoesNotDiscardRunningCompletion(t *testing.T) {
	scheduler, _, state, _ := scheduledResultFixture(t)
	owner := newScheduleOwner(scheduler.coordinator)
	owner.automationEnabled = func(string) bool { return false }
	owner.configure(scheduleExecutionWorker{}, &resultScheduleNotices{})
	if err := owner.open(t.Context()); err != nil {
		t.Fatal(err)
	}
	defer owner.close()
	facts := scheduleExecutionFacts{result: couchschedule.ExecutionResult{Exit: 0, Outcome: "quiet", Output: []byte("finished while off")}, ended: scheduler.coordinator.now()}
	if err := owner.complete(t.Context(), state, facts); err != nil {
		t.Fatal(err)
	}
	ledger, err := owner.state.ViewDurable(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	got := ledger.Tasks[state.Key.ID()]
	if got.Active != nil || got.Last == nil || got.Last.Outcome != "quiet" || got.Last.ID != state.Active.ID {
		t.Fatalf("running completion changed by switch: %+v", got)
	}
	record, output, err := scheduler.coordinator.runs.ReadDurable(t.Context(), scheduleRunKey(got))
	if err != nil || record.Outcome != "quiet" || string(output) != "finished while off" {
		t.Fatalf("completion facts lost: %+v %q %v", record, output, err)
	}
}

func TestScheduleAutomationDefaultIsEnabled(t *testing.T) {
	if !(&messageService{}).scheduleAutomationOn("/any") || !(&scheduleOwner{}).automationOn("/any") {
		t.Fatal("missing automation integration changed default")
	}
}
