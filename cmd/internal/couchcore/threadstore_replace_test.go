package couchcore

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"
	"time"
)

// replaceFixture is a real main store holding one named, retirable :0 record
// and the fresh record that would replace it at the same path.
func replaceFixture(t *testing.T) (*ThreadStore, ThreadRecord, ThreadRecord) {
	t.Helper()
	store, _ := newTestThreadStore(t)
	old := archivableThread(t, store, "couch-0000000000000001")
	named := "durable name"
	described := "durable description"
	old, err := store.ApplyThreadMetadata(old.Address, old.Revision, ThreadMetadataPatch{Name: &named, Description: &described})
	if err != nil {
		t.Fatal(err)
	}
	next := actionableTestThread("couch-0000000000000002", time.Unix(200, 0).UTC())
	next.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	return store, old, next
}

// assertReplaced is the one outcome a replace may have: the old record is in
// the archive byte for byte with its grace clock, the new one is in the working
// set, and the manifest names the new address and not the old.
func assertReplaced(t *testing.T, store *ThreadStore, old ThreadRecord, oldRaw []byte, next ThreadRecord) {
	t.Helper()
	if _, err := store.GetThread(next.Address); err != nil {
		t.Fatalf("new record not published: %v", err)
	}
	if _, err := store.GetThread(old.Address); !errors.Is(err, ErrThreadNotFound) {
		t.Fatalf("old record still in the working set: %v", err)
	}
	archived, err := os.ReadFile(store.archivePath(old.Address))
	if err != nil || !bytes.Equal(archived, oldRaw) {
		t.Fatalf("archive bytes differ from the old record (err %v)", err)
	}
	if _, err := os.Stat(store.archiveGracePath(old.Address)); err != nil {
		t.Fatalf("archive grace clock missing: %v", err)
	}
	snapshot, err := store.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Records) != 1 || snapshot.Records[0].Address != next.Address {
		t.Fatalf("manifest lists %+v, want only %v", snapshot.Records, next.Address)
	}
	history, err := store.ArchivedThreads()
	if err != nil || len(history) != 1 || history[0].Address != old.Address ||
		history[0].Name != "durable name" || history[0].Description != "durable description" {
		t.Fatalf("archive history = %+v, %v", history, err)
	}
}

func TestReplaceThreadExpectedArchivesAndCreatesInOneJournal(t *testing.T) {
	store, old, next := replaceFixture(t)
	oldRaw, err := os.ReadFile(store.recordPath(old.Address))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.ReplaceThreadExpected(old.Address, old.Revision, next); err != nil {
		t.Fatalf("ReplaceThreadExpected: %v", err)
	}
	assertReplaced(t, store, old, oldRaw, next)
}

// A crash after the journal is durable is completed by replay -- the same
// recovery the slot store's fresh replacement relies on -- so there is never a
// moment where the path holds both records or neither.
func TestReplaceThreadExpectedSurvivesACrashAfterTheJournal(t *testing.T) {
	store, old, next := replaceFixture(t)
	oldRaw, err := os.ReadFile(store.recordPath(old.Address))
	if err != nil {
		t.Fatal(err)
	}
	store.hooks.AfterJournal = func() error { return errors.New("interrupted") }
	if err := store.ReplaceThreadExpected(old.Address, old.Revision, next); err == nil {
		t.Fatal("the interrupted replace reported success")
	}
	store.hooks = threadStoreHooks{}
	if err := store.RecoverStoreJournal(); err != nil {
		t.Fatalf("RecoverStoreJournal: %v", err)
	}
	assertReplaced(t, store, old, oldRaw, next)
}

func assertNothingReplaced(t *testing.T, store *ThreadStore, old ThreadRecord, next ThreadRecord) {
	t.Helper()
	if _, err := os.Stat(store.archivePath(old.Address)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a refused replace wrote an archive: %v", err)
	}
	if _, err := store.GetThread(next.Address); err == nil {
		t.Fatal("a refused replace published the new record")
	}
	if _, err := os.Stat(store.journalPath()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a refused replace left a journal: %v", err)
	}
}

func TestReplaceThreadExpectedRefusesAStaleRevision(t *testing.T) {
	store, old, next := replaceFixture(t)
	summary := "moved on"
	if _, err := store.ApplyThreadMetadata(old.Address, old.Revision, ThreadMetadataPatch{PublishedSummary: &summary}); err != nil {
		t.Fatal(err)
	}
	var revision *ThreadRevisionError
	if err := store.ReplaceThreadExpected(old.Address, old.Revision, next); !errors.As(err, &revision) {
		t.Fatalf("stale revision err = %v, want *ThreadRevisionError", err)
	}
	assertNothingReplaced(t, store, old, next)
	if _, err := store.GetThread(old.Address); err != nil {
		t.Fatalf("refused replace disturbed the old record: %v", err)
	}
}

func TestReplaceThreadExpectedRefusesAnOpenParkOrStartClaim(t *testing.T) {
	store, old, next := replaceFixture(t)
	claimed, err := store.CommitStartClaim(old.Address, old.Revision, "repo", time.Unix(300, 0).UTC(), StartEvent{
		Kind: StartClaimed, Nonce: "start-0123456789abcdef", Shape: StartFreshExisting,
		Owner:   SupervisorOwner{PID: 77, Identity: "owner-couch"},
		Profile: old.LatestLaunchProfile,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.ReplaceThreadExpected(claimed.Address, claimed.Revision, next); err == nil {
		t.Fatal("replaced through an outstanding start claim")
	}
	assertNothingReplaced(t, store, old, next)
}

func TestReplaceThreadExpectedRefusesLocalLayout(t *testing.T) {
	local := testLocalThreadStore(t)
	r := recordAtCheckout(t, local.slot.WorktreeRoot, local.slot.WorktreeRoot, "couch-0123456789abcdef")
	created, err := local.CreateThread(r)
	if err != nil {
		t.Fatal(err)
	}
	next := r
	next.Address.Tag = "couch-1111111111111111"
	if err := local.ReplaceThreadExpected(created.Address, created.Revision, next); err == nil {
		t.Fatal("a slot store accepted the main store's replace; slots replace through replaceSlotCurrent")
	}
	if _, err := local.GetThread(created.Address); err != nil {
		t.Fatalf("refused replace disturbed the slot record: %v", err)
	}
}

// The :0 replace must never publish into a slot store: next routes exactly as
// CreateThread would route it, and a path inside an enrolled slot checkout
// lands on a different backend, which replace refuses.
func TestReplaceThreadExpectedRoutesNextLikeCreateThread(t *testing.T) {
	s, repository, _ := migrationFixture(t)
	if err := s.EnrollSlotRepository(context.Background(), repository); err != nil {
		t.Fatal(err)
	}
	old := archivableThread(t, s, "couch-0000000000000001")
	next := recordAtCheckout(t, repository.Slots[0].Identity.WorktreeRoot, repository.Slots[0].Identity.WorktreeRoot, "couch-0000000000000002")
	if err := s.ReplaceThreadExpected(old.Address, old.Revision, next); err == nil {
		t.Fatal("replace published a fresh record into a slot store")
	}
	assertNothingReplaced(t, s, old, next)
	if _, err := s.GetThread(old.Address); err != nil {
		t.Fatalf("refused replace disturbed the old record: %v", err)
	}
}
