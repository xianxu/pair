package couchcmd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

// This rig uses the real startup policy, declared dispatch, existing operation
// queue, adoption and terminal input. Only external processes and
// the host terminal are fake; every store and hub directory is private.
type hubStartupRig struct {
	rt       testRT
	c        *couchcore.Couch
	console  *couchtty.Console
	host     *hostty.FakeHost
	input    *io.PipeWriter
	first    couchcore.StartResult
	startup  couchcore.StartupObservation
	hubs     *hubRuntime
	paths    []string
	calls    chan dispatchedCall
	mu       sync.Mutex
	children []string
	history  []dispatchedCall
}

// The legacy detached flag remains for shared startup fixtures; every hub
// terminal must now be explicitly seeded as surviving.
func newHubStartupRig(t *testing.T, _ bool, beforeAdoption ...func(*hubStartupRig)) *hubStartupRig {
	return newHubStartupFixture(t, true, true, beforeAdoption...)
}

func newHubStartupRigWithFirst(t *testing.T, firstSurvived bool, beforeAdoption ...func(*hubStartupRig)) *hubStartupRig {
	return newHubStartupFixture(t, firstSurvived, true, beforeAdoption...)
}

func newHubStartupOwnershipRig(t *testing.T, _ bool, retainedOwnership bool, beforeAdoption ...func(*hubStartupRig)) *hubStartupRig {
	return newHubStartupFixture(t, true, retainedOwnership, beforeAdoption...)
}

