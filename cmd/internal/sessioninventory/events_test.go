package sessioninventory_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
)

func longCodexTranscript(suffix string) []byte {
	padding := []byte(`{"type":"session_meta","padding":"` + strings.Repeat("x", 1<<20) + `"}` + "\n")
	return append(bytes.Repeat(padding, 33), []byte(suffix)...)
}

func TestNativeEventsWithRuntimeReadsOnlyAuthorizedRootTranscript(t *testing.T) {
	t.Parallel()
	runtime := sessioninventorytest.NewFakeRuntime()
	rootArtifact := sessioninventory.Artifact{StorageRoot: "codex-sessions", RelativePath: "2026/08/28/root.jsonl", Kind: sessioninventory.ArtifactTranscript}
	childArtifact := sessioninventory.Artifact{StorageRoot: "codex-sessions", RelativePath: "2026/08/28/child.jsonl", Kind: sessioninventory.ArtifactTranscript}
	runtime.AddRoot(sessioninventory.StorageRoot{Agent: sessioninventory.AgentCodex, Name: "codex-sessions", Path: "/native"})
	runtime.PutFile(sessioninventory.FileEntry{Artifact: rootArtifact}, []byte(
		`{"type":"session_meta","payload":{"id":"root"}}`+"\n"+
			`{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"please inspect the durable watcher boundary"}]}}`+"\n"+
			`{"type":"response_item","payload":{"type":"function_call"}}`+"\n"))
	runtime.PutFile(sessioninventory.FileEntry{Artifact: childArtifact}, []byte(
		`{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"generated child prompt"}]}}`+"\n"))
	inventory := sessioninventory.Inventory{Forests: []sessioninventory.Forest{{Agent: sessioninventory.AgentCodex, Roots: []sessioninventory.Node{{
		StableID: "root-node", NativeID: "root", Role: sessioninventory.RoleRoot, Artifacts: []sessioninventory.Artifact{rootArtifact},
		Children: []sessioninventory.Node{{StableID: "child-node", NativeID: "child", Role: sessioninventory.RoleSubagent, Artifacts: []sessioninventory.Artifact{childArtifact}}},
	}}}}}

	events, diagnostics := sessioninventory.NativeEventsWithRuntime(runtime, inventory, sessioninventory.AgentCodex)
	if len(diagnostics) != 0 || len(events) != 2 {
		t.Fatalf("events=%#v diagnostics=%#v", events, diagnostics)
	}
	if events[0].Agent != sessioninventory.AgentCodex || events[0].RootNodeID != "root-node" || events[0].Event.Kind != sessioninventory.EventOperator || events[1].Event.Kind != sessioninventory.EventToolCall {
		t.Fatalf("events=%#v", events)
	}
	if events[0].Position >= events[1].Position {
		t.Fatalf("positions are not ordered: %#v", events)
	}
}

func TestNativeEventsWithRuntimeRejectsMultipleRootTranscripts(t *testing.T) {
	t.Parallel()
	runtime := sessioninventorytest.NewFakeRuntime()
	first := sessioninventory.Artifact{StorageRoot: "codex-sessions", RelativePath: "a.jsonl", Kind: sessioninventory.ArtifactTranscript}
	second := sessioninventory.Artifact{StorageRoot: "codex-sessions", RelativePath: "b.jsonl", Kind: sessioninventory.ArtifactTranscript}
	inventory := sessioninventory.Inventory{Forests: []sessioninventory.Forest{{Agent: sessioninventory.AgentCodex, Roots: []sessioninventory.Node{{
		StableID: "root-node", NativeID: "root", Role: sessioninventory.RoleRoot, Artifacts: []sessioninventory.Artifact{first, second},
	}}}}}
	events, diagnostics := sessioninventory.NativeEventsWithRuntime(runtime, inventory, sessioninventory.AgentCodex)
	if len(events) != 0 || len(diagnostics) != 1 || diagnostics[0].Code != sessioninventory.DiagnosticTurnUnusable {
		t.Fatalf("events=%#v diagnostics=%#v", events, diagnostics)
	}
}

func TestNativeEventsWithRuntimeReadsRecordsAfterThirtyTwoMiB(t *testing.T) {
	runtime := sessioninventorytest.NewFakeRuntime()
	artifact := sessioninventory.Artifact{StorageRoot: "codex-sessions", RelativePath: "long.jsonl", Kind: sessioninventory.ArtifactTranscript}
	runtime.PutFile(sessioninventory.FileEntry{Artifact: artifact}, longCodexTranscript(
		`{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"late prompt"}]}}`+"\n"+
			`{"type":"response_item","payload":{"type":"function_call"}}`+"\n"))
	inventory := sessioninventory.Inventory{Forests: []sessioninventory.Forest{{Agent: sessioninventory.AgentCodex, Roots: []sessioninventory.Node{{
		StableID: "root-node", NativeID: "root", Role: sessioninventory.RoleRoot, Artifacts: []sessioninventory.Artifact{artifact},
	}}}}}

	events, diagnostics := sessioninventory.NativeEventsWithRuntime(runtime, inventory, sessioninventory.AgentCodex)
	if len(diagnostics) != 0 || len(events) != 2 || events[0].Event.Text != "late prompt" || events[1].Event.Kind != sessioninventory.EventToolCall {
		t.Fatalf("events=%#v diagnostics=%#v", events, diagnostics)
	}
}

func TestTextEventWindowForRootStreamsBoundedTailAfterThirtyTwoMiB(t *testing.T) {
	runtime := sessioninventorytest.NewFakeRuntime()
	artifact := sessioninventory.Artifact{StorageRoot: "codex-sessions", RelativePath: "long.jsonl", Kind: sessioninventory.ArtifactTranscript}
	runtime.PutFile(sessioninventory.FileEntry{Artifact: artifact}, longCodexTranscript(
		`{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"late prompt"}]}}`+"\n"+
			`{"type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"late reply"}]}}`+"\n"))
	root := sessioninventory.Node{Agent: sessioninventory.AgentCodex, StableID: "root-node", Artifacts: []sessioninventory.Artifact{artifact}}

	events, err := sessioninventory.TextEventWindowForRoot(runtime, root, 40)
	if err != nil || len(events) != 2 || events[0].Text != "late prompt" || events[1].Text != "late reply" {
		t.Fatalf("events=%#v err=%v", events, err)
	}
}

func TestTextEventWindowForRootRejectsUnterminatedFinalRecord(t *testing.T) {
	runtime := sessioninventorytest.NewFakeRuntime()
	artifact := sessioninventory.Artifact{StorageRoot: "codex-sessions", RelativePath: "truncated.jsonl", Kind: sessioninventory.ArtifactTranscript}
	runtime.PutFile(sessioninventory.FileEntry{Artifact: artifact}, []byte(
		`{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"complete"}]}}`+"\n"+
			`{"type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"unterminated"}]}}`))
	root := sessioninventory.Node{Agent: sessioninventory.AgentCodex, StableID: "root-node", Artifacts: []sessioninventory.Artifact{artifact}}

	events, err := sessioninventory.TextEventWindowForRoot(runtime, root, 40)
	if err == nil || events != nil {
		t.Fatalf("events=%#v err=%v", events, err)
	}
}
