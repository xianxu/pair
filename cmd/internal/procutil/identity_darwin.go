//go:build darwin

package procutil

import (
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
	return darwinIdentity(pid, unix.SysctlRaw)
}

// Read the response length before decoding: SysctlKinfoProc collapses a
// successful zero-byte (absent PID) response into EIO, losing absence evidence.
func darwinIdentity(pid int, read func(string, ...int) ([]byte, error)) (string, error) {
	raw, err := read("kern.proc.pid", pid)
	if err != nil {
		return "", err
	}
	if len(raw) == 0 {
		return "", unix.ESRCH
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
