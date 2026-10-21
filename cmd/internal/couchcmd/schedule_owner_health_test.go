package couchcmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchschedule"
	"github.com/xianxu/pair/cmd/internal/operationalrun"
)

func ownerHealthClock(t *testing.T, o *scheduleOwner) *time.Time {
	t.Helper()
	now := o.coordinator.now()
	location, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatal(err)
	}
	o.coordinator.location = location
	o.coordinator.now = func() time.Time { return now }
	o.recovered = true
	// These tests exercise health publication, not consumption of another due slot.
	if err := o.state.Update(t.Context(), func(l *couchschedule.Ledger) error {
		for key, s := range l.Tasks {
			s.NextDue = now.Add(24 * time.Hour)
			l.Tasks[key] = s
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return &now
}
func hideOwnerHealthHub(t *testing.T, root string) func() {
	t.Helper()
	saved := root + "-temporarily-unreadable"
	if err := os.Rename(root, saved); err != nil {
		t.Fatal(err)
	}
	restore := func() {
		if _, err := os.Stat(saved); os.IsNotExist(err) {
			return
		}
		if _, err := os.Stat(root); err == nil {
			if err := os.Remove(root); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.Rename(saved, root); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(restore)
	return restore
}
func assertOwnerCatalogueHealth(t *testing.T, o *scheduleOwner, since time.Time, cause error, other string) {
	t.Helper()
	diagnostic := fmt.Sprintf("task folder unreadable since %s: %s", since.In(o.coordinator.location).Format("15:04"), cause)
	assertOwnerHealthDiagnostic(t, o, diagnostic, other)
}
func assertOwnerHealthDiagnostic(t *testing.T, o *scheduleOwner, diagnostic, other string) {
	t.Helper()
	snapshot := o.coordinator.view()
	if err := snapshot.Validate(); err != nil {
		t.Fatalf("invalid health snapshot: %+v %v", snapshot, err)
	}
	var rendered bytes.Buffer
	if err := renderScheduleStatus(&rendered, snapshot); err != nil {
		t.Fatal(err)
	}
	if other == "" {
		if snapshot.Error != diagnostic {
			t.Errorf("catalogue health changed without successful rediscovery: got %q want %q", snapshot.Error, diagnostic)
		}
	} else if !strings.Contains(snapshot.Error, diagnostic) || !strings.Contains(snapshot.Error, other) {
		t.Errorf("independent storage failure displaced catalogue health: got %q need %q and %q", snapshot.Error, diagnostic, other)
	}
	if !strings.Contains(rendered.String(), diagnostic) {
		t.Errorf("CLI hid catalogue outage: want %q in %q", diagnostic, rendered.String())
	}
}
func assertOwnerCatalogueHealthy(t *testing.T, o *scheduleOwner) {
	t.Helper()
	snapshot := o.coordinator.view()
	if err := snapshot.Validate(); err != nil {
		t.Fatal(err)
	}
	var rendered bytes.Buffer
	if err := renderScheduleStatus(&rendered, snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot.Error != "" || strings.Contains(rendered.String(), "task folder unreadable since") {
		t.Errorf("successful rediscovery did not clear catalogue outage: %+v text=%q", snapshot, rendered.String())
	}
}

func TestScheduleOwnerCatalogueHealthRetainedWhileFactsSettle(t *testing.T) {
	o, _, reserved := ownerRecoveryFixture(t, "manual", "reserved")
	fake := couchschedule.NewFakeTaskRuntime()
	o.loop.worker.executor = fake
	now := ownerHealthClock(t, o)
	fake.Enqueue(couchschedule.ExecutionResult{Exit: 0, Outcome: "quiet", Output: []byte("retained completed output")})
	// Obtain completed observations from the production worker while its declared
	// directory is still readable. Their owner turn may arrive after discovery fails.
	o.loop.worker.withDeadline = func(ctx context.Context, _ time.Time) (context.Context, context.CancelFunc) {
		return context.WithCancel(ctx)
	}
	facts := o.loop.worker.execute(t.Context(), reserved, []string{"private-argv"}, func(ctx context.Context, id couchschedule.OwnerIdentity, d couchschedule.Declaration) error {
		return o.authorize(ctx, reserved, id, d)
	})
	if facts.err != nil || facts.result.Outcome != "quiet" {
		t.Fatalf("worker did not return terminal facts: %+v", facts)
	}
	runs := fake.Runs()
	if len(runs) != 1 || !runs[0].Authorized || !runs[0].Finished {
		t.Fatalf("fixture bypassed production worker: %+v", runs)
	}
	o.collect(scheduleOwnerResult{job: scheduleOwnerJob{state: reserved}, facts: facts})
	restore := hideOwnerHealthHub(t, reserved.Key.Hub)
	since := *now
	cause := o.refresh(t.Context())
	if cause == nil {
		t.Fatal("missing real hub did not fail discovery")
	}
	assertOwnerCatalogueHealth(t, o, since, cause, "")
	if err := o.retry(t.Context(), newSchedulerClockFake(*now)); err != nil {
		t.Fatal(err)
	}
	ledger, err := o.state.ViewDurable(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	got := ledger.Tasks[reserved.Key.ID()]
	if got.Active != nil || got.Last == nil || got.Last.Outcome != "quiet" || got.Last.ID != reserved.Active.ID || len(o.loop.pending) != 0 {
		t.Fatalf("catalogue outage blocked terminal settlement: %+v pending=%v", got, o.loop.pending)
	}
	record, output, err := o.coordinator.runs.ReadDurable(t.Context(), operationalrun.RunKey{Hub: reserved.Key.HubID(), Task: reserved.Key.Task, ID: reserved.Active.ID})
	if err != nil || record.Outcome != "quiet" || !bytes.Equal(output, facts.result.Output) {
		t.Fatalf("terminal output lost during outage: %+v %q %v", record, output, err)
	}
	for i := 0; i < 3; i++ {
		*now = now.Add(45 * time.Second)
		if err := o.tick(t.Context(), newSchedulerClockFake(*now)); err != nil {
			t.Fatal(err)
		}
		assertOwnerCatalogueHealth(t, o, since, cause, "")
		if err := o.publish(t.Context()); err != nil {
			t.Fatal(err)
		}
		assertOwnerCatalogueHealth(t, o, since, cause, "")
	}
	// A second failed scan updates its cause but not the outage's original time.
	if err := os.WriteFile(reserved.Key.Hub, []byte("not a task directory"), 0600); err != nil {
		t.Fatal(err)
	}
	*now = now.Add(time.Minute)
	latest := o.refresh(t.Context())
	if latest == nil || latest.Error() == cause.Error() {
		t.Fatalf("changed real discovery failure not observed: %v", latest)
	}
	assertOwnerCatalogueHealth(t, o, since, latest, "")
	if err := o.tick(t.Context(), newSchedulerClockFake(*now)); err != nil {
		t.Fatal(err)
	}
	assertOwnerCatalogueHealth(t, o, since, latest, "")
	restore()
	if err := o.publish(t.Context()); err != nil {
		t.Fatal(err)
	}
	assertOwnerCatalogueHealth(t, o, since, latest, "")
	if err := o.refresh(t.Context()); err != nil {
		t.Fatal(err)
	}
	assertOwnerCatalogueHealthy(t, o)
	if len(fake.Runs()) != 1 || len(o.loop.execution) != 0 || len(o.loop.notices) != 0 {
		t.Fatal("health recovery replayed completed work")
	}
}

func TestScheduleOwnerCatalogueHealthSurvivesIndependentLedgerRecovery(t *testing.T) {
	o, _, state := ownerRecoveryFixture(t, "manual", "running")
	now := ownerHealthClock(t, o)
	restoreHub := hideOwnerHealthHub(t, state.Key.Hub)
	since := *now
	cause := o.refresh(t.Context())
	if cause == nil {
		t.Fatal("missing hub did not fail discovery")
	}
	restoreLedger := breakOwnerHealthLedger(t, o)
	storageErr := o.publish(t.Context())
	if storageErr == nil {
		t.Fatal("non-regular ledger did not block publication")
	}
	assertOwnerCatalogueHealth(t, o, since, cause, storageErr.Error())
	restoreLedger()
	for i := 0; i < 2; i++ {
		*now = now.Add(time.Minute)
		if err := o.publish(t.Context()); err != nil {
			t.Fatal(err)
		}
		assertOwnerCatalogueHealth(t, o, since, cause, "")
		if err := o.tick(t.Context(), newSchedulerClockFake(*now)); err != nil {
			t.Fatal(err)
		}
		assertOwnerCatalogueHealth(t, o, since, cause, "")
	}
	restoreHub()
	if err := o.publish(t.Context()); err != nil {
		t.Fatal(err)
	}
	assertOwnerCatalogueHealth(t, o, since, cause, "")
	if err := o.refresh(t.Context()); err != nil {
		t.Fatal(err)
	}
	assertOwnerCatalogueHealthy(t, o)
}

func TestScheduleOwnerCatalogueHealthFromStartupFailure(t *testing.T) {
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(base, "hub-not-created")
	now := time.Date(2026, 10, 21, 18, 42, 0, 0, time.UTC)
	location, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatal(err)
	}
	c := newScheduleCoordinator(filepath.Join(base, "namespace"), []string{root}, location, func() time.Time { return now })
	o := newScheduleOwner(c)
	o.configure(scheduleExecutionWorker{}, offlineScheduleNotices{})
	if err := o.open(t.Context()); err != nil {
		t.Fatal(err)
	}
	defer o.close()
	since := now
	cause := o.refresh(t.Context())
	if cause == nil {
		t.Fatal("startup unexpectedly discovered absent hub")
	}
	for i := 0; i < 3; i++ {
		now = now.Add(time.Minute)
		if err := o.tick(t.Context(), newSchedulerClockFake(now)); err != nil {
			t.Fatal(err)
		}
		assertOwnerCatalogueHealth(t, o, since, cause, "")
		latest := o.refresh(t.Context())
		if latest == nil {
			t.Fatal("missing startup hub unexpectedly recovered")
		}
		assertOwnerCatalogueHealth(t, o, since, latest, "")
	}
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	if err := o.publish(t.Context()); err != nil {
		t.Fatal(err)
	}
	assertOwnerCatalogueHealth(t, o, since, cause, "")
	if err := o.refresh(t.Context()); err != nil {
		t.Fatal(err)
	}
	assertOwnerCatalogueHealthy(t, o)
}

func breakOwnerHealthLedger(t *testing.T, o *scheduleOwner) func() {
	t.Helper()
	statePath := filepath.Join(o.coordinator.store.Dir, "state.json")
	saved := statePath + ".health-test-saved"
	if err := os.Rename(statePath, saved); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(statePath, 0700); err != nil {
		t.Fatal(err)
	}
	restoreLedger := func() {
		if _, err := os.Stat(saved); os.IsNotExist(err) {
			return
		}
		if err := os.Remove(statePath); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(saved, statePath); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(restoreLedger)
	return restoreLedger
}

func TestScheduleOwnerCatalogueHealthRetainsUnappliedRefreshFailure(t *testing.T) {
	o, _, state := ownerRecoveryFixture(t, "manual", "running")
	now := ownerHealthClock(t, o)
	since := *now
	restoreLedger := breakOwnerHealthLedger(t, o)
	cause := o.refresh(t.Context())
	if cause == nil {
		t.Fatal("catalogue refresh ignored unreadable authority")
	}
	diagnostic := fmt.Sprintf("catalogue refresh failed since %s: %s", since.In(o.coordinator.location).Format("15:04"), cause)
	assertOwnerHealthDiagnostic(t, o, diagnostic, "")
	restoreLedger()
	for i := 0; i < 2; i++ {
		*now = now.Add(time.Minute)
		if err := o.tick(t.Context(), newSchedulerClockFake(*now)); err != nil {
			t.Fatal(err)
		}
		assertOwnerHealthDiagnostic(t, o, diagnostic, "")
	}
	// A new discovery failure changes the explanation, preserving the outage's
	// initial time across failure classes until catalogue application succeeds.
	restoreHub := hideOwnerHealthHub(t, state.Key.Hub)
	latest := o.refresh(t.Context())
	if latest == nil {
		t.Fatal("missing task folder unexpectedly refreshed")
	}
	assertOwnerCatalogueHealth(t, o, since, latest, "")
	restoreHub()
	if err := o.publish(t.Context()); err != nil {
		t.Fatal(err)
	}
	assertOwnerCatalogueHealth(t, o, since, latest, "")
	if err := o.refresh(t.Context()); err != nil {
		t.Fatal(err)
	}
	assertOwnerCatalogueHealthy(t, o)
}

func TestScheduleOwnerCatalogueBusyReasonPreservesHistoryRefusal(t *testing.T) {
	o, _, state := ownerRecoveryFixture(t, "manual", "running")
	ownerHealthClock(t, o)
	path := filepath.Join(state.Key.Hub, "scripts", "scheduled", "report", "schedule")
	if err := os.WriteFile(path, []byte(`{"version":1,"cron":"*/2 * * * *","timeout":"1m","script_run":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := o.refresh(t.Context()); err != nil {
		t.Fatal(err)
	}
	if o.busy[state.Key.ID()] == "" {
		t.Fatal("fixture did not retain pending catalogue edit")
	}
	// A damaged existing native bundle is not an absent result. The production
	// event path must quarantine it while retaining the prior occurrence tuple.
	runDir := filepath.Join(o.coordinator.runs.Namespace, "schedule", state.Key.HubID(), state.Key.Task, "runs", state.Active.ID)
	if err := os.MkdirAll(runDir, 0700); err != nil {
		t.Fatal(err)
	}
	_, err := o.apply(t.Context(), state.Key, couchschedule.Event{Kind: couchschedule.TaskFinished, At: o.coordinator.now(), Attempt: state.Active.ID, Revision: state.Active.Revision, Exit: 0, Outcome: "quiet"}, &scheduleExecutionFacts{ended: o.coordinator.now(), result: couchschedule.ExecutionResult{Exit: 0, Outcome: "quiet"}})
	if err == nil {
		t.Fatal("damaged native result did not quarantine retained authority")
	}
	ledger, err := o.state.ViewDurable(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	quarantined := ledger.Tasks[state.Key.ID()]
	if quarantined.Allowed || !strings.Contains(quarantined.Refusal, "history") || quarantined.Active == nil || quarantined.Active.ID != state.Active.ID {
		t.Fatalf("fixture did not retain history quarantine: %+v", quarantined)
	}
	for i := 0; i < 2; i++ {
		if err := o.publish(t.Context()); err != nil {
			t.Fatal(err)
		}
		snapshot := o.coordinator.view()
		if err := snapshot.Validate(); err != nil {
			t.Fatal(err)
		}
		if len(snapshot.Tasks) != 1 || snapshot.Tasks[0].State != "refused" || !strings.Contains(snapshot.Tasks[0].Reason, quarantined.Refusal) {
			t.Errorf("busy reason hid durable history quarantine: snapshot=%+v want refusal %q", snapshot, quarantined.Refusal)
		}
		var rendered bytes.Buffer
		if err := renderScheduleStatus(&rendered, snapshot); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(rendered.String(), quarantined.Refusal) {
			t.Errorf("CLI concealed history repair cause behind pending edit: %q", rendered.String())
		}
	}
}

func TestScheduleOwnerHealthErrorBudgetKeepsIndependentCauses(t *testing.T) {
	catalogue := errors.New("catalogue-cause:" + strings.Repeat("C", 6000))
	authority := errors.New("authority-cause:" + strings.Repeat("A", 6000))
	for _, tc := range []struct {
		name              string
		current, retained error
		markers           []string
	}{
		{name: "nil"},
		{name: "one-long", current: catalogue, markers: []string{"catalogue-cause:"}},
		{name: "two-long", current: authority, retained: catalogue, markers: []string{"authority-cause:", "catalogue-cause:"}},
		{name: "nil-current", retained: catalogue, markers: []string{"catalogue-cause:"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			text := scheduleErrorText(tc.current, tc.retained)
			if len(text) > 4096 {
				t.Fatalf("health display exceeds transport budget: %d bytes", len(text))
			}
			if len(tc.markers) == 0 && text != "" {
				t.Fatalf("nil failures produced diagnostic %q", text)
			}
			for _, marker := range tc.markers {
				if !strings.Contains(text, marker) {
					t.Errorf("independent cause %q truncated away: %q", marker, text)
				}
			}
			snapshot := couchschedule.StatusSnapshot{Version: 1, Error: text, Hubs: []string{}, Tasks: []couchschedule.TaskStatus{}}
			if err := snapshot.Validate(); err != nil {
				t.Fatal(err)
			}
			var rendered bytes.Buffer
			if err := renderScheduleStatus(&rendered, snapshot); err != nil {
				t.Fatal(err)
			}
			for _, marker := range tc.markers {
				if !strings.Contains(rendered.String(), marker) {
					t.Errorf("CLI dropped cause %q", marker)
				}
			}
		})
	}
}
