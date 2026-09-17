package couchcore

import (
	"os"
	"regexp"
	"strings"
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
		// RESTATED for #256. These four asserted that the RECORD's own
		// bookkeeping could veto a resume: an incarnation in any occupied state,
		// or an open park. Both name things that die with couch -- the launcher
		// process and a transaction whose owner is that same process -- so the
		// veto fired hardest on exactly the threads a crash left recoverable.
		//
		// Resume now rests on the same evidence the classification does, which is
		// what keeps "the switcher offers it" and "the guard permits it" from
		// disagreeing. A record carrying a stale incarnation and a verified park
		// is resumable, because the park is the authority and the incarnation is
		// not evidence of anything.
		{name: "stale live incarnation does not veto a verified park", code: "", mutate: func(in *ResumeEligibilityInput) {
			in.Thread.Incarnations = []ThreadIncarnation{{State: IncarnationLive}}
		}},
		{name: "creating incarnation does not veto a verified park", code: "", mutate: func(in *ResumeEligibilityInput) {
			in.Thread.Incarnations = []ThreadIncarnation{{State: IncarnationCreating}}
		}},
		{name: "unknown incarnation does not veto a verified park", code: "", mutate: func(in *ResumeEligibilityInput) {
			in.Thread.Incarnations = []ThreadIncarnation{{State: IncarnationUnknown}}
		}},
		// RESTATED for #256 M2. These three asserted that a missing park RECEIPT
		// vetoes a cold resume. It does not and cannot: the receipt carries a
		// ParkIdentity and no conversation id, so it can say a park happened and
		// never that anything survived it. The ledger is the authority, and with
		// an established binding all three of these records resume.
		{name: "an open park no longer vetoes, and neither does a missing receipt", code: "", mutate: func(in *ResumeEligibilityInput) {
			in.Thread.VerifiedPark = nil
			in.Thread.Park = &ParkTransaction{Phase: ParkAwaitingCompletion}
			in.Thread.Incarnations = []ThreadIncarnation{{State: IncarnationLive}}
		}},
		{name: "no receipt and no conversation refuses, naming the binding", code: ResumeBindingUnbound, mutate: func(in *ResumeEligibilityInput) {
			in.Thread.VerifiedPark = nil
			in.Thread.ParkHistory = nil
			in.Binding = NativeBindingResolution{Status: sessioninventory.BindingUnbound}
		}},
		// A tombstone is no longer a VETO -- archive abandons orphaned parks as
		// a matter of course since M2, so vetoing on one would make "couch
		// crashed mid-park once" a permanent cold-resume ban. It survives as the
		// better EXPLANATION when there is nothing to resume into.
		{name: "a tombstone does not veto a resolvable conversation", code: "", mutate: func(in *ResumeEligibilityInput) {
			in.Thread.VerifiedPark = nil
			in.Thread.ParkHistory[len(in.Thread.ParkHistory)-1].Tombstoned = true
			in.Thread.ParkHistory[len(in.Thread.ParkHistory)-1].SuccessfulAttempt = 0
		}},
		{name: "tombstoned", code: ResumeTombstoned, mutate: func(in *ResumeEligibilityInput) {
			in.Thread.VerifiedPark = nil
			in.Thread.ParkHistory[len(in.Thread.ParkHistory)-1].Tombstoned = true
			in.Thread.ParkHistory[len(in.Thread.ParkHistory)-1].SuccessfulAttempt = 0
			in.Binding = NativeBindingResolution{Status: sessioninventory.BindingUnbound}
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
			// RESTATED for #256 M2: `binding` here is established, so there IS
			// a conversation to resume into and the cold path takes it. The
			// receipt never carried that answer.
			name: "without the detached proof the ledger answers instead",
		},
		{
			name: "a tombstoned history does not block a detached resume",
			mutate: func(r *ThreadRecord) {
				r.ParkHistory = []ParkTransaction{{Tombstoned: true, Closed: true}}
			},
			detached: true,
		},
		{
			// RESTATED for #256 M2. The scan is now an EXPLANATION, not a veto:
			// it runs only where there is no conversation to resume into, and
			// says "abandoned" rather than "unbound" because that is the more
			// useful answer. With the ledger resolving, the tombstone is history
			// about bookkeeping and nothing more.
			name: "a tombstoned history does not block a resolvable conversation",
			mutate: func(r *ThreadRecord) {
				r.ParkHistory = []ParkTransaction{{Tombstoned: true, Closed: true}}
			},
		},
		{
			// RESTATED for #272 -- this case WAS the bug, written as a
			// requirement. "An occupied incarnation refuses even with the
			// detached proof" is precisely what made three live muse
			// conversations unreachable: the incarnation named a dead launcher
			// while the detached proof named a session whose agent was still
			// running, and the dead one won.
			name: "a stale incarnation does not refuse a surviving session",
			mutate: func(r *ThreadRecord) {
				r.Incarnations = []ThreadIncarnation{{State: IncarnationLive, PID: 1, Identity: "x", StartedAt: time.Unix(2, 0).UTC()}}
			},
			detached: true,
			wantCode: "",
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

// TestEveryResumeDiagnosticCodeIsProducedBySomeSite is the guard
// ResumeDiagnosticCode lacked, and the class fix for an orphan the M1 review
// found: deleting occupiedResumeCode removed the only site emitting
// ResumeCreating, in the same commit, unnoticed.
//
// ThreadReason has had TestEveryReasonIsProducedBySomeShape for exactly this --
// threadreason.go cites it as the reason `unrecorded-child` was deleted rather
// than kept as a placeholder. A vocabulary with no produced-by guard grows
// values nothing can emit, and each one is a branch every reader must handle
// and no test can reach.
//
// The identifiers are DERIVED from the declaration rather than listed, so the
// guard cannot be satisfied by forgetting to add a row to it.
func TestEveryResumeDiagnosticCodeIsProducedBySomeSite(t *testing.T) {
	declaration, err := os.ReadFile("resume.go")
	if err != nil {
		t.Fatal(err)
	}
	identifiers := regexp.MustCompile(`(?m)^\s*(Resume\w+)\s+ResumeDiagnosticCode\s*=`).FindAllStringSubmatch(string(declaration), -1)
	if len(identifiers) < 5 {
		t.Fatalf("derived only %d codes from resume.go; the regex has drifted from the declaration", len(identifiers))
	}

	var body strings.Builder
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		b, readErr := os.ReadFile(name)
		if readErr != nil {
			t.Fatal(readErr)
		}
		body.Write(b)
	}
	source := body.String()

	for _, match := range identifiers {
		identifier := match[1]
		// One mention is the declaration; a producer or reader is a second.
		if strings.Count(source, identifier) < 2 {
			t.Errorf("nothing produces %s outside its declaration -- "+
				"delete it, or every reader carries a branch no test can reach", identifier)
		}
	}
}

// TestReAdoptionRefusalsClaimOnlyWhatWasProved pins WHICH code each exit emits,
// which no test did -- the gap that let an exit reached on "could not tell"
// emit ResumeNotRunning, whose declared meaning is "not running at all".
//
// A diagnostic code is a claim the operator reads: menu_reattach renders it on
// the row. Emitting "not running" over a live conversation is a false statement
// about the thing the operator most needs to be true.
func TestReAdoptionRefusalsClaimOnlyWhatWasProved(t *testing.T) {
	store, _ := newTestThreadStore(t)
	record := actionableTestThread("couch-00000000000000fb", time.Unix(100, 0).UTC())
	record.LatestLaunchProfile = &LaunchProfile{Agent: "muse", Argv: []string{}}
	record.Incarnations = []ThreadIncarnation{{PID: 4242, Identity: "tok", State: IncarnationLive}}
	created, err := store.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	proc := NewFakeProcOps()
	proc.SetUnknown(4242)

	_, err = (&Couch{Threads: store, Proc: proc}).clearLifecycleDebris(created)
	if got := ResumeDiagnosticOf(err); got != ResumeUnknown {
		t.Fatalf("an unprovable process reports %q; it must claim ignorance, not %q — the agent may well be running", got, ResumeNotRunning)
	}
}
