package launcher

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"sync"
)

// SessionOwnerPass shares only initial server evidence. After-pane checks and
// Revalidate always use the probe's live IO; a captured map never grants effects.
type SessionOwnerPass struct {
	parent context.Context
	source ServerStates
	probe  SessionOwnerProbe
	once   sync.Once
	done   chan struct{}
	states map[string]ServerState
	err    error
}

// NewSessionOwnerPass creates a lazy capture bounded by parent cancellation and
// the normal query timeout. Canceling a caller stops its wait, leaving capture
// available for siblings. The parent must end with the owning operation/pass.
// A nil source uses native server discovery; probe retains its native/fake IO.
func NewSessionOwnerPass(parent context.Context, source ServerStates, probe SessionOwnerProbe) *SessionOwnerPass {
	if source == nil {
		source = OSServerStates{Processes: probe.Processes}
	}
	return &SessionOwnerPass{parent: parent, source: source, probe: probe, done: make(chan struct{})}
}

func (p *SessionOwnerPass) capture() {
	ctx, cancel := context.WithTimeout(p.parent, zellijQueryTimeout)
	defer cancel()
	states, err := p.source.ServerStates(ctx)
	if err == nil {
		err = ctx.Err()
	}
	if err == nil {
		p.states = maps.Clone(states)
	}
	p.err = err
	close(p.done)
}

func (p *SessionOwnerPass) initial(ctx context.Context, name string) ([]SessionServerIdentity, error) {
	if err := p.canceled(ctx); err != nil {
		return nil, err
	}
	p.once.Do(func() { go p.capture() })
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-p.parent.Done():
		return nil, p.parent.Err()
	case <-p.done:
	}
	if err := p.canceled(ctx); err != nil {
		return nil, err
	}
	if p.err != nil {
		return nil, p.err
	}
	state, ok := p.states[name]
	if !ok {
		return nil, nil
	}
	switch state.Verdict {
	case ServerReachable, ServerOrphaned:
		return []SessionServerIdentity{state.Server}, nil
	default:
		return nil, fmt.Errorf("session %q initial server evidence unresolved or contested", name)
	}
}

func (p *SessionOwnerPass) canceled(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return p.parent.Err()
}

func (p *SessionOwnerPass) callerContext(ctx context.Context) (context.Context, func()) {
	ctx, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(p.parent, cancel)
	return ctx, func() { stop(); cancel() }
}

func (p *SessionOwnerPass) Probe(ctx context.Context, name, globalDataDir, scope, tag string) (SessionOwnerObservation, error) {
	ctx, cancel := p.callerContext(ctx)
	defer cancel()
	got, err := p.probe.probe(ctx, name, globalDataDir, scope, tag, p.initial)
	if canceled := p.canceled(ctx); canceled != nil {
		if cause := errors.Join(err, got.DiagnosticCause); cause != nil {
			return SessionOwnerObservation{Name: name}, &diagnosticCauseError{error: canceled, cause: cause}
		}
		return SessionOwnerObservation{Name: name}, canceled
	}
	return got, err
}

func (p *SessionOwnerPass) Revalidate(ctx context.Context, observation SessionOwnerObservation) error {
	ctx, cancel := p.callerContext(ctx)
	defer cancel()
	if err := p.canceled(ctx); err != nil {
		return err
	}
	err := p.probe.Revalidate(ctx, observation)
	if canceled := p.canceled(ctx); canceled != nil {
		return canceled
	}
	return err
}
