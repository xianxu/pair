package couchcore

import (
	"context"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
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

// couchWithOneRecordOfEveryShape builds a real store holding one record of each
// shape the shell used to drop, so the identity below is asserted over
// production code rather than a hand-built projection.
func couchWithOneRecordOfEveryShape(t *testing.T) (*Couch, []ThreadAddress) {
	t.Helper()
	store, _ := newTestThreadStore(t)
	active := time.Unix(100, 0).UTC()
	var addresses []ThreadAddress

	create := func(record ThreadRecord) ThreadRecord {
		t.Helper()
		created, err := store.CreateThread(record)
		if err != nil {
			t.Fatalf("create %s: %v", record.Address.Tag, err)
		}
		addresses = append(addresses, created.Address)
		return created
	}

	// Parked with an established binding: the one row that already worked.
	parked := actionableTestThread("couch-0000000000000001", active)
	parked.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	markActionableParked(&parked, active)
	parkedRecord := create(parked)

	// Parked whose binding was lost -- eight of the operator's thirteen.
	lost := actionableTestThread("couch-0000000000000002", active)
	lost.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	markActionableParked(&lost, active)
	create(lost)

	// A record claiming a live incarnation that no console hosts.
	stale := actionableTestThread("couch-0000000000000003", active)
	stale.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	stale.Incarnations = []ThreadIncarnation{{PID: 4242, Identity: "gone", State: IncarnationLive}}
	create(stale)

	// No saved launch profile.
	profileless := actionableTestThread("couch-0000000000000004", active)
	create(profileless)

	// A saved profile naming an agent this build cannot launch.
	badAgent := actionableTestThread("couch-0000000000000005", active)
	badAgent.LatestLaunchProfile = &LaunchProfile{Agent: "not-an-agent", Argv: []string{}}
	create(badAgent)

	// Nothing at all: no incarnation, no park, no session.
	gone := actionableTestThread("couch-0000000000000006", active)
	gone.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	create(gone)

	artifacts := NewFakeThreadArtifactCollisionChecker()
	artifacts.SetNativeBinding(parkedRecord.Address, "claude", sessioninventory.BindingEstablished, "native-root-1")
	couch := &Couch{Threads: store, Artifacts: artifacts, Path: NewFakePathOps(nil)}
	return couch, addresses
}

// The regression this issue is: nine of thirteen records reached no row.
// Asserted as an identity between the store and the projection -- not a count
// of the rows we expect to be actionable, but of the records that exist.
func TestInventoryEmitsOneRowPerManifestRecord(t *testing.T) {
	couch, addresses := couchWithOneRecordOfEveryShape(t)

	rows, err := couch.ActionableThreadInventoryContext(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(rows) != len(addresses) {
		t.Fatalf("rows = %d, records = %d -- the shell dropped %d",
			len(rows), len(addresses), len(addresses)-len(rows))
	}
	for _, address := range addresses {
		row, ok := findInventoryRow(rows, address)
		if !ok {
			t.Fatalf("record %+v produced no row", address)
		}
		if row.State == ThreadUnusable && row.Reason == "" {
			t.Fatalf("row %+v is unusable with no reason", row)
		}
	}
}

// Every row that is NOT actionable carries a reason the operator can read, and
// the actionable ones carry none.
func TestInventoryRowsCarryAReasonExactlyWhenUnusable(t *testing.T) {
	couch, _ := couchWithOneRecordOfEveryShape(t)
	rows, err := couch.ActionableThreadInventoryContext(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		if (row.State == ThreadUnusable) != (row.Reason != "") {
			t.Fatalf("row %+v: a reason iff unusable", row)
		}
	}
}

func findInventoryRow(rows []ActionableThreadSummary, address ThreadAddress) (ActionableThreadSummary, bool) {
	for _, row := range rows {
		if row.Address == address {
			return row, true
		}
	}
	return ActionableThreadSummary{}, false
}

// actionableRows is the pre-#181 shape of the projector: only the rows an
// operator can act on.
//
// The projector is now total, so tests whose subject is which records are
// ACTIONABLE -- the fail-closed property, unchanged -- filter for that
// explicitly instead of counting rows. That every record produces a row is a
// different property with its own tests above.
func actionableRows(records []ThreadRecord, live []LiveTTYObservation, parked []ParkedResumeObservation, detached []DetachedSessionObservation) []ActionableThreadSummary {
	evidence := make(map[ThreadAddress]ThreadEvidence, len(records))
	for _, record := range records {
		evidence[record.Address] = ThreadEvidence{ParkedStatus: ProofResolved, DetachedStatus: ProofResolved}
	}
	for _, observation := range live {
		item := evidence[observation.Address]
		item.Live = append(item.Live, observation.Process)
		item.ParkedStatus, item.DetachedStatus = ProofResolved, ProofResolved
		evidence[observation.Address] = item
	}
	for _, observation := range parked {
		item := evidence[observation.Address]
		item.Parked = append(item.Parked, observation)
		item.ParkedStatus, item.DetachedStatus = ProofResolved, ProofResolved
		evidence[observation.Address] = item
	}
	for _, observation := range detached {
		item := evidence[observation.Address]
		item.Detached = append(item.Detached, observation)
		item.ParkedStatus, item.DetachedStatus = ProofResolved, ProofResolved
		evidence[observation.Address] = item
	}
	var rows []ActionableThreadSummary
	for _, row := range ProjectActionableThreads(records, evidence) {
		switch row.State {
		case ThreadLive, ThreadParked, ThreadDetached:
			rows = append(rows, row)
		}
	}
	return rows
}

// countingPathOps and countingArtifacts count the per-record work the evidence
// pass does. They are test-local wrappers rather than counters on the shared
// fakes: only this guard cares, and a counter every other test carries is a
// counter every other test can accidentally assert on.
type countingPathOps struct {
	PathOps
	calls int
}

func (c *countingPathOps) Physical(path string) (string, error) {
	c.calls++
	return c.PathOps.Physical(path)
}

type countingArtifacts struct {
	*FakeThreadArtifactCollisionChecker
	resolveCalls  int
	detachQueries int
}

func (c *countingArtifacts) ResolveEstablished(ctx context.Context, scope, tag, agent string) (NativeBindingResolution, error) {
	c.resolveCalls++
	return c.FakeThreadArtifactCollisionChecker.ResolveEstablished(ctx, scope, tag, agent)
}

func (c *countingArtifacts) DetachedSessions(ctx context.Context, candidates []DetachedCandidate) ([]DetachedSessionObservation, error) {
	c.detachQueries++
	return c.FakeThreadArtifactCollisionChecker.DetachedSessions(ctx, candidates)
}

// The cost bound nothing observed before. BenchmarkMenu100 runs over a fixture
// slice of summaries and never reaches the inventory, the resolver, Physical or
// zellij, so it could not see per-record work at all. This can: resume-shaped
// records pay, and every other record is free.
func TestEvidencePassAsksOnlyAboutResumeShapedRecords(t *testing.T) {
	couch, _ := couchWithOneRecordOfEveryShape(t)
	paths := &countingPathOps{PathOps: couch.Path}
	artifacts := &countingArtifacts{
		FakeThreadArtifactCollisionChecker: couch.Artifacts.(*FakeThreadArtifactCollisionChecker),
	}
	couch.Path, couch.Artifacts = paths, artifacts

	if _, err := couch.ActionableThreadInventoryContext(context.Background(), nil); err != nil {
		t.Fatal(err)
	}

	// Of the six shapes, three are resume-shaped: the parked row with a
	// binding, the parked row that lost one, and the one with nothing left.
	// The stale incarnation, the profile-less record and the unsupported agent
	// must cost nothing at all.
	const resumeShaped = 3
	if paths.calls != resumeShaped {
		t.Fatalf("Physical called %d times, want %d -- a live or unstartable record must not pay", paths.calls, resumeShaped)
	}
	if artifacts.resolveCalls != resumeShaped {
		t.Fatalf("binding resolver called %d times, want %d", artifacts.resolveCalls, resumeShaped)
	}
	// One detach candidate (the record with no park and no incarnation), so
	// exactly one zellij query -- and it is a query per REFRESH, not per row.
	if artifacts.detachQueries != 1 {
		t.Fatalf("detached query ran %d times, want 1", artifacts.detachQueries)
	}
}

// The other half of the bound: with nothing detachable, the zellij query does
// not run at all.
func TestEvidencePassSkipsTheSessionQueryWithNoDetachCandidates(t *testing.T) {
	store, _ := newTestThreadStore(t)
	record := actionableTestThread("couch-0000000000000001", time.Unix(100, 0).UTC())
	record.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	markActionableParked(&record, record.LastActiveAt)
	if _, err := store.CreateThread(record); err != nil {
		t.Fatal(err)
	}
	artifacts := &countingArtifacts{FakeThreadArtifactCollisionChecker: NewFakeThreadArtifactCollisionChecker()}
	couch := &Couch{Threads: store, Artifacts: artifacts, Path: NewFakePathOps(nil)}

	if _, err := couch.ActionableThreadInventoryContext(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if artifacts.detachQueries != 0 {
		t.Fatalf("detached query ran %d times with no candidate", artifacts.detachQueries)
	}
}

// The two views over one store. `couch --list` showed thirteen rows while the
// switcher showed four, and nothing reconciled them -- the operator found that
// discrepancy before any test did, because no test compared them.
func TestBothInventoriesReportTheSamePopulationAndStates(t *testing.T) {
	couch, addresses := couchWithOneRecordOfEveryShape(t)

	switcher, err := couch.ActionableThreadInventoryContext(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	diagnostic, err := couch.ThreadInventoryContext(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if len(switcher) != len(addresses) || len(diagnostic) != len(addresses) {
		t.Fatalf("switcher=%d diagnostic=%d records=%d", len(switcher), len(diagnostic), len(addresses))
	}
	states := make(map[ThreadAddress]ActionableThreadState, len(switcher))
	reasons := make(map[ThreadAddress]ThreadReason, len(switcher))
	for _, row := range switcher {
		states[row.Address], reasons[row.Address] = row.State, row.Reason
	}
	for _, row := range diagnostic {
		if states[row.Address] != row.State || reasons[row.Address] != row.Reason {
			t.Fatalf("%+v: diagnostic says (%q,%q), switcher says (%q,%q)",
				row.Address, row.State, row.Reason, states[row.Address], reasons[row.Address])
		}
	}
}
