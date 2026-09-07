package couchcore

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// seedStartupDetached is the session-holding shape: a thread whose zellij
// session is alive with no client, so its layout is fixed and couch cannot
// change it without sending pair down the path that offers to DELETE it (#179).
func seedStartupDetached(t *testing.T, env *testEnv, tag ThreadTag, layout Layout) ThreadRecord {
	t.Helper()
	record := actionableTestThread(tag, time.Unix(100, 0).UTC())
	record.StartingPath, record.WorkingPath = "/repo", "/repo"
	record.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	record.Layout = layout
	created, err := env.Couch.Threads.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	env.Artifacts.SetDetachedSession(created.Address, "pair-"+string(tag))
	env.Artifacts.SetNativeBinding(created.Address, "claude", sessioninventory.BindingEstablished, "native-"+string(tag))
	return created
}

func TestStartInteractiveRefusesWhenASessionHoldsTheOtherLayout(t *testing.T) {
	env := newTestEnv(t, "/repo")
	env.Couch.Layout = Layout3
	held := seedStartupDetached(t, env, "couch-0000000000000001", Layout2)

	_, err := env.Couch.StartInteractive(context.Background(), StartArgs{Cwd: "/repo"})
	if err == nil {
		t.Fatal("couch started in layout3 alongside a layout2 session")
	}
	message := err.Error()
	for _, want := range []string{
		string(held.Address.Tag), // which thread
		"layout2",                // what it is
		"layout3",                // what was asked for
		"detached",               // why couch cannot change it
		"park",                   // the way forward
	} {
		if !strings.Contains(message, want) {
			t.Fatalf("refusal %q does not mention %q", message, want)
		}
	}
	// A refusal before any effect: the guard sits ahead of every launch path,
	// so nothing was started and nothing has to be rolled back.
	if len(env.Runner.Ops) != 0 {
		t.Fatalf("a refused startup still launched: %v", env.Runner.Ops)
	}
}

// Parked threads do not block, and this is the test that makes "park them
// first" a real remedy rather than advice: after parking, the same store that
// refused now starts.
func TestStartInteractiveProceedsWhenOnlyParkedThreadsDisagree(t *testing.T) {
	env := newTestEnv(t, "/repo")
	env.Couch.Layout = Layout3
	parked := seedStartupParked(t, env, "couch-0000000000000001", "/repo")
	env.Artifacts.SetPairSession(parked.Address, "pair-"+string(parked.Address.Tag), true)

	start, err := env.Couch.StartInteractive(context.Background(), StartArgs{Cwd: "/repo"})
	if err != nil {
		t.Fatalf("a parked thread in the other layout blocked startup: %v", err)
	}
	if start.Record.Thread != parked.Address {
		t.Fatalf("startup resumed %+v; want the parked thread %+v", start.Record.Thread, parked.Address)
	}
	// And it came back in COUCH's layout, not the one it was parked from.
	if got := env.Runner.Ops[0]; !strings.Contains(got, "--layout3") {
		t.Fatalf("cold resume argv = %q; want --layout3", got)
	}
}

func TestStartInteractiveAcceptsAMatchingSession(t *testing.T) {
	env := newTestEnv(t, "/repo")
	env.Couch.Layout = Layout3
	held := seedStartupDetached(t, env, "couch-0000000000000001", Layout3)
	env.Runner.AfterAcknowledge = func(string) error {
		env.Artifacts.SetPairSession(held.Address, "pair-"+string(held.Address.Tag), true)
		return nil
	}

	if _, err := env.Couch.StartInteractive(context.Background(), StartArgs{Cwd: "/repo"}); err != nil {
		t.Fatalf("a session in couch's OWN layout was refused: %v", err)
	}
}

// The ARCH-CONSTRAINTS budget, pinned rather than asserted in prose: the guard
// consumes the rows startup already read, so it must add no session
// enumeration of its own.
//
// The pin is the REFUSING run specifically. It stops at the guard, so every
// detached-session question it asked is one the guard is responsible for plus
// the single one ActionableThreadInventoryContext makes -- exactly 1 if the
// guard reuses those rows, 2 if it enumerates for itself. (Comparing against an
// admitted run would prove nothing: that run continues into a resume, whose own
// re-proof of detachment legitimately asks again.)
func TestGuardAddsNoSessionEnumeration(t *testing.T) {
	env := newTestEnv(t, "/repo")
	env.Couch.Layout = Layout3
	seedStartupDetached(t, env, "couch-0000000000000001", Layout2)

	if _, err := env.Couch.StartInteractive(context.Background(), StartArgs{Cwd: "/repo"}); err == nil {
		t.Fatal("the conflicting startup was admitted; this test needs the refusing path")
	}
	if got := env.Artifacts.DetachedQueries(); got != 1 {
		t.Fatalf("a refused startup asked for detached sessions %d times; want exactly 1 -- "+
			"the guard must reuse the inventory's rows, not enumerate for itself", got)
	}
}
