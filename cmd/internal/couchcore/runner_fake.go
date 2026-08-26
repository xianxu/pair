package couchcore

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/ptychild"
)

// FakeChild is the fake's per-child state, modelled across calls.
type FakeChild struct {
	Dir       string
	Argv      []string
	Env       []string
	Signals   []os.Signal
	diesOn    map[os.Signal]int
	alive     bool
	code      int
	done      chan struct{}
	Blocked   bool
	ExecCount int

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
	return f.start(dir, argv, env, false)
}

func (f *FakeRunner) StartBlocked(dir string, argv, env []string, _ time.Duration) (BlockedHandle, error) {
	h, err := f.start(dir, argv, env, true)
	if err != nil {
		return nil, err
	}
	return &fakeBlockedHandle{fakeHandle: h.(*fakeHandle)}, nil
}

func (f *FakeRunner) start(dir string, argv, env []string, blocked bool) (Handle, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failNext != nil {
		err := f.failNext
		f.failNext = nil
		return nil, err
	}
	id := fmt.Sprintf("couch-fake-%d", len(f.order)+1)
	// The terminal double is a real *ptychild.Child in fake mode, so a test
	// that needs this child to produce output calls Feed on it directly --
	// there is no second emit path to keep in step.
	child := ptychild.NewFakeChild(nil)
	f.children[id] = &FakeChild{
		Dir: dir, Argv: argv, Env: env,
		diesOn: map[os.Signal]int{},
		alive:  true, done: make(chan struct{}),
		Blocked:  blocked,
		terminal: child,
	}
	f.order = append(f.order, id)
	f.Ops = append(f.Ops, "start "+dir+": "+joinArgs(argv))
	if f.autoExit != nil && !blocked {
		c := f.children[id]
		c.alive, c.code = false, *f.autoExit
		close(c.done)
		// The TERMINAL double ends with the child. A fake with two notions of
		// "exited" -- one for the handle, one for its pty -- lets a console
		// test hang forever waiting on the half that never ends, which is
		// exactly how this was found.
		c.terminal.Exit(*f.autoExit)
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
	// End the terminal double with it. One child, one notion of "exited": a
	// fake whose handle has exited while its pty is still running lets a
	// console test hang forever on the half that never ends.
	c.terminal.Exit(code)
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

type fakeBlockedHandle struct {
	*fakeHandle
}

func (h *fakeBlockedHandle) Acknowledge() error {
	h.runner.mu.Lock()
	defer h.runner.mu.Unlock()
	c, ok := h.runner.children[h.id]
	if !ok || !c.alive || !c.Blocked {
		return fmt.Errorf("fake runner: blocked start %s already resolved", h.id)
	}
	c.Blocked = false
	c.ExecCount++
	h.runner.Ops = append(h.runner.Ops, "ack "+h.id)
	if h.runner.autoExit != nil {
		c.alive, c.code = false, *h.runner.autoExit
		close(c.done)
		c.terminal.Exit(*h.runner.autoExit)
	}
	return nil
}

func (h *fakeBlockedHandle) Cancel() error {
	h.runner.mu.Lock()
	defer h.runner.mu.Unlock()
	c, ok := h.runner.children[h.id]
	if !ok || !c.alive || !c.Blocked {
		return fmt.Errorf("fake runner: blocked start %s already resolved", h.id)
	}
	c.Blocked = false
	c.alive = false
	c.code = 1
	close(c.done)
	c.terminal.Exit(1)
	h.runner.Ops = append(h.runner.Ops, "cancel "+h.id)
	return nil
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

// Terminal exposes the pty double, so a fakeHandle can satisfy TerminalHandle.
func (f *FakeRunner) Terminal(id string) *ptychild.Child {
	f.mu.Lock()
	defer f.mu.Unlock()
	if c, ok := f.children[id]; ok {
		return c.terminal
	}
	return nil
}
