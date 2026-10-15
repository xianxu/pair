package sessioninventory

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCodexLatestConfigFeaturesTableReplacementSelectsLinkedCheckout(t *testing.T) {
	if _, err := exec.LookPath("sqlite3"); err != nil {
		t.Skip("sqlite3 unavailable")
	}
	primary, linked, _ := codexTestWorktrees(t)
	reader, q, home, _ := codexTestConfig(t, "forced_login_method='api'\n")
	privateHome := reader.getenv("HOME")
	t.Setenv("HOME", privateHome)
	q.CWD = filepath.Join(primary, "sub")
	sessions := filepath.Join(home, "sessions", "2026", "08", "28")
	if err := os.MkdirAll(sessions, 0700); err != nil {
		t.Fatal(err)
	}
	sql := "CREATE TABLE threads(id TEXT,rollout_path TEXT,updated_at_ms INTEGER,archived INTEGER,preview TEXT,cwd TEXT,source TEXT,model_provider TEXT);"
	ids := []string{"019d1111-1111-7111-8111-000000000001", "019d1111-1111-7111-8111-000000000002"}
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
		t.Fatalf("SQLite fixture: %s %v", out, err)
	}
	for _, tc := range []struct {
		name      string
		overrides []string
		want      string
		worktrees bool
	}{
		{"disabled-control", []string{"features={worktrees=false}"}, ids[0], false},
		{"native-leaf-replacement", []string{"features={worktrees=false}", "features={shell_snapshot=false}"}, ids[1], true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			q.Overrides = tc.overrides
			projection, err := readCodexSelectionConfig(context.Background(), q, reader)
			if err != nil || projection.Status != CodexConfigComplete || projection.Scope.Worktrees != tc.worktrees {
				t.Fatalf("configuration=%+v err=%v", projection, err)
			}
			native, err := NewOSRuntime(privateHome, filepath.Join(privateHome, "pair")).WithCodexScope(projection.Scope)
			if err != nil {
				t.Fatal(err)
			}
			result, err := ResolveNativeSelector(t.Context(), native, NativeSelectorRequest{Agent: AgentCodex, CWD: projection.Scope.CWD, Latest: true, Config: NativeSelectorConfig{CodexScope: &projection.Scope}})
			if err != nil || result.Status != NativeSelectorResolved || result.NativeID != tc.want {
				t.Fatalf("selected=%+v err=%v want=%s", result, err, tc.want)
			}
		})
	}
}
