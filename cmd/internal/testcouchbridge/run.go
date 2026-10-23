package testcouchbridge

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Owner is the existing fleet resource owner, including partially prepared
// resources. Every operation must honor its context and join its own children.
type Owner interface {
	Prepare(context.Context) error
	Start(context.Context) error
	Readiness(context.Context) (Readiness, error)
	Done() <-chan struct{}
	Close(context.Context) error
	AssertCleaned() error
	RetainedRoot() string
}

// BridgeIO exclusively owns its descriptors. Cancellation joins a blocked
// publication or input read; Close runs only after both have returned.
type BridgeIO interface {
	WaitStop(context.Context) (Event, error)
	Publish(context.Context, []byte) error
	Close() error
}

// RunBridge holds one fleet until EOF, a signal, failure or the fixed lifetime.
// Deadlines start together at entry, so completing a stage never renews them.
func RunBridge(ctx context.Context, owner Owner, pipes BridgeIO) error {
	lifetime, cancelLife := context.WithTimeout(ctx, LifetimeBudget)
	defer cancelLife()
	startup, cancelStart := context.WithTimeout(lifetime, StartupBudget)
	defer cancelStart()
	return runBridge(lifetime, startup, owner, pipes, StopBudget)
}

func runBridge(lifetime, startup context.Context, owner Owner, pipes BridgeIO, stopBudget time.Duration) error {
	work, cancelWork := context.WithCancel(startup)
	wait, cancelWait := context.WithCancel(lifetime)
	defer cancelWork()
	defer cancelWait()
	stops := make(chan Event, 1)
	waiterDone := make(chan struct{})
	go func() {
		defer close(waiterDone)
		e, err := pipes.WaitStop(wait)
		if err != nil {
			e = Event{Kind: Failed, Failure: fmt.Sprintf("control input: %v", err)}
		}
		// Publish the cause before cancellation can produce a secondary stage error.
		stops <- e
		cancelWork()
	}()
	type completion struct {
		event     Event
		readiness Readiness
	}
	completed := make(chan completion, 1)
	var active bool
	var readiness Readiness
	state, effect := Reduce(State{}, Event{Kind: Begin})
	var producer <-chan struct{}
	var startupDone = startup.Done()
	for {
		// An observed stop wins over any queued successful completion.
		select {
		case e := <-stops:
			state, effect = Reduce(state, e)
		default:
		}
		if effect == Stop {
			break
		}
		if effect != None {
			active = true
			current := effect
			go func() {
				c := completion{}
				var err error
				switch current {
				case Prepare:
					err = owner.Prepare(work)
					c.event.Kind = CandidateReady
				case Start:
					err = owner.Start(work)
					c.event.Kind = FleetReady
				case Check:
					c.readiness, err = owner.Readiness(work)
					c.event.Kind = EndpointReady
				case Publish:
					var line []byte
					line, err = EncodeReadiness(readiness)
					if err == nil {
						err = pipes.Publish(work, line)
					}
					c.event.Kind = ReadinessWritten
				}
				if err != nil {
					c.event = Event{Kind: Failed, Failure: fmt.Sprintf("bridge stage %d: %v", current, err)}
				}
				completed <- c
			}()
			effect = None
		}
		var event Event
		select {
		case event = <-stops:
		case <-lifetime.Done():
			event = Event{Kind: Deadline, Failure: fmt.Sprintf("bridge lifetime: %v", lifetime.Err())}
		case <-startupDone:
			event = Event{Kind: Deadline, Failure: fmt.Sprintf("bridge startup: %v", startup.Err())}
		case <-producer:
			event = Event{Kind: ProducerDied, Failure: "owned Couch exited"}
		case c := <-completed:
			active = false
			event = c.event
			if event.Kind == EndpointReady {
				readiness = c.readiness
			}
			if event.Kind == FleetReady {
				producer = owner.Done()
			}
			if event.Kind == ReadinessWritten {
				startupDone = nil
			}
		}
		select {
		case stop := <-stops:
			event = stop
		default:
		}
		state, effect = Reduce(state, event)
	}
	// A single budget covers cancellation, all joins, and the owner's cleanup.
	cleanup, cancelCleanup := context.WithTimeout(context.Background(), stopBudget)
	defer cancelCleanup()
	cancelWork()
	cancelWait()
	var joinErr error
	if active {
		select {
		case <-completed:
		case <-cleanup.Done():
			joinErr = fmt.Errorf("stage join: %w", cleanup.Err())
		}
	}
	select {
	case <-waiterDone:
	case <-cleanup.Done():
		joinErr = errors.Join(joinErr, fmt.Errorf("control join: %w", cleanup.Err()))
	}
	if joinErr != nil {
		return errors.Join(nonemptyError(state.Failure), joinErr, fmt.Errorf("cleanup uncertain; retained root %s", owner.RetainedRoot()))
	}
	ioErr := pipes.Close()
	closeErr := owner.Close(cleanup)
	var assertErr error
	if closeErr == nil {
		assertErr = owner.AssertCleaned()
	}
	cleanupErr := errors.Join(ioErr, closeErr, assertErr, cleanup.Err())
	if cleanupErr != nil {
		state, _ = Reduce(state, Event{Kind: CleanupUncertain, Failure: cleanupErr.Error()})
		return errors.Join(nonemptyError(state.Failure), cleanupErr, fmt.Errorf("cleanup uncertain; retained root %s", owner.RetainedRoot()))
	}
	state, _ = Reduce(state, Event{Kind: CleanupClean})
	return nonemptyError(state.Failure)
}

func nonemptyError(s string) error {
	if s == "" {
		return nil
	}
	return errors.New(s)
}
