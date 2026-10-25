package couchcmd

import (
	"context"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

// This runner controls the existing external-process boundary, not startup's
// queue or completion callbacks. Console.Run still owns all reductions.
type freshStartupHeldRunner struct {
	couchcore.Runner
	before func(context.Context, string) error
}

func (r freshStartupHeldRunner) StartBlocked(ctx context.Context, dir string, argv, env []string, timeout time.Duration) (couchcore.BlockedHandle, error) {
	if err := r.before(ctx, dir); err != nil {
		return nil, err
	}
	return r.Runner.StartBlocked(ctx, dir, argv, env, timeout)
}

// The targets are genuine catalog identities over private physical directories.
// Thread storage, declared dispatch, lifecycle workers and adoption are real.
func freshStartupRunFixture(t *testing.T) (*hubRuntime, []string, testRT, *couchcore.Couch, *couchtty.Console) {
	t.Helper()
	hubs, defs, _ := moveFixture(t)
	paths := make([]string, len(defs))
	catalog := &couchcore.SlotCatalogFake{Workspaces: map[string]couchcore.WorkspaceIdentity{}, Repositories: map[string]couchcore.SlotRepository{}}
	for i, def := range defs {
		paths[i] = def.Root
		n, address, branch := 0, filepath.Base(def.Root)+":0", "main"
		identity := couchcore.WorkspaceIdentity{SchemaVersion: 2, Repo: filepath.Base(def.Root), RepoIdentity: filepath.Join(def.Root, ".git"), PrimaryRoot: def.Root, FleetRoot: def.Workspace, EnvironmentRoot: def.Workspace, WorktreeRoot: def.Root, Kind: "primary", Slot: &n, Address: &address, RestingBranch: &branch}
		catalog.Workspaces[def.Root] = identity
		catalog.Repositories[def.Root] = couchcore.SlotRepository{Identity: identity}
	}
	rt := newRT(t, paths...)
	rt.runner = couchcore.NewFakeRunner()
	c, err := rt.NewCouch()
	if err != nil {
		t.Fatal(err)
	}
	c.Slots = catalog
	c.Layout = couchcore.Layout2
	host := hostty.NewFakeHost(ptychild.Size{Rows: 30, Cols: 120})
	reader, input := io.Pipe()
	console := couchtty.New(host, reader)
	var mu sync.Mutex
	var children []string
	rt.runner.AfterBlockedStart = func(id string) {
		mu.Lock()
		children = append(children, id)
		mu.Unlock()
		rt.runner.Terminal(id).SetSink(func(ctx context.Context, batch ptychild.OutputBatch) error { return console.Deliver(ctx, id, batch) })
	}
	rt.runner.AfterAcknowledge = func(id string) error {
		child := rt.runner.Child(id)
		addr := couchcore.ThreadAddress{RepoScope: envValue(child.Env, "COUCH_THREAD_SCOPE"), Tag: couchcore.ThreadTag(envValue(child.Env, "COUCH_THREAD_TAG"))}
		session := managedChildSession(t, rt.runner, id)
		rt.artifacts.SetPairSession(addr, session, true)
		rt.artifacts.SetSessionPresence(addr, couchcore.SessionObservation{State: couchcore.SessionPresent, Name: session})
		return nil
	}
	t.Cleanup(func() {
		console.Stop()
		_ = input.Close()
		_ = reader.Close()
		mu.Lock()
		defer mu.Unlock()
		for _, id := range children {
			rt.runner.SetExited(id, 0)
			_ = rt.runner.Terminal(id).Close()
		}
	})
	wireResolver(console, c, hubs)
	return hubs, paths, rt, c, console
}

func TestFreshStartupRunSerializesLaunchThroughAdoption(t *testing.T) {
	hubs, paths, rt, c, console := freshStartupRunFixture(t)
	initial, err := c.ObserveStartupContext(t.Context(), paths)
	if err != nil || initial.Kind != couchcore.StartupFresh {
		t.Fatalf("initial=%+v err=%v", initial, err)
	}
	first, err := startFreshPrimary(t.Context(), c, initial, paths[0], couchcore.StartResult{})
	if err != nil {
		t.Fatal(err)
	}
	if err = beginConsole(console, first); err != nil {
		t.Fatal(err)
	}
	launchEntered, launchRelease := make(chan struct{}), make(chan struct{})
	adoptEntered, adoptRelease := make(chan struct{}), make(chan struct{})
	var launchOnce, adoptOnce sync.Once
	releaseLaunch := func() { launchOnce.Do(func() { close(launchRelease) }) }
	releaseAdopt := func() { adoptOnce.Do(func() { close(adoptRelease) }) }
	t.Cleanup(releaseLaunch)
	t.Cleanup(releaseAdopt)
	var mu sync.Mutex
	var events []string
	record := func(s string) { mu.Lock(); events = append(events, s); mu.Unlock() }
	c.Runner = freshStartupHeldRunner{Runner: rt.runner, before: func(ctx context.Context, path string) error {
		record("launch:" + path)
		if path == paths[1] {
			close(launchEntered)
			select {
			case <-launchRelease:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	}}
	actual := console.Ops()
	failures := make(chan error, 8)
	console.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
		if call.Name == "attach" {
			start := call.TypedPayload.(couchcore.StartResult)
			path := string(start.Record.Args.Worktree)
			record("adopt:" + path)
			if path == paths[1] {
				close(adoptEntered)
				select {
				case <-adoptRelease:
				case <-call.Context.Done():
					return nil, call.Context.Err()
				}
			}
			value, err := actual(call)
			record("adopted:" + path)
			if err != nil {
				failures <- err
			}
			return value, err
		}
		value, err := actual(call)
		if err != nil {
			failures <- err
		}
		return value, err
	})
	startExplicitHubs(console, hubs, initial, c)
	done := make(chan int, 1)
	go func() { done <- console.Run(); close(done) }()
	t.Cleanup(func() {
		console.Stop()
		select {
		case <-done:
		case <-testwait.After(t):
			t.Error("fresh Run did not join")
		}
	})
	select {
	case <-launchEntered:
	case <-testwait.After(t):
		t.Fatal("extra launch not entered")
	}
	if got := len(console.ResourceInventory().Hosted); got != 1 {
		t.Fatalf("held launch has %d hosted children, want primary only", got)
	}
	releaseLaunch()
	select {
	case <-adoptEntered:
	case <-testwait.After(t):
		t.Fatal("extra adoption not entered")
	}
	if got := len(console.ResourceInventory().Hosted); got != 1 {
		t.Fatalf("held adoption has %d hosted children, want primary only", got)
	}
	releaseAdopt()
	var failed error
	testwait.Until(t, "three hubs adopted or definite effect failure", func() bool {
		select {
		case failed = <-failures:
			return true
		default:
		}
		// Installation publishes Hosted before the attach dispatcher returns.
		// Observe the completion marker too before asserting the final effect log.
		mu.Lock()
		completed := len(events) > 0 && events[len(events)-1] == "adopted:"+paths[2]
		mu.Unlock()
		return completed && len(console.ResourceInventory().Hosted) == 3
	})
	if failed != nil {
		mu.Lock()
		got := strings.Join(events, "\n")
		mu.Unlock()
		t.Fatalf("startup failed: %v; effects:\n%s", failed, got)
	}
	// A later live inventory must not retroactively change the startup decision.
	later, err := c.ObserveStartupContext(t.Context(), paths)
	if err != nil || later.Kind != couchcore.StartupNormal {
		t.Fatalf("later=%+v err=%v", later, err)
	}
	for _, path := range paths {
		if startupAutomationOn(&initial, path) {
			t.Fatalf("fresh hub %s gained automation after adoption", path)
		}
	}
	mu.Lock()
	got := strings.Join(events, "\n")
	mu.Unlock()
	want := strings.Join([]string{"launch:" + paths[1], "adopt:" + paths[1], "adopted:" + paths[1], "launch:" + paths[2], "adopt:" + paths[2], "adopted:" + paths[2]}, "\n")
	if got != want {
		t.Fatalf("startup effects not serial through adoption:\n%s\nwant:\n%s", got, want)
	}
}

func TestFreshStartupRunCancellationJoinsHeldEffect(t *testing.T) {
	for _, phase := range []string{"launch", "adoption"} {
		t.Run(phase, func(t *testing.T) {
			hubs, paths, rt, c, console := freshStartupRunFixture(t)
			initial, err := c.ObserveStartupContext(t.Context(), paths)
			if err != nil {
				t.Fatal(err)
			}
			first, err := startFreshPrimary(t.Context(), c, initial, paths[0], couchcore.StartResult{})
			if err != nil {
				t.Fatal(err)
			}
			if err = beginConsole(console, first); err != nil {
				t.Fatal(err)
			}
			entered, joined := make(chan struct{}), make(chan struct{})
			var mu sync.Mutex
			var launched []string
			c.Runner = freshStartupHeldRunner{Runner: rt.runner, before: func(ctx context.Context, path string) error {
				mu.Lock()
				launched = append(launched, path)
				mu.Unlock()
				if phase == "launch" && path == paths[1] {
					close(entered)
					<-ctx.Done()
					close(joined)
					return ctx.Err()
				}
				return nil
			}}
			actual := console.Ops()
			console.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
				if phase == "adoption" && call.Name == "attach" {
					close(entered)
					<-call.Context.Done()
					close(joined)
					return nil, call.Context.Err()
				}
				return actual(call)
			})
			startExplicitHubs(console, hubs, initial, c)
			done := make(chan int, 1)
			go func() { done <- console.Run(); close(done) }()
			t.Cleanup(func() {
				console.Stop()
				select {
				case <-done:
				case <-testwait.After(t):
					t.Error("cancelled Run did not join")
				}
			})
			select {
			case <-entered:
			case <-testwait.After(t):
				t.Fatal("held effect not entered")
			}
			console.Stop()
			select {
			case <-done:
			case <-testwait.After(t):
				t.Fatal("Run did not join cancellation")
			}
			select {
			case <-joined:
			default:
				t.Fatal("Run returned before held effect joined")
			}
			mu.Lock()
			defer mu.Unlock()
			if len(launched) != 1 || launched[0] != paths[1] {
				t.Fatalf("cancelled startup launched more hubs: %v", launched)
			}
		})
	}
}

