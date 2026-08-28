package sessioninventory

import (
	"reflect"
	"slices"
	"testing"
)

func TestResolveBindings(t *testing.T) {
	t.Parallel()
	inventory := bindingTestInventory()
	t.Run("ledger wins and lower disagreement stays negative", func(t *testing.T) {
		resolved := ResolveBindings(inventory, []BindingInput{{
			ScopeKey: "scope", Tag: "work", Agent: AgentClaude, LaunchPresent: true,
			LedgerRootNodeID: "root-a", ConfigRootNodeID: "root-b",
			LiveRounds: []RoundObservation{{RootNodeID: "root-b", PairPositions: []uint64{10}, NativePositions: []uint64{1}, Fingerprints: []string{"fp"}}},
		}})
		binding := onlyBinding(t, resolved)
		if binding.Status != BindingEstablished || binding.RootNodeID == nil || *binding.RootNodeID != "root-a" {
			t.Fatalf("binding = %#v", binding)
		}
		for _, evidence := range binding.Evidence {
			if evidence.Kind != EvidenceLedger && evidence.Positive {
				t.Errorf("lower contradictory evidence remained positive: %#v", evidence)
			}
		}
	})

	t.Run("later live rounds intersect ambiguity", func(t *testing.T) {
		resolved := ResolveBindings(inventory, []BindingInput{{
			ScopeKey: "scope", Tag: "work", Agent: AgentClaude, LaunchPresent: true,
			LiveRounds: []RoundObservation{
				{RootNodeID: "root-a", PairPositions: []uint64{10}},
				{RootNodeID: "root-b", PairPositions: []uint64{10}},
				{RootNodeID: "root-b", PairPositions: []uint64{20}},
				{RootNodeID: "root-c", PairPositions: []uint64{20}},
			},
		}})
		binding := onlyBinding(t, resolved)
		if binding.Status != BindingProvisional || binding.RootNodeID == nil || *binding.RootNodeID != "root-b" || len(resolved.Ambiguities) != 0 {
			t.Fatalf("binding=%#v ambiguities=%#v", binding, resolved.Ambiguities)
		}
	})

	t.Run("equal live candidates stay ambiguous", func(t *testing.T) {
		resolved := ResolveBindings(inventory, []BindingInput{{
			ScopeKey: "scope", Tag: "work", Agent: AgentClaude, LaunchPresent: true,
			LiveRounds: []RoundObservation{
				{RootNodeID: "root-a", PairPositions: []uint64{10}},
				{RootNodeID: "root-b", PairPositions: []uint64{10}},
			},
		}})
		binding := onlyBinding(t, resolved)
		if binding.Status != BindingAmbiguous || binding.RootNodeID != nil || len(binding.Candidates) != 2 || len(resolved.Ambiguities) != 1 {
			t.Fatalf("binding=%#v ambiguities=%#v", binding, resolved.Ambiguities)
		}
	})

	t.Run("global owner exclusivity resolves fixed point", func(t *testing.T) {
		inputs := []BindingInput{
			{ScopeKey: "scope-a", Tag: "work", Agent: AgentClaude, LaunchPresent: true, LedgerRootNodeID: "root-a"},
			{ScopeKey: "scope-b", Tag: "work", Agent: AgentClaude, LaunchPresent: true, LiveRounds: []RoundObservation{{RootNodeID: "root-a", PairPositions: []uint64{20}}, {RootNodeID: "root-b", PairPositions: []uint64{20}}}},
		}
		resolved := ResolveBindings(inventory, inputs)
		if len(resolved.Bindings) != 2 {
			t.Fatalf("bindings=%#v", resolved.Bindings)
		}
		roots := []string{*resolved.Bindings[0].RootNodeID, *resolved.Bindings[1].RootNodeID}
		slices.Sort(roots)
		if !slices.Equal(roots, []string{"root-a", "root-b"}) {
			t.Fatalf("roots=%v", roots)
		}
		if resolved.Bindings[0].Status != BindingEstablished || resolved.Bindings[1].Status != BindingProvisional {
			t.Fatalf("statuses=%v,%v", resolved.Bindings[0].Status, resolved.Bindings[1].Status)
		}
	})

	t.Run("equal-rank competing owners stay ambiguous", func(t *testing.T) {
		resolved := ResolveBindings(inventory, []BindingInput{
			{ScopeKey: "scope-a", Tag: "work", Agent: AgentClaude, LaunchPresent: true, LedgerRootNodeID: "root-a"},
			{ScopeKey: "scope-b", Tag: "work", Agent: AgentClaude, LaunchPresent: true, LedgerRootNodeID: "root-a"},
		})
		for _, binding := range resolved.Bindings {
			if binding.Status != BindingAmbiguous || binding.RootNodeID != nil {
				t.Fatalf("binding=%#v", binding)
			}
		}
		found := false
		for _, diagnostic := range resolved.Diagnostics {
			found = found || diagnostic.Code == DiagnosticBindingConflict
		}
		if !found {
			t.Fatalf("missing binding_conflict: %#v", resolved.Diagnostics)
		}
		if len(resolved.Ambiguities) != 1 || len(resolved.Ambiguities[0].BindingIDs) != 2 || !slices.Equal(resolved.Ambiguities[0].RootNodeIDs, []string{"root-a"}) {
			t.Fatalf("ambiguities=%#v", resolved.Ambiguities)
		}
	})
}

