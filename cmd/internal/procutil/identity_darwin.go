//go:build darwin

package procutil

import (
	"strconv"

	"golang.org/x/sys/unix"
)

// Identity returns the kernel-recorded start timestamp for this incarnation of
// pid. It changes even when the OS recycles the same numeric PID quickly.
func Identity(pid string) string {
	n, ok := positivePID(pid)
	if !ok {
		return ""
	}
	info, err := unix.SysctlKinfoProc("kern.proc.pid", n)
	if err != nil || info.Proc.P_pid != int32(n) {
		return ""
	}
	started := info.Proc.P_starttime
	return strconv.FormatInt(started.Sec, 10) + "." + strconv.FormatInt(int64(started.Usec), 10)
}
