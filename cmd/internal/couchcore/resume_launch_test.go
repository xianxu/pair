package couchcore

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

func createVerifiedResumeRecord(t *testing.T) (*ThreadStore, CouchNamespace, ThreadRecord) {
	t.Helper()
	store, ns, thread := createControllerThread(t)
	identity := ParkIdentity{
		Nonce: "park-resume-launch", Address: thread.Address,
		PID: 42, ProcessIdentity: "pair-helper",
	}
	begun, err := store.BeginPark(thread.Address, thread.Revision, identity)
	if err != nil {
		t.Fatal(err)
	}
	parked, err := store.FinalizePark(thread.Address, begun.Revision, identity, 1, time.Unix(700, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	return store, ns, parked
}

func createParkedThreadInCouch(t *testing.T, env *testEnv, profile LaunchProfile) ThreadRecord {
	t.Helper()
	record := validThreadRecord(t)
	record.StartingPath, record.WorkingPath = "/repo", "/repo/sub"
	env.Git.replies[GitCall{Dir: "/repo/sub", Args: "rev-parse --git-common-dir"}] = ".git"
	record.Reservation = false
	record.Incarnations = []ThreadIncarnation{{
		PID: 42, Identity: "pair-helper", State: IncarnationLive,
		RepoIdentity: "/repo/.git", LaunchProfile: &profile,
	}}
	record.LatestLaunchProfile = &profile
	created, err := env.Couch.Threads.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	identity := ParkIdentity{Nonce: "park-resume-profile", Address: created.Address, PID: 42, ProcessIdentity: "pair-helper"}
	begun, err := env.Couch.Threads.BeginPark(created.Address, created.Revision, identity)
	if err != nil {
		t.Fatal(err)
	}
	parked, err := env.Couch.Threads.FinalizePark(created.Address, begun.Revision, identity, 1, env.Now)
	if err != nil {
		t.Fatal(err)
	}
	return parked
}

func TestResumeLaunchExactProfileMatrix(t *testing.T) {
	for _, agent := range []string{"claude", "codex", "agy", "muse"} {
		for _, argv := range [][]string{
			{},
			{"resume", "old-native", "--literal", agent},
		} {
			name := agent + "/"
			if len(argv) == 0 {
				name += "empty"
			} else {
				name += "resume-looking"
			}
			t.Run(name, func(t *testing.T) {
				env := newTestEnv(t, "/repo")
				profile := LaunchProfile{Agent: agent, Argv: append([]string(nil), argv...)}
				if profile.Argv == nil {
					profile.Argv = []string{}
				}
				parked := createParkedThreadInCouch(t, env, profile)
				env.Artifacts.SetNativeBinding(parked.Address, agent, sessioninventory.BindingEstablished, "native-root-1")
				env.Runner.AfterAcknowledge = func(string) error {
					env.Artifacts.SetPairSession(parked.Address, "pair-"+string(parked.Address.Tag), true)
					return nil
				}
				env.Couch.RepoAgentDefault = func(_, _ string) (LaunchProfile, bool, error) {
					t.Fatal("resume consulted repository defaults")
					return LaunchProfile{}, false, nil
				}

				record, handle, err := env.Couch.Resume(parked.Address)
				if err != nil {
					t.Fatalf("Resume: %v", err)
				}
				child := env.Runner.Child(handle.ID())
				if record.Thread != parked.Address || child.Dir != "/repo/sub" ||
					!slices.Equal(child.Argv, []string{"pair", "resume", string(parked.Address.Tag), "--layout2"}) {
					t.Fatalf("resume launch = record %+v child %+v", record, child)
				}
				raw, err := launcher.BuildCouchResumeLaunchProfile(string(parked.Address.Tag), agent, profile.Argv, "native-root-1")
				if err != nil {
					t.Fatal(err)
				}
				wantEnv := []string{
					"COUCH_TREE=/repo", "COUCH_STORE_DIR=" + env.Dir,
					"COUCH_THREAD_SCOPE=" + parked.Address.RepoScope,
					"COUCH_THREAD_TAG=" + string(parked.Address.Tag),
					"COUCH_THREAD_RESUME=1",
					launcher.CouchLaunchProfileEnv + "=" + strings.TrimSpace(raw),
					"PAIR_USE_REPO_DEFAULT=",
				}
				if !slices.Equal(child.Env, wantEnv) {
					t.Fatalf("resume env = %q, want %q", child.Env, wantEnv)
				}
				if got := env.Artifacts.Calls(); len(got) != 0 {
					t.Fatalf("resume allocated a new tag: %v", got)
				}
			})
		}
	}
}

func TestResumeForkFailureRestoresVerifiedPark(t *testing.T) {
	env := newTestEnv(t, "/repo")
	parked := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
	env.Artifacts.SetNativeBinding(parked.Address, "claude", sessioninventory.BindingEstablished, "native-root-1")
	env.Runner.FailNextStart(errors.New("fork failed"))

	_, _, err := env.Couch.Resume(parked.Address)
	if err == nil || !strings.Contains(err.Error(), "fork failed") {
		t.Fatalf("Resume error = %v", err)
	}
	assertVerifiedParkRestored(t, env.Couch.Threads, parked)
}

func TestResumeAmbiguousAckKeepsUnknownOccupied(t *testing.T) {
	env := newTestEnv(t, "/repo")
	parked := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "codex", Argv: []string{"--saved"}})
	env.Artifacts.SetNativeBinding(parked.Address, "codex", sessioninventory.BindingEstablished, "native-root-1")
	env.Runner.AfterAcknowledge = func(string) error {
		env.Artifacts.SetPairSession(parked.Address, "pair-"+string(parked.Address.Tag), true)
		return errors.New("ack transport closed")
	}

	_, _, err := env.Couch.Resume(parked.Address)
	if err == nil || !strings.Contains(err.Error(), "ack transport closed") {
		t.Fatalf("Resume error = %v", err)
	}
	kept, getErr := env.Couch.Threads.GetThread(parked.Address)
	if getErr != nil {
		t.Fatal(getErr)
	}
	if kept.VerifiedPark == nil || len(kept.Incarnations) != 1 || kept.Incarnations[0].State != IncarnationUnknown {
		t.Fatalf("ambiguous resume state = %+v", kept)
	}
}

