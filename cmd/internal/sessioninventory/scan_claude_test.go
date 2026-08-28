package sessioninventory_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
)

func TestScanClaudeV1(t *testing.T) {
	t.Parallel()

	runtime := sessioninventorytest.NewFakeRuntime()
	loadNativeFixture(t, runtime, sessioninventory.AgentClaude, "claude-projects", filepath.Join("testdata", "native", "claude", "v1", "claude-projects"))
	got := inventoryFromScan(sessioninventory.ScanClaude(runtime))

	if len(got.Forests) != 1 || len(got.Forests[0].Roots) != 1 {
		t.Fatalf("forests = %#v", got.Forests)
	}
	root := got.Forests[0].Roots[0]
	if root.NativeID != "11111111-1111-4111-8111-111111111111" || !root.Resumable || root.Time == nil || root.Time.Source != sessioninventory.TimeSourceMetadata {
		t.Fatalf("root = %#v", root)
	}
	if len(root.Children) != 1 || root.Children[0].NativeID != "worker-a" || root.Children[0].ParentID == nil || *root.Children[0].ParentID != root.NativeID || root.Children[0].Resumable {
		t.Fatalf("children = %#v", root.Children)
	}
	if len(got.Forests[0].Orphans) != 1 || got.Forests[0].Orphans[0].NativeID != "contradicted" || got.Forests[0].Orphans[0].Role != sessioninventory.RoleUnknown {
		t.Fatalf("orphans = %#v, want later contradiction retained unbound", got.Forests[0].Orphans)
	}
	if root.Artifacts[0].Kind != sessioninventory.ArtifactTranscript || root.Children[0].Artifacts[0].Kind != sessioninventory.ArtifactTranscript {
		t.Fatalf("artifact kinds = %#v / %#v", root.Artifacts, root.Children[0].Artifacts)
	}
	if !diagnosticPresent(got.Diagnostics, sessioninventory.DiagnosticSchemaNearMiss) {
		t.Fatalf("diagnostics = %#v, want schema_near_miss", got.Diagnostics)
	}
}

func TestScanClaudePreservesRegularFilesBesideRejectedSymlink(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	runtime := sessioninventory.NewOSRuntime(home, t.TempDir())
	root := runtime.NativeRoots(sessioninventory.AgentClaude)[0]
	project := filepath.Join(root.Path, "-repo")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}
	regular := filepath.Join(project, "11111111-1111-4111-8111-111111111111.jsonl")
	if err := os.WriteFile(regular, []byte(`{"type":"user","timestamp":"2026-08-28T09:01:00Z","sessionId":"11111111-1111-4111-8111-111111111111","isSidechain":false}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside.jsonl")
	if err := os.WriteFile(outside, []byte("private\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(project, "22222222-2222-4222-8222-222222222222.jsonl")); err != nil {
		t.Fatal(err)
	}

	got := sessioninventory.ScanClaude(runtime)
	if len(got.Facts) != 1 || got.Facts[0].NativeID != "11111111-1111-4111-8111-111111111111" {
		t.Fatalf("facts = %#v, want regular root preserved", got.Facts)
	}
	if !diagnosticPresent(got.Diagnostics, sessioninventory.DiagnosticNodeMalformed) {
		t.Fatalf("diagnostics = %#v, want rejected non-regular-node diagnostic", got.Diagnostics)
	}
}
