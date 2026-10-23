package launcher

import (
	"context"
	"errors"
	"io"
	"time"
)

// HandoffExit reports the client, never the lifetime of its server or agent.
type HandoffExit struct {
	Code int
	Err  error
}
type HandoffClient interface{ Wait() HandoffExit }

// TerminalHandoffRuntime owns one attempt. Check observes again under the lease;
// Admit must honor cancellation and prove the exact generation Start requested.
// Neither operation may retire a server or retry attachment as creation.
type TerminalHandoffRuntime interface {
	Acquire(context.Context) (io.Closer, error)
	Check(context.Context) error
	Start(context.Context) (HandoffClient, error)
	Admit(context.Context) error
}

// CoordinateTerminalHandoff serializes admission, not the interactive client's
// lifetime. Every failed admission reaps its own client and joins its observer
// before releasing the lease. Process death releases the OS-backed lease.
func CoordinateTerminalHandoff(ctx context.Context, rt TerminalHandoffRuntime, bound time.Duration) (int, error) {
	return coordinateTerminalHandoff(ctx, ctx, rt, bound, true)
}

// The query budget ends at admission; it never owns the interactive client.
func coordinateTerminalHandoff(ctx, queryCtx context.Context, rt TerminalHandoffRuntime, bound time.Duration, create bool) (int, error) {
	if bound <= 0 {
		return 1, errors.New("terminal admission bound must be positive")
	}
	parent := queryCtx
	if create {
		parent = ctx
	}
	startup, cancel := context.WithTimeout(parent, bound)
	defer cancel()
	return coordinateTerminalHandoffContexts(ctx, queryCtx, startup, rt, create)
}

func coordinateTerminalHandoffContexts(ctx, queryCtx, startup context.Context, rt TerminalHandoffRuntime, create bool) (int, error) {
	if err := startup.Err(); err != nil {
		return 1, err
	}
	deadline, ok := startup.Deadline()
	if !ok {
		return 1, errors.New("terminal startup deadline is unavailable")
	}
	preflight, stopPreflight := context.WithDeadline(queryCtx, deadline)
	stopStartupLink := context.AfterFunc(startup, stopPreflight)
	defer stopStartupLink()
	defer stopPreflight()
	queryCtx = preflight
	if err := queryCtx.Err(); err != nil {
		return 1, err
	}
	lease, err := rt.Acquire(queryCtx)
	if err != nil {
		return 1, err
	}
	release := func() error {
		if lease == nil {
			return nil
		}
		l := lease
		lease = nil
		return l.Close()
	}
	defer release()
	if err = rt.Check(queryCtx); err != nil {
		return 1, err
	}
	if err := queryCtx.Err(); err != nil {
		return 1, err
	}
	if err := ctx.Err(); err != nil {
		return 1, err
	}
	if err := startup.Err(); err != nil {
		return 1, err
	}
	clientCtx, stopClient := context.WithCancel(ctx)
	defer stopClient()
	client, err := rt.Start(clientCtx)
	if err != nil {
		return 1, err
	}
	exited := make(chan HandoffExit, 1)
	go func() { exited <- client.Wait() }()
	admitCtx, stopObserver := context.WithCancel(startup)
	defer stopObserver()
	admitted := make(chan error, 1)
	go func() { admitted <- rt.Admit(admitCtx) }()
	select {
	case exit := <-exited:
		// Admission work belongs to this attempt, including post-birth queries
		// whose deadline is independent of the readiness wait.
		stopClient()
		stopObserver()
		observerErr := <-admitted
		return exit.Code, errors.Join(exit.Err, observerErr, startup.Err(), ctx.Err(), errors.New("terminal client exited before admission was verified"))
	case err = <-admitted:
		stopObserver()
		if err == nil && !create {
			err = queryCtx.Err()
		}
		if err == nil {
			err = startup.Err()
		}
		if err == nil {
			err = ctx.Err()
		}
		if err != nil {
			stopClient()
			exit := <-exited
			return exit.Code, err
		}
		if err = release(); err != nil {
			stopClient()
			exit := <-exited
			return exit.Code, err
		}
		exit := <-exited
		return exit.Code, errors.Join(exit.Err, ctx.Err())
	}
}
