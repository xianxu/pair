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
	const nativeID = "019d1111-1111-7111-8111-111111111111"
	mtime := time.Date(2026, 8, 28, 10, 9, 0, 0, time.UTC)
	runtime := sessioninventorytest.NewFakeRuntime()
	pairRoot := sessioninventory.StorageRoot{Name: "pair-data", Path: "/pair/scope"}
	runtime.SetPairDataRoot(pairRoot)
	ledgerArtifact := sessioninventory.Artifact{StorageRoot: pairRoot.Name, RelativePath: "ledger-work.jsonl"}
	launch := `{"v":2,"kind":"launch","scope_key":"scope","tag":"work","agent":"codex","pair_log_offset":0,"artifact_boundaries":[]}` + "\n"
	runtime.PutFile(sessioninventory.FileEntry{Artifact: ledgerArtifact}, []byte(launch))

	getenv := func(key string) string { return map[string]string{"PAIR_SCOPE_KEY": "scope", "PAIR_TAG": "work"}[key] }
	var stdout, stderr bytes.Buffer
	if code := sessioninventory.RunCLIWithRuntime([]string{"--activity", "--agent", "codex"}, getenv, runtime, &stdout, &stderr); code != 0 || stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("provisional code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	runtime, _, _ = proofBackedCodexFixture(t, nativeID, &mtime)
	stdout.Reset()
	if code := sessioninventory.RunCLIWithRuntime([]string{"--activity", "--agent", "codex"}, getenv, runtime, &stdout, &stderr); code != 0 || !strings.Contains(stdout.String(), `"last_activity_at":"2026-08-28T10:09:00Z"`) || strings.Contains(stdout.String(), "root_node_id") {
		t.Fatalf("established code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}
