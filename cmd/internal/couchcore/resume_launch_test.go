package couchcore

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"sync"
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

func TestResumeAdmissionPreservesVerifiedParkUntilRegistration(t *testing.T) {
	store, _, parked := createVerifiedResumeRecord(t)
	beforeProfile := cloneLaunchProfile(*parked.LatestLaunchProfile)
	policy := admissionPolicy(CapacityUnbounded, 0, CapacityActionUnknown)
	resolver := NewFakePolicyResolver()
	resolver.Queue(parked.WorkingPath, policy, nil)

	admitted, err := ReconcileResumeAdmission(context.Background(), store, resolver, ResumeAdmissionInput{
		Address: parked.Address, StartedAt: time.Unix(710, 0).UTC(),
		Owner: SupervisorOwner{PID: 7, Identity: "supervisor"}, Nonce: "start-resume", Profile: beforeProfile,
	})
	if err != nil {
		t.Fatalf("ReconcileResumeAdmission: %v", err)
	}
	if admitted.Address != parked.Address || admitted.VerifiedPark == nil || len(admitted.Incarnations) != 1 ||
		admitted.Incarnations[0].State != IncarnationCreating || admitted.Incarnations[0].Policy == nil ||
		admitted.Incarnations[0].Start == nil || admitted.Incarnations[0].Start.Nonce != "start-resume" ||
		admitted.Incarnations[0].Start.OwnerPID != 7 || admitted.Incarnations[0].Start.OwnerIdentity != "supervisor" ||
		admitted.Incarnations[0].Start.LaunchProfile == nil ||
		!reflect.DeepEqual(*admitted.Incarnations[0].Start.LaunchProfile, beforeProfile) ||
		!reflect.DeepEqual(*admitted.LatestLaunchProfile, beforeProfile) {
		t.Fatalf("admitted resume = %+v", admitted)
	}
}

func TestResumeAdmissionRefusalLeavesParkedRecordUntouched(t *testing.T) {
	store, ns, parked := createVerifiedResumeRecord(t)
	policy := admissionPolicy(CapacityBounded, 1, CapacityReject)
	incumbent := validThreadRecord(t)
	incumbent.Address.Tag = "couch-fedcba9876543210"
	incumbent.StartingPath, incumbent.WorkingPath = ns.Dir(), ns.Dir()
	incumbent.Reservation = false
	incumbent.Incarnations = []ThreadIncarnation{{State: IncarnationLive, PID: 99, Identity: "incumbent", Policy: &policy}}
	if _, err := store.CreateThread(incumbent); err != nil {
		t.Fatal(err)
	}
	resolver := NewFakePolicyResolver()
	resolver.Queue(parked.WorkingPath, policy, nil)

	_, err := ReconcileResumeAdmission(context.Background(), store, resolver, ResumeAdmissionInput{
		Address: parked.Address, StartedAt: time.Unix(720, 0).UTC(),
		Owner: SupervisorOwner{PID: 8, Identity: "supervisor"}, Nonce: "start-refused",
		Profile: cloneLaunchProfile(*parked.LatestLaunchProfile),
	})
	var full *CapacityExceededError
	if !errors.As(err, &full) {
		t.Fatalf("error = %T %v", err, err)
	}
	kept, err := store.GetThread(parked.Address)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(kept, parked) {
		t.Fatalf("refused parked record changed:\n got %+v\nwant %+v", kept, parked)
	}
}

func createParkedThreadInCouch(t *testing.T, env *testEnv, profile LaunchProfile) ThreadRecord {
	t.Helper()
	policy, err := env.Couch.PolicyResolver.ResolvePolicy(context.Background(), "/repo/sub")
	if err != nil {
		t.Fatal(err)
	}
	record := validThreadRecord(t)
	record.StartingPath, record.WorkingPath = "/repo", "/repo/sub"
	record.Reservation = false
	record.Incarnations = []ThreadIncarnation{{
		PID: 42, Identity: "pair-helper", State: IncarnationLive,
		Policy: &policy, LaunchProfile: &profile,
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

func TestResumeAdmissionAndRollbackMatrix(t *testing.T) {
	t.Run("acknowledgement definitely undelivered with proven absence", func(t *testing.T) {
		env := newTestEnv(t, "/repo")
		parked := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
		env.Artifacts.SetNativeBinding(parked.Address, "claude", sessioninventory.BindingEstablished, "native-root-1")
		env.Artifacts.SetPairSession(parked.Address, "pair-"+string(parked.Address.Tag), false)
		env.Runner.BeforeAcknowledge = func(string) error { return errors.New("ack not delivered") }

		_, _, err := env.Couch.Resume(parked.Address)
		if err == nil || !strings.Contains(err.Error(), "ack not delivered") {
			t.Fatalf("Resume error = %v", err)
		}
		assertVerifiedParkRestored(t, env.Couch.Threads, parked)
	})

	t.Run("missing registration without absence proof", func(t *testing.T) {
		env := newTestEnv(t, "/repo")
		env.Couch.resumeRegistrationTimeout = 5 * time.Millisecond
		parked := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "agy", Argv: []string{}})
		env.Artifacts.SetNativeBinding(parked.Address, "agy", sessioninventory.BindingEstablished, "native-root-1")

		_, _, err := env.Couch.Resume(parked.Address)
		if err == nil || !strings.Contains(err.Error(), "deadline exceeded") {
			t.Fatalf("Resume error = %v", err)
		}
		assertResumeUnknown(t, env.Couch.Threads, parked.Address, "")
	})

	t.Run("live promotion CAS failure", func(t *testing.T) {
		env := newTestEnv(t, "/repo")
		parked := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "muse", Argv: []string{}})
		env.Artifacts.SetNativeBinding(parked.Address, "muse", sessioninventory.BindingEstablished, "native-root-1")
		env.Artifacts.SetPairSession(parked.Address, "pair-"+string(parked.Address.Tag), true)
		var once sync.Once
		env.Artifacts.BeforePairSession = func(address ThreadAddress) error {
			var hookErr error
			once.Do(func() {
				current, err := env.Couch.Threads.GetThread(address)
				if err != nil {
					hookErr = err
					return
				}
				_, hookErr = env.Couch.Threads.UpdateExistingThread(address, current.Revision, func(next *ThreadRecord) error {
					next.Description = "concurrent description"
					return nil
				})
			})
			return hookErr
		}

		_, _, err := env.Couch.Resume(parked.Address)
		if err == nil || !strings.Contains(err.Error(), "promote registered thread") {
			t.Fatalf("Resume error = %v", err)
		}
		assertResumeUnknown(t, env.Couch.Threads, parked.Address, "concurrent description")
	})
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
