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

type executionPublisherFunc func(context.Context, operationalrun.RunKey, operationalrun.Record, []byte) error

func (f executionPublisherFunc) Publish(ctx context.Context, key operationalrun.RunKey, record operationalrun.Record, output []byte) error {
	return f(ctx, key, record, output)
}

func reservedExecutionFixture(t *testing.T) (*manualTaskRunner, *couchschedule.FakeTaskRuntime, couchschedule.TaskState, couchschedule.StateStore) {
	t.Helper()
	runner, fake, root := manualRunnerFixture(t)
	if err := os.WriteFile(filepath.Join(root, "scripts/scheduled/report/schedule"), []byte(`{"version":1,"cron":"* * * * *","timeout":"1m","script_run":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	at := time.Date(2020, 1, 2, 9, 0, 0, 0, time.UTC)
	runner.now = func() time.Time { return at.Add(10 * time.Second) }
	runner.withDeadline = func(ctx context.Context, deadline time.Time) (context.Context, context.CancelFunc) {
		return context.WithCancel(ctx)
	}
	runner.newID = func() (string, error) { t.Fatal("reserved execution allocated a new ID"); return "", nil }
	declaration, err := manualDeclaration(context.Background(), root, "report")
	if err != nil {
		t.Fatal(err)
	}
	state, _ := couchschedule.Advance(couchschedule.TaskState{}, couchschedule.Event{Kind: couchschedule.CatalogChanged, At: at.Add(-time.Second), Declaration: &declaration, Location: time.UTC})
	state, _ = couchschedule.Advance(state, couchschedule.Event{Kind: couchschedule.ClockObserved, At: at, RunID: "scheduled-existing", HubOnline: true, Location: time.UTC})
	if state.Active == nil || state.Active.Source != "scheduled" {
		t.Fatalf("missing reserved state: %+v", state)
	}
	store := couchschedule.StateStore{Dir: filepath.Join(runner.namespace, "schedule")}
	if err = store.Update(context.Background(), func(l *couchschedule.Ledger) error { l.Tasks[state.Key.ID()] = state; return nil }); err != nil {
		t.Fatal(err)
	}
	return runner, fake, state, store
}

func TestScheduleReservedExecutionPublishesBeforePendingNotice(t *testing.T) {
	for _, tt := range []struct {
		outcome string
		exit    int
		notice  string
		active  bool
	}{{"wake", 10, "pending", true}, {"failed", 2, "pending", true}, {"quiet", 0, "none", false}, {"", 10, "pending", true}, {"", 0, "none", false}} {
		t.Run(tt.outcome+tt.notice, func(t *testing.T) {
			runner, fake, reserved, store := reservedExecutionFixture(t)
			fake.Enqueue(couchschedule.ExecutionResult{Exit: tt.exit, Outcome: tt.outcome, Output: []byte("retained output"), OutputBytes: 1000, BuildKey: "binary-key"})
			published := false
			runner.history = executionPublisherFunc(func(ctx context.Context, key operationalrun.RunKey, record operationalrun.Record, output []byte) error {
				ledger, err := store.View(ctx)
				if err != nil {
					return err
				}
				state := ledger.Tasks[reserved.Key.ID()]
				if state.Active == nil || state.Active.Stage != "running" || state.Active.ID != reserved.Active.ID {
					t.Fatalf("authority cleared before publication: %+v", state)
				}
				if record.Source != "scheduled" || record.Notice != tt.notice || !record.ScheduledAt.Equal(reserved.Active.DueAt) || !record.StartedAt.Equal(reserved.Active.StartedAt) {
					t.Fatalf("incorrect scheduled record: %+v", record)
				}
				published = true
				return (operationalrun.Store{Namespace: runner.namespace}).Publish(ctx, key, record, output)
			})
			var deadline time.Time
			runner.withDeadline = func(ctx context.Context, due time.Time) (context.Context, context.CancelFunc) {
				deadline = due
				return context.WithCancel(ctx)
			}
			result, err := runner.runReserved(context.Background(), reserved, []string{"--flag", ""})
			if err != nil {
				t.Fatal(err)
			}
			if !published || !deadline.Equal(reserved.Active.Deadline) {
				t.Fatalf("reservation deadline replaced: %s", deadline)
			}
			ledger, err := store.View(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			state := ledger.Tasks[reserved.Key.ID()]
			if !state.NextDue.Equal(reserved.NextDue) || (state.Active != nil) != tt.active {
				t.Fatalf("completion changed recurrence/notice authority: %+v", state)
			}
			if tt.active && (state.Active.Stage != "notifying" || state.Last.Notice != "pending") {
				t.Fatalf("notice not retained: %+v", state)
			}
			expected := tt.outcome
			if expected == "" {
				expected = "wake"
				if tt.exit == 0 {
					expected = "quiet"
				}
			}
			if result.Outcome != expected {
				t.Fatalf("missing exit classification: %+v", result)
			}
			runs := fake.Runs()
			if len(runs) != 1 || !runs[0].Authorized || runs[0].Request.RunID != reserved.Active.ID || runs[0].Request.Revision != reserved.Active.Revision || !runs[0].Request.Deadline.Equal(reserved.Active.Deadline) || len(runs[0].Request.Args) != 2 || runs[0].Request.Args[1] != "" {
				t.Fatalf("reservation not reused: %+v", runs)
			}
		})
	}
}

func TestScheduleReservedLostPublicationRetainsUnknownAndNeverReplays(t *testing.T) {
	runner, fake, reserved, store := reservedExecutionFixture(t)
	runner.history = refusedHistory{}
	fake.Enqueue(couchschedule.ExecutionResult{Exit: 10, Outcome: "wake"})
	if _, err := runner.runReserved(context.Background(), reserved, nil); err == nil {
		t.Fatal("lost publication accepted")
	}
	ledger, err := store.View(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	state := ledger.Tasks[reserved.Key.ID()]
	if state.Active == nil || state.Active.Stage != "uncertain" || state.Last == nil || state.Last.Outcome != "unknown" {
		t.Fatalf("lost completion released authority: %+v", state)
	}
	if _, err = runner.runReserved(context.Background(), reserved, nil); err == nil {
		t.Fatal("unknown attempt replay accepted")
	}
	if len(fake.Runs()) != 1 {
		t.Fatal("unknown attempt reached executor twice")
	}
}

func TestScheduleReservedRequiresPersistedAuthority(t *testing.T) {
	runner, fake, reserved, store := reservedExecutionFixture(t)
	if err := store.Update(context.Background(), func(l *couchschedule.Ledger) error { delete(l.Tasks, reserved.Key.ID()); return nil }); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.runReserved(context.Background(), reserved, nil); err == nil {
		t.Fatal("unpersisted reservation accepted")
	}
	if len(fake.Runs()) != 0 {
		t.Fatal("executor called before durable reservation")
	}
}

type taskExecutorFunc func(context.Context, couchschedule.ExecutionRequest, func(couchschedule.OwnerIdentity) error) (couchschedule.ExecutionResult, error)

func (f taskExecutorFunc) Execute(ctx context.Context, request couchschedule.ExecutionRequest, owner func(couchschedule.OwnerIdentity) error) (couchschedule.ExecutionResult, error) {
	return f(ctx, request, owner)
}

func TestScheduleReservedFreshDeclarationRefusalPublishesBeforeNotice(t *testing.T) {
	runner, fake, reserved, store := reservedExecutionFixture(t)
	runner.executor = taskExecutorFunc(func(ctx context.Context, request couchschedule.ExecutionRequest, onOwner func(couchschedule.OwnerIdentity) error) (couchschedule.ExecutionResult, error) {
		path := filepath.Join(reserved.Key.Hub, "scripts/scheduled/report/schedule")
		if err := os.WriteFile(path, []byte(`{"version":1,"cron":"* * * * *","timeout":"2m","script_run":true}`), 0600); err != nil {
			t.Fatal(err)
		}
		return fake.Execute(ctx, request, onOwner)
	})
	published := false
	runner.history = executionPublisherFunc(func(ctx context.Context, key operationalrun.RunKey, record operationalrun.Record, output []byte) error {
		ledger, err := store.View(ctx)
		if err != nil {
			return err
		}
		state := ledger.Tasks[reserved.Key.ID()]
		if state.Active == nil || state.Active.Stage != "reserved" {
			t.Fatalf("refusal authority released before history: %+v", state)
		}
		if record.Source != "scheduled" || record.Outcome != "refused" || record.Notice != "pending" || !record.ScheduledAt.Equal(reserved.Active.DueAt) {
			t.Fatalf("wrong refusal history: %+v", record)
		}
		published = true
		return (operationalrun.Store{Namespace: runner.namespace}).Publish(ctx, key, record, output)
	})
	result, err := runner.runReserved(context.Background(), reserved, nil)
	if err == nil || result.Outcome != "refused" || !published {
		t.Fatalf("missing known refusal: %+v %v", result, err)
	}
	runs := fake.Runs()
	if len(runs) != 1 || runs[0].Authorized {
		t.Fatalf("changed declaration authorized execution: %+v", runs)
	}
	ledger, err := store.View(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	state := ledger.Tasks[reserved.Key.ID()]
	if state.Active == nil || state.Active.Stage != "notifying" || state.Last.Outcome != "refused" || state.Last.Notice != "pending" {
		t.Fatalf("known refusal lost scheduled notice: %+v", state)
	}
	if _, err = runner.runReserved(context.Background(), reserved, nil); err == nil {
		t.Fatal("known refused attempt replayed")
	}
	if len(fake.Runs()) != 1 {
		t.Fatal("refusal replay reached executor")
	}
}

func TestScheduleReservedUnknownExecutionRetainsAuthority(t *testing.T) {
	for _, kind := range []string{"lost-result", "uncertain"} {
		t.Run(kind, func(t *testing.T) {
			runner, fake, reserved, store := reservedExecutionFixture(t)
			if kind == "uncertain" {
				fake.Enqueue(couchschedule.ExecutionResult{Exit: -1, Outcome: "uncertain"})
			} else {
				runner.executor = taskExecutorFunc(func(ctx context.Context, request couchschedule.ExecutionRequest, onOwner func(couchschedule.OwnerIdentity) error) (couchschedule.ExecutionResult, error) {
					result, err := fake.Execute(ctx, request, onOwner)
					if err != nil {
						return result, err
					}
					return couchschedule.ExecutionResult{}, &couchschedule.ExecutionError{MayHaveStarted: true, Err: errors.New("result lost")}
				})
			}
			if _, err := runner.runReserved(context.Background(), reserved, nil); err == nil {
				t.Fatal("unknown execution accepted")
			}
			ledger, err := store.View(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			state := ledger.Tasks[reserved.Key.ID()]
			if state.Active == nil || state.Active.Stage != "uncertain" || state.Last == nil || state.Last.Outcome != "unknown" {
				t.Fatalf("unknown authority cleared: %+v", state)
			}
			if _, err = runner.runReserved(context.Background(), reserved, nil); err == nil {
				t.Fatal("unknown execution replay accepted")
			}
			if len(fake.Runs()) != 1 {
				t.Fatal("unknown execution reached executor again")
			}
		})
	}
}

func TestScheduleReservedExpiredBeforeLaunchPublishesTimeout(t *testing.T) {
	runner, fake, reserved, store := reservedExecutionFixture(t)
	runner.now = func() time.Time { return reserved.Active.Deadline.Add(time.Second) }
	runner.withDeadline = context.WithDeadline
	result, err := runner.runReserved(context.Background(), reserved, nil)
	if err == nil || result.Outcome != "timeout" {
		t.Fatalf("expired reservation: %+v %v", result, err)
	}
	if len(fake.Runs()) != 0 {
		t.Fatal("expired reservation reached executor")
	}
	record, _, err := (operationalrun.Store{Namespace: runner.namespace}).Read(context.Background(), operationalrun.RunKey{Hub: reserved.Key.HubID(), Task: reserved.Key.Task, ID: reserved.Active.ID})
	if err != nil || record.Outcome != "timeout" || record.Notice != "pending" || record.Source != "scheduled" {
		t.Fatalf("missing timeout history: %+v %v", record, err)
	}
	ledger, err := store.View(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	state := ledger.Tasks[reserved.Key.ID()]
	if state.Active == nil || state.Active.Stage != "notifying" || state.Last.Outcome != "timeout" || state.Last.Notice != "pending" || !state.NextDue.Equal(reserved.NextDue) {
		t.Fatalf("timeout lost pending/recurrence: %+v", state)
	}
}

func TestScheduleReservedLatePreackRefusalCannotPoisonRunningAttempt(t *testing.T) {
	runner, _, reserved, store := reservedExecutionFixture(t)
	runner.executor = taskExecutorFunc(func(ctx context.Context, request couchschedule.ExecutionRequest, onOwner func(couchschedule.OwnerIdentity) error) (couchschedule.ExecutionResult, error) {
		// Another cooperating worker wins authorization after this caller's view.
		if err := store.Update(ctx, func(l *couchschedule.Ledger) error {
			state := l.Tasks[reserved.Key.ID()]
			owner := couchschedule.OwnerIdentity{PID: 777, Start: "winning-owner"}
			state, _ = couchschedule.Advance(state, couchschedule.Event{Kind: couchschedule.OwnerRecorded, At: runner.now(), Attempt: reserved.Active.ID, Revision: reserved.Active.Revision, Owner: &owner})
			state, _ = couchschedule.Advance(state, couchschedule.Event{Kind: couchschedule.ReservationDurable, At: runner.now(), Attempt: reserved.Active.ID, Revision: reserved.Active.Revision})
			l.Tasks[reserved.Key.ID()] = state
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		err := onOwner(couchschedule.OwnerIdentity{PID: 778, Start: "losing-owner"})
		return couchschedule.ExecutionResult{}, &couchschedule.ExecutionError{Err: err}
	})
	if _, err := runner.runReserved(context.Background(), reserved, nil); err == nil {
		t.Fatal("late refusal unexpectedly succeeded")
	}
	key := operationalrun.RunKey{Hub: reserved.Key.HubID(), Task: reserved.Key.Task, ID: reserved.Active.ID}
	if _, _, err := (operationalrun.Store{Namespace: runner.namespace}).Read(context.Background(), key); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("losing worker poisoned live attempt history: %v", err)
	}
	ledger, err := store.View(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	state := ledger.Tasks[reserved.Key.ID()]
	if state.Active == nil || state.Active.Stage != "running" || state.Active.ControllerPID != 777 {
		t.Fatalf("losing worker changed owner: %+v", state)
	}
}

func TestScheduleReservedExpiresDuringOwnerAcknowledgement(t *testing.T) {
	runner, fake, reserved, store := reservedExecutionFixture(t)
	runner.executor = taskExecutorFunc(func(ctx context.Context, request couchschedule.ExecutionRequest, onOwner func(couchschedule.OwnerIdentity) error) (couchschedule.ExecutionResult, error) {
		runner.now = func() time.Time { return reserved.Active.Deadline.Add(time.Second) }
		return fake.Execute(ctx, request, onOwner)
	})
	result, err := runner.runReserved(context.Background(), reserved, nil)
	if err == nil || result.Outcome != "timeout" {
		t.Fatalf("late owner acknowledgement: %+v %v", result, err)
	}
	runs := fake.Runs()
	if len(runs) != 1 || runs[0].Authorized {
		t.Fatalf("expired owner authorized: %+v", runs)
	}
	ledger, err := store.View(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	state := ledger.Tasks[reserved.Key.ID()]
	if state.Active == nil || state.Active.Stage != "notifying" || state.Last.Outcome != "timeout" {
		t.Fatalf("late timeout not retained: %+v", state)
	}
}

func TestScheduleReservedCatalogueCancellationPublishesTerminalNotice(t *testing.T) {
	for _, change := range []string{"cancel", "revision", "removed"} {
		t.Run(change, func(t *testing.T) {
			runner, fake, reserved, store := reservedExecutionFixture(t)
			fake.Enqueue(couchschedule.ExecutionResult{Exit: 10, Outcome: "wake", Output: []byte("completed before cancellation joined")})
			fake.Hold()
			defer fake.Release()
			type completion struct {
				result couchschedule.ExecutionResult
				err    error
			}
			done := make(chan completion, 1)
			go func() {
				result, err := runner.runReserved(t.Context(), reserved, nil)
				done <- completion{result, err}
				close(done)
			}()
			defer func() {
				fake.Release()
				select {
				case <-done:
				case <-time.After(5 * time.Second):
					t.Error("held fake execution failed to join")
				}
			}()
			select {
			case <-fake.Started():
			case <-time.After(5 * time.Second):
				t.Fatal("reserved execution did not start")
			}
			var declaration *couchschedule.Declaration
			if change == "revision" {
				path := filepath.Join(reserved.Key.Hub, "scripts/scheduled/report/schedule")
				if err := os.WriteFile(path, []byte(`{"version":1,"cron":"*/2 * * * *","timeout":"1m","script_run":true}`), 0600); err != nil {
					t.Fatal(err)
				}
				current, err := manualDeclaration(t.Context(), reserved.Key.Hub, reserved.Key.Task)
				if err != nil {
					t.Fatal(err)
				}
				declaration = &current
			}
			var changed couchschedule.TaskState
			if err := store.Update(t.Context(), func(l *couchschedule.Ledger) error {
				state := l.Tasks[reserved.Key.ID()]
				event := couchschedule.Event{Kind: couchschedule.CatalogChanged, At: runner.now(), Declaration: declaration, Location: time.UTC, Attempt: reserved.Active.ID, Revision: reserved.Active.Revision}
				if change == "cancel" {
					event.Kind = couchschedule.CancelRequested
				}
				state, _ = couchschedule.Advance(state, event)
				if state.Active == nil || state.Active.Stage != "stopping" {
					return errors.New("catalogue change did not stop held execution")
				}
				changed = state
				l.Tasks[state.Key.ID()] = state
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			fake.Release()
			var got completion
			select {
			case got = <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("completion did not join")
			}
			if got.err != nil || got.result.Outcome != "wake" {
				t.Fatalf("confirmed execution was lost: %+v %v", got.result, got.err)
			}
			key := operationalrun.RunKey{Hub: reserved.Key.HubID(), Task: reserved.Key.Task, ID: reserved.Active.ID}
			record, output, err := (operationalrun.Store{Namespace: runner.namespace}).Read(t.Context(), key)
			if err != nil || record.Notice != "dropped: task cancelled" || record.Outcome != "wake" || string(output) != "completed before cancellation joined" {
				t.Fatalf("cancelled task left nonterminal history: %+v %q %v", record, output, err)
			}
			ledger, err := store.View(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			state := ledger.Tasks[reserved.Key.ID()]
			if state.Active != nil || state.Last == nil || state.Last.Notice != record.Notice || !state.NextDue.Equal(changed.NextDue) {
				t.Fatalf("history/authority diverged after catalogue change: %+v", state)
			}
			if _, err = runner.runReserved(t.Context(), reserved, nil); err == nil {
				t.Fatal("cancelled reservation replay accepted")
			}
			if len(fake.Runs()) != 1 {
				t.Fatal("cancelled completion replayed")
			}
		})
	}
}

func TestScheduleReservedCompletionRecordSurvivesLedgerFailure(t *testing.T) {
	runner, fake, reserved, store := reservedExecutionFixture(t)
	writes := 0
	injected := errors.New("completion ledger commit refused")
	runner.beforePersist = func() error {
		writes++
		if writes == 2 {
			return injected
		}
		return nil
	}
	fake.Enqueue(couchschedule.ExecutionResult{Exit: 10, Outcome: "wake", Output: []byte("saved before ledger commit")})
	result, err := runner.runReserved(t.Context(), reserved, nil)
	if !errors.Is(err, injected) || result.Outcome != "wake" {
		t.Fatalf("completion failure: %+v %v", result, err)
	}
	record, output, err := (operationalrun.Store{Namespace: runner.namespace}).Read(t.Context(), operationalrun.RunKey{Hub: reserved.Key.HubID(), Task: reserved.Key.Task, ID: reserved.Active.ID})
	if err != nil || record.Outcome != "wake" || record.Notice != "pending" || string(output) != "saved before ledger commit" {
		t.Fatalf("completion evidence lost: %+v %q %v", record, output, err)
	}
	ledger, err := store.View(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	state := ledger.Tasks[reserved.Key.ID()]
	if state.Active == nil || state.Active.Stage != "running" || state.Last != nil {
		t.Fatalf("failed ledger commit unexpectedly cleared authority: %+v", state)
	}
	c := newScheduleCoordinator(runner.namespace, []string{reserved.Key.Hub}, time.UTC, runner.now)
	a := newAutomaticScheduler(c, runner, offlineScheduleNotices{})
	if err = a.reconcileExecution(t.Context(), scheduleJobResult{job: scheduleJob{state: reserved}}); err != nil {
		t.Fatal(err)
	}
	ledger, err = store.View(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	state = ledger.Tasks[reserved.Key.ID()]
	if state.Active == nil || state.Active.Stage != "notifying" || state.Last == nil || state.Last.Outcome != "wake" || len(fake.Runs()) != 1 {
		t.Fatalf("saved result recovery replayed/lost execution: %+v", state)
	}
}
