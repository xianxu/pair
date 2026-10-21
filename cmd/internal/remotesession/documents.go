package remotesession

import (
	"context"
	"sync"
	"time"
)

type DocumentSummary struct {
	Ref   DocumentRef
	Title string
}
type DocumentSnapshot struct {
	Ref      DocumentRef
	Revision string
	Text     string
}
type DocumentResult struct {
	Outcome   string
	Revision  string
	ReceiptID string
}

// DocumentBackend is #758's registered-document adapter, not a generic nvim
// RPC or file service. ResolveNavigation must resolve eligible destinations
// before opening them. Every actual effect rechecks the exact ref/revision.
//
// Dispatch is a NONBLOCKING queue handoff. The supervisor calls it while holding
// admissionGate.mu, so it must not synchronously call Permit.EffectDoor or do IO.
// Its owned worker calls EffectDoor at the final nonblocking nvim/broker enqueue.
// A false handoff means no effect; an accepted effect may later be uncertain.
//
// Join quiesces only owned remote work/subscriptions; it never ends an editor.
// Backend workers and subscriptions must cancel/join before console reset. One
// bounded result is sent per accepted operation; callers never replay uncertainty.
// Subscribe permits one bounded stream per session, at most1MiB queued output;
// an oversized snapshot or slow subscriber is refused/closed, never accumulated.
type DocumentBackend interface {
	List(context.Context) ([]DocumentSummary, error)
	Read(context.Context, DocumentRef) (DocumentSnapshot, error)
	ResolveNavigation(context.Context, DocumentRef, NavigationTarget) (DocumentRef, error)
	Dispatch(context.Context, DocumentRequest, Permit) (<-chan DocumentResult, bool)
	Subscribe(context.Context, DocumentRef) (<-chan DocumentSnapshot, error)
	Join(context.Context) error
}

// admissionGate is the supervisor's single lifecycle/admission mutex. Permits
// share this exact state; no cached bool or independently mutable grant exists.
type admissionGate struct {
	mu    sync.Mutex
	state remoteState
}
type permitState struct {
	gate    *admissionGate
	session sessionRef
	ref     DocumentRef
	ctx     context.Context
	expires time.Time
	now     func() time.Time
	used    bool
}

// Permit has no public constructor or writable grant fields. Copies share the
// same one-use claim. JSON/text from a peer or webpage cannot construct one.
type Permit struct{ state *permitState }

func newPermit(gate *admissionGate, session sessionRef, ref DocumentRef, ctx context.Context, expires time.Time, now func() time.Time) Permit {
	return Permit{state: &permitState{gate: gate, session: session, ref: ref, ctx: ctx, expires: expires, now: now}}
}

// EffectDoor linearizes one effect against kill. enqueue must be nonblocking
// and return true ONLY if it handed off the effect; no disk/network wait may run
// in this callback. A successful handoff is never retrospectively rolled back.
func (p Permit) EffectDoor(ref DocumentRef, enqueue func() bool) bool {
	s := p.state
	if s == nil || s.gate == nil || s.ctx == nil || s.now == nil || s.session.done == nil || enqueue == nil {
		return false
	}
	s.gate.mu.Lock()
	defer s.gate.mu.Unlock()
	if s.used || ref != s.ref || !ref.valid() || s.gate.state.phase != phaseReady || s.gate.state.epoch != s.session.epoch || !s.now().Before(s.expires) || s.ctx.Err() != nil {
		return false
	}
	select {
	case <-s.session.done:
		return false
	default:
	}
	if !enqueue() {
		return false
	}
	s.used = true
	return true
}
