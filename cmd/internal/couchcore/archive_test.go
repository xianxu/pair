package couchcore

import (
	"context"
	"errors"
	"testing"
	"time"
)

func archivableThread(t *testing.T, store *ThreadStore, tag ThreadTag) ThreadRecord {
	t.Helper()
	record := actionableTestThread(tag, time.Unix(100, 0).UTC())
	record.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	created, err := store.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	return created
}

// Archiving is the operator's "delete": get this out of my switcher. It has to
// be complete -- gone from the manifest, so gone from every projection -- and
// reversible, because "start anew" should not mean "destroy the evidence".
func TestArchiveThreadRemovesItFromTheWorkingSetAndKeepsTheRecord(t *testing.T) {
	store, _ := newTestThreadStore(t)
	kept := archivableThread(t, store, "couch-0000000000000001")
	retired := archivableThread(t, store, "couch-0000000000000002")

	if err := store.ArchiveThread(retired.Address); err != nil {
		t.Fatalf("ArchiveThread: %v", err)
	}

	snapshot, err := store.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Records) != 1 || snapshot.Records[0].Address != kept.Address {
		t.Fatalf("working set = %+v, want only the kept thread", snapshot.Records)
	}
	if _, err := store.GetThread(retired.Address); err == nil {
		t.Fatal("an archived thread is still readable in the working set")
	}

	archived, err := store.ArchivedThreads()
	if err != nil {
		t.Fatal(err)
	}
	if len(archived) != 1 || archived[0].Address != retired.Address {
		t.Fatalf("archive = %+v, want the retired thread", archived)
	}
	// Same shape as it had: restoring is a file move plus a manifest re-add,
	// not a reconstruction.
	if archived[0].WorkingPath != retired.WorkingPath || archived[0].LatestLaunchProfile == nil {
		t.Fatalf("archived record lost detail: %+v", archived[0])
	}
}

// Archiving a thread couch is hosting would leave the console owning a record
// the store no longer lists -- a stale incarnation manufactured on purpose.
func TestArchiveThreadRefusesALiveOrParkingThread(t *testing.T) {
	store, _ := newTestThreadStore(t)

	live := archivableThread(t, store, "couch-0000000000000001")
	updated, err := store.UpdateExistingThread(live.Address, live.Revision, func(record *ThreadRecord) error {
		record.Incarnations = []ThreadIncarnation{{PID: 42, Identity: "pair-live", State: IncarnationLive}}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	live = updated
	if err := store.ArchiveThread(live.Address); err == nil {
		t.Fatal("archived a live thread")
	}
	if _, err := store.GetThread(live.Address); err != nil {
		t.Fatalf("refused archive still removed the record: %v", err)
	}
}

// An unusable thread is exactly what the operator wants gone, so no reason
// blocks the move -- the whole point is that debris leaves on their say-so.
func TestArchiveThreadAcceptsAnUnusableThread(t *testing.T) {
	store, _ := newTestThreadStore(t)
	broken := archivableThread(t, store, "couch-0000000000000001")
	if err := store.ArchiveThread(broken.Address); err != nil {
		t.Fatalf("ArchiveThread on debris: %v", err)
	}
}

func TestArchiveThreadRefusesAnUnknownAddress(t *testing.T) {
	store, _ := newTestThreadStore(t)
	err := store.ArchiveThread(ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-0000000000000009"})
	if err == nil {
		t.Fatal("archived a thread that does not exist")
	}
}

// The whole point of a complete delete: the session stops. A record filed while
// its agent keeps running is the forgotten thread couch exists to prevent, and
// the one-time cleanup produced exactly one of those before this existed.
func TestCouchArchiveStopsTheSessionBeforeMovingTheRecord(t *testing.T) {
	store, _ := newTestThreadStore(t)
	thread := archivableThread(t, store, "couch-0000000000000001")
	artifacts := NewFakeThreadArtifactCollisionChecker()
	artifacts.SetPairSession(thread.Address, "pair-"+string(thread.Address.Tag), true)
	couch := &Couch{Threads: store, Artifacts: artifacts, Path: NewFakePathOps(nil)}

	if _, err := couch.ArchiveThread(context.Background(), thread.Address); err != nil {
		t.Fatalf("ArchiveThread: %v", err)
	}
	if got := artifacts.Quiesces(); len(got) != 1 || got[0] != thread.Address {
		t.Fatalf("quiesced %+v, want exactly the archived thread", got)
	}
	archived, err := store.ArchivedThreads()
	if err != nil || len(archived) != 1 {
		t.Fatalf("archive = %+v, %v", archived, err)
	}
}

// Order is the property, not just the pair of effects. Archiving first and
// stopping second would leave a record in the archive with a live session
// behind it -- the exact state this action removes.
func TestCouchArchiveRefusesWhenTheSessionCannotBeStopped(t *testing.T) {
	store, _ := newTestThreadStore(t)
	thread := archivableThread(t, store, "couch-0000000000000001")
	artifacts := NewFakeThreadArtifactCollisionChecker()
	artifacts.QuiesceHook = func(ThreadAddress) error { return errors.New("zellij is not answering") }
	couch := &Couch{Threads: store, Artifacts: artifacts, Path: NewFakePathOps(nil)}

	if _, err := couch.ArchiveThread(context.Background(), thread.Address); err == nil {
		t.Fatal("archived a thread whose session could not be stopped")
	}
	if _, err := store.GetThread(thread.Address); err != nil {
		t.Fatalf("a refused archive still moved the record: %v", err)
	}
	archived, _ := store.ArchivedThreads()
	if len(archived) != 0 {
		t.Fatalf("archive = %+v, want nothing", archived)
	}
}