func newHubStartupFixture(t *testing.T, firstSurvived, retainedOwnership bool, beforeAdoption ...func(*hubStartupRig)) *hubStartupRig {
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
	// This live ordinary slot is visible to startup and joins the warm-only
	// background pass; explicit hubs remain independently queued.
	ordinary := filepath.Join(root, "ordinary-slot1")
	if err := os.Mkdir(ordinary, 0700); err != nil {
		t.Fatal(err)
	}
	number, address, branch := 1, "ops:1", "main-slot1"
	ordinaryIdentity := couchcore.WorkspaceIdentity{SchemaVersion: 2, Repo: "ops", RepoIdentity: filepath.Join(paths[0], ".git"), PrimaryRoot: paths[0], FleetRoot: root, EnvironmentRoot: root, WorktreeRoot: ordinary, Kind: "worktree", Slot: &number, Address: &address, RestingBranch: &branch}
	catalog.Workspaces[ordinary] = ordinaryIdentity
	rt := newRT(t, append(append([]string(nil), paths...), ordinary)...)

	rt.runner = couchcore.NewFakeRunner()
	// Exercise both owned-helper repair and routine borrowed reattachment.
	// Missing unique helper proof preserves lifecycle state and remains feedback.
	ordinaryScope, err := launcher.ResolveRepoScope(ordinary)
	if err != nil {
		t.Fatal(err)
	}
	seed, err := rt.NewCouch()
	if err != nil {
		t.Fatal(err)
	}
	profile := &couchcore.LaunchProfile{Agent: "claude", Argv: []string{}}
	var incarnations []couchcore.ThreadIncarnation
	if retainedOwnership {
		incarnations = []couchcore.ThreadIncarnation{{PID: 5151, Identity: "old-ordinary", State: couchcore.IncarnationLive, LaunchProfile: profile}}
	}
	other, err := seed.Threads.CreateThread(couchcore.ThreadRecord{
		SchemaVersion: couchcore.ThreadSchemaVersion,
		Address:       couchcore.ThreadAddress{RepoScope: ordinaryScope.Key, Tag: "couch-0102030405060708"},
		StartingPath:  ordinary, WorkingPath: ordinary, CreatedAt: time.Unix(1, 0).UTC(), Revision: 1,
		LatestLaunchProfile: profile,
		Incarnations:        incarnations,
	})
	if err != nil {
		t.Fatal(err)
	}
	rt.artifacts.SetDetachedSession(other.Address, "pair-"+other.Address.RepoScope+"-"+string(other.Address.Tag))
	// Directory selection authorizes only surviving terminal attachment. These
	// fixtures explicitly seed both hub sessions rather than relying on Fresh.
	for index, path := range paths {
		if index == 0 && !firstSurvived {
			continue
		}
		thread := seedDetachedThread(t, rt, path)
		name := "pair-" + thread.Address.RepoScope + "-" + string(thread.Address.Tag)
		rt.artifacts.SetDetachedSession(thread.Address, name)
		rt.artifacts.SetSessionPresence(thread.Address, couchcore.SessionObservation{State: couchcore.SessionPresent, Name: name})
	}
	c, err := rt.NewCouch()
	if err != nil {
		t.Fatal(err)
	}
	c.Slots = catalog
	c.Layout = couchcore.Layout2
	host := hostty.NewFakeHost(ptychild.Size{Rows: 30, Cols: 100})
	reader, input := io.Pipe()
	console := couchtty.New(host, reader)
	rig := &hubStartupRig{rt: rt, c: c, console: console, host: host, input: input, paths: paths, calls: make(chan dispatchedCall, 32)}
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
	// Pair's acknowledged launch creates a positively listed Zellij session.
	// A local fake terminal alone must not impersonate that external fact.
	rt.runner.AfterAcknowledge = func(id string) error {
		child := rt.runner.Child(id)
		address := couchcore.ThreadAddress{RepoScope: envValue(child.Env, "COUCH_THREAD_SCOPE"), Tag: couchcore.ThreadTag(envValue(child.Env, "COUCH_THREAD_TAG"))}
		rt.artifacts.SetPairSession(address, managedChildSession(t, rt.runner, id), true)
		rt.artifacts.SetSessionPresence(address, couchcore.SessionObservation{State: couchcore.SessionPresent, Name: managedChildSession(t, rt.runner, id)})
		return nil
	}
	rt.runner.AfterBlockedStart = func(id string) {
		rig.mu.Lock()
		rig.children = append(rig.children, id)
		rig.mu.Unlock()
		rt.runner.Terminal(id).SetSink(func(ctx context.Context, batch ptychild.OutputBatch) error { return console.Deliver(ctx, id, batch) })
	}
	hubs, err := prepareExplicitHubs(context.Background(), c, paths)
	if err != nil {
		t.Fatal(err)
	}
	rig.hubs = hubs
	first, err := c.StartInteractive(context.Background(), couchcore.StartArgs{Cwd: paths[0]})
	if err != nil {
		t.Fatal(err)
	}
	rig.first = first
	startup, err := c.ObserveStartupContext(context.Background(), paths)
	if err != nil {
		t.Fatal(err)
	}
	rig.startup = startup
	console.SetHubOwnership(hubs.snapshot())
	for _, setup := range beforeAdoption {
		setup(rig)
	}
	wireResolver(console, c, hubs)
	actual := console.Ops()
	console.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
		rig.mu.Lock()
		rig.history = append(rig.history, dispatchedCall{call, nil})
		rig.mu.Unlock()
		v, e := actual(call)
		rig.calls <- dispatchedCall{call, e}
		return v, e
	})
	if err := beginConsole(console, first, !startup.Hubs[paths[0]].Survived); err != nil {
		t.Fatal(err)
	}
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
	waitWarmAcceptance(t, "warm worker starts beside hubs", func() bool { r.mu.Lock(); defer r.mu.Unlock(); return len(r.children) == 3 })
	r.mu.Lock()
	count := len(r.children)
	r.mu.Unlock()
	if count != 3 {
		t.Fatalf("started %d children, want two hubs and one warm worker", count)
	}
}

type hubRefusingRunner struct {
	couchcore.Runner
	path string
}

func (r hubRefusingRunner) StartBlocked(ctx context.Context, dir string, argv, env []string, timeout time.Duration) (couchcore.BlockedHandle, error) {
	if dir == r.path {
		return nil, errors.New("isolated second hub launch refusal")
	}
	return r.Runner.StartBlocked(ctx, dir, argv, env, timeout)
}

func TestHubStartupLaterFailureLeavesFirstUsable(t *testing.T) {
	testHubStartupLaterFailure(t, false)
}

// The explicit hub queue and the warm-only pass start children independently,
// in no defined order (pair#697). Holding the second hub's start until the warm
// worker has started forces the order the failed run took.
func TestHubStartupLaterFailureAfterWarmWorkerLeavesFirstUsable(t *testing.T) {
	testHubStartupLaterFailure(t, true)
}

