package couchcore

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
)

// LifecycleParallelism bounds how many threads couch drives through a
// lifecycle operation at once: Leave's fan-out, the park worker and the
// console's queue workers (pair#205 D5). It is half the CPU cores, at least
// one (operator decision, 2026-10-07): each operation spawns zellij and helper
// processes, so the host's cores are the resource it competes for. Callers
// wait for a free unit; a bound is never a refusal.
var LifecycleParallelism = max(1, runtime.NumCPU()/2)

// ThreadBusyError refuses a lifecycle operation on a thread another operation
// holds. Refused, not queued: the operator learns at once that the gesture did
// not take (pair#214's operator decision, carried by #205).
type ThreadBusyError struct {
	Address ThreadAddress
	Running string
}

func (e *ThreadBusyError) Error() string {
	return fmt.Sprintf("%s is busy: %s is already running on it", e.Address.Tag, e.Running)
}

// IsThreadBusy reports whether err is, or wraps, a ThreadBusyError.
func IsThreadBusy(err error) bool {
	var busy *ThreadBusyError
	return errors.As(err, &busy)
}

type gateVerdict uint8

const (
	gateAdmit gateVerdict = iota
	gateReenter
	gateBusy
)

func (v gateVerdict) String() string {
	switch v {
	case gateAdmit:
		return "admit"
	case gateReenter:
		return "reenter"
	case gateBusy:
		return "busy"
	}
	return fmt.Sprintf("gateVerdict(%d)", uint8(v))
}

// gateToken identifies one hold. Re-entry matches the token, not the address,
// so a context that outlived its release cannot enter a later holder's hold.
type gateToken struct{ _ byte }

type gateHold struct {
	op    string
	token *gateToken
}

// gateDecision is the whole admission rule: one lifecycle operation per
// thread, re-entered only by the exact hold the caller's context carries.
func gateDecision(held map[ThreadAddress]gateHold, holds map[ThreadAddress]*gateToken, address ThreadAddress) (gateVerdict, string) {
	current, isHeld := held[address]
	if !isHeld {
		return gateAdmit, ""
	}
	if token, ok := holds[address]; ok && token == current.token {
		return gateReenter, ""
	}
	return gateBusy, current.op
}

type gateHoldsKey struct{}

func gateHolds(ctx context.Context) map[ThreadAddress]*gateToken {
	holds, _ := ctx.Value(gateHoldsKey{}).(map[ThreadAddress]*gateToken)
	return holds
}

// ThreadGate admits one lifecycle operation per thread. The zero value is
// ready. Holds live in memory only; each dies with its holder's release.
//
// Composite operations (relaunch parks then resumes) acquire once at the top
// and pass the returned context down, so the inner entries re-enter instead of
// refusing their own caller.
type ThreadGate struct {
	mu   sync.Mutex
	held map[ThreadAddress]gateHold
	// changed is closed and replaced on every release, waking waiters.
	changed chan struct{}
}

// acquire holds address for op. wait=false refuses a held thread with
// ThreadBusyError; wait=true blocks until it is released or ctx ends. The
// returned release is idempotent; on error it is nil.
func (g *ThreadGate) acquire(ctx context.Context, address ThreadAddress, op string, wait bool) (context.Context, func(), error) {
	if ctx == nil {
		ctx = context.Background()
	}
	for {
		g.mu.Lock()
		if g.held == nil {
			g.held = map[ThreadAddress]gateHold{}
			g.changed = make(chan struct{})
		}
		verdict, running := gateDecision(g.held, gateHolds(ctx), address)
		switch verdict {
		case gateReenter:
			g.mu.Unlock()
			return ctx, func() {}, nil
		case gateAdmit:
			token := &gateToken{}
			g.held[address] = gateHold{op: op, token: token}
			g.mu.Unlock()
			holds := map[ThreadAddress]*gateToken{address: token}
			for a, t := range gateHolds(ctx) {
				if a != address {
					holds[a] = t
				}
			}
			var once sync.Once
			return context.WithValue(ctx, gateHoldsKey{}, holds), func() {
				once.Do(func() {
					g.mu.Lock()
					if g.held[address].token == token {
						delete(g.held, address)
					}
					close(g.changed)
					g.changed = make(chan struct{})
					g.mu.Unlock()
				})
			}, nil
		}
		changed := g.changed
		g.mu.Unlock()
		if !wait {
			return ctx, nil, &ThreadBusyError{Address: address, Running: running}
		}
		select {
		case <-ctx.Done():
			return ctx, nil, ctx.Err()
		case <-changed:
		}
	}
}

func (c *Couch) gate() *ThreadGate {
	c.gateOnce.Do(func() {
		if c.threadGate == nil {
			c.threadGate = &ThreadGate{}
		}
	})
	return c.threadGate
}

// hold acquires the thread for op or refuses with ThreadBusyError.
func (c *Couch) hold(ctx context.Context, address ThreadAddress, op string) (context.Context, func(), error) {
	return c.gate().acquire(ctx, address, op, false)
}

// holdWait waits for the thread's current holder instead of refusing: the
// drains (Leave, RecoverActiveParks, AbortStarted), which must not skip a
// thread because some other operation reached it first.
func (c *Couch) holdWait(ctx context.Context, address ThreadAddress, op string) (context.Context, func(), error) {
	return c.gate().acquire(ctx, address, op, true)
}
