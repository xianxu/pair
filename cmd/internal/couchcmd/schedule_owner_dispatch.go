package couchcmd

import (
	"context"
	"errors"
	"sort"

	"github.com/xianxu/pair/cmd/internal/couchschedule"
)

// complete consumes worker facts on the owner turn. A retained canonical record
// wins on every retry, including a ledger failure after immutable publication.
func (o *scheduleOwner) complete(ctx context.Context, reserved couchschedule.TaskState, facts scheduleExecutionFacts) error {
	if reserved.Active == nil {
		return errors.New("completion lacks occurrence identity")
	}
	a := reserved.Active
	ledger, err := o.state.ViewDurable(ctx)
	if err != nil {
		return err
	}
	state := ledger.Tasks[reserved.Key.ID()]
	if state.Active == nil || state.Active.ID != a.ID || state.Active.Revision != a.Revision {
		return nil
	}
	ended := facts.ended
	if ended.IsZero() {
		ended = o.coordinator.now()
	}
	event := couchschedule.Event{Kind: couchschedule.TaskFinished, At: ended, Attempt: a.ID, Revision: a.Revision, Exit: facts.result.Exit, Outcome: facts.result.Outcome, HubOnline: o.hubOnline(reserved.Key.Hub)}
	if event.Outcome == "uncertain" || event.Outcome == "unknown" {
		event.Kind = couchschedule.OwnerUnknown
	}
	if facts.err != nil && facts.result.Outcome == "" {
		var executionErr *couchschedule.ExecutionError
		if errors.As(facts.err, &executionErr) && !executionErr.MayHaveStarted && state.Active.Stage == "reserved" {
			event.Kind = couchschedule.ExecutionRefused
			event.Outcome = "refused"
			if !ended.Before(a.Deadline) {
				event.Outcome = "timeout"
			}
			event.Notice = facts.err.Error()
		} else {
			event.Kind = couchschedule.OwnerUnknown
		}
	}
	if state.Active.Stage == "reserved" && event.Kind == couchschedule.TaskFinished {
		// A controller never received durable release. Its result cannot stand in
		// for a matching start acknowledgment, even if a runtime reports success.
		event.Kind = couchschedule.OwnerUnknown
	}
	_, err = o.apply(ctx, reserved.Key, event, &facts)
	if err == nil {
		delete(o.intents, reserved.Key.ID())
	}
	return err
}

func (o *scheduleOwner) hubOnline(root string) bool {
	if o.loop == nil || o.loop.port == nil {
		return false
	}
	_, ok := o.loop.port.Binding(root)
	return ok
}

