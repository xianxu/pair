package couchcore

import (
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

type resumeShape struct {
	name       string
	record     ThreadRecord
	binding    NativeBindingResolution
	pathExists bool
}

// asParked models what the PARK will produce, which is the question relaunch
// actually asks: "would this be resumable ONCE PARKED?"
//
// Clearing only the incarnations would leave a record with no verified park, so
// DecideResume would refuse ResumeLegacyUnverified and the comparison below
// would "disagree" for a reason that has nothing to do with the split.
//
// It deliberately does NOT use cloneThreadRecord: cloneArgv normalizes a nil
// Argv to an empty slice (launchprofile.go:172-178), so cloning would silently
// repair the "incomplete profile" shape and make the two predicates agree by
// accident. The transform must change occupancy and park state and nothing else.
func asParked(record ThreadRecord) ThreadRecord {
	parked := record
	parked.Incarnations = nil
	parked.Park = nil
	identity := ParkIdentity{
		Nonce: "park-0123456789abcdef", Address: parked.Address,
		PID: 42, ProcessIdentity: "parked-process",
	}
	parked.VerifiedPark = &VerifiedPark{
		Identity: identity, Attempt: 1, ParkedAt: time.Unix(2000, 0).UTC(),
	}
	return parked
}

func everyResumeShape(t *testing.T) []resumeShape {
	t.Helper()
	live := func(tag ThreadTag) ThreadRecord {
		record := actionableTestThread(tag, time.Unix(1000, 0).UTC())
		record.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
		record.Incarnations = []ThreadIncarnation{{
			PID: 42, Identity: "pair-live", State: IncarnationLive,
		}}
		return record
	}
	established := NativeBindingResolution{
		Status: sessioninventory.BindingEstablished, NativeID: "native-root-1",
	}

	noProfile := live("couch-0000000000000002")
	noProfile.LatestLaunchProfile = nil

	badAgent := live("couch-0000000000000003")
	badAgent.LatestLaunchProfile = &LaunchProfile{Agent: "not-an-agent", Argv: []string{}}

	nilArgv := live("couch-0000000000000004")
	nilArgv.LatestLaunchProfile = &LaunchProfile{Agent: "claude"}

	noPath := live("couch-0000000000000005")
	noPath.WorkingPath = ""

	return []resumeShape{
		{name: "healthy", record: live("couch-0000000000000001"), binding: established, pathExists: true},
		{name: "no saved profile", record: noProfile, binding: established, pathExists: true},
		{name: "unsupported agent", record: badAgent, binding: established, pathExists: true},
		{name: "incomplete profile", record: nilArgv, binding: established, pathExists: true},
		{name: "empty working path", record: noPath, binding: established, pathExists: true},
		{name: "working path gone", record: live("couch-0000000000000006"), binding: established, pathExists: false},
		{name: "provisional binding", record: live("couch-0000000000000007"),
			binding: NativeBindingResolution{Status: sessioninventory.BindingProvisional}, pathExists: true},
		{name: "unbound", record: live("couch-0000000000000008"),
			binding: NativeBindingResolution{Status: sessioninventory.BindingUnbound}, pathExists: true},
		{name: "established but no root id", record: live("couch-0000000000000009"),
			binding: NativeBindingResolution{Status: sessioninventory.BindingEstablished}, pathExists: true},
	}
}

// Relaunch asks "would this be resumable once parked?", which is DecideResume's
// rule set minus the occupancy a park is about to clear. Asserting the AGREEMENT
// is what stops the two drifting: pair#181 M3 shipped an archive guard that
// admitted `creating` while resume refused it, from exactly this kind of
// parallel derivation.
func TestResumePreconditionsMatchDecideResumeOnAPostParkRecord(t *testing.T) {
	for _, tc := range everyResumeShape(t) {
		t.Run(tc.name, func(t *testing.T) {
			precondition := CheckResumePreconditions(tc.record, tc.binding, tc.pathExists)
			_, resumeErr := DecideResume(ResumeEligibilityInput{
				Thread: asParked(tc.record), WorkingPathExists: tc.pathExists, Binding: tc.binding,
			})
			if (precondition == nil) != (resumeErr == nil) {
				t.Fatalf("precondition=%v, post-park resume=%v -- the two disagree", precondition, resumeErr)
			}
			if precondition == nil {
				return
			}
			// And they refuse for the SAME reason, so a relaunch refusal tells
			// the operator what a resume would have told them.
			if got, want := ResumeDiagnosticOf(precondition), ResumeDiagnosticOf(resumeErr); got != want {
				t.Fatalf("precondition refused %q, post-park resume refused %q", got, want)
			}
		})
	}
}
