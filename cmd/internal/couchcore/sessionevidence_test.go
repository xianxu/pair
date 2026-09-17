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

// TestAbsentBindingIsAbsentNotUnresolved keeps the two IO failures apart: a
// thread with no row in the index HAS been asked about and has no session, while
// a scope whose index could not be read has not been asked at all.
func TestAbsentBindingIsAbsentNotUnresolved(t *testing.T) {
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
