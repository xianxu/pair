package couchcmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/xianxu/pair/cmd/internal/couchschedule"
	"github.com/xianxu/pair/cmd/internal/operationalrun"
)

type taskHistoryError struct{ cause error }

func (e *taskHistoryError) Error() string { return e.cause.Error() }

func (e *taskHistoryError) Unwrap() error { return e.cause }

// quarantineTaskHistory handles only task-scoped history failures, inside a
// ledger transaction. The caller supplies the state whose authority and facts
// must survive. Ledger access/commit errors are handled by the transaction.

func quarantineTaskHistory(ctx context.Context, ledger *couchschedule.Ledger, state couchschedule.TaskState, operation string, err error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	state.Allowed = false
	state.Refusal = fmt.Sprintf("task %s history %s: %v", state.Key.Task, operation, err)
	if len(state.Refusal) > 4096 {
		state.Refusal = state.Refusal[:4096]
	}
	ledger.Tasks[state.Key.ID()] = state
	return nil
}

// Only a wholly absent bundle permits recovery from retained passive/unknown
// facts. A missing record or output member is damage, never reconstruction.

func missingRunBundle(namespace string, key operationalrun.RunKey, err error) bool {
	if !errors.Is(err, os.ErrNotExist) {
		return false
	}
	_, statErr := os.Lstat(filepath.Join(namespace, "schedule", key.Hub, key.Task, "runs", key.ID))
	return errors.Is(statErr, os.ErrNotExist)
}

func runSummaryForRecord(r operationalrun.Record) couchschedule.RunSummary {
	return couchschedule.RunSummary{ID: r.Key.ID, Revision: r.DeclarationRevision, Source: r.Source, DueAt: r.ScheduledAt, StartedAt: r.StartedAt, EndedAt: r.EndedAt, Exit: r.Exit, Outcome: r.Outcome, Notice: r.Notice, ManualIntentDigest: r.ManualIntentDigest}
}

// ensurePassiveRun is called under the existing state lock before replacing a
// retained summary. Its input facts must already be in the committed ledger.
// Native output cannot be reconstructed; only no-execution outcomes are repaired.

func (o *scheduleOwner) ensurePassiveRun(ctx context.Context, state couchschedule.TaskState) error {
	if state.Last == nil || state.Last.Source != "scheduled" || (state.Last.Outcome != "skipped" && state.Last.Outcome != "refused" && state.Last.Outcome != "completed") {
		return nil
	}
	_, err := o.ensureRetainedRun(ctx, state, false)
	return err
}

func (o *scheduleOwner) ensureRetainedRun(ctx context.Context, state couchschedule.TaskState, unknown bool) (operationalrun.Record, error) {
	history := o.coordinator.runs
	last := state.Last
	if last == nil {
		return operationalrun.Record{}, errors.New("missing retained run facts")
	}
	key := operationalrun.RunKey{Hub: state.Key.HubID(), Task: state.Key.Task, ID: last.ID}
	record, _, err := history.ReadDurable(ctx, key)
	if err == nil {
		return record, nil
	}
	if !missingRunBundle(history.Namespace, key, err) {
		return record, err
	}
	passive := last.Source == "scheduled" && (last.Outcome == "skipped" || last.Outcome == "refused" || last.Outcome == "completed")
	if !passive && !(unknown && last.Outcome == "unknown" && state.Active != nil && state.Active.Stage == "uncertain" && state.Active.ID == last.ID) {
		return record, fmt.Errorf("retained execution history unavailable; no facts or output reconstructed: %w", err)
	}
	revision := last.Revision
	if revision == "" && state.Active != nil && state.Active.ID == last.ID {
		revision = state.Active.Revision
	}
	if revision == "" {
		return record, errors.New("retained passive occurrence has no declaration revision; inspect history before repair")
	}
	proposed := operationalrun.Record{Version: 1, Key: key, DeclarationRevision: revision, Source: last.Source, ScheduledAt: last.DueAt, StartedAt: last.StartedAt, EndedAt: last.EndedAt, Exit: last.Exit, Outcome: last.Outcome, Notice: last.Notice, ManualIntentDigest: last.ManualIntentDigest}
	record, _, err = history.PublishOccurrence(ctx, key, proposed, nil)
	return record, err
}
