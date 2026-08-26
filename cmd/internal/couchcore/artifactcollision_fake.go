package couchcore

import "sync"

type fakeArtifactCollision struct {
	collision bool
	err       error
}

type FakeThreadArtifactCollisionChecker struct {
	mu     sync.Mutex
	values map[ThreadAddress]fakeArtifactCollision
	calls  []ThreadAddress
}

func NewFakeThreadArtifactCollisionChecker() *FakeThreadArtifactCollisionChecker {
	return &FakeThreadArtifactCollisionChecker{values: map[ThreadAddress]fakeArtifactCollision{}}
}

func (f *FakeThreadArtifactCollisionChecker) Set(address ThreadAddress, collision bool, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.values[address] = fakeArtifactCollision{collision: collision, err: err}
}

func (f *FakeThreadArtifactCollisionChecker) Collides(address ThreadAddress) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, address)
	value := f.values[address]
	return value.collision, value.err
}

func (f *FakeThreadArtifactCollisionChecker) Calls() []ThreadAddress {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]ThreadAddress{}, f.calls...)
}
