package couchcmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/xianxu/pair/cmd/internal/couchschedule"
	"github.com/xianxu/pair/cmd/internal/operationalrun"
)

// These methods run only on the owner's command turn. Neither transport nor
// the CLI owns a storage capability or releases a script directly.
func (o *scheduleOwner) submitManual(ctx context.Context, m couchschedule.ManualIntent) (couchschedule.ManualStatus, error) {
	if err := m.Validate(); err != nil {
		return couchschedule.ManualStatus{}, err
	}
	if !containsScheduleRoot(o.coordinator.roots, m.Hub) {
		return couchschedule.ManualStatus{}, errors.New("manual hub is not configured")
	}
	ledger, err := o.coordinator.store.View(ctx)
	if err != nil {
		return couchschedule.ManualStatus{}, err
	}
	key := couchschedule.TaskKey{Hub: m.Hub, Task: m.Task}
	state, exists := ledger.Tasks[key.ID()]
	// A duplicate retained ID never goes through fresh admission. Confirm native
	// evidence durably before treating it as an accepted occurrence.
	runKey := operationalrun.RunKey{Hub: key.HubID(), Task: m.Task, ID: m.ID}
	record, _, historyErr := o.coordinator.runs.ReadDurable(ctx, runKey)
	retainedDigest := ""
	retained := false
	if historyErr == nil {
		if record.Source != "manual" {
			return couchschedule.ManualStatus{}, errors.New("run ID belongs to a scheduled occurrence")
		}
		retained = true
		retainedDigest = record.ManualIntentDigest
	} else if !missingRunBundle(o.coordinator.runs.Namespace, runKey, historyErr) {
		return couchschedule.ManualStatus{}, historyErr
	}
	if state.Active != nil && state.Active.ID == m.ID {
		if state.Active.Source != "manual" {
			return couchschedule.ManualStatus{}, errors.New("run ID belongs to a scheduled occurrence")
		}
		if retained && retainedDigest != state.Active.ManualIntentDigest {
			return couchschedule.ManualStatus{}, errors.New("manual history and authority intent differ")
		}
		retained = true
		retainedDigest = state.Active.ManualIntentDigest
	}
	if state.Last != nil && state.Last.ID == m.ID {
		if state.Last.Source != "manual" {
			return couchschedule.ManualStatus{}, errors.New("run ID belongs to a scheduled occurrence")
		}
		if retained && retainedDigest != state.Last.ManualIntentDigest {
			return couchschedule.ManualStatus{}, errors.New("manual history and authority intent differ")
		}
		retained = true
		retainedDigest = state.Last.ManualIntentDigest
	}
	if retained {
		if retainedDigest == "" || retainedDigest != m.Digest() {
			return couchschedule.ManualStatus{}, errors.New("run ID has conflicting or legacy unknown manual intent")
		}
		if state.Active != nil && state.Active.ID == m.ID && historyErr == nil {
			if _, err = o.apply(ctx, key, couchschedule.Event{At: o.coordinator.now()}, nil); err != nil {
				return couchschedule.ManualStatus{}, err
			}
		}
		return o.observeManual(ctx, m.Hub, m.Task, m.ID, 0)
	}
	if !exists || state.Revision == "" || !state.RemovedAt.IsZero() {
		return couchschedule.ManualStatus{}, errors.New("task declaration is unavailable")
	}
	if o.busy[key.ID()] != "" || state.Active != nil {
		return couchschedule.ManualStatus{}, errors.New("task busy")
	}
	if !state.Allowed {
		return couchschedule.ManualStatus{}, fmt.Errorf("task refused: %s", state.Refusal)
	}
	if _, err = o.apply(ctx, key, couchschedule.Event{Kind: couchschedule.ManualRequested, At: o.coordinator.now(), RunID: m.ID, ManualIntentDigest: m.Digest(), Location: o.coordinator.location}, nil); err != nil {
		return couchschedule.ManualStatus{}, fmt.Errorf("manual reservation uncertain: %w", err)
	}
	// Bind argv only after the persisted reservation can be observed; a failed
	// commit must not leave executable work in the dispatch map.
	status, err := o.observeManual(ctx, m.Hub, m.Task, m.ID, 0)
	if err != nil {
		return couchschedule.ManualStatus{}, err
	}
	if status.Stage == "reserved" {
		if o.intents == nil {
			o.intents = map[string]couchschedule.ManualIntent{}
		}
		m.Args = append([]string(nil), m.Args...)
		o.intents[key.ID()] = m
	}
	return status, nil
}

