package couchcore

import (
	"fmt"
	"os"
	"sync"
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
	Ops      []string
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
	f.children[id] = &FakeChild{
		Dir: dir, Argv: argv, Env: env,
		diesOn: map[os.Signal]int{},
		alive:  true, done: make(chan struct{}),
	}
	f.order = append(f.order, id)
	f.Ops = append(f.Ops, "start "+dir+": "+joinArgs(argv))
	return &fakeHandle{runner: f, id: id}, nil
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
