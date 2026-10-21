package couchcmd

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchschedule"
	"github.com/xianxu/pair/cmd/internal/operationalrun"
)

// scheduleOwner is the only scheduler state/history writer in its namespace.
// Its event loop owns the capability; execution and transport return facts.
type scheduleOwner struct {
	automationEnabled func(hubRoot string) bool
	coordinator       *scheduleCoordinator
	state             *couchschedule.StateOwner
	busy              map[string]string
	intents           map[string]couchschedule.ManualIntent
	operations        map[string]*scheduleOwnerOperation
	loop              *scheduleOwnerLoop
	recovered         bool
}

func newScheduleOwner(c *scheduleCoordinator) *scheduleOwner {
	return &scheduleOwner{coordinator: c, busy: map[string]string{}, intents: map[string]couchschedule.ManualIntent{}, operations: map[string]*scheduleOwnerOperation{}}
}

type scheduleOwnerOperation struct {
	id     string
	cancel context.CancelFunc
}

func (o *scheduleOwner) cancelExecution(key couchschedule.TaskKey, id string) {
	if op := o.operations[key.ID()]; op != nil && op.id == id && op.cancel != nil {
		op.cancel()
	}
}

func (o *scheduleOwner) authorize(ctx context.Context, state couchschedule.TaskState, identity couchschedule.OwnerIdentity, declaration couchschedule.Declaration) error {
	if state.Active == nil || !declaration.Allowed || declaration.Key != state.Key || declaration.Revision != state.Active.Revision || declaration.Spec != state.Spec {
		return errors.New("execution declaration changed")
	}
	ledger, err := o.state.ViewDurable(ctx)
	if err != nil {
		return err
	}
	current := ledger.Tasks[state.Key.ID()]
	if !sameReservedAttempt(current, state) || !current.Allowed || !o.coordinator.now().Before(state.Active.Deadline) {
		return errors.New("execution authority changed or expired")
	}
	if current.Active.Source == "scheduled" && !o.automationOn(current.Key.Hub) {
		return errors.New("automation off: scheduled execution refused")
	}
	event := couchschedule.Event{Kind: couchschedule.OwnerRecorded, At: o.coordinator.now(), Attempt: state.Active.ID, Revision: state.Active.Revision, Owner: &identity}
	if _, err := o.apply(ctx, state.Key, event, nil); err != nil {
		return err
	}
	event.Kind = couchschedule.ReservationDurable
	decision, err := o.apply(ctx, state.Key, event, nil)
	if err != nil {
		return err
	}
	if len(decision.Effects) != 1 || decision.Effects[0].Kind != couchschedule.StartTask {
		return errors.New("execution start authorization refused")
	}
	return nil
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
		var err error
		decision, localErr, err = o.applyInLedger(ctx, ledger, key, event, facts)
		return err
	})
	if err != nil {
		return decision, err
	}
	if localErr != nil {
		return decision, &taskHistoryError{cause: localErr}
	}
	// Passive consumption and unknown classification are durable before their
	// diagnostic/no-execution bundles can be published. Never reconstruct native
	// execution completion here.
	if (event.Kind == couchschedule.ClockObserved && decision.Next == couchschedule.SettlementPublishHistory) || (decision.State.Active != nil && decision.State.Active.Stage == "uncertain" && decision.State.Last != nil && decision.State.Last.Outcome == "unknown") {
		state := decision.State
		record, err := o.ensureRetainedRun(ctx, state, state.Active != nil && state.Active.Stage == "uncertain")
		if err != nil {
			if commitErr := o.state.Update(ctx, func(l *couchschedule.Ledger) error {
				return quarantineTaskHistory(ctx, l, l.Tasks[key.ID()], "passive publication", err)
			}); commitErr != nil {
				return decision, commitErr
			}
			return decision, &taskHistoryError{cause: err}
		}
		summary := runSummaryForRecord(record)
		err = o.state.Update(ctx, func(l *couchschedule.Ledger) error {
			decision = couchschedule.PlanSettlement(l.Tasks[key.ID()], &summary, couchschedule.Event{At: event.At})
			if decision.Kind == couchschedule.SettlementQuarantined {
				return errors.New(decision.Reason)
			}
			l.Tasks[key.ID()] = decision.State
			return nil
		})
		if err != nil {
			return decision, err
		}
	}
	return decision, nil
}

// applyInLedger shares canonical-first history settlement between individual
// commands and timer batches. Its candidate becomes authority only when the
// enclosing owner transaction commits. Task-local failures retain prior facts.
func (o *scheduleOwner) applyInLedger(ctx context.Context, ledger *couchschedule.Ledger, key couchschedule.TaskKey, event couchschedule.Event, facts *scheduleExecutionFacts) (couchschedule.Settlement, error, error) {
	var decision couchschedule.Settlement
	var localErr error
	err := func() error {
		prior := ledger.Tasks[key.ID()]
		if event.Kind == couchschedule.ManualRequested || event.Kind == couchschedule.ClockObserved {
			if err := o.ensurePassiveRun(ctx, prior); err != nil {
				localErr = err
				return quarantineTaskHistory(ctx, ledger, prior, "prior passive history", err)
			}
		}
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
	}()
	return decision, localErr, err
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
			// Unknown diagnostics contain only previously committed facts.
			// A retained history quarantine also needs durable proof before
			// discovery may restore policy, including Last-only passive rows.
			var canonical *couchschedule.RunSummary
			var historyErr error
			retained := prior.Last != nil && (prior.Active == nil || prior.Last.ID == prior.Active.ID)
			if retainedUnknownSummary(prior) || (retained && !prior.Allowed && strings.Contains(prior.Refusal, " history ")) {
				record, err := o.ensureRetainedRun(ctx, prior, retainedUnknownSummary(prior))
				historyErr = err
				if err == nil {
					summary := runSummaryForRecord(record)
					canonical = &summary
				}
			} else {
				canonical, historyErr = o.canonical(ctx, prior)
			}
			err := historyErr
			if err != nil {
				if err := quarantineTaskHistory(ctx, ledger, prior, "owner settlement", err); err != nil {
					return err
				}
				continue
			}
			// History quarantine is diagnostic, not a catalogue edit. Release
			// it only after durable evidence and an unchanged declaration agree;
			// genuine policy/revision edits still wait for occurrence settlement.
			if canonical != nil && !prior.Allowed && strings.Contains(prior.Refusal, " history ") {
				if d := catalogue[key]; d != nil && d.Key == prior.Key && d.Revision == prior.Revision && d.Spec == prior.Spec && prior.RemovedAt.IsZero() {
					prior.Allowed, prior.Refusal = d.Allowed, d.Reason
				}
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
