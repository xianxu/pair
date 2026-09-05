package couchcore

import (
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

func verifiedResumeThread(t *testing.T) ThreadRecord {
	t.Helper()
	store, _, thread := createControllerThread(t)
	identity := ParkIdentity{
		Nonce: "park-resume-eligible", Address: thread.Address,
		PID: 42, ProcessIdentity: "pair-helper",
	}
	begun, err := store.BeginPark(thread.Address, thread.Revision, identity)
	if err != nil {
		t.Fatal(err)
	}
	parked, err := store.FinalizePark(thread.Address, begun.Revision, identity, 1, time.Unix(600, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	return parked
}

func TestDecideResumeEligibilityMatrix(t *testing.T) {
	base := verifiedResumeThread(t)
	established := NativeBindingResolution{Status: sessioninventory.BindingEstablished, NativeID: "native-root-1"}

	t.Run("verified parked", func(t *testing.T) {
		got, err := DecideResume(ResumeEligibilityInput{Thread: base, WorkingPathExists: true, Binding: established})
		if err != nil {
			t.Fatalf("DecideResume: %v", err)
		}
		if got.Address != base.Address || got.WorkingPath != base.WorkingPath || got.RequiredSessionID != "native-root-1" ||
			got.Profile.Agent != base.LatestLaunchProfile.Agent || len(got.Profile.Argv) != len(base.LatestLaunchProfile.Argv) {
			t.Fatalf("decision = %+v", got)
		}
		got.Profile.Argv[0] = "mutated"
		if base.LatestLaunchProfile.Argv[0] == "mutated" {
			t.Fatal("decision aliases persisted profile")
		}
	})

	tests := []struct {
		name   string
		code   ResumeDiagnosticCode
		mutate func(*ResumeEligibilityInput)
	}{
		{name: "live", code: ResumeLive, mutate: func(in *ResumeEligibilityInput) {
			in.Thread.Incarnations = []ThreadIncarnation{{State: IncarnationLive}}
		}},
		{name: "creating", code: ResumeCreating, mutate: func(in *ResumeEligibilityInput) {
			in.Thread.Incarnations = []ThreadIncarnation{{State: IncarnationCreating}}
		}},
		{name: "unknown", code: ResumeUnknown, mutate: func(in *ResumeEligibilityInput) {
			in.Thread.Incarnations = []ThreadIncarnation{{State: IncarnationUnknown}}
		}},
		{name: "parking", code: ResumeParking, mutate: func(in *ResumeEligibilityInput) {
			in.Thread.VerifiedPark = nil
			in.Thread.Park = &ParkTransaction{Phase: ParkAwaitingCompletion}
			in.Thread.Incarnations = []ThreadIncarnation{{State: IncarnationLive}}
		}},
		{name: "tombstoned", code: ResumeTombstoned, mutate: func(in *ResumeEligibilityInput) {
			in.Thread.VerifiedPark = nil
			in.Thread.ParkHistory[len(in.Thread.ParkHistory)-1].Tombstoned = true
			in.Thread.ParkHistory[len(in.Thread.ParkHistory)-1].SuccessfulAttempt = 0
		}},
		{name: "legacy unverified", code: ResumeLegacyUnverified, mutate: func(in *ResumeEligibilityInput) {
			in.Thread.VerifiedPark = nil
			in.Thread.ParkHistory = nil
		}},
		{name: "missing path", code: ResumePathMissing, mutate: func(in *ResumeEligibilityInput) {
			in.WorkingPathExists = false
		}},
		{name: "empty path", code: ResumePathMissing, mutate: func(in *ResumeEligibilityInput) {
			in.Thread.WorkingPath = ""
		}},
		{name: "missing profile", code: ResumeProfileMissing, mutate: func(in *ResumeEligibilityInput) {
			in.Thread.LatestLaunchProfile = nil
		}},
		{name: "profile missing agent", code: ResumeProfileInvalid, mutate: func(in *ResumeEligibilityInput) {
			in.Thread.LatestLaunchProfile.Agent = ""
		}},
		{name: "profile null argv", code: ResumeProfileInvalid, mutate: func(in *ResumeEligibilityInput) {
			in.Thread.LatestLaunchProfile.Argv = nil
		}},
		{name: "unsupported agent", code: ResumeAgentUnsupported, mutate: func(in *ResumeEligibilityInput) {
			in.Thread.LatestLaunchProfile.Agent = "unknown-agent"
		}},
		{name: "provisional binding", code: ResumeBindingProvisional, mutate: func(in *ResumeEligibilityInput) {
			in.Binding = NativeBindingResolution{Status: sessioninventory.BindingProvisional}
		}},
		{name: "ambiguous binding", code: ResumeBindingAmbiguous, mutate: func(in *ResumeEligibilityInput) {
			in.Binding = NativeBindingResolution{Status: sessioninventory.BindingAmbiguous}
		}},
		{name: "unbound binding", code: ResumeBindingUnbound, mutate: func(in *ResumeEligibilityInput) {
			in.Binding = NativeBindingResolution{Status: sessioninventory.BindingUnbound}
		}},
		{name: "established without root", code: ResumeBindingRootMissing, mutate: func(in *ResumeEligibilityInput) {
			in.Binding = NativeBindingResolution{Status: sessioninventory.BindingEstablished}
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := ResumeEligibilityInput{Thread: cloneThreadRecord(base), WorkingPathExists: true, Binding: established}
			test.mutate(&input)
			if _, err := DecideResume(input); ResumeDiagnosticOf(err) != test.code {
				t.Fatalf("DecideResume error = %v, code=%q; want %q", err, ResumeDiagnosticOf(err), test.code)
			}
		})
	}
}

// A detached thread has no verified park -- nothing was torn down. Its
// authority is the surviving zellij session.
//
// The tombstone cross is the case that would otherwise ship a permanently
// unreattachable class: the ParkHistory scan refuses on ANY tombstoned entry
// with no break, and AbandonPark appends tombstones permanently, so a thread
// once abandoned mid-park and later detached must still resume.
func TestDecideResumeAcceptsDetachedWithoutVerifiedPark(t *testing.T) {
	address := ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-0001020304050607"}
	profile := &LaunchProfile{Agent: "claude", Argv: []string{}}
	binding := NativeBindingResolution{Status: sessioninventory.BindingEstablished, NativeID: "native-1"}

	base := func() ThreadRecord {
		return ThreadRecord{
			SchemaVersion: ThreadSchemaVersion, Address: address,
			StartingPath: "/repo", WorkingPath: "/repo",
			CreatedAt: time.Unix(1, 0).UTC(), Revision: 1,
			LatestLaunchProfile: profile,
		}
	}

	tests := []struct {
		name     string
		mutate   func(*ThreadRecord)
		detached bool
		wantCode ResumeDiagnosticCode
	}{
		{name: "detached proof admits a record with no verified park", detached: true},
		{
			name:     "without the proof the same record refuses",
			wantCode: ResumeLegacyUnverified,
		},
		{
			name: "a tombstoned history does not block a detached resume",
			mutate: func(r *ThreadRecord) {
				r.ParkHistory = []ParkTransaction{{Tombstoned: true, Closed: true}}
			},
			detached: true,
		},
		{
			name: "a tombstoned history still blocks a NON-detached resume",
			mutate: func(r *ThreadRecord) {
				r.ParkHistory = []ParkTransaction{{Tombstoned: true, Closed: true}}
			},
			wantCode: ResumeTombstoned,
		},
		{
			name: "an occupied incarnation refuses even with the detached proof",
			mutate: func(r *ThreadRecord) {
				r.Incarnations = []ThreadIncarnation{{State: IncarnationLive, PID: 1, Identity: "x", StartedAt: time.Unix(2, 0).UTC()}}
			},
			detached: true,
			wantCode: ResumeLive,
		},
		{
			name:     "a detached record still needs a saved launch profile",
			mutate:   func(r *ThreadRecord) { r.LatestLaunchProfile = nil },
			detached: true,
			wantCode: ResumeProfileMissing,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			record := base()
			if test.mutate != nil {
				test.mutate(&record)
			}
			_, err := DecideResume(ResumeEligibilityInput{
				Thread: record, WorkingPathExists: true, Binding: binding, Detached: test.detached,
			})
			if test.wantCode == "" {
				if err != nil {
					t.Fatalf("DecideResume() = %v, want acceptance", err)
				}
				return
			}
			if got := ResumeDiagnosticOf(err); got != test.wantCode {
				t.Fatalf("DecideResume() diagnostic = %q, want %q (err %v)", got, test.wantCode, err)
			}
		})
	}
}

// This test used to assert that a detached resume STILL requires an established
// native binding. #179 reverses that: the operator could not reattach a session
// whose agent was demonstrably running, because couch demanded the proof a COLD
// resume needs -- the transcript id Pair relaunches with -- on a path that
// relaunches nothing.
//
// It is inverted rather than deleted, because the reversal is worth recording
// where the superseded claim lived. What replaces the binding as the warm
// path's authority is the session itself: input.Detached, an unambiguous name
// binding to this exact address, live, with zero clients. The cold path is
// unchanged and TestDecideResumeStillRefusesAColdResumeWithoutAnEstablishedBinding
// is what says so.
func TestDetachedResumeDoesNotRequireAnEstablishedBinding(t *testing.T) {
	address := ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-0001020304050607"}
	record := ThreadRecord{
		SchemaVersion: ThreadSchemaVersion, Address: address,
		StartingPath: "/repo", WorkingPath: "/repo",
		CreatedAt: time.Unix(1, 0).UTC(), Revision: 1,
		LatestLaunchProfile: &LaunchProfile{Agent: "claude", Argv: []string{}},
	}
	for _, status := range []sessioninventory.BindingStatus{
		sessioninventory.BindingProvisional,
		sessioninventory.BindingAmbiguous,
		sessioninventory.BindingUnbound,
	} {
		eligible, err := DecideResume(ResumeEligibilityInput{
			Thread: record, WorkingPathExists: true, Detached: true,
			Binding: NativeBindingResolution{Status: status},
		})
		if err != nil {
			t.Fatalf("binding %q refused a warm reattach: %v", status, err)
		}
		if eligible.RequiredSessionID != "" {
			t.Fatalf("warm reattach carried RequiredSessionID %q from a %q binding",
				eligible.RequiredSessionID, status)
		}
	}
}
