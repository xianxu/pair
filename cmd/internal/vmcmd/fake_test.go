package vmcmd

import (
	"context"
	"fmt"
	"sync"
)

// fakeRuntime models resources, rather than returning a scripted call list.
// A gate can hold an effect after entry; ignoreCancellation models an effect
// racing with cancellation and returning a late successful outcome.
type fakeRuntime struct {
	mu                                                      sync.Mutex
	acquired, prepared, child, running, collected, released bool
	calls                                                   []Effect
	fail                                                    map[Effect]error
	gates                                                   map[Effect]chan struct{}
	entered                                                 chan Effect
	ignoreCancellation                                      bool
	commandExit                                             int
	contexts                                                map[Effect]context.Context
}

func newFakeRuntime() *fakeRuntime {
	return &fakeRuntime{fail: map[Effect]error{}, gates: map[Effect]chan struct{}{}, entered: make(chan Effect, 16), contexts: map[Effect]context.Context{}}
}
func (f *fakeRuntime) effect(ctx context.Context, e Effect) error {
	f.mu.Lock()
	f.calls = append(f.calls, e)
	f.contexts[e] = ctx
	f.mu.Unlock()
	f.entered <- e
	if gate := f.gates[e]; gate != nil {
		if f.ignoreCancellation {
			<-gate
		} else {
			select {
			case <-gate:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	if !f.ignoreCancellation {
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	if err := f.fail[e]; err != nil {
		return err
	}
	return nil
}
func (f *fakeRuntime) Acquire(ctx context.Context, _ RunRequest) error {
	if err := f.effect(ctx, Acquire); err != nil {
		return err
	}
	f.acquired = true
	return nil
}
func (f *fakeRuntime) Prepare(ctx context.Context) error {
	if !f.acquired {
		return fmt.Errorf("prepare without position")
	}
	f.prepared = true // A failed preparation can leave a partial snapshot.
	if err := f.effect(ctx, Prepare); err != nil {
		return err
	}
	return nil
}
func (f *fakeRuntime) Start(ctx context.Context) error {
	if !f.prepared {
		return fmt.Errorf("start without snapshot")
	}
	f.child = true
	f.running = true // Startup can fail after the native child exists.
	if err := f.effect(ctx, Start); err != nil {
		return err
	}
	return nil
}
func (f *fakeRuntime) AwaitGuest(ctx context.Context) error {
	if !f.running {
		return fmt.Errorf("ready without running VM")
	}
	return f.effect(ctx, AwaitGuest)
}
func (f *fakeRuntime) Execute(ctx context.Context) (int, error) {
	if !f.running {
		return 0, fmt.Errorf("execute without VM")
	}
	return f.commandExit, f.effect(ctx, Execute)
}
func (f *fakeRuntime) Collect(ctx context.Context) error {
	if !f.acquired {
		return fmt.Errorf("collect without position")
	}
	if err := f.effect(ctx, Collect); err != nil {
		return err
	}
	f.collected = true
	return nil
}
func (f *fakeRuntime) Stop(ctx context.Context) error {
	err := f.effect(ctx, Stop)
	if err == nil {
		f.running = false
	}
	f.child = false
	return err
}
func (f *fakeRuntime) Release(ctx context.Context) error {
	err := f.effect(ctx, Release)
	if f.child {
		return fmt.Errorf("released before stop joined")
	}
	// Closing the local handle is unconditional after joining, including when
	// the budget expired. Resource deletion still requires successful cleanup.
	f.acquired = false
	if err != nil {
		return err
	}
	if f.running {
		return fmt.Errorf("VM termination unconfirmed; retaining claim and snapshot")
	}
	f.prepared = false
	f.released = true
	return nil
}
