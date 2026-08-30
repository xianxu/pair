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
