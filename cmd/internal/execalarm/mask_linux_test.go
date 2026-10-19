package execalarm

import (
	"golang.org/x/sys/unix"
	"runtime"
)

func blockAlarmForTest() error {
	runtime.LockOSThread()
	var mask unix.Sigset_t
	mask.Val[0] |= 1 << (unix.SIGALRM - 1)
	return unix.PthreadSigmask(unix.SIG_BLOCK, &mask, nil)
}
