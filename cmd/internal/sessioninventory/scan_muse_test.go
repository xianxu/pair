package sessioninventory_test

import (
	"path/filepath"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
)

func TestScanMuseV1(t *testing.T) {
	t.Parallel()

	runtime := sessioninventorytest.NewFakeRuntime()
	loadNativeFixture(t, runtime, sessioninventory.AgentMuse, "muse-sessions", filepath.Join("testdata", "native", "muse", "v1", "muse-sessions"))
	got := inventoryFromScan(sessioninventory.ScanMuse(runtime))

	if len(got.Forests) != 1 || len(got.Forests[0].Roots) != 1 {
		t.Fatalf("forests = %#v", got.Forests)
	}
	root := got.Forests[0].Roots[0]
	if root.NativeID != "77777777-7777-4777-8777-777777777777" || !root.Resumable || root.Time == nil || root.Time.Source != sessioninventory.TimeSourceBirth {
		t.Fatalf("root = %#v", root)
	}
	if len(root.Children) != 1 || root.Children[0].NativeID != "88888888-8888-4888-8888-888888888888" || root.Children[0].ParentID == nil || *root.Children[0].ParentID != root.NativeID || root.Children[0].Resumable {
		t.Fatalf("children = %#v", root.Children)
	}
	if len(got.Forests[0].Orphans) != 1 || got.Forests[0].Orphans[0].NativeID != "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa" || got.Forests[0].Orphans[0].Role != sessioninventory.RoleUnknown {
		t.Fatalf("orphans = %#v, want later contradiction retained unbound", got.Forests[0].Orphans)
	}
	if !diagnosticPresent(got.Diagnostics, sessioninventory.DiagnosticSchemaNearMiss) {
		t.Fatalf("diagnostics = %#v, want schema_near_miss", got.Diagnostics)
	}
}
