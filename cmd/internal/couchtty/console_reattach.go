package couchtty

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

// statusSpinnerInterval paces the status row's spinner while a thread is
// reattaching. The pass lasts seconds, and each tick repaints one row.
const statusSpinnerInterval = 120 * time.Millisecond

// ArmReattachPass arms the background reattach pass (pair#206) with the startup
// root it must never reattach. Call it once, after the initial attach commits
// and BEFORE Run: the Run loop is what admits inventories, so arming first
// guarantees the first inventory reduced is the one the pass seeds from.
func (c *Console) ArmReattachPass(root couchcore.ThreadAddress) {
	c.reattachArmOnce.Do(func() {
		c.mu.Lock()
		factory := c.sessionOwnerPass
		c.mu.Unlock()
		ctx, cancel := context.WithCancel(c.lifetime)
		if factory != nil {
			scoped, cancelScope := factory(ctx)
			ctx = scoped
			cancelParent := cancel
			cancel = func() {
				cancelScope()
				cancelParent()
			}
		}
		c.mu.Lock()
		if c.lifetime.Err() != nil {
			c.mu.Unlock()
			cancel()
			return
		}
		c.reattachContext, c.reattachCancel = ctx, cancel
		// This is pre-Run configuration. Reduce under the state lock without
		// publishing an empty panel before initial notices are admitted.
		c.ensureMenuLocked(couchcore.ThreadAddress{})
		c.menu, _ = ReduceMenu(c.menu, MenuEvent{Kind: MenuEventReattachArm, Address: root})
		c.mu.Unlock()
		c.observeLifecycleArm()
	})
}

// SetSessionOwnerPass supplies a lazy context factory before ArmReattachPass.
// Creating the scope must perform no process or session observation; workers
// capture evidence when they run. Ordinary operations keep the console lifetime.
func (c *Console) SetSessionOwnerPass(factory func(context.Context) (context.Context, context.CancelFunc)) {
	c.mu.Lock()
	c.sessionOwnerPass = factory
	c.mu.Unlock()
}

func (c *Console) releaseReattachPassLocked() context.CancelFunc {
	cancel := c.reattachCancel
	c.reattachContext, c.reattachCancel = nil, nil
	return cancel
}

func (c *Console) completedReattachPassLocked() context.CancelFunc {
	if c.menu.Reattach.Phase != ReattachDone {
		return nil
	}
	return c.releaseReattachPassLocked()
}

// runBackgroundOperation enqueues one reattach-pass attempt. It never touches
// the operator's in-flight slot: its origin is built from the effect itself and
// marked Background.
//
// Its queue key cannot collide with an operator attempt's, but NOT because of
// the "reattach" prefix: the pass and the operator draw attempt numbers from one
// counter (MenuState.OperationSequence), so every key is already unique. The
// prefix only makes a pass entry recognisable in the queue. A mutation that
// moves it into the operator's namespace survives the sweep for exactly that
// reason -- it is an equivalent mutant, recorded as such.
func (c *Console) runBackgroundOperation(effect MenuEffect) {
	origin := MenuOperationOrigin{
		Operation: effect.Operation, Attempt: effect.Attempt, Background: true,
		Address: couchcore.ThreadAddress{RepoScope: effect.Args["repo-scope"], Tag: couchcore.ThreadTag(effect.Args["tag"])},
	}
	// Traced whether or not a dispatcher is wired: an attempt that cannot run
	// still finishes, through finishOperation, which traces its end.
	c.traceEvent(traceReattachStart, origin.Address, fmt.Sprintf("attempt=%d", effect.Attempt))
	c.mu.Lock()
	fn := c.ops
	parent := c.reattachContext
	if parent == nil {
		parent = c.lifetime
	}
	c.mu.Unlock()
	if fn == nil {
		c.finishOperation(operationCompletion{name: effect.Operation, origin: origin, err: errors.New("no action dispatcher wired")})
		return
	}
	requestArgs := cloneOperationArgs(effect.Args)
	key := fmt.Sprintf("reattach\x00%d", effect.Attempt)
	attempt, attemptErr := c.beginLifecycleAttempt(effect.Operation, requestArgs, origin.Address)
	if attemptErr != nil {
		c.finishOperation(operationCompletion{name: effect.Operation, origin: origin, err: attemptErr})
		return
	}
	// Every attempt must complete, or the pass waits on its in-flight set forever.
	// Enqueue refuses an exact request that is already pending, returning
	// accepted=false with no error, and then nothing completes this attempt.
	// That cannot happen here, because the shared counter never repeats an
	// attempt number, so a pass key is never pending twice. If keys ever stop
	// being unique, finish a refused attempt here, as the error path below
	// does (#206 close review).
	_, err := c.operationQueue.Enqueue(operationRequest{attempt: attempt, key: key, name: effect.Operation, origin: origin, run: func() (any, error) {
		operationContext, cancelOperation := context.WithCancel(parent)
		defer cancelOperation()
		if attempt != nil {
			operationContext = couchcore.WithOperationAttempt(operationContext, attempt)
		}
		// Stop or pass release can cancel an enqueued attempt before dispatch.
		if err := operationContext.Err(); err != nil {
			return nil, err
		}
		return fn(couchcore.OperationCall{Name: effect.Operation, Args: requestArgs, Implicit: true, Context: operationContext})
	}})
	if err != nil {
		c.finishOperation(operationCompletion{attempt: attempt, key: key, name: effect.Operation, origin: origin, err: err})
	}
}

// ReattachPassArmed reports whether the background reattach pass was armed on
// this console, whatever phase it has reached since. It exists for couchcmd's
// wiring test, the way ActionableProvider serves its own: the arming decision
// lives in couchcmd, and only the console knows whether it happened.
func (c *Console) ReattachPassArmed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.menu.Reattach.Phase != ReattachIdle
}
