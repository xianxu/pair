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
}

func newDetachFixture(t *testing.T) *detachFixture {
	t.Helper()
	ns := testCouchNamespace(t)
	store := NewThreadStore(ns)
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
		store: store, proc: proc, artifact: artifact,
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
		if _, err := f.store.RetireIncarnation(f.address, f.revision, f.identity); err != nil {
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
func TestCouchDetachRetriesARevisionConflictAfterTeardown(t *testing.T) {
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
