package couchcore

import (
	"context"
	"fmt"
	"sync"

	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

type fakeArtifactCollision struct {
	collision bool
	err       error
}

type FakeThreadArtifactCollisionChecker struct {
	mu     sync.Mutex
	values map[ThreadAddress]fakeArtifactCollision
	calls  []ThreadAddress
	// detachedQueries counts DetachedSessions calls, so a test can pin an IO
	// BUDGET rather than only a result -- #198's guard is required to add no
	// session enumeration of its own, and a budget nothing measures is a rule
	// that cannot fail.
	detachedQueries    int
	released           []ThreadAddress
	registrations      map[ThreadAddress]fakeRegistration
	autoEstablish      bool
	quiesced           []ThreadAddress
	bindingResolutions int
	detachedCandidates int
	QuiesceHook        func(ThreadAddress) error
	pairSessions       map[ThreadAddress]PairSessionBinding
	nativeBindings     map[nativeBindingKey]NativeBindingResolution
	triggeredQuit      []TriggeredQuit
	TriggerQuitHook    func(string, launcher.QuitIntent) error
	// BeforeRegistration lets an integration test interleave a durable state
	// change at the registration boundary. It is called outside mu because the
	// hook may consult another stateful fake or call back into this one.
	BeforeRegistration func(ThreadAddress) error
	BeforePairSession  func(ThreadAddress) error

	detachedSessions map[ThreadAddress]string
	// DetachedSessionsHook lets a test fail the observation, or interleave a
	// durable change at the moment the projector asks who is detached.
	DetachedSessionsHook func([]ThreadAddress) error

	sessionPresence map[ThreadAddress]SessionObservation
	// presenceQueries counts SessionPresence calls, mirroring detachedQueries:
	// #256 gathers presence for EVERY record, so "one host-wide call, not one
	// per record" is a budget a test must be able to pin rather than trust.
	presenceQueries int
	// SessionPresenceHook lets a test fail the observation. A failure must leave
	// every thread UNRESOLVED, never "no session".
	SessionPresenceHook func([]ThreadAddress) error
}

// SetSessionPresence declares what the host's zellij sessions say about one
// address. An address never set is absent from the answer, so it reads the zero
// value -- unresolved -- which is what an unasked question must look like.
func (f *FakeThreadArtifactCollisionChecker) SetSessionPresence(address ThreadAddress, observation SessionObservation) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.sessionPresence == nil {
		f.sessionPresence = map[ThreadAddress]SessionObservation{}
	}
	f.sessionPresence[address] = observation
}

// SessionPresenceQueries is the IO budget: how many times the host was asked.
func (f *FakeThreadArtifactCollisionChecker) SessionPresenceQueries() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.presenceQueries
}

