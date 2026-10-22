package launcher

import (
	"bytes"
	"context"
	"encoding/json"

	"errors"
	"fmt"
	"os"

	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/zellijquery"
	"github.com/xianxu/pair/cmd/internal/zellijreply"
)

type SessionOwnerIO interface {
	SessionServers(context.Context, string) ([]SessionServerIdentity, error)
	SessionPresent(context.Context, string) (bool, error)
	SessionPanes(context.Context, string) ([]byte, error)
	Socket(path string) SocketState
}

type SessionOwnerProbe struct {
	IO        SessionOwnerIO
	Processes procutil.Source
}

func (p SessionOwnerProbe) runtime() SessionOwnerIO {
	if p.IO != nil {
		return p.IO
	}
	return osSessionOwnerIO{processes: p.Processes}
}

func (p SessionOwnerProbe) Probe(ctx context.Context, name, globalDataDir, scope, tag string) (SessionOwnerObservation, error) {
	return p.probe(ctx, name, globalDataDir, scope, tag, p.runtime().SessionServers)
}

// probe varies only the initial evidence; every authority fence reads live IO.
func (p SessionOwnerProbe) probe(ctx context.Context, name, globalDataDir, scope, tag string, initial func(context.Context, string) ([]SessionServerIdentity, error)) (SessionOwnerObservation, error) {
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
	before, err := initial(ctx, name)
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
	// The existing exact-generation checks bracket the complete observation pair.
	// Confirmation never retries invalid first evidence or adds host table scans.
	const generationChanged = "session server changed during observation"
	var lastQuery ZellijQueryObservation
	var hasQuery bool
	read := func(ctx context.Context) ([]byte, error) {
		lastQuery, hasQuery = ZellijQueryObservation{ExitCode: -1}, false
		previous := zellijquery.Observer(ctx)
		ctx = WithZellijQueryObserver(ctx, func(event ZellijQueryObservation) {
			lastQuery, hasQuery = snapshotNativeQueryObservation(event), true
			if previous != nil {
				previous(event)
			}
		})
		raw, err := io.SessionPanes(ctx, name)
		if !hasQuery && len(raw) > 0 {
			lastQuery = snapshotNativeQueryObservation(ZellijQueryObservation{Stdout: raw, StdoutBytes: len(raw), ExitCode: -1})
			hasQuery = true
		}
		if err != nil {
			return nil, err
		}

		if len(bytes.TrimSpace(raw)) == 0 {
			return nil, ErrZellijNoReply
		}
		return raw, nil
	}
	var invalidObservation error
	raw, err := zellijreply.Confirm(ctx, read, func(raw []byte) (string, error) {
		projection, err := projectSessionOwnerReply(expected, raw)
		if err != nil {
			invalidObservation = err
			if !json.Valid(raw) {
				invalidObservation = errors.New("invalid live pane JSON")
			}
		}
		return projection, err
	})
	if err != nil && !errors.Is(err, zellijreply.ErrMismatch) && hasQuery {
		diagnostic := err.Error()
		if invalidObservation != nil {
			diagnostic = invalidObservation.Error()
		}
		result.DiagnosticCause = nativeQueryFailureEvidence(ctx, lastQuery, diagnostic)
	}
	if ctx.Err() != nil {
		return result, result.WithDiagnosticCause(ctx.Err())
	}
	after, generationErr := io.SessionServers(ctx, name)
	if generationErr != nil {
		return result, result.WithDiagnosticCause(generationErr)
	}
	if len(after) != 1 || before[0] != after[0] {
		result.Diagnostic = generationChanged
		return result, nil
	}
	if err != nil {
		if invalidObservation != nil {
			result.Diagnostic = invalidObservation.Error()
			return result, nil
		}
		if errors.Is(err, zellijreply.ErrMismatch) {
			result.Diagnostic = "session owner observations disagree"
			return result, nil
		}
		return result, fmt.Errorf("observe live session owner: %w", result.WithDiagnosticCause(err))
	}
	panes, err := zellijreply.ParsePanes(raw, zellijreply.PaneRequirements{})
	if err != nil {
		return result, err
	}
	result = ClassifySessionOwner(expected, panes)
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

type osSessionOwnerIO struct{ processes procutil.Source }

func (o osSessionOwnerIO) SessionServers(ctx context.Context, name string) ([]SessionServerIdentity, error) {
	return newOSSessionQuiescenceOps(o.processes).SessionServers(ctx, name)
}
func (osSessionOwnerIO) SessionPresent(ctx context.Context, name string) (bool, error) {
	raw, err := (ZellijSource{}).runContext(ctx, "list-sessions", "--no-formatting")
	if err != nil {
		return false, err
	}
	present, exited, err := sessionListingState(raw, name)
	return present && !exited, err
}
func (osSessionOwnerIO) Socket(path string) SocketState {
	return ObserveSocket(os.Lstat(path))
}
func (osSessionOwnerIO) SessionPanes(ctx context.Context, name string) ([]byte, error) {
	return (ZellijSource{}).runContext(ctx, "--session", name, "action", "list-panes", "--json", "--command")
}

func (r OSRuntime) ObserveSessionOwner(ctx context.Context, name, globalDataDir, scope, tag string) (SessionOwnerObservation, error) {
	return (SessionOwnerProbe{Processes: r.Processes}).Probe(ctx, name, globalDataDir, scope, tag)
}
func (r OSRuntime) RevalidateSessionOwner(ctx context.Context, observation SessionOwnerObservation) error {
	return (SessionOwnerProbe{Processes: r.Processes}).Revalidate(ctx, observation)
}
