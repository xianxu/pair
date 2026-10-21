package couchcmd

import (
	"context"
	"errors"

	"github.com/xianxu/pair/cmd/internal/couchschedule"
)

// Timer/discovery work is batched at the owner boundary: a full-ledger JSON
// publication costs O(tasks), so publishing once per task would be quadratic.
// The reservation batch commits before any newly consumed occurrence history.
func (o *scheduleOwner) reserveDueBatch(ctx context.Context, clock scheduleClock, online map[string]bool) error {
	view, err := o.state.ViewDurable(ctx)
	if err != nil {
		return err
	}
	due := false
	for _, s := range view.Tasks {
		if containsScheduleRoot(o.coordinator.roots, s.Key.Hub) && !s.NextDue.IsZero() && !clock.Now().Before(s.NextDue) {
			due = true
			break
		}
	}
	if !due {
		return nil
	}
	var publish []couchschedule.TaskState
	var local []error
	err = o.state.Update(ctx, func(l *couchschedule.Ledger) error {
		for key, s := range l.Tasks {
			if !containsScheduleRoot(o.coordinator.roots, s.Key.Hub) || s.NextDue.IsZero() || clock.Now().Before(s.NextDue) {
				continue
			}
			// A settled occurrence does not make an observed pending catalogue
			// edit disappear. Rediscovery must apply it before another reservation.
			if o.busy[key] != "" && s.Active == nil {
				continue
			}
			if err := ensurePassiveRun(ctx, o.coordinator.runs, s); err != nil {
				local = append(local, err)
				if err := quarantineTaskHistory(ctx, l, s, "prior passive history", err); err != nil {
					return err
				}
				continue
			}
			event := couchschedule.Event{Kind: couchschedule.ClockObserved, At: clock.Now(), RunID: scheduledRunID(s), HubOnline: online[s.Key.Hub], Location: o.coordinator.location}
			decision := couchschedule.PlanSettlement(s, nil, event)
			if decision.Kind == couchschedule.SettlementQuarantined {
				cause := errors.New(decision.Reason)
				local = append(local, cause)
				if err := quarantineTaskHistory(ctx, l, s, "due consumption", cause); err != nil {
					return err
				}
				continue
			}
			l.Tasks[key] = decision.State
			if decision.Next == couchschedule.SettlementPublishHistory {
				publish = append(publish, decision.State)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if err = o.publishRetainedBatch(ctx, publish, clock); err != nil {
		return err
	}
	return ownerBatchErrors(local)
}

func (o *scheduleOwner) recoverBatch(ctx context.Context, clock scheduleClock) error {
	view, err := o.state.ViewDurable(ctx)
	if err != nil {
		return err
	}
	active := false
	for _, s := range view.Tasks {
		if s.Active != nil && containsScheduleRoot(o.coordinator.roots, s.Key.Hub) {
			active = true
			break
		}
	}
	if !active {
		return nil
	}
	var publish []couchschedule.TaskState
	var local []error
	err = o.state.Update(ctx, func(l *couchschedule.Ledger) error {
		for key, s := range l.Tasks {
			if s.Active == nil || !containsScheduleRoot(o.coordinator.roots, s.Key.Hub) {
				continue
			}
			canonical, readErr := o.canonical(ctx, s)
			if readErr != nil {
				local = append(local, readErr)
				if err := quarantineTaskHistory(ctx, l, s, "startup evidence", readErr); err != nil {
					return err
				}
				continue
			}
			a := s.Active
			event := couchschedule.Event{At: clock.Now()}
			if canonical == nil && ((a.Stage == "reserved" && (a.Source == "manual" || s.Spec.ScriptRun)) || a.Stage == "running" || a.Stage == "stopping" || a.Stage == "uncertain") {
				event.Kind = couchschedule.OwnerUnknown
				event.Attempt = a.ID
				event.Revision = a.Revision
			}
			decision := couchschedule.PlanSettlement(s, canonical, event)
			if decision.Kind == couchschedule.SettlementQuarantined {
				cause := errors.New(decision.Reason)
				local = append(local, cause)
				if err := quarantineTaskHistory(ctx, l, s, "startup adoption", cause); err != nil {
					return err
				}
				continue
			}
			l.Tasks[key] = decision.State
			if canonical == nil && decision.State.Active != nil && decision.State.Active.Stage == "uncertain" && decision.State.Last != nil {
				publish = append(publish, decision.State)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if err = o.publishRetainedBatch(ctx, publish, clock); err != nil {
		return err
	}
	return ownerBatchErrors(local)
}

// Every input summary is already committed authority. This path can preserve
// passive/unknown facts; it cannot manufacture a native execution completion.
func (o *scheduleOwner) publishRetainedBatch(ctx context.Context, states []couchschedule.TaskState, clock scheduleClock) error {
	if len(states) == 0 {
		return nil
	}
	type evidence struct {
		state   couchschedule.TaskState
		summary couchschedule.RunSummary
		err     error
	}
	confirmed := make([]evidence, 0, len(states))
	for _, s := range states {
		record, err := ensureRetainedRun(ctx, o.coordinator.runs, s, s.Active != nil && s.Active.Stage == "uncertain")
		confirmed = append(confirmed, evidence{state: s, summary: runSummaryForRecord(record), err: err})
	}
	var local []error
	err := o.state.Update(ctx, func(l *couchschedule.Ledger) error {
		for _, proof := range confirmed {
			key := proof.state.Key.ID()
			current := l.Tasks[key]
			if current.Last == nil || proof.state.Last == nil || current.Last.ID != proof.state.Last.ID {
				continue
			}
			if proof.err != nil {
				local = append(local, proof.err)
				if err := quarantineTaskHistory(ctx, l, current, "retained publication", proof.err); err != nil {
					return err
				}
				continue
			}
			decision := couchschedule.PlanSettlement(current, &proof.summary, couchschedule.Event{At: clock.Now()})
			if decision.Kind == couchschedule.SettlementQuarantined {
				cause := errors.New(decision.Reason)
				local = append(local, cause)
				if err := quarantineTaskHistory(ctx, l, current, "retained adoption", cause); err != nil {
					return err
				}
				continue
			}
			l.Tasks[key] = decision.State
		}
		return nil
	})
	if err != nil {
		return err
	}
	return ownerBatchErrors(local)
}

func ownerBatchErrors(local []error) error {
	if len(local) == 0 {
		return nil
	}
	return &taskHistoryError{cause: errors.Join(local...)}
}
