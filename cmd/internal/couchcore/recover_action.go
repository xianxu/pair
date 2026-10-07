package couchcore

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

// Recover is the switcher's one "do the right recovery" action (#399 Task 9b):
// it runs exactly the actor steps the recovery report computes for a row --
// [resume], [reap, resume] or [reboot] -- so the switcher and the report cannot
// disagree. Its confirmation follows those steps (ConfirmByPlan).

// RecoverTarget names the row to recover: a slot by its host checkout, or a
// thread by its exact address -- the forms reap and reboot take.
type RecoverTarget struct {
	Path    string
	Address ThreadAddress
}

// RecoverPreview is what recover will do on a row, derived from the report.
// Hold non-empty means it will do nothing, and says why.
type RecoverPreview struct {
	Steps []string
	// Confirm is true exactly when Steps contains a destructive step (reap or
	// reboot).
	Confirm bool
	// Text is one sentence naming what will happen.
	Text string
	Hold string
}

// Recover refusal codes: why a recover ran no step.
const (
	RecoverNoThread    = "no-thread"             // no Couch row stands for the target
	RecoverHeld        = "held"                  // the report holds the row
	RecoverStale       = "stale"                 // the row's plan changed since the preview
	RecoverUnconfirmed = "confirmation-required" // a destructive plan without a confirmation
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

// DeriveRecoverPreview decides recover for one row from its report row (nil
// when no report row stands for the thread, e.g. a thread outside the enrolled
// repositories). Pure.
//
// A held report row holds recover. A report row with actor steps gives them.
// Otherwise -- no report row, or one that suggests nothing because nothing is
// at risk (idle, landed) -- the same rule over ActorActions decides: an offered
// reap is [reap, resume], an offered resume is [resume], reboot alone is
// [reboot], and nothing offered holds.
func DeriveRecoverPreview(row ActionableThreadSummary, report *RecoverRow) RecoverPreview {
	var steps []string
	if report != nil {
		if len(report.Next.Hold) > 0 {
			hold := strings.Join(report.Next.Hold, ", ")
			if report.Reason != "" {
				hold += ": " + report.Reason
			}
			return RecoverPreview{Hold: hold}
		}
		for _, step := range report.Next.Steps {
			if slices.Contains(recoverSteps, step.Action) {
				steps = append(steps, step.Action)
			}
		}
	}
	if len(steps) == 0 {
		offered := ActorActions(ActorRowFactsOf(row))
		switch {
		case slices.Contains(offered, "reap"):
			steps = []string{"reap", "resume"}
		case slices.Contains(offered, "resume"):
			steps = []string{"resume"}
		case slices.Contains(offered, "reboot"):
			steps = []string{"reboot"}
		default:
			return RecoverPreview{Hold: string(HoldNoActorAction) + ": " + row.Label() + " is " + rowStateWord(row)}
		}
	}
	preview := RecoverPreview{Steps: steps, Confirm: slices.Contains(steps, "reap") || slices.Contains(steps, "reboot")}
	preview.Text = recoverText(row, steps)
	return preview
}

// recoverText is the one sentence naming what the steps do to the row.
func recoverText(row ActionableThreadSummary, steps []string) string {
	words := make([]string, 0, len(steps))
	for _, step := range steps {
		switch step {
		case "reap":
			if o := row.Orphan; o != nil {
				words = append(words, fmt.Sprintf("reap orphaned server PID %d and everything under it", o.PID))
			} else {
				words = append(words, "reap its orphaned server and everything under it")
			}
		case "resume":
			words = append(words, "resume "+row.Label())
		case "reboot":
			words = append(words, "archive this conversation and start a fresh agent")
		default:
			words = append(words, step)
		}
	}
	return strings.Join(words, ", then ")
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

// PrepareRecover previews recover on one row: the inventory row, then the
// report's decision for it. It runs `sdlc` through RecoverPlan (seconds), so a
// caller on a UI thread runs it off that thread (prepare-recover). It changes
// nothing.
func (c *Couch) PrepareRecover(ctx context.Context, target RecoverTarget) (RecoverPreview, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	rows, err := c.ActionableThreadInventoryContext(ctx, nil)
	if err != nil {
		return RecoverPreview{}, err
	}
	row, ok := recoverRow(rows, target)
	if !ok {
		return RecoverPreview{}, &RecoverRefusal{Code: RecoverNoThread, Detail: "no Couch thread stands for that target"}
	}
	plan, err := c.RecoverPlan(ctx)
	if err != nil {
		return RecoverPreview{}, err
	}
	return DeriveRecoverPreview(row, recoverReportRowFor(plan, row)), nil
}

// Recover re-derives the preview and runs its steps in order through the
// declared operations, re-reading the row before each (after a reap the row is
// no longer orphaned, and resume must address the row as it now is). It stops
// at the first error and returns the last step's result unchanged, so a
// resume's or reboot's started child is adopted exactly as theirs is.
//
// previewSteps are the steps the operator was shown; nil means none were (the
// CLI), and the confirmation is then plan-blind. ORDER (ARCH-ORDER): a hold, a
// stale preview and a missing confirmation all refuse before the first step.
func (c *Couch) Recover(ctx context.Context, target RecoverTarget, previewSteps []string, confirmed bool) (any, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	preview, err := c.PrepareRecover(ctx, target)
	if err != nil {
		return nil, err
	}
	switch {
	case preview.Hold != "":
		return nil, &RecoverRefusal{Code: RecoverHeld, Detail: preview.Hold}
	case previewSteps != nil && !slices.Equal(previewSteps, preview.Steps):
		return nil, &RecoverRefusal{Code: RecoverStale, Detail: fmt.Sprintf(
			"the row changed; review again (was %s, now %s)", strings.Join(previewSteps, " → "), strings.Join(preview.Steps, " → "))}
	case preview.Confirm && !confirmed:
		return nil, &RecoverRefusal{Code: RecoverUnconfirmed, Detail: "re-run with --confirm to " + preview.Text}
	}
	executors := OperationExecutors{LiveOwner: CouchLiveOwnerExecutor(c)}
	var value any
	for i, step := range preview.Steps {
		rows, err := c.ActionableThreadInventoryContext(ctx, nil)
		if err != nil {
			return nil, err
		}
		row, ok := recoverRow(rows, target)
		if !ok {
			return nil, &RecoverRefusal{Code: RecoverNoThread, Detail: fmt.Sprintf("no Couch thread stands for that target before %s (step %d of %d)", step, i+1, len(preview.Steps))}
		}
		if !slices.Contains(ActorActions(ActorRowFactsOf(row)), step) {
			return nil, &RecoverRefusal{Code: RecoverStale, Detail: fmt.Sprintf("%s is %s, which does not admit %s (step %d of %d); review again", row.Label(), rowStateWord(row), step, i+1, len(preview.Steps))}
		}
		value, err = DispatchOperation(executors, OperationCall{Name: step, Args: ActorOperationArgs(row, step), Implicit: true, Context: ctx})
		if err != nil {
			if i == 0 {
				return nil, err
			}
			return nil, fmt.Errorf("recover: %s (step %d of %d, after %s): %w", step, i+1, len(preview.Steps), strings.Join(preview.Steps[:i], ", "), err)
		}
	}
	return value, nil
}
