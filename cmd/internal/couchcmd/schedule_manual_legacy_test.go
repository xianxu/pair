package couchcmd

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchschedule"
	"github.com/xianxu/pair/cmd/internal/operationalrun"
	"github.com/xianxu/pair/cmd/internal/rowtext"
)

func runScheduleManual(ctx context.Context, a *scheduleArgs, rt Runtime, root string, stdout, stderr io.Writer) int {
	permits, err := schedulePermitsDir(rt)
	if err != nil {
		fmt.Fprintln(stderr, "couch schedule:", err)
		return 1
	}
	var outputErr error
	executor := couchschedule.Executor{OnOutput: func(output []byte) {
		if outputErr == nil {
			_, outputErr = stdout.Write(output)
		}
	}}
	runner := manualTaskRunner{namespace: rt.StoreDir(), permitsDir: permits, executor: executor, now: time.Now, newID: func() (string, error) {
		var bytes [16]byte
		if _, err := rand.Read(bytes[:]); err != nil {
			return "", err
		}
		return hex.EncodeToString(bytes[:]), nil
	}}
	result, err := runner.run(ctx, root, a.task, a.args)
	if outputErr != nil {
		fmt.Fprintln(stderr, "couch schedule: output:", outputErr)
		return 1
	}
	if err != nil {
		fmt.Fprintln(stderr, "couch schedule:", rowtext.Sanitize(err.Error()))
		return 1
	}
	if result.Error != "" {
		fmt.Fprintln(stderr, "couch schedule:", rowtext.Sanitize(result.Error))
	}
	if result.Exit < 0 || result.Exit > 255 {
		return 1
	}
	return result.Exit
}

type runPublisher interface {
	Publish(context.Context, operationalrun.RunKey, operationalrun.Record, []byte) error
}

type runOccurrencePublisher interface {
	PublishOccurrence(context.Context, operationalrun.RunKey, operationalrun.Record, []byte) (operationalrun.Record, []byte, error)
}

type runReader interface {
	Read(context.Context, operationalrun.RunKey) (operationalrun.Record, []byte, error)
}

type durableRunReader interface {
	ReadDurable(context.Context, operationalrun.RunKey) (operationalrun.Record, []byte, error)
}

type runNoticeFinalizer interface {
	FinalizeNotice(context.Context, operationalrun.RunKey, string, string) error
}

type manualTaskRunner struct {
	namespace, permitsDir string
	executor              couchschedule.TaskExecutor
	history               runPublisher
	now                   func() time.Time
	newID                 func() (string, error)
	withDeadline          func(context.Context, time.Time) (context.Context, context.CancelFunc)
	beforePersist         func() error
}

func (r *manualTaskRunner) run(ctx context.Context, root, task string, args []string) (couchschedule.ExecutionResult, error) {
	zero := couchschedule.ExecutionResult{Exit: -1}
	if !couchschedule.ValidTaskName(task) {
		return zero, errors.New("invalid task name")
	}
	declaration, err := manualDeclaration(ctx, root, task)
	if err != nil {
		return zero, err
	}
	if !declaration.Allowed {
		return zero, fmt.Errorf("task refused: %s", declaration.Reason)
	}
	id, err := r.newID()
	if err != nil {
		return zero, err
	}
	now := r.now()
	deadline := now.Add(declaration.Spec.Timeout)
	ctx, cancel := r.executionContext(ctx, deadline)
	defer cancel()
	store := couchschedule.StateStore{Dir: filepath.Join(r.namespace, "schedule"), BeforePersist: r.beforePersist}
	key := declaration.Key
	var reserved couchschedule.TaskState
	var historyErr error
	err = updateScheduleFixture(store, ctx, func(ledger *couchschedule.Ledger) error {
		state := ledger.Tasks[key.ID()]
		if state.Active != nil {
			return fmt.Errorf("task already active (%s, %s); inspect schedule status/history before repair", state.Active.ID, state.Active.Stage)
		}
		state, _ = couchschedule.Advance(state, couchschedule.Event{Kind: couchschedule.CatalogChanged, At: now, Declaration: &declaration, Location: time.Local})
		if err := ensurePassiveRun(ctx, r.nativeHistory(), state); err != nil {
			historyErr = err
			return quarantineTaskHistory(ctx, ledger, state, "manual replacement", err)
		}
		state, _ = couchschedule.Advance(state, couchschedule.Event{Kind: couchschedule.ManualRequested, At: now, RunID: id})
		if state.Active == nil || state.Active.ID != id {
			return errors.New("manual task reservation refused")
		}
		reserved = state
		ledger.Tasks[key.ID()] = state
		return nil
	})
	if err != nil {
		return zero, err
	}
	if historyErr != nil {
		return zero, historyErr
	}
	return r.runReserved(ctx, reserved, args)
}

