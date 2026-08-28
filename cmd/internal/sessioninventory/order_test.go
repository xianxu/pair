package sessioninventory

import (
	"slices"
	"strings"
	"testing"
)

func TestSortInventory(t *testing.T) {
	t.Parallel()

	present := nativeTime("2026-08-28T09:00:00Z")
	input := Inventory{
		Forests: []Forest{
			{Agent: AgentMuse, Roots: []Node{{Agent: AgentMuse, NativeID: "missing-time", Role: RoleRoot}, {Agent: AgentMuse, NativeID: "present-time", Role: RoleRoot, Time: present}}},
			{Agent: AgentClaude, Roots: []Node{{Agent: AgentClaude, NativeID: "b", Role: RoleRoot, Artifacts: []Artifact{{StorageRoot: "claude", RelativePath: "z"}, {StorageRoot: "claude", RelativePath: "a"}}}, {Agent: AgentClaude, NativeID: "a", Role: RoleRoot}}},
		},
		Diagnostics: []Diagnostic{
			{Code: DiagnosticParentMissing, Agent: AgentMuse, StableID: "z"},
			{Code: DiagnosticNodeMalformed, Agent: AgentClaude, StableID: "a"},
		},
	}

	got := SortInventory(input)
	if got.Forests[0].Agent != AgentClaude || got.Forests[1].Agent != AgentMuse {
		t.Fatalf("forest order = %#v", got.Forests)
	}
	if ids(got.Forests[0].Roots) != "a,b" {
		t.Fatalf("claude root order = %s, want a,b", ids(got.Forests[0].Roots))
	}
	if ids(got.Forests[1].Roots) != "present-time,missing-time" {
		t.Fatalf("time order = %s, want present-time,missing-time", ids(got.Forests[1].Roots))
	}
	wantArtifacts := []Artifact{{StorageRoot: "claude", RelativePath: "a"}, {StorageRoot: "claude", RelativePath: "z"}}
	if !slices.Equal(got.Forests[0].Roots[1].Artifacts, wantArtifacts) {
		t.Fatalf("artifact order = %#v, want %#v", got.Forests[0].Roots[1].Artifacts, wantArtifacts)
	}
	if input.Forests[0].Agent != AgentMuse {
		t.Fatal("SortInventory mutated its input")
	}
}

func TestStableIDUsesLengthPrefixedParts(t *testing.T) {
	t.Parallel()

	if StableID("test", "ab", "c") == StableID("test", "a", "bc") {
		t.Fatal("ambiguous concatenations produced the same stable ID")
	}
	if got, want := StableID("node", "claude", "root"), StableID("node", "claude", "root"); got != want {
		t.Fatalf("StableID is not deterministic: %q != %q", got, want)
	} else if !strings.HasPrefix(got, "node-") || len(got) != len("node-")+24 {
		t.Fatalf("StableID = %q, want kind plus 24 lowercase hex characters", got)
	}
}

func FuzzStableIDLengthPrefixes(f *testing.F) {
	f.Add("ab", "c")
	f.Add("", "value")
	f.Fuzz(func(t *testing.T, left, right string) {
		if left == right {
			t.Skip()
		}
		if StableID("test", left, right) == StableID("test", left+right) {
			t.Fatalf("parts %q, %q collided with their concatenation", left, right)
		}
	})
}

func ids(nodes []Node) string {
	var result string
	for i, node := range nodes {
		if i > 0 {
			result += ","
		}
		result += node.NativeID
	}
	return result
}