func TestResolveBindingsIsPermutationStable(t *testing.T) {
	t.Parallel()
	inventory := bindingTestInventory()
	inputs := []BindingInput{
		{ScopeKey: "b", Tag: "work", Agent: AgentClaude, LaunchPresent: true, ConfigRootNodeID: "root-b"},
		{ScopeKey: "a", Tag: "work", Agent: AgentClaude, LaunchPresent: true, LedgerRootNodeID: "root-a"},
	}
	want := ResolveBindings(inventory, inputs)
	slices.Reverse(inputs)
	inventory.Forests[0].Roots[0], inventory.Forests[0].Roots[1] = inventory.Forests[0].Roots[1], inventory.Forests[0].Roots[0]
	if got := ResolveBindings(inventory, inputs); !reflect.DeepEqual(got, want) {
		t.Fatalf("permutation changed result\ngot=%#v\nwant=%#v", got, want)
	}
}

func TestParentPropagationRequiresEstablishedRoot(t *testing.T) {
	t.Parallel()
	inventory := bindingTestInventory()
	established := Binding{StableID: "binding", Status: BindingEstablished, RootNodeID: ptrString("root-a")}
	propagated := PropagateBinding(established, inventory.Forests[0])
	if len(propagated) != 2 || propagated[0].NodeID != "child-a" || len(propagated[0].EdgeProvenance) != 1 || propagated[1].NodeID != "grandchild-a" {
		t.Fatalf("propagated=%#v", propagated)
	}
	provisional := established
	provisional.Status = BindingProvisional
	if got := PropagateBinding(provisional, inventory.Forests[0]); len(got) != 0 {
		t.Fatalf("provisional parentage strengthened binding: %#v", got)
	}
}

func bindingTestInventory() Inventory {
	childEdge := &ParentEdge{StableID: "edge-child", ParentID: "native-a", ChildID: "native-child", Provenance: []EdgeProvenance{{Schema: "fixture", Artifact: Artifact{StorageRoot: "root", RelativePath: "child", Kind: ArtifactTranscript}}}}
	grandchildEdge := &ParentEdge{StableID: "edge-grandchild", ParentID: "native-child", ChildID: "native-grandchild", Provenance: []EdgeProvenance{{Schema: "fixture", Artifact: Artifact{StorageRoot: "root", RelativePath: "grandchild", Kind: ArtifactTranscript}}}}
	return Inventory{Forests: []Forest{{Agent: AgentClaude, Roots: []Node{
		{StableID: "root-a", NativeID: "native-a", Role: RoleRoot, Children: []Node{{StableID: "child-a", NativeID: "native-child", Role: RoleSubagent, ParentEdge: childEdge, Children: []Node{{StableID: "grandchild-a", NativeID: "native-grandchild", Role: RoleSubagent, ParentEdge: grandchildEdge}}}}},
		{StableID: "root-b", NativeID: "native-b", Role: RoleRoot},
		{StableID: "root-c", NativeID: "native-c", Role: RoleRoot},
	}}}}
}

func onlyBinding(t *testing.T, inventory Inventory) Binding {
	t.Helper()
	if len(inventory.Bindings) != 1 {
		t.Fatalf("bindings=%#v", inventory.Bindings)
	}
	return inventory.Bindings[0]
}

func ptrString(value string) *string { return &value }