func testHubStartupLaterFailure(t *testing.T, warmFirst bool) {
	r := newHubStartupRig(t, false)
	// Warm attachment owns one retry; both helper attempts must fail here.
	r.c.Runner = hubRefusingRunner{Runner: r.rt.runner, path: r.paths[1]}
	if warmFirst {
		actual := r.console.Ops()
		r.console.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
			if call.Name == "start-hub" {
				testwait.Until(t, "warm worker starts before the second hub", func() bool {
					r.mu.Lock()
					defer r.mu.Unlock()
					return len(r.children) >= 2
				})
			}
			v, e := actual(call)
			r.calls <- dispatchedCall{call, e}
			return v, e
		})
	}
	startExplicitHubs(r.console, r.hubs)
	r.run(t)
	call := r.await(t, "start-hub")
	if call.err == nil || !strings.Contains(call.err.Error(), "isolated second hub launch refusal") || !strings.Contains(call.err.Error(), r.paths[1]) || !strings.Contains(call.err.Error(), "diagnostics ") {
		t.Fatalf("second launch did not report its injected refusal: %v", call.err)
	}
	// Startup failure is an attention item; ordinary tab rendering no longer
	// carries the general notice feed.
	testwait.Until(t, "startup failure attention", func() bool {
		return strings.Contains(r.host.Written(), couchtty.AttentionGlyph+"1")
	})
	openWarningAttention(t, r.input, r.host, "hub failed")
	if _, err := r.input.Write([]byte("\x1b[97;7u")); err != nil {
		t.Fatal(err)
	}
	r.assertFirstUsable(t)
	waitWarmAcceptance(t, "warm worker starts despite failed hub", func() bool { r.mu.Lock(); defer r.mu.Unlock(); return len(r.children) == 2 })
	r.mu.Lock()
	count := len(r.children)
	r.mu.Unlock()
	if count != 2 {
		t.Fatalf("failed hub should retain first hub plus warm worker, got %d children", count)
	}
}

func TestHubStartupDetachedExtraLaunchesWithoutDuplicateWarmAttempt(t *testing.T) {
	r := newHubStartupRig(t, false)
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
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
	// LIFO cleanup releases the test barrier before Run cleanup joins workers.
	t.Cleanup(func() { once.Do(func() { close(release) }) })
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
	benchScope, err := launcher.ResolveRepoScope(r.paths[1])
	if err != nil {
		t.Fatal(err)
	}
	for _, call := range history {
		if call.call.Name == "resume" && call.call.Args["repo-scope"] == benchScope.Key {
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
	// The single-hub fixture must not leave its second seeded terminal as an
	// ordinary warm worker once ownership narrows to the first root.
	extraScope, _ := launcher.ResolveRepoScope(r.paths[1])
	r.rt.artifacts.SetDetachedSession(couchcore.ThreadAddress{RepoScope: extraScope.Key, Tag: "couch-0102030405060708"}, "")

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
		t.Fatal("single hub did not arm warm worker attachment")
	}
	waitWarmAcceptance(t, "single hub starts warm worker", func() bool { r.mu.Lock(); defer r.mu.Unlock(); return len(r.children) == 2 })
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.children) != 2 {
		t.Fatalf("single hub plus warm worker started %d children", len(r.children))
	}
	for _, call := range r.history {
		if call.call.Name == "start-hub" {
			t.Fatal("single hub queued extra launch")
		}
	}
}

func TestHubStartupMissingFirstKeepsSwitcherWhileExtraSurvives(t *testing.T) {
	r := newHubStartupRigWithFirst(t, false)
	screen := startupWarmScreen(t, 100, 30, r.host.Written)
	if r.first.Handle != nil || r.startup.Kind != couchcore.StartupNormal || r.startup.Hubs[r.paths[0]].Survived || !r.startup.Hubs[r.paths[1]].Survived {
		t.Fatalf("mixed startup: first=%+v observation=%+v", r.first, r.startup)
	}
	if !r.console.ReattachPassArmed() {
		t.Fatal("missing first hub prevented warm pass")
	}
	startExplicitHubs(r.console, r.hubs)
	r.run(t)
	if call := r.await(t, "start-hub"); call.err != nil {
		t.Fatal(call.err)
	}
	waitWarmAcceptance(t, "surviving extra and worker attached", func() bool { r.mu.Lock(); defer r.mu.Unlock(); return len(r.children) == 2 })
	waitWarmAcceptance(t, "missing first leaves switcher focused", func() bool { return strings.Contains(r.host.Written(), "threads") })
	if _, err := r.input.Write([]byte("SWITCHER-INPUT")); err != nil {
		t.Fatal(err)
	}
	waitWarmAcceptance(t, "switcher receives input while extra stays background", func() bool { return strings.Contains(screen(), "SWITCHER-INPUT") })
	r.mu.Lock()
	children := append([]string(nil), r.children...)
	r.mu.Unlock()
	for _, id := range children {
		child := r.rt.runner.Child(id)
		if strings.Contains(strings.Join(child.Argv, " "), "--layout") {
			t.Fatalf("implicit startup created native agent: %v", child.Argv)
		}
		if len(r.rt.runner.Terminal(id).Writes()) != 0 {
			t.Fatalf("background hub stole switcher input: %s", id)
		}
	}
}

