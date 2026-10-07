package launcher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/xianxu/pair/cmd/internal/zellijpane"
)

type SessionOwnerIO interface {
	SessionServers(context.Context, string) ([]SessionServerIdentity, error)
	SessionPresent(context.Context, string) (bool, error)
	SessionPanes(context.Context, string) ([]byte, error)
	Socket(path string) SocketState
}

type SessionOwnerProbe struct{ IO SessionOwnerIO }

func (p SessionOwnerProbe) runtime() SessionOwnerIO {
	if p.IO != nil {
		return p.IO
	}
	return osSessionOwnerIO{}
}

func (p SessionOwnerProbe) Probe(ctx context.Context, name, globalDataDir, scope, tag string) (SessionOwnerObservation, error) {
	result := SessionOwnerObservation{Name: name}
	expected, err := sessionExpectedOwner(globalDataDir, scope, tag)
	if err != nil {
		return result, err
	}
	if name == "" {
		return result, errors.New("empty session name")
	}
	ctx, cancel := context.WithTimeout(ctx, zellijQueryTimeout)
	defer cancel()
	io := p.runtime()
	before, err := io.SessionServers(ctx, name)
	if err != nil {
		return result, fmt.Errorf("observe session server: %w", err)
	}
	if len(before) == 0 {
		present, err := io.SessionPresent(ctx, name)
		if err != nil {
			return result, err
		}
		after, err := io.SessionServers(ctx, name)
		if err != nil {
			return result, err
		}
		if !present && len(after) == 0 {
			result.State = SessionOwnerAbsent
			return result, nil
		}
		result.Diagnostic = "live session has no stable exact server identity"
		return result, nil
	}
	if len(before) != 1 || before[0].PID <= 0 || before[0].Identity == "" || before[0].Session != name {
		result.Diagnostic = "ambiguous session server identity"
		return result, nil
	}
	// Ask the socket before the server: an orphan has no socket to answer
	// list-panes, and its failure used to surface as a raw exit status (#399).
	switch io.Socket(before[0].Socket) {
	case SocketGone:
		result.State = SessionOwnerOrphaned
		result.Server = before[0]
		result.Diagnostic = OrphanDiagnostic(name, before[0].PID)
		return result, nil
	case SocketUnknown:
		result.Diagnostic = "session server socket unreadable"
		return result, nil
	}
	raw, err := io.SessionPanes(ctx, name)
	if err != nil {
		return result, fmt.Errorf("observe live session panes: %w", err)
	}
	after, err := io.SessionServers(ctx, name)
	if err != nil {
		return result, err
	}
	if len(after) != 1 || before[0] != after[0] {
		result.Diagnostic = "session server changed during observation"
		return result, nil
	}
	if !json.Valid(raw) {
		result.Diagnostic = "invalid live pane JSON"
		return result, nil
	}
	result = ClassifySessionOwner(expected, zellijpane.Parse(raw))
	result.Name = name
	result.Server = before[0]
	return result, nil
}

func (p SessionOwnerProbe) Revalidate(ctx context.Context, observation SessionOwnerObservation) error {
	if observation.State != SessionOwnerOwned || observation.Server.PID <= 0 || observation.Server.Identity == "" || observation.Server.Session != observation.Name {
		return errors.New("session ownership was not positively established")
	}
	ctx, cancel := context.WithTimeout(ctx, zellijQueryTimeout)
	defer cancel()
	servers, err := p.runtime().SessionServers(ctx, observation.Name)
	if err != nil {
		return err
	}
	if len(servers) != 1 || servers[0] != observation.Server {
		return fmt.Errorf("session %q server generation changed", observation.Name)
	}
	return nil
}

type osSessionOwnerIO struct{}

func (osSessionOwnerIO) SessionServers(ctx context.Context, name string) ([]SessionServerIdentity, error) {
	return newOSSessionQuiescenceOps().SessionServers(ctx, name)
}
func (osSessionOwnerIO) SessionPresent(ctx context.Context, name string) (bool, error) {
	raw, err := (ZellijSource{}).runContext(ctx, "list-sessions", "--no-formatting")
	if err != nil {
		return false, err
	}
	present, exited := sessionRowState(string(raw), name)
	return present && !exited, nil
}
func (osSessionOwnerIO) Socket(path string) SocketState {
	return ObserveSocket(os.Lstat(path))
}
func (osSessionOwnerIO) SessionPanes(ctx context.Context, name string) ([]byte, error) {
	return (ZellijSource{}).runContext(ctx, "--session", name, "action", "list-panes", "--json", "--command")
}

func (r OSRuntime) ObserveSessionOwner(ctx context.Context, name, globalDataDir, scope, tag string) (SessionOwnerObservation, error) {
	return (SessionOwnerProbe{}).Probe(ctx, name, globalDataDir, scope, tag)
}
func (r OSRuntime) RevalidateSessionOwner(ctx context.Context, observation SessionOwnerObservation) error {
	return (SessionOwnerProbe{}).Revalidate(ctx, observation)
}
