package couchcore

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

// TestReAdoptionExitsAreTotalAndCoded is the RULE the M1 review's round 2 asked
// for, replacing two ad-hoc tests that between them reached one of three exits.
//
// Every combination of {park shape} x {process liveness} x {incarnation state}
// must either retire cleanly or refuse with a NON-EMPTY ResumeDiagnosticCode,
// and must never leave the record half-changed. It fails whenever a new exit is
// added, which is what the two hand-written tests could not do -- the round-1
// fix coded the store's retire error, and round 2 reproduced the identical wedge
// through a different uncoded exit.
//
// It also carries both round-2 Criticals as regressions:
//   - unobservable launcher + surviving session must not wedge startup (BR-11)
//   - a failed retirement must not have already destroyed the park (BR-12)
func TestReAdoptionExitsAreTotalAndCoded(t *testing.T) {
	type liveness int
	const (
		dead liveness = iota
		unknown
		alive
	)
	// "foreign" is covered separately by TestForeignOwnedParkIsRepresentableAndRefused,
	// which builds that record through the real store -- it IS representable via
	// validateLifecycle's replacement_incarnation exception, contrary to an
	// earlier note here that read only the main clause.
	for _, park := range []string{"none", "matching"} {
		for _, live := range []liveness{dead, unknown, alive} {
			for _, state := range []IncarnationState{IncarnationLive, IncarnationUnknown} {
				name := park + "-park/" + map[liveness]string{dead: "dead", unknown: "unknown", alive: "alive"}[live] + "/" + string(state)
				t.Run(name, func(t *testing.T) {
					store, _ := newTestThreadStore(t)
					record := actionableTestThread("couch-00000000000000e1", time.Unix(100, 0).UTC())
					record.LatestLaunchProfile = &LaunchProfile{Agent: "muse", Argv: []string{}}
					record.Incarnations = []ThreadIncarnation{{PID: 64734, Identity: "tok", State: state}}
					if park != "none" {
						record.Park = wedgedParkFixture(record.Address)
						record.Park.Identity.PID = 64734
						record.Park.Identity.ProcessIdentity = "tok"
					}
					created, err := store.CreateThread(record)
					if err != nil {
						t.Fatal(err)
					}
					if park != "none" {
						// Advance past the park's own RecordRevision so an
						// abandon could legitimately land; otherwise the CAS
						// would refuse for an unrelated reason.
						label := "brain"
						if created, err = store.ApplyThreadMetadata(created.Address, created.Revision, ThreadMetadataPatch{Name: &label}); err != nil {
							t.Fatal(err)
						}
					}

					proc := NewFakeProcOps()
					switch live {
					case unknown:
						proc.SetUnknown(64734)
					case alive:
						proc.Set(64734, "tok")
					}

					couch := &Couch{Threads: store, Proc: proc}
					retired, err := couch.retireDeadIncarnationBeforeStart(created)

					// The expected outcome per cell, not merely
					// self-consistency: retirement is destructive, so exactly
					// which combinations MAY perform it is the rule. Only a
					// confirmed-dead process and a live recorded incarnation --
					// everything else must refuse, and refuse legibly.
					wantRetire := live == dead && state == IncarnationLive
					if wantRetire && err != nil {
						t.Fatalf("a dead process with a live incarnation must retire, got %v", err)
					}
					if !wantRetire && err == nil {
						t.Fatalf("retired on %s: an unprovable process or a non-live incarnation must refuse, or CommitStartClaim refuses later with a bare store error", name)
					}

					after, readErr := store.GetThread(created.Address)
					if readErr != nil {
						t.Fatal(readErr)
					}
					if err != nil {
						if ResumeDiagnosticOf(err) == "" {
							t.Fatalf("refusal carries no diagnostic code: %v — startup cannot decorate it, so it wedges the whole tree", err)
						}
						// BR-12: a refusal must not have already destroyed the
						// park. AbandonPark's tombstone is permanent, so a
						// failure after it leaves a thread that can be neither
						// resumed nor archived.
						if created.Park != nil && after.Park == nil {
							t.Fatalf("the park was abandoned and then the retirement refused (%v); the tombstone is permanent, so this thread is now unrecoverable", err)
						}
						return
					}
					if retired == nil {
						// Declining to act is only honest when there was
						// nothing to act on.
						if len(after.Incarnations) != 0 {
							t.Fatalf("no refusal and no retirement, but the record still carries %d incarnation(s) — CommitStartClaim will refuse with a bare store error", len(after.Incarnations))
						}
						return
					}
					if len(after.Incarnations) != 0 {
						t.Fatalf("reported retirement but %d incarnation(s) remain", len(after.Incarnations))
					}
					if after.Park != nil {
						t.Fatal("reported retirement but the park survives; the next CommitStartClaim refuses")
					}
				})
			}
		}
	}
}

