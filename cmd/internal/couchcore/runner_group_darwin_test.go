//go:build darwin

package couchcore

import (
	"errors"
	"os/exec"
	"syscall"
	"testing"
)

// Pin the Darwin outcome seen in #542: the sole child is already reaped,
// but the process-group probe reports EPERM instead of ESRCH. Keep the
// membership observation real; only order the ambiguous syscall result.
func TestOwnedProcessGroupAmbiguousProbeAfterReaping(t *testing.T) {
	child := exec.Command("/bin/sh", "-c", "exit 0")
	child.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := child.Run(); err != nil {
		t.Fatal(err)
	}
	pid := child.Process.Pid
	if err := syscall.Kill(pid, 0); !errors.Is(err, syscall.ESRCH) {
		t.Fatalf("reaped child probe: %v", err)
	}
	if ownedProcessGroupProbeAlive(pid, func(group int, signal syscall.Signal) error {
		if group != -pid || signal != 0 {
			t.Fatalf("probe = (%d, %v)", group, signal)
		}
		return syscall.EPERM
	}, processGroupMembers) {
		t.Fatal("empty group reported alive after ambiguous Darwin EPERM")
	}
}

func TestOwnedProcessGroupAmbiguousProbePreservesLiveMember(t *testing.T) {
	child := exec.Command("/bin/sleep", "30")
	child.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = child.Process.Kill(); _ = child.Wait() }()
	if !ownedProcessGroupProbeAlive(child.Process.Pid, func(int, syscall.Signal) error { return syscall.EPERM }, processGroupMembers) {
		t.Fatal("live group member lost after EPERM")
	}
}