func (r *manualTaskRunner) executionContext(ctx context.Context, deadline time.Time) (context.Context, context.CancelFunc) {
	if r.withDeadline != nil {
		return r.withDeadline(ctx, deadline)
	}
	return context.WithDeadline(ctx, deadline)
}

// runReserved executes exactly one already-persisted reservation. It never
// allocates an ID or advances recurrence; completion publication precedes the
// authority transition that permits a notice or clears the attempt.

func (r *manualTaskRunner) runReserved(ctx context.Context, reserved couchschedule.TaskState, args []string) (couchschedule.ExecutionResult, error) {
	zero := couchschedule.ExecutionResult{Exit: -1}
	if reserved.Active == nil || !couchschedule.ValidTaskName(reserved.Key.Task) {
		return zero, errors.New("task execution requires an unowned reserved attempt")
	}
	attempt := *reserved.Active
	if attempt.Source != "manual" && attempt.Source != "scheduled" {
		return zero, errors.New("invalid task reservation source")
	}
	ctx, cancel := r.executionContext(ctx, attempt.Deadline)
	defer cancel()
	store := couchschedule.StateStore{Dir: filepath.Join(r.namespace, "schedule"), BeforePersist: r.beforePersist}
	expired := !r.now().Before(attempt.Deadline)
	viewCtx := ctx
	var viewStop context.CancelFunc
	if expired {
		viewCtx, viewStop = context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer viewStop()
	}
	ledger, err := store.View(viewCtx)
	if err != nil && errors.Is(err, context.DeadlineExceeded) && !r.now().Before(attempt.Deadline) {
		expired = true
		viewCtx, viewStop = context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer viewStop()
		ledger, err = store.View(viewCtx)
	}
	if err != nil {
		return zero, err
	}
	current := ledger.Tasks[reserved.Key.ID()]
	history := r.historyPort()
	runKey := operationalrun.RunKey{Hub: reserved.Key.HubID(), Task: reserved.Key.Task, ID: attempt.ID}
	reader, durable := history.(durableRunReader)
	_, readable := history.(runReader)
	if durable || readable {
		if !durable {
			return zero, persistTaskHistoryFailure(viewCtx, store, reserved.Key, "execution evidence", errors.New("existing run reader cannot confirm durability; no replay"))
		}
		record, output, readErr := reader.ReadDurable(viewCtx, runKey)
		if readErr == nil {
			return r.adoptStoredOccurrence(viewCtx, store, reserved, record, output)
		}
		if !errors.Is(readErr, os.ErrNotExist) {
			return zero, persistTaskHistoryFailure(viewCtx, store, reserved.Key, "execution evidence", fmt.Errorf("existing run evidence unavailable; no replay: %w", readErr))
		}
		switch history.(type) {
		case operationalrun.Store, *operationalrun.Store:
			if !missingRunBundle(r.namespace, runKey, readErr) {
				return zero, persistTaskHistoryFailure(viewCtx, store, reserved.Key, "execution evidence", errors.New("existing run bundle is damaged; no replay"))
			}
		}
	}
	if attempt.Source == "scheduled" && !reserved.Spec.ScriptRun {
		return zero, errors.New("notify-only reservation cannot execute a script")
	}
	if !sameReservedAttempt(current, reserved) || attempt.Revision != reserved.Revision {
		return zero, errors.New("task reservation changed before execution")
	}
	if expired {
		return r.finishPrelaunch(viewCtx, store, reserved, context.DeadlineExceeded, "timeout")
	}
	root, task, id, key, deadline := reserved.Key.Hub, reserved.Key.Task, attempt.ID, reserved.Key, attempt.Deadline
	declaration := couchschedule.Declaration{Key: key, Revision: attempt.Revision, Spec: reserved.Spec, Allowed: reserved.Allowed}
	helper := ""
	if declaration.Spec.Heavy {
		workflow, e := filepath.EvalSymlinks(filepath.Join(root, "Makefile.workflow"))
		if e == nil {
			helper = filepath.Join(filepath.Dir(workflow), "scripts", "test-lock.py")
		}
		if e != nil {
			return r.finishRefused(ctx, store, reserved, fmt.Errorf("resolve machine admission helper: %w", e))
		}
	}
	request := couchschedule.ExecutionRequest{Namespace: r.namespace, HubRoot: root, Task: task, RunID: id, Revision: declaration.Revision, CacheDir: filepath.Join(r.namespace, "schedule", "cache"), PermitsDir: r.permitsDir, TestLockHelper: helper, Args: append([]string(nil), args...), Spec: declaration.Spec, Deadline: deadline}
	result, executeErr := r.executor.Execute(ctx, request, func(owner couchschedule.OwnerIdentity) error {
		current, err := manualDeclaration(ctx, root, task)
		if err != nil {
			return err
		}
		if !current.Allowed || current.Revision != declaration.Revision || current.Spec != declaration.Spec {
			return errors.New("task declaration or owner changed before execution authorization")
		}
		return updateScheduleFixture(store, ctx, func(ledger *couchschedule.Ledger) error {
			state := ledger.Tasks[key.ID()]
			if !sameReservedAttempt(state, reserved) {
				return errors.New("task reservation changed")
			}
			state, _ = couchschedule.Advance(state, couchschedule.Event{Kind: couchschedule.OwnerRecorded, At: r.now(), Attempt: id, Revision: declaration.Revision, Owner: &owner})
			if state.Active == nil || state.Active.ControllerPID <= 0 || state.Active.ControllerStart == "" || state.Active.ControllerPID != owner.PID || state.Active.ControllerStart != owner.Start {
				return errors.New("execution owner observation refused")
			}
			state, effects := couchschedule.Advance(state, couchschedule.Event{Kind: couchschedule.ReservationDurable, At: r.now(), Attempt: id, Revision: declaration.Revision})
			if len(effects) != 1 || effects[0].Kind != couchschedule.StartTask {
				return errors.New("task authorization expired")
			}
			ledger.Tasks[key.ID()] = state
			return nil
		})
	})
	cleanup, stop := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer stop()
	if executeErr != nil && result.Outcome == "" {
		var executionError *couchschedule.ExecutionError
		if errors.As(executeErr, &executionError) && !executionError.MayHaveStarted {
			outcome := "refused"
			if !r.now().Before(attempt.Deadline) {
				outcome = "timeout"
			}
			return r.finishPrelaunch(cleanup, store, reserved, executeErr, outcome)
		}
		_ = r.markUnknown(cleanup, store, reserved)
		return result, fmt.Errorf("execution result unknown; no replay: %w", executeErr)
	}
	if result.Outcome == "" {
		result.Outcome = "failed"
		if result.Exit == 0 {
			result.Outcome = "quiet"
		}
		if result.Exit == 10 {
			result.Outcome = "wake"
		}
	}
	ended := r.now()
	if result.Outcome == "uncertain" {
		result.Outcome = "unknown"
	}
	var exit *int
	if result.Exit >= 0 {
		code := result.Exit
		exit = &code
	}
	record := operationalrun.Record{Version: 1, Key: operationalrun.RunKey{Hub: key.HubID(), Task: task, ID: id}, DeclarationRevision: declaration.Revision, Source: attempt.Source, ScheduledAt: attempt.DueAt, StartedAt: attempt.StartedAt, EndedAt: ended, Exit: exit, Outcome: result.Outcome, Notice: executionNotice(attempt.Source, result.Outcome)}
	record.Unknown = map[string]json.RawMessage{}
	bytes := result.OutputBytes
	if bytes < int64(len(result.Output)) {
		bytes = int64(len(result.Output))
	}
	record.Unknown["output_bytes"], _ = json.Marshal(bytes)
	record.Unknown["build_key"], _ = json.Marshal(result.BuildKey)
	if result.Error != "" {
		record.Unknown["error"], _ = json.Marshal(result.Error)
	}
	if result.Outcome == "unknown" {
		canonical, output, err := publishRunOccurrence(cleanup, history, record, result.Output)
		if err != nil {
			_ = r.markUnknown(cleanup, store, reserved)
			return result, persistTaskHistoryFailure(cleanup, store, reserved.Key, "completion publication", fmt.Errorf("run result publication uncertain; no replay: %w", err))
		}
		return r.adoptStoredOccurrence(cleanup, store, reserved, canonical, output)
	}
	var publicationErr error
	published := false
	err = updateScheduleFixture(store, cleanup, func(ledger *couchschedule.Ledger) error {
		state := ledger.Tasks[key.ID()]
		active := state.Active
		if active == nil || active.ID != attempt.ID || active.Revision != attempt.Revision || active.Source != attempt.Source || !active.StartedAt.Equal(attempt.StartedAt) || !active.DueAt.Equal(attempt.DueAt) || !active.Deadline.Equal(attempt.Deadline) {
			return errors.New("completion no longer owns reservation")
		}
		next, _ := couchschedule.Advance(state, couchschedule.Event{Kind: couchschedule.TaskFinished, At: ended, Attempt: id, Revision: declaration.Revision, Exit: result.Exit, Outcome: result.Outcome, HubOnline: true})
		if next.Last == nil || next.Last.ID != id || next.Last.Outcome != result.Outcome {
			return errors.New("execution completion transition refused")
		}
		if attempt.Source == "scheduled" {
			record.Notice = "dropped: task cancelled"
			if result.Outcome == "quiet" {
				record.Notice = "none"
			} else if next.Active != nil && next.Active.ID == id && next.Active.Stage == "notifying" {
				record.Notice = "pending"
			}
			next.Last.Notice = record.Notice
		}
		// Completion and catalogue cancellation serialize here: publication uses
		// the same transition that clears execution or grants notice authority.
		canonical, output, publishErr := publishRunOccurrence(cleanup, history, record, result.Output)
		publicationErr = publishErr
		if publishErr != nil {
			return publishErr
		}
		published = true
		canonical, err = finalizeCancelledOccurrence(cleanup, history, state, canonical)
		if err != nil {
			publicationErr = err
			return err
		}
		next, err = recoverRunOccurrence(state, canonical, r.now())
		if err != nil {
			publicationErr = err
			return err
		}
		result = executionResultForRecord(canonical, output)
		ledger.Tasks[key.ID()] = next
		return nil
	})
	if err != nil {
		if publicationErr != nil {
			_ = r.markUnknown(cleanup, store, reserved)
			return result, persistTaskHistoryFailure(cleanup, store, reserved.Key, "completion publication", fmt.Errorf("run result publication uncertain; no replay: %w", publicationErr))
		}
		if published {
			return result, fmt.Errorf("run record saved but authority completion uncertain; inspect history: %w", err)
		}
		return result, fmt.Errorf("run completion authority uncertain; no replay: %w", err)
	}

	if result.Outcome == "unknown" {
		return result, errors.New("execution result unknown; no replay")
	}
	return result, executeErr
}

