package couchcore

import (
	"context"
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

// A missing selected hub remains idle regardless of other retained layouts.
func TestStartupMissingHubLeavesOtherRetainedLayoutsUntouched(t *testing.T) {
	for _, parked := range []bool{false, true} {
		env := newTestEnv(t, "/repo")
		address := detachedThreadAt(t, env, "/other", "layout")
		bumpLayout(t, env, address, "")
		if parked {
			current, err := env.Couch.Threads.GetThread(address)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := env.Couch.Threads.updateExistingThread(address, current.Revision, func(next *ThreadRecord) error {
				markActionableParked(next, time.Unix(100, 0).UTC())
				return nil
			}); err != nil {
				t.Fatal(err)
			}
		}
		start, err := env.Couch.StartInteractive(context.Background(), StartArgs{Cwd: "/repo"})
		if err != nil || start.Handle != nil || len(env.Runner.Ops) != 0 {
			t.Fatalf("startup: %+v %v %v", start, err, env.Runner.Ops)
		}
	}
}
