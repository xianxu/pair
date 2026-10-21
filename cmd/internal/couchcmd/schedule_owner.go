package couchcmd

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchschedule"
	"github.com/xianxu/pair/cmd/internal/operationalrun"
)

// scheduleOwner is the only scheduler state/history writer in its namespace.
// Its event loop owns the capability; execution and transport return facts.
type scheduleOwner struct {
	coordinator *scheduleCoordinator
	state       *couchschedule.StateOwner
	busy        map[string]string
}

func newScheduleOwner(c *scheduleCoordinator) *scheduleOwner {
	return &scheduleOwner{coordinator: c, busy: map[string]string{}}
}
func (o *scheduleOwner) open(ctx context.Context) error {
	state, err := o.coordinator.store.Own(ctx)
	if err != nil {
		return err
	}
	o.state = state
	return nil
}
func (o *scheduleOwner) close() error { return o.state.Close() }

func (o *scheduleOwner) apply(ctx context.Context, key couchschedule.TaskKey, event couchschedule.Event, facts *scheduleExecutionFacts) (couchschedule.Settlement, error) {
	var decision couchschedule.Settlement
	var localErr error
	if _, err := o.state.ViewDurable(ctx); err != nil {
		return decision, err
	}
	err := o.state.Update(ctx, func(ledger *couchschedule.Ledger) error {
		prior := ledger.Tasks[key.ID()]
		canonical, err := o.canonical(ctx, prior)
		if err != nil {
			localErr = err
			return quarantineTaskHistory(ctx, ledger, prior, "event settlement", err)
		}
		decision = couchschedule.PlanSettlement(prior, canonical, event)
		if decision.Kind == couchschedule.SettlementQuarantined {
			localErr = errors.New(decision.Reason)
			return quarantineTaskHistory(ctx, ledger, prior, "event settlement", localErr)
		}
		candidate := decision.State
		if decision.Next == couchschedule.SettlementPublishHistory && event.Kind != couchschedule.ClockObserved {
			record, output := ownerResultRecord(candidate, facts)
			record, _, err = o.coordinator.runs.PublishOccurrence(ctx, record.Key, record, output)
			if err != nil {
				localErr = err
				return quarantineTaskHistory(ctx, ledger, prior, "result publication", err)
			}
			summary := runSummaryForRecord(record)
			decision = couchschedule.PlanSettlement(prior, &summary, couchschedule.Event{At: event.At})
			if decision.Kind == couchschedule.SettlementQuarantined {
				localErr = errors.New(decision.Reason)
				return quarantineTaskHistory(ctx, ledger, prior, "result adoption", localErr)
			}
			candidate = decision.State
		} else if decision.Next == couchschedule.SettlementFinalizeHistory {
			last := candidate.Last
			if last == nil {
				return errors.New("notice finalization requires retained facts")
			}
			runKey := operationalrun.RunKey{Hub: key.HubID(), Task: key.Task, ID: last.ID}
			if err = o.coordinator.runs.FinalizeNotice(ctx, runKey, last.Revision, last.Notice); err != nil {
				localErr = err
				return quarantineTaskHistory(ctx, ledger, prior, "notice finalization", err)
			}
			record, _, err := o.coordinator.runs.ReadDurable(ctx, runKey)
			if err != nil {
				localErr = err
				return quarantineTaskHistory(ctx, ledger, prior, "notice adoption", err)
			}
			summary := runSummaryForRecord(record)
			decision = couchschedule.PlanSettlement(prior, &summary, couchschedule.Event{At: event.At})
			if decision.Kind == couchschedule.SettlementQuarantined {
				localErr = errors.New(decision.Reason)
				return quarantineTaskHistory(ctx, ledger, prior, "notice adoption", localErr)
			}
			candidate = decision.State
		}
		ledger.Tasks[key.ID()] = candidate
		return nil
	})
	if err != nil {
		return decision, err
	}
	if localErr != nil {
		return decision, &taskHistoryError{cause: localErr}
	}
	return decision, nil
}

func ownerResultRecord(state couchschedule.TaskState, facts *scheduleExecutionFacts) (operationalrun.Record, []byte) {
	last := state.Last
	notice := last.Notice
	if notice == "" {
		notice = executionNotice(last.Source, last.Outcome)
	}
	record := operationalrun.Record{Version: 1, Key: operationalrun.RunKey{Hub: state.Key.HubID(), Task: state.Key.Task, ID: last.ID}, DeclarationRevision: last.Revision, Source: last.Source, ScheduledAt: last.DueAt, StartedAt: last.StartedAt, EndedAt: last.EndedAt, Exit: last.Exit, Outcome: last.Outcome, Notice: notice, ManualIntentDigest: last.ManualIntentDigest}
	var output []byte
	if facts != nil {
		output = facts.result.Output
		record.Unknown = map[string]json.RawMessage{}
		record.Unknown["build_key"], _ = json.Marshal(facts.result.BuildKey)
		count := facts.result.OutputBytes
		if count < int64(len(output)) {
			count = int64(len(output))
		}
		record.Unknown["output_bytes"], _ = json.Marshal(count)
		if facts.err != nil {
			record.Unknown["error"], _ = json.Marshal(facts.err.Error())
		} else if facts.result.Error != "" {
			record.Unknown["error"], _ = json.Marshal(facts.result.Error)
		}
	}
	return record, output
}

