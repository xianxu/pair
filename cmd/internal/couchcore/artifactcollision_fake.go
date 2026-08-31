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
	mu              sync.Mutex
	values          map[ThreadAddress]fakeArtifactCollision
	calls           []ThreadAddress
	released        []ThreadAddress
	registrations   map[ThreadAddress]fakeRegistration
	autoEstablish   bool
	quiesced        []ThreadAddress
	QuiesceHook     func(ThreadAddress) error
	pairSessions    map[ThreadAddress]PairSessionBinding
	nativeBindings  map[nativeBindingKey]NativeBindingResolution
	triggeredQuit   []TriggeredQuit
	TriggerQuitHook func(string, launcher.QuitIntent) error
	// BeforeRegistration lets an integration test interleave a durable state
	// change at the registration boundary. It is called outside mu because the
	// hook may consult another stateful fake or call back into this one.
	BeforeRegistration func(ThreadAddress) error
	BeforePairSession  func(ThreadAddress) error
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
		values:         map[ThreadAddress]fakeArtifactCollision{},
		registrations:  map[ThreadAddress]fakeRegistration{},
		pairSessions:   map[ThreadAddress]PairSessionBinding{},
		nativeBindings: map[nativeBindingKey]NativeBindingResolution{},
	}
}

func (f *FakeThreadArtifactCollisionChecker) SetNativeBinding(address ThreadAddress, agent string, status sessioninventory.BindingStatus, nativeID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nativeBindings[nativeBindingKey{Address: address, Agent: agent}] = NativeBindingResolution{Status: status, NativeID: nativeID}
}

func (f *FakeThreadArtifactCollisionChecker) ResolveEstablished(_ context.Context, repoScope, tag, agent string) (NativeBindingResolution, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	resolution, ok := f.nativeBindings[nativeBindingKey{Address: ThreadAddress{RepoScope: repoScope, Tag: ThreadTag(tag)}, Agent: agent}]
	if !ok {
		resolution.Status = sessioninventory.BindingUnbound
	}
	if code := bindingResumeDiagnostic(resolution); code != "" {
		return resolution, refuseResume(code, "native session binding is not one exact established root")
	}
	return resolution, nil
}

func (f *FakeThreadArtifactCollisionChecker) SetPairSession(address ThreadAddress, name string, present bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.pairSessions[address] = PairSessionBinding{Name: name, Present: present}
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

func (f *FakeThreadArtifactCollisionChecker) Quiesce(address ThreadAddress) error {
	f.mu.Lock()
	f.quiesced = append(f.quiesced, address)
	hook := f.QuiesceHook
	f.mu.Unlock()
	if hook != nil {
		return hook(address)
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

func (f *FakeThreadArtifactCollisionChecker) Calls() []ThreadAddress {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]ThreadAddress{}, f.calls...)
}
