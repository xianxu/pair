package couchcore

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/pairlifecycle"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

func presenceAddress(tag ThreadTag) ThreadAddress {
	return ThreadAddress{RepoScope: "816fc349d3faebf8", Tag: tag}
}

// TestSessionPresenceIsThreeValued pins the distinction the old evidence could
// not express: "we could not ask" is not "there is no session".
//
// It matters because after #256 recoverability is keyed to the session, and
// `session-gone` is archive-eligible. Collapsing an unresolved question into
// absence would offer to retire a thread whose agent is still running.
func TestSessionPresenceIsThreeValued(t *testing.T) {
	live := presenceAddress("couch-0000000000000001")
	exited := presenceAddress("couch-0000000000000002")
	unlisted := presenceAddress("couch-0000000000000003")
	contestedA := presenceAddress("couch-0000000000000004")
	contestedB := presenceAddress("couch-0000000000000005")
	duplicated := presenceAddress("couch-0000000000000006")

	bindings := []SessionNameBinding{
		{Address: live, SessionName: "live-session"},
		{Address: exited, SessionName: "exited-session"},
		{Address: unlisted, SessionName: "unlisted-session"},
		// One name claimed by two addresses: couch cannot tell whose it is.
		{Address: contestedA, SessionName: "contested"},
		{Address: contestedB, SessionName: "contested"},
		{Address: duplicated, SessionName: "duplicated"},
	}
	sessions := []launcher.Session{
		{Name: "live-session", State: launcher.SessionLive},
		{Name: "exited-session", State: launcher.SessionExited},
		{Name: "contested", State: launcher.SessionLive},
		// Two rows for one name: the snapshot itself is contradictory.
		{Name: "duplicated", State: launcher.SessionLive},
		{Name: "duplicated", State: launcher.SessionExited},
	}

	got := ProjectSessionPresence(bindings, sessions, claimsFromSessionBindings(bindings))

	for _, tc := range []struct {
		name    string
		address ThreadAddress
		want    SessionState
	}{
		{"listed and not exited", live, SessionPresent},
		{"listed but exited", exited, SessionAbsent},
		{"bound to a name no session carries", unlisted, SessionAbsent},
		{"two addresses claim one name", contestedA, SessionUnresolved},
		{"two addresses claim one name (other side)", contestedB, SessionUnresolved},
		{"two snapshot rows share one name", duplicated, SessionUnresolved},
	} {
		if state := got[tc.address].State; state != tc.want {
			t.Errorf("%s: state = %v, want %v", tc.name, state, tc.want)
		}
	}

	if name := got[live].Name; name != "live-session" {
		t.Errorf("a present observation must carry its session name, got %q", name)
	}
}

// TestUnresolvedIsTheZeroValue is a guard, not a tautology: every consumer that
// forgets to populate an observation must fail CLOSED. If absence were the zero
// value, a gather branch that silently stopped running would assert "no session"
// for every thread it skipped -- which is the shape of the refusals #181 removed.
func TestUnresolvedIsTheZeroValue(t *testing.T) {
	var zero SessionObservation
	if zero.State != SessionUnresolved {
		t.Fatalf("zero value is %v; an unpopulated observation must not claim absence", zero.State)
	}
}

// TestProjectionAnswersOnlyForBindingsItWasGiven pins the division of labour:
// the PURE projector answers only for bindings handed to it, and the CALLER owns
// the distinction between "no row in a readable index" (asked, absent) and
// "scope unreadable" (never asked), because only the caller knows which
// happened. That caller-side branch is covered against the production checker in
// TestSessionPresenceAnswersThroughTheProductionChecker.
func TestProjectionAnswersOnlyForBindingsItWasGiven(t *testing.T) {
	address := presenceAddress("couch-0000000000000007")
	got := ProjectSessionPresence(nil, []launcher.Session{{Name: "something", State: launcher.SessionLive}}, nil)
	if state := got[address].State; state != SessionUnresolved {
		t.Fatalf("an address absent from the projection input reads %v; callers must supply its binding", state)
	}
}

