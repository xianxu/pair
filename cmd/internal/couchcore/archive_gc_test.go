package couchcore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/storagegc"
)

func archiveDetachFixture(t *testing.T) (*ThreadStore, *storagegc.Coordinator, ArchiveDetachRequest) {
	t.Helper()
	s, c := retentionStore(t)
	record := archivableThread(t, s, "couch-0000000000000001")
	if err := s.ArchiveThread(record.Address); err != nil {
		t.Fatal(err)
	}
	a, err := readTestArchiveGrace(s, record.Address)
	if err != nil {
		t.Fatal(err)
	}
	return s, c, ArchiveDetachRequest{OperationID: "root-operation", Address: a.Address, RecordHash: a.RecordHash, ArchivedAt: a.ArchivedAt}
}
func TestArchiveDetachReceiptSurvivesEveryCrashStep(t *testing.T) {
	for fault := -1; fault < 3; fault++ {
		t.Run(fmt.Sprint(fault), func(t *testing.T) {
			s, c, req := archiveDetachFixture(t)
			crash := errors.New("crash")
			if fault < 0 {
				s.hooks.AfterJournal = func() error { return crash }
			} else {
				s.hooks.AfterTarget = func(i int) error {
					if i == fault {
						return crash
					}
					return nil
				}
			}
			err := c.WithLock(context.Background(), func(l *storagegc.Locked) error { return s.DetachArchive(l, req) })
			if !errors.Is(err, crash) {
				t.Fatalf("fault not reached %v", err)
			}
			s.hooks = threadStoreHooks{}
			if err := c.WithLock(context.Background(), func(l *storagegc.Locked) error { return s.DetachArchive(l, req) }); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(s.archivePath(req.Address)); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("archive not detached")
			}
			if _, err := os.Stat(s.archiveGracePath(req.Address)); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("grace not detached")
			}
			if _, err := os.Stat(s.archiveReceiptPath(req.OperationID)); err != nil {
				t.Fatal("root receipt missing")
			}
			// Old operation replay must not touch a newer archive incarnation.
			if err := writeAtomicBytes(s.archivePath(req.Address), []byte("new incarnation")); err != nil {
				t.Fatal(err)
			}
			if err := c.WithLock(context.Background(), func(l *storagegc.Locked) error {
				if err := s.DetachArchive(l, req); err != nil {
					return err
				}
				return s.ForgetArchiveReceipt(l, req)
			}); err != nil {
				t.Fatal(err)
			}
			raw, err := os.ReadFile(s.archivePath(req.Address))
			if err != nil || string(raw) != "new incarnation" {
				t.Fatal("receipt replay touched replacement")
			}
		})
	}
}
func TestArchiveDetachRefusesVisibleOrChangedArchive(t *testing.T) {
	for _, kind := range []string{"visible", "hash", "time", "operation"} {
		t.Run(kind, func(t *testing.T) {
			s, c, req := archiveDetachFixture(t)
			switch kind {
			case "visible":
				archivableThread(t, s, req.Address.Tag)
			case "hash":
				req.RecordHash = string(make([]byte, 64))
			case "time":
				req.ArchivedAt = req.ArchivedAt.Add(time.Second)
			case "operation":
				req.OperationID = "../escape"
			}
			if err := c.WithLock(context.Background(), func(l *storagegc.Locked) error { return s.DetachArchive(l, req) }); err == nil {
				t.Fatal("unsafe detach accepted")
			}
			if _, err := os.Stat(s.archivePath(req.Address)); err != nil {
				t.Fatal("archive changed on refusal")
			}
			if _, err := os.Stat(s.journalPath()); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("refusal published recovery journal")
			}
		})
	}
}

func TestArchiveDetachRejectsReadTokenAndConflictingReceipt(t *testing.T) {
	s, c, req := archiveDetachFixture(t)
	if err := c.WithReadLock(context.Background(), func(l *storagegc.Locked) error {
		if err := s.DetachArchive(l, req); err == nil {
			t.Fatal("read-only token permitted archive mutation")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := c.WithLock(context.Background(), func(l *storagegc.Locked) error { return s.DetachArchive(l, req) }); err != nil {
		t.Fatal(err)
	}
	changed := req
	changed.ArchivedAt = changed.ArchivedAt.Add(time.Second)
	if err := c.WithLock(context.Background(), func(l *storagegc.Locked) error { return s.DetachArchive(l, changed) }); err == nil {
		t.Fatal("receipt reused for different identity")
	}
	if err := c.WithLock(context.Background(), func(l *storagegc.Locked) error { return s.ForgetArchiveReceipt(l, changed) }); err == nil {
		t.Fatal("different identity forgot receipt")
	}
}
func TestArchiveDetachReceiptCleanupRecoversEachStep(t *testing.T) {
	for _, afterJournal := range []bool{false, true} {
		t.Run(fmt.Sprint(afterJournal), func(t *testing.T) {
			s, c, req := archiveDetachFixture(t)
			if err := c.WithLock(context.Background(), func(l *storagegc.Locked) error { return s.DetachArchive(l, req) }); err != nil {
				t.Fatal(err)
			}
			crash := errors.New("crash")
			if afterJournal {
				s.hooks.AfterJournal = func() error { return crash }
			} else {
				s.hooks.AfterTarget = func(int) error { return crash }
			}
			if err := c.WithLock(context.Background(), func(l *storagegc.Locked) error { return s.ForgetArchiveReceipt(l, req) }); !errors.Is(err, crash) {
				t.Fatalf("fault %v", err)
			}
			s.hooks = threadStoreHooks{}
			if err := c.WithLock(context.Background(), func(l *storagegc.Locked) error { return s.ForgetArchiveReceipt(l, req) }); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(s.archiveReceiptPath(req.OperationID)); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("receipt cleanup did not recover")
			}
		})
	}
}