// TestUnobservableLauncherWithLiveSessionRefusesLegibly is BR-11 named directly,
// because the table above proves the property while this names the incident.
//
// A record whose launcher cannot be observed plus a surviving session classifies
// `detached`, is ranked highest by SelectResumableRoot and auto-selected at
// startup. The Dead-only retire gate correctly declines it — and before this fix
// CommitStartClaim then refused with a bare `already has 1 incarnation(s)`,
// which startup cannot decorate, so `couch` refused to start in that tree at all.
// The same fixture spawns normally at the milestone's base commit.
func TestUnobservableLauncherWithLiveSessionRefusesLegibly(t *testing.T) {
	store, _ := newTestThreadStore(t)
	record := actionableTestThread("couch-00000000000000e2", time.Unix(100, 0).UTC())
	record.LatestLaunchProfile = &LaunchProfile{Agent: "muse", Argv: []string{}}
	record.Incarnations = []ThreadIncarnation{{PID: 4242, Identity: "tok", State: IncarnationLive}}
	created, err := store.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	proc := NewFakeProcOps()
	proc.SetUnknown(4242)

	couch := &Couch{Threads: store, Proc: proc}
	_, err = couch.retireDeadIncarnationBeforeStart(created)
	if err == nil {
		t.Fatal("an unobservable launcher was silently accepted; CommitStartClaim will refuse next, uncoded")
	}
	if ResumeDiagnosticOf(err) == "" {
		t.Fatalf("refusal carries no diagnostic code: %v", err)
	}
}

// TestEveryStartupResumeFailureIsActionable pins BR-11's rule where it belongs:
// at the CONSUMER that needed a marker, not on every producer.
//
// startupResumeRefusal used to decorate only errors carrying a
// ResumeDiagnosticCode, so an internal failure -- a store CAS, a repo-identity
// lookup -- reached the operator as a raw message with no next step and refused
// `couch` in the whole tree. Forcing every producer to carry a code "fixed" that
// by changing what the code MEANS, which broke the readers that used the
// distinction. The guidance does not depend on the code, so it is given
// unconditionally and the code goes back to meaning one thing.
func TestEveryStartupResumeFailureIsActionable(t *testing.T) {
	address := ThreadAddress{RepoScope: "816fc349d3faebf8", Tag: "couch-00000000000000f1"}
	for _, tc := range []struct {
		name string
		err  error
	}{
		{"a structured refusal", refuseResume(ResumeNotDetached, "not warm")},
		{"a bare internal error", errors.New("thread already has 1 incarnation(s)")},
		{"a wrapped internal error", fmt.Errorf("commit: %w", errors.New("revision conflict"))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			decorated := startupResumeRefusal(address, tc.err)
			if decorated == nil {
				t.Fatal("a startup failure must never be dropped")
			}
			for _, want := range []string{"couch --show", "pair", "will not start a second"} {
				if !strings.Contains(decorated.Error(), want) {
					t.Fatalf("refusal is not actionable — missing %q:\n%s", want, decorated)
				}
			}
			// The original must stay reachable: errors.Is/As and Unwrap are how
			// callers distinguish a deadline from a refusal, and rebuilding the
			// error from its string silently broke that.
			if !errors.Is(decorated, tc.err) {
				t.Fatalf("decoration hid the original error: %v", decorated)
			}
		})
	}
}

