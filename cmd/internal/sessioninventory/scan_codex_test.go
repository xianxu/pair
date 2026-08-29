package sessioninventory_test

import (
	"path/filepath"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
)

func TestScanCodexV1(t *testing.T) {
	t.Parallel()

	runtime := sessioninventorytest.NewFakeRuntime()
	loadNativeFixture(t, runtime, sessioninventory.AgentCodex, "codex-sessions", filepath.Join("testdata", "native", "codex", "v1", "codex-sessions"))
	got := inventoryFromScan(sessioninventory.ScanCodex(runtime))

	if len(got.Forests) != 1 || len(got.Forests[0].Roots) != 1 {
		t.Fatalf("forests = %#v", got.Forests)
	}
	root := got.Forests[0].Roots[0]
	if root.NativeID != "019d1111-1111-7111-8111-111111111111" || !root.Resumable || root.Time == nil || root.Time.Source != sessioninventory.TimeSourceMetadata {
		t.Fatalf("root = %#v", root)
	}
	if len(root.Children) != 2 || root.Children[0].ParentID == nil || root.Children[1].ParentID == nil {
		t.Fatalf("children = %#v, want current and legacy subagent forms", root.Children)
	}
	for _, child := range root.Children {
		if *child.ParentID != root.NativeID || child.Resumable || child.Artifacts[0].Kind != sessioninventory.ArtifactTranscript {
			t.Fatalf("child = %#v", child)
		}
	}
	if !diagnosticPresent(got.Diagnostics, sessioninventory.DiagnosticSchemaNearMiss) {
		t.Fatalf("diagnostics = %#v, want schema_near_miss", got.Diagnostics)
	}
}

func TestScanCodexRetainsMetadataPathDisagreementUnbound(t *testing.T) {
	t.Parallel()

	const pathID = "019d5555-5555-7555-8555-555555555555"
	runtime := codexRuntimeWithRecord(t, pathID, `{"timestamp":"2026-08-28T10:04:00Z","type":"session_meta","payload":{"id":"019d6666-6666-7666-8666-666666666666","source":"cli"}}`)
	got := inventoryFromScan(sessioninventory.ScanCodex(runtime))

	if len(got.Forests) != 1 || len(got.Forests[0].Roots) != 0 || len(got.Forests[0].Orphans) != 1 {
		t.Fatalf("forests = %#v, want one unbound orphan", got.Forests)
	}
	orphan := got.Forests[0].Orphans[0]
	if orphan.NativeID != pathID || orphan.Role != sessioninventory.RoleUnknown || orphan.Resumable {
		t.Fatalf("orphan = %#v, want disputed path-owned node", orphan)
	}
	if !diagnosticPresent(got.Diagnostics, sessioninventory.DiagnosticParentConflict) {
		t.Fatalf("diagnostics = %#v, want parent_conflict", got.Diagnostics)
	}
}

func TestScanCodexRejectsUnallowlistedChildSources(t *testing.T) {
	t.Parallel()

	const childID = "019d7777-7777-7777-8777-777777777777"
	cases := map[string]string{
		"nested parent mismatch": `{"timestamp":"2026-08-28T10:05:00Z","type":"session_meta","payload":{"id":"019d7777-7777-7777-8777-777777777777","parent_thread_id":"019d1111-1111-7111-8111-111111111111","source":{"subagent":{"thread_spawn":{"agent_nickname":"worker","agent_path":null,"agent_role":"explorer","depth":1,"parent_thread_id":"019d8888-8888-7888-8888-888888888888"}}}}}`,
		"unknown sibling key":    `{"timestamp":"2026-08-28T10:05:00Z","type":"session_meta","payload":{"id":"019d7777-7777-7777-8777-777777777777","parent_thread_id":"019d1111-1111-7111-8111-111111111111","source":{"subagent":{"thread_spawn":{"agent_nickname":"worker","agent_path":null,"agent_role":"explorer","depth":1,"parent_thread_id":"019d1111-1111-7111-8111-111111111111","future":true}}}}}`,
		"string subagent":        `{"timestamp":"2026-08-28T10:05:00Z","type":"session_meta","payload":{"id":"019d7777-7777-7777-8777-777777777777","parent_thread_id":"019d1111-1111-7111-8111-111111111111","source":"subagent"}}`,
	}
	for name, record := range cases {
		record := record
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := sessioninventory.ScanCodex(codexRuntimeWithRecord(t, childID, record))
			if len(got.Facts) != 0 {
				t.Fatalf("facts = %#v, want none", got.Facts)
			}
			if !diagnosticPresent(got.Diagnostics, sessioninventory.DiagnosticSchemaNearMiss) {
				t.Fatalf("diagnostics = %#v, want schema_near_miss", got.Diagnostics)
			}
		})
	}
}

