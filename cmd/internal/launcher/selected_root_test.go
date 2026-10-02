package launcher

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestSelectedRootValidationBeforeNativeEffects(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(root, "selected")
	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, "file")
	if err := os.WriteFile(file, []byte("not a directory"), 0600); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(root, "alias")
	if err := os.Symlink(directory, alias); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", filepath.Join(root, "home"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "ambient"))
	t.Setenv("PAIR_DATA_DIR", "")
	t.Setenv("PAIR_COUCH_SESSION_INTENT", "")
	t.Setenv("PAIR_COUCH_LAUNCH_PROFILE", "")
	for _, invalid := range []string{"relative", directory + "/../selected", filepath.Join(root, "missing"), file, alias} {
		t.Setenv("COUCH_PAIR_DATA_DIR", invalid)
		var out, diagnostic bytes.Buffer
		code, err := LaunchNative([]string{"list"}, root, &out, &diagnostic)
		if err != nil || code == 0 || !strings.Contains(diagnostic.String(), "COUCH_PAIR_DATA_DIR") {
			t.Errorf("invalid selected root %q: %d %v %s", invalid, code, err, &diagnostic)
		}
	}
	for _, absent := range []string{"home", "ambient", "missing"} {
		if _, err := os.Stat(filepath.Join(root, absent)); !os.IsNotExist(err) {
			t.Errorf("refusal initialized %s: %v", absent, err)
		}
	}
	if got, err := ResolveGlobalDataDir("ignored", "ignored", directory); err != nil || got != directory {
		t.Fatalf("valid selected root: %q %v", got, err)
	}
	if got, err := ResolveGlobalDataDir(root, "", ""); err != nil || got != ResolveDataDir(root, "") {
		t.Fatalf("standalone global default: %q %v", got, err)
	}
}

