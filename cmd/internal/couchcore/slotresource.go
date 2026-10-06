package couchcore

import "fmt"

// SlotResourceID names one piece of a :1+ slot's state (pair#387). A slot is
// not one thing: its state is spread over git, the filesystem, weave and Couch,
// and each piece is a resource with an owner, a desired state, an observation
// and an idempotent converge step. SlotResources is the table; the reconciler,
// couch --show and the recovery report all read it.
type SlotResourceID string

const (
	ResourceEnv          SlotResourceID = "env"
	ResourceStore        SlotResourceID = "store"
	ResourceIntent       SlotResourceID = "intent"
	ResourceBranch       SlotResourceID = "branch"
	ResourceUpstream     SlotResourceID = "upstream"
	ResourceRegistration SlotResourceID = "registration"
	ResourceHost         SlotResourceID = "host"
	ResourceDeps         SlotResourceID = "deps"
	// ResourceDep is the template for one dependency clone; an observation
	// names each instance as dep:<path relative to the environment>.
	ResourceDep   SlotResourceID = "dep"
	ResourceSetup SlotResourceID = "setup"
	ResourceAgent SlotResourceID = "agent"
)

// DepResource is the instance ID of the dependency clone at rel (a path
// relative to the slot environment).
func DepResource(rel string) SlotResourceID { return SlotResourceID(string(ResourceDep) + ":" + rel) }

// ResourceKind says who owns a resource and what reconcile may do to it.
type ResourceKind uint8

const (
	// KindDerived is recreatable from a source; reconcile converges it.
	KindDerived ResourceKind = iota + 1
	// KindInternal is Couch's own state; reconcile preserves it.
	KindInternal
	// KindUserData is the operator's; reconcile adopts it and never deletes it.
	KindUserData
	// KindRuntime is a running process; reconcile only observes it.
	KindRuntime
	// KindExternal is another system's file; reconcile only reads it.
	KindExternal
)

func (k ResourceKind) String() string {
	switch k {
	case KindDerived:
		return "derived"
	case KindInternal:
		return "internal"
	case KindUserData:
		return "user data"
	case KindRuntime:
		return "runtime"
	case KindExternal:
		return "external"
	}
	return fmt.Sprintf("kind(%d)", uint8(k))
}

// ObservedState is what an observation proved about a resource. Unknown is
// never absence: a failed probe proves nothing (lessons: probes are
// three-valued).
type ObservedState uint8

const (
	StatePresent ObservedState = iota + 1
	StateAbsent
	StateBroken
	StateUnknown
)

func (s ObservedState) String() string {
	switch s {
	case StatePresent:
		return "present"
	case StateAbsent:
		return "absent"
	case StateBroken:
		return "broken"
	case StateUnknown:
		return "unknown"
	}
	return fmt.Sprintf("state(%d)", uint8(s))
}

// AllObservedStates is the observation domain, for derived tests.
func AllObservedStates() []ObservedState {
	return []ObservedState{StatePresent, StateAbsent, StateBroken, StateUnknown}
}

// SlotResourceSpec is one row of the table.
type SlotResourceSpec struct {
	ID   SlotResourceID
	Kind ResourceKind
	// DependsOn are the resources that must be converged before this one.
	DependsOn []SlotResourceID
	// Layout names the SlotLayout methods that locate this resource; every
	// SlotLayout method belongs to exactly one resource.
	Layout []string
	// DesiredAbsent: converged means absent (a legacy file reconcile retires);
	// every other resource converges to present.
	DesiredAbsent bool
	// Removable: reconcile may make this resource absent. Only derived
	// resources are, and a checkout holding user data only by SetAside.
	Removable bool
	// SetAside: removal moves the directory whole into saved work, because it
	// may hold the operator's files.
	SetAside bool
	// SubStates refine a present or absent observation where the plan needs
	// more than the four states (enumerated here so derived domains cannot
	// skip one).
	SubStates []string
	// BrokenSubs are the readings a broken observation of this resource can
	// carry ("" is a broken reading without one).
	BrokenSubs []string
	// Summary is the operator-facing description (couch --show).
	Summary string
}