func (r *manualTaskRunner) markUnknown(ctx context.Context, store couchschedule.StateStore, state couchschedule.TaskState) error {
	return updateScheduleFixture(store, ctx, func(ledger *couchschedule.Ledger) error {
		current := ledger.Tasks[state.Key.ID()]
		current, _ = couchschedule.Advance(current, couchschedule.Event{Kind: couchschedule.OwnerUnknown, At: r.now(), Attempt: state.Active.ID, Revision: state.Active.Revision})
		ledger.Tasks[state.Key.ID()] = current
		return nil
	})
}

func (r *manualTaskRunner) finishRefused(ctx context.Context, store couchschedule.StateStore, state couchschedule.TaskState, cause error) (couchschedule.ExecutionResult, error) {
	return r.finishPrelaunch(ctx, store, state, cause, "refused")
}

func (r *manualTaskRunner) finishPrelaunch(ctx context.Context, store couchschedule.StateStore, state couchschedule.TaskState, cause error, outcome string) (couchschedule.ExecutionResult, error) {
	cleanup, stop := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer stop()
	ended := r.now()
	record := operationalrun.Record{Version: 1, Key: operationalrun.RunKey{Hub: state.Key.HubID(), Task: state.Key.Task, ID: state.Active.ID}, DeclarationRevision: state.Active.Revision, Source: state.Active.Source, ScheduledAt: state.Active.DueAt, StartedAt: state.Active.StartedAt, EndedAt: ended, Outcome: outcome, Notice: executionNotice(state.Active.Source, outcome)}
	errorRaw, _ := json.Marshal(cause.Error())
	record.Unknown = map[string]json.RawMessage{"error": errorRaw}
	history := r.historyPort()
	result := couchschedule.ExecutionResult{Exit: -1}
	var historyErr error
	if err := updateScheduleFixture(store, cleanup, func(ledger *couchschedule.Ledger) error {
		current := ledger.Tasks[state.Key.ID()]
		if !sameReservedAttempt(current, state) {
			return errors.New("prelaunch completion no longer owns reserved attempt")
		}
		// Keep authorization and prelaunch completion mutually exclusive. A stale
		// losing callback must never publish refusal over a live winner's run ID.
		canonical, output, err := publishRunOccurrence(cleanup, history, record, nil)
		if err != nil {
			historyErr = err
			return quarantineTaskHistory(cleanup, ledger, current, "prelaunch publication", err)
		}
		next, err := recoverRunOccurrence(current, canonical, r.now())
		if err != nil {
			historyErr = err
			return quarantineTaskHistory(cleanup, ledger, current, "prelaunch adoption", err)
		}
		result = executionResultForRecord(canonical, output)
		ledger.Tasks[state.Key.ID()] = next
		return nil
	}); err != nil {
		return couchschedule.ExecutionResult{Exit: -1}, err
	}
	if historyErr != nil {
		return result, historyErr
	}
	if result.Outcome == "unknown" {
		return result, errors.New("execution result unknown; no replay")
	}
	if result.Outcome != outcome {
		return result, nil
	}
	if result.Error != "" && result.Error != cause.Error() {
		return result, errors.New(result.Error)
	}
	return result, cause
}