func TestSelectedRootNativeCreateListResume(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	pairHome := mustPairHome(t)
	binary := filepath.Join(root, "pair")
	build := exec.Command("go", "build", "-o", binary, "../../pair-go")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v %s", err, out)
	}
	home, ambient, selected := filepath.Join(root, "home"), filepath.Join(root, "ambient"), filepath.Join(root, "selected")
	repo, bin := filepath.Join(root, "repo"), filepath.Join(root, "bin")
	for _, dir := range []string{home, ambient, selected, filepath.Join(repo, "subdir"), bin} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	if out, err := exec.Command("git", "-C", repo, "init").CombinedOutput(); err != nil {
		t.Fatalf("git: %v %s", err, out)
	}
	// Model zellij's persistent session inventory across separate real Pair
	// invocations. No real server, agent, or operator terminal is involved.
	fake := `#!/bin/sh
case "$1" in
  list-sessions)
    if [ -f "$PAIR_ROOT_TEST_STATE" ]; then cat "$PAIR_ROOT_TEST_STATE"; fi
    exit 0 ;;
  --session) printf 'CLIENT_ID ZELLIJ_PANE_ID RUNNING_COMMAND\n'; exit 0 ;;
  --config-dir)
    if [ "$3" = attach ]; then
      printf 'attach\n' >> "$PAIR_ROOT_TEST_EFFECTS"
    else
      printf '%s\n' "$PAIR_SESSION_NAME" > "$PAIR_ROOT_TEST_STATE"
      printf 'create\n' >> "$PAIR_ROOT_TEST_EFFECTS"
    fi
    printf '%s\n%s\n' "$COUCH_PAIR_DATA_DIR" "$PAIR_DATA_DIR" > "$PAIR_ROOT_TEST_ENV"
    exit 0 ;;
esac
exit 0
`
	for name, body := range map[string]string{"zellij": fake, "claude": "#!/bin/sh\nexit 0\n"} {
		if err := os.WriteFile(filepath.Join(bin, name), []byte(body), 0700); err != nil {
			t.Fatal(err)
		}
	}
	scope, err := ResolveRepoScope(repo)
	if err != nil {
		t.Fatal(err)
	}
	const tag = "1-repo-1"
	if _, err := ClaimNewThreadAddress(selected, scope, tag); err != nil {
		t.Fatal(err)
	}
	paths := NewScopedPaths(selected, scope, tag)
	baseEnv := []string{"PATH=" + bin + ":/usr/bin:/bin", "HOME=" + home, "XDG_DATA_HOME=" + ambient,
		"PAIR_HOME=" + pairHome, "COUCH_PAIR_DATA_DIR=" + selected, "COUCH_THREAD_SCOPE=" + scope.Key, "COUCH_THREAD_TAG=" + tag,
		"PAIR_ROOT_TEST_STATE=" + filepath.Join(root, "sessions"), "PAIR_ROOT_TEST_EFFECTS=" + filepath.Join(root, "effects"), "PAIR_ROOT_TEST_ENV=" + filepath.Join(root, "child-env")}
	run := func(extra []string, args ...string) (string, error) {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, binary, args...)
		cmd.Dir = filepath.Join(repo, "subdir")
		cmd.Env = append(append([]string(nil), baseEnv...), extra...)
		// Hosted sidecars remain in this actor-owned process group. Collect the
		// exact group after each invocation, including timeout/failure paths.
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		out, err := cmd.CombinedOutput()
		if cmd.Process != nil {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		return string(out), err
	}
	if out, err := run(nil, "resume", tag); err != nil {
		t.Fatalf("create: %v %s", err, out)
	}
	for _, path := range []string{paths.Ledger(), paths.Draft(), paths.Agent(), paths.ThreadClaim()} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("selected scoped artifact %s: %v", path, err)
		}
	}
	if err := RegisterExistingCouchThread(selected, scope, tag); err != nil {
		t.Fatalf("global claim was not established: %v", err)
	}
	index, err := NewScopedOSRuntime(selected, paths.ScopeDir(), pairHome).ReadSessionNameIndex()
	if err != nil || len(index.Entries) == 0 {
		t.Fatalf("selected binding index: %+v %v", index, err)
	}
	if out, err := run(nil, "list"); err != nil || !strings.Contains(out, tag) {
		t.Fatalf("selected list: %v %s", err, out)
	}
	continuations := filepath.Join(repo, "workshop", "continuation")
	if err := os.MkdirAll(continuations, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(continuations, "20261001T000000-selected-root.md"), []byte("---\nagent: claude\n---\n\n## Next action\nContinue selected-root work.\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if out, err := run(nil, "continue"); err != nil || !strings.Contains(out, "selected-root") {
		t.Fatalf("continuation listing from repository subdir: %v %s", err, out)
	}
	if out, err := run([]string{"PAIR_DATA_DIR=" + paths.ScopeDir()}, "resume", tag); err != nil {
		t.Fatalf("scoped warm resume: %v %s", err, out)
	}
	effects, err := os.ReadFile(filepath.Join(root, "effects"))
	if err != nil || string(effects) != "create\nattach\n" {
		t.Fatalf("resume recreated rather than reused selected thread: %q %v", effects, err)
	}
	childEnv, err := os.ReadFile(filepath.Join(root, "child-env"))
	if err != nil || string(childEnv) != selected+"\n"+paths.ScopeDir()+"\n" {
		t.Fatalf("descendant roots: %q %v", childEnv, err)
	}
	for _, bad := range []string{selected, filepath.Join(selected, "repos", "other"), filepath.Join(root, "outside")} {
		if out, err := run([]string{"PAIR_DATA_DIR=" + bad}, "list"); err == nil || !strings.Contains(out, "PAIR_DATA_DIR") {
			t.Errorf("conflicting scoped override %q accepted: %v %s", bad, err, out)
		}
	}
	for _, path := range []string{filepath.Join(ambient, "pair", "repos"), filepath.Join(home, ".local", "share", "pair", "repos"), filepath.Join(selected, "ledger-"+tag+".jsonl")} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("artifact escaped selected scope at %s: %v", path, err)
		}
	}
}
