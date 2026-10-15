//go:build darwin

package procutil

import (
	"errors"
	"fmt"
	"strconv"
	"unsafe"

	"golang.org/x/sys/unix"
)

// Identity returns the kernel-recorded start timestamp for this incarnation of
// pid. It changes even when the OS recycles the same numeric PID quickly.
func Identity(pid string) string {
	n, ok := positivePID(pid)
	if !ok {
		return ""
	}
	identity, _ := nativeIdentity(n)
	return identity
}

func nativeIdentity(pid int) (string, error) {
	return darwinIdentity(pid, unix.SysctlRaw, func(pid int) error { return unix.Kill(pid, 0) })
}

// Read the response length before decoding: SysctlKinfoProc collapses a
// successful zero-byte response into EIO. XNU also omits SIDL/shadow processes,
// so only an independent signal-zero ESRCH confirms that an empty result is gone.
func darwinIdentity(pid int, read func(string, ...int) ([]byte, error), exists func(int) error) (string, error) {
	raw, err := read("kern.proc.pid", pid)
	if err != nil {
		return "", err
	}
	if len(raw) == 0 {
		probeErr := exists(pid)
		if errors.Is(probeErr, unix.ESRCH) {
			return "", unix.ESRCH
		}
		return "", fmt.Errorf("empty process identity; existence check: %v: %w", probeErr, unix.EIO)
	}
	if len(raw) != unix.SizeofKinfoProc {
		return "", unix.EIO
	}
	var info unix.KinfoProc
	copy((*[unix.SizeofKinfoProc]byte)(unsafe.Pointer(&info))[:], raw)
	if info.Proc.P_pid != int32(pid) {
		return "", unix.EIO
	}
	started := info.Proc.P_starttime
	birth := strconv.FormatInt(started.Sec, 10) + "." + strconv.FormatInt(int64(started.Usec), 10)
	if info.Proc.P_stat == darwinZombieStatus {
		// Preserve the public Identity token while snapshots omit terminal rows.
		return birth, unix.ESRCH
	}
	return birth, nil
}
