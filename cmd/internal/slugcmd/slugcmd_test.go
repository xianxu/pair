package slugcmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
)

func TestInventoryTurnsNeverFallsBackFromProvisional(t *testing.T) {
	t.Parallel()
	runtime := slugRuntime(t, false)
	turns, status, err := inventoryTurns(runtime, "scope", "testtag", sessioninventory.AgentCodex)
	if err != nil || status != sessioninventory.BindingProvisional || turns != nil {
		t.Fatalf("turns=%#v status=%s err=%v", turns, status, err)
	}
}

func TestInventoryTurnsUsesEstablishedRoot(t *testing.T) {
	t.Parallel()
	runtime := slugRuntime(t, true)
	turns, status, err := inventoryTurns(runtime, "scope", "testtag", sessioninventory.AgentCodex)
	if err != nil || status != sessioninventory.BindingEstablished || len(turns) != 2 {
		t.Fatalf("turns=%#v status=%s err=%v", turns, status, err)
	}
}

func TestInventoryTurnsStreamsLongEstablishedRoot(t *testing.T) {
	const nativeID = "019e8178-79c2-7862-91db-e8fa1be3b162"
	padding := []byte(`{"type":"session_meta","padding":"` + strings.Repeat("x", 1<<20) + `"}` + "\n")
	content := []byte(`{"timestamp":"2026-05-31T21:36:56Z","type":"session_meta","payload":{"id":"` + nativeID + `","parent_thread_id":null,"source":"cli"}}` + "\n")
	content = append(content, bytes.Repeat(padding, 33)...)
	content = append(content, []byte(
		`{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"late prompt"}]}}`+"\n"+
			`{"type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"late reply"}]}}`+"\n")...)
	runtime := slugRuntimeContent(t, true, content)

	turns, status, err := inventoryTurns(runtime, "scope", "testtag", sessioninventory.AgentCodex)
	if err != nil || status != sessioninventory.BindingEstablished || len(turns) != 2 || turns[0].Text != "late prompt" || turns[1].Text != "late reply" {
		t.Fatalf("turns=%#v status=%s err=%v", turns, status, err)
	}
}

func slugRuntime(t *testing.T, established bool) *sessioninventorytest.FakeRuntime {
	t.Helper()
	const nativeID = "019e8178-79c2-7862-91db-e8fa1be3b162"
	content := []byte(
		`{"timestamp":"2026-05-31T21:36:56Z","type":"session_meta","payload":{"id":"` + nativeID + `","parent_thread_id":null,"source":"cli"}}` + "\n" +
			`{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"prompt"}]}}` + "\n" +
			`{"type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"reply"}]}}` + "\n")
	return slugRuntimeContent(t, established, content)
}

func slugRuntimeContent(t *testing.T, established bool, content []byte) *sessioninventorytest.FakeRuntime {
	t.Helper()
	const nativeID = "019e8178-79c2-7862-91db-e8fa1be3b162"
	runtime := sessioninventorytest.NewFakeRuntime()
	nativeRoot := sessioninventory.StorageRoot{Agent: sessioninventory.AgentCodex, Name: "codex-sessions", Path: "/native/codex"}
	runtime.AddRoot(nativeRoot)
	artifact := sessioninventory.Artifact{StorageRoot: nativeRoot.Name, RelativePath: "2026/05/31/rollout-test-" + nativeID + ".jsonl"}
	runtime.PutFile(sessioninventory.FileEntry{Artifact: artifact, StableFileID: "stable", GenerationToken: "gen", MutationToken: "mutation"}, content)
	pairRoot := sessioninventory.StorageRoot{Name: "pair-data", Path: "/pair/scope"}
	runtime.SetPairDataRoot(pairRoot)
	launch, err := sessionledger.EncodeRecord(sessionledger.Record{Version: 2, Kind: sessionledger.RecordLaunch, ScopeKey: "scope", Tag: "testtag", Agent: "codex", LaunchArtifactBoundaries: []sessionledger.LaunchArtifactBoundary{}})
	if err != nil {
		t.Fatal(err)
	}
	ledger := append(launch, '\n')
	if established {
		state, _ := json.Marshal(sessioninventory.ScannerState{Version: 1, Agent: sessioninventory.AgentCodex, NativeID: nativeID, IdentityAnchor: nativeID, Role: sessioninventory.RoleRoot, ScannerSchema: "codex-v1", FirstRecordValidated: true})
		proof := sessionledger.AuthorizationProof{Version: 1, RootNativeID: nativeID, ScannerSchema: "codex-v1", ScannerState: state, Artifacts: []sessionledger.ArtifactProof{{StorageRoot: artifact.StorageRoot, RelativePath: artifact.RelativePath, StableFileID: "stable", GenerationToken: "gen", MutationToken: "mutation", Size: int64(len(content)), ParserCompleteOffset: int64(len(content))}}}
		binding, err := sessionledger.EncodeRecord(sessionledger.Record{Version: 2, Kind: sessionledger.RecordBinding, ScopeKey: "scope", Tag: "testtag", Agent: "codex", LaunchOrdinal: 1, RootNativeID: nativeID, AuthorizationProof: &proof})
		if err != nil {
			t.Fatal(err)
		}
		ledger = append(ledger, append(binding, '\n')...)
	}
	runtime.PutFile(sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: pairRoot.Name, RelativePath: "ledger-testtag.jsonl"}}, ledger)
	return runtime
}