// SlotResources is the slot resource table, in topological order.
func SlotResources() []SlotResourceSpec {
	return []SlotResourceSpec{
		{ID: ResourceEnv, Kind: KindDerived, Layout: []string{"Env"}, BrokenSubs: []string{SubForeign}, Summary: "environment directory"},
		{ID: ResourceStore, Kind: KindInternal, DependsOn: []SlotResourceID{ResourceEnv}, Layout: []string{"Store", "SavedWork"}, BrokenSubs: []string{SubForeign}, Summary: "Couch slot store (preserved)"},
		{ID: ResourceIntent, Kind: KindDerived, Layout: []string{"Intent"}, DesiredAbsent: true, Removable: true, Summary: "legacy creation intent"},
		{ID: ResourceBranch, Kind: KindUserData, Layout: []string{"RestingBranch", "RestingRef"}, BrokenSubs: []string{SubElsewhere}, Summary: "resting branch (adopted, never deleted)"},
		{ID: ResourceUpstream, Kind: KindDerived, DependsOn: []SlotResourceID{ResourceBranch}, BrokenSubs: []string{SubConflict}, Summary: "resting branch upstream configuration"},
		{ID: ResourceRegistration, Kind: KindDerived, DependsOn: []SlotResourceID{ResourceBranch, ResourceEnv}, Layout: []string{"Registrations"}, Removable: true, BrokenSubs: []string{SubStale, SubLocked}, Summary: "git worktree registration"},
		{ID: ResourceHost, Kind: KindDerived, DependsOn: []SlotResourceID{ResourceRegistration, ResourceEnv, ResourceUpstream}, Layout: []string{"Host"}, Removable: true, SetAside: true, BrokenSubs: []string{SubForeign, SubMismatched, SubUnreadable}, Summary: "host checkout"},
		{ID: ResourceDeps, Kind: KindExternal, DependsOn: []SlotResourceID{ResourceHost}, Summary: "dependency declaration (construct/deps)"},
		{ID: ResourceDep, Kind: KindDerived, DependsOn: []SlotResourceID{ResourceDeps, ResourceEnv}, Removable: true, SetAside: true, BrokenSubs: []string{SubForeign, SubUnreadable, SubNotLayer}, Summary: "dependency clone"},
		{ID: ResourceSetup, Kind: KindDerived, DependsOn: []SlotResourceID{ResourceHost, ResourceDep}, Layout: []string{"SetupLock"}, SubStates: []string{SubWithWarning, SubFailedKnown, SubLockHeld, SubMarkerValid}, BrokenSubs: []string{""}, Summary: "weave setup and Couch's marker"},
		{ID: ResourceAgent, Kind: KindRuntime, DependsOn: []SlotResourceID{ResourceHost}, Summary: "agent session and thread record"},
	}
}

// Desired is the state this resource converges to.
func (s SlotResourceSpec) Desired() ObservedState {
	if s.DesiredAbsent {
		return StateAbsent
	}
	return StatePresent
}

// SlotResource returns the spec for id (a dep:<rel> instance resolves to the
// dep template).
func SlotResource(id SlotResourceID) (SlotResourceSpec, bool) {
	base := id
	if len(id) > len(ResourceDep) && id[:len(ResourceDep)+1] == ResourceDep+":" {
		base = ResourceDep
	}
	for _, r := range SlotResources() {
		if r.ID == base {
			return r, true
		}
	}
	return SlotResourceSpec{}, false
}

// SlotResourceOrder is the converge order (a topological order of the
// DependsOn graph, stable in table order). It fails on an unknown edge or a
// cycle, which TestSlotResourcesFormADAG pins.
func SlotResourceOrder() ([]SlotResourceID, error) {
	specs := SlotResources()
	index := make(map[SlotResourceID]int, len(specs))
	for i, s := range specs {
		if _, dup := index[s.ID]; dup {
			return nil, fmt.Errorf("duplicate slot resource %s", s.ID)
		}
		index[s.ID] = i
	}
	const (
		unvisited = iota
		visiting
		done
	)
	state := make([]int, len(specs))
	var order []SlotResourceID
	var visit func(i int) error
	visit = func(i int) error {
		switch state[i] {
		case visiting:
			return fmt.Errorf("slot resource cycle at %s", specs[i].ID)
		case done:
			return nil
		}
		state[i] = visiting
		for _, d := range specs[i].DependsOn {
			j, ok := index[d]
			if !ok {
				return fmt.Errorf("slot resource %s depends on undeclared %s", specs[i].ID, d)
			}
			if err := visit(j); err != nil {
				return err
			}
		}
		state[i] = done
		order = append(order, specs[i].ID)
		return nil
	}
	for i := range specs {
		if err := visit(i); err != nil {
			return nil, err
		}
	}
	return order, nil
}

// SlotResourceDependents is every resource that transitively depends on id
// (the nodes an unknown observation of id stops).
func SlotResourceDependents(id SlotResourceID) map[SlotResourceID]bool {
	out := map[SlotResourceID]bool{}
	changed := true
	for changed {
		changed = false
		for _, s := range SlotResources() {
			if out[s.ID] {
				continue
			}
			for _, d := range s.DependsOn {
				if d == id || out[d] {
					out[s.ID] = true
					changed = true
					break
				}
			}
		}
	}
	return out
}
