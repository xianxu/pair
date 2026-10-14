package couchcmd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

// This rig uses the real startup policy, declared dispatch, existing operation
// queue, reattach pass, adoption and terminal input. Only external processes and
// the host terminal are fake; every store and hub directory is private.
type hubStartupRig struct {
	rt       testRT
	c        *couchcore.Couch
	console  *couchtty.Console
	host     *hostty.FakeHost
	input    *io.PipeWriter
	first    couchcore.StartResult
	hubs     *hubRuntime
	paths    []string
	calls    chan dispatchedCall
	mu       sync.Mutex
	children []string
	history  []dispatchedCall
}

func newHubStartupRig(t *testing.T, detached bool) *hubStartupRig {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	paths := []string{filepath.Join(root, "ops"), filepath.Join(root, "bench")}
	catalog := &couchcore.SlotCatalogFake{Workspaces: map[string]couchcore.WorkspaceIdentity{}, Repositories: map[string]couchcore.SlotRepository{}}
	for _, p := range paths {
		if err := os.Mkdir(p, 0700); err != nil {
			t.Fatal(err)
		}
		number, address, branch := 0, filepath.Base(p)+":0", "main"
		identity := couchcore.WorkspaceIdentity{SchemaVersion: 2, Repo: filepath.Base(p), RepoIdentity: filepath.Join(p, ".git"), PrimaryRoot: p, FleetRoot: root, EnvironmentRoot: root, WorktreeRoot: p, Kind: "primary", Slot: &number, Address: &address, RestingBranch: &branch}
		catalog.Workspaces[p] = identity
		catalog.Repositories[p] = couchcore.SlotRepository{Identity: identity}
	}
	rt := newRT(t, paths...)
	rt.runner = couchcore.NewFakeRunner()
	if detached {
		thread := seedDetachedThread(t, rt, paths[1])
		rt.artifacts.SetDetachedSession(thread.Address, "pair-"+string(thread.Address.Tag))
	}
	c, err := rt.NewCouch()
	if err != nil {
		t.Fatal(err)
	}
	c.Slots = catalog
	c.Layout = couchcore.Layout2
	host := hostty.NewFakeHost(ptychild.Size{Rows: 30, Cols: 600})
	reader, input := io.Pipe()
	console := couchtty.New(host, reader)
	rig := &hubStartupRig{rt: rt, c: c, console: console, host: host, input: input, paths: paths, calls: make(chan dispatchedCall, 32)}
	rt.runner.AfterBlockedStart = func(id string) {
		rig.mu.Lock()
		rig.children = append(rig.children, id)
		rig.mu.Unlock()
		rt.runner.Terminal(id).SetSink(func(ctx context.Context, batch ptychild.OutputBatch) error { return console.Deliver(ctx, id, batch) })
	}
	wireResolver(console, c)
	first, err := c.StartInteractive(context.Background(), couchcore.StartArgs{Cwd: paths[0]})
	if err != nil {
		t.Fatal(err)
	}
	rig.first = first
	hubs, err := prepareExplicitHubs(context.Background(), c, paths)
	if err != nil {
		t.Fatal(err)
	}
	rig.hubs = hubs
	actual := console.Ops()
	console.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
		rig.mu.Lock()
		rig.history = append(rig.history, dispatchedCall{call, nil})
		rig.mu.Unlock()
		v, e := actual(call)
		rig.calls <- dispatchedCall{call, e}
		return v, e
	})
	if err := beginConsole(console, first, true); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		console.Stop()
		_ = input.Close()
		_ = reader.Close()
		rig.mu.Lock()
		children := append([]string(nil), rig.children...)
		rig.mu.Unlock()
		for _, id := range children {
			rt.runner.SetExited(id, 0)
		}
	})
	return rig
}

func (r *hubStartupRig) run(t *testing.T) {
	t.Helper()
	done := make(chan int, 1)
	go func() { done <- r.console.Run() }()
	t.Cleanup(func() {
		r.console.Stop()
		_ = r.input.Close()
		select {
		case <-done:
		case <-testwait.After(t):
			t.Error("hub console did not stop")
		}
	})
}
func (r *hubStartupRig) await(t *testing.T, name string) dispatchedCall {
	t.Helper()
	for {
		select {
		case call := <-r.calls:
			if call.call.Name == name {
				return call
			}
		case <-testwait.After(t):
			t.Fatalf("no %s completion; terminal: %s", name, r.host.Written())
		}
	}
}
func (r *hubStartupRig) assertFirstUsable(t *testing.T) {
	t.Helper()
	terminal := r.first.Handle.(couchcore.TerminalHandle).Terminal()
	if _, err := r.input.Write([]byte("FIRST-HUB-INPUT")); err != nil {
		t.Fatal(err)
	}
	waitWarmAcceptance(t, "first hub retains keyboard focus", func() bool { return bytes.Contains(bytes.Join(terminal.Writes(), nil), []byte("FIRST-HUB-INPUT")) })
	terminal.Feed([]byte("FIRST-HUB-OUTPUT"))
	waitWarmAcceptance(t, "first hub remains visible", func() bool { return strings.Contains(r.host.Written(), "FIRST-HUB-OUTPUT") })
}

