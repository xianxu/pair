package couchcmd

import (
	"context"
	"errors"
	"path/filepath"
	"sort"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchschedule"
	"github.com/xianxu/pair/cmd/internal/operationalrun"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/strictjson"
)

// scheduleCoordinator owns catalog refresh and immutable status publication.
// Its service lifetime owns all background work; socket handlers only view it.
type scheduleCoordinator struct {
	roots         []string
	store         couchschedule.StateStore
	runs          operationalrun.Store
	catalog       couchschedule.Catalog
	location      *time.Location
	now           func() time.Time
	snapshot      atomic.Pointer[couchschedule.StatusSnapshot]
	stopOwner     func(couchschedule.Attempt) error
	ownerIdentity func(int) string
}

func newScheduleCoordinator(namespace string, roots []string, location *time.Location, now func() time.Time) *scheduleCoordinator {
	c := &scheduleCoordinator{roots: append([]string(nil), roots...), runs: operationalrun.Store{Namespace: namespace}, store: couchschedule.StateStore{Dir: filepath.Join(namespace, "schedule")}, location: location, now: now}
	sort.Strings(c.roots)
	snapshot := couchschedule.BuildStatus(nil, location.String())
	snapshot.Hubs = append([]string(nil), c.roots...)
	snapshot.Error = "waiting for scheduler catalogue"
	c.snapshot.Store(&snapshot)
	c.stopOwner = func(a couchschedule.Attempt) error {
		return couchschedule.StopOwner(couchschedule.OwnerIdentity{PID: a.ControllerPID, Start: a.ControllerStart})
	}
	c.ownerIdentity = func(pid int) string {
		if pid <= 0 {
			return ""
		}
		return procutil.StrictIdentity(strconv.Itoa(pid))
	}
	return c
}

func (c *scheduleCoordinator) view() couchschedule.StatusSnapshot {
	old := c.snapshot.Load()
	if old == nil {
		return couchschedule.StatusSnapshot{Version: 1, Tasks: []couchschedule.TaskStatus{}, Hubs: append([]string(nil), c.roots...), Error: "scheduler unavailable"}
	}
	copy := *old
	copy.Hubs = append([]string(nil), old.Hubs...)
	copy.Tasks = append([]couchschedule.TaskStatus{}, old.Tasks...)
	for i := range copy.Tasks {
		if copy.Tasks[i].Active != nil {
			active := *copy.Tasks[i].Active
			active.Unknown = strictjson.CloneRaw(active.Unknown)
			active.NoticeBinding = append(active.NoticeBinding[:0:0], active.NoticeBinding...)
			copy.Tasks[i].Active = &active
		}
		if copy.Tasks[i].Last != nil {
			last := *copy.Tasks[i].Last
			last.Unknown = strictjson.CloneRaw(last.Unknown)
			if last.Exit != nil {
				code := *last.Exit
				last.Exit = &code
			}
			copy.Tasks[i].Last = &last
		}
	}
	return copy
}

func (c *scheduleCoordinator) blocked(err error) error {
	snapshot := c.view()
	snapshot.Error = err.Error()
	if len(snapshot.Error) > 4096 {
		snapshot.Error = snapshot.Error[:4096]
	}
	c.snapshot.Store(&snapshot)
	return err
}

func (c *scheduleCoordinator) refresh(ctx context.Context) error {
	declarations, err := c.catalog.Discover(ctx, c.roots)
	if err != nil {
		return c.blocked(err)
	}
	now := c.now()
	var stopped []couchschedule.Attempt
	localRefusals := map[string]error{}
	err = c.store.Update(ctx, func(ledger *couchschedule.Ledger) error {
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
func (c *scheduleCoordinator) publish(ctx context.Context) error {
	ledger, err := c.store.View(ctx)
	if err != nil {
		return c.blocked(err)
	}
	states := make([]couchschedule.TaskState, 0, len(ledger.Tasks))
	configured := map[string]bool{}
	for _, root := range c.roots {
		configured[root] = true
	}
	for _, state := range ledger.Tasks {
		if configured[state.Key.Hub] {
			states = append(states, state)
		}
	}
	snapshot := couchschedule.BuildStatus(states, c.location.String())
	snapshot.Hubs = append([]string(nil), c.roots...)
	c.snapshot.Store(&snapshot)
	return nil
}

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
