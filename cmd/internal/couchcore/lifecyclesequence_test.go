package couchcore

import (
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/pairlifecycle"
)

// TestTheWorldDecidesWhateverTheRecordSaysAboutItself is #256's claim, stated as
// the property rather than as one test per incident.
//
// ARCH-ORDER asks which events the caller cannot block. Four apply, and none of
// them is a transaction that can be rolled back -- they are things that already
// happened to processes couch does not own:
//
//	couch killed, session survives   -> classify from the session
//	couch killed, session also dies  -> the ledger decides gone vs resumable
//	park times out, process dies     -> nothing reads record.Park
//	liveness probe returns Unknown   -> fail closed
//
// Each leaves DIFFERENT bookkeeping behind and the SAME external world. The
// property is that the bookkeeping does not survive into the verdict: every
// record shape below classifies identically within a column. That is the whole
// of #271 and #272 -- a clean `alt+d` and a couch crash are indistinguishable
// from outside, because the zellij server is PPID 1 at birth and outlives both.
//
// There is no rollback column because the classifier holds no state between
// events: it is a pure function of (record, evidence), which is what the re-cut
// bought.
func TestTheWorldDecidesWhateverTheRecordSaysAboutItself(t *testing.T) {
	active := time.Unix(1000, 0).UTC()
	base := func(tag ThreadTag) ThreadRecord {
		record := actionableTestThread(tag, active)
		record.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
		return record
	}

	// Four records. One external world each time.
	cleanDetach := base("couch-0000000000000101")

	crashed := base("couch-0000000000000102")
	crashed.Incarnations = []ThreadIncarnation{{PID: 4242, Identity: "launcher", State: IncarnationLive}}

	parkTimedOut := base("couch-0000000000000103")
	parkTimedOut.Incarnations = []ThreadIncarnation{{PID: 4242, Identity: "launcher", State: IncarnationLive}}
	parkTimedOut.Revision = 2
	parkTimedOut.Park = &ParkTransaction{
		Identity: ParkIdentity{
			Nonce: "park-0123456789abcdef", Address: parkTimedOut.Address,
			PID: 4242, ProcessIdentity: "launcher",
		},
		BaseRevision: 1, RecordRevision: 2, Phase: ParkAwaitingCompletion,
		Attempts: []ParkAttempt{{
			Number: 1, TimedOut: true,
			Failure: &ParkFailure{
				Code:       pairlifecycle.FailureTimeout,
				Diagnostic: "matching completion was not observed",
			},
		}},
	}

	claimAbandoned := base("couch-0000000000000104")
	claimAbandoned.Incarnations = []ThreadIncarnation{{
		State: IncarnationCreating,
		Start: &ThreadStartClaim{
			Nonce: "start-0123456789abcdef", OwnerPID: 900, OwnerIdentity: "dead-couch",
		},
	}}

	records := map[string]ThreadRecord{
		"clean detach":       cleanDetach,
		"couch crashed":      crashed,
		"park timed out":     parkTimedOut,
		"start claim orphan": claimAbandoned,
	}

	// StartOwner: Dead throughout. Every shape here describes something that
	// already finished dying; the fail-closed direction gets its own column.
	for _, world := range []struct {
		name       string
		evidence   ThreadEvidence
		wantState  ActionableThreadState
		wantReason ThreadReason
	}{
		{
			name: "the session survived",
			evidence: ThreadEvidence{
				Session: SessionObservation{State: SessionPresent}, StartOwner: Dead,
			},
			wantState: ThreadDetached,
		},
		{
			name: "the session is gone and so is the conversation",
			evidence: ThreadEvidence{
				Session:    SessionObservation{State: SessionAbsent},
				StartOwner: Dead, ParkedStatus: ProofResolved,
			},
			wantState: ThreadUnusable, wantReason: ReasonSessionGone,
		},
		{
			name: "the session is gone but the ledger still resolves",
			evidence: ThreadEvidence{
				Session:    SessionObservation{State: SessionAbsent},
				StartOwner: Dead, ParkedStatus: ProofResolved,
				// Carried in the world literal, not injected from wantState
				// below: an input derived from the expectation makes the table
				// self-consistent instead of a specification. The address is
				// filled per record inside the loop because each shape has its
				// own, which is data the world cannot know.
				Parked: []ParkedResumeObservation{{Agent: "claude", NativeID: "native-1"}},
			},
			wantState: ThreadParked,
		},
		{
			name: "the probe could not answer",
			evidence: ThreadEvidence{
				Session: SessionObservation{State: SessionUnresolved}, StartOwner: Dead,
			},
			wantState: ThreadUnusable, wantReason: ReasonUnknown,
		},
	} {
		t.Run(world.name, func(t *testing.T) {
			for shape, record := range records {
				evidence := world.evidence
				if len(evidence.Parked) == 1 {
					// Only the address is per-record; the world already decided
					// whether a conversation resolves at all.
					observation := evidence.Parked[0]
					observation.Address = record.Address
					evidence.Parked = []ParkedResumeObservation{observation}
				}
				state, reason := ClassifyThread(record, evidence)
				if state != world.wantState || reason != world.wantReason {
					t.Errorf("%s = (%q, %q), want (%q, %q) — the record's own bookkeeping reached the verdict",
						shape, state, reason, world.wantState, world.wantReason)
				}
			}
		})
	}
}

// TestUnknownNeverBecomesGone is the fail-closed direction on its own, because
// it is the one asymmetry in the table above: `session-gone` is archive-eligible
// and `unknown` is not, so a collapse in that direction offers to retire live
// work. Every evidence field that can be unresolved gets a row.
func TestUnknownNeverBecomesGone(t *testing.T) {
	record := actionableTestThread("couch-0000000000000105", time.Unix(1000, 0).UTC())
	record.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}

	for _, tc := range []struct {
		name     string
		evidence ThreadEvidence
		want     ActionableThreadState
	}{
		{
			name:     "the session question could not be asked",
			evidence: ThreadEvidence{ParkedStatus: ProofResolved, StartOwner: Dead},
			want:     ThreadUnusable,
		},
		{
			name:     "the ledger could not be read",
			evidence: ThreadEvidence{Session: SessionObservation{State: SessionAbsent}, StartOwner: Dead},
			want:     ThreadUnusable,
		},
		{
			name: "nothing could be asked at all",
			// The zero ThreadEvidence: every field at its unresolved value.
			evidence: ThreadEvidence{},
			want:     ThreadUnusable,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state, reason := ClassifyThread(record, tc.evidence)
			if state != tc.want || reason != ReasonUnknown {
				t.Fatalf("= (%q, %q), want (%q, %q): ignorance is not absence, and absence is archive-eligible",
					state, reason, tc.want, ReasonUnknown)
			}
		})
	}
}
