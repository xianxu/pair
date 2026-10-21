package couchcmd

import (
	"context"
	"errors"
	"strings"

	"github.com/xianxu/pair/cmd/internal/couchschedule"
	"github.com/xianxu/pair/cmd/internal/operationalrun"
)

func ensureUncertainRuns(ctx context.Context, store couchschedule.StateStore, history operationalrun.Store) error {
	ledger, err := store.View(ctx)
	if err != nil {
		return err
	}
	needed := false
	for _, s := range ledger.Tasks {
		if s.Active != nil && s.Active.Stage == "uncertain" && s.Last != nil && s.Last.Outcome == "unknown" {
			needed = true
			break
		}
	}
	if !needed {
		return nil
	}
	return updateScheduleFixture(store, ctx, func(l *couchschedule.Ledger) error {
		for key, s := range l.Tasks {
			if s.Active == nil || s.Active.Stage != "uncertain" || s.Last == nil || s.Last.Outcome != "unknown" {
				continue
			}
			record, err := ensureRetainedRun(ctx, history, s, true)
			if err == nil && (record.Source != s.Active.Source || record.DeclarationRevision != s.Active.Revision) {
				err = errors.New("result identity differs from retained authority")
			}
			if err != nil {
				if err := quarantineTaskHistory(ctx, l, s, "unknown recovery", err); err != nil {
					return err
				}
				continue
			}
			if record.Notice == "pending" {
				if err := history.FinalizeNotice(ctx, record.Key, record.DeclarationRevision, "uncertain"); err != nil {
					if err := quarantineTaskHistory(ctx, l, s, "unknown notice finalization", err); err != nil {
						return err
					}
					continue
				}
				record.Notice = "uncertain"
			}
			summary := runSummaryForRecord(record)
			next, _ := couchschedule.Advance(s, couchschedule.NewRecoveryEvent(record.EndedAt, s.Active.ID, s.Active.Revision, summary))
			if next.Last == nil || next.Last.ID != record.Key.ID {
				if err := quarantineTaskHistory(ctx, l, s, "unknown adoption", errors.New("canonical uncertain history refused by task state")); err != nil {
					return err
				}
				continue
			}
			l.Tasks[key] = next
		}
		return nil
	})
}

// A synchronous caller still receives its task error, while other consumers can
// observe the same refusal through the shared status projection.

func persistTaskHistoryFailure(ctx context.Context, store couchschedule.StateStore, key couchschedule.TaskKey, operation string, cause error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err := updateScheduleFixture(store, ctx, func(l *couchschedule.Ledger) error {
		return quarantineTaskHistory(ctx, l, l.Tasks[key.ID()], operation, cause)
	}); err != nil {
		return err
	}
	return cause
}

// Catalogue refresh must not erase a visible history quarantine while the
// retained evidence is still unreadable. Existing repair rules handle wholly
// absent passive/unknown bundles; execution output is never reconstructed.

func retainedHistoryQuarantine(ctx context.Context, history operationalrun.Store, state couchschedule.TaskState) error {
	if state.Allowed || !strings.Contains(state.Refusal, " history ") {
		return nil
	}
	if state.Last != nil && (state.Active == nil || state.Last.ID == state.Active.ID) {
		_, err := ensureRetainedRun(ctx, history, state, retainedUnknownSummary(state))
		return err
	}
	if state.Active != nil {
		_, _, err := history.ReadDurable(ctx, operationalrun.RunKey{Hub: state.Key.HubID(), Task: state.Key.Task, ID: state.Active.ID})
		return err
	}
	return nil
}

// Legacy compatibility delegates repair to the production owner implementation.
func ensurePassiveRun(ctx context.Context, history operationalrun.Store, state couchschedule.TaskState) error {
	owner := &scheduleOwner{coordinator: &scheduleCoordinator{runs: history}}
	return owner.ensurePassiveRun(ctx, state)
}
func ensureRetainedRun(ctx context.Context, history operationalrun.Store, state couchschedule.TaskState, unknown bool) (operationalrun.Record, error) {
	owner := &scheduleOwner{coordinator: &scheduleCoordinator{runs: history}}
	return owner.ensureRetainedRun(ctx, state, unknown)
}
