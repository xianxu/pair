//go:build darwin

package vmcmd

import (
	"fmt"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"golang.org/x/sys/unix"
)

func nativeChildTerminal(pid int) (bool, error) {
	info, err := unix.SysctlKinfoProc("kern.proc.pid", pid)
	if err != nil {
		return false, err
	}
	if info == nil || int(info.Proc.P_pid) != pid {
		return false, fmt.Errorf("owned child %d observation unavailable", pid)
	}
	return info.Proc.P_stat == 5, nil // Darwin SZOMB; observation does not reap.
}

func nativeGroupPresent(pid int) (bool, error) { return procutil.ProcessGroupMembers(pid) }