func (f *FakeThreadArtifactCollisionChecker) SessionPresence(ctx context.Context, addresses []ThreadAddress) (map[ThreadAddress]SessionObservation, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f.mu.Lock()
	f.presenceQueries++
	f.mu.Unlock()
	if hook := f.SessionPresenceHook; hook != nil {
		if err := hook(addresses); err != nil {
			return nil, err
		}
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make(map[ThreadAddress]SessionObservation, len(addresses))
	for _, address := range addresses {
		if observation, ok := f.sessionPresence[address]; ok {
			out[address] = observation
		}
	}
	return out, nil
}

type nativeBindingKey struct {
	Address ThreadAddress
	Agent   string
}

type TriggeredQuit struct {
	Session string
	Intent  launcher.QuitIntent
}

type fakeRegistration struct {
	evidence RegistrationEvidence
	err      error
}

func NewFakeThreadArtifactCollisionChecker() *FakeThreadArtifactCollisionChecker {
	return &FakeThreadArtifactCollisionChecker{
		values:           map[ThreadAddress]fakeArtifactCollision{},
		registrations:    map[ThreadAddress]fakeRegistration{},
		pairSessions:     map[ThreadAddress]PairSessionBinding{},
		nativeBindings:   map[nativeBindingKey]NativeBindingResolution{},
		detachedSessions: map[ThreadAddress]string{},
	}
}

// SetDetachedSession marks one thread as having a live zellij session with no
// client attached. An empty name clears it.
func (f *FakeThreadArtifactCollisionChecker) SetDetachedSession(address ThreadAddress, sessionName string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if sessionName == "" {
		delete(f.detachedSessions, address)
		return
	}
	f.detachedSessions[address] = sessionName
}

// DetachedSessions answers only for addresses the caller asked about, exactly
// as the real checker does -- a fake that answered for the whole world would
// hide a caller that forgot to pass its candidates.
func (f *FakeThreadArtifactCollisionChecker) DetachedSessions(ctx context.Context, candidates []DetachedCandidate) ([]DetachedSessionObservation, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f.mu.Lock()
	f.detachedQueries++
	f.detachedCandidates += len(candidates)
	f.mu.Unlock()
	if hook := f.DetachedSessionsHook; hook != nil {
		addresses := make([]ThreadAddress, 0, len(candidates))
		for _, candidate := range candidates {
			addresses = append(addresses, candidate.Address)
		}
		if err := hook(addresses); err != nil {
			return nil, err
		}
	}
	f.mu.Lock()
	defer f.mu.Unlock()

	// Through the SAME pure rule production uses, rather than a map lookup that
	// answers whatever the test set up (ARCH-MOCK). The duplicate-name refusal
	// lives in ProjectDetachedSessions, so a fake that skipped it would let
	// every fake-backed test pass on a proof production would refuse.
	bindings := make([]SessionNameBinding, 0, len(candidates))
	sessions := make([]launcher.Session, 0, len(candidates))
	for _, candidate := range candidates {
		name := f.detachedSessions[candidate.Address]
		if name == "" {
			continue
		}
		bindings = append(bindings, SessionNameBinding{
			Address: candidate.Address, SessionName: name,
			Agent: candidate.Agent,
		})
		sessions = append(sessions, launcher.Session{Name: name, State: launcher.SessionDetached})
	}
	// Claims come from every thread this fake knows about, not just the ones
	// asked for, counted by the same function production uses. The fake's map is
	// already one binding per thread, so it has no union to take.
	return ProjectDetachedSessions(bindings, sessions, claimsFromBindings(f.detachedSessions))
}

func (f *FakeThreadArtifactCollisionChecker) SetNativeBinding(address ThreadAddress, agent string, status sessioninventory.BindingStatus, nativeID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nativeBindings[nativeBindingKey{Address: address, Agent: agent}] = NativeBindingResolution{Status: status, NativeID: nativeID}
}

func (f *FakeThreadArtifactCollisionChecker) ResolveEstablished(_ context.Context, repoScope, tag, agent string) (NativeBindingResolution, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.bindingResolutions++
	resolution, ok := f.nativeBindings[nativeBindingKey{Address: ThreadAddress{RepoScope: repoScope, Tag: ThreadTag(tag)}, Agent: agent}]
	if !ok {
		resolution.Status = sessioninventory.BindingUnbound
	}
	if code := bindingResumeDiagnostic(resolution); code != "" {
		// The fake refuses through the SAME constructor as the real resolver, so
		// a test can never pass on wording production would not produce.
		return resolution, refuseBinding(code)
	}
	return resolution, nil
}

func (f *FakeThreadArtifactCollisionChecker) SetPairSession(address ThreadAddress, name string, present bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.pairSessions[address] = PairSessionBinding{Name: name, Present: present}
}

func (f *FakeThreadArtifactCollisionChecker) PairSessionContext(ctx context.Context, address ThreadAddress) (PairSessionBinding, error) {
	if err := ctx.Err(); err != nil {
		return PairSessionBinding{}, err
	}
	binding, err := f.PairSession(address)
	if err == nil {
		err = ctx.Err()
	}
	return binding, err
}

func (f *FakeThreadArtifactCollisionChecker) PairSession(address ThreadAddress) (PairSessionBinding, error) {
	f.mu.Lock()
	hook := f.BeforePairSession
	f.mu.Unlock()
	if hook != nil {
		if err := hook(address); err != nil {
			return PairSessionBinding{}, err
		}
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	binding, ok := f.pairSessions[address]
	if !ok || binding.Name == "" {
		return PairSessionBinding{}, fmt.Errorf("exact Pair session binding is absent for %+v", address)
	}
	return binding, nil
}

func (f *FakeThreadArtifactCollisionChecker) TriggerQuit(session string, intent launcher.QuitIntent) error {
	f.mu.Lock()
	f.triggeredQuit = append(f.triggeredQuit, TriggeredQuit{Session: session, Intent: intent})
	hook := f.TriggerQuitHook
	f.mu.Unlock()
	if hook != nil {
		return hook(session, intent)
	}
	return nil
}

func (f *FakeThreadArtifactCollisionChecker) TriggeredQuits() []TriggeredQuit {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]TriggeredQuit(nil), f.triggeredQuit...)
}

