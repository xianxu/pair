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
	want := fmt.Sprintf("{\"forests\":[{\"agent\":\"muse\",\"roots\":[{\"stable_id\":%q,\"agent\":\"muse\",\"native_id\":\"root\",\"role\":\"root\",\"parent_id\":null,\"time\":null,\"resumable\":true,\"artifacts\":[{\"storage_root\":\"muse-sessions\",\"relative_path\":\"2026/08/28/root/session.jsonl\",\"kind\":\"transcript\"}],\"children\":[]}],\"orphans\":[]}],\"diagnostics\":[]}\n", StableID("node", "muse", "root"))
	if string(got) != want {
		t.Fatalf("projection bytes\n got: %s\nwant: %s", got, want)
	}

	shuffled := Inventory{Diagnostics: append([]Diagnostic(nil), inventory.Diagnostics...), Forests: append([]Forest(nil), inventory.Forests...)}
	gotAgain, err := RenderForestProjection(shuffled)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotAgain) != string(got) {
		t.Fatalf("projection changed across equivalent input\nfirst: %s\nagain: %s", got, gotAgain)
	}
}
