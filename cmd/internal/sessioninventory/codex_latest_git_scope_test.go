package sessioninventory

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// Native worktree discovery canonicalizes admin targets, while native trust
// rejects original admin aliases. Unsupported aliases must retain native
// selection rather than silently remove a newer matching checkout.
func TestCodexLatestGitAdminAliasCannotSelectOlderConversation(t *testing.T) {
	if _, err := exec.LookPath("sqlite3"); err != nil {
		t.Skip("sqlite3 unavailable")
	}
	primary, linked, admin := codexTestWorktrees(t)
	reader, q, home, _ := codexTestConfig(t, "forced_login_method='api'\n")
	privateHome := reader.getenv("HOME")
	t.Setenv("HOME", privateHome)
	q.CWD = filepath.Join(primary, "sub")
	sessions := filepath.Join(home, "sessions", "2026", "08", "28")
	if err := os.MkdirAll(sessions, 0700); err != nil {
		t.Fatal(err)
	}
	ids := []string{"019d1111-1111-7111-8111-000000000011", "019d1111-1111-7111-8111-000000000012"}
	sql := "PRAGMA journal_mode=DELETE;CREATE TABLE threads(id TEXT,rollout_path TEXT,updated_at_ms INTEGER,archived INTEGER,preview TEXT,cwd TEXT,source TEXT,model_provider TEXT);"
	for n, cwd := range []string{q.CWD, filepath.Join(linked, "sub")} {
		path := filepath.Join(sessions, "rollout-test-"+ids[n]+".jsonl")
		payload := codexEligibilityMeta()
		payload["id"] = ids[n]
		payload["cwd"] = cwd
		if err := os.WriteFile(path, append(codexEligibilityLine(t, "session_meta", payload), '\n'), 0600); err != nil {
			t.Fatal(err)
		}
		sql += fmt.Sprintf("INSERT INTO threads VALUES(%s,%s,%d,0,'visible',%s,'cli','openai');", codexSQLQuote(ids[n]), codexSQLQuote(path), (n+1)*100, codexSQLQuote(cwd))
	}
	if out, err := exec.Command("sqlite3", filepath.Join(home, "state_5.sqlite"), sql).CombinedOutput(); err != nil {
		t.Fatalf("fixture: %s %v", out, err)
	}
	codexTestFreezeWAL(t, filepath.Join(home, "state_5.sqlite"))
	pick := func() string {
		t.Helper()
		p, err := readCodexSelectionConfig(context.Background(), q, reader)
		if err != nil {
			t.Fatal(err)
		}
		if p.Status == CodexConfigCompatibility {
			return "compatibility"
		}
		if p.Status != CodexConfigComplete {
			t.Fatalf("unexpected projection %+v", p)
		}
		rt, err := NewOSRuntime(privateHome, filepath.Join(privateHome, "pair")).WithCodexScope(p.Scope)
		if err != nil {
			t.Fatal(err)
		}
		result, err := ResolveNativeSelector(t.Context(), rt, NativeSelectorRequest{Agent: AgentCodex, CWD: p.Scope.CWD, Latest: true, Config: NativeSelectorConfig{CodexScope: &p.Scope}})
		if err != nil || result.Status != NativeSelectorResolved {
			t.Fatalf("selection %+v %v", result, err)
		}
		return result.NativeID
	}
	if got := pick(); got != ids[1] {
		t.Fatalf("native-valid control disconnected: got %s want %s", got, ids[1])
	}
	alias := admin + "-alias"
	if err := os.Symlink(admin, alias); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(linked, ".git"), []byte("gitdir: "+alias+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if got := pick(); got != ids[1] && got != "compatibility" {
		t.Fatalf("native-valid aliased linked checkout was silently excluded: selected %s, want newer %s or compatibility", got, ids[1])
	}
}
