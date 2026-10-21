package couchcmd

import (
	"context"
	"errors"
	"path/filepath"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchschedule"
	"github.com/xianxu/pair/cmd/internal/operationalrun"
)

func (c *scheduleCoordinator) refresh(ctx context.Context) error {
	declarations, err := c.catalog.Discover(ctx, c.roots)
	if err != nil {
		return c.blocked(err)
	}
	now := c.now()
	var stopped []couchschedule.Attempt
	localRefusals := map[string]error{}
	err = updateScheduleFixture(c.store, ctx, func(ledger *couchschedule.Ledger) error {
		for key, state := range ledger.Tasks {
			if state.Active == nil || state.Active.Source != "manual" {
				continue
			}
			a := state.Active
			record, _, err := c.runs.ReadDurable(ctx, operationalrun.RunKey{Hub: state.Key.HubID(), Task: state.Key.Task, ID: a.ID})
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if err == nil {
				if record.Source != "manual" || record.DeclarationRevision != a.Revision {
					err = errors.New("manual result identity differs from retained authority")
				} else if record.Completed() || record.Outcome == "unknown" {
					summary := runSummaryForRecord(record)
					state, _ = couchschedule.Advance(state, couchschedule.NewRecoveryEvent(now, a.ID, a.Revision, summary))
					ledger.Tasks[key] = state
					continue
				}
			}
			// A wholly absent bundle has no completion evidence. A missing member
			// of an existing bundle is damage, not permission to forget authority.
			missing := missingRunBundle(filepath.Dir(c.store.Dir), operationalrun.RunKey{Hub: state.Key.HubID(), Task: state.Key.Task, ID: a.ID}, err)
			if err != nil && !missing {
				localRefusals[key] = err
				continue
			} else {
				// Empty/mismatching native identity is inconclusive about work-group
				// lifetime. Retain unknown authority rather than clearing/replaying.
				if a.ControllerPID > 0 && a.ControllerStart != "" && c.ownerIdentity != nil && c.ownerIdentity(a.ControllerPID) == a.ControllerStart {
					continue
				}
			}
			if state.Active.Stage != "uncertain" || state.Last == nil || state.Last.ID != a.ID || state.Last.Outcome != "unknown" {
				state, _ = couchschedule.Advance(state, couchschedule.Event{Kind: couchschedule.OwnerUnknown, At: now, Attempt: a.ID, Revision: a.Revision})
			}
			ledger.Tasks[key] = state
		}
		seen := map[string]bool{}
		for _, declaration := range declarations {
			key := declaration.Key.ID()
			seen[key] = true
			prior := ledger.Tasks[key]
			if err := localRefusals[key]; err != nil {
				if err := quarantineTaskHistory(ctx, ledger, prior, "manual recovery", err); err != nil {
					return err
				}
				continue
			}
			if err := retainedHistoryQuarantine(ctx, c.runs, prior); err != nil {
				if err := quarantineTaskHistory(ctx, ledger, prior, "catalogue recovery", err); err != nil {
					return err
				}
				continue
			}
			state, effects := couchschedule.Advance(prior, couchschedule.Event{Kind: couchschedule.CatalogChanged, At: now, Declaration: &declaration, Location: c.location})
			if err := c.finalizeCancelledScheduleNotice(ctx, prior, state); err != nil {
				if err := quarantineTaskHistory(ctx, ledger, prior, "catalogue cancellation", err); err != nil {
					return err
				}
				continue
			}

			ledger.Tasks[key] = state
			for _, effect := range effects {
				if effect.Kind == couchschedule.StopTask && state.Active != nil {
					stopped = append(stopped, *state.Active)
				}
			}
		}
		for key, state := range ledger.Tasks {
			if seen[key] {
				continue
			}
			prior := state
			if err := localRefusals[key]; err != nil {
				if err := quarantineTaskHistory(ctx, ledger, prior, "manual recovery", err); err != nil {
					return err
				}
				continue
			}
			if err := retainedHistoryQuarantine(ctx, c.runs, prior); err != nil {
				if err := quarantineTaskHistory(ctx, ledger, prior, "catalogue recovery", err); err != nil {
					return err
				}
				continue
			}
			state, effects := couchschedule.Advance(state, couchschedule.Event{Kind: couchschedule.CatalogChanged, At: now, Location: c.location})
			if err := c.finalizeCancelledScheduleNotice(ctx, prior, state); err != nil {
				if err := quarantineTaskHistory(ctx, ledger, prior, "catalogue cancellation", err); err != nil {
					return err
				}
				continue
			}
			if state.Active == nil && !state.RemovedAt.IsZero() && !now.Before(state.RemovedAt.Add(30*24*time.Hour)) && (state.Last == nil || (state.Last.Outcome != "unknown" && state.Last.Outcome != "uncertain")) {
				delete(ledger.Tasks, key)
				continue
			}

			ledger.Tasks[key] = state
			for _, effect := range effects {
				if effect.Kind == couchschedule.StopTask && state.Active != nil {
					stopped = append(stopped, *state.Active)
				}
			}
		}
		return nil
	})
	if err != nil {
		return c.blocked(err)
	}
	if err := ensureUncertainRuns(ctx, c.store, c.runs); err != nil {
		return c.blocked(err)
	}
	for _, owner := range stopped {
		if err := c.stopOwner(owner); err != nil {
			return c.blocked(err)
		}
	}
	return c.publish(ctx)
}

// publish refreshes only the memory projection, without rediscovering tasks.

func (c *scheduleCoordinator) run(ctx context.Context, ticks <-chan time.Time) {
	_ = c.refresh(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-ticks:
			if !ok {
				return
			}
			_ = c.refresh(ctx)
		}
	}
}

// A catalogue cancellation terminalizes notice history before releasing the
// retained attempt. Known execution facts and output remain untouched.

func (c *scheduleCoordinator) finalizeCancelledScheduleNotice(ctx context.Context, prior, next couchschedule.TaskState) error {
	if prior.Active == nil || prior.Active.Source != "scheduled" || prior.Last == nil || prior.Last.Notice != "pending" || next.Active != nil || next.Last == nil {
		return nil
	}
	return c.runs.FinalizeNotice(ctx, scheduleRunKey(prior), prior.Active.Revision, next.Last.Notice)
}