func (o *scheduleOwner) observeManual(ctx context.Context, hub, task, id string, offset int) (couchschedule.ManualStatus, error) {
	m := couchschedule.ManualIntent{ID: id, Hub: hub, Task: task}
	if err := m.Validate(); err != nil {
		return couchschedule.ManualStatus{}, err
	}
	if !containsScheduleRoot(o.coordinator.roots, hub) || offset < 0 || offset > couchschedule.MaxManualOutput {
		return couchschedule.ManualStatus{}, errors.New("invalid manual observation")
	}
	ledger, err := o.coordinator.store.View(ctx)
	if err != nil {
		return couchschedule.ManualStatus{}, err
	}
	key := couchschedule.TaskKey{Hub: hub, Task: task}
	state := ledger.Tasks[key.ID()]
	status := couchschedule.ManualStatus{ID: id, Hub: hub, Task: task}
	// Inspection does not synchronize, reconcile, create, or repair history.
	runKey := operationalrun.RunKey{Hub: key.HubID(), Task: task, ID: id}
	record, output, err := o.coordinator.runs.Read(ctx, runKey)
	if err == nil {
		if record.Source != "manual" {
			return status, errors.New("run ID is not manual")
		}
		status.Stage = "done"
		if record.Outcome == "unknown" || record.Outcome == "uncertain" {
			status.Stage = "unknown"
		}
		status.Outcome = record.Outcome
		status.Exit = record.Exit
		if len(output) > couchschedule.MaxManualOutput || offset > len(output) {
			return status, errors.New("manual output offset or stored size is invalid")
		}
		status.Offset = offset
		status.Total = len(output)
		status.Output = append([]byte(nil), output[offset:min(offset+couchschedule.MaxManualOutputPage, len(output))]...)
		return status, status.Validate()
	}
	if !missingRunBundle(o.coordinator.runs.Namespace, runKey, err) {
		return status, err
	}
	if a := state.Active; a != nil && a.ID == id {
		if a.Source != "manual" {
			return status, errors.New("run ID is not manual")
		}
		status.Stage = a.Stage
		if a.Stage == "uncertain" {
			status.Stage = "unknown"
		}
		status.Error = state.Refusal
		if offset != 0 {
			return status, errors.New("active manual run has no output page")
		}
		return status, status.Validate()
	}
	if last := state.Last; last != nil && last.ID == id {
		if last.Source != "manual" {
			return status, errors.New("run ID is not manual")
		}
		if offset != 0 {
			return status, errors.New("retained output is unavailable")
		}
		status.Stage = "done"
		if last.Outcome == "unknown" || last.Outcome == "uncertain" {
			status.Stage = "unknown"
		}
		status.Outcome = last.Outcome
		status.Exit = last.Exit
		status.Error = "retained output is unavailable"
		return status, status.Validate()
	}
	return status, errors.New("manual run ID is not retained")
}

func (o *scheduleOwner) cancelManual(ctx context.Context, hub, task, id string) (couchschedule.ManualStatus, error) {
	status, err := o.observeManual(ctx, hub, task, id, 0)
	if err != nil {
		return status, err
	}
	if status.Terminal() {
		return status, nil
	}
	key := couchschedule.TaskKey{Hub: hub, Task: task}
	ledger, err := o.coordinator.store.View(ctx)
	if err != nil {
		return status, err
	}
	state := ledger.Tasks[key.ID()]
	if state.Active == nil || state.Active.ID != id || state.Active.Source != "manual" {
		return status, errors.New("manual cancellation authority changed")
	}
	event := couchschedule.Event{Kind: couchschedule.CancelRequested, At: o.coordinator.now(), Attempt: id, Revision: state.Active.Revision}
	if _, err = o.apply(ctx, key, event, nil); err != nil {
		return status, err
	}
	o.cancelExecution(key, id)
	delete(o.intents, key.ID())
	return o.observeManual(ctx, hub, task, id, 0)
}
