package couchcore

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// startupFixture builds a cwd thread plus `others` detached threads at other
// paths, all in couch's layout, and returns the env.
func startupFixture(t *testing.T, others int, otherLayout Layout) (*testEnv, ThreadAddress) {
	t.Helper()
	env := newTestEnv(t, "/repo")
	env.Couch.Layout = NormalizeLayout("layout2")
	env.Couch.resumeRegistrationTimeout = 2 * time.Second

	cwd := detachedThreadAt(t, env, "/repo", "cwd")
	for i := 0; i < others; i++ {
		address := detachedThreadAt(t, env, fmt.Sprintf("/other-%02d", i), fmt.Sprintf("o%02d", i))
		if otherLayout != "" {
			bumpLayout(t, env, address, otherLayout)
		}
	}
	return env, cwd
}

func detachedThreadAt(t *testing.T, env *testEnv, path, suffix string) ThreadAddress {
	t.Helper()
	return detachedThreadIn(t, env, path, path, suffix)
}

// detachedThreadIn is detachedThreadAt for a record whose working path is
// somewhere inside the repository rooted at root, not at the root itself.
func detachedThreadIn(t *testing.T, env *testEnv, root, path, suffix string) ThreadAddress {
	t.Helper()
	// The scope key is the REAL one for this repository. A hand-written key
	// would never match what StartInteractive resolves, and the narrowing
	// predicate compares scope keys -- so the fixture would silently exercise
	// the "nothing in this repository" branch and pass for the wrong reason.
	scope, err := launcher.ResolveRepoScope(root)
	if err != nil {
		t.Fatal(err)
	}
	profile := LaunchProfile{Agent: "claude", Argv: []string{}}
	record := validThreadRecord(t)
	record.Address.RepoScope = scope.Key
	record.Address.Tag = ThreadTag(fmt.Sprintf("couch-%016x", threadCounter(suffix)))
	record.StartingPath, record.WorkingPath = path, path
	record.Reservation = false
	record.LatestLaunchProfile = &profile
	record.Layout = NormalizeLayout("layout2")
	created, createErr := env.Couch.Threads.CreateThread(record)
	if createErr != nil {
		t.Fatal(createErr)
	}
	name := "pair-" + string(created.Address.Tag)
	env.Artifacts.SetDetachedSession(created.Address, name)
	env.Artifacts.SetPairSession(created.Address, name, true)
	// A binding is seeded for tests that later park this thread; warm startup
	// must not consult it.
	env.Artifacts.SetNativeBinding(created.Address, "claude", sessioninventory.BindingEstablished, "native-"+suffix)
	return created.Address
}

func bumpLayout(t *testing.T, env *testEnv, address ThreadAddress, layout Layout) {
	t.Helper()
	current, err := env.Couch.Threads.GetThread(address)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := env.Couch.Threads.updateExistingThread(address, current.Revision, func(next *ThreadRecord) error {
		next.Layout = layout
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

// threadCounter derives a stable tag suffix from a label, so a fixture's tags
// are deterministic within one test.
func threadCounter(suffix string) uint64 {
	var h uint64 = 1469598103934665603
	for _, b := range []byte(suffix) {
		h ^= uint64(b)
		h *= 1099511628211
	}
	return h
}

func TestStartupAttachesOnlySelectedHub(t *testing.T) {
	for _, n := range []int{0, 20, 40} {
		env, hub := startupFixture(t, n, Layout3)
		selectedQueries := 0
		env.Artifacts.ListedSessionHook = func(address ThreadAddress) error {
			if address != hub {
				t.Fatalf("startup asked about unrelated session %+v", address)
			}
			selectedQueries++
			return nil
		}
		env.Artifacts.SetSessionPresence(hub, SessionObservation{State: SessionPresent, Name: "pair-" + string(hub.Tag)})
		if _, err := env.Couch.ObserveStartupContext(context.Background(), []string{"/repo"}); err != nil {
			t.Fatal(err)
		}
		before, err := env.Couch.Threads.PreviewSnapshot()
		if err != nil {
			t.Fatal(err)
		}
		result, err := env.Couch.StartInteractive(context.Background(), StartArgs{Cwd: "/repo"})
		if err != nil || result.Handle == nil || result.Record.Thread != hub || len(env.Runner.Ops) != 2 || len(env.Artifacts.Calls()) != 0 || env.Artifacts.DetachedQueries() != 0 {
			t.Fatalf("startup: %+v %v %v calls=%v", result, err, env.Runner.Ops, env.Artifacts.Calls())
		}
		if selectedQueries != 4 || env.Artifacts.SessionPresenceQueries() != 1 {
			t.Fatalf("selected proof calls=%d bulk presence calls=%d", selectedQueries, env.Artifacts.SessionPresenceQueries())
		}
		after, err := env.Couch.Threads.PreviewSnapshot()
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(before, after) {
			t.Fatal("hub attachment changed retained slots")
		}
	}
}

func TestStartupNumberedHubAttachesWithoutProvisionOrOtherRecordMutation(t *testing.T) {
	env, f := managedStartFixture(t)
	args := StartArgs{Cwd: f.Primary, Action: StartCreate}
	prepared, err := env.Couch.PrepareStart(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	actor, _, err := env.Couch.SpawnPrepared(context.Background(), args, prepared.Resolution.Fingerprint)
	if err != nil {
		t.Fatal(err)
	}
	current, currentErr := env.Couch.Threads.PreviewThreadContext(context.Background(), actor.Thread)
	if currentErr != nil {
		t.Fatal(currentErr)
	}
	env.Artifacts.SetPairSession(actor.Thread, recordSessionName(current), true)
	// Keep the retained client alive so this fixture exercises attachment only,
	// rather than the existing selected dead-client helper repair.
	env.Proc.Set(actor.PID, actor.Identity)
	before, err := env.Couch.Threads.PreviewSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	opsBefore, provisionBefore := len(env.Runner.Ops), f.WeaveCalls
	nativeBefore := env.Artifacts.BindingResolutions()
	result, err := env.Couch.StartInteractive(context.Background(), StartArgs{Cwd: f.host(1)})
	if err != nil || result.Handle == nil || result.Record.Thread != actor.Thread || result.Record.Shape != StartWarmReattach {
		t.Fatalf("numbered attach: %+v err=%v", result, err)
	}
	if f.WeaveCalls != provisionBefore || len(env.Runner.Ops) != opsBefore+2 || env.Artifacts.BindingResolutions() != nativeBefore {
		t.Fatal("numbered attachment provisioned or cold-resumed")
	}
	after, err := env.Couch.Threads.PreviewSnapshot()
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("numbered attachment changed retained records: %v", err)
	}
}
