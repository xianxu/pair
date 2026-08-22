package ptychild

import (
	"os"
	"sync"
)

// NewFakeChild returns a Child with no process behind it: a seeded replay ring,
// a live screen scanner, and recorded writes and resizes.
//
// It is the stateful double ARCH-MOCK asks for on this seam, and it lives in a
// non-test file for the same reason FakeRunner and FakeHost do -- a switcher's
// tests need a child whose output they control, and production and test flow
// must share the same type or the tests prove nothing about the real path.
//
// Contract:
//   - Feed(p) appends to the ring and the screen, exactly as the real pump does.
//   - Write records into Writes() instead of reaching a pty.
//   - Resize records into Resizes().
//   - Wait returns ExitCode (default 0) immediately; Done is true once Exit is
//     called, or immediately if the child was never "running".
func NewFakeChild(output []byte) *Child {
	c := &Child{
		ring:   NewRing(DefaultRingBytes),
		screen: &Screen{},
		done:   make(chan struct{}),
		fake:   &fakeState{},
	}
	if len(output) > 0 {
		c.Feed(output)
	}
	return c
}

type fakeState struct {
	mu      sync.Mutex
	writes  [][]byte
	resizes []Size
	exited  bool
}

// Feed pushes bytes through the same path the real pump uses. On a real child
// it is how a test would inject output; on a fake it is the only source.
func (c *Child) Feed(p []byte) {
	chunk := append([]byte(nil), p...)
	c.mu.Lock()
	c.ring.Append(chunk)
	c.screen.Feed(chunk)
	c.mu.Unlock()
	if c.sink != nil {
		c.sink(chunk)
	}
}

// SetSink installs a sink after construction, for a fake whose consumer is not
// known at the point it is built.
func (c *Child) SetSink(sink func([]byte)) { c.sink = sink }

// Writes returns what the caller wrote to this child.
func (c *Child) Writes() [][]byte {
	if c.fake == nil {
		return nil
	}
	c.fake.mu.Lock()
	defer c.fake.mu.Unlock()
	return append([][]byte(nil), c.fake.writes...)
}

// Resizes returns the sizes this child was set to, in order.
func (c *Child) Resizes() []Size {
	if c.fake == nil {
		return nil
	}
	c.fake.mu.Lock()
	defer c.fake.mu.Unlock()
	return append([]Size(nil), c.fake.resizes...)
}

// Exit ends a fake child with the given code, unblocking Wait.
func (c *Child) Exit(code int) {
	if c.fake == nil {
		return
	}
	c.fake.mu.Lock()
	already := c.fake.exited
	c.fake.exited = true
	c.fake.mu.Unlock()
	if already {
		return
	}
	c.code = code
	close(c.done)
}

// Signal on a fake records nothing and succeeds: a test that cares about
// signalling asserts on Writes or on Exit, not on a syscall that never happened.
func (c *Child) fakeSignal(os.Signal) error { return nil }
