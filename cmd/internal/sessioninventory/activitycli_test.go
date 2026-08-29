package sessioninventory_test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
)

func TestActivityCLIEmitsOnlyEstablishedAuthorizedActivity(t *testing.T) {
	t.Parallel()
	runtime := sessioninventorytest.NewFakeRuntime()
	nativeRoot := sessioninventory.StorageRoot{Agent: sessioninventory.AgentCodex, Name: "codex-sessions", Path: "/native/codex"}
	runtime.AddRoot(nativeRoot)
	const nativeID = "019d1111-1111-7111-8111-111111111111"
	mtime := time.Date(2026, 8, 28, 10, 9, 0, 0, time.UTC)
	artifact := sessioninventory.Artifact{StorageRoot: nativeRoot.Name, RelativePath: "2026/08/28/rollout-test-" + nativeID + ".jsonl"}
	runtime.PutFile(sessioninventory.FileEntry{Artifact: artifact, ModTime: &mtime}, []byte(
		`{"timestamp":"2026-08-28T10:04:00Z","type":"session_meta","payload":{"id":"`+nativeID+`","parent_thread_id":null,"source":"cli"}}`+"\n"))
	pairRoot := sessioninventory.StorageRoot{Name: "pair-data", Path: "/pair/scope"}
	runtime.SetPairDataRoot(pairRoot)
	ledgerArtifact := sessioninventory.Artifact{StorageRoot: pairRoot.Name, RelativePath: "ledger-work.jsonl"}
	launch := `{"v":1,"kind":"launch","scope_key":"scope","tag":"work","agent":"codex","pair_log_offset":0,"native_watermarks":[]}` + "\n"
	runtime.PutFile(sessioninventory.FileEntry{Artifact: ledgerArtifact}, []byte(launch))

	getenv := func(key string) string { return map[string]string{"PAIR_SCOPE_KEY": "scope", "PAIR_TAG": "work"}[key] }
	var stdout, stderr bytes.Buffer
	if code := sessioninventory.RunCLIWithRuntime([]string{"--activity", "--agent", "codex"}, getenv, runtime, &stdout, &stderr); code != 0 || stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("provisional code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	runtime.PutFile(sessioninventory.FileEntry{Artifact: ledgerArtifact}, []byte(launch+
		`{"v":1,"kind":"binding","scope_key":"scope","tag":"work","agent":"codex","launch_ordinal":1,"root_native_id":"`+nativeID+`"}`+"\n"))
	stdout.Reset()
	if code := sessioninventory.RunCLIWithRuntime([]string{"--activity", "--agent", "codex"}, getenv, runtime, &stdout, &stderr); code != 0 || !strings.Contains(stdout.String(), `"last_activity_at":"2026-08-28T10:09:00Z"`) || strings.Contains(stdout.String(), "root_node_id") {
		t.Fatalf("established code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}
