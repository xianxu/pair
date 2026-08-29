package contextcmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
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
	runtime.PutFile(sessioninventory.FileEntry{Artifact: transcript}, []byte(
		`{"timestamp":"2026-08-28T10:04:00Z","type":"session_meta","payload":{"id":"`+nativeID+`","parent_thread_id":null,"source":"cli"}}`+"\n"+
			`{"type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":60}}}}`+"\n"))
	pairRoot := sessioninventory.StorageRoot{Name: "pair-data", Path: "/pair/scope"}
	runtime.SetPairDataRoot(pairRoot)
	ledger := `{"v":1,"kind":"launch","scope_key":"scope","tag":"T","agent":"codex","pair_log_offset":0,"native_watermarks":[]}` + "\n"
	if established {
		ledger += `{"v":1,"kind":"binding","scope_key":"scope","tag":"T","agent":"codex","launch_ordinal":1,"root_native_id":"` + nativeID + `"}` + "\n"
	}
	runtime.PutFile(sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: pairRoot.Name, RelativePath: "ledger-T.jsonl"}}, []byte(ledger))
	return runtime
}
