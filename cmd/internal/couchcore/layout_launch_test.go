package couchcore

import (
	"slices"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// A default-constructed Couch is layout2, which is what every caller that never
// heard of #198 gets. Without this, Couch.Layout's zero value is Layout("") and
// Flag() would emit a bare "--".
func TestDefaultCouchIsLayout2(t *testing.T) {
	env := newTestEnv(t, "/repo")
	if env.Couch.Layout != Layout2 {
		t.Fatalf("default Couch.Layout = %q; want Layout2", env.Couch.Layout)
	}
	if _, _, err := env.Couch.Spawn(StartArgs{Cwd: "/repo"}); err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	if got := env.Runner.Ops[0]; !strings.Contains(got, "--layout2") {
		t.Fatalf("default argv = %q; want --layout2", got)
	}
}

func TestColdStartSendsCouchLayout(t *testing.T) {
	env := newTestEnv(t, "/repo")
	env.Couch.Layout = Layout3
	if _, _, err := env.Couch.Spawn(StartArgs{Cwd: "/repo"}); err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	got := env.Runner.Ops[0]
	if !strings.Contains(got, "--layout3") || strings.Contains(got, "--layout2") {
		t.Fatalf("argv = %q; want --layout3 and no --layout2", got)
	}
}

// The witness is what the startup guard reads, so a cold start has to leave it
// behind or the next couch cannot tell what layout this session is in.
func TestColdStartRecordsTheWitness(t *testing.T) {
	env := newTestEnv(t, "/repo")
	env.Couch.Layout = Layout3
	record, _, err := env.Couch.Spawn(StartArgs{Cwd: "/repo"})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	thread, err := env.Couch.Threads.GetThread(record.Thread)
	if err != nil {
		t.Fatal(err)
	}
	if thread.Layout != Layout3 {
		t.Fatalf("witness = %q; want Layout3", thread.Layout)
	}
}

// #179 restated at the new mechanism, and the reason this is a separate test
// from the one in warmresume_test.go: that one runs a DEFAULT couch, so it
// would still pass if the warm branch started echoing c.Layout. This one runs a
// layout3 couch, where sending the flag would ask a live layout2 session to
// change layout -- the path that offers to DELETE it.
func TestWarmReattachSendsNoLayoutEvenInLayout3(t *testing.T) {
	env := newTestEnv(t, "/repo")
	env.Couch.Layout = Layout3
	record := validThreadRecord(t)
	record.StartingPath, record.WorkingPath = "/repo", "/repo"
	record.Reservation = false
	record.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	record.Layout = Layout2
	created, err := env.Couch.Threads.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	env.Artifacts.SetDetachedSession(created.Address, "pair-"+string(created.Address.Tag))
	env.Runner.AfterAcknowledge = func(string) error {
		env.Artifacts.SetPairSession(created.Address, "pair-"+string(created.Address.Tag), true)
		return nil
	}

	_, handle, err := env.Couch.Resume(created.Address)
	if err != nil {
		t.Fatalf("warm reattach refused: %v", err)
	}
	child := env.Runner.Child(handle.ID())
	if !slices.Equal(child.Argv, []string{"pair", "resume", string(created.Address.Tag)}) {
		t.Fatalf("warm argv = %q; want a bare `pair resume <tag>` even under a layout3 couch", child.Argv)
	}

	// And the witness must not be rewritten: the session is still layout2, so
	// claiming layout3 would make the store lie to the next startup guard.
	thread, err := env.Couch.Threads.GetThread(created.Address)
	if err != nil {
		t.Fatal(err)
	}
	if thread.Layout != Layout2 {
		t.Fatalf("warm reattach overwrote the witness: %q; want Layout2", thread.Layout)
	}
}

// The migration the operator described: "from layout2 to layout3, really just
// start a single right pane". A parked thread cold-resumed under a layout3
// couch BECOMES layout3, so the witness has to follow the session -- otherwise
// the store keeps claiming the old layout and the NEXT `couch --layout3`
// startup is refused citing a thread that is already layout3.
//
// It starts with no layout field at all, which is the real backfill case: every
// thread in the operator's store predates #198.
func TestColdResumeMigratesTheWitnessToCouchLayout(t *testing.T) {
	env := newTestEnv(t, "/repo")
	env.Couch.Layout = Layout3
	profile := LaunchProfile{Agent: "claude", Argv: []string{}}
	parked := createParkedThreadInCouch(t, env, profile)
	if parked.Layout != "" {
		t.Fatalf("fixture already carries a layout %q; the migration case needs none", parked.Layout)
	}
	env.Artifacts.SetNativeBinding(parked.Address, "claude", sessioninventory.BindingEstablished, "native-root-1")
	env.Runner.AfterAcknowledge = func(string) error {
		env.Artifacts.SetPairSession(parked.Address, "pair-"+string(parked.Address.Tag), true)
		return nil
	}

	_, handle, err := env.Couch.Resume(parked.Address)
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}
	child := env.Runner.Child(handle.ID())
	if !slices.Equal(child.Argv, []string{"pair", "resume", string(parked.Address.Tag), "--layout3"}) {
		t.Fatalf("cold resume argv = %q; want --layout3", child.Argv)
	}
	thread, err := env.Couch.Threads.GetThread(parked.Address)
	if err != nil {
		t.Fatal(err)
	}
	if thread.Layout != Layout3 {
		t.Fatalf("witness after cold resume = %q; want Layout3 -- the store would now lie to the guard", thread.Layout)
	}
}
