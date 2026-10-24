package testcouch

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/testcouch/agentfixture"
	"github.com/xianxu/pair/cmd/internal/testpair"
)

func startActorCLIFixture(t *testing.T, agents ...string) (*Fleet, string) {
	t.Helper()
	root := testpair.ShortRoot(t)
	for _, dir := range []string{"home", "codex", "control", "repo"} {
		if err := os.Mkdir(filepath.Join(root, dir), 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, agentfixture.OwnershipFile), []byte("actor-cancel"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(filepath.Join(root, "repo"))
	t.Setenv("COUCH_ISOLATED_ROOT", root)
	t.Setenv("HOME", filepath.Join(root, "home"))
	t.Setenv("CODEX_HOME", filepath.Join(root, "codex"))
	for _, key := range []string{"COUCH_TREE", "COUCH_STORE_DIR", "COUCH_IDENTITY_DIR", "PAIR_DATA_DIR", "COUCH_PAIR_DATA_DIR", "CLAUDE_CONFIG_DIR", "XDG_CONFIG_HOME", "XDG_CACHE_HOME", "XDG_DATA_HOME", "XDG_STATE_HOME", "XDG_RUNTIME_DIR", "ZELLIJ_SOCKET_DIR", "ZELLIJ_CONFIG_DIR"} {
		t.Setenv(key, filepath.Join(root, key))
	}
	worktree := filepath.Join(root, "repo")
	t.Setenv("COUCH_TREE", worktree)
	dir := agentfixture.ActorDirectory(filepath.Join(root, "control"), worktree)
	agent := "claude"
	if len(agents) > 0 {
		agent = agents[0]
	}
	config := agentfixture.Config{Root: root, ControlDir: dir, Home: filepath.Join(root, "home"), CodexHome: filepath.Join(root, "codex"), Agent: agent, CouchBinary: "/bin/sh", RunID: "actor-cancel"}
	input, writer := io.Pipe()
	fixtureCtx, stop := context.WithCancel(t.Context())
	finished := make(chan error, 1)
	go func() { finished <- agentfixture.Run(fixtureCtx, config, input, io.Discard) }()
	t.Cleanup(func() {
		stop()
		_ = writer.Close()
		if err := <-finished; err != nil {
			t.Error(err)
		}
	})
	waitCtx, end := context.WithTimeout(t.Context(), 4*time.Second)
	defer end()
	if err := waitUntil(waitCtx, func() (bool, error) { _, err := os.Stat(filepath.Join(dir, "ready.json")); return err == nil, nil }); err != nil {
		t.Fatal(err)
	}
	f := &Fleet{Roots: RootsPlan{Root: root, Mutable: map[string]string{"control": filepath.Join(root, "control")}}, Workspaces: Workspaces{Primary: worktree}, Options: Options{RuntimeBudget: time.Hour}, RunID: config.RunID, ctx: t.Context()}
	return f, dir
}

func TestActorCLICancellationJoinsAndRetiresOwnedRequest(t *testing.T) {
	f, dir := startActorCLIFixture(t)
	root := f.Roots.Root
	waitCtx, end := context.WithTimeout(t.Context(), 4*time.Second)
	defer end()
	caller, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	marker := filepath.Join(root, "pid")
	go func() {
		_, err := f.ActorCLI(caller, 0, "-c", `printf '%s' "$$" > "$1"; exec sleep 30`, "fixture", marker)
		done <- err
	}()
	var pid int
	if err := waitUntil(waitCtx, func() (bool, error) {
		raw, err := os.ReadFile(marker)
		pid, _ = strconv.Atoi(string(raw))
		return err == nil && pid > 0, nil
	}); err != nil {
		t.Fatal(err)
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("caller cancellation lost: %v", err)
		}
	case <-waitCtx.Done():
		t.Fatal("ActorCLI did not join cancellation")
	}
	if err := syscall.Kill(pid, 0); err != syscall.ESRCH {
		t.Fatalf("ActorCLI returned before child exited: pid=%d err=%v", pid, err)
	}
	for _, sub := range []string{"commands", "results", "cancellations"} {
		entries, err := os.ReadDir(filepath.Join(dir, sub))
		if err != nil || len(entries) != 0 {
			t.Fatalf("ActorCLI retained %s: count=%d err=%v", sub, len(entries), err)
		}
	}
}

func TestActorCLIUncertainDescendantRetainsSlotAndStopsNewWork(t *testing.T) {
	f, dir := startActorCLIFixture(t)
	marker := filepath.Join(f.Roots.Root, "descendant-pid")
	result, err := f.ActorCLI(t.Context(), 0, "-c", `sleep 30 >/dev/null 2>&1 & printf '%s' "$!" > "$1"; exit 0`, "fixture", marker)
	raw, readErr := os.ReadFile(marker)
	if readErr != nil {
		t.Fatal(readErr)
	}
	pid, parseErr := strconv.Atoi(string(raw))
	if parseErr != nil {
		t.Fatal(parseErr)
	}
	generation := procutil.StrictIdentity(strconv.Itoa(pid))
	t.Cleanup(func() {
		if generation != "" && procutil.StrictIdentity(strconv.Itoa(pid)) == generation {
			if e := syscall.Kill(pid, syscall.SIGKILL); e != nil && e != syscall.ESRCH {
				t.Error(e)
			}
		}
	})
	if !errors.Is(err, agentfixture.ErrCleanupUncertain) || !result.CleanupUncertain {
		t.Fatalf("uncertainty lost: result=%+v error=%v", result, err)
	}
	for _, sub := range []string{"commands", "results", "cancellations"} {
		if _, e := os.Stat(filepath.Join(dir, sub, result.ID+".json")); e != nil {
			t.Fatalf("lost %s: %v", sub, e)
		}
	}
	if e := agentfixture.RetireResult(dir, result.ID); !errors.Is(e, agentfixture.ErrCleanupUncertain) {
		t.Fatalf("uncertain slot retired: %v", e)
	}
	forbidden := filepath.Join(f.Roots.Root, "must-not-start")
	if _, e := f.ActorCLI(t.Context(), 0, "-c", `printf wrong > "$1"`, "fixture", forbidden); !errors.Is(e, agentfixture.ErrCleanupUncertain) {
		t.Fatalf("new work admitted: %v", e)
	}
	if _, e := os.Stat(forbidden); !os.IsNotExist(e) {
		t.Fatalf("failed fleet executed more work: %v", e)
	}
}
