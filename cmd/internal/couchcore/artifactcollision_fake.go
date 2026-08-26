package couchcore

import (
	"sync"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

type fakeArtifactCollision struct {
	collision bool
	err       error
}

type FakeThreadArtifactCollisionChecker struct {
	mu            sync.Mutex
	values        map[ThreadAddress]fakeArtifactCollision
	calls         []ThreadAddress
	released      []ThreadAddress
	registrations map[ThreadAddress]fakeRegistration
	autoEstablish bool
}

type fakeRegistration struct {
	evidence RegistrationEvidence
	err      error
}

func NewFakeThreadArtifactCollisionChecker() *FakeThreadArtifactCollisionChecker {
	return &FakeThreadArtifactCollisionChecker{
		values:        map[ThreadAddress]fakeArtifactCollision{},
		registrations: map[ThreadAddress]fakeRegistration{},
	}
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
	defer f.mu.Unlock()
	if value, ok := f.registrations[address]; ok {
		return value.evidence, value.err
	}
	if f.autoEstablish {
		return RegistrationEstablished, nil
	}
	return RegistrationAbsent, nil
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
