package sessioninventory

import (
	"encoding/json"
	"slices"
	"testing"
	"time"
)

func TestBuildForest(t *testing.T) {
	t.Parallel()

	rootTime := nativeTime("2026-08-28T10:00:00Z")
	childTime := nativeTime("2026-08-28T10:01:00Z")
	missingParent := "missing"
	rootID := "root-1"
	facts := []Fact{
		{Agent: AgentCodex, NativeID: "orphan-1", Role: RoleSubagent, ParentID: &missingParent, Time: childTime, Artifacts: []Artifact{{StorageRoot: "codex", RelativePath: "sessions/orphan.jsonl"}}},
		{Agent: AgentCodex, NativeID: rootID, Role: RoleRoot, Time: rootTime, Artifacts: []Artifact{{StorageRoot: "codex", RelativePath: "sessions/root.jsonl"}}},
		{Agent: AgentCodex, NativeID: "child-1", Role: RoleSubagent, ParentID: &rootID, Time: childTime, Artifacts: []Artifact{{StorageRoot: "codex", RelativePath: "sessions/z-child.jsonl"}, {StorageRoot: "codex", RelativePath: "sessions/a-child.jsonl"}}},
		{Agent: AgentCodex, NativeID: rootID, Role: RoleRoot, Time: rootTime, Artifacts: []Artifact{{StorageRoot: "codex", RelativePath: "sessions/root.jsonl"}, {StorageRoot: "codex", RelativePath: "sessions/root-sidecar.json"}}},
	}

	got := BuildForest(facts)
	if len(got.Forests) != 1 {
		t.Fatalf("forests = %d, want 1", len(got.Forests))
	}
	forest := got.Forests[0]
	if forest.Agent != AgentCodex {
		t.Fatalf("agent = %q, want %q", forest.Agent, AgentCodex)
	}
	if len(forest.Roots) != 1 || forest.Roots[0].NativeID != rootID {
		t.Fatalf("roots = %#v, want root-1", forest.Roots)
	}
	root := forest.Roots[0]
	if len(root.Children) != 1 || root.Children[0].NativeID != "child-1" {
		t.Fatalf("children = %#v, want child-1", root.Children)
	}
	wantArtifacts := []Artifact{
		{StorageRoot: "codex", RelativePath: "sessions/root-sidecar.json"},
		{StorageRoot: "codex", RelativePath: "sessions/root.jsonl"},
	}
	if !slices.Equal(root.Artifacts, wantArtifacts) {
		t.Fatalf("root artifacts = %#v, want %#v", root.Artifacts, wantArtifacts)
	}
	if len(forest.Orphans) != 1 || forest.Orphans[0].NativeID != "orphan-1" {
		t.Fatalf("orphans = %#v, want orphan-1", forest.Orphans)
	}
	if !hasDiagnostic(got.Diagnostics, DiagnosticParentMissing, "orphan-1") {
		t.Fatalf("diagnostics = %#v, want parent_missing for orphan-1", got.Diagnostics)
	}
	if root.StableID != StableID("node", string(AgentCodex), rootID) {
		t.Fatalf("stable id = %q, want ID derived from agent and native ID", root.StableID)
	}
}

func TestBuildForestFailsClosedOnConflictingParentFacts(t *testing.T) {
	t.Parallel()

	rootA, rootB := "root-a", "root-b"
	facts := []Fact{
		{Agent: AgentClaude, NativeID: rootA, Role: RoleRoot},
		{Agent: AgentClaude, NativeID: rootB, Role: RoleRoot},
		{Agent: AgentClaude, NativeID: "child", Role: RoleSubagent, ParentID: &rootA},
		{Agent: AgentClaude, NativeID: "child", Role: RoleSubagent, ParentID: &rootB},
	}

	got := BuildForest(facts)
	forest := got.Forests[0]
	if len(forest.Roots[0].Children)+len(forest.Roots[1].Children) != 0 {
		t.Fatalf("conflicting child was attached: %#v", forest.Roots)
	}
	if len(forest.Orphans) != 1 || forest.Orphans[0].NativeID != "child" || forest.Orphans[0].ParentID != nil {
		t.Fatalf("conflicting child = %#v, want detached orphan", forest.Orphans)
	}
	if !hasDiagnostic(got.Diagnostics, DiagnosticParentConflict, "child") {
		t.Fatalf("diagnostics = %#v, want parent_conflict", got.Diagnostics)
	}
}

