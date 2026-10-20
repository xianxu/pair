package vmcmd

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Runtime owns the resources of one Run. Every method must honor its context,
// join any work it starts before returning, and tolerate partial initialization
// during cleanup. Acquire must unwind partial ownership before returning an
// error; success transfers ownership even when its context has just expired.
// Stop returns an error unless VM termination is confirmed. After an unconfirmed
// stop, Release must retain the durable occupied claim and VM/snapshot inputs,
// closing only local handles so the next acquisition refuses unsafe reuse.
// Release must never remove resources still consumed by a child or live VM.
// These obligations also apply when a cleanup context expires.
type Runtime interface {
	Acquire(context.Context, RunRequest) error
	Prepare(context.Context) error
	Start(context.Context) error
	AwaitGuest(context.Context) error
	Execute(context.Context) (int, error)
	Collect(context.Context) error
	Stop(context.Context) error
	Release(context.Context) error
}

const cleanupTimeout = 10 * time.Second

// Run executes only the effects requested by Advance. Runtime calls are joined
// synchronously: cancellation cannot release a position behind an active effect.
// The request budget covers acquisition through command execution; all cleanup
// shares a separate bounded budget, independent of the caller's cancellation.
func Run(ctx context.Context, r RunRequest, rt Runtime) RunState {
	runCtx, cancelRun := context.WithTimeout(ctx, r.Timeout)
	defer cancelRun()
	var cleanupCtx context.Context
	var cancelCleanup context.CancelFunc
	defer func() {
		if cancelCleanup != nil {
			cancelCleanup()
		}
	}()
	state := NewRunState()
	effect := NoEffect
	advance := func(e Event) {
		next, requested, err := Advance(state, e)
		if err != nil {
			panic(fmt.Sprintf("vmcmd Run invariant: %v", err))
		}
		state, effect = next, requested
	}
	advance(Event{Kind: Begin})
	callerInterrupted := false
	cleanupInterrupted := false
	interruption := func(err error, detail string) Event {
		kind := Cancelled
		if errors.Is(err, context.DeadlineExceeded) {
			kind = TimedOut
		}
		return Event{Kind: kind, Detail: detail + ": " + err.Error()}
	}
	for effect != Complete {
		current := effect
		cleanup := current == Collect || current == Stop || current == Release
		opCtx := runCtx
		if cleanup {
			if cleanupCtx == nil {
				cleanupCtx, cancelCleanup = context.WithTimeout(context.WithoutCancel(ctx), cleanupTimeout)
			}
			opCtx = cleanupCtx
		} else if err := runCtx.Err(); err != nil {
			advance(interruption(err, string(current)))
			callerInterrupted = true
			continue
		}
		var err error
		var event Event
		switch current {
		case Acquire:
			err = rt.Acquire(opCtx, r)
			event.Kind = Acquired
		case Prepare:
			err = rt.Prepare(opCtx)
			event.Kind = Prepared
		case Start:
			err = rt.Start(opCtx)
			event.Kind = Started
		case AwaitGuest:
			err = rt.AwaitGuest(opCtx)
			event.Kind = GuestReady
		case Execute:
			event.Exit, err = rt.Execute(opCtx)
			event.Kind = CommandEnded
		case Collect:
			err = rt.Collect(opCtx)
			event.Kind = Collected
		case Stop:
			err = rt.Stop(opCtx)
			event.Kind = Stopped
		case Release:
			err = rt.Release(opCtx)
			event.Kind = Released
		default:
			panic(fmt.Sprintf("vmcmd Run unknown effect %q", current))
		}
		if !cleanup {
			if interrupted := runCtx.Err(); interrupted != nil {
				// Acquire may have transferred ownership just as cancellation arrived.
				// Publish that transfer before asking the reducer to clean it up.
				if current == Acquire && err == nil {
					advance(event)
				}
				detail := string(current)
				if err != nil {
					detail += ": " + err.Error()
				}
				advance(interruption(interrupted, detail))
				callerInterrupted = true
				continue
			}
		} else {
			// Cancellation during cleanup changes the result but does not start a
			// second cleanup or abandon the effect whose outcome is being joined.
			if interrupted := ctx.Err(); interrupted != nil && !callerInterrupted {
				advance(interruption(interrupted, "caller during cleanup"))
				callerInterrupted = true
			}
			if interrupted := cleanupCtx.Err(); interrupted != nil && !cleanupInterrupted {
				advance(interruption(interrupted, "cleanup budget"))
				cleanupInterrupted = true
			}
		}
		if err != nil {
			event = Event{Kind: Failed, Detail: fmt.Sprintf("%s: %v", current, err)}
		}
		// Invalid exit codes are runtime failures, not reducer inputs.
		if event.Kind == CommandEnded && (event.Exit < 0 || event.Exit > 255) {
			event = Event{Kind: Failed, Detail: "execute returned invalid exit code"}
		}
		advance(event)
	}
	return state
}