func TestStartupSummaryDescribesMissingHubAndAutomation(t *testing.T) {
	observation := couchcore.StartupObservation{Kind: couchcore.StartupNormal, Hubs: map[string]couchcore.StartupHubObservation{"/first": {}, "/extra": {Survived: true}}, MissingHubs: []string{"/first"}, ParkedCount: 3}
	text := startupSummary(observation, "/first")
	for _, want := range []string{"normal", "missing hubs: /first", "verified parked: 3", "hub automation off; say turn it on"} {
		if !strings.Contains(text, want) {
			t.Fatalf("summary %q misses %q", text, want)
		}
	}
	if text = startupSummary(observation, "/extra"); !strings.Contains(text, "hubs /first (hub automation off; say turn it on)") || strings.Contains(text, "hubs /extra (hub automation off") {
		t.Fatalf("mixed startup failed to name only missing hub's disabled automation: %s", text)
	}
}

// startupPaintBarrierHost exposes the distinction between bytes reaching the
// host and the presenter completing the startup frame. It changes no production seam.
type startupPaintBarrierHost struct {
	*hostty.FakeHost
	entered   chan struct{}
	release   chan struct{}
	cancelled chan struct{}
	once      sync.Once
}

func (h *startupPaintBarrierHost) WriteContext(ctx context.Context, p []byte) (int, error) {
	n, err := h.FakeHost.WriteContext(ctx, p)
	if err != nil || !bytes.Contains(p, []byte("startup: normal")) {
		return n, err
	}
	h.once.Do(func() {
		close(h.entered)
		select {
		case <-h.release:
		case <-ctx.Done():
			close(h.cancelled)
		}
		err = ctx.Err()
	})
	return n, err
}

func TestRunConsoleObservesStartupBeforeArmingProducers(t *testing.T) {
	hubs, defs, _ := moveFixture(t)
	paths := []string{defs[0].Root, defs[1].Root, defs[2].Root}
	rt := newRT(t, paths...)
	retained := seedDetachedThread(t, rt, paths[0])
	name := "pair-" + retained.Address.RepoScope + "-" + string(retained.Address.Tag)
	rt.artifacts.SetDetachedSession(retained.Address, name)
	rt.artifacts.SetSessionPresence(retained.Address, couchcore.SessionObservation{State: couchcore.SessionPresent, Name: name})
	c, err := rt.NewCouch()
	if err != nil {
		t.Fatal(err)
	}
	host := &startupPaintBarrierHost{FakeHost: hostty.NewFakeHost(ptychild.Size{Rows: 30, Cols: 160}), entered: make(chan struct{}), release: make(chan struct{}), cancelled: make(chan struct{})}
	console := couchtty.New(host, strings.NewReader(""))
	beforeArm := 0
	rt.artifacts.SessionPresenceHook = func([]couchcore.ThreadAddress) error {
		if !console.ReattachPassArmed() {
			beforeArm++
		}
		return nil
	}
	var output bytes.Buffer
	done := make(chan int, 1)
	go func() {
		done <- runConsole(console, c, couchcore.StartResult{}, &output, hubs)
		close(done)
	}()
	t.Cleanup(func() {
		console.Stop()
		select {
		case <-done:
		case <-testwait.After(t):
			t.Error("startup console cleanup did not join Run")
		}
	})
	select {
	case <-host.entered:
	case <-testwait.After(t):
		t.Fatal("startup frame did not reach held write")
	}
	select {
	case code := <-done:
		t.Fatalf("console exited %d before startup paint was released", code)
	default:
	}
	// Queue a normal host termination while the startup paint is held. Only
	// Run handles it, after publication; Stop would cancel the paint itself.
	host.Terminate(syscall.SIGTERM)
	close(host.release)
	select {
	case code := <-done:
		if code != 0 {
			t.Fatalf("console startup=%d: %s", code, output.String())
		}
	case <-testwait.After(t):
		t.Fatal("startup console did not stop")
	}
	select {
	case <-host.cancelled:
		t.Fatal("startup paint cancelled before release")
	default:
	}
	if beforeArm != 1 {
		t.Fatalf("bulk startup observations before producers=%d", beforeArm)
	}
	if !console.ReattachPassArmed() || len(rt.runner.Ops) != 0 {
		t.Fatalf("missing hubs launched or warm pass unarmed: %v", rt.runner.Ops)
	}
	if !strings.Contains(host.Written(), "startup: normal") {
		t.Fatalf("startup notice missing: %s", host.Written())
	}
}

