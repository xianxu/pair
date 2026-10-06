package couchcore

import (
	"fmt"
	"testing"
)

const planTestPrimary = "/f/pair"

// healthyObservation is a converged slot with one dependency clone.
func healthyObservation() SlotObservation {
	dep := DeclaredDep{Rel: "ariadne", Path: "/f/worktree/pair-slot1/ariadne", Owner: "/f/worktree/pair-slot1/pair", Present: true}
	return SlotObservation{Primary: planTestPrimary, Number: 1, Agent: AgentNone, Resources: []ResourceObservation{
		{ID: ResourceEnv, State: StatePresent},
		{ID: ResourceStore, State: StatePresent},
		{ID: ResourceIntent, State: StateAbsent},
		{ID: ResourceBranch, State: StatePresent},
		{ID: ResourceUpstream, State: StatePresent},
		{ID: ResourceRegistration, State: StatePresent, Branch: "issue-work"},
		{ID: ResourceHost, State: StatePresent, Branch: "issue-work"},
		{ID: ResourceDeps, State: StatePresent},
		{ID: DepResource("ariadne"), State: StatePresent, Dep: &dep},
		{ID: ResourceSetup, State: StatePresent},
		{ID: ResourceAgent, State: StateAbsent},
	}}
}

// perturbation sets one resource to a state and reading.
type perturbation struct {
	id    SlotResourceID
	state ObservedState
	sub   string
}

func (p perturbation) String() string { return fmt.Sprintf("%s=%s/%s", p.id, p.state, p.sub) }

// slotPerturbations is the derived single-resource domain: every resource of
// the table (the dep template as its instance), every observed state, and for
// broken every declared broken reading, plus every declared sub-state on
// present and absent.
func slotPerturbations() []perturbation {
	var out []perturbation
	for _, spec := range SlotResources() {
		id := spec.ID
		if id == ResourceDep {
			id = DepResource("ariadne")
		}
		if id == ResourceAgent {
			continue // the agent is varied through the agent domain
		}
		for _, state := range AllObservedStates() {
			subs := []string{""}
			if state == StateBroken && len(spec.BrokenSubs) > 0 {
				subs = spec.BrokenSubs
			}
			for _, sub := range subs {
				out = append(out, perturbation{id, state, sub})
			}
		}
		for _, sub := range spec.SubStates {
			out = append(out, perturbation{id, StatePresent, sub}, perturbation{id, StateAbsent, sub})
		}
	}
	return out
}

func perturb(o SlotObservation, ps ...perturbation) SlotObservation {
	out := o
	out.Resources = append([]ResourceObservation(nil), o.Resources...)
	for _, p := range ps {
		for i := range out.Resources {
			if out.Resources[i].ID == p.id {
				out.Resources[i].State, out.Resources[i].Sub = p.state, p.sub
			}
		}
	}
	return out
}

// planInvariants are stated independently of PlanSlot's rules.
func planInvariants(t *testing.T, label string, o SlotObservation, plan SlotPlan) {
	t.Helper()
	order, _ := SlotResourceOrder()
	rank := map[SlotResourceID]int{}
	for i, id := range order {
		rank[id] = i
	}
	unknownBlocks := map[SlotResourceID]bool{}
	for _, r := range o.Resources {
		if r.State == StateUnknown && r.ID != ResourceAgent {
			unknownBlocks[r.ID] = true
			for id := range SlotResourceDependents(templateOf(r.ID)) {
				unknownBlocks[id] = true
			}
		}
	}
	running, known := AgentRunning(o.Agent)
	last := -1
	for _, s := range plan.Steps {
		r, ok := o.Get(s.Resource)
		if !ok {
			t.Errorf("%s: step %s targets an unobserved resource", label, s)
			continue
		}
		// I1: a converged resource yields no step.
		spec, _ := SlotResource(r.ID)
		if r.State == spec.Desired() && (r.Sub == "" || r.Sub == SubWithWarning) {
			t.Errorf("%s: I1 step %s on converged %s", label, s, r.ID)
		}
		// I2: nothing unknown, nothing depending on an unknown.
		if unknownBlocks[r.ID] || unknownBlocks[templateOf(r.ID)] {
			t.Errorf("%s: I2 step %s on/under an unknown resource", label, s)
		}
		// I3: SetAside is the only step that removes a checkout, and only one
		// broken on positive evidence.
		if templateOf(r.ID) == ResourceDep && s.Step != StepSetAside {
			t.Errorf("%s: I3 step %s on a dependency clone", label, s)
		}
		if s.Step == StepSetAside && !(r.State == StateBroken && r.Sub == SubUnreadable) {
			t.Errorf("%s: I3 set-aside of %s/%s, not unreadable", label, r.State, r.Sub)
		}
		// I4: never under a running or unknown agent.
		if s.Step == StepSetAside && (running || !known) {
			t.Errorf("%s: I4 set-aside under agent %s", label, o.Agent)
		}
		// I5: the store and the resting branch are never removed.
		if r.ID == ResourceStore || (r.ID == ResourceBranch && s.Step != StepCreateBranch) {
			t.Errorf("%s: I5 step %s on preserved %s", label, s, r.ID)
		}
		// I6: converge order.
		if rank[templateOf(r.ID)] < last {
			t.Errorf("%s: I6 step %s out of topological order", label, s)
		}
		last = rank[templateOf(r.ID)]
	}
}

