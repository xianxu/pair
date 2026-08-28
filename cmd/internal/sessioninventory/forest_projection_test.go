package sessioninventory

import (
	"fmt"
	"testing"
)

func TestForestProjectionUsesCanonicalBytes(t *testing.T) {
	t.Parallel()

	inventory := BuildForest([]Fact{{
		Agent:     AgentMuse,
		NativeID:  "root",
		Role:      RoleRoot,
		Resumable: true,
		Artifacts: []Artifact{{StorageRoot: "muse-sessions", RelativePath: "2026/08/28/root/session.jsonl", Kind: ArtifactTranscript}},
	}})
	got, err := RenderForestProjection(inventory)
	if err != nil {
		t.Fatal(err)
	}
	want := fmt.Sprintf("{\"forests\":[{\"agent\":\"muse\",\"roots\":[{\"stable_id\":%q,\"agent\":\"muse\",\"native_id\":\"root\",\"role\":\"root\",\"parent_id\":null,\"time\":null,\"resumable\":true,\"artifacts\":[{\"storage_root\":\"muse-sessions\",\"relative_path\":\"2026/08/28/root/session.jsonl\",\"kind\":\"transcript\"}],\"parent_edge\":null,\"children\":[]}],\"orphans\":[]}],\"diagnostics\":[]}\n", StableID("node", "muse", "root"))
	if string(got) != want {
		t.Fatalf("projection bytes\n got: %s\nwant: %s", got, want)
	}

	other := BuildForest([]Fact{{Agent: AgentClaude, NativeID: "z", Role: RoleRoot}, {Agent: AgentClaude, NativeID: "a", Role: RoleRoot}})
	combined := Inventory{Forests: append(append([]Forest(nil), inventory.Forests...), other.Forests...), Diagnostics: []Diagnostic{
		diagnostic(DiagnosticStorageAbsent, AgentMuse, nil, "z"),
		diagnostic(DiagnosticNodeMalformed, AgentClaude, nil, "a"),
	}}
	canonical, err := RenderForestProjection(combined)
	if err != nil {
		t.Fatal(err)
	}
	shuffled := Inventory{Forests: []Forest{combined.Forests[1], combined.Forests[0]}, Diagnostics: []Diagnostic{combined.Diagnostics[1], combined.Diagnostics[0]}}
	shuffled.Forests[0].Roots[0], shuffled.Forests[0].Roots[1] = shuffled.Forests[0].Roots[1], shuffled.Forests[0].Roots[0]
	gotAgain, err := RenderForestProjection(shuffled)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotAgain) != string(canonical) {
		t.Fatalf("projection changed across reordered forests, roots, and diagnostics\nfirst: %s\nagain: %s", canonical, gotAgain)
	}
}
