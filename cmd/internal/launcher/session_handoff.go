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
	if bound <= 0 {
		return 1, errors.New("terminal admission bound must be positive")
	}
	lease, err := rt.Acquire(ctx)
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
	if err = rt.Check(ctx); err != nil {
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
	admitCtx, stopObserver := context.WithTimeout(ctx, bound)
	defer stopObserver()
	admitted := make(chan error, 1)
	go func() { admitted <- rt.Admit(admitCtx) }()
	select {
	case exit := <-exited:
		stopObserver()
		<-admitted
		return exit.Code, errors.Join(exit.Err, errors.New("terminal client exited before admission was verified"))
	case err = <-admitted:
		stopObserver()
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
