package couchcmd

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/testpair"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

func TestSessionStaleCommandBoundsAndFailure(t *testing.T) {
	for _, tc := range []struct {
		name, script string
		timeout      time.Duration
		wantErr      bool
	}{
		{"success", "printf 'stale: ## Threads (unstamped)\\n'", time.Second, false},
		{"failure", "printf 'broken check' >&2; exit 7", time.Second, true},
		{"overflow", "while :; do printf 'xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx\\n'; done", time.Second, true},
		{"cancel", "/bin/sleep 30 & wait", 40 * time.Millisecond, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), tc.timeout)
			defer cancel()
			cmd := exec.CommandContext(ctx, "/bin/sh", "-c", tc.script)
			cmd.Env = testpair.Environment(t, t.TempDir())
			started := time.Now()
			out, err := runStaleCommand(cmd)
			if (err != nil) != tc.wantErr {
				t.Fatalf("out %q err %v", out, err)
			}
			if len(out) > couchmessage.MaxBodyBytes {
				t.Fatal("unbounded output")
			}
			if !tc.wantErr && !strings.Contains(out, "Threads") {
				t.Fatal("lost output")
			}
			if time.Since(started) > 3*time.Second {
				t.Fatal("command/drain deadline not enforced")
			}
		})
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/missing-tl-lead-fixture")
	cmd.Env = testpair.Environment(t, t.TempDir())
	if _, err := runStaleCommand(cmd); err == nil {
		t.Fatal("missing command succeeded")
	}
}

func TestSessionStaleCommandCancellationKillsDescendant(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	root := t.TempDir()
	pidFile := filepath.Join(root, "child.pid")
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", `/bin/sleep 30 & child=$!; printf '%s' "$child" > "$1"; wait`, "fixture", pidFile)
	cmd.Env = testpair.Environment(t, root)
	done := make(chan error, 1)
	go func() { _, err := runStaleCommand(cmd); done <- err }()
	var pid int
	testwait.Until(t, "child started", func() bool {
		raw, err := os.ReadFile(pidFile)
		if err != nil {
			return false
		}
		pid, _ = strconv.Atoi(string(raw))
		return pid > 0
	})
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("cancel succeeded")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("check did not stop")
	}
	testwait.Until(t, "owned descendant collected", func() bool { return errors.Is(syscall.Kill(pid, 0), syscall.ESRCH) })
}

func TestSessionStaleCommandSuccessCollectsClosedPipeDescendant(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	root := t.TempDir()
	pidFile := filepath.Join(root, "child.pid")
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", `/bin/sleep 30 >/dev/null 2>&1 & child=$!; printf '%s' "$child" > "$1"; exit 0`, "fixture", pidFile)
	cmd.Env = testpair.Environment(t, root)
	if _, err := runStaleCommand(cmd); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(string(raw))
	if err != nil || pid <= 0 {
		t.Fatalf("child PID %q: %v", raw, err)
	}
	collected := false
	defer func() {
		if !collected {
			_ = syscall.Kill(pid, syscall.SIGKILL)
		}
	}()
	until := time.Now().Add(time.Second)
	for time.Now().Before(until) {
		if errors.Is(syscall.Kill(pid, 0), syscall.ESRCH) {
			collected = true
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("successful command left a live child after closing its output pipes")
}
