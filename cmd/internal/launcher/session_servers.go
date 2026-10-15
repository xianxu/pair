package launcher

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
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
		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		if server, ok := parseServerCommand(pid, strings.Join(fields[1:], " ")); ok {
			out = append(out, server)
		}
	}
	return out
}

func parseServerCommand(pid int, command string) (SessionServerIdentity, bool) {
	fields := strings.Fields(command)
	if pid <= 0 || len(fields) != 3 {
		return SessionServerIdentity{}, false
	}
	socket := fields[2]
	session := socketSession(socket)
	if session == "" || !isExactZellijServerCommand(command, session) {
		return SessionServerIdentity{}, false
	}
	return SessionServerIdentity{PID: pid, Session: session, Socket: socket}, true
}

// serverProcesses preserves unavailable candidate evidence; an unreadable
// zellij cannot establish that no server exists. Successful argv remains the
// matching authority, including processes whose argv[0] was changed.
func serverProcesses(rows []procutil.Observation) ([]SessionServerIdentity, error) {
	var out []SessionServerIdentity
	for _, row := range rows {
		if row.Zombie || procutil.IsProcessGone(row.Err) {
			continue
		}
		if row.Err != nil {
			if row.Name == "zellij" || row.Name == "" {
				return nil, fmt.Errorf("process inspection unavailable for possible zellij server %d: %w", row.PID, row.Err)
			}
			continue
		}
		server, ok := parseServerCommand(row.PID, strings.Join(row.Args, " "))
		if !ok {
			continue
		}
		if row.Identity == "" {
			return nil, fmt.Errorf("process identity unavailable for zellij server %d", row.PID)
		}
		server.Identity = row.Identity
		out = append(out, server)
	}
	return out, nil
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
// server on the host with its socket verdict, by session name. One native snapshot plus an
// Lstat per server -- never a probe per thread.
type ServerStates interface {
	ServerStates(ctx context.Context) (map[string]ServerState, error)
}

// OSServerStates reads the host. An orphan's start identity is read too, since
// reap must act on that exact generation; an orphan whose identity cannot be
// read is unresolved, not orphaned.
type OSServerStates struct{ Processes procutil.Source }

func (s OSServerStates) ServerStates(ctx context.Context) (map[string]ServerState, error) {
	rows, err := s.Processes.SnapshotNamed(ctx, "zellij")
	if err != nil {
		return nil, err
	}
	servers, err := serverProcesses(rows)
	if err != nil {
		return nil, err
	}
	sockets := make(map[string]SocketState, len(servers))
	for _, server := range servers {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		sockets[server.Socket] = ObserveSocket(os.Lstat(server.Socket))
	}
	return ClassifyServers(servers, sockets), nil
}
