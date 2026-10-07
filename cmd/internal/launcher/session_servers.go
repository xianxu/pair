package launcher

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/xianxu/pair/cmd/internal/procutil"
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

// ServerVerdict is what the snapshot says about one session name's server. A
// tagged value rather than flags, so contradictory combinations (orphaned AND
// unresolved) cannot be represented (#399 M1 review).
type ServerVerdict uint8

const (
	// ServerUnresolved is the zero value: the evidence cannot decide (an
	// unreadable socket, or an orphan whose identity cannot be read). Callers
	// fail closed on it.
	ServerUnresolved ServerVerdict = iota
	// ServerReachable: one server, socket present.
	ServerReachable
	// ServerOrphaned: one server, socket gone (ENOENT).
	ServerOrphaned
	// ServerContested: several servers claim the name. It outranks a live
	// list-sessions row: an orphan and a fresh server sharing a name must not
	// read as one healthy session.
	ServerContested
)

// ServerState is one session name's server verdict. A name with no server is
// absent from the map, never a zero ServerState.
type ServerState struct {
	Server  SessionServerIdentity
	Verdict ServerVerdict
}

// ClassifyServers is the orphan rule over one snapshot.
func ClassifyServers(servers []SessionServerIdentity, sockets map[string]SocketState) map[string]ServerState {
	count := map[string]int{}
	for _, s := range servers {
		count[s.Session]++
	}
	out := make(map[string]ServerState, len(servers))
	for _, s := range servers {
		state := ServerState{Server: s, Verdict: ServerUnresolved}
		switch {
		case count[s.Session] > 1:
			state = ServerState{Verdict: ServerContested}
		case sockets[s.Socket] == SocketGone:
			state.Verdict = ServerOrphaned
		case sockets[s.Socket] == SocketPresent:
			state.Verdict = ServerReachable
		}
		out[s.Session] = state
	}
	return out
}

// ServerStates is the one bulk server observation for a refresh: every zellij
// server on the host with its socket verdict, by session name. One `ps` plus an
// Lstat per server -- never a probe per thread.
type ServerStates interface {
	ServerStates(ctx context.Context) (map[string]ServerState, error)
}

// OSServerStates reads the host. An orphan's start identity is read too, since
// reap must act on that exact generation; an orphan whose identity cannot be
// read is unresolved, not orphaned.
type OSServerStates struct{}

func (OSServerStates) ServerStates(ctx context.Context) (map[string]ServerState, error) {
	out, err := exec.CommandContext(ctx, "ps", "-axo", "pid=,command=").Output()
	if err != nil {
		return nil, err
	}
	servers := ParseServerProcesses(string(out))
	sockets := make(map[string]SocketState, len(servers))
	for _, s := range servers {
		sockets[s.Socket] = ObserveSocket(os.Lstat(s.Socket))
	}
	states := ClassifyServers(servers, sockets)
	for name, state := range states {
		if state.Verdict != ServerOrphaned {
			continue
		}
		state.Server.Identity = procutil.Identity(strconv.Itoa(state.Server.PID))
		if state.Server.Identity == "" {
			state = ServerState{Verdict: ServerUnresolved}
		}
		states[name] = state
	}
	return states, nil
}
