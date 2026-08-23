package couchcore

import (
	"fmt"
	"os"
	"sync"

	"github.com/xianxu/pair/cmd/internal/ptychild"
)

// FakeChild is the fake's per-child state, modelled across calls.
type FakeChild struct {
	Dir     string
	Argv    []string
	Env     []string
	Signals []os.Signal
	diesOn  map[os.Signal]int
	alive   bool
	code    int
	done    chan struct{}

	// terminal is the pty double. It is a real *ptychild.Child in its fake
	// mode -- the SAME type PtyRunner hands out -- so the console cannot be
	// exercising a different shape in tests than in production (ARCH-MOCK).
	terminal *ptychild.Child
}

// FakeRunner is the stateful double ARCH-MOCK requires.
//
// Contract, fixed here rather than inferred from tests:
//   - Start records {argv, dir, env}, marks the child alive, and returns a
//     handle with a deterministic id (couch-fake-N).
//   - Signal appends to the child's signal log and does NOT kill it.
//   - SetExited(id, code) is the only thing that ends a child; it unblocks Wait.
//   - Wait blocks until exited; returns immediately if already exited.
//   - Handles record into the Runner's Ops log, not their own, so ordering
//     across children is assertable.
type FakeRunner struct {
	mu       sync.Mutex
	children map[string]*FakeChild
	order    []string
	failNext error
	autoExit *int
	Ops      []string

	// Sink mirrors PtyRunner's: installed on each child at Start, tagged with
	// the handle id.
	Sink func(id string, chunk []byte)
}

var _ Runner = (*FakeRunner)(nil)

func NewFakeRunner() *FakeRunner {
	return &FakeRunner{children: map[string]*FakeChild{}}
}

// FailNextStart makes the next Start return err, so a caller's cleanup path
// can be exercised without a real process failure.
func (f *FakeRunner) FailNextStart(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failNext = err
}

func (f *FakeRunner) Start(dir string, argv, env []string) (Handle, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failNext != nil {
		err := f.failNext
		f.failNext = nil
		return nil, err
	}
	id := fmt.Sprintf("couch-fake-%d", len(f.order)+1)
	child := ptychild.NewFakeChild(nil)
	if f.Sink != nil {
		sink := f.Sink
		child.SetSink(func(chunk []byte) { sink(id, chunk) })
	}
	f.children[id] = &FakeChild{
		Dir: dir, Argv: argv, Env: env,
		diesOn: map[os.Signal]int{},
		alive:  true, done: make(chan struct{}),
		terminal: child,
	}
	f.order = append(f.order, id)
	f.Ops = append(f.Ops, "start "+dir+": "+joinArgs(argv))
	if f.autoExit != nil {
		c := f.children[id]
		c.alive, c.code = false, *f.autoExit
		close(c.done)
	}
	return &fakeHandle{runner: f, id: id}, nil
}

// AutoExit makes every subsequent Start return an already-exited child.
//
// It exists because `couch start` blocks on Handle.Wait for the child's
// lifetime, which is right in production and makes a CLI test hang forever
// against a fake that never finishes. Modelling "the child ran and exited" is
// the honest way to drive that path.
func (f *FakeRunner) AutoExit(code int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.autoExit = &code
}

// SetDiesOn scripts a child's disposition for one signal: receiving it exits
// the child with code.
//
// The default is that NO signal kills, which is the conservative model -- a
// real process may catch, ignore or delay one, and pair's own restart loop
// depends on catching SIGUSR2. Scripting the other disposition explicitly is
// what lets the live conformance check compare both against real processes
// rather than assuming one.
func (f *FakeRunner) SetDiesOn(id string, sig os.Signal, code int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if c, ok := f.children[id]; ok {
		c.diesOn[sig] = code
	}
}

// SetExited ends a child and unblocks any Wait on it.
func (f *FakeRunner) SetExited(id string, code int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	c, ok := f.children[id]
	if !ok || !c.alive {
		return
	}
	c.alive, c.code = false, code
	close(c.done)
}

func (f *FakeRunner) Child(id string) FakeChild {
	f.mu.Lock()
	defer f.mu.Unlock()
	if c, ok := f.children[id]; ok {
		return *c
	}
	return FakeChild{}
}

func (f *FakeRunner) Signals(id string) []os.Signal {
	f.mu.Lock()
	defer f.mu.Unlock()
	if c, ok := f.children[id]; ok {
		return append([]os.Signal(nil), c.Signals...)
	}
	return nil
}

type fakeHandle struct {
	runner *FakeRunner
	id     string
}

func (h *fakeHandle) ID() string { return h.id }

// Terminal makes the fake handle a TerminalHandle, exactly as PtyRunner's is.
// A console test that type-asserts the capability therefore takes the same
// branch production takes.
func (h *fakeHandle) Terminal() *ptychild.Child { return h.runner.Terminal(h.id) }

var _ TerminalHandle = (*fakeHandle)(nil)

func (h *fakeHandle) PID() int {
	h.runner.mu.Lock()
	defer h.runner.mu.Unlock()
	for i, id := range h.runner.order {
		if id == h.id {
			return 1000 + i
		}
	}
	return 0
}

func (h *fakeHandle) Identity() string { return "fake-identity-" + h.id }

func (h *fakeHandle) Alive() bool {
	h.runner.mu.Lock()
	defer h.runner.mu.Unlock()
	c, ok := h.runner.children[h.id]
	return ok && c.alive
}

func (h *fakeHandle) Signal(sig os.Signal) error {
	h.runner.mu.Lock()
	defer h.runner.mu.Unlock()
	c, ok := h.runner.children[h.id]
	if !ok {
		return fmt.Errorf("fake runner: no child %s", h.id)
	}
	c.Signals = append(c.Signals, sig)
	h.runner.Ops = append(h.runner.Ops, "signal "+h.id+": "+sig.String())
	if code, fatal := c.diesOn[sig]; fatal && c.alive {
		c.alive, c.code = false, code
		close(c.done)
	}
	return nil
}

func (h *fakeHandle) Wait() int {
	h.runner.mu.Lock()
	c, ok := h.runner.children[h.id]
	h.runner.mu.Unlock()
	if !ok {
		return -1
	}
	<-c.done
	h.runner.mu.Lock()
	defer h.runner.mu.Unlock()
	return c.code
}

// Emit pushes output from a fake child, the stand-in for its pty producing
// bytes. It runs the same path a real child's pump does -- ring, screen, sink --
// because they are the same type.
func (f *FakeRunner) Emit(id string, chunk []byte) {
	f.mu.Lock()
	c, ok := f.children[id]
	f.mu.Unlock()
	if ok && c.terminal != nil {
		c.terminal.Feed(chunk)
	}
}

// Terminal exposes the pty double, so a fakeHandle can satisfy TerminalHandle.
func (f *FakeRunner) Terminal(id string) *ptychild.Child {
	f.mu.Lock()
	defer f.mu.Unlock()
	if c, ok := f.children[id]; ok {
		return c.terminal
	}
	return nil
}
