package slugcmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
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
	runtime := slugRuntime(t, true)
	const nativeID = "019e8178-79c2-7862-91db-e8fa1be3b162"
	artifact := sessioninventory.Artifact{StorageRoot: "codex-sessions", RelativePath: "2026/05/31/rollout-test-" + nativeID + ".jsonl"}
	padding := []byte(`{"type":"session_meta","padding":"` + strings.Repeat("x", 1<<20) + `"}` + "\n")
	content := []byte(`{"timestamp":"2026-05-31T21:36:56Z","type":"session_meta","payload":{"id":"` + nativeID + `","parent_thread_id":null,"source":"cli"}}` + "\n")
	content = append(content, bytes.Repeat(padding, 33)...)
	content = append(content, []byte(
		`{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"late prompt"}]}}`+"\n"+
			`{"type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"late reply"}]}}`+"\n")...)
	runtime.PutFile(sessioninventory.FileEntry{Artifact: artifact}, content)

	turns, status, err := inventoryTurns(runtime, "scope", "testtag", sessioninventory.AgentCodex)
	if err != nil || status != sessioninventory.BindingEstablished || len(turns) != 2 || turns[0].Text != "late prompt" || turns[1].Text != "late reply" {
		t.Fatalf("turns=%#v status=%s err=%v", turns, status, err)
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
		`{"timestamp":"2026-05-31T21:36:56Z","type":"session_meta","payload":{"id":"`+nativeID+`","parent_thread_id":null,"source":"cli"}}`+"\n"+
			`{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"prompt"}]}}`+"\n"+
			`{"type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"reply"}]}}`+"\n"))
	pairRoot := sessioninventory.StorageRoot{Name: "pair-data", Path: "/pair/scope"}
	runtime.SetPairDataRoot(pairRoot)
	ledger := `{"v":1,"kind":"launch","scope_key":"scope","tag":"testtag","agent":"codex","pair_log_offset":0,"native_watermarks":[]}` + "\n"
	if established {
		ledger += `{"v":1,"kind":"binding","scope_key":"scope","tag":"testtag","agent":"codex","launch_ordinal":1,"root_native_id":"` + nativeID + `"}` + "\n"
	}
	runtime.PutFile(sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: pairRoot.Name, RelativePath: "ledger-testtag.jsonl"}}, []byte(ledger))
	return runtime
}
