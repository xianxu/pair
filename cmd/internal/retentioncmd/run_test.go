package retentioncmd

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/storagegc"
)

func cliFixture(t *testing.T) (map[string]string, *storagegc.Coordinator) {
	t.Helper()
	c, err := storagegc.NewCoordinator(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return map[string]string{"PAIR_DATA_DIR": c.Root, "PAIR_TAG": "tag"}, c
}
func invoke(env map[string]string, args ...string) (int, string, string) {
	var out, err bytes.Buffer
	code := Run(args, func(k string) string { return env[k] }, &out, &err)
	return code, strings.TrimSpace(out.String()), err.String()
}
func TestRegisterActualWriterLifecycle(t *testing.T) {
	env, c := cliFixture(t)
	child := exec.Command("sleep", "30")
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = child.Process.Kill(); _ = child.Wait() }()
	code, id, stderr := invoke(env, "register", "--pid", strconv.Itoa(child.Process.Pid), "--role", "editor")
	if code != 0 || id == "" || stderr != "" {
		t.Fatalf("register: %d %q %q", code, id, stderr)
	}
	owner, err := storagegc.SelectedOwner(c.Root, "", "tag")
	if err != nil {
		t.Fatal(err)
	}
	state, err := c.ReadOwner(owner)
	if err != nil || len(state.Processes) != 1 || state.Processes[0].Process.PID != child.Process.Pid {
		t.Fatalf("wrong writer: %+v %v", state, err)
	}
	if code, _, stderr := invoke(env, "release", "--id", id); code != 0 {
		t.Fatal(stderr)
	}
	state, err = c.ReadOwner(owner)
	if err != nil || len(state.Processes) != 0 {
		t.Fatalf("release: %+v %v", state, err)
	}
}
func TestUseCompleteAndUnchanged(t *testing.T) {
	for _, action := range []string{"complete", "unchanged"} {
		t.Run(action, func(t *testing.T) {
			env, c := cliFixture(t)
			code, id, stderr := invoke(env, "begin", "--pid", strconv.Itoa(os.Getpid()), "--target", "draft")
			if code != 0 || id == "" {
				t.Fatalf("begin: %d %s", code, stderr)
			}
			owner, _ := storagegc.SelectedOwner(c.Root, "", "tag")
			before, err := c.ReadOwner(owner)
			if err != nil || len(before.Intents) != 1 {
				t.Fatalf("intent: %+v %v", before, err)
			}
			if code, _, stderr := invoke(env, action, "--id", id); code != 0 {
				t.Fatal(stderr)
			}
			after, err := c.ReadOwner(owner)
			if err != nil || len(after.Intents) != 0 {
				t.Fatalf("completion: %+v %v", after, err)
			}
			if action == "unchanged" && !before.Activity.LastUse.Equal(after.Activity.LastUse) {
				t.Fatal("unchanged refreshed use")
			}
			if action == "complete" && !after.Activity.LastUse.After(before.Activity.LastUse) {
				t.Fatal("complete did not publish use")
			}
		})
	}
}
func TestInvalidCommandDoesNotInitializeStorage(t *testing.T) {
	for _, args := range [][]string{nil, {"gc"}, {"register"}, {"register", "--pid", "0", "--role", "editor"}, {"register", "--pid", "2147483646", "--role", "editor"}, {"register", "--pid", "abc", "--role", "editor"}, {"register", "--pid", strconv.Itoa(os.Getpid()), "--role", "editor", "extra"}, {"register", "--pid", strconv.Itoa(os.Getpid()), "--role", "editor", "--unknown", "x"}, {"begin", "--pid", strconv.Itoa(os.Getpid())}, {"complete"}, {"release", "--id", ""}, {"register", "--pid", strconv.Itoa(os.Getpid()), "--pid", strconv.Itoa(os.Getpid()), "--role", "editor"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			env, c := cliFixture(t)
			if code, _, _ := invoke(env, args...); code == 0 {
				t.Fatal("invalid command accepted")
			}
			files, err := os.ReadDir(c.Root)
			if err != nil || len(files) != 0 {
				t.Fatalf("invalid command wrote state: %v %v", files, err)
			}
		})
	}
}
func TestMissingEnvironmentNeverUsesHome(t *testing.T) {
	for _, key := range []string{"PAIR_DATA_DIR", "PAIR_TAG"} {
		t.Run(key, func(t *testing.T) {
			env, c := cliFixture(t)
			env["HOME"] = t.TempDir()
			delete(env, key)
			if code, _, _ := invoke(env, "register", "--pid", strconv.Itoa(os.Getpid()), "--role", "editor"); code == 0 {
				t.Fatal("missing environment accepted")
			}
			for _, path := range []string{c.Root, env["HOME"]} {
				files, err := os.ReadDir(path)
				if err != nil || len(files) != 0 {
					t.Fatalf("fallback wrote %s: %v %v", path, files, err)
				}
			}
		})
	}
	env, c := cliFixture(t)
	env["PAIR_DATA_DIR"] = filepath.Join(c.Root, "missing")
	if code, _, _ := invoke(env, "register", "--pid", strconv.Itoa(os.Getpid()), "--role", "editor"); code == 0 {
		t.Fatal("missing directory accepted")
	}
}

