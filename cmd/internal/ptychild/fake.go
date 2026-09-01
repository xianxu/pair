package ptychild

import (
	"fmt"
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
// Contract -- deliberately the SAME SHAPE as a real Child, because a fake whose
// lifecycle differs from production makes tests written against it lie. The M1
// boundary review caught this doc claiming the opposite of the code (BR-3), so
// TestFakeChildConformsToRealChildLifecycle now pins the pairing:
//
//   - Feed(p) appends to the ring and the screen, exactly as the real pump does.
//   - Write records into Writes() instead of reaching a pty.
//   - Resize records into Resizes().
//   - A fresh fake is RUNNING: Done() is false and Wait() BLOCKS, exactly as a
//     real child does before it exits.
//   - Exit(code) is what ends it -- the fake's stand-in for the process exiting.
//     It unblocks Wait, which then returns code, and flips Done to true.
//   - Close() ends it too, as Exit(0), mirroring the real Close that shuts the
//     pty and lets the pump reap.
//   - AFTER it has ended, Write / Resize / Signal all return an error, because
//     that is what the real Child does once its pty is closed and its process
//     reaped ("file already closed", an ioctl error, "process already
//     finished"). A fake that cheerfully accepts writes to a dead child lets a
//     test pass against a sequence production would have rejected (BR-18).
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
	batch := c.outputBatchLocked(chunk)
	sink := c.sink
	c.mu.Unlock()
	if sink != nil {
		sink(batch)
	}
}

// SetSink installs a sink after construction, for a fake whose consumer is not
// known at the point it is built.
func (c *Child) SetSink(sink func(OutputBatch)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sink = sink
}

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

// fakeSignal succeeds while the child is running and fails once it has ended --
// the real Child returns "process already finished" from os.Process.Signal, and
// a fake that returns nil there would let a test pass against a call production
// rejects (BR-18).
func (c *Child) fakeSignal(os.Signal) error {
	if c.Done() {
		return fmt.Errorf("ptychild: signal a child that has exited")
	}
	return nil
}