func TestHubStartupTwoHubsPreserveFirstFocus(t *testing.T) {
	r := newHubStartupRig(t, false)
	startExplicitHubs(r.console, r.hubs)
	r.run(t)
	if call := r.await(t, "start-hub"); call.err != nil {
		t.Fatal(call.err)
	}
	r.await(t, "attach")
	r.assertFirstUsable(t)
	r.mu.Lock()
	count := len(r.children)
	r.mu.Unlock()
	if count != 2 {
		t.Fatalf("started %d children, want exactly two", count)
	}
}

func TestHubStartupLaterFailureLeavesFirstUsable(t *testing.T) {
	r := newHubStartupRig(t, false)
	r.rt.runner.FailNextStart(errors.New("isolated second hub launch refusal"))
	startExplicitHubs(r.console, r.hubs)
	r.run(t)
	if call := r.await(t, "start-hub"); call.err == nil {
		t.Fatal("second launch unexpectedly succeeded")
	}
	waitWarmAcceptance(t, "visible hub refusal", func() bool { return strings.Contains(r.host.Written(), "failed:") })
	r.assertFirstUsable(t)
	r.mu.Lock()
	count := len(r.children)
	r.mu.Unlock()
	if count != 1 {
		t.Fatalf("failed hub spawned %d children", count)
	}
}

func TestHubStartupDetachedExtraIsNotAlsoReattachedByPass(t *testing.T) {
	r := newHubStartupRig(t, true)
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	t.Cleanup(func() { once.Do(func() { close(release) }) })
	actual := r.console.Ops()
	r.console.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
		if call.Name == "start-hub" {
			close(entered)
			select {
			case <-release:
			case <-call.Context.Done():
				return nil, call.Context.Err()
			}
		}
		return actual(call)
	})
	startExplicitHubs(r.console, r.hubs)
	r.run(t)
	select {
	case <-entered:
	case <-testwait.After(t):
		t.Fatal("extra hub did not enter queue")
	}
	if _, err := r.input.Write([]byte{0}); err != nil {
		t.Fatal(err)
	}
	waitWarmAcceptance(t, "background inventory includes detached extra hub", func() bool {
		return strings.Contains(r.host.Written(), "bench") && strings.Contains(r.host.Written(), "threads")
	})
	if _, err := r.input.Write([]byte("\x1b")); err != nil {
		t.Fatal(err)
	}
	once.Do(func() { close(release) })
	if call := r.await(t, "start-hub"); call.err != nil {
		t.Fatal(call.err)
	}
	r.await(t, "attach")
	// A barrier queued after inventory and adoption observes admission past
	// that prefix. Record dispatcher entries before their potentially slow
	// effects, because queue workers can execute concurrently.
	barrier := make(chan struct{})
	if err := r.console.EnqueueRemoteOperation("startup-test-barrier", "list", func(ctx context.Context) (couchcore.OperationCall, error) {
		return couchcore.OperationCall{Name: "list"}, nil
	}, func() { close(barrier) }, func(any, error) {}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-barrier:
	case <-testwait.After(t):
		t.Fatal("queue barrier did not run")
	}
	r.mu.Lock()
	history := append([]dispatchedCall(nil), r.history...)
	r.mu.Unlock()
	for _, call := range history {
		if call.call.Name == "resume" {
			t.Fatalf("extra hub was also admitted to reattach pass: %+v, %v", call.call, call.err)
		}
	}
	terminal := r.first.Handle.(couchcore.TerminalHandle).Terminal()
	terminal.Feed([]byte("INVENTORY-FOCUS-RETURNED"))
	waitWarmAcceptance(t, "switcher returned to first hub", func() bool { return strings.Contains(r.host.Written(), "INVENTORY-FOCUS-RETURNED") })
	r.assertFirstUsable(t)
}

func TestHubStartupSingleHubDoesNotEnqueueAnotherLaunch(t *testing.T) {
	r := newHubStartupRig(t, false)
	// The existing no-arguments branch never calls startExplicitHubs. Even an
	// explicit single-directory launch has no extra job and no alternate loop.
	single, err := prepareExplicitHubs(context.Background(), r.c, r.paths[:1])
	if err != nil {
		t.Fatal(err)
	}
	startExplicitHubs(r.console, single)
	r.run(t)
	r.assertFirstUsable(t)
	if !r.console.ReattachPassArmed() {
		t.Fatal("lost existing single background pass")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.children) != 1 {
		t.Fatalf("single hub started %d children", len(r.children))
	}
	for _, call := range r.history {
		if call.call.Name == "start-hub" {
			t.Fatal("single hub queued extra launch")
		}
	}
}