func assertResumeUnknown(t *testing.T, store *ThreadStore, address ThreadAddress, description string) {
	t.Helper()
	got, err := store.GetThread(address)
	if err != nil {
		t.Fatal(err)
	}
	if got.VerifiedPark == nil || len(got.Incarnations) != 1 || got.Incarnations[0].State != IncarnationUnknown || got.Description != description {
		t.Fatalf("resume unknown state = %+v", got)
	}
}

func assertVerifiedParkRestored(t *testing.T, store *ThreadStore, parked ThreadRecord) {
	t.Helper()
	got, err := store.GetThread(parked.Address)
	if err != nil {
		t.Fatal(err)
	}
	if got.VerifiedPark == nil || !reflect.DeepEqual(got.VerifiedPark, parked.VerifiedPark) ||
		len(got.Incarnations) != 0 || !reflect.DeepEqual(got.LatestLaunchProfile, parked.LatestLaunchProfile) {
		t.Fatalf("resume rollback = %+v, want verified park %+v", got, parked)
	}
}

// The milestone's headline capability, pinned at the SEAM rather than at the
// pure functions beneath it.
//
// DecideResume and ProjectActionableThreads are tested with `Detached` hand-fed,
// which proves the rules and nothing about the code that DERIVES the value.
// Mutating ResumeContext's observation to `if false && …` left the whole suite
// green -- exactly the trap workshop/lessons.md gained an entry for this round.
// Here the ONLY difference between admissible and refused is
// SetDetachedSession, so that mutation reddens.
func TestResumeContextDerivesTheDetachedProofItself(t *testing.T) {
	for _, test := range []struct {
		name     string
		detached bool
	}{
		{name: "no detached session refuses for want of a verified park"},
		{name: "a detached session makes the same record admissible", detached: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			env := newTestEnv(t, "/repo")
			profile := LaunchProfile{Agent: "claude", Argv: []string{}}
			// A record with no verified park and no incarnation: the shape an
			// alt+d detach leaves behind.
			record := validThreadRecord(t)
			record.StartingPath, record.WorkingPath = "/repo", "/repo/sub"
			env.Git.replies[GitCall{Dir: "/repo/sub", Args: "rev-parse --git-common-dir"}] = ".git"
			record.Reservation = false
			record.LatestLaunchProfile = &profile
			created, err := env.Couch.Threads.CreateThread(record)
			if err != nil {
				t.Fatal(err)
			}
			env.Artifacts.SetNativeBinding(created.Address, "claude", sessioninventory.BindingEstablished, "native-root-1")
			env.Artifacts.SetPairSession(created.Address, "pair-"+string(created.Address.Tag), true)
			if test.detached {
				env.Artifacts.SetDetachedSession(created.Address, "pair-"+string(created.Address.Tag))
			}

			_, _, err = env.Couch.ResumeContext(context.Background(), created.Address)

			got := ResumeDiagnosticOf(err)
			if test.detached {
				if got == ResumeLegacyUnverified {
					t.Fatalf("a proved-detached thread was refused for want of a verified park: %v", err)
				}
				return
			}
			if got != ResumeLegacyUnverified {
				t.Fatalf("diagnostic = %q (err %v), want %q", got, err, ResumeLegacyUnverified)
			}
		})
	}
}