func TestPlanSlotDomain(t *testing.T) {
	domain := slotPerturbations()
	pairs := 0
	for _, agent := range AllEvidenceAgents() {
		for i, a := range domain {
			o := healthyObservation()
			o.Agent = agent
			single := perturb(o, a)
			plan, err := PlanSlot(PlanInput{Observation: single})
			if err != nil {
				t.Fatalf("%s: %v", a, err)
			}
			planInvariants(t, fmt.Sprintf("agent=%s %s", agent, a), single, plan)
			for _, b := range domain[i+1:] {
				if b.id == a.id {
					continue
				}
				both := perturb(o, a, b)
				plan, err := PlanSlot(PlanInput{Observation: both})
				if err != nil {
					t.Fatalf("%s %s: %v", a, b, err)
				}
				planInvariants(t, fmt.Sprintf("agent=%s %s %s", agent, a, b), both, plan)
				pairs++
			}
		}
	}
	if pairs < 1000 {
		t.Fatalf("pair domain has only %d points; the generator is not reading the table", pairs)
	}
	if plan, _ := PlanSlot(PlanInput{Observation: healthyObservation()}); !plan.Empty() {
		t.Fatalf("a healthy slot plans %+v", plan)
	}
}

func TestPlanSlotNamedCases(t *testing.T) {
	t.Run("tools:1 interrupted setup", func(t *testing.T) {
		o := perturb(healthyObservation(), perturbation{DepResource("ariadne"), StateAbsent, ""}, perturbation{ResourceSetup, StateAbsent, ""})
		plan, _ := PlanSlot(PlanInput{Observation: o})
		if len(plan.Steps) != 1 || plan.Steps[0].Step != StepCompile || plan.Steps[0].KeepOnFailure {
			t.Fatalf("plan %+v, want one compile that fails loudly (no valid marker)", plan)
		}
	})
	t.Run("valid marker, missing dependency", func(t *testing.T) {
		o := perturb(healthyObservation(), perturbation{DepResource("ariadne"), StateAbsent, ""}, perturbation{ResourceSetup, StateAbsent, SubMarkerValid})
		plan, _ := PlanSlot(PlanInput{Observation: o})
		if len(plan.Steps) != 1 || plan.Steps[0].Step != StepCompile || !plan.Steps[0].KeepOnFailure {
			t.Fatalf("plan %+v, want one compile that keeps the slot usable", plan)
		}
	})
	t.Run("deleted slot directory", func(t *testing.T) {
		o := perturb(healthyObservation(),
			perturbation{ResourceEnv, StateAbsent, ""}, perturbation{ResourceStore, StatePending, ""},
			perturbation{ResourceRegistration, StateBroken, SubStale}, perturbation{ResourceHost, StatePending, ""},
			perturbation{ResourceDeps, StatePending, ""}, perturbation{DepResource("ariadne"), StatePending, ""}, perturbation{ResourceSetup, StatePending, ""})
		plan, _ := PlanSlot(PlanInput{Observation: o})
		if len(plan.Steps) != 1 || plan.Steps[0].Step != StepMkdirEnv {
			t.Fatalf("first pass %+v, want mkdir-env", plan)
		}
		// Second pass: the environment exists, the host does not.
		o = perturb(o, perturbation{ResourceEnv, StatePresent, ""}, perturbation{ResourceHost, StateAbsent, ""})
		plan, _ = PlanSlot(PlanInput{Observation: o})
		if len(plan.Steps) != 1 || plan.Steps[0].Step != StepWorktreeAdd || !plan.Steps[0].Force || plan.Steps[0].Branch != "issue-work" {
			t.Fatalf("second pass %+v, want a forced re-add on the registration's branch issue-work", plan)
		}
	})
	t.Run("unreadable host: repair first, then set aside", func(t *testing.T) {
		o := perturb(healthyObservation(), perturbation{ResourceHost, StateBroken, SubUnreadable},
			perturbation{ResourceDeps, StatePending, ""}, perturbation{DepResource("ariadne"), StatePending, ""}, perturbation{ResourceSetup, StatePending, ""})
		plan, _ := PlanSlot(PlanInput{Observation: o})
		if len(plan.Steps) != 1 || plan.Steps[0].Step != StepRepairHost {
			t.Fatalf("first %+v, want repair-host", plan)
		}
		plan, _ = PlanSlot(PlanInput{Observation: o, Attempted: map[string]bool{StepKey(plan.Steps[0]): true}})
		if len(plan.Steps) != 1 || plan.Steps[0].Step != StepSetAside || plan.Steps[0].Path != "/f/worktree/pair-slot1/pair" {
			t.Fatalf("after repair %+v, want set-aside of the host", plan)
		}
		o.Agent = AgentDetached
		plan, _ = PlanSlot(PlanInput{Observation: o, Attempted: map[string]bool{"repair-host host": true}})
		if len(plan.Steps) != 0 || len(plan.Stops) != 1 || plan.Stops[0].Reason != StopReasonAgentLive {
			t.Fatalf("under a detached agent %+v, want only the agent-live hold", plan)
		}
	})
	t.Run("unknown host blocks exactly its dependents", func(t *testing.T) {
		// Written out by hand, independent of SlotResourceDependents: an
		// unknown host stops at host, and nothing on deps, the clone or setup
		// is planned, while the unrelated legacy intent still converges.
		o := perturb(healthyObservation(), perturbation{ResourceHost, StateUnknown, ""},
			perturbation{DepResource("ariadne"), StateBroken, SubUnreadable}, perturbation{ResourceSetup, StateAbsent, ""},
			perturbation{ResourceIntent, StatePresent, ""})
		plan, _ := PlanSlot(PlanInput{Observation: o})
		if len(plan.Steps) != 1 || plan.Steps[0].Step != StepRemoveIntent {
			t.Fatalf("steps %+v, want only remove-intent", plan.Steps)
		}
		if len(plan.Stops) != 1 || plan.Stops[0].Resource != ResourceHost || plan.Stops[0].Class != StopUnknown {
			t.Fatalf("stops %+v, want exactly the unknown host", plan.Stops)
		}
	})
	t.Run("a held weave lock stops only pending setup work", func(t *testing.T) {
		o := perturb(healthyObservation(), perturbation{ResourceSetup, StatePresent, SubLockHeld})
		if plan, _ := PlanSlot(PlanInput{Observation: o}); !plan.Empty() {
			t.Fatalf("converged slot with a running compile plans %+v", plan)
		}
		o = perturb(healthyObservation(), perturbation{ResourceSetup, StateAbsent, SubLockHeld})
		plan, _ := PlanSlot(PlanInput{Observation: o})
		if len(plan.Steps) != 0 || len(plan.Stops) != 1 || plan.Stops[0].Class != StopRetryable {
			t.Fatalf("pending setup under a held lock plans %+v, want one retryable stop", plan)
		}
	})
	t.Run("an unknown agent holds with its own reason", func(t *testing.T) {
		o := perturb(healthyObservation(), perturbation{DepResource("ariadne"), StateBroken, SubUnreadable})
		o.Agent = AgentUnusableUnknown
		plan, _ := PlanSlot(PlanInput{Observation: o})
		if len(plan.Stops) != 1 || plan.Stops[0].Reason != StopReasonAgentUnknown {
			t.Fatalf("plan %+v, want the unknown-agent hold", plan)
		}
	})
	t.Run(":0 is refused", func(t *testing.T) {
		o := healthyObservation()
		o.Number = 0
		if _, err := PlanSlot(PlanInput{Observation: o}); err == nil {
			t.Fatal(":0 planned")
		}
	})
}
