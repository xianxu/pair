package couchmessage

import (
	"net"

	"golang.org/x/sys/unix"
)

// PeerPID is the PID of the process on the other end of a unix connection, as
// the kernel recorded it at connect time.
func PeerPID(conn *net.UnixConn) (int, error) {
	raw, err := conn.SyscallConn()
	if err != nil {
		return 0, err
	}
	var pid int
	var inner error
	if err := raw.Control(func(fd uintptr) {
		pid, inner = unix.GetsockoptInt(int(fd), unix.SOL_LOCAL, unix.LOCAL_PEERPID)
	}); err != nil {
		return 0, err
	}
	return pid, inner
}
