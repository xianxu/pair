package couchcore

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

func warmDetachedRecord(t *testing.T) ThreadRecord {
	t.Helper()
	record := actionableTestThread("couch-0000000000000001", time.Unix(100, 0).UTC())
	record.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	return record
}

// The operator's pair-couch-24: the zellij session is alive with zero clients,
// so the agent is RUNNING and reattaching restarts nothing. The native session
// id is what a COLD resume needs to relaunch the agent with `--resume <id>`;
// the warm path consumes it nowhere, so demanding it refuses a thread that
// would reattach fine.
func TestDecideResumeAcceptsADetachedThreadWithNoNativeBinding(t *testing.T) {
	for _, status := range []sessioninventory.BindingStatus{
		sessioninventory.BindingProvisional,
		sessioninventory.BindingUnbound,
		sessioninventory.BindingAmbiguous,
	} {
		t.Run(string(status), func(t *testing.T) {
			eligible, err := DecideResume(ResumeEligibilityInput{
				Thread:            warmDetachedRecord(t),
				WorkingPathExists: true,
				Detached:          true,
				Binding:           NativeBindingResolution{Status: status},
			})
			if err != nil {
				t.Fatalf("warm reattach refused: %v", err)
			}
			if eligible.RequiredSessionID != "" {
				t.Fatalf("warm reattach carries RequiredSessionID %q -- nothing on that path consumes it",
					eligible.RequiredSessionID)
			}
		})
	}
}

// The cold path is unchanged, and this is what says so: a verified park with
// the same binding states must still refuse, naming its diagnostic. Relaxing
// the warm path must not leak into the path that really does relaunch an agent.
func TestDecideResumeStillRefusesAColdResumeWithoutAnEstablishedBinding(t *testing.T) {
	for status, want := range map[sessioninventory.BindingStatus]ResumeDiagnosticCode{
		sessioninventory.BindingProvisional: ResumeBindingProvisional,
		sessioninventory.BindingUnbound:     ResumeBindingUnbound,
		sessioninventory.BindingAmbiguous:   ResumeBindingAmbiguous,
	} {
		t.Run(string(status), func(t *testing.T) {
			record := warmDetachedRecord(t)
			markActionableParked(&record, record.LastActiveAt)
			_, err := DecideResume(ResumeEligibilityInput{
				Thread:            record,
				WorkingPathExists: true,
				Binding:           NativeBindingResolution{Status: status},
			})
			if got := ResumeDiagnosticOf(err); got != want {
				t.Fatalf("cold resume with %s = %q, want %q", status, got, want)
			}
		})
	}
}

// The launch shape of a warm reattach, asserted end to end through Resume.
//
// Both omissions are load-bearing and neither is visible from the pure layer:
// the COUCH_LAUNCH_PROFILE env must be ABSENT (not empty -- Pair distinguishes
// them), because it carries ResumeRequired, which Pair honours only at a create
// boundary and a live session is an attach boundary; and `--layout2` must not
// be sent, because a running session already has its layout and asking for a
// different one sends Pair down a path that offers to DELETE it.
func TestWarmReattachSendsNoResumeProfileAndNoLayout(t *testing.T) {
	env := newTestEnv(t, "/repo")
	profile := LaunchProfile{Agent: "claude", Argv: []string{}}
	record := validThreadRecord(t)
	record.StartingPath, record.WorkingPath = "/repo", "/repo"
	record.Reservation = false
	record.LatestLaunchProfile = &profile
	created, err := env.Couch.Threads.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}

	// Detached: a live zellij session with no client, and NO usable native
	// binding -- the operator's pair-couch-24 exactly.
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
		t.Fatalf("warm argv = %q, want a bare `pair resume <tag>`", child.Argv)
	}
	for _, entry := range child.Env {
		if strings.HasPrefix(entry, launcher.CouchLaunchProfileEnv+"=") {
			t.Fatalf("warm reattach sent a resume profile: %q", entry)
		}
		if strings.HasPrefix(entry, "PAIR_USE_REPO_DEFAULT=") {
			t.Fatalf("warm reattach sent a repo-default flag: %q", entry)
		}
	}
	// The couch-ownership env still travels: it is what marks the child as
	// couch's, and it is independent of the resume authority.
	wantOwnership := []string{
		"COUCH_THREAD_SCOPE=" + created.Address.RepoScope,
		"COUCH_THREAD_TAG=" + string(created.Address.Tag),
		"COUCH_THREAD_RESUME=1",
	}
	for _, want := range wantOwnership {
		if !slices.Contains(child.Env, want) {
			t.Fatalf("warm env = %q, missing %q", child.Env, want)
		}
	}
}
