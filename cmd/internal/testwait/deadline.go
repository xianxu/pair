package testwait

import (
	"context"
	"sync"
	"time"
)

// cueDeadline advertises a budget without starting a wall clock. Its terminal
// error is published before the underlying cancellation wakes any consumer.
type cueDeadline struct {
	context.Context
	at   time.Time
	mu   sync.Mutex
	err  error
	done chan struct{}
}

// A distinct Done channel prevents context's cancelCtx fast path from
// substituting Canceled for this context's DeadlineExceeded. AfterFunc keeps
// derived-context propagation notification-based, without a waiting goroutine.
func (c *cueDeadline) Done() <-chan struct{}          { return c.done }
func (c *cueDeadline) AfterFunc(f func()) func() bool { return context.AfterFunc(c.Context, f) }

func (c *cueDeadline) Deadline() (time.Time, bool) { return c.at, true }
func (c *cueDeadline) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.err
}

// DeadlineOnCue returns a context whose budget expires only on expire. Parent
// cancellation still propagates; the first expiry or cancellation wins. cancel
// releases the parent's notification even when neither parent nor cue fires.
func DeadlineOnCue(parent context.Context, budget time.Duration) (context.Context, func(), context.CancelFunc) {
	base, stop := context.WithCancelCause(context.WithoutCancel(parent))
	c := &cueDeadline{Context: base, at: time.Now().Add(budget), done: make(chan struct{})}
	if at, ok := parent.Deadline(); ok && at.Before(c.at) {
		c.at = at
	}
	finish := func(err, cause error) {
		c.mu.Lock()
		defer c.mu.Unlock()
		if c.err != nil {
			return
		}
		c.err = err
		stop(cause)
		close(c.done)
	}
	parentDone := func() { finish(parent.Err(), context.Cause(parent)) }
	stopParent := context.AfterFunc(parent, parentDone)
	if parent.Err() != nil {
		parentDone()
	}
	terminate := func(err error) {
		// Parent notification can still be queued when a later cue arrives. Observe
		// an already-terminal parent synchronously so that cue cannot replace it.
		if parent.Err() != nil {
			parentDone()
		} else {
			finish(err, err)
		}
		stopParent()
	}
	expire := func() { terminate(context.DeadlineExceeded) }
	cancel := func() { terminate(context.Canceled) }
	return c, expire, cancel
}