// claimsFromSessionBindings mirrors what production passes: the claim count is
// derived from every binding the resolver READ. Here the test's binding list is
// the whole read, so the two coincide.
func claimsFromSessionBindings(bindings []SessionNameBinding) map[string]int {
	claims := map[string]int{}
	for _, binding := range bindings {
		if binding.SessionName != "" {
			claims[binding.SessionName]++
		}
	}
	return claims
}

// TestSessionEvidenceReachesEveryRecord is the gather half of Task 1, and the
// structural fix #256 turns on.
//
// gatherThreadEvidence gated physicalization, binding resolution AND the
// detached question behind `resumeShaped` -- no incarnation, no park, a saved
// profile. A record carrying an incarnation was therefore never asked about its
// session, which is precisely why a dead launcher could hide a running agent
// (#272): the evidence that would have saved the thread was never collected.
//
// The budget matters as much as the answer. One host-wide call, not one per
// record -- otherwise the fix trades a correctness bug for a startup that scales
// with the store.
func TestSessionEvidenceReachesEveryRecord(t *testing.T) {
	store, _ := newTestThreadStore(t)
	artifacts := NewFakeThreadArtifactCollisionChecker()

	// Three shapes the resume-shaped gate excluded, plus one it admitted.
	shapes := map[string]func(*ThreadRecord){
		"carries an incarnation": func(r *ThreadRecord) {
			r.Incarnations = []ThreadIncarnation{{PID: 4242, Identity: "tok", State: IncarnationLive}}
		},
		"park in flight": func(r *ThreadRecord) {
			// A park identity must match a recorded incarnation -- validation
			// enforces it, which is the same fact #256 rests on: the park's
			// process IS the incarnation's, not a second one to probe.
			r.Incarnations = []ThreadIncarnation{{PID: 4242, Identity: "tok", State: IncarnationLive}}
			r.Park = &ParkTransaction{
				Identity: ParkIdentity{
					Nonce: "park-0123456789abcdef", Address: r.Address,
					PID: 4242, ProcessIdentity: "tok",
				},
				BaseRevision: 1, RecordRevision: 2, Phase: ParkAwaitingCompletion,
				// The live wedged record's exact shape: attempt 1 timed out with
				// a timeout failure. Validation requires the marker and the
				// failure to agree, so a bare TimedOut would be refused.
				Attempts: []ParkAttempt{{
					Number: 1, TimedOut: true,
					Failure: &ParkFailure{
						Code:       pairlifecycle.FailureTimeout,
						Diagnostic: "matching completion was not observed",
					},
				}},
			}
		},
		"no launch profile": func(r *ThreadRecord) {},
		"resume shaped": func(r *ThreadRecord) {
			r.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
		},
	}

	tags := map[string]ThreadTag{
		"carries an incarnation": "couch-00000000000000a1",
		"park in flight":         "couch-00000000000000a2",
		"no launch profile":      "couch-00000000000000a3",
		"resume shaped":          "couch-00000000000000a4",
	}
	addresses := map[string]ThreadAddress{}
	for name, shape := range shapes {
		record := actionableTestThread(tags[name], time.Unix(100, 0).UTC())
		shape(&record)
		created, err := store.CreateThread(record)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		addresses[name] = created.Address
		artifacts.SetSessionPresence(created.Address, SessionObservation{
			State: SessionPresent, Name: "session-" + string(tags[name]),
		})
	}

	couch := &Couch{Threads: store, Artifacts: artifacts, Path: NewFakePathOps(nil)}
	_, evidence, err := couch.gatherThreadEvidence(context.Background(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	for name, address := range addresses {
		if got := evidence[address].Session.State; got != SessionPresent {
			t.Errorf("%s: session evidence = %v, want present — the gather branch did not reach this shape", name, got)
		}
	}
	if queries := artifacts.SessionPresenceQueries(); queries != 1 {
		t.Errorf("SessionPresence called %d times for %d records; it must be one host-wide question", queries, len(addresses))
	}
}

// TestSessionPresenceFailureLeavesEveryThreadUnresolved is the fail-closed half.
// A host couch could not question must not be reported as a host with no
// sessions -- `session-gone` is archive-eligible, so that collapse would offer
// to retire live work.
func TestSessionPresenceFailureLeavesEveryThreadUnresolved(t *testing.T) {
	store, _ := newTestThreadStore(t)
	record := actionableTestThread("couch-00000000000000b1", time.Unix(100, 0).UTC())
	created, err := store.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	artifacts := NewFakeThreadArtifactCollisionChecker()
	artifacts.SetSessionPresence(created.Address, SessionObservation{State: SessionPresent})
	artifacts.SessionPresenceHook = func([]ThreadAddress) error {
		return errors.New("zellij unavailable")
	}

	couch := &Couch{Threads: store, Artifacts: artifacts, Path: NewFakePathOps(nil)}
	_, evidence, err := couch.gatherThreadEvidence(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("a failed session probe must not fail the whole round: %v", err)
	}
	if got := evidence[created.Address].Session.State; got != SessionUnresolved {
		t.Fatalf("session state after a failed probe = %v, want unresolved", got)
	}
}

// wedgedParkFixture is the operator's live record shape: a park whose attempt
// timed out and whose process is long gone.
func wedgedParkFixture(address ThreadAddress) *ParkTransaction {
	return &ParkTransaction{
		Identity: ParkIdentity{
			Nonce: "park-0123456789abcdef", Address: address,
			PID: 64734, ProcessIdentity: "1789535173.46673",
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
}

// TestWedgedParkDoesNotWedgeClassification is pair#271, measured.
//
// Record couch-e1a31510b7033d08 rev 293: park phase awaiting_completion since
// 2026-09-15 22:53, pid 64734 confirmed gone (ESRCH). `if record.Park != nil {
// return ThreadBusy }` was total and unconditional and sat above every branch
// that consults evidence, so the row read `parking…` for ~18 hours while the
// switcher offered it exactly two actions: name and describe.
func TestWedgedParkDoesNotWedgeClassification(t *testing.T) {
	record := actionableTestThread("couch-e1a31510b7033d08", time.Unix(100, 0).UTC())
	// The live record's own profile: muse, which is what made its rows
	// unarchivable rather than merely unlabelled.
	record.LatestLaunchProfile = &LaunchProfile{Agent: "muse", Argv: []string{}}
	record.Incarnations = []ThreadIncarnation{{
		PID: 64734, Identity: "1789535173.46673", State: IncarnationLive,
	}}
	record.Park = wedgedParkFixture(record.Address)

	state, reason := ClassifyThread(record, ThreadEvidence{
		Session: SessionObservation{State: SessionAbsent},
	})

	if state == ThreadBusy {
		t.Fatal("an open park transaction still decides recoverability")
	}
	if state != ThreadUnusable || reason != ReasonSessionGone {
		t.Fatalf("got %v/%q, want unusable/session-gone: the park is gone and so is the session", state, reason)
	}
}

// TestDeadLauncherWithLiveSessionIsDetached is pair#272, measured.
//
// The launcher dies with couch; the zellij server is PPID 1 and does not. So a
// clean detach and a couch crash leave IDENTICAL external state, and the only
// thing that used to distinguish them was whether couch survived long enough to
// clear the incarnation. Both must classify the same.
func TestDeadLauncherWithLiveSessionIsDetached(t *testing.T) {
	crashed := actionableTestThread("couch-95293a9b6c0d459a", time.Unix(100, 0).UTC())
	crashed.LatestLaunchProfile = &LaunchProfile{Agent: "muse", Argv: []string{}}
	crashed.Incarnations = []ThreadIncarnation{{
		PID: 81935, Identity: "dead-launcher", State: IncarnationLive,
	}}

	// The same thread after a clean detach: the bookkeeping ran.
	detached := crashed
	detached.Incarnations = nil

	evidence := ThreadEvidence{Session: SessionObservation{
		State: SessionPresent, Name: "📁pair-couch-32",
	}}

	crashedState, crashedReason := ClassifyThread(crashed, evidence)
	detachedState, _ := ClassifyThread(detached, evidence)

	if crashedState != ThreadDetached {
		t.Fatalf("crashed = %v/%q, want detached — the agent is still running behind its session", crashedState, crashedReason)
	}
	if crashedState != detachedState {
		t.Fatalf("crash classifies %v but clean detach classifies %v; identical external state must classify identically", crashedState, detachedState)
	}
}

// TestAbsentLiveEvidenceProvesNothing is the rule stated directly. The Live
// union stays POSITIVE-ONLY: a thread couch does not host is not thereby dead.
func TestAbsentLiveEvidenceProvesNothing(t *testing.T) {
	record := actionableTestThread("couch-00000000000000c1", time.Unix(100, 0).UTC())
	record.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	record.Incarnations = []ThreadIncarnation{{PID: 9999, Identity: "gone", State: IncarnationLive}}

	if state, reason := ClassifyThread(record, ThreadEvidence{
		Session: SessionObservation{State: SessionUnresolved},
	}); state != ThreadUnusable || reason != ReasonUnknown {
		t.Fatalf("got %v/%q, want unusable/unknown — we could not ask, so nothing is settled", state, reason)
	}
}

// TestUnknownLivenessNeverRetiresAnIncarnation is the fail-closed half of
// re-adoption, and it was unpinned: the M1 boundary review mutation-checked
// `!= Dead` into `== Live` and NOTHING failed across three packages.
//
// The guard's own comment says retiring an incarnation whose process is alive
// "would abandon a running agent", and #256's Done-when states it directly --
// Unknown observations cannot authorize destructive recovery. A rule that
// expensive needs a test that fails when it is inverted.
func TestUnknownLivenessNeverRetiresAnIncarnation(t *testing.T) {
	for _, tc := range []struct {
		name       string
		liveness   func(*FakeProcOps, int)
		wantRetire bool
	}{
		// An unset pid is Dead to FakeProcOps -- the honest "no such process".
		{"dead is the only proof that retires", func(p *FakeProcOps, pid int) {}, true},
		{"unknown must not retire", func(p *FakeProcOps, pid int) { p.SetUnknown(pid) }, false},
		{"live must not retire", func(p *FakeProcOps, pid int) { p.Set(pid, "tok") }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store, _ := newTestThreadStore(t)
			record := actionableTestThread("couch-00000000000000d1", time.Unix(100, 0).UTC())
			record.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
			record.Incarnations = []ThreadIncarnation{{PID: 4242, Identity: "tok", State: IncarnationLive}}
			created, err := store.CreateThread(record)
			if err != nil {
				t.Fatal(err)
			}
			proc := NewFakeProcOps()
			tc.liveness(proc, 4242)

			couch := &Couch{Threads: store, Proc: proc}
			retired, err := couch.retireDeadIncarnationBeforeStart(created)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantRetire != (retired != nil) {
				t.Fatalf("retired=%v, want %v", retired != nil, tc.wantRetire)
			}
			current, err := store.GetThread(created.Address)
			if err != nil {
				t.Fatal(err)
			}
			if got := len(current.Incarnations); tc.wantRetire == (got == 1) {
				t.Fatalf("durable incarnations = %d with wantRetire=%v; an unprovable process must leave the record untouched", got, tc.wantRetire)
			}
		})
	}
}

// TestParkedRowSurvivesAnUnresolvedSessionQuestion pins I1 from the M1 review.
//
// A parked thread's resume authority is DURABLE -- a verified park plus a
// resolvable native id -- and does not depend on the session existing, because
// park tore that session down on purpose. Refusing it because one
// `list-sessions` failed would demote every parked row in the store at once.
func TestParkedRowSurvivesAnUnresolvedSessionQuestion(t *testing.T) {
	record := actionableTestThread("couch-00000000000000d2", time.Unix(100, 0).UTC())
	record.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	markActionableParked(&record, record.LastActiveAt)
	proof := []ParkedResumeObservation{{Address: record.Address, Agent: "claude", NativeID: "native-1"}}

	for _, tc := range []struct {
		name    string
		session SessionState
	}{
		{"session answered absent", SessionAbsent},
		{"session question failed", SessionUnresolved},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state, reason := ClassifyThread(record, ThreadEvidence{
				Parked: proof, ParkedStatus: ProofResolved,
				Session: SessionObservation{State: tc.session},
			})
			if state != ThreadParked {
				t.Fatalf("got %v/%q, want parked — cold-resume authority does not depend on the session", state, reason)
			}
		})
	}
}

// TestOrphanedParkDoesNotWedgeTheResumeChain is C1 from the M1 boundary review:
// a regression this milestone INTRODUCED, and the fourth site of its own class.
//
// Re-adoption made a park-open record reachable for the first time, which made
// RetireIncarnation's open-park precondition live. The chain was: classify →
// detached, SelectResumableRoot ranks detached highest, DecideResume permits,
// and then the store refuses with a raw error carrying no ResumeDiagnosticCode --
// so startup, which only decorates coded refusals, wedged `couch` in that tree
// entirely. Before M1 the record classified `busy` and startup spawned normally.
//
// This is #271's own live fixture: couch-e1a31510b7033d08 sat park-open for ~18
// hours across restarts while its zellij session survived, because
// `zellij delete-session` never ran.
func TestOrphanedParkDoesNotWedgeTheResumeChain(t *testing.T) {
	store, _ := newTestThreadStore(t)
	record := actionableTestThread("couch-e1a31510b7033d08", time.Unix(100, 0).UTC())
	record.LatestLaunchProfile = &LaunchProfile{Agent: "muse", Argv: []string{}}
	record.Incarnations = []ThreadIncarnation{{PID: 64734, Identity: "1789535173.46673", State: IncarnationLive}}
	record.Park = wedgedParkFixture(record.Address)
	record.Park.Identity.PID = 64734
	record.Park.Identity.ProcessIdentity = "1789535173.46673"
	created, err := store.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	// Advance the record past the park's own RecordRevision, which is what a
	// park that has been sitting open across restarts looks like: AbandonPark
	// writes at expectedRevision+1 and refuses to reuse a revision the
	// transaction already claimed.
	name := "brain"
	bumped, err := store.ApplyThreadMetadata(created.Address, created.Revision, ThreadMetadataPatch{Name: &name})
	if err != nil {
		t.Fatal(err)
	}
	created = bumped

	// The park's owner is the incarnation's process -- park copies it -- so the
	// one probe that proves the incarnation dead proves the park orphaned too.
	couch := &Couch{Threads: store, Proc: NewFakeProcOps()}
	retired, err := couch.retireDeadIncarnationBeforeStart(created)
	if err != nil {
		t.Fatalf("an orphaned park still blocks re-adoption: %v", err)
	}
	if retired == nil {
		t.Fatal("nothing was retired; the park-open record stayed occupied")
	}
	if retired.Park != nil {
		t.Fatal("the orphaned park survived; RetireIncarnation would refuse the next attempt too")
	}
	if len(retired.Incarnations) != 0 {
		t.Fatalf("incarnations = %d, want none — CommitStartClaim refuses a record that still has one", len(retired.Incarnations))
	}
}

// TestResumeFailuresCarryADiagnosticCode is the other half of C1, and the more
// general rule: startup decorates a refusal only when it carries a
// ResumeDiagnosticCode (startup.go). A bare store error therefore reaches the
// operator as an internal message with no next step, and refuses the whole tree.
func TestResumeFailuresCarryADiagnosticCode(t *testing.T) {
	store, _ := newTestThreadStore(t)
	record := actionableTestThread("couch-00000000000000d3", time.Unix(100, 0).UTC())
	record.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	record.Incarnations = []ThreadIncarnation{{PID: 4242, Identity: "tok", State: IncarnationLive}}
	record.Park = wedgedParkFixture(record.Address)
	// A park owned by a DIFFERENT process than the incarnation: this probe
	// cannot prove it dead, so the resume must refuse -- but legibly.
	record.Park.Identity.PID = 4242
	record.Park.Identity.ProcessIdentity = "tok"
	created, err := store.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpdateExistingThread(created.Address, created.Revision, func(next *ThreadRecord) error {
		next.Park.Attempts = append(next.Park.Attempts, ParkAttempt{Number: 2})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	stale, err := store.GetThread(created.Address)
	if err != nil {
		t.Fatal(err)
	}
	// Stale revision: AbandonPark will refuse on the CAS.
	stale.Revision = created.Revision

	couch := &Couch{Threads: store, Proc: NewFakeProcOps()}
	_, err = couch.retireDeadIncarnationBeforeStart(stale)
	if err == nil {
		t.Fatal("expected a refusal")
	}
	if ResumeDiagnosticOf(err) == "" {
		t.Fatalf("refusal carries no diagnostic code: %v — startup cannot decorate it, so it wedges the tree", err)
	}
}