func TestKnownShutdownRunDoesNotAdoptEmptyStartResults(t *testing.T) {
	hubs, paths, rt, c, console := freshStartupRunFixture(t)
	// Retained history plus confirmed absence is shutdown, never fresh.
	retained := seedDetachedThread(t, rt, paths[0])
	rt.artifacts.SetSessionPresence(retained.Address, couchcore.SessionObservation{State: couchcore.SessionAbsent})
	initial, err := c.ObserveStartupContext(t.Context(), paths)
	if err != nil || initial.Kind != couchcore.StartupFullShutdown {
		t.Fatalf("initial=%+v err=%v", initial, err)
	}
	var mu sync.Mutex
	starts, attaches := 0, 0
	completed := make(chan struct{}, 1)
	actual := console.Ops()
	console.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
		mu.Lock()
		if call.Name == "attach" {
			attaches++
		}
		if call.Name == "start-hub" {
			starts++
		}
		mu.Unlock()
		value, err := actual(call)
		if call.Name == "start-hub" && call.Args["path"] == paths[2] {
			completed <- struct{}{}
		}
		return value, err
	})
	startExplicitHubs(console, hubs, initial, c)
	done := make(chan int, 1)
	go func() { done <- console.Run(); close(done) }()
	t.Cleanup(func() {
		console.Stop()
		select {
		case <-done:
		case <-testwait.After(t):
			t.Error("shutdown Run did not join")
		}
	})
	select {
	case <-completed:
	case <-testwait.After(t):
		t.Fatal("empty startup results did not finish the serial queue")
	}
	console.Stop()
	select {
	case <-done:
	case <-testwait.After(t):
		t.Fatal("shutdown Run did not stop")
	}
	mu.Lock()
	defer mu.Unlock()
	if starts != 2 || attaches != 0 || len(rt.runner.Ops) != 0 {
		t.Fatalf("empty results: starts=%d attaches=%d process effects=%v", starts, attaches, rt.runner.Ops)
	}
}
