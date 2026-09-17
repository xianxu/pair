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
		artifacts.SetSessionPresence(created.Address, SessionObservation{State: SessionPresent})
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
		// Asked and answered: no conversation in the ledger either. Since #256
		// M2 that question is put to every session-less record, so leaving it
		// unresolved here would make this assert `unknown` for a reason that
		// has nothing to do with #271.
		ParkedStatus: ProofResolved,
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

	evidence := ThreadEvidence{Session: SessionObservation{State: SessionPresent}}

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

// TestReAdoptionExitsAreTotalAndCoded is the RULE, replacing tests written one
// per incident.
//
// Every shape `validateLifecycle` ACCEPTS must either be cleared or refused with
// a non-empty ResumeDiagnosticCode, and must never leave the record
// half-changed. The dimensions are the ones whose absence let a shape through:
//
//   - park presence — round 4: a foreign-owned park was tombstoned on the
//     strength of a probe of a different process
//   - incarnation COUNT — round 5: an open park with zero incarnations was never
//     cleared, so CommitStartClaim refused uncoded and couch would not start
//   - process liveness and incarnation state — the destructive-act guards
//   - the START CLAIM and its OWNER — #256 M2: a claim is a THIRD process, the
//     couch that began the transaction, and it outlives neither the helper nor
//     the park owner reliably. Folded into the shape dimension because the
//     validator ties a claim to `creating` and forbids it anywhere else.
//
// It fails whenever a new exit appears, which is what the hand-written tests it
// replaced could not do.
func TestReAdoptionExitsAreTotalAndCoded(t *testing.T) {
	type liveness int
	const (
		dead liveness = iota
		unknown
		alive
	)
	names := map[liveness]string{dead: "dead", unknown: "unknown", alive: "alive"}

	// The dimensions are DELIBERATELY wider than the shapes anyone enumerated,
	// and `CreateThread`'s refusal is the oracle that trims them back. Writing
	// the bounds by hand is the same mistake as writing the sweep as a list of
	// sites: it covers the shapes the author thought of. Two of this function's
	// defects were shapes nobody had -- a foreign-owned park, then an open park
	// with zero incarnations -- so the domain must come from `validateLifecycle`,
	// not from here. A new exit without a cell now fails this test.
	// shape folds the incarnation's state together with the start claim it may
	// carry, because validateLifecycle ties the two: a claim is legal only on a
	// `creating` incarnation, and `creating` is legal only with a claim. Keeping
	// them as independent dimensions would generate only cells the oracle skips.
	type shape struct {
		name  string
		state IncarnationState
		// claimOwner is the liveness of the couch that made the start claim, or
		// nil when this shape carries no claim. It is a SEPARATE process from
		// the incarnation, which is the whole reason it needs its own column.
		claimOwner *liveness
	}
	ownerDead, ownerUnknown, ownerAlive := dead, unknown, alive
	shapes := []shape{
		{name: "live", state: IncarnationLive},
		{name: "unknown-state", state: IncarnationUnknown},
		{name: "creating-no-claim", state: IncarnationCreating},
		{name: "claim/owner-dead", state: IncarnationCreating, claimOwner: &ownerDead},
		{name: "claim/owner-unknown", state: IncarnationCreating, claimOwner: &ownerUnknown},
		{name: "claim/owner-alive", state: IncarnationCreating, claimOwner: &ownerAlive},
	}
	const claimOwnerPID = 4242

	for _, park := range []string{"none", "matching", "foreign"} {
		for _, count := range []int{0, 1, 2} {
			for _, live := range []liveness{dead, unknown, alive} {
				for _, sh := range shapes {
					state := sh.state
					if count == 0 && (live != dead || state != IncarnationLive) {
						continue // liveness/state are meaningless with no incarnation
					}
					if park == "none" && count == 0 {
						continue // nothing in the way; the function declines by design
					}
					name := fmt.Sprintf("%s-park/%d-incarnation/%s/%s", park, count, names[live], sh.name)
					t.Run(name, func(t *testing.T) {
						store, _ := newTestThreadStore(t)
						record := actionableTestThread("couch-00000000000000e1", time.Unix(100, 0).UTC())
						record.LatestLaunchProfile = &LaunchProfile{Agent: "muse", Argv: []string{}}
						for i := 0; i < count; i++ {
							incarnation := ThreadIncarnation{
								PID: 64734 + i, Identity: fmt.Sprintf("tok-%d", i), State: state,
							}
							if sh.claimOwner != nil {
								incarnation.Start = &ThreadStartClaim{
									Nonce:    fmt.Sprintf("start-0123456789abcde%d", i),
									OwnerPID: claimOwnerPID, OwnerIdentity: "supervisor",
								}
							}
							record.Incarnations = append(record.Incarnations, incarnation)
						}
						if count == 1 {
							record.Incarnations[0].Identity = "tok"
						}
						if park != "none" {
							record.Park = wedgedParkFixture(record.Address)
							record.Park.Identity.PID = 64734
							record.Park.Identity.ProcessIdentity = "tok"
							if park == "foreign" {
								// Owned by a process that is NOT an incarnation.
								// Valid only through the replacementUnknown
								// escape, which the oracle below enforces.
								record.Park.Identity.PID = 42
								record.Park.Identity.ProcessIdentity = "original-owner"
							}
							if count == 0 || park == "foreign" {
								// The replacementUnknown escape: zero matches are
								// valid when the phase is `unknown` and the
								// transaction carries a replacement_incarnation
								// failure. This is the shape round 5 found.
								record.Park.Phase = ParkUnknown
								record.Park.Attempts = []ParkAttempt{{
									Number: 1,
									Failure: &ParkFailure{
										Code:       pairlifecycle.FailureReplacementIncarnation,
										Diagnostic: "a replacement incarnation appeared",
									},
								}}
							}
						}
						created, err := store.CreateThread(record)
						if err != nil {
							t.Skipf("validateLifecycle refuses this shape, so it is outside the domain: %v", err)
						}
						if park != "none" {
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
						if sh.claimOwner != nil {
							switch *sh.claimOwner {
							case unknown:
								proc.SetUnknown(claimOwnerPID)
							case alive:
								proc.Set(claimOwnerPID, "supervisor")
							}
						}

						couch := &Couch{Threads: store, Proc: proc}
						retired, err := couch.clearLifecycleDebris(created)

						// Which cells MAY clear: the process (and, when a park is
						// present, its owner -- the same pid in this fixture)
						// proved dead, and a live recorded incarnation if there
						// is one at all.
						// A foreign park's owner is a different process, and this
						// fixture leaves it unset -- i.e. dead -- so it clears on
						// the same terms. Two incarnations never clear: the
						// function refuses rather than guessing which is current.
						// A start claim is cleared on its OWN owner's death plus
						// the forked process's -- two probes, two processes --
						// and rolls back rather than retiring, because neither
						// retirement transition takes an incarnation with a
						// start still open.
						//
						// A `live` and an `unknown` incarnation both clear, by
						// two DIFFERENT transitions: the unproven one is retired
						// only because the screen above proved this exact
						// {PID, identity} dead, which is the evidence detach
						// does not have (#256 M3). `creating` still refuses --
						// it names a start nothing here is driving.
						wantClear := live == dead && count <= 1
						if count == 1 {
							if sh.claimOwner != nil {
								wantClear = wantClear && *sh.claimOwner == dead
							} else {
								wantClear = wantClear && (state == IncarnationLive || state == IncarnationUnknown)
							}
						}
						if wantClear && err != nil {
							t.Fatalf("a clearable shape refused: %v", err)
						}
						if !wantClear && err == nil {
							t.Fatalf("cleared %s: an unprovable process or an unretirable incarnation must refuse, or a later store guard refuses uncoded", name)
						}

						after, readErr := store.GetThread(created.Address)
						if readErr != nil {
							t.Fatal(readErr)
						}
						if err != nil {
							if ResumeDiagnosticOf(err) == "" {
								t.Fatalf("refusal carries no diagnostic code: %v — startup cannot decorate it, so it wedges the whole tree", err)
							}
							if created.Park != nil && after.Park == nil {
								t.Fatalf("the park was abandoned and then the call refused (%v); the tombstone is permanent, so this thread is now unrecoverable", err)
							}
							return
						}
						if retired == nil {
							t.Fatalf("no refusal and no result for %s; the record still carries %d incarnation(s) and park=%v", name, len(after.Incarnations), after.Park != nil)
						}
						if len(after.Incarnations) != 0 {
							t.Fatalf("reported clearance but %d incarnation(s) remain — CommitStartClaim refuses those", len(after.Incarnations))
						}
						if after.Park != nil {
							t.Fatal("reported clearance but the park survives — CommitStartClaim refuses that too, uncoded")
						}
					})
				}
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
	_, err = couch.clearLifecycleDebris(created)
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
	_, err = couch.clearLifecycleDebris(created)
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
