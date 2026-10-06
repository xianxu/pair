package couchcore

import (
	"reflect"
	"testing"
)

func TestSlotResourcesFormADAG(t *testing.T) {
	order, err := SlotResourceOrder()
	if err != nil {
		t.Fatal(err)
	}
	if len(order) != len(SlotResources()) {
		t.Fatalf("order has %d resources, table %d", len(order), len(SlotResources()))
	}
	position := map[SlotResourceID]int{}
	for i, id := range order {
		position[id] = i
	}
	for _, s := range SlotResources() {
		for _, d := range s.DependsOn {
			if position[d] >= position[s.ID] {
				t.Errorf("%s converges before its dependency %s", s.ID, d)
			}
		}
	}
	again, _ := SlotResourceOrder()
	if !reflect.DeepEqual(order, again) {
		t.Fatal("SlotResourceOrder is not stable")
	}
}

func TestSlotResourceKinds(t *testing.T) {
	for _, s := range SlotResources() {
		if s.Removable && s.Kind != KindDerived {
			t.Errorf("%s is removable but %s; only derived resources may be made absent", s.ID, s.Kind)
		}
		if s.SetAside && !s.Removable {
			t.Errorf("%s sets aside but is not removable", s.ID)
		}
		if s.Summary == "" {
			t.Errorf("%s has no summary", s.ID)
		}
	}
	for _, id := range []SlotResourceID{ResourceStore, ResourceBranch, ResourceAgent, ResourceDeps} {
		if s, _ := SlotResource(id); s.Removable {
			t.Errorf("%s must never be removable", id)
		}
	}
	// Every checkout that can hold the operator's files is set aside, never deleted.
	for _, id := range []SlotResourceID{ResourceHost, ResourceDep} {
		if s, _ := SlotResource(id); !s.SetAside {
			t.Errorf("%s must be set aside, not deleted", id)
		}
	}
	if s, ok := SlotResource(DepResource("ariadne")); !ok || s.ID != ResourceDep {
		t.Fatal("dep:<rel> does not resolve to the dep template")
	}
}

func TestSlotResourceDependentsAreTransitive(t *testing.T) {
	got := SlotResourceDependents(ResourceHost)
	for _, id := range []SlotResourceID{ResourceDeps, ResourceDep, ResourceSetup, ResourceAgent} {
		if !got[id] {
			t.Errorf("%s does not depend on host transitively", id)
		}
	}
	for _, id := range []SlotResourceID{ResourceEnv, ResourceBranch, ResourceIntent, ResourceHost} {
		if got[id] {
			t.Errorf("%s wrongly depends on host", id)
		}
	}
}

// TestSlotLayoutMethodsBelongToOneResource is the consistency half of the
// coverage audit: the SlotLayout method set (read by reflection, not listed)
// and the table's Layout names are a bijection.
func TestSlotLayoutMethodsBelongToOneResource(t *testing.T) {
	owner := map[string]SlotResourceID{}
	for _, s := range SlotResources() {
		for _, m := range s.Layout {
			if prev, dup := owner[m]; dup {
				t.Errorf("SlotLayout.%s belongs to both %s and %s", m, prev, s.ID)
			}
			owner[m] = s.ID
		}
	}
	typ := reflect.TypeOf(SlotLayout{})
	methods := map[string]bool{}
	for i := 0; i < typ.NumMethod(); i++ {
		name := typ.Method(i).Name
		methods[name] = true
		if _, ok := owner[name]; !ok {
			t.Errorf("SlotLayout.%s is not the location of any slot resource", name)
		}
	}
	for m, id := range owner {
		if !methods[m] {
			t.Errorf("resource %s names SlotLayout.%s, which does not exist", id, m)
		}
	}
}