func codexRuntimeWithRecord(t *testing.T, nativeID, record string) *sessioninventorytest.FakeRuntime {
	t.Helper()
	runtime := sessioninventorytest.NewFakeRuntime()
	runtime.AddRoot(sessioninventory.StorageRoot{Agent: sessioninventory.AgentCodex, Name: "codex-sessions", Path: "unused"})
	runtime.PutFile(sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{
		StorageRoot:  "codex-sessions",
		RelativePath: "2026/08/28/rollout-test-" + nativeID + ".jsonl",
	}}, []byte(record+"\n"))
	return runtime
}

func TestIncrementalCodexRequiresFirstSessionMetaAndDisputesConflicts(t *testing.T) {
	t.Parallel()
	nativeID := "019d1111-1111-7111-8111-111111111111"
	entry := sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: "codex-sessions", RelativePath: "2026/08/28/rollout-root-" + nativeID + ".jsonl"}}
	first := []sessioninventory.FramedJSONLRecord{{Bytes: []byte(`{"timestamp":"2026-08-28T09:00:00Z","type":"session_meta","payload":{"id":"` + nativeID + `","parent_thread_id":null,"source":"cli"}}`)}}
	state, diagnostics, err := sessioninventory.ValidateCodexDelta(entry, nil, first)
	if err != nil || len(diagnostics) != 0 || !state.FirstRecordValidated || state.Disputed || state.Role != sessioninventory.RoleRoot {
		t.Fatalf("state=%#v diagnostics=%#v err=%v", state, diagnostics, err)
	}
	prior := state
	conflict := []sessioninventory.FramedJSONLRecord{{Bytes: []byte(`{"type":"session_meta","payload":{"id":"019d9999-9999-7999-8999-999999999999","parent_thread_id":null,"source":"cli"}}`)}}
	state, diagnostics, err = sessioninventory.ValidateCodexDelta(entry, &prior, conflict)
	if err != nil || !state.Disputed || !diagnosticPresent(diagnostics, sessioninventory.DiagnosticParentConflict) || prior.Disputed {
		t.Fatalf("state=%#v diagnostics=%#v prior=%#v err=%v", state, diagnostics, prior, err)
	}
}

func TestIncrementalCodexRejectsMissingFirstSessionMeta(t *testing.T) {
	t.Parallel()
	nativeID := "019d1111-1111-7111-8111-111111111111"
	entry := sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: "codex-sessions", RelativePath: "2026/08/28/rollout-root-" + nativeID + ".jsonl"}}
	state, diagnostics, err := sessioninventory.ValidateCodexDelta(entry, nil, []sessioninventory.FramedJSONLRecord{{Bytes: []byte(`{"type":"event_msg","payload":{}}`)}})
	if err != nil || !state.Disputed || state.FirstRecordValidated || !diagnosticPresent(diagnostics, sessioninventory.DiagnosticSchemaNearMiss) {
		t.Fatalf("state=%#v diagnostics=%#v err=%v", state, diagnostics, err)
	}
}

func TestIncrementalCodexNeverRecoversFromInvalidFirstRecord(t *testing.T) {
	t.Parallel()
	nativeID := "019d1111-1111-7111-8111-111111111111"
	entry := sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: "codex-sessions", RelativePath: "2026/08/28/rollout-root-" + nativeID + ".jsonl"}}
	records := []sessioninventory.FramedJSONLRecord{
		{Bytes: []byte(`{"type":"event_msg","payload":{}}`)},
		{Bytes: []byte(`{"type":"session_meta","payload":{"id":"` + nativeID + `","parent_thread_id":null,"source":"cli"}}`)},
	}
	state, _, err := sessioninventory.ValidateCodexDelta(entry, nil, records)
	if err != nil || !state.Disputed || !state.FirstRecordValidated {
		t.Fatalf("state=%#v err=%v", state, err)
	}
}
