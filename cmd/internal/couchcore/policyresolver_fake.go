package couchcore

import (
	"context"
	"sync"
)

type fakePolicyResponse struct {
	result PolicyResult
	err    error
}

// FakePolicyResolver is stateful: callers can queue policy epochs for one path
// and verify the order in which admission refreshed them.
type FakePolicyResolver struct {
	mu              sync.Mutex
	responses       map[string][]fakePolicyResponse
	calls           []string
	defaultResponse fakePolicyResponse
}

func NewFakePolicyResolver() *FakePolicyResolver {
	return &FakePolicyResolver{
		responses: map[string][]fakePolicyResponse{},
		defaultResponse: fakePolicyResponse{result: PolicyResult{
			PolicyVersion: 1,
			PolicyDigest:  "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			RepoIdentity:  "fake-repo",
			AdmissionKey:  "fake-unbounded",
			Capacity:      PolicyCapacity{Kind: CapacityUnbounded},
		}},
	}
}

func (f *FakePolicyResolver) SetDefault(result PolicyResult, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.defaultResponse = fakePolicyResponse{result: result, err: err}
}

func (f *FakePolicyResolver) Queue(path string, result PolicyResult, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.responses[path] = append(f.responses[path], fakePolicyResponse{result: result, err: err})
}

func (f *FakePolicyResolver) ResolvePolicy(_ context.Context, path string) (PolicyResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, path)
	queued := f.responses[path]
	if len(queued) == 0 {
		return f.defaultResponse.result, f.defaultResponse.err
	}
	next := queued[0]
	f.responses[path] = queued[1:]
	return next.result, next.err
}

func (f *FakePolicyResolver) Calls() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string{}, f.calls...)
}