func (o *scheduleOwner) refresh(ctx context.Context) error {
	c := o.coordinator
	declarations, err := c.catalog.Discover(ctx, c.roots)
	if err != nil {
		return c.blocked(err)
	}
	if o.state == nil {
		return errors.New("scheduler owner unavailable")
	}
	if _, err := o.state.ViewDurable(ctx); err != nil {
		return c.blocked(err)
	}
	nextBusy := map[string]string{}
	err = o.state.Update(ctx, func(ledger *couchschedule.Ledger) error {
		catalogue := make(map[string]*couchschedule.Declaration, len(declarations))
		for i := range declarations {
			d := &declarations[i]
			catalogue[d.Key.ID()] = d
		}
		keys := make(map[string]struct{}, len(ledger.Tasks)+len(catalogue))
		for key := range ledger.Tasks {
			keys[key] = struct{}{}
		}
		for key := range catalogue {
			keys[key] = struct{}{}
		}
		for key := range keys {
			prior := ledger.Tasks[key]
			if prior.Key.Hub != "" && !containsScheduleRoot(c.roots, prior.Key.Hub) {
				continue
			}
			canonical, err := o.canonical(ctx, prior)
			if err != nil {
				if err := quarantineTaskHistory(ctx, ledger, prior, "owner settlement", err); err != nil {
					return err
				}
				continue
			}
			decision := couchschedule.PlanSettlement(prior, canonical, couchschedule.Event{Kind: couchschedule.CatalogChanged, At: c.now(), Declaration: catalogue[key], Location: c.location})
			if decision.Kind == couchschedule.SettlementQuarantined {
				if err := quarantineTaskHistory(ctx, ledger, prior, "owner settlement", errors.New(decision.Reason)); err != nil {
					return err
				}
				continue
			}
			if decision.Busy {
				nextBusy[key] = decision.Reason
			}
			state := decision.State
			if state.Key.Hub == "" {
				continue
			}
			if state.Active == nil && !state.RemovedAt.IsZero() && !c.now().Before(state.RemovedAt.Add(30*24*time.Hour)) && (state.Last == nil || (state.Last.Outcome != "unknown" && state.Last.Outcome != "uncertain")) {
				delete(ledger.Tasks, key)
				continue
			}
			ledger.Tasks[key] = state
		}
		return nil
	})
	if err != nil {
		return c.blocked(err)
	}
	o.busy = nextBusy
	return o.publish(ctx)
}

// canonical confirms only retained active evidence. Settled passive ticks do
// not reread or republish old output, including records expired by retention.
func (o *scheduleOwner) canonical(ctx context.Context, state couchschedule.TaskState) (*couchschedule.RunSummary, error) {
	if state.Active == nil {
		return nil, nil
	}
	a := state.Active
	key := operationalrun.RunKey{Hub: state.Key.HubID(), Task: state.Key.Task, ID: a.ID}
	record, _, err := o.coordinator.runs.ReadDurable(ctx, key)
	if missingRunBundle(o.coordinator.runs.Namespace, key, err) && (a.Stage == "reserved" || a.Stage == "running" || a.Stage == "stopping" || a.Stage == "uncertain") {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	summary := runSummaryForRecord(record)
	return &summary, nil
}

func (o *scheduleOwner) publish(ctx context.Context) error {
	ledger, err := o.state.ViewDurable(ctx)
	if err != nil {
		return o.coordinator.blocked(err)
	}
	states := make([]couchschedule.TaskState, 0, len(ledger.Tasks))
	for _, s := range ledger.Tasks {
		if containsScheduleRoot(o.coordinator.roots, s.Key.Hub) {
			states = append(states, s)
		}
	}
	snapshot := couchschedule.BuildStatus(states, o.coordinator.location.String())
	snapshot.Hubs = append([]string(nil), o.coordinator.roots...)
	for i := range snapshot.Tasks {
		row := &snapshot.Tasks[i]
		if reason := o.busy[(couchschedule.TaskKey{Hub: row.Hub, Task: row.Task}).ID()]; reason != "" {
			row.Reason = reason
		}
	}
	o.coordinator.snapshot.Store(&snapshot)
	return nil
}
