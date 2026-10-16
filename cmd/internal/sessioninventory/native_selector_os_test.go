package sessioninventory_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	si "github.com/xianxu/pair/cmd/internal/sessioninventory"
)

func TestNativeSelectorCodexSQLiteReadOnlyAndRootValidation(t *testing.T) {
	if _, err := exec.LookPath("sqlite3"); err != nil {
		t.Skip("sqlite3 unavailable")
	}
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	state := filepath.Join(home, ".codex")
	cwd := filepath.Join(home, "work")
	sessions := filepath.Join(state, "sessions", "2026", "08", "28")
	for _, p := range []string{state, cwd, sessions} {
		if err := os.MkdirAll(p, 0700); err != nil {
			t.Fatal(err)
		}
	}
	id := "019d1111-1111-7111-8111-111111111111"
	path := filepath.Join(sessions, "rollout-test-"+id+".jsonl")
	raw := fmt.Sprintf("{\"timestamp\":\"2026-08-28T10:00:00Z\",\"type\":\"session_meta\",\"payload\":{\"id\":%q,\"source\":\"vscode\",\"cwd\":%q}}\n", id, cwd)
	if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	db := filepath.Join(state, "state_5.sqlite")
	query := fmt.Sprintf("CREATE TABLE threads(id TEXT,rollout_path TEXT,updated_at_ms INTEGER,archived INTEGER,preview TEXT,cwd TEXT,source TEXT,model_provider TEXT); INSERT INTO threads VALUES('%s','%s',100000,0,'eligible','%s','vscode','openai');", id, path, cwd)
	if out, err := exec.Command("sqlite3", db, query).CombinedOutput(); err != nil {
		t.Fatalf("fixture SQLite: %s %v", out, err)
	}
	codexLatestFreezeWAL(t, db)
	before, err := os.ReadFile(db)
	if err != nil {
		t.Fatal(err)
	}
	rt := si.NewOSRuntime(home, filepath.Join(home, "pair"))
	q := si.NativeSelectorRequest{Agent: si.AgentCodex, CWD: cwd, Latest: true, Config: si.NativeSelectorConfig{Provider: "openai", ProviderResolved: true, SQLiteHome: state, SQLiteHomeResolved: true}}
	result, err := si.ResolveNativeSelector(t.Context(), rt, q)
	if err != nil || result.Status != si.NativeSelectorResolved || result.NativeID != id {
		t.Fatalf("Codex SQLite resolution=%+v %v", result, err)
	}
	after, err := os.ReadFile(db)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("selector changed native SQLite: %v", err)
	}
	if err := os.WriteFile(path, []byte("invalid native root\n"), 0600); err != nil {
		t.Fatal(err)
	}
	result, err = si.ResolveNativeSelector(t.Context(), rt, q)
	if err != nil || result.Status == si.NativeSelectorResolved {
		t.Fatalf("invalid native root authorized: %+v %v", result, err)
	}
}
