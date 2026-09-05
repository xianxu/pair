package couchcore

import (
	"context"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"
)

type detachFixture struct {
	couch    *Couch
	store    *ThreadStore
	proc     *FakeProcOps
	artifact *FakeThreadArtifactCollisionChecker
	address  ThreadAddress
	identity ProcessIdentity
	revision uint64
	// hooks lets a test open the read-then-CAS window that detach's retry loop
	// exists for. Installed after construction, so the fixture stays quiet
	// unless a test asks.
	hooks *threadStoreHooks
}

func newDetachFixture(t *testing.T) *detachFixture {
	t.Helper()
	ns := testCouchNamespace(t)
	hooks := &threadStoreHooks{}
	store := newThreadStoreWithHooks(ns, threadStoreHooks{
		AfterGetThread: func(a ThreadAddress) error {
			if hooks.AfterGetThread == nil {
				return nil
			}
			return hooks.AfterGetThread(a)
		},
	})
	address := ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-0001020304050607"}
	identity := ProcessIdentity{PID: 4242, Identity: "start-token"}

	seed := validThreadRecord(t)
	seed.Address, seed.StartingPath, seed.WorkingPath = address, ns.Dir(), ns.Dir()
	record, err := store.CreateThread(seed)
	if err != nil {
		t.Fatal(err)
	}
	record, err = store.UpdateExistingThread(address, record.Revision, func(next *ThreadRecord) error {
		next.Reservation = false
		next.Incarnations = []ThreadIncarnation{{
			State: IncarnationLive, PID: identity.PID, Identity: identity.Identity, StartedAt: time.Unix(10, 0).UTC(),
		}}
		next.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	proc := NewFakeProcOps()
	proc.Set(identity.PID, identity.Identity)
	proc.DiesOn = map[int]os.Signal{identity.PID: syscall.SIGTERM}
	artifact := NewFakeThreadArtifactCollisionChecker()
	artifact.SetPairSession(address, "pair-one", true)

	return &detachFixture{
		couch: &Couch{
			Namespace: ns, Threads: store, Proc: proc, Artifacts: artifact,
			Clock: FixedClock{T: time.Unix(100, 0).UTC()}, sleep: func(time.Duration) {},
		},
		store: store, proc: proc, artifact: artifact, hooks: hooks,
		address: address, identity: identity, revision: record.Revision,
	}
}

func TestCouchDetach(t *testing.T) {
	t.Run("retires the incarnation and leaves the session alone", func(t *testing.T) {
		f := newDetachFixture(t)

		if _, err := f.couch.Detach(context.Background(), f.address); err != nil {
			t.Fatalf("Detach() = %v", err)
		}

		record, err := f.store.GetThread(f.address)
		if err != nil {
			t.Fatal(err)
		}
		if len(record.Incarnations) != 0 {
			t.Fatalf("incarnations = %+v, want none", record.Incarnations)
		}
		if record.VerifiedPark != nil {
			t.Fatal("detach wrote a verified park -- it is not a park")
		}
		if record.LatestLaunchProfile == nil {
			t.Fatal("detach lost the launch profile reattach needs")
		}
		// The entire difference from park, asserted rather than assumed.
		if got := len(f.artifact.Quiesces()); got != 0 {
			t.Fatalf("detach quiesced the session %d times -- the session must survive", got)
		}
		if got := f.artifact.TriggeredQuits(); len(got) != 0 {
			t.Fatalf("detach triggered a Pair quit: %+v", got)
		}
	})

	t.Run("signals the process GROUP and never escalates to SIGKILL", func(t *testing.T) {
		f := newDetachFixture(t)
		if _, err := f.couch.Detach(context.Background(), f.address); err != nil {
			t.Fatal(err)
		}

		group := f.proc.GroupSignals[f.identity.PID]
		if len(group) == 0 {
			t.Fatal("detach signalled the pid alone -- the sidecars share the actor's group and would be orphaned")
		}
		for _, sig := range f.proc.Signals[f.identity.PID] {
			if sig == syscall.SIGKILL {
				t.Fatal("detach escalated to SIGKILL -- an agent mid-write is worse to truncate than to leave running")
			}
		}
	})

	t.Run("a client that will not exit leaves the thread live", func(t *testing.T) {
		f := newDetachFixture(t)
		f.proc.DiesOn = nil // ignores SIGTERM

		_, err := f.couch.Detach(context.Background(), f.address)
		if err == nil {
			t.Fatal("Detach() succeeded while the client was still alive")
		}
		record, readErr := f.store.GetThread(f.address)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if len(record.Incarnations) != 1 {
			t.Fatalf("a failed detach retired the incarnation anyway: %+v", record.Incarnations)
		}
	})

	t.Run("a vanished session is not a detach", func(t *testing.T) {
		f := newDetachFixture(t)
		f.artifact.SetPairSession(f.address, "pair-one", false)

		_, err := f.couch.Detach(context.Background(), f.address)
		if err == nil || !strings.Contains(err.Error(), "session") {
			t.Fatalf("Detach() = %v, want a refusal naming the session", err)
		}
		record, readErr := f.store.GetThread(f.address)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if len(record.Incarnations) != 1 {
			t.Fatalf("a vanished session still retired the incarnation: %+v", record.Incarnations)
		}
	})

	t.Run("a thread with no live incarnation refuses", func(t *testing.T) {
		f := newDetachFixture(t)
		if _, err := f.store.RetireIncarnation(f.address, f.revision, f.identity, time.Unix(1, 0)); err != nil {
			t.Fatal(err)
		}
		if _, err := f.couch.Detach(context.Background(), f.address); err == nil {
			t.Fatal("Detach() succeeded on an already-detached thread")
		}
	})

	t.Run("a canceled context stops before any signal", func(t *testing.T) {
		f := newDetachFixture(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		if _, err := f.couch.Detach(ctx, f.address); err == nil {
			t.Fatal("Detach() ignored a canceled context")
		}
		if len(f.proc.Signals[f.identity.PID]) != 0 {
			t.Fatal("detach signalled despite a canceled context")
		}
	})
}

// The record's revision is read before a SIGTERM, a bounded wait and two zellij
// observations. Anything that writes in that window used to abandon a thread
// whose client was already dead -- leaving the stale-IncarnationLive state
// pair#171 describes, reached from an ordinary failure path rather than a crash.
// Named for what it asserts, not for what it looks like it asserts. It bumps the
// revision BEFORE detach's loop begins, and the loop re-reads on every attempt,
// so the bump is absorbed by the first read and the CAS succeeds — the retry
// branch is never reached. Confirmed: replacing that branch's `continue` with a
// panic leaves this test green.
//
// It was called ...RetriesARevisionConflict..., which is why the branch went
// untested for as long as it did: the suite appeared to cover it.
// TestDetachRecordsOneActivityTimeHoweverManyAttemptsItTakes is the one that
// actually enters it, via threadStoreHooks.AfterGetThread.
func TestCouchDetachAbsorbsAConcurrentWriteBeforeItsLoop(t *testing.T) {
	f := newDetachFixture(t)

	// Move the revision while detach is between its observation and its CAS.
	bumped := false
	f.artifact.BeforePairSession = func(ThreadAddress) error {
		if bumped {
			return nil
		}
		bumped = true
		record, err := f.store.GetThread(f.address)
		if err != nil {
			return err
		}
		_, err = f.store.UpdateExistingThread(f.address, record.Revision, func(next *ThreadRecord) error {
			next.Description = "edited mid-detach"
			return nil
		})
		return err
	}

	if _, err := f.couch.Detach(context.Background(), f.address); err != nil {
		t.Fatalf("Detach() = %v -- a concurrent write must not abandon a dead client", err)
	}
	record, err := f.store.GetThread(f.address)
	if err != nil {
		t.Fatal(err)
	}
	if len(record.Incarnations) != 0 {
		t.Fatalf("incarnations = %+v, want the retire to have retried through the conflict", record.Incarnations)
	}
	if record.Description != "edited mid-detach" {
		t.Fatalf("the concurrent edit was lost: %q", record.Description)
	}
}

// A detached thread had no recorded activity at all -- LastActiveAt was written
// only by park -- so the switcher rendered its age from the zero time and stated
// `detached · 106751d ago`, which is MaxInt64 nanoseconds rather than a duration
// anyone computed (pair#187).
func TestDetachRecordsWhenTheThreadWasLastActive(t *testing.T) {
	f := newDetachFixture(t)
	before, err := f.store.GetThread(f.address)
	if err != nil {
		t.Fatal(err)
	}
	if !before.LastActiveAt.IsZero() {
		t.Fatalf("fixture already has LastActiveAt %v; the test cannot show detach setting it", before.LastActiveAt)
	}

	if _, err := f.couch.Detach(context.Background(), f.address); err != nil {
		t.Fatalf("Detach() = %v", err)
	}

	after, err := f.store.GetThread(f.address)
	if err != nil {
		t.Fatal(err)
	}
	if after.LastActiveAt.IsZero() {
		t.Fatal("detach left LastActiveAt unset, so the row has no age to render and will state a saturated one")
	}
	if want := time.Unix(100, 0).UTC(); !after.LastActiveAt.Equal(want) {
		t.Errorf("LastActiveAt = %v, want the detach clock's %v", after.LastActiveAt, want)
	}
}

// MonotonicLastActiveAt is on the detach path and nothing pinned it: reverting
// the fold to a bare assignment left the whole couchcore suite green. Reachable
// for real — park at T1, relaunch, detach under a backward wall clock — and the
// consequence is a reduced recency that mis-ranks the row in SelectResumableRoot.
func TestDetachNeverMovesTheActivityTimeBackwards(t *testing.T) {
	f := newDetachFixture(t)
	// Later than the fixture's clock (Unix 100), as a park would have left it.
	const parked = 500
	if _, err := f.store.UpdateExistingThread(f.address, f.revision, func(next *ThreadRecord) error {
		next.LastActiveAt = time.Unix(parked, 0).UTC()
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := f.couch.Detach(context.Background(), f.address); err != nil {
		t.Fatalf("Detach() = %v", err)
	}

	record, err := f.store.GetThread(f.address)
	if err != nil {
		t.Fatal(err)
	}
	if want := time.Unix(parked, 0).UTC(); !record.LastActiveAt.Equal(want) {
		t.Errorf("LastActiveAt = %v, want %v -- a backward clock must not reduce recorded recency",
			record.LastActiveAt, want)
	}
}

// The read-once invariant, now that the retry branch is reachable.
//
// This test was written once before and DELETED because it could not fail:
// nothing could force a retry, so the loop ran a single time and both clock
// placements agreed. The close review's answer was better than my conclusion --
// not "the branch is untestable" but "add the seam, here" -- so
// threadStoreHooks.AfterGetThread now opens the read-then-CAS window that is the
// only moment a conflict can be injected.
func TestDetachRecordsOneActivityTimeHoweverManyAttemptsItTakes(t *testing.T) {
	f := newDetachFixture(t)
	first := time.Unix(1_000, 0).UTC()
	f.couch.Clock = &sequenceClock{times: []time.Time{
		first,
		time.Unix(2_000, 0).UTC(),
		time.Unix(3_000, 0).UTC(),
	}}

	// Bump the revision in the window between the LOOP's read and its CAS.
	//
	// Which read matters, and getting it wrong is why the first two attempts at
	// this test were vacuous. Detach reads the record twice: once for its
	// preconditions (detach.go:64) and once per loop attempt (detach.go:118).
	// Bumping on the first is absorbed -- the loop re-reads and its CAS
	// succeeds, so no retry happens and the test passes while proving nothing.
	// Only a bump after the loop's own read opens a window it must retry
	// through.
	reads := 0
	bumped := false
	f.hooks.AfterGetThread = func(address ThreadAddress) error {
		reads++
		if reads < 2 || bumped {
			return nil
		}
		bumped = true
		record, err := f.store.GetThread(address)
		if err != nil {
			return err
		}
		_, err = f.store.UpdateExistingThread(address, record.Revision, func(next *ThreadRecord) error {
			next.Description = "edited between the read and the CAS"
			return nil
		})
		return err
	}

	if _, err := f.couch.Detach(context.Background(), f.address); err != nil {
		t.Fatalf("Detach() = %v -- the retry must absorb a conflict, not fail", err)
	}
	if !bumped {
		t.Fatal("the conflict never fired, so no retry happened and this proves nothing")
	}
	record, err := f.store.GetThread(f.address)
	if err != nil {
		t.Fatal(err)
	}
	if !record.LastActiveAt.Equal(first) {
		t.Errorf("LastActiveAt = %v, want the FIRST clock reading %v -- a later value means the clock was read inside the retry loop",
			record.LastActiveAt, first)
	}
}
