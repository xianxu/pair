package hostty

import (
	"os"
	"strings"
	"sync"

	"github.com/xianxu/pair/cmd/internal/ptychild"
)

// FakeHost is the stateful double ARCH-MOCK asks for: a terminal whose size a
// test sets, whose resizes a test fires, and whose output a test reads.
//
// Contract, fixed here rather than inferred from tests:
//   - Size returns whatever SetSize last set.
//   - SetSize changes the size AND delivers one coalesced resize.
//   - MakeRaw increments RawDepth; its restore decrements once, however many
//     times it is called.
//   - Write appends to Written().
type FakeHost struct {
	mu         sync.Mutex
	size       ptychild.Size
	written    strings.Builder
	rawDepth   int
	resized    chan struct{}
	terminated chan os.Signal
	closed     bool
}

var _ Host = (*FakeHost)(nil)
var _ TerminationHost = (*FakeHost)(nil)

func NewFakeHost(size ptychild.Size) *FakeHost {
	return &FakeHost{
		size: size, resized: make(chan struct{}, 1),
		terminated: make(chan os.Signal, 1),
	}
}

func (h *FakeHost) Write(p []byte) (int, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.written.Write(p)
}

func (h *FakeHost) Written() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.written.String()
}

// Reset clears the captured output, so a test can assert on one repaint rather
// than on everything since construction.
func (h *FakeHost) Reset() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.written.Reset()
}

func (h *FakeHost) Size() (ptychild.Size, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.size, nil
}

// SetSize resizes the fake terminal and delivers a coalesced wake.
//
// After Close it is INERT, matching OSHost: a real host absorbs a SIGWINCH
// burst after teardown without complaint. The first version sent outside the
// lock and never consulted closed, so a post-Close SetSize panicked with "send
// on closed channel" -- in the double M2's console tests are built on, which
// would crash a run rather than fail it (BR-18).
func (h *FakeHost) SetSize(s ptychild.Size) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.size = s
	if h.closed {
		return
	}
	select {
	case h.resized <- struct{}{}:
	default:
	}
}

func (h *FakeHost) MakeRaw() (func() error, error) {
	h.mu.Lock()
	h.rawDepth++
	h.mu.Unlock()

	var once sync.Once
	return func() error {
		once.Do(func() {
			h.mu.Lock()
			h.rawDepth--
			h.mu.Unlock()
		})
		return nil
	}, nil
}

// RawDepth is how many un-restored MakeRaw calls are outstanding. A console
// that leaks one leaves the operator's terminal in raw mode.
func (h *FakeHost) RawDepth() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.rawDepth
}

func (h *FakeHost) Resized() <-chan struct{} { return h.resized }

func (h *FakeHost) Terminated() <-chan os.Signal { return h.terminated }

// Terminate injects the process-level signal Console would receive through an
// OSHost. Bursts coalesce exactly like signal.Notify on a capacity-one channel.
func (h *FakeHost) Terminate(sig os.Signal) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return
	}
	select {
	case h.terminated <- sig:
	default:
	}
}

// Close matches OSHost: it releases anyone ranging over Resized().
func (h *FakeHost) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.closed {
		h.closed = true
		close(h.resized)
		close(h.terminated)
	}
	return nil
}

func (h *FakeHost) Closed() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.closed
}
