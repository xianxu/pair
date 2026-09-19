package launcher

import (
	"context"
	"errors"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

// #288: the birth watch ends a zellij client hung on a dead-at-birth server by
// cancelling the launch context. The handoff must then return, and only once
// the child is reaped -- a child left behind would hold Couch's pty open, and
// the helper would linger as a zombie that reads as alive. The scripts exec
// their sleeper: a forked orphan would keep this test binary's stdout open.

func TestCancelledHandoffEndsAndReapsTheChild(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cmd := killOnCancel(exec.CommandContext(ctx, "sh", "-c", "exec sleep 60"))
	time.AfterFunc(50*time.Millisecond, cancel)
	start := time.Now()
	code, err := runBlockingHandoff(cmd)
	if err != nil || code != -1 {
		t.Fatalf("handoff = %d, %v; want -1 (signalled), nil", code, err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("a SIGTERM-able child took %s to end", elapsed)
	}
	if err := syscall.Kill(cmd.Process.Pid, 0); !errors.Is(err, syscall.ESRCH) {
		t.Fatalf("child %d still exists after the handoff returned: %v", cmd.Process.Pid, err)
	}
}

func TestCancelledHandoffKillsAChildThatIgnoresSIGTERM(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cmd := killOnCancel(exec.CommandContext(ctx, "sh", "-c", `trap "" TERM; exec sleep 60`))
	cmd.WaitDelay = 200 * time.Millisecond
	time.AfterFunc(50*time.Millisecond, cancel)
	start := time.Now()
	if _, err := runBlockingHandoff(cmd); err != nil {
		t.Fatalf("handoff error = %v", err)
	}
	if elapsed := time.Since(start); elapsed > 1500*time.Millisecond {
		t.Fatalf("a TERM-deaf child took %s to end; WaitDelay should escalate to SIGKILL", elapsed)
	}
	if err := syscall.Kill(cmd.Process.Pid, 0); !errors.Is(err, syscall.ESRCH) {
		t.Fatalf("child %d still exists after the handoff returned: %v", cmd.Process.Pid, err)
	}
}

func TestUncancelledHandoffReturnsTheChildsExitCode(t *testing.T) {
	cmd := killOnCancel(exec.CommandContext(context.Background(), "sh", "-c", "exit 3"))
	if code, err := runBlockingHandoff(cmd); err != nil || code != 3 {
		t.Fatalf("handoff = %d, %v; want 3, nil", code, err)
	}
}
