package hostty

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/xianxu/pair/cmd/internal/ttyio"
	"golang.org/x/sys/unix"
	"golang.org/x/term"

	"github.com/xianxu/pair/cmd/internal/ptychild"
)

// OSHost is Host over a real terminal.
type OSHost struct {
	fd        int
	transport *ttyio.File
	initErr   error
	watchDone chan struct{}
	closeErr  error
	in        *os.File
	out       *os.File

	resized    chan struct{}
	sigs       chan os.Signal
	terminated chan os.Signal
	once       sync.Once
}

var _ Host = (*OSHost)(nil)
var _ TerminationHost = (*OSHost)(nil)

// NewOSHost wraps a terminal. in is the fd measured and switched to raw mode;
// out is where the console draws.
func NewOSHost(in, out *os.File) *OSHost {
	h := &OSHost{
		in: in, out: out, resized: make(chan struct{}, 1),
		terminated: make(chan os.Signal, 1),
	}
	if in != nil {
		h.fd = int(in.Fd())
	}
	h.transport, h.initErr = ttyio.NewFile(in, out, false)
	h.watchDone = make(chan struct{})
	if in != nil {
		h.sigs = make(chan os.Signal, 1)
		signal.Notify(h.sigs, syscall.SIGWINCH)
		signal.Notify(h.terminated, syscall.SIGTERM, syscall.SIGHUP)
		go h.watch()
	}
	if in == nil {
		close(h.watchDone)
	}
	return h
}

// watch turns SIGWINCH into a coalesced wake. signal.Notify already drops
// signals when its buffer is full, and the non-blocking send below does the
// same for the wake channel -- so a drag delivers one pending wake, not N.
func (h *OSHost) watch() {
	// The watcher owns resized's lifetime: it is the only sender, so closing it
	// here (after the signal source is closed) cannot race a send.
	defer close(h.watchDone)
	defer close(h.resized)
	for range h.sigs {
		select {
		case h.resized <- struct{}{}:
		default:
		}
	}
}

func (h *OSHost) Write(p []byte) (int, error) { return h.WriteContext(context.Background(), p) }
func (h *OSHost) WriteContext(ctx context.Context, p []byte) (int, error) {
	if h.initErr != nil {
		return 0, h.initErr
	}
	return h.transport.WriteContext(ctx, p)
}
func (h *OSHost) Read(p []byte) (int, error) { return h.ReadContext(context.Background(), p) }
func (h *OSHost) ReadContext(ctx context.Context, p []byte) (int, error) {
	if h.initErr != nil {
		return 0, h.initErr
	}
	return h.transport.ReadContext(ctx, p)
}

func (h *OSHost) Size() (ptychild.Size, error) {
	if h.in == nil {
		return ptychild.Size{}, fmt.Errorf("hostty: no terminal to measure")
	}
	if h.initErr != nil {
		return ptychild.Size{}, h.initErr
	}
	ws, err := unix.IoctlGetWinsize(h.fd, unix.TIOCGWINSZ)
	if err != nil {
		return ptychild.Size{}, fmt.Errorf("hostty: measure terminal: %w", err)
	}
	return ptychild.Size{Rows: ws.Row, Cols: ws.Col}, nil
}

func (h *OSHost) MakeRaw() (func() error, error) {
	if h.in == nil {
		return func() error { return nil }, nil
	}
	if h.initErr != nil {
		return nil, h.initErr
	}
	fd := h.fd
	state, err := term.MakeRaw(fd)
	if err != nil {
		return nil, fmt.Errorf("hostty: raw mode: %w", err)
	}

	var once sync.Once
	return func() error {
		var rerr error
		once.Do(func() { rerr = term.Restore(fd, state) })
		return rerr
	}, nil
}

func (h *OSHost) Resized() <-chan struct{}     { return h.resized }
func (h *OSHost) Terminated() <-chan os.Signal { return h.terminated }

// Close stops watching for resizes and releases anyone ranging over Resized().
//
// Closing `resized` is load-bearing, not tidiness: a consumer written as
// `for range host.Resized()` would otherwise block forever and leak (BR-2). The
// watcher goroutine closes it, after its own source is closed, so there is
// exactly one writer and no send-on-closed race.
func (h *OSHost) Close() error {
	h.once.Do(func() {
		if h.sigs != nil {
			signal.Stop(h.sigs)
			close(h.sigs)
		} else {
			close(h.resized)
		}
		signal.Stop(h.terminated)
		close(h.terminated)
		<-h.watchDone
		if h.transport != nil {
			h.closeErr = h.transport.Close()
		}
	})
	return h.closeErr
}
