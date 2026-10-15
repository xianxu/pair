package wrapcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/xianxu/pair/cmd/internal/launcher"
	si "github.com/xianxu/pair/cmd/internal/sessioninventory"
)

func scopedAdmissionCommand(t *testing.T) (*exec.Cmd, *nativeAdmissionWorld, string) {
	t.Helper()
	h := t.TempDir()
	ch := filepath.Join(h, "codex")
	cwd := filepath.Join(h, "cwd")
	for _, p := range []string{ch, cwd} {
		if err := os.Mkdir(p, 0700); err != nil {
			t.Fatal(err)
		}
	}
	config := filepath.Join(ch, "config.toml")
	if err := os.WriteFile(config, []byte("forced_login_method='api'\n[features]\nworktrees=false\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cmd, w := nativeAdmissionTestCommand(t)
	cmd.Dir = cwd
	cmd.Env = []string{"HOME=" + h, "CODEX_HOME=" + ch}
	// Both initial admission and handoff use the production reader and the child's
	// exact environment. No native executable, DB query, or credential is opened.
	projection, err := scopedAdmissionRead(context.Background(), si.CodexConfigInvocation{CWD: cwd, Mode: si.CodexModeTUI}, func(k string) string {
		switch k {
		case "HOME":
			return h
		case "CODEX_HOME":
			return ch
		}
		return ""
	})
	if err != nil || projection.Status != si.CodexConfigComplete {
		t.Fatalf("private API fixture not complete: %+v %v", projection, err)
	}
	w.codexScope, err = launcher.EncodeNativeAdmissionCodexScope(projection.Scope)
	if err != nil {
		t.Fatal(err)
	}
	w.digest, err = launcher.NativeAdmissionScopedCommandDigest("codex", cmd.Args, w.codexScope)
	if err != nil {
		t.Fatal(err)
	}
	return cmd, w, config
}
func TestNativeAdmissionCodexScopeChangeDoesNotConsume(t *testing.T) {
	for _, kind := range []string{"config", "environment", "cwd", "unknown", "executor", "startup-cwd"} {
		t.Run(kind, func(t *testing.T) {
			cmd, w, config := scopedAdmissionCommand(t)
			switch kind {
			case "config":
				os.WriteFile(config, []byte("forced_login_method='api'\nmodel_provider='other'\n[features]\nworktrees=false\n"), 0600)
			case "unknown":
				os.WriteFile(config, []byte("forced_login_method='chatgpt'\n"), 0600)
			case "environment":
				cmd.Env = append(cmd.Env, "CODEX_SQLITE_HOME="+t.TempDir())
			case "startup-cwd":
				h := filepath.Dir(filepath.Dir(config))
				other := filepath.Join(h, "other")
				ch := filepath.Join(other, "codex")
				if err := os.MkdirAll(ch, 0700); err != nil {
					t.Fatal(err)
				}
				raw, err := os.ReadFile(config)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(ch, "config.toml"), raw, 0600); err != nil {
					t.Fatal(err)
				}
				cmd.Args = append([]string{cmd.Args[0], "-C", cmd.Dir}, cmd.Args[1:]...)
				cmd.Dir = other
				cmd.Env = []string{"HOME=" + h, "CODEX_HOME=codex"}
				// Match the final argv so refusal must come from namespace verification.
				w.digest, _ = launcher.NativeAdmissionScopedCommandDigest("codex", cmd.Args, w.codexScope)
			case "executor":
				cmd.Env = append(cmd.Env, "CODEX_EXEC_SERVER_URL=unix:///private/executor")
			case "cwd":
				cmd.Dir = t.TempDir()
			}
			spawned := false
			_, err := startNativePTYWithCodexConfigReader(context.Background(), w, "valid-grant", "codex", cmd, &bytes.Buffer{}, func(*exec.Cmd) (*os.File, error) { spawned = true; return nil, nil }, scopedAdmissionRead)
			if err == nil || spawned || w.state != "reserved" || len(w.events) != 0 {
				t.Fatalf("changed scope consumed or spawned: state=%s events=%v err=%v", w.state, w.events, err)
			}
		})
	}
}
func TestNativeAdmissionCodexScopeUnchangedPublishes(t *testing.T) {
	for _, kind := range []string{"plain", "cwd-override", "exec", "parent-config-flags", "relative-home", "couch-shell-snapshot"} {
		t.Run(kind, func(t *testing.T) {
			cmd, w, _ := scopedAdmissionCommand(t)
			// Explicit child env wins over the operator shell environment.
			t.Setenv("CODEX_SQLITE_HOME", t.TempDir())
			switch kind {
			case "relative-home":
				h := filepath.Dir(cmd.Dir)
				cmd.Dir = h
				cmd.Env = []string{"HOME=" + h, "CODEX_HOME=codex"}
				cmd.Args = append([]string{cmd.Args[0], "-C", "cwd"}, cmd.Args[1:]...)
			case "cwd-override":
				cmd.Dir = filepath.Dir(cmd.Dir)
				cmd.Args = append([]string{cmd.Args[0], "-C", "cwd"}, cmd.Args[1:]...)
			case "exec":
				cmd.Args = append([]string{cmd.Args[0], "exec"}, cmd.Args[1:]...)
				var scope si.CodexSelectionScope
				if err := json.Unmarshal([]byte(w.codexScope), &scope); err != nil {
					t.Fatal(err)
				}
				scope.Mode = si.CodexModeExec
				var err error
				w.codexScope, err = launcher.EncodeNativeAdmissionCodexScope(scope)
				if err != nil {
					t.Fatal(err)
				}
			case "couch-shell-snapshot":
				cmd.Args = append([]string{cmd.Args[0], "--disable", "shell_snapshot"}, cmd.Args[1:]...)
			case "parent-config-flags":
				cmd.Args = append([]string{cmd.Args[0], "-c", "features.shell_snapshot=false"}, cmd.Args[1:]...)
			}
			var err error
			w.digest, err = launcher.NativeAdmissionScopedCommandDigest("codex", cmd.Args, w.codexScope)
			if err != nil {
				t.Fatal(err)
			}
			_, err = startNativePTYWithCodexConfigReader(context.Background(), w, "valid-grant", "codex", cmd, &bytes.Buffer{}, func(cmd *exec.Cmd) (*os.File, error) {
				w.childAlive = true
				cmd.Process = &os.Process{Pid: 123}
				return nil, nil
			}, scopedAdmissionRead)
			if err != nil || w.state != "running" {
				t.Fatalf("unchanged scope refused: %s %v", w.state, err)
			}
		})
	}
}

func scopedAdmissionRead(ctx context.Context, q si.CodexConfigInvocation, getenv func(string) string) (si.CodexConfigProjection, error) {
	reader := si.CodexConfigReader{SystemDir: filepath.Join(getenv("HOME"), "system"), ManagedPolicy: func() (si.CodexManagedPolicyStatus, error) { return si.CodexManagedPolicyAbsent, nil }}
	return reader.Read(ctx, q, getenv)
}
