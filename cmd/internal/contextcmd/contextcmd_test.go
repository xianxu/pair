package contextcmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
)

func TestRunUsesEstablishedInventoryRoot(t *testing.T) {
	t.Parallel()
	runtime := contextRuntime(t, true)
	var stdout bytes.Buffer
	code := RunWithRuntime([]string{"T", "codex"}, Env{PairScopeKey: "scope"}, runtime, &stdout)
	if code != 0 || strings.TrimSpace(stdout.String()) != "60" {
		t.Fatalf("code=%d stdout=%q", code, stdout.String())
	}
}

func TestRunProvisionalBindingPrintsNothing(t *testing.T) {
	t.Parallel()
	runtime := contextRuntime(t, false)
	var stdout bytes.Buffer
	if code := RunWithRuntime([]string{"T", "codex"}, Env{PairScopeKey: "scope"}, runtime, &stdout); code != 0 || stdout.Len() != 0 {
		t.Fatalf("code=%d stdout=%q", code, stdout.String())
	}
}

func contextRuntime(t *testing.T, established bool) *sessioninventorytest.FakeRuntime {
	t.Helper()
	const nativeID = "019d1111-1111-7111-8111-111111111111"
	runtime := sessioninventorytest.NewFakeRuntime()
	nativeRoot := sessioninventory.StorageRoot{Agent: sessioninventory.AgentCodex, Name: "codex-sessions", Path: "/native/codex"}
	runtime.AddRoot(nativeRoot)
	transcript := sessioninventory.Artifact{StorageRoot: nativeRoot.Name, RelativePath: "2026/08/28/rollout-test-" + nativeID + ".jsonl"}
	content := []byte(
		`{"timestamp":"2026-08-28T10:04:00Z","type":"session_meta","payload":{"id":"` + nativeID + `","parent_thread_id":null,"source":"cli"}}` + "\n" +
			`{"type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":60}}}}` + "\n")
	runtime.PutFile(sessioninventory.FileEntry{Artifact: transcript, StableFileID: "stable", GenerationToken: "gen", MutationToken: "mutation"}, content)
	pairRoot := sessioninventory.StorageRoot{Name: "pair-data", Path: "/pair/scope"}
	runtime.SetPairDataRoot(pairRoot)
	launch, err := sessionledger.EncodeRecord(sessionledger.Record{Version: 2, Kind: sessionledger.RecordLaunch, ScopeKey: "scope", Tag: "T", Agent: "codex", LaunchArtifactBoundaries: []sessionledger.LaunchArtifactBoundary{}})
	if err != nil {
		t.Fatal(err)
	}
	ledger := append(launch, '\n')
	if established {
		state, _ := json.Marshal(sessioninventory.ScannerState{Version: 1, Agent: sessioninventory.AgentCodex, NativeID: nativeID, IdentityAnchor: nativeID, Role: sessioninventory.RoleRoot, ScannerSchema: "codex-v1", FirstRecordValidated: true})
		proof := sessionledger.AuthorizationProof{Version: 1, RootNativeID: nativeID, ScannerSchema: "codex-v1", ScannerState: state, Artifacts: []sessionledger.ArtifactProof{{StorageRoot: transcript.StorageRoot, RelativePath: transcript.RelativePath, StableFileID: "stable", GenerationToken: "gen", MutationToken: "mutation", Size: int64(len(content)), ParserCompleteOffset: int64(len(content))}}}
		binding, err := sessionledger.EncodeRecord(sessionledger.Record{Version: 2, Kind: sessionledger.RecordBinding, ScopeKey: "scope", Tag: "T", Agent: "codex", LaunchOrdinal: 1, RootNativeID: nativeID, AuthorizationProof: &proof})
		if err != nil {
			t.Fatal(err)
		}
		ledger = append(ledger, append(binding, '\n')...)
	}
	runtime.PutFile(sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: pairRoot.Name, RelativePath: "ledger-T.jsonl"}}, ledger)
	return runtime
}
