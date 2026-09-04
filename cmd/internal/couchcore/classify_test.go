package couchcore

import (
	"testing"
	"time"
)

// classifyCase is one record shape plus the evidence the shell resolved about
// it. wasActionableBefore records what the pre-#181 projector answered, so the
// characterization test below can prove M1 changed only the refusals.
type classifyCase struct {
	name                string
	record              ThreadRecord
	evidence            ThreadEvidence
	wantState           ActionableThreadState
	wantReason          ThreadReason
	wasActionableBefore bool
}

func classifyProfile() *LaunchProfile {
	return &LaunchProfile{Agent: "claude", Argv: []string{}}
}

func everyThreadShape(t *testing.T) []classifyCase {
	t.Helper()
	active := time.Unix(1000, 0).UTC()
	live := func() ThreadRecord {
		record := actionableTestThread("couch-0000000000000001", active)
		record.LatestLaunchProfile = classifyProfile()
		record.Incarnations = []ThreadIncarnation{{
			PID: 42, Identity: "pair-live", State: IncarnationLive,
		}}
		return record
	}
	liveObservation := []ProcessIdentity{{PID: 42, Identity: "pair-live"}}

	parked := func() ThreadRecord {
		record := actionableTestThread("couch-0000000000000002", active)
		record.LatestLaunchProfile = classifyProfile()
		markActionableParked(&record, active)
		return record
	}
	parkedProof := func(record ThreadRecord) []ParkedResumeObservation {
		return []ParkedResumeObservation{{Address: record.Address, Agent: "claude", NativeID: "native-1"}}
	}

	detached := func() ThreadRecord {
		record := actionableTestThread("couch-0000000000000003", active)
		record.LatestLaunchProfile = classifyProfile()
		return record
	}
	detachedProof := func(record ThreadRecord) []DetachedSessionObservation {
		return []DetachedSessionObservation{{
			Address: record.Address, SessionName: "pair-three", Agent: "claude", NativeID: "native-3",
		}}
	}
	// The operator's pair-couch-24: a live session whose binding never landed.
	detachedNoBinding := func(record ThreadRecord) []DetachedSessionObservation {
		return []DetachedSessionObservation{{
			Address: record.Address, SessionName: "pair-three", Agent: "claude",
		}}
	}

	resolved := func(e ThreadEvidence) ThreadEvidence {
		e.ParkedStatus, e.DetachedStatus = ProofResolved, ProofResolved
		return e
	}

	invalid := actionableTestThread("couch-0000000000000004", active)
	invalid.SchemaVersion = 0

	reservation := actionableTestThread("couch-0000000000000005", active)
	reservation.Reservation = true

	// A park in flight is a park of something RUNNING: the store requires the
	// active transaction's identity to match a live incarnation.
	parking := actionableTestThread("couch-0000000000000006", active)
	parking.LatestLaunchProfile = classifyProfile()
	parking.Incarnations = []ThreadIncarnation{{
		PID: 42, Identity: "pair-parking", State: IncarnationLive,
	}}
	parking.Revision = 2
	parking.Park = &ParkTransaction{
		Identity: ParkIdentity{
			Nonce: "park-0123456789abcdef", Address: parking.Address,
			PID: 42, ProcessIdentity: "pair-parking",
		},
		BaseRevision:   1,
		RecordRevision: 2,
		Phase:          ParkAwaitingCompletion,
		Attempts:       []ParkAttempt{{Number: 1}},
	}

	noProfile := detached()
	noProfile.Address.Tag = "couch-0000000000000007"
	noProfile.LatestLaunchProfile = nil

	badAgent := detached()
	badAgent.Address.Tag = "couch-0000000000000008"
	badAgent.LatestLaunchProfile = &LaunchProfile{Agent: "not-an-agent", Argv: []string{}}

	staleLive := live()
	staleLive.Address.Tag = "couch-0000000000000009"

	unrecorded := detached()
	unrecorded.Address.Tag = "couch-000000000000000a"

	parkedRecord, detachedRecord := parked(), detached()
	pathBroken := detached()
	pathBroken.Address.Tag = "couch-000000000000000b"

	return []classifyCase{
		{
			name: "live and hosted", record: live(),
			evidence:  resolved(ThreadEvidence{Live: liveObservation}),
			wantState: ThreadLive, wasActionableBefore: true,
		},
		{
			name: "live record nothing hosts it", record: staleLive,
			evidence:  resolved(ThreadEvidence{}),
			wantState: ThreadUnusable, wantReason: ReasonStaleIncarnation,
		},
		{
			name: "hosted child with no incarnation", record: unrecorded,
			evidence:  resolved(ThreadEvidence{Live: liveObservation}),
			wantState: ThreadUnusable, wantReason: ReasonUnrecordedChild,
		},
		{
			name: "verified park with its resume proof", record: parkedRecord,
			evidence:  resolved(ThreadEvidence{Parked: parkedProof(parkedRecord)}),
			wantState: ThreadParked, wasActionableBefore: true,
		},
		{
			name: "verified park whose binding was lost", record: parked(),
			evidence:  resolved(ThreadEvidence{}),
			wantState: ThreadUnusable, wantReason: ReasonBindingLost,
		},
		{
			name: "verified park whose proof could not be resolved", record: parked(),
			evidence:  ThreadEvidence{DetachedStatus: ProofResolved},
			wantState: ThreadUnusable, wantReason: ReasonUnknown,
		},
		{
			name: "detached with its resume proof", record: detachedRecord,
			evidence:  resolved(ThreadEvidence{Detached: detachedProof(detachedRecord)}),
			wantState: ThreadDetached, wasActionableBefore: true,
		},
		{
			name: "detached whose session is alive but binding lost", record: detachedRecord,
			evidence:  resolved(ThreadEvidence{Detached: detachedNoBinding(detachedRecord)}),
			wantState: ThreadUnusable, wantReason: ReasonBindingLost,
		},
		{
			name: "no incarnation and no session", record: detached(),
			evidence:  resolved(ThreadEvidence{}),
			wantState: ThreadUnusable, wantReason: ReasonSessionGone,
		},
		{
			name: "no incarnation and the session question could not be asked", record: detached(),
			evidence:  ThreadEvidence{ParkedStatus: ProofResolved},
			wantState: ThreadUnusable, wantReason: ReasonUnknown,
		},
		{
			name: "reservation that never started", record: reservation,
			evidence:  resolved(ThreadEvidence{}),
			wantState: ThreadUnusable, wantReason: ReasonNeverStarted,
		},
		{
			name: "park transaction in flight", record: parking,
			evidence:  resolved(ThreadEvidence{}),
			wantState: ThreadBusy,
		},
		{
			name: "record that fails validation", record: invalid,
			evidence:  resolved(ThreadEvidence{}),
			wantState: ThreadUnusable, wantReason: ReasonInvalid,
		},
		{
			name: "no saved launch profile", record: noProfile,
			evidence:  resolved(ThreadEvidence{}),
			wantState: ThreadUnusable, wantReason: ReasonProfileMissing,
		},
		{
			name: "unsupported saved agent", record: badAgent,
			evidence:  resolved(ThreadEvidence{}),
			wantState: ThreadUnusable, wantReason: ReasonAgentUnsupported,
		},
		{
			name: "working path could not be physicalized", record: pathBroken,
			evidence:  resolved(ThreadEvidence{PathError: errTestPathBroken}),
			wantState: ThreadUnusable, wantReason: ReasonPathMissing,
		},
	}
}

