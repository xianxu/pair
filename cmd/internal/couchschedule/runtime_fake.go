package couchschedule

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// FakeExecution is a retained attempt in FakeTaskRuntime's controller model.
type FakeExecution struct {
	Request                         ExecutionRequest
	Owner                           OwnerIdentity
	Authorized, Finished, Cancelled bool
}

// FakeTaskRuntime models owned execution, durable acknowledgement, task
// exclusion, two execution permits and cancellation across calls.
type FakeTaskRuntime struct {
	mu      sync.Mutex
	runs    []FakeExecution
	results []ExecutionResult
	active  map[TaskKey]bool
	gate    chan struct{}
	started chan ExecutionRequest
	permits chan struct{}
}

var _ TaskExecutor = (*FakeTaskRuntime)(nil)

func NewFakeTaskRuntime() *FakeTaskRuntime {
	return &FakeTaskRuntime{active: map[TaskKey]bool{}, started: make(chan ExecutionRequest, 256), permits: make(chan struct{}, 2)}
}
func (f *FakeTaskRuntime) Enqueue(result ExecutionResult) {
	f.mu.Lock()
	defer f.mu.Unlock()
	result.Output = append([]byte(nil), result.Output...)
	f.results = append(f.results, result)
}
func (f *FakeTaskRuntime) Hold() {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.gate == nil {
		f.gate = make(chan struct{})
	}
}
func (f *FakeTaskRuntime) Release() {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.gate != nil {
		close(f.gate)
		f.gate = nil
	}
}
func (f *FakeTaskRuntime) Started() <-chan ExecutionRequest { return f.started }
func (f *FakeTaskRuntime) Runs() []FakeExecution {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := append([]FakeExecution(nil), f.runs...)
	for i := range out {
		out[i].Request.Args = append([]string(nil), out[i].Request.Args...)
	}
	return out
}

func (f *FakeTaskRuntime) Execute(ctx context.Context, r ExecutionRequest, onOwner func(OwnerIdentity) error) (ExecutionResult, error) {
	if err := ctx.Err(); err != nil {
		return ExecutionResult{}, err
	}
	if onOwner == nil {
		return ExecutionResult{}, errors.New("execution requires durable owner acknowledgement")
	}
	key := TaskKey{Hub: r.HubRoot, Task: r.Task}
	f.mu.Lock()
	if f.active[key] {
		f.mu.Unlock()
		return ExecutionResult{Exit: -1, Outcome: "failed", Error: "scheduled task is already running"}, nil
	}
	f.active[key] = true
	index := len(f.runs)
	owner := OwnerIdentity{PID: 100000 + index, Start: fmt.Sprintf("fake-controller-%d", index)}
	r.Args = append([]string(nil), r.Args...)
	f.runs = append(f.runs, FakeExecution{Request: r, Owner: owner})
	f.mu.Unlock()
	defer func() { f.mu.Lock(); f.runs[index].Finished = true; delete(f.active, key); f.mu.Unlock() }()
	if err := onOwner(owner); err != nil {
		return ExecutionResult{}, &ExecutionError{Owner: owner, Err: err}
	}
	f.mu.Lock()
	f.runs[index].Authorized = true
	f.mu.Unlock()
	cancelled := func() (ExecutionResult, error) {
		f.mu.Lock()
		f.runs[index].Cancelled = true
		f.mu.Unlock()
		return ExecutionResult{Exit: -1, Outcome: "cancelled", Error: ctx.Err().Error()}, nil
	}
	select {
	case f.permits <- struct{}{}:
	case <-ctx.Done():
		return cancelled()
	}
	defer func() { <-f.permits }()
	f.mu.Lock()
	gate := f.gate
	result := ExecutionResult{Exit: 0, Outcome: "quiet"}
	if len(f.results) > 0 {
		result = f.results[0]
		f.results = f.results[1:]
	}
	f.mu.Unlock()
	select {
	case f.started <- r:
	case <-ctx.Done():
		return cancelled()
	}
	if gate != nil {
		select {
		case <-gate:
		case <-ctx.Done():
			return cancelled()
		}
	}
	result.Output = append([]byte(nil), result.Output...)
	return result, nil
}
