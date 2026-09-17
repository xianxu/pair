package couchcore

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// classifyCase is one record shape plus the evidence the shell resolved about
// it. wasActionableBefore records what the pre-#181 projector answered, so the
// characterization test below can prove M1 changed only the refusals.
type classifyCase struct {
	name       string
	record     ThreadRecord
	evidence   ThreadEvidence
	wantState  ActionableThreadState
	wantReason ThreadReason
	// wasActionableBefore is the pre-#181 projector's verdict, kept as a
	// characterization ratchet.
	wasActionableBefore bool
	// newlyActionable marks a shape this issue DELIBERATELY admits that the old
	// projector refused. It replaces a name-matched exception, which broke the
	// moment a case was renamed -- an exception keyed to prose is a guard that
	// silently stops guarding.
	newlyActionable bool
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
	resolved := func(e ThreadEvidence) ThreadEvidence {
		e.ParkedStatus = ProofResolved
		e.Session = SessionObservation{State: SessionAbsent}
		return e
	}
	withSession := func(e ThreadEvidence) ThreadEvidence {
		e.ParkedStatus = ProofResolved
		e.Session = SessionObservation{State: SessionPresent}
		return e
	}
	// A start couch has claimed and not finished: the ONE thing still read from
	// an incarnation, and it is a record of couch's own operation rather than a
	// claim about an external process.
	starting := detached()
	starting.Address.Tag = "couch-000000000000000c"
	starting.Incarnations = []ThreadIncarnation{{
		State: IncarnationCreating,
		Start: &ThreadStartClaim{
			Nonce: "start-0123456789abcdef", OwnerPID: 4242, OwnerIdentity: "supervisor",
		},
	}}

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
			// RESTATED for #272. This was `stale-incarnation`: a record claiming
			// a live incarnation that nothing hosts. But the incarnation names
			// the LAUNCHER, which dies with couch, so that reason described
			// every couch crash as a lost thread. With no session either, the
			// honest answer is the same one a record with no incarnation gets.
			name: "recorded incarnation is gone and so is the session", record: staleLive,
			evidence:  resolved(ThreadEvidence{}),
			wantState: ThreadUnusable, wantReason: ReasonSessionGone,
		},
		{
			// RESTATED for #272. This was `unrecorded-child`: a hosted process
			// for a record carrying no incarnation, treated as a contradiction
			// to fail closed on. It is not a contradiction any more -- couch
			// hosting the process IS the live proof, and the incarnation is not
			// consulted, so there are no two sides to disagree.
			name: "couch hosts it and the record says nothing", record: unrecorded,
			evidence:  resolved(ThreadEvidence{Live: liveObservation}),
			wantState: ThreadLive, newlyActionable: true,
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
			evidence:  ThreadEvidence{Session: SessionObservation{State: SessionAbsent}},
			wantState: ThreadUnusable, wantReason: ReasonUnknown,
		},
		{
			// The asymmetry #256 M2 keeps deliberately. A failed
			// `list-sessions` must not demote every parked row -- couch tore
			// this session down itself, so the session answer adds nothing that
			// the receipt has not already settled. Contrast with "no park
			// receipt and an unaskable session" below, where the session may be
			// ALIVE and `parked` would invite a second agent onto it.
			name: "verified park whose session could not be asked about", record: parkedRecord,
			evidence: ThreadEvidence{
				ParkedStatus: ProofResolved, Parked: parkedProof(parkedRecord),
			},
			wantState: ThreadParked, wasActionableBefore: true,
		},
		{
			name: "no park receipt and an unaskable session", record: detachedRecord,
			evidence: ThreadEvidence{
				ParkedStatus: ProofResolved, Parked: parkedProof(detachedRecord),
			},
			wantState: ThreadUnusable, wantReason: ReasonUnknown,
		},
		{
			// The session's own presence is the warm proof. It no longer needs a
			// separate detached observation, which cost a `list-clients` per
			// candidate to produce and answered a question the ACTION path
			// re-asks anyway.
			name: "session survived its host", record: detachedRecord,
			evidence:  withSession(ThreadEvidence{}),
			wantState: ThreadDetached, wasActionableBefore: true,
		},
		{
			// #272's shape: the launcher died, the session did not. This and the
			// row above are the SAME external world, and they must classify
			// identically -- that is the whole issue.
			name: "session survived a dead launcher", record: staleLive,
			evidence:  withSession(ThreadEvidence{}),
			wantState: ThreadDetached, newlyActionable: true,
		},
		{
			name: "no incarnation and no session", record: detached(),
			evidence:  resolved(ThreadEvidence{}),
			wantState: ThreadUnusable, wantReason: ReasonSessionGone,
		},
		{
			// #256 M2, the SAFETY half. Identical record to the row above --
			// no park receipt, no session -- but its ledger still names a
			// conversation. It read `session-gone`, which is archive-eligible,
			// because the ledger was only ever read for records carrying a
			// receipt. Nobody asked, so archive could discard a live thread of
			// work without saying so.
			name: "no park receipt, but the ledger still resolves", record: detachedRecord,
			evidence:  resolved(ThreadEvidence{Parked: parkedProof(detachedRecord)}),
			wantState: ThreadParked, newlyActionable: true,
		},
		{
			// The other side: an unreadable ledger is not an empty one.
			// `unknown` is not archive-eligible; `session-gone` is.
			name: "the ledger could not be read", record: detached(),
			evidence:  ThreadEvidence{Session: SessionObservation{State: SessionAbsent}},
			wantState: ThreadUnusable, wantReason: ReasonUnknown,
		},
		{
			name: "the session question could not be asked", record: detached(),
			evidence:  ThreadEvidence{ParkedStatus: ProofResolved},
			wantState: ThreadUnusable, wantReason: ReasonUnknown,
		},
		{
			name: "reservation that never started", record: reservation,
			evidence:  resolved(ThreadEvidence{}),
			wantState: ThreadUnusable, wantReason: ReasonNeverStarted,
		},
		{
			// RESTATED for #271. This was ThreadBusy, unconditionally, from a
			// branch above every evidence-consulting one -- so a park whose
			// process died 18 hours earlier still read `parking…` forever, with
			// no timeout, no expiry and no owner check. The park is no longer
			// consulted at all: the session answers.
			name: "park in flight whose session is still up", record: parking,
			evidence:  withSession(ThreadEvidence{}),
			wantState: ThreadDetached, newlyActionable: true,
		},
		{
			name: "park that timed out and whose session is gone", record: parking,
			evidence:  resolved(ThreadEvidence{}),
			wantState: ThreadUnusable, wantReason: ReasonSessionGone,
		},
		{
			// ThreadBusy's only remaining producer. Without it the window
			// between claiming a start and the launcher acquiring a pid would
			// classify `session-gone` -- an archive-eligible reason -- for a
			// thread starting normally.
			name: "start this couch is driving", record: starting,
			evidence:  resolved(ThreadEvidence{StartOwner: Live}),
			wantState: ThreadBusy,
		},
		{
			// Fail closed. An owner nothing could probe is not a dead owner,
			// and releasing the row on ignorance would offer archive on a
			// thread that is starting normally.
			name: "start whose owner could not be probed", record: starting,
			evidence:  resolved(ThreadEvidence{}),
			wantState: ThreadBusy,
		},
		{
			// #256 M2: the claim outlived the couch that made it. The agent it
			// started is still there, so the row reports the world -- it used
			// to read `starting...` forever, offering neither resume nor
			// archive.
			name: "start claimed by a couch that is gone, session survived", record: starting,
			evidence:  withSession(ThreadEvidence{StartOwner: Dead}),
			wantState: ThreadDetached, newlyActionable: true,
		},
		{
			// The same driverless claim with nothing left behind it. Archive is
			// the only honest offer, and `session-gone` is what makes it.
			name: "start claimed by a couch that is gone, session too", record: starting,
			evidence:  resolved(ThreadEvidence{StartOwner: Dead}),
			wantState: ThreadUnusable, wantReason: ReasonSessionGone,
		},
		{
			// #256 M2, BR-33: the driverless claim whose LEDGER still resolves.
			// A fourth producer of `parked`, and the one that reaches the action
			// guards still carrying a `creating` incarnation -- so every guard
			// must clear that debris rather than trip over it.
			name: "start claimed by a couch that is gone, but the ledger resolves", record: starting,
			evidence:  resolved(ThreadEvidence{StartOwner: Dead, Parked: parkedProof(starting)}),
			wantState: ThreadParked, newlyActionable: true,
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
// pre-#181 projector accepted, except #248 intentionally admits unbound warm sessions.
func TestClassifyThreadAcceptsExactlyWhatTheOldProjectorAccepted(t *testing.T) {
	for _, tc := range everyThreadShape(t) {
		state, _ := ClassifyThread(tc.record, tc.evidence)
		actionable := state == ThreadLive || state == ThreadParked || state == ThreadDetached
		if actionable != (tc.wasActionableBefore || tc.newlyActionable) {
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

// Every reason is reachable. A reason nothing produces is a vocabulary that has
// drifted from the code, which is how `invalid` came to have a label, an Enter
// notice and a documented archive exit that no real store could ever reach.
//
// The vocabulary has TWO producers and the guard covers both: ClassifyThread
// answers about a record, and the projector answers about a manifest entry that
// never became one. Exempting the second would be the same drift one layer up.
func TestEveryReasonIsProducedBySomeShape(t *testing.T) {
	produced := map[ThreadReason]bool{}
	for _, tc := range everyThreadShape(t) {
		if _, reason := ClassifyThread(tc.record, tc.evidence); reason != "" {
			produced[reason] = true
		}
	}
	for _, row := range ProjectActionableThreads(ThreadProjectionInput{
		Unreadable: []ThreadAddress{{RepoScope: "scope", Tag: "couch-0000000000000001"}},
	}) {
		if row.Reason != "" {
			produced[row.Reason] = true
		}
	}
	for _, reason := range AllThreadReasons() {
		if !produced[reason] {
			t.Errorf("nothing produces reason %q", reason)
		}
	}
}

// The STATE vocabulary's produced-by guard, and the justification the action
// tables lean on when they skip `archived`.
//
// Both directions, because each catches a different drift: a state with no
// producer is a branch no test can reach (the reason half of this has caught
// two), and `archived` having one would mean the switcher can render a row the
// action predicates were written to consider impossible. `archived` belongs to
// BuildArchivedInventory, which projects retired records WITHOUT classifying
// them, so the classifying projection must never emit it.
func TestProjectionNeverProducesArchived(t *testing.T) {
	produced := map[ActionableThreadState]bool{}
	for _, tc := range everyThreadShape(t) {
		state, _ := ClassifyThread(tc.record, tc.evidence)
		produced[state] = true
	}
	for _, row := range ProjectActionableThreads(ThreadProjectionInput{
		Unreadable: []ThreadAddress{{RepoScope: "scope", Tag: "couch-0000000000000001"}},
	}) {
		produced[row.State] = true
	}
	if produced[ThreadArchived] {
		t.Errorf("the classifying projection produced %q; the action tables skip it as impossible", ThreadArchived)
	}
	for _, state := range AllThreadStates() {
		if state == ThreadArchived {
			continue
		}
		if !produced[state] {
			t.Errorf("nothing produces state %q", state)
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
	// A prober that answers. Without one, #256 M3 reads every recorded process
	// as UNPROVEN and the `stale` shape above classifies `unknown` -- an honest
	// answer to "couch cannot probe", and a fixture that quietly stops covering
	// the shape it was built for. The fake's table is empty, so pid 4242 is
	// proved Dead, which is what "a record claiming an incarnation that no
	// console hosts" was always meant to model.
	couch := &Couch{Threads: store, Artifacts: artifacts, Proc: NewFakeProcOps(), Path: NewFakePathOps(nil)}
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
// `detached` is now expressed as SESSION PRESENCE rather than a per-candidate
// observation: the session's own survival is the warm proof.
func actionableRows(records []ThreadRecord, live []LiveTTYObservation, parked []ParkedResumeObservation, detached []DetachedSessionObservation) []ActionableThreadSummary {
	evidence := make(map[ThreadAddress]ThreadEvidence, len(records))
	asked := func(item ThreadEvidence) ThreadEvidence {
		item.ParkedStatus = ProofResolved
		if item.Session.State == SessionUnresolved {
			item.Session.State = SessionAbsent
		}
		return item
	}
	for _, record := range records {
		evidence[record.Address] = asked(ThreadEvidence{})
	}
	for _, observation := range live {
		item := evidence[observation.Address]
		item.Live = append(item.Live, observation.Process)
		evidence[observation.Address] = asked(item)
	}
	for _, observation := range parked {
		item := evidence[observation.Address]
		item.Parked = append(item.Parked, observation)
		evidence[observation.Address] = asked(item)
	}
	seenDetached := map[ThreadAddress]bool{}
	for _, observation := range detached {
		item := evidence[observation.Address]
		switch {
		case seenDetached[observation.Address], observation.SessionName == "":
			// Two observations for one address, or a nameless one, prove
			// nothing -- ProjectSessionPresence fails closed on exactly these,
			// so the helper must model the same refusal.
			item.Session = SessionObservation{State: SessionUnresolved}
		default:
			item.Session = SessionObservation{State: SessionPresent}
		}
		seenDetached[observation.Address] = true
		evidence[observation.Address] = item
		continue
	}
	var rows []ActionableThreadSummary
	for _, row := range ProjectActionableThreads(ThreadProjectionInput{Records: records, Evidence: evidence}) {
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

	// RESTATED for #256. "Resume-shaped" is now about RESUME AUTHORITY, not
	// about the bookkeeping, so a record carrying an incarnation pays too -- it
	// has to, because its session may have outlived its launcher and the path
	// is what a reattach needs. Four of the six shapes have a usable profile;
	// the profile-less record and the unsupported agent still cost nothing.
	const resumeShaped = 4
	if paths.calls != resumeShaped {
		t.Fatalf("Physical called %d times, want %d -- a live or unstartable record must not pay", paths.calls, resumeShaped)
	}
	// RESTATED for #256 M2. The ledger read used to be gated on
	// `record.VerifiedPark != nil` -- two records here. It is now gated on the
	// SESSION: every resume-shaped record whose session is not up pays, because
	// the receipt was never the authority over whether a conversation survives.
	// All four resume-shaped records in this fixture have no session, so all
	// four pay. The bound that matters is the one below: a row couch is hosting,
	// or whose session is up, still pays nothing.
	if artifacts.resolveCalls != resumeShaped {
		t.Fatalf("binding resolver called %d times, want %d -- one per resume-shaped record with no session", artifacts.resolveCalls, resumeShaped)
	}
	// RESTATED for #256. The refresh asks PRESENCE, one host-wide call covering
	// every record, and never asks for clients -- a `list-clients` costs ~250 ms
	// per live session (#228) and the reattach path re-observes attach state
	// before committing anyway.
	if artifacts.detachQueries != 0 {
		t.Fatalf("the refresh ran %d client-counting queries; it must ask none", artifacts.detachQueries)
	}
	if queries := artifacts.SessionPresenceQueries(); queries != 1 {
		t.Fatalf("SessionPresence ran %d times, want exactly one host-wide query", queries)
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

// detachedThreadStore is one record shaped exactly like the operator's
// pair-couch-24: no incarnation, no verified park, a saved profile -- the only
// shape that can be detached, and therefore the only one whose session question
// can go unanswered.
func detachedThreadStore(t *testing.T) (*ThreadStore, ThreadAddress) {
	t.Helper()
	store, _ := newTestThreadStore(t)
	record := actionableTestThread("couch-0000000000000001", time.Unix(100, 0).UTC())
	record.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	created, err := store.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	return store, created.Address
}

// The shell half of the absence-vs-negative rule, which lived only in the pure
// layer: mutating `if detachErr == nil` to always-true produced no failure
// anywhere in the suite. A failed session query must leave the question
// UNRESOLVED -- asserting session-gone here hands retirement a reason to act on
// because one subprocess call failed.
func TestAFailedSessionQueryLeavesTheRowUnknownRatherThanGone(t *testing.T) {
	store, address := detachedThreadStore(t)
	artifacts := NewFakeThreadArtifactCollisionChecker()
	artifacts.SetNativeBinding(address, "claude", sessioninventory.BindingEstablished, "native-root-1")
	artifacts.SessionPresenceHook = func([]ThreadAddress) error {
		return errors.New("zellij is not answering")
	}
	couch := &Couch{Threads: store, Artifacts: artifacts, Path: NewFakePathOps(nil)}

	rows, err := couch.ActionableThreadInventoryContext(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows = %+v, want one", rows)
	}
	if rows[0].State != ThreadUnusable || rows[0].Reason != ReasonUnknown {
		t.Fatalf("row = %+v, want unusable/unknown -- a failed query is not evidence of absence", rows[0])
	}
}

// The same rule one configuration down: a couch that cannot observe sessions at
// all has not learned that they are gone.
func TestACouchThatCannotObserveSessionsSaysUnknownRatherThanGone(t *testing.T) {
	store, address := detachedThreadStore(t)
	_ = address
	artifacts := bindingOnlyArtifacts{binding: NativeBindingResolution{
		Status: sessioninventory.BindingEstablished, NativeID: "native-root-1",
	}}
	couch := &Couch{Threads: store, Artifacts: artifacts, Path: NewFakePathOps(nil)}

	rows, err := couch.ActionableThreadInventoryContext(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].State != ThreadUnusable || rows[0].Reason != ReasonUnknown {
		t.Fatalf("rows = %+v, want one unusable/unknown", rows)
	}
}

// bindingOnlyArtifacts resolves bindings but cannot observe sessions. It
// implements ThreadArtifactController and NativeBindingResolver and NOTHING
// else -- embedding the fake would inherit DetachedSessions and defeat the
// point, which is the degraded shape the classifier must not mistake for an
// answer.
type bindingOnlyArtifacts struct {
	binding NativeBindingResolution
}

func (bindingOnlyArtifacts) Claim(ThreadAddress) (ThreadArtifactClaim, error) {
	return noopThreadArtifactClaim{}, nil
}
func (bindingOnlyArtifacts) Release(ThreadAddress) error { return nil }
func (bindingOnlyArtifacts) Registration(ThreadAddress) (RegistrationEvidence, error) {
	return RegistrationEstablished, nil
}
func (bindingOnlyArtifacts) Quiesce(ThreadAddress) error { return nil }
func (b bindingOnlyArtifacts) ResolveEstablished(context.Context, string, string, string) (NativeBindingResolution, error) {
	return b.binding, nil
}

// TestWarmRowsAskNoLedgerQuestion is the bound that keeps #256 M2's widening
// affordable, stated as the rule rather than as a count.
//
// The ledger read answers "is there a conversation to resume into?", which only
// a thread with no session needs asking. A row couch is hosting, and a row whose
// session outlived its launcher, both reattach onto something that is already
// there -- so neither may pay for a per-record file read on every refresh.
func TestWarmRowsAskNoLedgerQuestion(t *testing.T) {
	store, _ := newTestThreadStore(t)
	active := time.Unix(100, 0).UTC()

	hosted := actionableTestThread("couch-00000000000000f1", active)
	hosted.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	hosted.Incarnations = []ThreadIncarnation{{PID: 4242, Identity: "hosted", State: IncarnationLive}}
	if _, err := store.CreateThread(hosted); err != nil {
		t.Fatal(err)
	}
	detached := actionableTestThread("couch-00000000000000f2", active)
	detached.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	if _, err := store.CreateThread(detached); err != nil {
		t.Fatal(err)
	}

	artifacts := &countingArtifacts{FakeThreadArtifactCollisionChecker: NewFakeThreadArtifactCollisionChecker()}
	artifacts.SetSessionPresence(hosted.Address, SessionObservation{State: SessionPresent})
	artifacts.SetSessionPresence(detached.Address, SessionObservation{State: SessionPresent})
	proc := NewFakeProcOps()
	proc.Set(4242, "hosted")
	couch := &Couch{Threads: store, Artifacts: artifacts, Proc: proc, Path: NewFakePathOps(nil)}

	rows, err := couch.ActionableThreadInventoryContext(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	states := map[ThreadAddress]ActionableThreadState{}
	for _, row := range rows {
		states[row.Address] = row.State
	}
	if states[hosted.Address] != ThreadLive || states[detached.Address] != ThreadDetached {
		t.Fatalf("fixture did not produce a live and a detached row: %+v", states)
	}
	if artifacts.resolveCalls != 0 {
		t.Fatalf("the ledger was read %d times for warm rows; neither needs a cold-resume proof", artifacts.resolveCalls)
	}
}

// TestSessionAbsentWithResolvableLedgerIsResumable is the safety half of #256
// M2, end to end through the production gather path.
//
// The thread was never parked -- no receipt, no ParkHistory -- and its session
// is gone. Its conversation is still recorded in `ledger-<tag>.jsonl`, which is
// the only place a native conversation id ever lives. Before this, the ledger
// was read only for records carrying a VerifiedPark, so this row read
// `session-gone` and the switcher offered to archive a resumable conversation
// without a word about what would be lost.
func TestSessionAbsentWithResolvableLedgerIsResumable(t *testing.T) {
	store, _ := newTestThreadStore(t)
	record := actionableTestThread("couch-00000000000000f5", time.Unix(100, 0).UTC())
	record.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	created, err := store.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	if created.VerifiedPark != nil {
		t.Fatal("fixture carries a park receipt; it must not, or it proves the old rule")
	}

	artifacts := NewFakeThreadArtifactCollisionChecker()
	artifacts.SetSessionPresence(created.Address, SessionObservation{State: SessionAbsent})
	artifacts.SetNativeBinding(created.Address, "claude", sessioninventory.BindingEstablished, "native-root-1")
	couch := &Couch{Threads: store, Artifacts: artifacts, Path: NewFakePathOps(nil)}

	rows, err := couch.ActionableThreadInventoryContext(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows = %+v, want one", rows)
	}
	if rows[0].State != ThreadParked {
		t.Fatalf("= %q/%q, want parked: the ledger names a conversation to resume into",
			rows[0].State, rows[0].Reason)
	}
}
