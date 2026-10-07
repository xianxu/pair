package couchcore

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

// Recover is the switcher's one "do the right recovery" action (#399 Task 9b):
// it runs exactly the actor steps the recovery report computes for a row --
// [resume], [reap, resume] or [reboot] -- so the switcher and the report cannot
// disagree. It never asks: choosing it is the consent (operator, 2026-10-07).

// RecoverTarget names the row to recover: a slot by its host checkout, or a
// thread by its exact address -- the forms reap and reboot take.
type RecoverTarget struct {
	Path    string
	Address ThreadAddress
}

// Recover refusal codes: why a recover ran no step, or stopped before the next.
const (
	RecoverNoThread = "no-thread" // no Couch row stands for the target
	RecoverHeld     = "held"      // the report holds the row
	RecoverChanged  = "changed"   // the row stopped admitting the next step
)

// RecoverRefusal is a recover that ran no step (or stopped before the next
// one because the row no longer admitted it), and why.
type RecoverRefusal struct {
	Code   string
	Detail string
}

func (r *RecoverRefusal) Error() string { return "recover refused (" + r.Code + "): " + r.Detail }

// recoverSteps are the report steps recover runs: the actor operations. The
// report's other steps (ask-agent-restore, reconcile) are not recover's: resume
// and reboot reconcile the workspace first, and the restore request is a
// message to the slot's agent, which the report prints.
var recoverSteps = []string{"reap", "resume", "reboot"}

// RecoverOffered is the switcher's and the socket's offer rule: recover
// appears wherever the admission table offers any actor operation. Cheap: the
// report runs only when recover is chosen.
func RecoverOffered(actions []string) bool { return len(actions) > 0 }

// deriveRecoverSteps decides recover for one row from its report row (nil
// when no report row stands for the thread, e.g. a thread outside the enrolled
// repositories). Pure. hold non-empty means recover does nothing, and says why.
//
// A held report row holds recover. A report row with actor steps gives them.
// Otherwise -- no report row, or one that suggests nothing because nothing is
// at risk (idle, landed) -- the same rule over ActorActions decides: an offered
// reap is [reap, resume], an offered resume is [resume], reboot alone is
// [reboot], and nothing offered holds.
func deriveRecoverSteps(row ActionableThreadSummary, report *RecoverRow) (steps []string, hold string) {
	if report != nil {
		if len(report.Next.Hold) > 0 {
			hold := strings.Join(report.Next.Hold, ", ")
			if report.Reason != "" {
				hold += ": " + report.Reason
			}
			return nil, hold
		}
		for _, step := range report.Next.Steps {
			if slices.Contains(recoverSteps, step.Action) {
				steps = append(steps, step.Action)
			}
		}
	}
	if len(steps) > 0 {
		return steps, ""
	}
	offered := ActorActions(ActorRowFactsOf(row))
	switch {
	case slices.Contains(offered, "reap"):
		return []string{"reap", "resume"}, ""
	case slices.Contains(offered, "resume"):
		return []string{"resume"}, ""
	case slices.Contains(offered, "reboot"):
		return []string{"reboot"}, ""
	}
	return nil, string(HoldNoActorAction) + ": " + row.Label() + " is " + rowStateWord(row)
}

// recoverReportRowFor finds the report row standing for a switcher row: a slot
// row by its host checkout, a repository's :0 by its primary path (the row
// IsPrimaryRow accepts). Nil when none does. Pure.
func recoverReportRowFor(plan RecoverPlan, row ActionableThreadSummary) *RecoverRow {
	for i := range plan.Rows {
		candidate := &plan.Rows[i]
		path := filepath.Clean(candidate.Path)
		if row.Target.Kind == ThreadTargetSlot {
			if filepath.Clean(row.Target.Slot.WorktreeRoot) == path {
				return candidate
			}
			continue
		}
		if _, number := splitRecoverAddress(candidate.Address); number != 0 {
			continue
		}
		if scope, err := launcher.ResolveRepoScope(path); err == nil && IsPrimaryRow(row, path, scope.Key) {
			return candidate
		}
	}
	return nil
}

