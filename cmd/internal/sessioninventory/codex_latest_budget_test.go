package sessioninventory

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func codexLatestBudgetRuntime(t *testing.T) (*nativeSelectorRuntime, Artifact) {
	t.Helper()
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	a := Artifact{StorageRoot: "codex-sessions", RelativePath: "2026/08/28/rollout-test-019d1111-1111-7111-8111-000000000001.jsonl", Kind: ArtifactTranscript}
	p := filepath.Join(home, ".codex", "sessions", filepath.FromSlash(a.RelativePath))
	if err = os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(p, []byte("{\"timestamp\":\"2026-08-28T10:00:00Z\",\"type\":\"session_meta\",\"payload\":{\"id\":\"019d1111-1111-7111-8111-000000000001\",\"timestamp\":\"2026-08-28T10:00:00Z\",\"cwd\":\"/selected\",\"originator\":\"codex_cli_rs\",\"cli_version\":\"0.162.1\"}}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	rt, err := nativeSelectorCodexRuntime(NewOSRuntime(home, filepath.Join(home, "pair")), filepath.Join(home, ".codex"))
	if err != nil {
		t.Fatal(err)
	}
	r, err := newNativeSelectorRuntime(context.Background(), rt)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := r.Close(); err != nil {
			t.Error(err)
		}
	})
	if _, err = r.StatFile(a); err != nil {
		t.Fatal(err)
	}
	return r, a
}
func TestCodexLatestReverseScanCrossesChunksAndRejectsPartialTail(t *testing.T) {
	for _, partial := range []bool{false, true} {
		t.Run(fmt.Sprint(partial), func(t *testing.T) {
			r, a := codexLatestBudgetRuntime(t)
			root := r.NativeRoots(AgentCodex)[0]
			p := filepath.Join(root.Path, filepath.FromSlash(a.RelativePath))
			raw := "{\"timestamp\":\"2026-08-28T10:00:01Z\",\"type\":\"turn_context\",\"payload\":{\"cwd\":\"/selected\",\"approval_policy\":\"never\",\"sandbox_policy\":{\"type\":\"read-only\"},\"model\":\"test-model\"}}\n" + fmt.Sprintf("{\"timestamp\":\"2026-08-28T10:00:02Z\",\"type\":\"world_state\",\"payload\":{\"full\":true,\"state\":{\"text\":%q}}}\n", strings.Repeat("x", 256<<10))
			if partial {
				raw += `{"type":"turn_context",`
			}
			if err := os.WriteFile(p, []byte(raw), 0600); err != nil {
				t.Fatal(err)
			}
			r.StatFile(a)
			cwd, err := latestCodexTurnCWD(r, a)
			if partial {
				if err == nil {
					t.Fatal("partial tail established cwd")
				}
				return
			}
			if err != nil || cwd != "/selected" {
				t.Fatalf("reverse cwd=%q err=%v", cwd, err)
			}
		})
	}
}
func TestCodexLatestReverseScanNeverEstablishesAbsenceBeyondBudget(t *testing.T) {
	r, a := codexLatestBudgetRuntime(t)
	root := r.NativeRoots(AgentCodex)[0]
	p := filepath.Join(root.Path, filepath.FromSlash(a.RelativePath))
	h, err := os.OpenFile(p, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	line := fmt.Sprintf("{\"timestamp\":\"2026-08-28T10:00:02Z\",\"type\":\"world_state\",\"payload\":{\"full\":true,\"state\":{\"text\":%q}}}\n", strings.Repeat("x", 64<<10))
	for n := 0; n < 520; n++ {
		if _, err = h.WriteString(line); err != nil {
			t.Fatal(err)
		}
	}
	h.Close()
	r.StatFile(a)
	if _, err = latestCodexTurnCWD(r, a); err == nil {
		t.Fatal("exhausted reverse scan established absence")
	}
	if r.bytes > 32<<20 {
		t.Fatalf("reverse read budget=%d", r.bytes)
	}
}
func TestCodexLatestSQLiteSharesReadAndMetadataBudgets(t *testing.T) {
	if _, err := exec.LookPath("sqlite3"); err != nil {
		t.Skip("sqlite3 unavailable")
	}
	for _, metadata := range []bool{false, true} {
		t.Run(fmt.Sprint(metadata), func(t *testing.T) {
			r, _ := codexLatestBudgetRuntime(t)
			var home string
			for _, root := range r.Runtime.NativeRoots(AgentCodex) {
				if root.Name == "codex-state" {
					home = root.Path
				}
			}
			if home == "" {
				t.Fatal("missing private state root")
			}
			sql := "CREATE TABLE threads(id TEXT, rollout_path TEXT,updated_at_ms INTEGER,cwd TEXT,archived INTEGER,preview TEXT);INSERT INTO threads VALUES('id','path',1,'cwd',0,'visible')"
			if out, err := exec.Command("sqlite3", filepath.Join(home, "state_5.sqlite"), sql).CombinedOutput(); err != nil {
				t.Fatalf("fixture: %s %v", out, err)
			}
			codexTestFreezeWAL(t, filepath.Join(home, "state_5.sqlite"))
			if rows, err := codexLatestRows(r, "", 0, 2); err != nil || len(rows) != 1 {
				t.Fatalf("unexhausted WAL control rows=%v err=%v", rows, err)
			}
			if metadata {
				r.entries = RecoveryMetadataLimit
			} else {
				r.bytes = RecoveryTotalReadLimit - 1
			}
			if _, err := codexLatestRows(r, "", 0, 2); err == nil {
				t.Fatal("exhausted shared budget accepted SQL evidence")
			}
		})
	}
}
