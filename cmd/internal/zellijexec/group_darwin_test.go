//go:build darwin

package zellijexec

import (
	"context"
	"errors"
	"os/exec"
	"syscall"
	"testing"

	"github.com/xianxu/pair/cmd/internal/procutil"
)

func TestStopGroupResolvesDarwinAmbiguousSignal(t *testing.T) {
	t.Run("independently empty", func(t *testing.T) {
		cmd := exec.CommandContext(context.Background(), "/bin/sh", "-c", "exit 0")
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}
		if err := stopGroup(cmd.Process.Pid, func(group int, signal syscall.Signal) error {
			if group != -cmd.Process.Pid || signal != syscall.SIGKILL {
				t.Fatalf("signal wrong group: %d %v", group, signal)
			}
			return syscall.EPERM
		}, procutil.ProcessGroupMembers); err != nil {
			t.Fatalf("independently empty group: %v", err)
		}
	})
	t.Run("live member", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		input, gate := testPipe(t)
		cmd := exec.CommandContext(ctx, "/bin/cat")
		cmd.Stdin = input
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { cancel(); gate.Close(); _ = cmd.Wait() })
		if err := stopGroup(cmd.Process.Pid, func(int, syscall.Signal) error { return syscall.EPERM }, procutil.ProcessGroupMembers); !errors.Is(err, syscall.EPERM) {
			t.Fatalf("lost live group error: %v", err)
		}
	})
}
