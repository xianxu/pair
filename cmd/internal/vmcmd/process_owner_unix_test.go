//go:build darwin || linux

package vmcmd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

func TestOwnedProcessPinsGenerationUntilFinalSignal(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", "printf final; exit 7")
	owner := configureSnapshotCommand(cmd)
	var out bytes.Buffer
	cmd.Stdout = owner.Writer(&out)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	if err := owner.AwaitTerminal(ctx); err != nil {
		t.Fatal(err)
	}
	if cmd.ProcessState != nil {
		t.Fatal("terminal child reaped before final group signal")
	}
	signals := 0
	realSignal := owner.signalGroup
	owner.signalGroup = func(pid int) error {
		signals++
		if cmd.ProcessState != nil {
			t.Error("signal after PID pin released")
		}
		return realSignal(pid)
	}
	waitErr, cleanupErr := owner.Finish(ctx)
	var exit *exec.ExitError
	if cleanupErr != nil || !errors.As(waitErr, &exit) || exit.ExitCode() != 7 {
		t.Fatalf("lost parent result: %v %v", waitErr, cleanupErr)
	}
	if out.String() != "final" {
		t.Fatalf("output not joined: %q", out.String())
	}
	if err := owner.Signal(); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Cancel(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		t.Fatal(err)
	}
	_, _ = owner.Writer(alwaysFailWriter{}).Write([]byte("late output"))
	_, err := owner.Finish(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if signals != 1 {
		t.Fatalf("signalled replacement generation: %d", signals)
	}
}

func TestOwnedProcessWriterFailureTerminatesPinnedChild(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", "printf overflow; while :; do :; done")
	owner := configureSnapshotCommand(cmd)
	cmd.Stdout = owner.Writer(alwaysFailWriter{})
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	err := owner.AwaitTerminal(ctx)
	if !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("lost output failure: %v", err)
	}
	_, cleanupErr := owner.Finish(ctx)
	if cleanupErr != nil {
		t.Fatal(cleanupErr)
	}
}

type alwaysFailWriter struct{}

func (alwaysFailWriter) Write([]byte) (int, error) { return 0, io.ErrShortWrite }

func TestOwnedProcessAmbiguousFinalSignalNeedsPositiveAbsence(t *testing.T) {
	for _, gone := range []bool{true, false} {
		t.Run(map[bool]string{true: "gone", false: "unknown"}[gone], func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, "/bin/sh", "-c", "exit 0")
			owner := configureSnapshotCommand(cmd)
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			if err := owner.AwaitTerminal(ctx); err != nil {
				t.Fatal(err)
			}
			owner.signalGroup = func(int) error { return syscall.EPERM }
			owner.groupPresent = func(int) (bool, error) {
				if gone {
					return false, nil
				}
				return false, syscall.EIO
			}
			_, err := owner.Finish(ctx)
			if gone && err != nil {
				t.Fatal(err)
			}
			if !gone && !errors.Is(err, ErrUnconfirmedProcessCleanup) {
				t.Fatalf("unknown accepted: %v", err)
			}
		})
	}
}

func TestOwnedProcessPreservesTerminalResultAcrossCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", "exit 7")
	owner := configureSnapshotCommand(cmd)
	if err := cmd.Start(); err != nil {
		cancel()
		t.Fatal(err)
	}
	bound, stop := context.WithTimeout(context.Background(), time.Second)
	defer stop()
	for {
		terminal, err := nativeChildTerminal(cmd.Process.Pid)
		if err != nil {
			t.Fatal(err)
		}
		if terminal {
			break
		}
		select {
		case <-bound.Done():
			t.Fatal("child not terminal")
		case <-time.After(time.Millisecond):
		}
	}
	cancel()
	if err := owner.AwaitTerminal(ctx); err != nil {
		t.Errorf("cancellation replaced terminal parent result: %v", err)
	}
	waitErr, cleanupErr := owner.Finish(bound)
	var exit *exec.ExitError
	if cleanupErr != nil || !errors.As(waitErr, &exit) || exit.ExitCode() != 7 {
		t.Fatalf("terminal result not preserved: %v %v", waitErr, cleanupErr)
	}
}
