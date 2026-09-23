package couchcore

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestEnrolledSlotPublicLifecycleUsesLocalAuthority(t *testing.T) {
	s, repository, r := migrationFixture(t)
	if err := s.EnrollSlotRepository(context.Background(), repository); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetThread(r.Address)
	if err != nil {
		t.Fatalf("public lookup: %v", err)
	}
	changed, err := s.updateExistingThread(r.Address, got.Revision, func(r *ThreadRecord) error { r.Name = "slot metadata"; return nil })
	if err != nil {
		t.Fatal(err)
	}
	snap, err := s.Snapshot()
	if err != nil || len(snap.Records) != 1 || snap.Records[0].Name != changed.Name {
		t.Fatalf("snapshot=%+v %v", snap, err)
	}
	if _, err := os.Stat(s.recordPath(r.Address)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("global slot record recreated: %v", err)
	}
	if err := s.ArchiveThread(r.Address); err != nil {
		t.Fatal(err)
	}
	history, err := s.ArchivedThreads()
	if err != nil || len(history) != 1 || history[0].Name != changed.Name {
		t.Fatalf("history=%+v %v", history, err)
	}
	if err := s.RestoreThread(r.Address); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetThread(r.Address); err != nil {
		t.Fatal(err)
	}
}

func TestEnrolledSlotArchiveListingIgnoresStaleGlobalCopy(t *testing.T) {
	s, repository, r := migrationFixture(t)
	if err := s.EnrollSlotRepository(context.Background(), repository); err != nil {
		t.Fatal(err)
	}
	if err := s.ArchiveThread(r.Address); err != nil {
		t.Fatal(err)
	}
	local := newSlotThreadStore(s.namespace, repository.Slots[0].Identity)
	raw, err := os.ReadFile(local.archivePath(r.Address))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(s.archivePath(r.Address)), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(s.archivePath(r.Address), raw, 0600); err != nil {
		t.Fatal(err)
	}
	listed, err := s.ArchivedThreads()
	if err != nil || len(listed) != 1 {
		t.Fatalf("archives=%+v err=%v", listed, err)
	}
}
