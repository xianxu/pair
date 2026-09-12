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
	c.reduceMenu(MenuEvent{Kind: MenuEventReattachArm, Address: root})
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
	c.mu.Unlock()
	if fn == nil {
		c.finishOperation(operationCompletion{name: effect.Operation, origin: origin, err: errors.New("no action dispatcher wired")})
		return
	}
	requestArgs := cloneOperationArgs(effect.Args)
	key := fmt.Sprintf("reattach\x00%d", effect.Attempt)
	_, err := c.operationQueue.Enqueue(operationRequest{key: key, name: effect.Operation, origin: origin, run: func() (any, error) {
		operationContext, cancelOperation := context.WithCancel(c.lifetime)
		defer cancelOperation()
		return fn(couchcore.OperationCall{Name: effect.Operation, Args: requestArgs, Implicit: true, Context: operationContext})
	}})
	if err != nil {
		c.finishOperation(operationCompletion{key: key, name: effect.Operation, origin: origin, err: err})
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
