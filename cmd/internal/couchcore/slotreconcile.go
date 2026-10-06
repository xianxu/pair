package couchcore

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"
)

// slotWorld is what the reconcile loop needs from the world: observe the
// slot, apply one step, and hold the host creation lease around git and
// filesystem steps. The production world is git and the filesystem
// (osSlotWorld); SlotWorld (tests) is an in-memory model of the same
// contract.
type slotWorld interface {
	lock() error
	unlock()
	observe(ctx context.Context) SlotObservation
	apply(ctx context.Context, step PlannedStep) error
}

// ReconcileResult is what a reconcile run did and where it ended.
type ReconcileResult struct {
	Observation SlotObservation
	Plan        SlotPlan
	Executed    []PlannedStep
	// Warnings: failures that left a working slot usable (KeepOnFailure).
	Warnings []StepFailure
}

// StepFailure is one step that did not converge, with its cause.
type StepFailure struct {
	Step PlannedStep
	Err  error
}

func (f StepFailure) Error() string {
	return fmt.Sprintf("%s did not converge: %v", f.Step.Resource, f.Err)
}

// ReconcileError is a run that stopped on a failed step or made no progress.
// Result is the state it stopped in.
type ReconcileError struct {
	Failure StepFailure
	Result  ReconcileResult
}

func (e *ReconcileError) Error() string { return e.Failure.Error() }
func (e *ReconcileError) Unwrap() error { return e.Failure.Err }

// errNoProgress: a step reported success but the slot did not change.
var errNoProgress = errors.New("the step reported success but nothing changed")

// reconcileLoop converges a slot (pair#387): observe → plan → apply, repeated
// until the plan is empty or holds only stops, a step fails, or a pass makes
// no progress. It keeps no state between runs (level-triggered), so recovery
// from any interruption is running it again. Within a run, Attempted keeps a
// step whose effect was already tried from being planned again.
func reconcileLoop(ctx context.Context, w slotWorld, savedWorkFull func() bool) (ReconcileResult, error) {
	var result ReconcileResult
	attempted := map[string]bool{}
	bound := len(SlotResources()) + 2
	for pass := 0; pass < bound; pass++ {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		if err := w.lock(); err != nil {
			if len(result.Executed) == 0 {
				return result, err
			}
			// Steps already ran: report the state they reached, not a bare error.
			return result, &ReconcileError{Failure: StepFailure{Step: result.Executed[len(result.Executed)-1], Err: err}, Result: result}
		}
		obs := w.observe(ctx)
		plan, err := PlanSlot(PlanInput{Observation: obs, Attempted: attempted, SavedWorkFull: savedWorkFull != nil && savedWorkFull()})
		result.Observation, result.Plan = obs, plan
		if err != nil {
			w.unlock()
			return result, err
		}
		if len(plan.Retried) > 0 {
			w.unlock()
			return result, &ReconcileError{Failure: StepFailure{Step: plan.Retried[0], Err: errNoProgress}, Result: result}
		}
		if len(plan.Steps) == 0 {
			w.unlock()
			return result, nil
		}
		for _, step := range plan.Steps {
			attempted[StepKey(step)] = true
			err := w.apply(ctx, step)
			result.Executed = append(result.Executed, step)
			if err == nil {
				continue
			}
			w.unlock()
			failure := StepFailure{Step: step, Err: err}
			if step.KeepOnFailure && ctx.Err() == nil {
				result.Warnings = append(result.Warnings, failure)
				result.Observation = w.observe(ctx)
				result.Plan, _ = PlanSlot(PlanInput{Observation: result.Observation, Attempted: attempted, SavedWorkFull: savedWorkFull != nil && savedWorkFull()})
				return result, nil
			}
			return result, &ReconcileError{Failure: failure, Result: result}
		}
		w.unlock()
	}
	return result, &ReconcileError{Failure: StepFailure{Step: PlannedStep{Resource: ResourceEnv}, Err: fmt.Errorf("no convergence within %d passes", bound)}, Result: result}
}

// osSlotWorld is the production world: git and the filesystem through the
// provisioner's command seam, under the host creation lease.
type osSlotWorld struct {
	p        *WorkspaceProvisioner
	layout   SlotLayout
	agent    EvidenceAgent
	remote   string
	progress io.Writer
	lease    *HostCreationLease
	req      ReconcileRequest
}

func (w *osSlotWorld) lock() error {
	lease, err := AcquireHostCreationLease(w.layout.common)
	if err != nil {
		return err
	}
	w.lease = lease
	return nil
}

func (w *osSlotWorld) unlock() {
	if w.lease != nil {
		w.lease.Close()
		w.lease = nil
	}
}

func (w *osSlotWorld) observe(ctx context.Context) SlotObservation {
	return ObserveSlot(ctx, SlotObserveInput{IO: w.p.IO, Layout: w.layout, Remote: w.remote, Agent: w.agent, IgnoreMemo: w.req.IgnoreMemo})
}

func (w *osSlotWorld) apply(ctx context.Context, step PlannedStep) error {
	cv := &slotConverger{p: w.p, layout: w.layout, lease: w.lease, remote: w.remote, progress: w.progress,
		agentNow: w.req.AgentNow, rename: w.req.rename}
	if step.Step != StepCompile {
		return cv.converge(ctx, step)
	}
	// weave compile runs without the lease (it can take minutes); the marker
	// is written only after the lease is retaken and the registration re-read.
	w.unlock()
	err := cv.compileSetup(ctx, func() (*HostCreationLease, error) {
		lease, err := AcquireHostCreationLease(w.layout.common)
		w.lease = lease
		return lease, err
	})
	return err
}

// ReconcileRequest names the slot to converge.
type ReconcileRequest struct {
	Layout   SlotLayout
	Agent    EvidenceAgent
	Remote   string
	Progress io.Writer
	// AgentNow re-reads the agent right before a set-aside (Couch supplies it).
	AgentNow func(context.Context) EvidenceAgent
	// IgnoreMemo compiles even when the same inputs failed before (R5).
	IgnoreMemo bool
	rename     func(oldpath, newpath string) error
}

// Reconcile converges a :1+ slot to a working slot (pair#387).
func (p *WorkspaceProvisioner) Reconcile(ctx context.Context, req ReconcileRequest) (ReconcileResult, error) {
	if ctx == nil {
		return ReconcileResult{}, errors.New("reconcile requires a context")
	}
	w := &osSlotWorld{p: p, layout: req.Layout, agent: req.Agent, remote: req.Remote, progress: req.Progress, req: req}
	defer w.unlock()
	// Saved work past retention is collected by its owner, at the start of
	// every run on the slot. Best effort: an entry that cannot be removed now
	// is collected by a later run.
	if removed, err := collectSavedWork(req.Layout, time.Now()); req.Progress != nil {
		for _, entry := range removed {
			fmt.Fprintf(req.Progress, "removed saved work past retention: %s\n", entry)
		}
		if err != nil {
			fmt.Fprintf(req.Progress, "saved-work collection deferred: %v\n", err)
		}
	}
	return reconcileLoop(ctx, w, func() bool {
		n, err := savedWorkEntries(req.Layout)
		return err == nil && n >= MaxSavedWork
	})
}
