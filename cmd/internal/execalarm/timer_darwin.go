package execalarm

import (
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

func armTimer(seconds int) error {
	timer := unix.Itimerval{Interval: unix.Timeval{Sec: int64(seconds)}, Value: unix.Timeval{Sec: int64(seconds)}}
	_, _, err := syscall.Syscall(syscall.SYS_SETITIMER, 0, uintptr(unsafe.Pointer(&timer)), 0)
	runtime.KeepAlive(&timer)
	if err != 0 {
		return err
	}
	return nil
}

func unblockAlarm() error {
	mask := uint32(1 << (syscall.SIGALRM - 1))
	_, _, err := syscall.Syscall(syscall.SYS_SIGPROCMASK, 2, uintptr(unsafe.Pointer(&mask)), 0)
	runtime.KeepAlive(&mask)
	if err != 0 {
		return err
	}
	return nil
}