// TestResumeCodeStillMeansAStructuredRefusal is the other half: the code must
// NOT be applied to internal failures, or every reader that branches on it --
// the background reattach pass renders the error's first line only when the code
// is empty -- starts showing "resume-unknown" instead of what went wrong.
func TestResumeCodeStillMeansAStructuredRefusal(t *testing.T) {
	if got := ResumeDiagnosticOf(errors.New("thread already has 1 incarnation(s)")); got != "" {
		t.Fatalf("an internal error reports code %q; callers use an empty code to mean \"render the message\"", got)
	}
	if got := ResumeDiagnosticOf(refuseResume(ResumeNotDetached, "not warm")); got != ResumeNotDetached {
		t.Fatalf("a structured refusal lost its code: %q", got)
	}
}

// TestForeignOwnedParkIsRepresentableAndRefused is the test the "unrepresentable"
// claim should have had before the guard was deleted.
//
// The claim read `validateLifecycle`'s main clause -- "active park identity
// matches %d incarnations" -- and missed its exception one line above:
// `threadrecord/lifecycle.go` permits ZERO matches when the phase is `unknown`
// and the transaction carries a `replacement_incarnation` failure, which
// `park.go` produces. So the record is representable, and this test builds it
// THROUGH THE REAL STORE to prove it rather than asserting it.
//
// What that made possible: probing the incarnation and then tombstoning the
// park is a permanent, irreversible claim about a process nothing looked at --
// one that may be alive and mid-park. Its own FinalizePark then fails with
// "park abandon identity does not match active transaction", so the park never
// completes and #275's audit trail for it is gone.
func TestForeignOwnedParkIsRepresentableAndRefused(t *testing.T) {
	store, _ := newTestThreadStore(t)
	record := actionableTestThread("couch-00000000000000f9", time.Unix(100, 0).UTC())
	record.LatestLaunchProfile = &LaunchProfile{Agent: "muse", Argv: []string{}}
	// The live incarnation is the REPLACEMENT; the park belongs to the original.
	record.Incarnations = []ThreadIncarnation{{PID: 99, Identity: "replacement", State: IncarnationLive}}
	record.Park = &ParkTransaction{
		Identity: ParkIdentity{
			Nonce: "park-0123456789abcdef", Address: record.Address,
			PID: 42, ProcessIdentity: "original-owner",
		},
		BaseRevision: 1, RecordRevision: 2, Phase: ParkUnknown,
		Attempts: []ParkAttempt{{
			Number: 1,
			Failure: &ParkFailure{
				Code:       pairlifecycle.FailureReplacementIncarnation,
				Diagnostic: "a replacement incarnation appeared",
			},
		}},
	}

	created, err := store.CreateThread(record)
	if err != nil {
		t.Fatalf("the store REFUSED the fixture, so the unrepresentability claim would have held: %v", err)
	}
	label := "brain"
	if created, err = store.ApplyThreadMetadata(created.Address, created.Revision, ThreadMetadataPatch{Name: &label}); err != nil {
		t.Fatal(err)
	}

	// The incarnation's process is dead; the PARK's owner is alive.
	proc := NewFakeProcOps()
	proc.Set(42, "original-owner")

	couch := &Couch{Threads: store, Proc: proc}
	_, err = couch.retireDeadIncarnationBeforeStart(created)
	if err == nil {
		t.Fatal("abandoned a park whose owner is alive and was never probed")
	}
	if ResumeDiagnosticOf(err) == "" {
		t.Fatalf("refusal carries no diagnostic code: %v", err)
	}
	after, readErr := store.GetThread(created.Address)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if after.Park == nil {
		t.Fatal("the live owner's park was tombstoned; its FinalizePark can now never match, and the audit trail is gone")
	}
}
