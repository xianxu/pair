//go:build darwin || linux

package vmcmd

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// ownedProcess pins its direct child until the last permitted group signal.
// Terminal observation never reaps. Finish retires every signal path before
// Wait releases the PID, and only observes the group after that point.
type ownedProcess struct {
	cmd                *exec.Cmd
	mu                 sync.Mutex
	retired            bool
	outputErr          error
	signalGroup        func(int) error
	groupPresent       func(int) (bool, error)
	terminal           func(int) (bool, error)
	finishOnce         sync.Once
	done               chan struct{}
	waitErr, signalErr error
}

func configureSnapshotCommand(cmd *exec.Cmd) *ownedProcess {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	owner := &ownedProcess{cmd: cmd, done: make(chan struct{}), terminal: nativeChildTerminal, groupPresent: nativeGroupPresent,
		signalGroup: func(pid int) error { return syscall.Kill(-pid, syscall.SIGKILL) }}
	cmd.Cancel = func() error {
		err := owner.Signal()
		if err == nil || errors.Is(err, syscall.EPERM) {
			return os.ErrProcessDone
		}
		return err
	}
	// exec.Cmd's WaitDelay can issue its own Process.Kill after ownership has
	// retired. All signals and writer failures instead pass through this owner.
	cmd.WaitDelay = 0
	return owner
}

func (o *ownedProcess) signalLocked() error {
	if o.retired || o.cmd.Process == nil {
		return nil
	}
	err := o.signalGroup(o.cmd.Process.Pid)
	if errors.Is(err, syscall.ESRCH) {
		return nil
	}
	return err
}
func (o *ownedProcess) Signal() error { o.mu.Lock(); defer o.mu.Unlock(); return o.signalLocked() }

func (o *ownedProcess) Writer(w io.Writer) io.Writer {
	if w == nil {
		w = io.Discard
	}
	return ownedProcessWriter{owner: o, writer: w}
}

type ownedProcessWriter struct {
	owner  *ownedProcess
	writer io.Writer
}

func (w ownedProcessWriter) Write(p []byte) (int, error) {
	n, err := w.writer.Write(p)
	if err == nil && n != len(p) {
		err = io.ErrShortWrite
	}
	if err != nil {
		w.owner.mu.Lock()
		w.owner.outputErr = errors.Join(w.owner.outputErr, err)
		signalErr := w.owner.signalLocked()
		w.owner.mu.Unlock()
		return n, errors.Join(err, signalErr)
	}
	return n, nil
}

func (o *ownedProcess) AwaitTerminal(ctx context.Context) error {
	for {
		o.mu.Lock()
		outputErr := o.outputErr
		o.mu.Unlock()
		if outputErr != nil {
			return outputErr
		}
		terminal, err := o.terminal(o.cmd.Process.Pid)
		if err != nil {
			return err
		}
		if terminal {
			return nil
		}
		if err := ctx.Err(); err != nil {
			return errors.Join(err, o.Signal())
		}
		timer := time.NewTimer(5 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			// Observe a concurrently terminal parent before replacing its exit.
			continue
		case <-timer.C:
		}
	}
}

func (o *ownedProcess) Finish(ctx context.Context) (error, error) {
	o.finishOnce.Do(func() {
		o.mu.Lock()
		o.signalErr = o.signalLocked()
		o.retired = true
		o.mu.Unlock()
		// No cancellation callback or writer can signal a numeric PID after this.
		go func() { o.waitErr = o.cmd.Wait(); close(o.done) }()
	})
	select {
	case <-o.done:
	case <-ctx.Done():
		return nil, errors.Join(ErrUnconfirmedProcessCleanup, ctx.Err(), o.signalErr)
	}
	if err := awaitSnapshotGroupGone(ctx, func() (bool, error) { return o.groupPresent(o.cmd.Process.Pid) }); err != nil {
		return o.waitErr, errors.Join(ErrUnconfirmedProcessCleanup, o.signalErr, err)
	}
	// EPERM can describe a zombie-only group while the direct child is pinned.
	// Positive final absence resolves it; other signal failures stay visible.
	if o.signalErr != nil && !errors.Is(o.signalErr, syscall.EPERM) {
		return o.waitErr, o.signalErr
	}
	return o.waitErr, nil
}

func cloneFallbackAllowed(err error) bool {
	return errors.Is(err, syscall.EXDEV) || errors.Is(err, syscall.ENOTSUP) || errors.Is(err, syscall.EOPNOTSUPP)
}

func awaitSnapshotGroupGone(ctx context.Context, present func() (bool, error)) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		found, err := present()
		if err != nil {
			return err
		}
		if !found {
			return nil
		}
		timer := time.NewTimer(5 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}