func TestBareCwdHubBindsSwitcherAutomationToVerifiedWorkspace(t *testing.T) {
	root, _ := filepath.EvalSymlinks(t.TempDir())
	primary := filepath.Join(root, "repo")
	layout := couchcore.NewSlotLayout(primary, "", 7)
	path := layout.Host()
	if err := os.MkdirAll(path, 0700); err != nil {
		t.Fatal(err)
	}
	t.Chdir(path)
	number, address, branch := 7, "repo:7", "main-slot7"
	identity := couchcore.WorkspaceIdentity{SchemaVersion: 2, Repo: "repo", RepoIdentity: filepath.Join(primary, ".git"), PrimaryRoot: primary, FleetRoot: root, EnvironmentRoot: layout.Env(), WorktreeRoot: path, Kind: "slot", Slot: &number, Address: &address, RestingBranch: &branch}
	rt := newRT(t, path)
	retained := seedDetachedThread(t, rt, path)
	name := "pair-" + retained.Address.RepoScope + "-" + string(retained.Address.Tag)
	rt.artifacts.SetDetachedSession(retained.Address, name)
	rt.artifacts.SetSessionPresence(retained.Address, couchcore.SessionObservation{State: couchcore.SessionPresent, Name: name})
	c, err := rt.NewCouch()
	if err != nil {
		t.Fatal(err)
	}
	catalog := &couchcore.SlotCatalogFake{Workspaces: map[string]couchcore.WorkspaceIdentity{path: identity}}
	c.Slots = catalog
	verified, roots, verifyErr := prepareConsoleHubs(context.Background(), c)
	if verifyErr != nil || verified == nil || len(roots) != 1 || roots[0] != path || verified.snapshot().Hubs()[0].Slot != (couchcore.HubSlot{Repository: primary, Number: 7}) {
		t.Fatalf("bare identity guessed instead of using verified :7: hubs=%v roots=%v err=%v", verified, roots, verifyErr)
	}
	before, _ := c.Threads.PreviewSnapshot()
	host := hostty.NewFakeHost(ptychild.Size{Rows: 30, Cols: 240})
	reader, input := io.Pipe()
	console := couchtty.New(host, reader)
	screen := startupWarmScreen(t, 240, 30, host.Written)
	controlled := make(chan string, 1)
	console.SetAutomationControl(func(hub, action string) (bool, error) { controlled <- hub + "|" + action; return true, nil })
	done := make(chan int, 1)
	go func() { done <- runConsole(console, c, couchcore.StartResult{}, io.Discard) }()
	t.Cleanup(func() {
		console.Stop()
		input.Close()
		reader.Close()
		select {
		case <-done:
		case <-testwait.After(t):
			t.Error("bare console did not stop")
		}
	})
	waitWarmAcceptance(t, "bare hub row ready", func() bool { return strings.Contains(screen(), "Continue") })
	if _, err := input.Write([]byte{7}); err != nil {
		t.Fatal(err)
	}
	got := ""
	waitWarmAcceptance(t, "bare hub toggle accepted or refused", func() bool {
		select {
		case got = <-controlled:
			return true
		default:
			return strings.Contains(screen(), "no verified hub selected")
		}
	})
	if got != path+"|toggle" {
		t.Fatalf("bare verified :7 hub toggle=%q, screen=%s", got, screen())
	}
	console.Stop()
	after, _ := c.Threads.PreviewSnapshot()
	if !reflect.DeepEqual(before, after) || len(rt.runner.Ops) != 0 || len(catalog.Calls) != 0 || rt.artifacts.BindingResolutions() != 0 {
		t.Fatalf("bare ownership binding changed stores or launched/provisioned: ops=%v discover=%v", rt.runner.Ops, catalog.Calls)
	}
}

