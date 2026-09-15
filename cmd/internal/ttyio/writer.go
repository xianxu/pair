// Package ttyio provides context-aware terminal IO. Nonblocking descriptor flags
// belong to the transport lifetime, including all concurrent application readers.
package ttyio

import (
	"context"
	"errors"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sys/unix"
)

// Writer reports the accepted prefix even when writing fails. Implementations
// must honor cancellation; launching an uncancellable Write goroutine is not one.
type Writer interface {
	WriteContext(context.Context, []byte) (int, error)
}

type descriptor struct {
	file      *os.File
	fd, flags int
}

type File struct {
	input, output int
	descriptors   []descriptor
	owned         bool
	mu            sync.Mutex
	closed        bool
	done          chan struct{}
	operations    sync.WaitGroup
	writes        chan struct{}
	polls         atomic.Uint64
	closeOnce     sync.Once
	closeErr      error
}

// NewFile acquires descriptors before any read/write pump starts. If stdin and
// stdout share an open-file description, every application reader must use Read
// here too. Close restores flags after all operations have left, and closes the
// files only when owned. It never duplicates descriptors to pretend flags isolate.
func NewFile(input, output *os.File, owned bool) (*File, error) {
	if output == nil {
		return nil, errors.New("ttyio: missing output")
	}
	f := &File{input: -1, output: -1, owned: owned, done: make(chan struct{}), writes: make(chan struct{}, 1)}
	// Resolve every Fd before changing flags: os.File.Fd may change pollability.
	files := []*os.File{input, output}
	seen := map[int]bool{}
	for _, file := range files {
		if file == nil {
			continue
		}
		fd := int(file.Fd())
		if file == input {
			f.input = fd
		}
		if file == output {
			f.output = fd
		}
		if seen[fd] {
			continue
		}
		seen[fd] = true
		f.descriptors = append(f.descriptors, descriptor{file: file, fd: fd})
	}
	// Capture all original states first; two descriptors may share their flags.
	for i := range f.descriptors {
		d := &f.descriptors[i]
		flags, err := unix.FcntlInt(uintptr(d.fd), unix.F_GETFL, 0)
		if err != nil {
			return nil, err
		}
		d.flags = flags
	}
	for i, d := range f.descriptors {
		if err := unix.SetNonblock(d.fd, true); err != nil {
			for _, old := range f.descriptors[:i] {
				_, _ = unix.FcntlInt(uintptr(old.fd), unix.F_SETFL, old.flags)
			}
			return nil, err
		}
	}
	return f, nil
}

func (f *File) begin() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return os.ErrClosed
	}
	f.operations.Add(1)
	return nil
}
func (f *File) stopped(ctx context.Context) error {
	select {
	case <-f.done:
		return os.ErrClosed
	default:
		return ctx.Err()
	}
}
func (f *File) wait(ctx context.Context, fd int, event int16) error {
	if err := f.stopped(ctx); err != nil {
		return err
	}
	timeout := 50
	if deadline, ok := ctx.Deadline(); ok {
		left := time.Until(deadline)
		if left <= 0 {
			return context.DeadlineExceeded
		}
		if left < 50*time.Millisecond {
			timeout = int((left + time.Millisecond - 1) / time.Millisecond)
		}
	}
	f.polls.Add(1)
	_, err := unix.Poll([]unix.PollFd{{Fd: int32(fd), Events: event}}, timeout)
	if err != nil && !errors.Is(err, unix.EINTR) {
		return err
	}
	return f.stopped(ctx)
}
func (f *File) PollWaits() uint64          { return f.polls.Load() }
func (f *File) Read(p []byte) (int, error) { return f.ReadContext(context.Background(), p) }
func (f *File) ReadContext(ctx context.Context, p []byte) (int, error) {
	if err := f.begin(); err != nil {
		return 0, err
	}
	defer f.operations.Done()
	if f.input < 0 {
		return 0, errors.New("ttyio: missing input")
	}
	if len(p) == 0 {
		return 0, nil
	}
	for {
		if err := f.stopped(ctx); err != nil {
			return 0, err
		}
		n, err := unix.Read(f.input, p)
		if n > 0 {
			return n, nil
		}
		if err == nil {
			return 0, io.EOF
		}
		if errors.Is(err, unix.EINTR) {
			continue
		}
		if !errors.Is(err, unix.EAGAIN) && !errors.Is(err, unix.EWOULDBLOCK) {
			return 0, err
		}
		if err = f.wait(ctx, f.input, unix.POLLIN); err != nil {
			return 0, err
		}
	}
}
func (f *File) Write(p []byte) (int, error) { return f.WriteContext(context.Background(), p) }
func (f *File) WriteContext(ctx context.Context, p []byte) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := f.begin(); err != nil {
		return 0, err
	}
	defer f.operations.Done()
	select {
	case f.writes <- struct{}{}:
		defer func() { <-f.writes }()
	case <-ctx.Done():
		return 0, ctx.Err()
	case <-f.done:
		return 0, os.ErrClosed
	}
	total := 0
	for len(p) > 0 {
		if err := f.stopped(ctx); err != nil {
			return total, err
		}
		n, err := unix.Write(f.output, p)
		if n > 0 {
			total += n
			p = p[n:]
		}
		if err == nil {
			if n == 0 {
				return total, io.ErrNoProgress
			}
			continue
		}
		if errors.Is(err, unix.EINTR) {
			continue
		}
		if !errors.Is(err, unix.EAGAIN) && !errors.Is(err, unix.EWOULDBLOCK) {
			return total, err
		}
		if err = f.wait(ctx, f.output, unix.POLLOUT); err != nil {
			return total, err
		}
	}
	return total, nil
}
func (f *File) Close() error {
	f.closeOnce.Do(func() {
		f.mu.Lock()
		f.closed = true
		close(f.done)
		f.mu.Unlock()
		f.operations.Wait()
		for _, d := range f.descriptors {
			_, err := unix.FcntlInt(uintptr(d.fd), unix.F_SETFL, d.flags)
			f.closeErr = errors.Join(f.closeErr, err)
		}
		if f.owned {
			for _, d := range f.descriptors {
				f.closeErr = errors.Join(f.closeErr, d.file.Close())
			}
		}
	})
	return f.closeErr
}