func TestScopedBeginPreservesExactTarget(t *testing.T) {
	env, c := cliFixture(t)
	scopeDir := filepath.Join(c.Root, "repos", "scope")
	if err := os.MkdirAll(scopeDir, 0700); err != nil {
		t.Fatal(err)
	}
	env["PAIR_DATA_DIR"], env["PAIR_SCOPE_KEY"] = scopeDir, "scope"
	capture := filepath.Join(scopeDir, "parked-scrollback-tag-20260913T010101.raw")
	code, id, stderr := invoke(env, "begin", "--pid", strconv.Itoa(os.Getpid()), "--target", capture)
	if code != 0 || id == "" {
		t.Fatalf("scoped begin: %d %s", code, stderr)
	}
	owner, err := storagegc.SelectedOwner(scopeDir, "scope", "tag")
	if err != nil {
		t.Fatal(err)
	}
	state, err := c.ReadOwner(owner)
	if err != nil || len(state.Intents) != 1 || state.Intents[0].Target != capture {
		t.Fatalf("wrong target: %+v %v", state, err)
	}
	if code, _, stderr := invoke(env, "unchanged", "--id", id); code != 0 {
		t.Fatal(stderr)
	}
}

func TestConflictingScopeDoesNotInitialize(t *testing.T) {
	env, c := cliFixture(t)
	scopeDir := filepath.Join(c.Root, "repos", "scope")
	if err := os.MkdirAll(scopeDir, 0700); err != nil {
		t.Fatal(err)
	}
	env["PAIR_DATA_DIR"], env["PAIR_SCOPE_KEY"] = scopeDir, "other"
	if code, _, _ := invoke(env, "register", "--pid", strconv.Itoa(os.Getpid()), "--role", "editor"); code == 0 {
		t.Fatal("conflicting namespace accepted")
	}
	if _, err := os.Stat(filepath.Join(c.Root, ".retention")); !os.IsNotExist(err) {
		t.Fatalf("invalid scope created metadata: %v", err)
	}
}

func TestRepeatedRegisterReusesActualWriterRole(t *testing.T) {
	env, _ := cliFixture(t)
	args := []string{"register", "--pid", strconv.Itoa(os.Getpid()), "--role", "draft-editor"}
	code, first, stderr := invoke(env, args...)
	if code != 0 {
		t.Fatal(stderr)
	}
	code, second, stderr := invoke(env, args...)
	if code != 0 || first != second {
		t.Fatalf("repeated role changed ID: %s %s %s", first, second, stderr)
	}
}

func TestEditorRegisterAcknowledgesLaunchReservation(t *testing.T) {
	env, c := cliFixture(t)
	ctx := context.Background()
	o, err := storagegc.SelectedOwner(c.Root, "", "tag")
	if err != nil {
		t.Fatal(err)
	}
	p, err := storagegc.CurrentProcessIdentity(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	start, err := c.ReserveStart(ctx, o, p, []string{"draft-editor"})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.MarkStartSpawned(ctx, o, start, p); err != nil {
		t.Fatal(err)
	}
	env["PAIR_RETENTION_START_ID"] = start
	if code, _, stderr := invoke(env, "register", "--pid", strconv.Itoa(p.PID), "--role", "draft-editor"); code != 0 {
		t.Fatal(stderr)
	}
	state, err := c.ReadOwner(o)
	if err != nil || len(state.Starts) != 0 || len(state.Processes) != 1 {
		t.Fatalf("editor did not acknowledge: %+v %v", state, err)
	}
}

func TestRegisterOptionalExactTarget(t *testing.T) {
	env, c := cliFixture(t)
	target := filepath.Join(c.Root, "capture.raw")
	code, _, stderr := invoke(env, "register", "--pid", strconv.Itoa(os.Getpid()), "--role", "scrollback-viewer", "--target", target)
	if code != 0 {
		t.Fatal(stderr)
	}
	owner, _ := storagegc.SelectedOwner(c.Root, "", "tag")
	state, err := c.ReadOwner(owner)
	if err != nil || len(state.Processes) != 1 || state.Processes[0].Target != target {
		t.Fatalf("target lost: %+v %v", state, err)
	}
}

func TestResolveStartRequiresExplicitAbsenceAndDeadExactParent(t *testing.T) {
	env, c := cliFixture(t)
	ctx := context.Background()
	o, _ := storagegc.SelectedOwner(c.Root, "", "tag")
	child := exec.Command("sleep", "30")
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = child.Process.Kill(); _ = child.Wait() })
	parent, err := storagegc.CurrentProcessIdentity(child.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}
	id, err := c.ReserveStart(ctx, o, parent, []string{"wrapper"})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.MarkStartSpawned(ctx, o, id, parent); err != nil {
		t.Fatal(err)
	}
	args := []string{"resolve-start", "--id", id, "--parent-pid", strconv.Itoa(parent.PID), "--parent-birth", parent.Birth, "--unregistered-children-absent", "yes"}
	if code, _, _ := invoke(env, args...); code == 0 {
		t.Fatal("live parent resolved")
	}
	if err := child.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = child.Wait()
	if code, _, _ := invoke(env, args[:len(args)-2]...); code == 0 {
		t.Fatal("implicit absence accepted")
	}
	if code, _, stderr := invoke(env, args...); code != 0 {
		t.Fatalf("explicit dead-parent resolution failed: %s", stderr)
	}
	state, err := c.ReadOwner(o)
	if err != nil || len(state.Starts) != 0 {
		t.Fatalf("start remains: %+v %v", state.Starts, err)
	}
}
