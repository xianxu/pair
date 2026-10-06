package couchcore

import (
	"context"
	"errors"
	"fmt"
)

// SlotWorld is an in-memory model of a slot for the reconcile loop: what the
// observer would read, and what each converge step does to it. It agrees with
// real git on every perturbation the fixture can produce
// (TestReconcileAgreesWithRealGit); the pair domain runs only here.
type SlotWorld struct {
	truth map[SlotResourceID]ResourceObservation
	// SourceKnown: compile can clone a missing dependency.
	SourceKnown bool
	// RepairFixes: git worktree repair restores a broken host.
	RepairFixes bool
	// CrashAfter aborts the run after that many applied steps (0: never).
	CrashAfter int
	Agent      EvidenceAgent
	applied    int
	Effects    []string
}

var errInjectedCrash = errors.New("injected crash")

const fakeDep = "ariadne"

func newSlotWorld() *SlotWorld {
	o := healthyObservation()
	w := &SlotWorld{truth: map[SlotResourceID]ResourceObservation{}, SourceKnown: true, RepairFixes: true, Agent: AgentNone}
	for _, r := range o.Resources {
		w.truth[r.ID] = r
	}
	return w
}

// set records one perturbation, with the couplings real disks have: a
// deleted environment takes the host directory with it, and a host directory
// that is gone leaves its registration stale.
func (w *SlotWorld) set(p perturbation) {
	r := w.truth[p.id]
	r.State, r.Sub = p.state, p.sub
	w.truth[p.id] = r
	if p.id == ResourceEnv && p.state == StateAbsent {
		if w.truth[ResourceHost].State != StateUnknown {
			w.set(perturbation{ResourceHost, StateAbsent, ""})
		}
		if dep := DepResource(fakeDep); w.truth[dep].State != StateUnknown {
			w.set(perturbation{dep, StateAbsent, ""}) // the clone lived in the environment
		}
	}
	if p.id == ResourceHost && p.state == StateAbsent {
		if reg := w.truth[ResourceRegistration]; reg.State == StatePresent {
			reg.State, reg.Sub = StateBroken, SubStale
			w.truth[ResourceRegistration] = reg
		}
	}
}

func (w *SlotWorld) get(id SlotResourceID) ResourceObservation { return w.truth[id] }

func (w *SlotWorld) lock() error { return nil }
func (w *SlotWorld) unlock()     {}

// observe derives what ObserveSlot would read from the truth.
func (w *SlotWorld) observe(context.Context) SlotObservation {
	t := map[SlotResourceID]ResourceObservation{}
	for id, r := range w.truth {
		t[id] = r
	}
	pend := func(id SlotResourceID, why string) {
		t[id] = ResourceObservation{ID: id, State: StatePending, Reason: why}
	}
	if t[ResourceEnv].State != StatePresent {
		pend(ResourceStore, "waits for env")
	}
	if t[ResourceBranch].State != StatePresent {
		pend(ResourceUpstream, "waits for branch")
	}
	if t[ResourceEnv].State != StatePresent {
		pend(ResourceHost, "waits for env")
	}
	dep := DepResource(fakeDep)
	if t[ResourceHost].State != StatePresent {
		pend(ResourceDeps, "waits for host")
		pend(dep, "waits for host")
		pend(ResourceSetup, "waits for host")
	} else if t[ResourceDeps].State != StatePresent {
		pend(dep, "waits for deps")
	}
	if s := t[ResourceSetup]; s.State == StatePresent && s.Sub == "" {
		if d := t[dep]; d.State == StateAbsent || d.State == StateBroken {
			s.State, s.Sub = StateAbsent, SubMarkerValid
			t[ResourceSetup] = s
		}
	}
	obs := SlotObservation{Primary: planTestPrimary, Number: 1, Agent: w.Agent}
	for _, id := range []SlotResourceID{ResourceEnv, ResourceStore, ResourceIntent, ResourceBranch, ResourceUpstream, ResourceRegistration, ResourceHost, ResourceDeps, dep, ResourceSetup, ResourceAgent} {
		obs.Resources = append(obs.Resources, t[id])
	}
	return obs
}

func (w *SlotWorld) apply(_ context.Context, s PlannedStep) error {
	if w.CrashAfter > 0 && w.applied >= w.CrashAfter {
		panic(errInjectedCrash) // the process dies: nothing returns
	}
	w.applied++
	w.Effects = append(w.Effects, s.String())
	present := func(id SlotResourceID) {
		w.truth[id] = ResourceObservation{ID: id, State: StatePresent, Branch: w.truth[id].Branch, Dep: w.truth[id].Dep}
	}
	switch s.Step {
	case StepMkdirEnv:
		present(ResourceEnv)
	case StepRemoveIntent:
		w.set(perturbation{ResourceIntent, StateAbsent, ""})
	case StepCreateBranch:
		present(ResourceBranch)
	case StepSetUpstream:
		present(ResourceUpstream)
	case StepRepairHost:
		if w.RepairFixes {
			present(ResourceHost)
		}
	case StepWorktreeAdd:
		present(ResourceHost)
		present(ResourceRegistration)
		r := w.truth[ResourceHost]
		r.Branch = s.Branch
		w.truth[ResourceHost] = r
		if w.truth[ResourceDeps].State == StatePending {
			present(ResourceDeps)
		}
	case StepSetAside:
		if s.Resource == ResourceHost {
			w.set(perturbation{ResourceHost, StateAbsent, ""})
			w.set(perturbation{ResourceRegistration, StateBroken, SubStale})
		} else {
			w.set(perturbation{s.Resource, StateAbsent, ""})
		}
	case StepCompile:
		dep := DepResource(fakeDep)
		if d := w.truth[dep]; d.State == StateAbsent {
			if !w.SourceKnown {
				return fmt.Errorf("Error: missing substrate %s declared in host: record its source in construct/deps", fakeDep)
			}
			present(dep)
		}
		present(ResourceSetup)
	default:
		return fmt.Errorf("fake: unknown step %s", s.Step)
	}
	return nil
}
