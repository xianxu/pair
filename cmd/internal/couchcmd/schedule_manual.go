package couchcmd

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	err = store.Update(ctx, func(ledger *couchschedule.Ledger) error {
		state := ledger.Tasks[key.ID()]
		if state.Active != nil {
			return fmt.Errorf("task already active (%s, %s); inspect schedule status/history before repair", state.Active.ID, state.Active.Stage)
		}
		state, _ = couchschedule.Advance(state, couchschedule.Event{Kind: couchschedule.CatalogChanged, At: now, Declaration: &declaration, Location: time.Local})
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
	if reserved.Active == nil || reserved.Active.Stage != "reserved" || reserved.Active.ControllerPID != 0 || reserved.Active.Revision != reserved.Revision || !couchschedule.ValidTaskName(reserved.Key.Task) {
		return zero, errors.New("task execution requires an unowned reserved attempt")
	}
	attempt := *reserved.Active
	if attempt.Source != "manual" && attempt.Source != "scheduled" {
		return zero, errors.New("invalid task reservation source")
	}
	if attempt.Source == "scheduled" && !reserved.Spec.ScriptRun {
		return zero, errors.New("notify-only reservation cannot execute a script")
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
	if !sameReservedAttempt(current, reserved) {
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
		return store.Update(ctx, func(ledger *couchschedule.Ledger) error {
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
	history := r.history
	if history == nil {
		history = operationalrun.Store{Namespace: r.namespace}
	}

	if result.Outcome == "unknown" {
		if err := history.Publish(cleanup, record.Key, record, result.Output); err != nil {
			_ = r.markUnknown(cleanup, store, reserved)
			return result, fmt.Errorf("run result publication uncertain; no replay: %w", err)
		}
		_ = r.markUnknown(cleanup, store, reserved)
		return result, errors.New("execution cleanup unknown; inspect retained owner and history")
	}
	var publicationErr error
	published := false
	err = store.Update(cleanup, func(ledger *couchschedule.Ledger) error {
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
		if publicationErr = history.Publish(cleanup, record.Key, record, result.Output); publicationErr != nil {
			return publicationErr
		}
		published = true
		ledger.Tasks[key.ID()] = next
		return nil
	})
	if err != nil {
		if publicationErr != nil {
			_ = r.markUnknown(cleanup, store, reserved)
			return result, fmt.Errorf("run result publication uncertain; no replay: %w", publicationErr)
		}
		if published {
			return result, fmt.Errorf("run record saved but authority completion uncertain; inspect history: %w", err)
		}
		return result, fmt.Errorf("run completion authority uncertain; no replay: %w", err)
	}

	return result, executeErr
}

func manualDeclaration(ctx context.Context, root, task string) (couchschedule.Declaration, error) {
	declarations, err := (couchschedule.Catalog{}).Discover(ctx, []string{root})
	if err != nil {
		return couchschedule.Declaration{}, err
	}
	for _, declaration := range declarations {
		if declaration.Key.Hub == root && declaration.Key.Task == task {
			return declaration, nil
		}
	}
	return couchschedule.Declaration{}, errors.New("scheduled task not found in the current hub")
}

func (r *manualTaskRunner) markUnknown(ctx context.Context, store couchschedule.StateStore, state couchschedule.TaskState) error {
	return store.Update(ctx, func(ledger *couchschedule.Ledger) error {
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
	history := r.history
	if history == nil {
		history = operationalrun.Store{Namespace: r.namespace}
	}
	if err := store.Update(cleanup, func(ledger *couchschedule.Ledger) error {
		current := ledger.Tasks[state.Key.ID()]
		if !sameReservedAttempt(current, state) {
			return errors.New("prelaunch completion no longer owns reserved attempt")
		}
		// Keep authorization and prelaunch completion mutually exclusive. A stale
		// losing callback must never publish refusal over a live winner's run ID.
		if err := history.Publish(cleanup, record.Key, record, nil); err != nil {
			return fmt.Errorf("pre-execution refusal publication failed: %w", err)
		}
		current, _ = couchschedule.Advance(current, couchschedule.Event{Kind: couchschedule.ExecutionRefused, At: ended, Attempt: state.Active.ID, Revision: state.Active.Revision, Notice: cause.Error(), Outcome: outcome, HubOnline: true})
		ledger.Tasks[state.Key.ID()] = current
		return nil
	}); err != nil {
		return couchschedule.ExecutionResult{Exit: -1}, err
	}
	return couchschedule.ExecutionResult{Exit: -1, Outcome: outcome, Error: cause.Error()}, cause
}

// Use the account authority rather than a selected Couch/XDG data directory:
// changing namespaces cannot create another pair of machine execution slots.
func schedulePermitsDir(rt Runtime) (string, error) {
	if isolated := rt.Getenv("COUCH_ISOLATED_ROOT"); isolated != "" {
		root, err := canonicalFuture(isolated)
		if err != nil {
			return "", err
		}
		return filepath.Join(root, "schedule-permits"), nil
	}
	lookup := localAccountHome
	if production, ok := rt.(OSRuntime); ok && production.accountHome != nil {
		lookup = production.accountHome
	}
	home, err := lookup()
	if err != nil {
		return "", err
	}
	root, err := canonicalFuture(home)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, ".local", "share", "pair-host", "schedule-permits"), nil
}

func sameReservedAttempt(current, reserved couchschedule.TaskState) bool {
	a, b := current.Active, reserved.Active
	return a != nil && b != nil && a.Stage == "reserved" && a.ControllerPID == 0 && a.ControllerStart == "" && current.Key == reserved.Key && current.Revision == reserved.Revision && current.Spec == reserved.Spec && current.Allowed && a.ID == b.ID && a.Revision == b.Revision && a.Source == b.Source && a.StartedAt.Equal(b.StartedAt) && a.Deadline.Equal(b.Deadline) && a.DueAt.Equal(b.DueAt)
}
func executionNotice(source, outcome string) string {
	if source == "manual" {
		return "manual"
	}
	if outcome == "quiet" {
		return "none"
	}
	if outcome == "unknown" {
		return "uncertain"
	}
	return "pending"
}
