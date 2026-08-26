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
	mu       sync.Mutex
	values   map[ThreadAddress]fakeArtifactCollision
	calls    []ThreadAddress
	released []ThreadAddress
}

func NewFakeThreadArtifactCollisionChecker() *FakeThreadArtifactCollisionChecker {
	return &FakeThreadArtifactCollisionChecker{values: map[ThreadAddress]fakeArtifactCollision{}}
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
