package launcher

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	si "github.com/xianxu/pair/cmd/internal/sessioninventory"
)

func TestCodexResumeOSUsesPrivateAPIOnlyScope(t *testing.T) {
	if _, err := exec.LookPath("sqlite3"); err != nil {
		t.Skip("sqlite3 unavailable")
	}
	for _, prefix := range [][]string{nil, {"exec"}} {
		t.Run(fmt.Sprint(prefix), func(t *testing.T) {
			root, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			home, codex, state, cwd := filepath.Join(root, "home"), filepath.Join(root, "native"), filepath.Join(root, "state"), filepath.Join(root, "work")
			sessions := filepath.Join(codex, "sessions", "2026", "10", "15")
			for _, p := range []string{home, codex, state, cwd, sessions, filepath.Join(root, "system")} {
				if err := os.MkdirAll(p, 0700); err != nil {
					t.Fatal(err)
				}
			}
			t.Setenv("HOME", home)
			t.Setenv("CODEX_HOME", codex)
			t.Setenv("CODEX_SQLITE_HOME", filepath.Join(root, "wrong-env-state"))
			t.Setenv("CODEX_EXEC_SERVER_URL", "")
			if err := os.WriteFile(filepath.Join(codex, "config.toml"), []byte("forced_login_method='api'\nsqlite_home='../state'\n"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(codex, "runtime.config.toml"), []byte("[features]\nworktrees=false\n"), 0600); err != nil {
				t.Fatal(err)
			}
			id := "019d1111-1111-7111-8111-111111111111"
			path := filepath.Join(sessions, "rollout-test-"+id+".jsonl")
			raw := fmt.Sprintf("{\"timestamp\":\"2026-10-15T10:00:00Z\",\"type\":\"session_meta\",\"payload\":{\"id\":%q,\"source\":\"cli\",\"cwd\":%q}}\n", id, cwd)
			if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
				t.Fatal(err)
			}
			db := filepath.Join(state, "state_5.sqlite")
			sql := fmt.Sprintf("CREATE TABLE threads(id TEXT,rollout_path TEXT,updated_at_ms INTEGER,archived INTEGER,preview TEXT,cwd TEXT,source TEXT,model_provider TEXT); INSERT INTO threads VALUES('%s','%s',100000,0,'visible','%s','cli','openai');", id, path, cwd)
			if out, err := exec.Command("sqlite3", db, sql).CombinedOutput(); err != nil {
				t.Fatalf("fixture %s %v", out, err)
			}
			reader := si.CodexConfigReader{SystemDir: filepath.Join(root, "system"), ManagedPolicy: func() (si.CodexManagedPolicyStatus, error) { return si.CodexManagedPolicyAbsent, nil }}
			rt := OSRuntime{DataDir: filepath.Join(root, "pair"), codexConfigReader: &reader}
			args := append([]string{"-p", "runtime"}, prefix...)
			args = append(args, "resume", "--last", "continue work")
			selector, err := ParseNativeResumeSelector("codex", args)
			if err != nil {
				t.Fatal(err)
			}
			result, err := rt.ResolveNativeResume(t.Context(), NativeResumeRequest{Agent: "codex", CWD: cwd, Selector: selector})
			if err != nil || result.Status != si.NativeSelectorResolved || result.NativeID != id || result.CodexScope == nil {
				t.Fatalf("resolution=%+v err=%v", result, err)
			}
			if result.CodexScope.CodexHome != codex || result.CodexScope.SQLiteHome != state {
				t.Fatalf("wrong physical namespace: %+v", result.CodexScope)
			}
			if !reflect.DeepEqual(args, selector.Args) {
				t.Fatal("selection changed original args")
			}
			pinned, err := ParseNativeResumeSelector("codex", result.Args)
			if err != nil || pinned.NativeID != id || pinned.Kind != NativeResumeExplicit {
				t.Fatalf("not pinned: %+v %v", pinned, err)
			}
			// A later arrival cannot retarget explicit selection; the same custom home
			// remains usable even when latest eligibility becomes cloud-unknown.
			if err := os.WriteFile(filepath.Join(codex, "config.toml"), []byte("sqlite_home='../state'\n"), 0600); err != nil {
				t.Fatal(err)
			}
			checked, err := rt.ResolveNativeResume(t.Context(), NativeResumeRequest{Agent: "codex", CWD: cwd, Selector: pinned})
			if err != nil || checked.NativeID != id {
				t.Fatalf("explicit custom-home revalidation=%+v %v", checked, err)
			}
			unknown, err := rt.ResolveNativeResume(t.Context(), NativeResumeRequest{Agent: "codex", CWD: cwd, Selector: selector})
			if err != nil || unknown.Status != si.NativeSelectorUnsupported || unknown.Reason != string(si.CodexConfigCloudUnknown) || unknown.CodexScope != nil {
				t.Fatalf("cloud-unknown gained authority: %+v %v", unknown, err)
			}
		})
	}
}