// Use the account authority rather than a selected Couch/XDG data directory:
// changing namespaces cannot create another pair of machine execution slots.

func (r *manualTaskRunner) nativeHistory() operationalrun.Store {
	switch history := r.historyPort().(type) {
	case operationalrun.Store:
		return history
	case *operationalrun.Store:
		return *history
	default:
		return operationalrun.Store{Namespace: r.namespace}
	}
}

func (r *manualTaskRunner) historyPort() runPublisher {
	if r.history != nil {
		return r.history
	}
	return operationalrun.Store{Namespace: r.namespace}
}

func publishRunOccurrence(ctx context.Context, history runPublisher, record operationalrun.Record, output []byte) (operationalrun.Record, []byte, error) {
	if publisher, ok := history.(runOccurrencePublisher); ok {
		return publisher.PublishOccurrence(ctx, record.Key, record, output)
	}
	if err := history.Publish(ctx, record.Key, record, output); err != nil {
		return operationalrun.Record{}, nil, err
	}
	return record, output, nil
}

func executionResultForRecord(record operationalrun.Record, output []byte) couchschedule.ExecutionResult {
	result := couchschedule.ExecutionResult{Exit: -1, Outcome: record.Outcome, Output: append([]byte(nil), output...), OutputBytes: int64(len(output))}
	if record.Exit != nil {
		result.Exit = *record.Exit
	}
	var total int64
	_ = json.Unmarshal(record.Unknown["output_bytes"], &total)
	if total > result.OutputBytes {
		result.OutputBytes = total
	}
	_ = json.Unmarshal(record.Unknown["build_key"], &result.BuildKey)
	_ = json.Unmarshal(record.Unknown["error"], &result.Error)
	if !record.Completed() {
		result.Outcome = "unknown"
	}
	return result
}