// tick reserves before effects and visits each task once. Long scripts and gate
// waits are admitted to the fixed workers and never run on this owner turn.
func (o *scheduleOwner) tick(ctx context.Context, clock scheduleClock) error {
	ledger, err := o.state.ViewDurable(ctx)
	if err != nil {
		return err
	}
	keys := make([]string, 0, len(ledger.Tasks))
	for key, s := range ledger.Tasks {
		if containsScheduleRoot(o.coordinator.roots, s.Key.Hub) {
			keys = append(keys, key)
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := ledger.Tasks[keys[i]], ledger.Tasks[keys[j]]
		if a.Key.Hub != b.Key.Hub {
			return a.Key.Hub < b.Key.Hub
		}
		return a.Key.Task < b.Key.Task
	})
	// One binding observation per hub for due consumption and candidate selection.
	online := map[string]bool{}
	for _, root := range o.coordinator.roots {
		online[root] = o.hubOnline(root)
	}
	if !o.recovered {
		if err = o.recoverBatch(ctx, clock); err != nil && !ownerTaskLocal(err) {
			return err
		}
	}
	if err = o.reserveDueBatch(ctx, clock, online); err != nil && !ownerTaskLocal(err) {
		return err
	}
	o.recovered = true
	ledger, err = o.state.ViewDurable(ctx)
	if err != nil {
		return err
	}
	for _, key := range keys {
		state := ledger.Tasks[key]
		if state.Active == nil {
			continue
		}
		if !state.Allowed || o.busy[key] != "" {
			// Busy catalogue changes cannot create another occurrence; already
			// retained notice/result settlement still progresses below.
			if !state.Allowed {
				continue
			}
		}
		if o.loop == nil {
			continue
		}
		if _, pending := o.loop.pending[key]; pending {
			continue
		}
		if _, active := o.loop.active[key]; active {
			continue
		}
		a := state.Active
		switch a.Stage {
		case "reserved":
			if o.busy[key] != "" {
				continue
			}
			if a.Source == "manual" {
				intent, ok := o.intents[key]
				if !ok || intent.ID != a.ID || intent.Digest() != a.ManualIntentDigest {
					_, err = o.apply(ctx, state.Key, couchschedule.Event{Kind: couchschedule.OwnerUnknown, At: clock.Now(), Attempt: a.ID, Revision: a.Revision}, nil)
				} else {
					o.launchExecution(ctx, state, intent.Args)
				}
			} else if state.Spec.ScriptRun {
				o.launchExecution(ctx, state, []string{"script-run"})
			} else {
				if !o.noticeCapacity() {
					continue
				}
				decision, applyErr := o.apply(ctx, state.Key, couchschedule.Event{Kind: couchschedule.ReservationDurable, At: clock.Now(), Attempt: a.ID, Revision: a.Revision, HubOnline: online[state.Key.Hub]}, nil)
				err = applyErr
				if applyErr == nil {
					state = decision.State
					a = state.Active
				}
			}
		case "running", "stopping", "uncertain":
			// Completed worker facts are handled by retry; retained recovery is a
			// batched catalogue/startup operation, not a full ledger write per task.
		case "delivery":
			notice, known, active := "uncertain", false, false
			if o.loop.port != nil {
				notice, known, active = o.loop.port.Observe(a.ReceiptID)
			}
			if !active {
				if !known {
					notice = "uncertain"
				}
				if notice != "" {
					_, err = o.apply(ctx, state.Key, couchschedule.Event{Kind: couchschedule.NoticeObserved, At: clock.Now(), Attempt: a.ID, Revision: a.Revision, Notice: notice, HubOnline: online[state.Key.Hub]}, nil)
				}
			}
		}
		if err != nil {
			if !ownerTaskLocal(err) {
				return err
			}
			continue
		}
		if a == nil || a.Stage != "notifying" {
			continue
		}
		if !clock.Now().Before(a.NoticeDeadline) || !online[state.Key.Hub] {
			_, err = o.apply(ctx, state.Key, couchschedule.Event{Kind: couchschedule.NoticeIntentDurable, At: clock.Now(), Attempt: a.ID, Revision: a.Revision, HubOnline: online[state.Key.Hub]}, nil)
		} else {
			if !o.noticeCapacity() {
				continue
			}
			if a.ReceiptID == "" {
				_, output, readErr := o.coordinator.runs.ReadDurable(ctx, scheduleRunKey(state))
				if readErr != nil {
					if commitErr := o.state.Update(ctx, func(l *couchschedule.Ledger) error {
						return quarantineTaskHistory(ctx, l, state, "notice preparation", readErr)
					}); commitErr != nil {
						return commitErr
					}
					continue
				}
				binding, _ := o.loop.port.Binding(state.Key.Hub)
				decision, applyErr := o.apply(ctx, state.Key, couchschedule.Event{Kind: couchschedule.NoticeFrozen, At: clock.Now(), Attempt: a.ID, Revision: a.Revision, ReceiptID: "schedule-" + a.ID, NoticeBody: scheduleNoticeBody(state, output), NoticeBinding: binding}, nil)
				if applyErr != nil {
					if !ownerTaskLocal(applyErr) {
						return applyErr
					}
					continue
				}
				state = decision.State
				a = state.Active
				if a == nil {
					continue
				}
			}
			decision, applyErr := o.apply(ctx, state.Key, couchschedule.Event{Kind: couchschedule.NoticeIntentDurable, At: clock.Now(), Attempt: a.ID, Revision: a.Revision, HubOnline: true}, nil)
			err = applyErr
			if applyErr == nil && len(decision.Effects) == 1 && decision.Effects[0].Kind == couchschedule.SendNotice {
				o.launchNotice(ctx, clock, decision.State)
			}
		}
		if err != nil && !ownerTaskLocal(err) {
			return err
		}
	}
	return o.publish(ctx)
}

func ownerTaskLocal(err error) bool { var local *taskHistoryError; return errors.As(err, &local) }

func (o *scheduleOwner) noticeCapacity() bool {
	n := 0
	for _, notice := range o.loop.active {
		if notice {
			n++
		}
	}
	return n < 2
}