func (f *FakeThreadArtifactCollisionChecker) Set(address ThreadAddress, collision bool, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.values[address] = fakeArtifactCollision{collision: collision, err: err}
}

func (f *FakeThreadArtifactCollisionChecker) Claim(address ThreadAddress) (ThreadArtifactClaim, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, address)
	value := f.values[address]
	if value.err != nil {
		return nil, value.err
	}
	if value.collision {
		return nil, launcher.ErrThreadAddressClaimed
	}
	return noopThreadArtifactClaim{}, nil
}

func (f *FakeThreadArtifactCollisionChecker) Release(address ThreadAddress) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.released = append(f.released, address)
	return nil
}

func (f *FakeThreadArtifactCollisionChecker) SetRegistration(address ThreadAddress, evidence RegistrationEvidence, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.registrations[address] = fakeRegistration{evidence: evidence, err: err}
}

func (f *FakeThreadArtifactCollisionChecker) AutoEstablish(enabled bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.autoEstablish = enabled
}

func (f *FakeThreadArtifactCollisionChecker) Registration(address ThreadAddress) (RegistrationEvidence, error) {
	f.mu.Lock()
	hook := f.BeforeRegistration
	f.mu.Unlock()
	if hook != nil {
		if err := hook(address); err != nil {
			return RegistrationUnknown, err
		}
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if value, ok := f.registrations[address]; ok {
		return value.evidence, value.err
	}
	if f.autoEstablish {
		return RegistrationEstablished, nil
	}
	return RegistrationAbsent, nil
}

// Quiesce models what production quiescing DOES, not merely that it was asked
// for: launcher.QuiesceThreadSession runs `zellij delete-session --force` and
// kills that session's server, so the session stops existing.
//
// Recording the call alone made every later observation blind to it (pair#230):
// a test could quiesce a thread's session and still observe the thread as
// detached and resumable, so an assertion that a failed warm reattach LEFT its
// session alone passed whether or not the session had been deleted. A fake that
// is laxer than production hides exactly the bug it is standing in for.
//
// The call log is kept -- some callers assert the request, not the effect.
func (f *FakeThreadArtifactCollisionChecker) Quiesce(address ThreadAddress) error {
	f.mu.Lock()
	f.quiesced = append(f.quiesced, address)
	hook := f.QuiesceHook
	f.mu.Unlock()
	if hook != nil {
		// A hook that refuses models a quiesce that did not take effect, so the
		// session survives -- the retry loop's whole reason for existing.
		if err := hook(address); err != nil {
			return err
		}
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.detachedSessions, address)
	if binding, ok := f.pairSessions[address]; ok {
		binding.Present = false
		f.pairSessions[address] = binding
	}
	return nil
}

func (f *FakeThreadArtifactCollisionChecker) Quiesces() []ThreadAddress {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]ThreadAddress{}, f.quiesced...)
}

func (f *FakeThreadArtifactCollisionChecker) Releases() []ThreadAddress {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]ThreadAddress{}, f.released...)
}

// DetachedQueries is how many times the detached-session question was asked.
// BindingResolutions counts ResolveEstablished calls. Startup's cost is not
// only its zellij queries: each resolution reads that thread's own ledger, so a
// count that grows with the store is the same defect shape in a different
// currency (pair#206 M1).
func (f *FakeThreadArtifactCollisionChecker) BindingResolutions() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.bindingResolutions
}

// DetachedCandidatesAsked counts the CANDIDATES passed across every call, which
// is what the cost scales with: each becomes one `list-clients`, about 250 ms
// against a real detached session. DetachedQueries counts batched calls and so
// cannot see a fan-out growing (pair#206 M1).
func (f *FakeThreadArtifactCollisionChecker) DetachedCandidatesAsked() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.detachedCandidates
}

func (f *FakeThreadArtifactCollisionChecker) DetachedQueries() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.detachedQueries
}

func (f *FakeThreadArtifactCollisionChecker) Calls() []ThreadAddress {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]ThreadAddress{}, f.calls...)
}

// The fake must satisfy the same seams production does, or a test can pass
// against a double that production could not substitute.
var _ DetachedSessionResolver = (*FakeThreadArtifactCollisionChecker)(nil)
