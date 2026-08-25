package hostty

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/creack/pty"
	"golang.org/x/term"

	"github.com/xianxu/pair/cmd/internal/ptychild"
)

// OSHost is Host over a real terminal.
type OSHost struct {
	in  *os.File
	out *os.File

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
		h.sigs = make(chan os.Signal, 1)
		signal.Notify(h.sigs, syscall.SIGWINCH)
		signal.Notify(h.terminated, syscall.SIGTERM, syscall.SIGHUP)
		go h.watch()
	}
	return h
}

// watch turns SIGWINCH into a coalesced wake. signal.Notify already drops
// signals when its buffer is full, and the non-blocking send below does the
// same for the wake channel -- so a drag delivers one pending wake, not N.
func (h *OSHost) watch() {
	// The watcher owns resized's lifetime: it is the only sender, so closing it
	// here (after the signal source is closed) cannot race a send.
	defer close(h.resized)
	for range h.sigs {
		select {
		case h.resized <- struct{}{}:
		default:
		}
	}
}

func (h *OSHost) Write(p []byte) (int, error) {
	if h.out == nil {
		return 0, fmt.Errorf("hostty: no output terminal")
	}
	return h.out.Write(p)
}

func (h *OSHost) Size() (ptychild.Size, error) {
	if h.in == nil {
		return ptychild.Size{}, fmt.Errorf("hostty: no terminal to measure")
	}
	ws, err := pty.GetsizeFull(h.in)
	if err != nil {
		return ptychild.Size{}, fmt.Errorf("hostty: measure terminal: %w", err)
	}
	return ptychild.Size{Rows: ws.Rows, Cols: ws.Cols}, nil
}

func (h *OSHost) MakeRaw() (func() error, error) {
	if h.in == nil {
		return func() error { return nil }, nil
	}
	fd := int(h.in.Fd())
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
	})
	return nil
}
