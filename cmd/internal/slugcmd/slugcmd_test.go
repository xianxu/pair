package slugcmd

import (
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
)

func TestInventoryTranscriptNeverFallsBackFromProvisional(t *testing.T) {
	t.Parallel()
	runtime := slugRuntime(t, false)
	data, status, err := inventoryTranscript(runtime, "scope", "testtag", sessioninventory.AgentCodex)
	if err != nil || status != sessioninventory.BindingProvisional || data != nil {
		t.Fatalf("data=%q status=%s err=%v", data, status, err)
	}
}

func TestInventoryTranscriptUsesEstablishedRoot(t *testing.T) {
	t.Parallel()
	runtime := slugRuntime(t, true)
	data, status, err := inventoryTranscript(runtime, "scope", "testtag", sessioninventory.AgentCodex)
	if err != nil || status != sessioninventory.BindingEstablished || len(data) == 0 {
		t.Fatalf("data=%q status=%s err=%v", data, status, err)
	}
}

func slugRuntime(t *testing.T, established bool) *sessioninventorytest.FakeRuntime {
	t.Helper()
	const nativeID = "019e8178-79c2-7862-91db-e8fa1be3b162"
	runtime := sessioninventorytest.NewFakeRuntime()
	nativeRoot := sessioninventory.StorageRoot{Agent: sessioninventory.AgentCodex, Name: "codex-sessions", Path: "/native/codex"}
	runtime.AddRoot(nativeRoot)
	artifact := sessioninventory.Artifact{StorageRoot: nativeRoot.Name, RelativePath: "2026/05/31/rollout-test-" + nativeID + ".jsonl"}
	runtime.PutFile(sessioninventory.FileEntry{Artifact: artifact}, []byte(
		`{"timestamp":"2026-05-31T21:36:56Z","type":"session_meta","payload":{"id":"`+nativeID+`","parent_thread_id":null,"source":"cli"}}`+"\n"))
	pairRoot := sessioninventory.StorageRoot{Name: "pair-data", Path: "/pair/scope"}
	runtime.SetPairDataRoot(pairRoot)
	ledger := `{"v":1,"kind":"launch","scope_key":"scope","tag":"testtag","agent":"codex","pair_log_offset":0,"native_watermarks":[]}` + "\n"
	if established {
		ledger += `{"v":1,"kind":"binding","scope_key":"scope","tag":"testtag","agent":"codex","launch_ordinal":1,"root_native_id":"` + nativeID + `"}` + "\n"
	}
	runtime.PutFile(sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: pairRoot.Name, RelativePath: "ledger-testtag.jsonl"}}, []byte(ledger))
	return runtime
}
