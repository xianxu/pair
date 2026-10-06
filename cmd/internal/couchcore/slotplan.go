package couchcore

import (
	"fmt"
	"strings"
)

// ConvergeStep is one idempotent action that moves one resource toward its
// desired state (pair#387).
type ConvergeStep string

const (
	StepMkdirEnv     ConvergeStep = "mkdir-env"
	StepRemoveIntent ConvergeStep = "remove-intent"
	StepCreateBranch ConvergeStep = "create-branch"
	StepSetUpstream  ConvergeStep = "set-upstream"
	StepRepairHost   ConvergeStep = "repair-host"
	StepWorktreeAdd  ConvergeStep = "worktree-add"
	StepSetAside     ConvergeStep = "set-aside"
	StepCompile      ConvergeStep = "compile"
)

// PlannedStep is a step with the facts its converge needs.
type PlannedStep struct {
	Step     ConvergeStep
	Resource SlotResourceID
	// Path: the checkout SetAside moves.
	Path string
	// Branch: the branch WorktreeAdd checks out (the registration's own, else
	// the resting branch); Force re-adds over a stale registration.
	Branch string
	Force  bool
	// KeepOnFailure: a failure leaves a working slot usable (rule 4); the run
	// ends with a warning instead of an error.
	KeepOnFailure bool
}

func (s PlannedStep) String() string {
	parts := []string{string(s.Step), string(s.Resource)}
	if s.Path != "" {
		parts = append(parts, s.Path)
	}
	if s.Branch != "" {
		parts = append(parts, "on "+s.Branch)
	}
	return strings.Join(parts, " ")
}

// StopClass says why the plan stops at a resource.
type StopClass string

const (
	// StopUnknown: the resource could not be observed; nothing depending on it
	// is planned.
	StopUnknown StopClass = "unknown"
	// StopHandoff: reconcile cannot fix it; the repository's :0 agent can.
	StopHandoff StopClass = "handoff"
	// StopRetryable: something else is running (weave setup); run it again.
	StopRetryable StopClass = "retryable"
	// StopHold: reconcile may not act now (a live agent, saved work full).
	StopHold StopClass = "hold"
)

// Stop reasons the outcome severity reads (OutcomeSeverity, Task 2.4).
const (
	StopReasonAgentLive      = "agent-live"
	StopReasonSavedWorkFull  = "saved-work-full"
	StopReasonSetupRunning   = "setup-running"
	StopReasonSetupKnownFail = "setup-failed-known"
)

// PlanStop is one place the plan cannot proceed.
type PlanStop struct {
	Resource SlotResourceID
	Class    StopClass
	Reason   string
}

// SlotPlan is the converge plan for one observation. It is pure data.
type SlotPlan struct {
	Steps []PlannedStep
	Stops []PlanStop
}

// Empty: nothing to do and nothing in the way, the slot is converged.
func (p SlotPlan) Empty() bool { return len(p.Steps) == 0 && len(p.Stops) == 0 }

// OnlyStops: nothing can be done until a stop clears.
func (p SlotPlan) OnlyStops() bool { return len(p.Steps) == 0 && len(p.Stops) > 0 }

// PlanInput is what PlanSlot decides from.
type PlanInput struct {
	Observation SlotObservation
	// Attempted names steps already executed in this run (StepKey), so a step
	// whose converge did not change the observation is not planned again in
	// the same run (RepairHost before SetAside, the no-progress guard).
	Attempted map[string]bool
	// SavedWorkFull: the slot's saved-work entry limit is reached.
	SavedWorkFull bool
}

// StepKey identifies a step within one run.
func StepKey(s PlannedStep) string { return string(s.Step) + " " + string(s.Resource) }

// templateOf maps a dep:<rel> instance to the dep template.
func templateOf(id SlotResourceID) SlotResourceID {
	if strings.HasPrefix(string(id), string(ResourceDep)+":") {
		return ResourceDep
	}
	return id
}

