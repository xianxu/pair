package couchtty

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

// ErrRemotePending refuses a remote operation whose key is already queued.
var ErrRemotePending = errors.New("an operation for this slot is already pending")

// remoteOperation is a socket-originated job's completion state: the call its
// prepare step admitted (written on the queue goroutine before the completion
// is sent, read by finishOperation after it is received, so the channel
// orders the two) and the caller's outcome hook.
type remoteOperation struct {
	call     couchcore.OperationCall
	finished func(any, error)
}

// EnqueueRemoteOperation runs a socket-originated operation through the same
// queue, dispatcher (c.ops) and completion path (finishOperation: adoption,
// menu result) as a switcher keypress (pair#367 M2). prepare runs ON the
// queue, so admission is judged against the inventory at execution time. The
// job runs in the background: its child is adopted without taking focus
// (PreserveFocus), and with Attempt 0 it never touches the operator's
// InFlight operation.
//
// Single outcome owner: when it returns an error (ErrRemotePending, a full
// queue, no dispatcher) it has not called, and never will call, started or
// finished; the caller reports that outcome. When it returns nil, started
// fires once when the job begins and finished exactly once after adoption.
func (c *Console) EnqueueRemoteOperation(key, op string, prepare func(context.Context) (couchcore.OperationCall, error),
	started func(), finished func(any, error)) error {
	c.mu.Lock()
	fn := c.ops
	c.mu.Unlock()
	if fn == nil {
		return errors.New("no action dispatcher wired")
	}
	remote := &remoteOperation{finished: finished}
	accepted, err := c.operationQueue.Enqueue(operationRequest{
		key: key, name: op, remote: remote,
		origin: MenuOperationOrigin{Operation: op, PreserveFocus: true},
		run: func() (any, error) {
			started()
			ctx, cancel := context.WithCancel(c.lifetime)
			defer cancel()
			call, err := prepare(ctx)
			if err != nil {
				return nil, err
			}
			if call.Name != op {
				return nil, fmt.Errorf("prepared %q for a %s request", call.Name, op)
			}
			call.Implicit, call.Context = true, ctx
			remote.call = call
			return fn(call)
		},
	})
	switch {
	case err != nil:
		return err
	case !accepted:
		return ErrRemotePending
	}
	return nil
}

// remoteResumeCompletion recognizes a remote resume's completion without an
// origin field of its own: a resume with Attempt 0 (never the operator's,
// whose attempts count from 1, nor the reattach pass's) and no continuation
// ID (never a continuation replacement).
func remoteResumeCompletion(origin MenuOperationOrigin) bool {
	return origin.Operation == "resume" && origin.Attempt == 0 && origin.ContinuationID == ""
}

// remoteOperationAddress is the row a remote job's prepared call named: by
// exact tag, or by a slot's host checkout in the menu's inventory.
func remoteOperationAddress(state MenuState, remote *remoteOperation) (couchcore.ThreadAddress, bool) {
	if remote == nil {
		return couchcore.ThreadAddress{}, false
	}
	args := remote.call.Args
	if tag := args["tag"]; tag != "" {
		return couchcore.ThreadAddress{RepoScope: args["repo-scope"], Tag: couchcore.ThreadTag(tag)}, true
	}
	if path := args["path"]; path != "" {
		for _, row := range menuRows(state) {
			if row.Target.Kind == couchcore.ThreadTargetSlot && filepath.Clean(row.Target.Slot.WorktreeRoot) == filepath.Clean(path) {
				return row.Address, true
			}
		}
	}
	return couchcore.ThreadAddress{}, false
}
