package sessioninventory_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
)

func TestScanQoderV1(t *testing.T) {
	t.Parallel()

	runtime := sessioninventorytest.NewFakeRuntime()
	loadNativeFixture(t, runtime, sessioninventory.AgentQoder, "qoder-projects", filepath.Join("testdata", "native", "qoder", "v1", "qoder-projects"))
	got := inventoryFromScan(sessioninventory.ScanQoder(runtime))

	if len(got.Forests) != 1 || len(got.Forests[0].Roots) != 1 {
		t.Fatalf("forests = %#v", got.Forests)
	}
	root := got.Forests[0].Roots[0]
	if root.NativeID != "11111111-1111-4111-8111-111111111111" || !root.Resumable {
		t.Fatalf("root = %#v", root)
	}
	if root.Time == nil || root.Time.Source != sessioninventory.TimeSourceMetadata || !root.Time.Value.Equal(time.Date(2026, 8, 28, 9, 0, 30, 0, time.UTC)) {
		t.Fatalf("root chronology = %#v, want the runtime-config epoch-millis instant", root.Time)
	}
	if len(root.Children) != 1 || root.Children[0].NativeID != "aExplore-2222333344445555" || root.Children[0].ParentID == nil || *root.Children[0].ParentID != root.NativeID || root.Children[0].Resumable {
		t.Fatalf("children = %#v", root.Children)
	}
	if len(got.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v, want qoder bookkeeping records to scan clean", got.Diagnostics)
	}
}

func TestValidateQoderDelta(t *testing.T) {
	t.Parallel()
	entry := sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: "qoder-projects", RelativePath: "-repo/11111111-1111-4111-8111-111111111111.jsonl"}}
	first := []sessioninventory.FramedJSONLRecord{{Bytes: []byte(`{"type":"workspace-directories","sessionId":"11111111-1111-4111-8111-111111111111","directories":["/repo"]}`)}}
	state, _, err := sessioninventory.ValidateQoderDelta(entry, nil, first)
	if err != nil || state.Disputed || !state.FirstRecordValidated {
		t.Fatalf("state=%#v err=%v", state, err)
	}
	// A record naming a different session disputes the file.
	state, diagnostics, err := sessioninventory.ValidateQoderDelta(entry, &state, []sessioninventory.FramedJSONLRecord{{Bytes: []byte(`{"type":"user","sessionId":"99999999-9999-4999-8999-999999999999","message":{"role":"user","content":"x"}}`)}})
	if err != nil || !state.Disputed || !diagnosticPresent(diagnostics, sessioninventory.DiagnosticNodeMalformed) {
		t.Fatalf("state=%#v diagnostics=%#v", state, diagnostics)
	}
}

func TestIncrementalQoderMalformedSuffixFailsClosed(t *testing.T) {
	t.Parallel()
	entry := sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: "qoder-projects", RelativePath: "-repo/11111111-1111-4111-8111-111111111111.jsonl"}}
	state, diagnostics, err := sessioninventory.ValidateQoderDelta(entry, nil, []sessioninventory.FramedJSONLRecord{{Bytes: []byte(`{"sessionId":`)}})
	if err != nil || !state.Disputed || !diagnosticPresent(diagnostics, sessioninventory.DiagnosticSchemaNearMiss) {
		t.Fatalf("state=%#v diagnostics=%#v err=%v", state, diagnostics, err)
	}
}
