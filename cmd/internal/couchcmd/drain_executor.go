package couchcmd

import (
	"context"
	"errors"
	"sync"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

// drainAdmissionTask owns one target's admission attempt. A cancelled task still
// runs so its owner can retire the receipt, but must check ctx before any effect.
type drainAdmissionTask struct {
	target string
	run    func(context.Context)
}

// drainAdmissionExecutor bounds concurrency and storage while preserving each
// target's notice order, including drain followed immediately by lift. A busy
// target never occupies a worker merely waiting for its predecessor.
type drainAdmissionExecutor struct {
	ctx     context.Context
	cancel  context.CancelFunc
	mu      sync.Mutex
	changed *sync.Cond
	pending []drainAdmissionTask
	active  map[string]bool
	workers sync.WaitGroup
}

func newDrainAdmissionExecutor(parent context.Context) *drainAdmissionExecutor {
	ctx, cancel := context.WithCancel(parent)
	executor := &drainAdmissionExecutor{ctx: ctx, cancel: cancel, active: make(map[string]bool)}
	executor.changed = sync.NewCond(&executor.mu)
	// Join the cancellation observer along with the workers: parent cancellation
	// wakes even an empty executor, without an unowned AfterFunc callback.
	executor.workers.Add(1)
	go func() {
		defer executor.workers.Done()
		<-ctx.Done()
		executor.mu.Lock()
		executor.changed.Broadcast()
		executor.mu.Unlock()
	}()
	executor.workers.Add(4)
	for range 4 {
		go executor.work()
	}
	return executor
}

// enqueue either accepts the entire batch or changes nothing. Active tasks
// consume the same bounded capacity as queued tasks.
func (executor *drainAdmissionExecutor) enqueue(tasks []drainAdmissionTask) error {
	executor.mu.Lock()
	defer executor.mu.Unlock()
	if executor.ctx.Err() != nil {
		return errors.New("drain admission executor is closed")
	}
	if len(tasks) > maxSlotOperationReceipts*couchmessage.MaxActors-len(executor.pending)-len(executor.active) {
		return errors.New("drain admission executor capacity exhausted")
	}
	executor.pending = append(executor.pending, tasks...)
	executor.changed.Broadcast()
	return nil
}

func (executor *drainAdmissionExecutor) work() {
	defer executor.workers.Done()
	for {
		executor.mu.Lock()
		index := -1
		for index < 0 {
			for i, task := range executor.pending {
				if !executor.active[task.target] {
					index = i
					break
				}
			}
			if index >= 0 {
				break
			}
			if executor.ctx.Err() != nil && len(executor.pending) == 0 {
				executor.mu.Unlock()
				return
			}
			executor.changed.Wait()
		}
		task := executor.pending[index]
		copy(executor.pending[index:], executor.pending[index+1:])
		executor.pending[len(executor.pending)-1] = drainAdmissionTask{}
		executor.pending = executor.pending[:len(executor.pending)-1]
		executor.active[task.target] = true
		executor.mu.Unlock()

		ctx, cancel := context.WithTimeout(executor.ctx, couchmessage.AdmissionTimeout)
		task.run(ctx)
		cancel()

		executor.mu.Lock()
		delete(executor.active, task.target)
		executor.changed.Broadcast()
		executor.mu.Unlock()
	}
}

// close is idempotent and joins all callbacks, including already-cancelled
// callbacks for tasks which were queued when cancellation occurred.
func (executor *drainAdmissionExecutor) close() {
	executor.cancel()
	executor.workers.Wait()
}