func validateOccurrenceAuthority(state couchschedule.TaskState, record operationalrun.Record) error {
	if state.Key.HubID() != record.Key.Hub || state.Key.Task != record.Key.Task {
		return errors.New("canonical occurrence differs from retained task")
	}
	if active := state.Active; active != nil {
		if active.ID == record.Key.ID && active.Source == record.Source && active.Revision == record.DeclarationRevision {
			return nil
		}
	} else if last := state.Last; last != nil && last.ID == record.Key.ID && last.Source == record.Source && (last.Revision == "" || last.Revision == record.DeclarationRevision) && record.Completed() && record.Notice != "pending" {
		return nil
	}
	return errors.New("canonical occurrence differs from retained authority")
}

func recoverRunOccurrence(state couchschedule.TaskState, record operationalrun.Record, at time.Time) (couchschedule.TaskState, error) {
	if err := validateOccurrenceAuthority(state, record); err != nil {
		return state, err
	}
	summary := runSummaryForRecord(record)
	next, _ := couchschedule.Advance(state, couchschedule.NewRecoveryEvent(at, record.Key.ID, record.DeclarationRevision, summary))
	if next.Last == nil || next.Last.ID != record.Key.ID || next.Last.Revision != record.DeclarationRevision || !next.Last.StartedAt.Equal(summary.StartedAt) || !next.Last.EndedAt.Equal(summary.EndedAt) || !next.Last.DueAt.Equal(summary.DueAt) || next.Last.Outcome != summary.Outcome {
		return state, errors.New("canonical occurrence recovery refused")
	}
	return next, nil
}

