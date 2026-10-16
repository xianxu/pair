package launcher

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/testwait"
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
	code, err := runBlockingHandoff(cmd)
	if err != nil || code != -1 {
		t.Fatalf("handoff = %d, %v; want -1 (signalled), nil", code, err)
	}
	if sig := endingSignal(cmd); sig != syscall.SIGTERM {
		t.Fatalf("a SIGTERM-able child ended by %v, want SIGTERM", sig)
	}
	if err := syscall.Kill(cmd.Process.Pid, 0); !errors.Is(err, syscall.ESRCH) {
		t.Fatalf("child %d still exists after the handoff returned: %v", cmd.Process.Pid, err)
	}
}

func TestCancelledHandoffKillsAChildThatIgnoresSIGTERM(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Cancel only once the trap is set: a TERM that beats it ends the child
	// by SIGTERM and never exercises the escalation (a loaded run did).
	trapped := filepath.Join(t.TempDir(), "trapped")
	cmd := killOnCancel(exec.CommandContext(ctx, "sh", "-c", `trap "" TERM; : > "$1"; exec sleep 60`, "sh", trapped))
	cmd.WaitDelay = 200 * time.Millisecond
	handoff := make(chan error, 1)
	go func() { _, err := runBlockingHandoff(cmd); handoff <- err }()
	testwait.Until(t, "the child to set its TERM trap", func() bool { _, err := os.Stat(trapped); return err == nil })
	cancel()
	select {
	case err := <-handoff:
		if err != nil {
			t.Fatalf("handoff error = %v", err)
		}
	case <-testwait.After(t):
		t.Fatal("the handoff never returned from a TERM-deaf child")
	}
	if sig := endingSignal(cmd); sig != syscall.SIGKILL {
		t.Fatalf("a TERM-deaf child ended by %v; WaitDelay should escalate to SIGKILL", sig)
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

// os/exec's contract: when Cancel ran and the child then exits 0, Run returns
// ctx.Err(). A client that was already quitting cleanly when the birth watch's
// cancel landed is such a child. What the launcher must act on is how the
// client ended -- the watch knows about the cancel itself -- so the handoff
// reports the child's own status whenever the child ran. Reporting the
// context error instead turned a clean quit into "failed to launch" and
// skipped its quit cleanup (#288 BR-2).
func TestCancelledHandoffReportsAChildsCleanExit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	trapped := filepath.Join(t.TempDir(), "trapped")
	cmd := killOnCancel(exec.CommandContext(ctx, "sh", "-c", `trap "exit 0" TERM; : > "$1"; while :; do sleep 0.05; done`, "sh", trapped))
	done := make(chan struct{})
	var code int
	var err error
	go func() { code, err = runBlockingHandoff(cmd); close(done) }()
	// Join before TempDir cleanup, including when readiness fails.
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-testwait.After(t):
			t.Error("the handoff never returned during cleanup")
		}
	})
	// A timer can signal a loaded shell before its clean-exit trap is set.
	testwait.Until(t, "the child to set its TERM trap", func() bool {
		select {
		case <-done:
			t.Fatalf("handoff ended before trap readiness: %d, %v", code, err)
		default:
		}
		_, err := os.Stat(trapped)
		return err == nil
	})
	cancel()
	select {
	case <-done:
	case <-testwait.After(t):
		t.Fatal("the handoff never returned from a clean-exit child")
	}
	if err != nil || code != 0 {
		t.Fatalf("handoff = %d, %v; want the child's own 0, nil", code, err)
	}
}

func TestHandoffThatCannotStartReportsTheError(t *testing.T) {
	cmd := killOnCancel(exec.CommandContext(context.Background(), "/nonexistent/zellij"))
	if code, err := runBlockingHandoff(cmd); err == nil || code != 1 {
		t.Fatalf("handoff = %d, %v; want 1 and the start error", code, err)
	}
}

// endingSignal is the signal that ended cmd's process, or 0 if it exited on
// its own: how a child ended, not how long it took, separates a kill from
// sleep 60 running out (#479).
func endingSignal(cmd *exec.Cmd) syscall.Signal {
	status, ok := cmd.ProcessState.Sys().(syscall.WaitStatus)
	if !ok || !status.Signaled() {
		return 0
	}
	return status.Signal()
}
