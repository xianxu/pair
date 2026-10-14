//go:build darwin

package procutil

import (
	"golang.org/x/sys/unix"
	"testing"
)

func TestDarwinUnreapedChildStatus(t *testing.T) {
	pid := exitedUnreapedChild(t)
	info, err := unix.SysctlKinfoProc("kern.proc.pid", pid)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("exited unreaped PID %d: kernel P_stat=%d (sys/proc.h SZOMB=5)", pid, info.Proc.P_stat)
	if info.Proc.P_stat != 5 {
		t.Fatalf("unexpected terminated status %d", info.Proc.P_stat)
	}
}