func finalizeCancelledOccurrence(ctx context.Context, history runPublisher, state couchschedule.TaskState, record operationalrun.Record) (operationalrun.Record, error) {
	if err := validateOccurrenceAuthority(state, record); err != nil {
		return record, err
	}
	if state.Active != nil && record.Source == "scheduled" && record.Notice == "pending" && (state.Active.Stage == "stopping" || state.Active.Revision != state.Revision) {
		finalizer, ok := history.(runNoticeFinalizer)
		if !ok {
			return record, errors.New("cancelled canonical occurrence needs durable notice finalization")
		}
		if err := finalizer.FinalizeNotice(ctx, record.Key, record.DeclarationRevision, "dropped: task cancelled"); err != nil {
			return record, err
		}
		record.Notice = "dropped: task cancelled"
	}
	return record, nil
}

func (r *manualTaskRunner) adoptStoredOccurrence(ctx context.Context, store couchschedule.StateStore, reserved couchschedule.TaskState, record operationalrun.Record, output []byte) (couchschedule.ExecutionResult, error) {
	result := executionResultForRecord(record, output)
	var historyErr error
	err := updateScheduleFixture(store, ctx, func(ledger *couchschedule.Ledger) error {
		state := ledger.Tasks[reserved.Key.ID()]
		var err error
		record, err = finalizeCancelledOccurrence(ctx, r.historyPort(), state, record)
		if err != nil {
			historyErr = err
			return quarantineTaskHistory(ctx, ledger, state, "canonical finalization", err)
		}
		next, err := recoverRunOccurrence(state, record, r.now())
		if err != nil {
			historyErr = err
			return quarantineTaskHistory(ctx, ledger, state, "canonical adoption", err)
		}
		ledger.Tasks[state.Key.ID()] = next
		return nil
	})
	if err != nil {
		return result, fmt.Errorf("canonical occurrence adoption failed; no replay: %w", err)
	}
	if historyErr != nil {
		return result, historyErr
	}
	if result.Outcome == "unknown" {
		return result, errors.New("execution result unknown; no replay")
	}
	return result, nil
}