func TestBuildForestFailsClosedOnParentCycle(t *testing.T) {
	t.Parallel()

	a, b := "a", "b"
	got := BuildForest([]Fact{
		{Agent: AgentMuse, NativeID: a, Role: RoleSubagent, ParentID: &b},
		{Agent: AgentMuse, NativeID: b, Role: RoleSubagent, ParentID: &a},
	})

	if len(got.Forests) != 1 || len(got.Forests[0].Roots) != 0 || len(got.Forests[0].Orphans) != 2 {
		t.Fatalf("cycle result = %#v, want two detached orphans", got.Forests)
	}
	for _, nativeID := range []string{a, b} {
		if !hasDiagnostic(got.Diagnostics, DiagnosticParentConflict, nativeID) {
			t.Fatalf("diagnostics = %#v, want parent_conflict for %s", got.Diagnostics, nativeID)
		}
	}
}

func TestBuildForestIsPermutationInvariant(t *testing.T) {
	t.Parallel()

	rootID := "root"
	facts := []Fact{
		{Agent: AgentMuse, NativeID: rootID, Role: RoleRoot, Artifacts: []Artifact{{StorageRoot: "muse", RelativePath: "z"}}},
		{Agent: AgentMuse, NativeID: "child", Role: RoleSubagent, ParentID: &rootID, Artifacts: []Artifact{{StorageRoot: "muse", RelativePath: "b"}}},
		{Agent: AgentMuse, NativeID: rootID, Role: RoleRoot, Artifacts: []Artifact{{StorageRoot: "muse", RelativePath: "a"}}},
	}
	want, err := json.Marshal(BuildForest(facts))
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < len(facts); i++ {
		permuted := append([]Fact(nil), facts...)
		permuted[0], permuted[i] = permuted[i], permuted[0]
		got, err := json.Marshal(BuildForest(permuted))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != string(want) {
			t.Fatalf("permutation %d changed result\n got: %s\nwant: %s", i, got, want)
		}
	}
}

func FuzzBuildForestPermutation(f *testing.F) {
	f.Add([]byte{0, 1, 2, 3, 4, 5})
	f.Add([]byte{9, 9, 9, 9})
	f.Fuzz(func(t *testing.T, input []byte) {
		if len(input) > 32 {
			input = input[:32]
		}
		agents := []Agent{AgentClaude, AgentCodex, AgentAgy, AgentMuse}
		roles := []Role{RoleRoot, RoleSubagent}
		ids := []string{"a", "b", "c", "d"}
		facts := make([]Fact, 0, len(input))
		for i, value := range input {
			id := ids[int(value)%len(ids)]
			var parent *string
			if value&1 != 0 {
				parentID := ids[int(value>>1)%len(ids)]
				parent = &parentID
			}
			facts = append(facts, Fact{
				Agent:    agents[int(value>>2)%len(agents)],
				NativeID: id,
				Role:     roles[int(value>>4)%len(roles)],
				ParentID: parent,
				Artifacts: []Artifact{{
					StorageRoot:  "native",
					RelativePath: "sessions/" + ids[i%len(ids)],
				}},
			})
		}
		forward, err := json.Marshal(BuildForest(facts))
		if err != nil {
			t.Fatal(err)
		}
		slices.Reverse(facts)
		reversed, err := json.Marshal(BuildForest(facts))
		if err != nil {
			t.Fatal(err)
		}
		if string(forward) != string(reversed) {
			t.Fatalf("fact order changed canonical inventory\nforward: %s\nreverse: %s", forward, reversed)
		}
	})
}

func nativeTime(value string) *NativeTime {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		panic(err)
	}
	return &NativeTime{Value: parsed, Source: TimeSourceMetadata}
}

func hasDiagnostic(diagnostics []Diagnostic, code DiagnosticCode, nativeID string) bool {
	return slices.ContainsFunc(diagnostics, func(d Diagnostic) bool {
		return d.Code == code && d.NativeID != nil && *d.NativeID == nativeID
	})
}
