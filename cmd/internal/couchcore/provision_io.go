package couchcore

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// ProvisionCommand describes one bounded foreground provisioning command.
type ProvisionCommand struct {
	Dir, Program string
	Args         []string
	Lease        *os.File
	Progress     io.Writer
	Timeout      time.Duration
	StreamOutput bool
}

type ProvisionIO interface {
	Run(context.Context, ProvisionCommand) ([]byte, error)
}

type OSProvisionIO struct{ Env []string }

func (runner OSProvisionIO) Run(ctx context.Context, request ProvisionCommand) ([]byte, error) {
	if request.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, request.Timeout)
		defer cancel()
	}
	callerCtx := ctx
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	cmd := exec.CommandContext(ctx, request.Program, request.Args...)
	cmd.Dir = request.Dir
	cmd.Env = runner.Env
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if request.Lease != nil {
		cmd.ExtraFiles = []*os.File{request.Lease}
	}
	cmd.WaitDelay = time.Second
	// CommandContext owns and joins this cancellation callback. Always finish the
	// group stop even if its leader exits first: descendants can retain the lease.
	cmd.Cancel = func() error {
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		timer := time.NewTimer(2 * time.Second)
		defer timer.Stop()
		<-timer.C
		killErr := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if errors.Is(killErr, syscall.ESRCH) {
			killErr = nil
		}
		return errors.Join(err, killErr)
	}
	diagnostics := &provisionOutput{limit: 64 << 10, tail: true, progress: request.Progress}
	stdout := &provisionOutput{limit: 1 << 20, onOverflow: cancel}
	if request.StreamOutput {
		stdout = diagnostics
	}
	cmd.Stdout = stdout
	cmd.Stderr = diagnostics
	err := cmd.Run()
	// A producer that exits while leaving pipes open must not leave its process
	// group running after WaitDelay closed those pipes.
	if errors.Is(err, exec.ErrWaitDelay) && cmd.Process != nil {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	if stdout.overflow {
		err = errors.Join(err, errors.New("provision command stdout exceeds 1 MiB"))
	}
	if callerCtx.Err() != nil {
		err = errors.Join(err, callerCtx.Err())
	}
	if err != nil && len(diagnostics.data) > 0 {
		err = fmt.Errorf("%w: %s", err, diagnostics.data)
	}
	return stdout.data, err
}

// A shared writer serializes stdout/stderr progress, including callers whose
// writer is not concurrency safe. It retains a diagnostic tail or capped output.
type provisionOutput struct {
	mu         sync.Mutex
	data       []byte
	limit      int
	tail       bool
	overflow   bool
	progress   io.Writer
	onOverflow context.CancelFunc
}

func (w *provisionOutput) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	n := len(p)
	if w.progress != nil {
		if _, err := w.progress.Write(p); err != nil {
			return 0, err
		}
	}
	if w.tail {
		if len(p) >= w.limit {
			w.data = append(w.data[:0], p[len(p)-w.limit:]...)
		} else {
			excess := len(w.data) + len(p) - w.limit
			if excess > 0 {
				copy(w.data, w.data[excess:])
				w.data = w.data[:len(w.data)-excess]
			}
			w.data = append(w.data, p...)
		}
	} else {
		available := w.limit - len(w.data)
		if len(p) > available {
			p = p[:available]
			w.overflow = true
			if w.onOverflow != nil {
				w.onOverflow()
			}
		}
		w.data = append(w.data, p...)
	}
	return n, nil
}