// The property that makes the inventory honest: every record produces a row.
// The cross product, not a sample -- a future branch that forgets to classify
// something fails here instead of silently vanishing.
func TestClassifyThreadIsTotalOverEveryRecordShape(t *testing.T) {
	for _, tc := range everyThreadShape(t) {
		t.Run(tc.name, func(t *testing.T) {
			state, reason := ClassifyThread(tc.record, tc.evidence)
			if state == "" {
				t.Fatalf("ClassifyThread returned no state")
			}
			if (state == ThreadUnusable) != (reason != "") {
				t.Fatalf("state=%q reason=%q -- a reason iff unusable", state, reason)
			}
			if state != tc.wantState || reason != tc.wantReason {
				t.Fatalf("= (%q, %q), want (%q, %q)", state, reason, tc.wantState, tc.wantReason)
			}
		})
	}
}

// The characterization half: the accepting branches must be exactly what the
// pre-#181 projector accepted, so M1 provably changes only the refusals.
func TestClassifyThreadAcceptsExactlyWhatTheOldProjectorAccepted(t *testing.T) {
	for _, tc := range everyThreadShape(t) {
		state, _ := ClassifyThread(tc.record, tc.evidence)
		actionable := state == ThreadLive || state == ThreadParked || state == ThreadDetached
		if actionable != tc.wasActionableBefore {
			t.Fatalf("%s: actionable=%v, previously %v", tc.name, actionable, tc.wasActionableBefore)
		}
	}
}

// A live row must not be refused for evidence that only resume candidates need.
// Today a record carrying an incarnation never reaches path physicalization or
// a profile read (actionableinventory.go:237), and M1 keeps that true.
func TestClassifyThreadDoesNotApplyResumeShapedRefusalsToALiveRow(t *testing.T) {
	record := actionableTestThread("couch-000000000000000c", time.Unix(1000, 0).UTC())
	record.Incarnations = []ThreadIncarnation{{PID: 42, Identity: "pair-live", State: IncarnationLive}}
	record.LatestLaunchProfile = nil

	state, reason := ClassifyThread(record, ThreadEvidence{
		Live:      []ProcessIdentity{{PID: 42, Identity: "pair-live"}},
		PathError: errTestPathBroken,
	})

	if state != ThreadLive || reason != "" {
		t.Fatalf("= (%q, %q), want a live row: a running agent whose directory moved is still running", state, reason)
	}
}

// Every reason is reachable from some record shape. A reason nothing produces
// is a vocabulary that has drifted from the classifier.
func TestEveryReasonIsProducedBySomeShape(t *testing.T) {
	produced := map[ThreadReason]bool{}
	for _, tc := range everyThreadShape(t) {
		if _, reason := ClassifyThread(tc.record, tc.evidence); reason != "" {
			produced[reason] = true
		}
	}
	for _, reason := range AllThreadReasons() {
		if !produced[reason] {
			t.Errorf("no record shape produces reason %q", reason)
		}
	}
}

var errTestPathBroken = errTestPath{}

type errTestPath struct{}

func (errTestPath) Error() string { return "working path is unavailable" }
