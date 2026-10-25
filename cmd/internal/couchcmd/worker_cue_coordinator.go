package couchcmd

import (
	"context"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/couchcore"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

type workerCueAttempt struct {
	binding couchmessage.Binding
	ordinal uint64
}

// cueWorker uses the wrapper witness as the lifetime authority. This bounded
// supervisor record merely prevents another attempt while a report or an ack
// is delayed. An unchanged incarnation costs at most one ready-record proof.
func (c *supervisionCoordinator) cueWorker(ctx context.Context, live couchmessage.SlotLiveness) {
	report := couchmessage.CopyWorkerCueReport(live.WorkerCue)
	to := live.Binding
	c.service.mu.Lock()
	workspace, known := c.service.workspaces[to]
	c.service.mu.Unlock()
	if !known || c.service.authority.isHub(workspace) {
		return
	}
	if ctx.Err() != nil || report == nil || report.Validate() != nil || !c.service.isConnected(to) {
		return
	}
	if report.State != couchmessage.WorkerCueNeeded {
		// Positive wrapper-owned evidence outlives receipt expiry and permits this
		// supervisor to forget its redundant attempt record. Keep uncertain attempts
		// whose wrapper still reports needed across disconnect/reconnect.
		if report.ID == couchmessage.WorkerStartNoticeID(to, report.LaunchOrdinal) {
			c.mu.Lock()
			if old, ok := c.workerCues[to.Slot]; ok && old.binding == to && old.ordinal <= report.LaunchOrdinal {
				delete(c.workerCues, to.Slot)
			}
			c.mu.Unlock()
		}
		return
	}
	hub, available := c.service.hubBindingFor(to)
	if !available {
		return
	}
	attempt := workerCueAttempt{binding: to, ordinal: report.LaunchOrdinal}
	c.mu.Lock()
	old, exists := c.workerCues[to.Slot]
	if exists && old.binding == to && old.ordinal >= attempt.ordinal {
		c.mu.Unlock()
		return
	}
	if !exists && len(c.workerCues) >= couchmessage.MaxActors {
		first := !c.cueCapacityNotified
		c.cueCapacityNotified = true
		c.mu.Unlock()
		if first && c.notice != nil {
			c.warn(couchcore.NoticeWorkerCueUnavailable, to, "worker start cue tracking is full; untracked workers receive no automatic orientation; hub should supply it")
		}
		return
	}
	if c.workerCues == nil {
		c.workerCues = map[string]workerCueAttempt{}
	}
	c.workerCues[to.Slot] = attempt
	c.mu.Unlock()
	skip := func(reason string) {
		if c.notice != nil {
			c.warn(couchcore.NoticeWorkerCueUnavailable, to, fmt.Sprintf("worker start cue to %s not delivered: %s; hub may supply orientation", to.Slot, reason))
		}
	}
	if c.service.authority.child == nil {
		skip("current native launch is unknown")
		return
	}
	proofCtx, cancel := context.WithTimeout(ctx, couchmessage.AdmissionTimeout)
	ordinal, identity, err := c.service.authority.child(proofCtx, to)
	proofErr := proofCtx.Err()
	cancel()
	if err != nil || proofErr != nil || ordinal != report.LaunchOrdinal || identity == "" {
		skip("current native launch was not proved")
		return
	}
	// Child proof can yield. Recheck the exact current receiver, report, role and
	// owner before handing its one self-addressed cue to the ordinary broker.
	current := c.service.liveness.Load()
	if current == nil || ctx.Err() != nil {
		return
	}
	latest, ok := (*current)[to.Slot]
	owner, available := c.service.hubBindingFor(to)
	if !ok || latest.Binding != to || latest.WorkerCue == nil || *latest.WorkerCue != *report || !c.service.isConnected(to) || c.service.isHubBinding(to) || !available || owner != hub {
		skip("receiver or owning hub changed before admission")
		return
	}
	id := couchmessage.WorkerStartNoticeID(to, report.LaunchOrdinal)
	if _, err := c.sendGenerated(ctx, to, to, id, couchmessage.WorkerStartNoticeBody, false, c.service.captureHubInputPermit(to)); err != nil {
		skip("admission was refused or uncertain; no automatic retry")
	}
}
