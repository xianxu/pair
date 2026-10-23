//go:build unix

package testcouchbridge

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

const pipePollSlice = 25 * time.Millisecond

// PipeIO exclusively owns its stdio pipe ends. Ownership transfers on entry to
// NewPipeIO, including failure; callers must not access or close them afterward.
// O_NONBLOCK belongs to the shared open-file description: any preexisting dup
// also observes it. The external parent must hold only the opposite pipe ends.
// No internal goroutine outlives a call; Close revokes admission, joins active
// calls, then releases the exact descriptors.
type PipeIO struct {
	input, output              *os.File
	inputFD, outputFD          int
	signals                    <-chan os.Signal
	stop                       chan struct{}
	mu                         sync.Mutex
	closeMu                    sync.Mutex
	calls                      sync.WaitGroup
	closed, waiting, published bool
	closeErr                   error
}

func NewPipeIO(input, output *os.File, signals <-chan os.Signal) (_ *PipeIO, result error) {
	accepted := false
	defer func() {
		if !accepted {
			if input != nil {
				result = errors.Join(result, input.Close())
			}
			if output != nil && output != input {
				result = errors.Join(result, output.Close())
			}
		}
	}()
	if input == nil || output == nil || input == output {
		return nil, fmt.Errorf("distinct owned input/output pipes required")
	}
	inFD, err := configurePipe(input, false)
	if err != nil {
		return nil, fmt.Errorf("input pipe: %w", err)
	}
	outFD, err := configurePipe(output, true)
	if err != nil {
		return nil, fmt.Errorf("output pipe: %w", err)
	}
	accepted = true
	return &PipeIO{input: input, output: output, inputFD: inFD, outputFD: outFD, signals: signals, stop: make(chan struct{})}, nil
}

func configurePipe(file *os.File, write bool) (int, error) {
	info, err := file.Stat()
	if err != nil {
		return 0, err
	}
	if info.Mode()&os.ModeNamedPipe == 0 {
		return 0, fmt.Errorf("stdio must be a pipe, not %s", info.Mode().Type())
	}
	conn, err := file.SyscallConn()
	if err != nil {
		return 0, err
	}
	fd := -1
	var configureErr error
	err = conn.Control(func(raw uintptr) {
		flags, e := unix.FcntlInt(raw, unix.F_GETFL, 0)
		if e != nil {
			configureErr = e
			return
		}
		mode := flags & unix.O_ACCMODE
		if (!write && mode == unix.O_WRONLY) || (write && mode == unix.O_RDONLY) {
			configureErr = fmt.Errorf("pipe has incompatible access mode")
			return
		}
		configureErr = unix.SetNonblock(int(raw), true)
		if configureErr == nil {
			fd = int(raw)
		}
	})
	return fd, errors.Join(err, configureErr)
}

func (p *PipeIO) admit(publish bool) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return os.ErrClosed
	}
	if publish {
		if p.published {
			return fmt.Errorf("readiness publication already attempted")
		}
		p.published = true
	} else {
		if p.waiting {
			return fmt.Errorf("stop observation already started")
		}
		p.waiting = true
	}
	p.calls.Add(1)
	return nil
}

func (p *PipeIO) status(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case <-p.stop:
		return os.ErrClosed
	default:
		return nil
	}
}

// poll bounds cancellation latency without a background reader/writer or an
// unjoined syscall. Nonblocking read/write remains authoritative after readiness.
func (p *PipeIO) poll(ctx context.Context, fd int, events int16) (int16, error) {
	if err := p.status(ctx); err != nil {
		return 0, err
	}
	timeout := pipePollSlice
	if deadline, ok := ctx.Deadline(); ok {
		if remaining := time.Until(deadline); remaining < timeout {
			timeout = remaining
		}
	}
	if timeout <= 0 {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		return 0, context.DeadlineExceeded
	}
	milliseconds := int((timeout + time.Millisecond - 1) / time.Millisecond)
	fds := []unix.PollFd{{Fd: int32(fd), Events: events}}
	_, err := unix.Poll(fds, milliseconds)
	if err == unix.EINTR {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if err := p.status(ctx); err != nil {
		return 0, err
	}
	if fds[0].Revents&unix.POLLNVAL != 0 {
		return 0, os.ErrClosed
	}
	return fds[0].Revents, nil
}

func (p *PipeIO) WaitStop(ctx context.Context) (Event, error) {
	if err := p.admit(false); err != nil {
		return Event{}, err
	}
	defer p.calls.Done()
	signals := p.signals
	for {
		if err := p.status(ctx); err != nil {
			return Event{}, err
		}
		select {
		case signal, ok := <-signals:
			if !ok {
				signals = nil
			} else if signal == syscall.SIGTERM {
				return Event{Kind: TERM}, nil
			} else if signal == syscall.SIGINT {
				return Event{Kind: INT}, nil
			}
		default:
		}
		ready, err := p.poll(ctx, p.inputFD, unix.POLLIN)
		if err != nil {
			return Event{}, err
		}
		if ready&(unix.POLLIN|unix.POLLHUP|unix.POLLERR) == 0 {
			continue
		}
		var data [1]byte
		n, err := unix.Read(p.inputFD, data[:])
		if err == unix.EAGAIN || err == unix.EWOULDBLOCK || err == unix.EINTR {
			continue
		}
		if err != nil {
			return Event{}, err
		}
		if n > 0 {
			return Event{Kind: Failed}, fmt.Errorf("bridge stdin accepts EOF only, not data")
		}
		return Event{Kind: EOF}, nil
	}
}

// Publish writes exactly one already-encoded readiness line. A partial or failed
// write is uncertain and consumes the publication attempt; it is never retried.
func (p *PipeIO) Publish(ctx context.Context, line []byte) error {
	if err := p.admit(true); err != nil {
		return err
	}
	defer p.calls.Done()
	if len(line) == 0 || len(line) > ReadinessLimit || line[len(line)-1] != '\n' || bytes.Count(line, []byte{'\n'}) != 1 {
		return fmt.Errorf("readiness must be one line of at most %d bytes", ReadinessLimit)
	}
	for len(line) > 0 {
		if err := p.status(ctx); err != nil {
			return err
		}
		ready, err := p.poll(ctx, p.outputFD, unix.POLLOUT)
		if err != nil {
			return err
		}
		if ready&(unix.POLLOUT|unix.POLLHUP|unix.POLLERR) == 0 {
			continue
		}
		n, err := unix.Write(p.outputFD, line)
		if n > 0 {
			line = line[n:]
		}
		if err == unix.EAGAIN || err == unix.EWOULDBLOCK || err == unix.EINTR {
			continue
		}
		if err != nil {
			return err
		}
		if n == 0 {
			return fmt.Errorf("readiness pipe made no write progress")
		}
	}
	return nil
}

func (p *PipeIO) Close() error {
	p.closeMu.Lock()
	defer p.closeMu.Unlock()
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return p.closeErr
	}
	p.closed = true
	close(p.stop)
	p.mu.Unlock()
	p.calls.Wait()
	p.closeErr = errors.Join(p.input.Close(), p.output.Close())
	return p.closeErr
}