func recoverRow(rows []ActionableThreadSummary, target RecoverTarget) (ActionableThreadSummary, bool) {
	return reapRow(rows, ReapTarget(target))
}

// recoverSettle bounds how long Recover waits, between steps, for the row to
// admit the next one. A reap can end a server whose client Couch is hosting
// (the live orphan): the row reads live until that client's exit lands, and
// resume is admitted only after it does.
const (
	recoverSettle     = 5 * time.Second
	recoverSettlePoll = 200 * time.Millisecond
)

// Recover derives the row's steps and runs them in order through the declared
// operations, re-reading the row before each (after a reap the row is no
// longer orphaned, and resume must address the row as it now is). It stops at
// the first error and returns the last step's result unchanged, so a resume's
// or reboot's started child is adopted exactly as theirs is.
//
// ORDER (ARCH-ORDER): a hold refuses before the first step. The report runs
// `sdlc` (seconds), which is why recover is a live-owner operation run off the
// UI thread like every other.
func (c *Couch) Recover(ctx context.Context, target RecoverTarget) (any, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	rows, err := c.ActionableThreadInventoryContext(ctx, nil)
	if err != nil {
		return nil, err
	}
	row, ok := recoverRow(rows, target)
	if !ok {
		return nil, &RecoverRefusal{Code: RecoverNoThread, Detail: "no Couch thread stands for that target"}
	}
	plan, err := c.RecoverPlan(ctx)
	if err != nil {
		return nil, err
	}
	steps, hold := deriveRecoverSteps(row, recoverReportRowFor(plan, row))
	if hold != "" {
		return nil, &RecoverRefusal{Code: RecoverHeld, Detail: hold}
	}
	sleep := c.sleep
	if sleep == nil {
		sleep = time.Sleep
	}
	executors := OperationExecutors{LiveOwner: CouchLiveOwnerExecutor(c)}
	var value any
	for i, step := range steps {
		row, err := c.awaitRecoverStep(ctx, target, step, sleep)
		if err == nil {
			value, err = DispatchOperation(executors, OperationCall{Name: step, Args: ActorOperationArgs(row, step), Implicit: true, Context: ctx})
		}
		if err != nil {
			if i == 0 {
				return nil, err
			}
			return nil, fmt.Errorf("recover: %s (step %d of %d, after %s): %w", step, i+1, len(steps), strings.Join(steps[:i], ", "), err)
		}
	}
	return value, nil
}

// awaitRecoverStep re-reads the row until it admits step, for at most
// recoverSettle: the previous step's effect (a hosted client exiting after its
// server was reaped) can land a moment after that step returned. The first
// step is admitted at once unless the row changed since the steps were derived.
func (c *Couch) awaitRecoverStep(ctx context.Context, target RecoverTarget, step string, sleep func(time.Duration)) (ActionableThreadSummary, error) {
	for waited := time.Duration(0); ; waited += recoverSettlePoll {
		rows, err := c.ActionableThreadInventoryContext(ctx, nil)
		if err != nil {
			return ActionableThreadSummary{}, err
		}
		row, ok := recoverRow(rows, target)
		if !ok {
			return ActionableThreadSummary{}, &RecoverRefusal{Code: RecoverNoThread, Detail: "no Couch thread stands for that target"}
		}
		if slices.Contains(ActorActions(ActorRowFactsOf(row)), step) {
			return row, nil
		}
		if waited >= recoverSettle || ctx.Err() != nil {
			return ActionableThreadSummary{}, &RecoverRefusal{Code: RecoverChanged, Detail: fmt.Sprintf("%s is %s, which does not admit %s; read the report again", row.Label(), rowStateWord(row), step)}
		}
		sleep(recoverSettlePoll)
	}
}
