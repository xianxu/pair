package couchcore

import (
	"slices"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// The process default must reach both child argv and its persisted witness.
func TestDefaultCouchIsLayout3(t *testing.T) {
	env := newTestEnv(t, "/repo")
	if env.Couch.Layout != Layout3 {
		t.Fatalf("default Couch.Layout = %q; want Layout3", env.Couch.Layout)
	}
	record, handle, err := env.Couch.Spawn(StartArgs{Cwd: "/repo"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"pair", "resume", string(record.Thread.Tag), "--layout3"}
	if got := env.Runner.Child(handle.ID()).Argv; !slices.Equal(got, want) {
		t.Fatalf("argv = %q; want %q", got, want)
	}
	thread, err := env.Couch.Threads.GetThread(record.Thread)
	if err != nil || thread.Layout != Layout3 {
		t.Fatalf("witness = %+v, %v; want Layout3", thread, err)
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

// A warm reattach must not echo the process layout: asking a live layout2
// session to change to layout3 reaches Pair's destructive conflict path.
// This also checks that reattachment preserves the existing layout witness.
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
