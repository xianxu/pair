package couchcmd

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchschedule"
)

// Execution workers return observations; the namespace owner decides whether
// those observations may become durable authority or immutable run history.
type scheduleExecutionFacts struct {
	result couchschedule.ExecutionResult
	ended  time.Time
	err    error
}

type scheduleExecutionWorker struct {
	namespace, permitsDir string
	executor              couchschedule.TaskExecutor
	now                   func() time.Time
	withDeadline          func(context.Context, time.Time) (context.Context, context.CancelFunc)
}

func (w scheduleExecutionWorker) execute(ctx context.Context, state couchschedule.TaskState, args []string, ready func(context.Context, couchschedule.OwnerIdentity, couchschedule.Declaration) error) scheduleExecutionFacts {
	finish := func(result couchschedule.ExecutionResult, err error) scheduleExecutionFacts {
		return scheduleExecutionFacts{result: result, ended: w.now(), err: err}
	}
	refuse := func(err error) scheduleExecutionFacts {
		return finish(couchschedule.ExecutionResult{Exit: -1}, &couchschedule.ExecutionError{Err: err})
	}
	a := state.Active
	if a == nil || a.Stage != "reserved" || !couchschedule.ValidTaskName(state.Key.Task) || (a.Source != "manual" && a.Source != "scheduled") {
		return refuse(errors.New("execution requires a reserved occurrence"))
	}
	if a.Source == "scheduled" && !state.Spec.ScriptRun {
		return refuse(errors.New("notify-only occurrence cannot execute"))
	}
	if w.executor == nil || ready == nil {
		return refuse(errors.New("execution requires runtime and owner authorization"))
	}
	deadline := w.withDeadline
	if deadline == nil {
		deadline = context.WithDeadline
	}
	ctx, cancel := deadline(ctx, a.Deadline)
	defer cancel()
	if !w.now().Before(a.Deadline) {
		return refuse(context.DeadlineExceeded)
	}
	helper := ""
	if state.Spec.Heavy {
		workflow, err := filepath.EvalSymlinks(filepath.Join(state.Key.Hub, "Makefile.workflow"))
		if err != nil {
			return refuse(fmt.Errorf("resolve machine admission helper: %w", err))
		}
		helper = filepath.Join(filepath.Dir(workflow), "scripts", "test-lock.py")
	}
	request := couchschedule.ExecutionRequest{Namespace: w.namespace, HubRoot: state.Key.Hub, Task: state.Key.Task, RunID: a.ID, Revision: a.Revision, CacheDir: filepath.Join(w.namespace, "schedule", "cache"), PermitsDir: w.permitsDir, TestLockHelper: helper, Args: append([]string(nil), args...), Spec: state.Spec, Deadline: a.Deadline}
	result, err := w.executor.Execute(ctx, request, func(owner couchschedule.OwnerIdentity) error {
		declaration, err := manualDeclaration(ctx, state.Key.Hub, state.Key.Task)
		if err != nil {
			return err
		}
		if !declaration.Allowed || declaration.Revision != a.Revision || declaration.Spec != state.Spec {
			return errors.New("task declaration or owner changed before execution authorization")
		}
		return ready(ctx, owner, declaration)
	})
	result.Output = append([]byte(nil), result.Output...)
	return finish(result, err)
}
