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
	var cred *unix.Ucred
	var inner error
	if err := raw.Control(func(fd uintptr) {
		cred, inner = unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED)
	}); err != nil {
		return 0, err
	}
	if inner != nil {
		return 0, inner
	}
	return int(cred.Pid), nil
}