// PlanSlot decides the converge steps for one observation (pair#387). Rules:
//  1. Unknown blocks: an unknown resource is a stop, and nothing that depends
//     on it transitively is planned.
//  2. User data is never deleted: a checkout broken on positive evidence is set
//     aside whole (SetAside), the only step that removes one.
//  3. No removal under a running or unknown agent (a hold).
//  4. Repair never makes a working slot worse: a compile under a valid marker
//     keeps the slot usable on failure.
//  5. :0 is never planned.
//  6. Idempotent by construction: each step's precondition is the observed
//     state it fixes, so a converged resource yields no step.
func PlanSlot(in PlanInput) (SlotPlan, error) {
	o := in.Observation
	if o.Number <= 0 {
		return SlotPlan{}, fmt.Errorf("slot :%d is not reconcilable; :0 is the primary checkout", o.Number)
	}
	var plan SlotPlan
	blocked := map[SlotResourceID]bool{}
	for _, r := range o.Resources {
		if r.State == StateUnknown && r.ID != ResourceAgent {
			plan.Stops = append(plan.Stops, PlanStop{Resource: r.ID, Class: StopUnknown, Reason: r.Reason})
			for id := range SlotResourceDependents(templateOf(r.ID)) {
				blocked[id] = true
			}
		}
	}
	running, known := AgentRunning(o.Agent)
	mayRemove := known && !running
	get := func(id SlotResourceID) ResourceObservation { r, _ := o.Get(id); return r }
	step := func(s PlannedStep) {
		if blocked[templateOf(s.Resource)] || in.Attempted[StepKey(s)] {
			return
		}
		plan.Steps = append(plan.Steps, s)
	}
	stop := func(id SlotResourceID, class StopClass, reason string) {
		if blocked[templateOf(id)] {
			return
		}
		plan.Stops = append(plan.Stops, PlanStop{Resource: id, Class: class, Reason: reason})
	}
	setAside := func(r ResourceObservation, path string) {
		switch {
		case !mayRemove:
			stop(r.ID, StopHold, StopReasonAgentLive)
		case in.SavedWorkFull:
			stop(r.ID, StopHold, StopReasonSavedWorkFull)
		default:
			step(PlannedStep{Step: StepSetAside, Resource: r.ID, Path: path})
		}
	}

	if env := get(ResourceEnv); env.State == StateAbsent {
		step(PlannedStep{Step: StepMkdirEnv, Resource: ResourceEnv})
	} else if env.State == StateBroken {
		stop(ResourceEnv, StopHandoff, env.Reason)
	}
	if store := get(ResourceStore); store.State == StateBroken {
		stop(ResourceStore, StopHandoff, store.Reason)
	}
	if get(ResourceIntent).State == StatePresent {
		step(PlannedStep{Step: StepRemoveIntent, Resource: ResourceIntent})
	}
	branch := get(ResourceBranch)
	switch branch.State {
	case StateAbsent:
		step(PlannedStep{Step: StepCreateBranch, Resource: ResourceBranch})
	case StateBroken:
		stop(ResourceBranch, StopHandoff, branch.Reason)
	}
	switch up := get(ResourceUpstream); up.State {
	case StateAbsent:
		step(PlannedStep{Step: StepSetUpstream, Resource: ResourceUpstream})
	case StateBroken:
		stop(ResourceUpstream, StopHandoff, up.Reason)
	}
	reg := get(ResourceRegistration)
	if reg.State == StateBroken && reg.Sub == SubLocked {
		stop(ResourceRegistration, StopHandoff, reg.Reason)
	}
	host := get(ResourceHost)
	hostReady := get(ResourceEnv).State == StatePresent && branch.State == StatePresent && get(ResourceUpstream).State == StatePresent &&
		(reg.State == StateAbsent || (reg.State == StateBroken && reg.Sub == SubStale))
	switch {
	case host.State == StateAbsent && hostReady:
		target := reg.Branch
		if target == "" {
			target = NewSlotLayout(o.Primary, "", o.Number).RestingBranch()
		}
		step(PlannedStep{Step: StepWorktreeAdd, Resource: ResourceHost, Branch: target, Force: reg.State == StateBroken})
	case host.State == StateBroken && host.Sub == SubForeign:
		stop(ResourceHost, StopHandoff, host.Reason)
	case host.State == StateBroken && (host.Sub == SubMismatched || host.Sub == SubUnreadable):
		repair := PlannedStep{Step: StepRepairHost, Resource: ResourceHost}
		switch {
		case !in.Attempted[StepKey(repair)]:
			step(repair)
		case host.Sub == SubUnreadable:
			setAside(host, NewSlotLayout(o.Primary, "", o.Number).Host())
		default:
			stop(ResourceHost, StopHandoff, host.Reason)
		}
	}

	setupReady := host.State == StatePresent && get(ResourceDeps).State == StatePresent
	for _, r := range o.Resources {
		if templateOf(r.ID) != ResourceDep || r.Sub == SubOutside {
			continue
		}
		switch {
		case r.State == StateBroken && r.Sub == SubUnreadable:
			setupReady = false
			setAside(r, r.Dep.Path)
		case r.State == StateBroken:
			setupReady = false
			stop(r.ID, StopHandoff, r.Reason)
		case r.State == StateUnknown:
			setupReady = false
		}
	}
	setup := get(ResourceSetup)
	switch {
	case setup.Sub == SubLockHeld:
		stop(ResourceSetup, StopRetryable, StopReasonSetupRunning)
	case setup.Sub == SubFailedKnown:
		stop(ResourceSetup, StopHandoff, StopReasonSetupKnownFail+": "+setup.Reason)
	case (setup.State == StateAbsent || setup.State == StateBroken) && setupReady:
		step(PlannedStep{Step: StepCompile, Resource: ResourceSetup, KeepOnFailure: setup.Sub == SubMarkerValid})
	}
	return plan, nil
}
