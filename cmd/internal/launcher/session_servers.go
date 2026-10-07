package launcher

import (
	"errors"
	"io/fs"
	"strconv"
	"strings"
)

// A zellij server's argv is `zellij --server <socket>`, and the socket's base
// name is the session name. The socket path is the one fact that tells a
// reachable server from an ORPHANED one -- alive, with its agent still running,
// but unreachable because the socket is gone (#399: a test deleted $TMPDIR and
// orphaned every server on the host).

// ParseServerProcesses reads `ps -axo pid=,command=` text and returns the exact
// zellij server rows, socket path included. Anything else is skipped.
func ParseServerProcesses(raw string) []SessionServerIdentity {
	var out []SessionServerIdentity
	for _, line := range strings.Split(raw, "\n") {
		fields := strings.Fields(line)
		if len(fields) != 4 {
			continue
		}
		socket := fields[3]
		session := socketSession(socket)
		if session == "" || !isExactZellijServerCommand(strings.Join(fields[1:], " "), session) {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil || pid <= 0 {
			continue
		}
		out = append(out, SessionServerIdentity{PID: pid, Session: session, Socket: socket})
	}
	return out
}

func socketSession(socket string) string {
	i := strings.LastIndexByte(socket, '/')
	return socket[i+1:]
}

// SocketState is what one Lstat says about a server's socket path.
type SocketState uint8

const (
	// SocketUnknown is the zero value: anything but a clear answer. It must
	// never make an orphan -- an unreadable socket directory would otherwise
	// turn every live server into one.
	SocketUnknown SocketState = iota
	SocketPresent
	// SocketGone is ENOENT and only ENOENT.
	SocketGone
)

func (s SocketState) String() string {
	switch s {
	case SocketPresent:
		return "present"
	case SocketGone:
		return "gone"
	}
	return "unknown"
}

// ObserveSocket is the pure reading of one Lstat result.
func ObserveSocket(info fs.FileInfo, err error) SocketState {
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return SocketGone
	case err != nil:
		return SocketUnknown
	case info != nil && info.Mode()&fs.ModeSocket != 0:
		return SocketPresent
	}
	return SocketUnknown
}

// ServerState is one session name's server verdict.
type ServerState struct {
	Server SessionServerIdentity
	// Orphaned: exactly one server for the name, and its socket is gone.
	Orphaned bool
	// Unresolved: the evidence cannot decide -- an unknown socket, or several
	// servers claiming one name. Callers must fail closed on it.
	Unresolved bool
}

// ClassifyServers is the orphan rule over one snapshot.
func ClassifyServers(servers []SessionServerIdentity, sockets map[string]SocketState) map[string]ServerState {
	count := map[string]int{}
	for _, s := range servers {
		count[s.Session]++
	}
	out := make(map[string]ServerState, len(servers))
	for _, s := range servers {
		state := ServerState{Server: s}
		switch {
		case count[s.Session] > 1:
			state = ServerState{Unresolved: true}
		case sockets[s.Socket] == SocketGone:
			state.Orphaned = true
		case sockets[s.Socket] != SocketPresent:
			state.Unresolved = true
		}
		out[s.Session] = state
	}
	return out
}
