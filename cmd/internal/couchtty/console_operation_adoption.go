package couchtty

import (
	"context"
	"errors"
	"sync/atomic"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

// prepareOperationCompletion runs only on the bounded lifecycle workers. Its
// return is the final adoption outcome; Run remains the sole reduction owner.
func (c *Console) prepareOperationCompletion(completed operationCompletion) operationCompletion {
	if completed.prepared {
		return completed
	}
	completed.prepared = true
	if completed.owner == nil {
		completed.owner = new(atomic.Bool)
	}
	if completed.err != nil {
		return completed
	}
	child, ok := completed.value.(couchcore.StartedChild)
	if !ok {
		return completed
	}
	started, hasChild := child.Started()
	if !hasChild {
		return completed
	}
	c.mu.Lock()
	fn := c.ops
	c.mu.Unlock()
	if fn == nil {
		completed.err = errors.New("no action dispatcher wired")
		return completed
	}
	address := completed.origin.Address
	if started.Record.Thread != (couchcore.ThreadAddress{}) {
		address = started.Record.Thread
	}
	// Installation owns routing, never navigation. Run decides whether the
	// operator's current intent still authorizes selecting the adopted child.
	ctx := c.lifetime
	if completed.attempt != nil {
		ctx = couchcore.WithOperationAttempt(ctx, completed.attempt)
	} else if completed.remote != nil && completed.remote.call.Context != nil {
		if attempt := couchcore.OperationAttemptFromContext(completed.remote.call.Context); attempt != nil {
			ctx = couchcore.WithOperationAttempt(ctx, attempt)
		}
	}
	_, completed.err = fn(couchcore.OperationCall{
		Name: "attach", Context: ctx, Implicit: true, TypedPayload: started,
		Args: map[string]string{"repo-scope": address.RepoScope, "tag": string(address.Tag), "background": "true"},
	})
	return completed
}

// A stopped event loop cannot consume another completion. Settle the accepted
// remote caller once; adopted children stay with Console teardown and rejected
// children stay with the declared attach operation's AbortStarted owner.
func (c *Console) abandonOperationCompletion(completed operationCompletion) {
	if completed.owner != nil && !completed.owner.CompareAndSwap(false, true) {
		return
	}
	err := completed.err
	if err == nil {
		err = context.Canceled
	}
	completed.err = err
	completed = c.finishLifecycleAttempt(completed)
	err = completed.err
	if completed.remote != nil && completed.remote.finished != nil {
		completed.remote.finished(completed.value, err)
	}
}
