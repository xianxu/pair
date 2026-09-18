package couchcore

import (
	"context"
	"testing"
	"time"
)

// startClaimFixture is a record mid-start: one creating incarnation carrying a
// ThreadStartClaim, owned by pid 4242. It is the shape a couch writes between
// claiming a start and the launcher registering, and the shape it LEAVES behind
// if it dies in that window.
func startClaimFixture(t *testing.T, store *ThreadStore, tag ThreadTag, ownerPID int) ThreadRecord {
	t.Helper()
	record := actionableTestThread(tag, time.Unix(100, 0).UTC())
	record.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	record.Incarnations = []ThreadIncarnation{{
		State: IncarnationCreating,
		Start: &ThreadStartClaim{
			Nonce: "start-0123456789abcdef", OwnerPID: ownerPID, OwnerIdentity: "supervisor",
		},
	}}
	created, err := store.CreateThread(record)
	if err != nil {
		t.Fatalf("the start-claim fixture is not a shape the store accepts: %v", err)
	}
	return created
}

// TestStartOwnerLivenessReachesTheClassifier is the gather half of Task 4.
//
// `startClaimed` is the one piece of bookkeeping #256 left in the classification
// path, justified because a ThreadStartClaim "is recoverable on its own terms".
// Those terms are {OwnerPID, OwnerIdentity}. Until this, nothing read them, so a
// couch that died mid-start left a row reading `starting...` with no timeout, no
// owner check and no way out -- the #271 wedge one level down.
func TestStartOwnerLivenessReachesTheClassifier(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(*FakeProcOps)
		want  Liveness
	}{
		{
			name:  "owner still running",
			setup: func(proc *FakeProcOps) { proc.Set(4242, "supervisor") },
			want:  Live,
		},
		{
			name:  "owner gone",
			setup: func(proc *FakeProcOps) {},
			want:  Dead,
		},
		{
			name:  "pid recycled by another process",
			setup: func(proc *FakeProcOps) { proc.Set(4242, "someone-else") },
			want:  Dead,
		},
		{
			name:  "owner could not be probed",
			setup: func(proc *FakeProcOps) { proc.SetUnknown(4242) },
			want:  Unknown,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store, _ := newTestThreadStore(t)
			created := startClaimFixture(t, store, "couch-00000000000000c1", 4242)
			proc := NewFakeProcOps()
			tc.setup(proc)

			couch := &Couch{
				Threads: store, Artifacts: NewFakeThreadArtifactCollisionChecker(),
				Proc: proc, Path: NewFakePathOps(nil),
			}
			_, evidence, err := couch.gatherThreadEvidence(context.Background(), nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			if got := evidence[created.Address].StartOwner; got != tc.want {
				t.Fatalf("StartOwner = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestStartClaimDomainsCoincideAndTheGapFailsClosed pins the seam between two
// readings of the same fact.
//
// startInFlight asks startClaimed -- true when ANY incarnation carries a Start
// -- and then reads evidence the gather produced through
// CurrentStartTransaction, which resolves only when EXACTLY one does. If those
// domains ever came apart, a record startClaimed calls busy would carry no
// StartOwner at all.
//
// They do not come apart, and the reason is the validator, not the reader: it
// refuses more than one tracked start and refuses a Start outside `creating`.
// That claim is pinned HERE, through the real store, rather than trusted from a
// reading of validateLifecycle -- the M1 boundary review twice found an
// "unrepresentable" claim that was false because an exception went unread.
//
// The gap is closed twice over: even if a shape got through, Liveness's zero
// value is Unknown, so an unwritten StartOwner keeps the row busy rather than
// releasing it to archive.
func TestStartClaimDomainsCoincideAndTheGapFailsClosed(t *testing.T) {
	if Liveness(0) != Unknown {
		t.Fatalf("Liveness zero value = %v; ThreadEvidence.StartOwner relies on it being Unknown", Liveness(0))
	}

	claim := func(nonce string) *ThreadStartClaim {
		return &ThreadStartClaim{Nonce: nonce, OwnerPID: 4242, OwnerIdentity: "supervisor"}
	}
	for _, tc := range []struct {
		name         string
		incarnations []ThreadIncarnation
	}{
		{
			name: "two tracked starts",
			incarnations: []ThreadIncarnation{
				{State: IncarnationCreating, Start: claim("start-0123456789abcdef")},
				{State: IncarnationCreating, Start: claim("start-fedcba9876543210")},
			},
		},
		{
			name: "a start claim on a live incarnation",
			incarnations: []ThreadIncarnation{
				{PID: 77, Identity: "tok", State: IncarnationLive, Start: claim("start-0123456789abcdef")},
			},
		},
		{
			name: "a start claim on an unknown incarnation",
			incarnations: []ThreadIncarnation{
				{PID: 77, Identity: "tok", State: IncarnationUnknown, Start: claim("start-0123456789abcdef")},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store, _ := newTestThreadStore(t)
			record := actionableTestThread("couch-00000000000000c2", time.Unix(100, 0).UTC())
			record.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
			record.Incarnations = tc.incarnations

			created, err := store.CreateThread(record)
			if err != nil {
				// The store is the oracle: this shape is out of the domain, so
				// the two readings cannot disagree about it.
				return
			}
			// It got in, so the domains MUST agree about it.
			if !startClaimed(created) {
				t.Fatalf("the fixture stopped being a start claim")
			}
			if _, err := CurrentStartTransaction(created); err != nil {
				t.Fatalf("the store accepted a shape CurrentStartTransaction cannot resolve (%v); "+
					"startInFlight would read an unwritten StartOwner", err)
			}
		})
	}
}

// TestOwnStartInFlightIsNotReleased is the regression the escape must not cause.
// Between claiming a start and the launcher acquiring a pid there is no session
// and no live process, so without the busy branch a thread starting NORMALLY
// classifies `session-gone` -- archive-eligible. Probing the owner must not
// reopen that.
func TestOwnStartInFlightIsNotReleased(t *testing.T) {
	store, _ := newTestThreadStore(t)
	created := startClaimFixture(t, store, "couch-00000000000000c3", 4242)

	proc := NewFakeProcOps()
	proc.Set(4242, "supervisor")
	artifacts := NewFakeThreadArtifactCollisionChecker()
	artifacts.SetSessionPresence(created.Address, SessionObservation{State: SessionAbsent})

	couch := &Couch{Threads: store, Artifacts: artifacts, Proc: proc, Path: NewFakePathOps(nil)}
	snapshot, evidence, err := couch.gatherThreadEvidence(context.Background(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	state, reason := ClassifyThread(snapshot.Records[0], evidence[created.Address])
	if state != ThreadBusy {
		t.Fatalf("= (%q, %q), want busy: a start this couch is driving has no session yet", state, reason)
	}
}