func TestHubStartupEmptyConfiguredHubRemainsListed(t *testing.T) {
	r := newHubStartupRigWithFirst(t, false)
	rows, err := r.console.ActionableProvider()(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	state := couchtty.NewMenuState(rows, couchcore.ThreadAddress{})
	state, _ = couchtty.ReduceMenu(state, couchtty.MenuEvent{Kind: couchtty.MenuEventHubOwnership, HubOwnership: r.hubs.snapshot()})
	rendered := couchtty.RenderMenuView(state, 240, 30, time.Time{}, false)
	if !strings.Contains(rendered.Body, "Ctrl+Space: start "+r.paths[0]) {
		t.Fatalf("empty configured launch hub disappeared from startup switcher: hub=%s UI=%s", r.paths[0], rendered.Body)
	}
}

func TestBareCwdHubResolverFailureStaysUnknownAndUsable(t *testing.T) {
	path, _ := filepath.EvalSymlinks(t.TempDir())
	t.Chdir(path)
	rt := newRT(t, path)
	c, err := rt.NewCouch()
	if err != nil {
		t.Fatal(err)
	}
	catalog := &couchcore.SlotCatalogFake{Errors: map[string]error{path: errors.New("workspace identity unavailable")}}
	c.Slots = catalog
	hubs, roots, err := prepareConsoleHubs(context.Background(), c)
	if err == nil || hubs != nil || len(roots) != 1 || roots[0] != path {
		t.Fatalf("unverified hub was guessed: %v %v %v", hubs, roots, err)
	}
	want := startupSummary(couchcore.StartupObservation{
		Kind: couchcore.StartupUnknown, Diagnostic: err,
		MissingHubs: []string{path}, ParkedUnknown: true,
		Hubs: map[string]couchcore.StartupHubObservation{path: {Unknown: true}},
	}, path)
	before, _ := c.Threads.PreviewSnapshot()
	host := hostty.NewFakeHost(ptychild.Size{Rows: 30, Cols: 240})
	reader, writer := io.Pipe()
	console := couchtty.New(host, reader)
	assertCopied := noticeCopyAssertion(t, console)
	done := make(chan int, 1)
	go func() { done <- runConsole(console, c, couchcore.StartResult{}, io.Discard) }()
	t.Cleanup(func() {
		console.Stop()
		_ = writer.Close()
		select {
		case <-done:
		case <-testwait.After(t):
			t.Error("unknown startup console did not stop")
		}
	})
	waitWarmAcceptance(t, "unverified bare hub has visible unknown notice", func() bool { return strings.Contains(host.Written(), couchtty.ChromeStatusText(want)) })
	assertCopied(writer, want)
	if _, err := console.Ops()(couchcore.OperationCall{Name: "list"}); err != nil {
		t.Fatalf("unknown hub blocked switcher operations: %v", err)
	}
	console.Stop()
	after, _ := c.Threads.PreviewSnapshot()
	if !reflect.DeepEqual(before, after) || len(rt.runner.Ops) != 0 || len(catalog.Calls) != 0 || rt.artifacts.BindingResolutions() != 0 {
		t.Fatalf("unverified bare startup mutated or launched: ops=%v discover=%v", rt.runner.Ops, catalog.Calls)
	}
}

func TestBareOutsideCheckoutDoesNotGuessHubOwnership(t *testing.T) {
	t.Chdir(t.TempDir())
	rt := newRT(t)
	c, err := rt.NewCouch()
	if err != nil {
		t.Fatal(err)
	}
	c.Slots = &couchcore.SlotCatalogFake{}
	hubs, roots, err := prepareConsoleHubs(context.Background(), c)
	if err != nil || hubs != nil || len(roots) != 0 {
		t.Fatalf("outside checkout invented hub: %v %v %v", hubs, roots, err)
	}
}

func TestStartupSummaryMissingHubNamesExplicitRecovery(t *testing.T) {
	for _, kind := range []couchcore.StartupKind{couchcore.StartupFullShutdown, couchcore.StartupUnknown, couchcore.StartupNormal} {
		observation := couchcore.StartupObservation{Kind: kind, Hubs: map[string]couchcore.StartupHubObservation{"/ops": {}}, MissingHubs: []string{"/ops"}}
		text := startupSummary(observation, "/ops")
		for _, want := range []string{"switcher", "Resume from disk", "Ctrl+Space", "Start"} {
			if !strings.Contains(text, want) {
				t.Fatalf("%s missing-hub notice gives no recovery %q: %s", kind, want, text)
			}
		}
	}
	observation := couchcore.StartupObservation{Kind: couchcore.StartupNormal, Hubs: map[string]couchcore.StartupHubObservation{"/ops": {Survived: true}}}
	if text := startupSummary(observation, "/ops"); strings.Contains(text, "Resume from disk") {
		t.Fatalf("surviving hub was told to recover: %s", text)
	}
}
