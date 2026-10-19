package execalarm

import (
	"runtime"
	"syscall"
	"unsafe"
)

func blockAlarmForTest() error {
	runtime.LockOSThread()
	mask := uint32(1 << (syscall.SIGALRM - 1))
	_, _, err := syscall.Syscall(syscall.SYS_SIGPROCMASK, 1, uintptr(unsafe.Pointer(&mask)), 0)
	runtime.KeepAlive(&mask)
	if err != 0 {
		return err
	}
	return nil
}
