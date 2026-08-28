package sessioninventory_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
)

func TestRunCLIResultMatrix(t *testing.T) {
	t.Parallel()

	t.Run("normal absent storage emits complete empty JSON", func(t *testing.T) {
		runtime := sessioninventorytest.NewFakeRuntime()
		root := sessioninventory.StorageRoot{Agent: sessioninventory.AgentCodex, Name: "codex-sessions", Path: "/native/codex"}
		runtime.AddRoot(root)
		runtime.SetError(sessioninventorytest.OperationListFiles, root.Name, sessioninventory.ErrStorageAbsent)
		var stdout, stderr bytes.Buffer
		code := sessioninventory.RunCLIWithRuntime([]string{"--agent", "codex", "--json"}, env(map[string]string{"PAIR_SCOPE_KEY": "scope"}), runtime, &stdout, &stderr)
		if code != 0 || stderr.Len() != 0 || !strings.Contains(stdout.String(), `"schema_version":1`) || !strings.Contains(stdout.String(), `"forests":[]`) {
			t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
		}
	})

	t.Run("unsupported agent is usage exit one", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := sessioninventory.RunCLIWithRuntime([]string{"--agent", "other"}, env(nil), sessioninventorytest.NewFakeRuntime(), &stdout, &stderr)
		if code != 1 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "unsupported agent") {
			t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
		}
	})

	t.Run("present unreadable storage is fatal exit two", func(t *testing.T) {
		runtime := sessioninventorytest.NewFakeRuntime()
		root := sessioninventory.StorageRoot{Agent: sessioninventory.AgentCodex, Name: "codex-sessions", Path: "/native/codex"}
		runtime.AddRoot(root)
		runtime.SetError(sessioninventorytest.OperationListFiles, root.Name, errors.New("secret /home/name"))
		var stdout, stderr bytes.Buffer
		code := sessioninventory.RunCLIWithRuntime([]string{"--agent", "codex", "--json"}, env(nil), runtime, &stdout, &stderr)
		if code != 2 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "storage_unreadable") || strings.Contains(stderr.String(), "/home/name") {
			t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
		}
	})

	t.Run("conformance skip is redacted success", func(t *testing.T) {
		runtime := sessioninventorytest.NewFakeRuntime()
		var stdout, stderr bytes.Buffer
		code := sessioninventory.RunCLIWithRuntime([]string{"--agent", "muse", "--conformance"}, env(nil), runtime, &stdout, &stderr)
		if code != 0 || stderr.Len() != 0 || !strings.Contains(stdout.String(), `"status":"skip"`) {
			t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
		}
	})

	t.Run("current scope projects established ledger binding", func(t *testing.T) {
		runtime := sessioninventorytest.NewFakeRuntime()
		nativeRoot := sessioninventory.StorageRoot{Agent: sessioninventory.AgentCodex, Name: "codex-sessions", Path: "/native/codex"}
		runtime.AddRoot(nativeRoot)
		const nativeID = "019d1111-1111-7111-8111-111111111111"
		runtime.PutFile(sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: nativeRoot.Name, RelativePath: "2026/08/28/rollout-test-" + nativeID + ".jsonl"}}, []byte(
			`{"timestamp":"2026-08-28T10:04:00Z","type":"session_meta","payload":{"id":"`+nativeID+`","parent_thread_id":null,"source":"cli"}}`+"\n"))
		pairRoot := sessioninventory.StorageRoot{Name: "pair-data", Path: "/pair/scope"}
		runtime.SetPairDataRoot(pairRoot)
		ledger := `{"v":1,"kind":"launch","scope_key":"scope","tag":"work","agent":"codex","pair_log_offset":0,"native_watermarks":[]}` + "\n" +
			`{"v":1,"kind":"binding","scope_key":"scope","tag":"work","agent":"codex","launch_ordinal":1,"root_native_id":"` + nativeID + `"}` + "\n"
		runtime.PutFile(sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: pairRoot.Name, RelativePath: "ledger-work.jsonl"}}, []byte(ledger))
		var stdout, stderr bytes.Buffer
		code := sessioninventory.RunCLIWithRuntime([]string{"--agent", "codex", "--json"}, env(map[string]string{"PAIR_SCOPE_KEY": "scope"}), runtime, &stdout, &stderr)
		if code != 0 || stderr.Len() != 0 || !strings.Contains(stdout.String(), `"status":"established"`) || !strings.Contains(stdout.String(), `"scope_key":"scope"`) {
			t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
		}
	})

	t.Run("render writer failure is exit two", func(t *testing.T) {
		var stderr bytes.Buffer
		code := sessioninventory.RunCLIWithRuntime([]string{"--json"}, env(nil), sessioninventorytest.NewFakeRuntime(), failingWriter{}, &stderr)
		if code != 2 || !strings.Contains(stderr.String(), "render write failed") {
			t.Fatalf("code=%d stderr=%q", code, stderr.String())
		}
	})
}

func env(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }
