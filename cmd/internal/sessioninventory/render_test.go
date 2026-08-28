package sessioninventory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRenderV1IsStableAndUsesExplicitNulls(t *testing.T) {
	t.Parallel()
	created := time.Date(2026, 8, 28, 12, 0, 0, 123, time.UTC)
	rootID := "node-root"
	input := Inventory{
		Forests: []Forest{{Agent: AgentCodex, Roots: []Node{{
			StableID: rootID, Agent: AgentCodex, NativeID: "native-root", Role: RoleRoot,
			Time: &NativeTime{Value: created, Source: TimeSourceMetadata}, Resumable: true,
			Artifacts: []Artifact{{StorageRoot: "codex-sessions", RelativePath: "2026/08/28/root.jsonl", Kind: ArtifactTranscript}},
		}}}},
		Bindings:    []Binding{{StableID: "binding-a", ScopeKey: "scope", Tag: "work", Agent: AgentCodex, RootNodeID: &rootID, Status: BindingEstablished}},
		Diagnostics: []Diagnostic{{Code: DiagnosticStorageAbsent, Agent: AgentClaude, Detail: "private detail"}},
	}
	jsonOutput, err := RenderV1(input, RenderJSON)
	if err != nil {
		t.Fatal(err)
	}
	wantFragments := []string{
		`"schema_version":1`, `"correlations":[{"binding_id":"binding-a"`,
		`"parent_native_id":null`, `"created_at":"2026-08-28T12:00:00.000000123Z"`,
		`"diagnostic_id":`, `"native_id":null`, `"storage_root":null`, `"relative_path":null`,
	}
	for _, fragment := range wantFragments {
		if !strings.Contains(string(jsonOutput), fragment) {
			t.Fatalf("JSON missing %s:\n%s", fragment, jsonOutput)
		}
	}
	if strings.Contains(string(jsonOutput), "private detail") {
		t.Fatalf("public schema leaked diagnostic detail: %s", jsonOutput)
	}

	humanA, err := RenderV1(input, RenderHuman)
	if err != nil {
		t.Fatal(err)
	}
	humanB, _ := RenderV1(Inventory{Diagnostics: input.Diagnostics, Bindings: input.Bindings, Forests: input.Forests}, RenderHuman)
	if string(humanA) != string(humanB) || !strings.Contains(string(humanA), "codex roots=1 orphans=0") || !strings.HasSuffix(string(humanA), "\n") {
		t.Fatalf("unstable human rendering:\n%s\n---\n%s", humanA, humanB)
	}
}

func TestRenderV1EmptyGoldens(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name   string
		format RenderFormat
	}{
		{"inventory-empty.json", RenderJSON},
		{"inventory-empty.txt", RenderHuman},
	} {
		want, err := os.ReadFile(filepath.Join("testdata", "golden", test.name))
		if err != nil {
			t.Fatal(err)
		}
		got, err := RenderV1(Inventory{}, test.format)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != string(want) {
			t.Fatalf("%s mismatch\ngot:  %q\nwant: %q", test.name, got, want)
		}
	}
}
