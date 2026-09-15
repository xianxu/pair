package couchcore

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/storagegc"
)

func TestStorePublicationKilledPublisher(t *testing.T) {
	for _, phase := range []string{"journal", "archive", "grace", "receipt"} {
		t.Run(phase, func(t *testing.T) {
			s, c := retentionStore(t)
			record := archivableThread(t, s, "couch-0000000000000001")
			if phase == "receipt" {
				if err := s.ArchiveThread(record.Address); err != nil {
					t.Fatal(err)
				}
			}
			target := s.journalPath()
			switch phase {
			case "archive":
				target = s.archivePath(record.Address)
			case "grace":
				target = s.archiveGracePath(record.Address)
			case "receipt":
				target = s.archiveReceiptPath("killed-publication")
			}
			command := exec.Command(os.Args[0], "-test.run=^TestStorePublicationChild$")
			command.Env = append(os.Environ(), "PAIR_STORE_PUBLICATION_CHILD=1", "PAIR_STORE_PUBLICATION_NAMESPACE="+s.namespace.Dir(), "PAIR_STORE_PUBLICATION_ROOT="+c.Root, "PAIR_STORE_PUBLICATION_PHASE="+phase)
			stdout, err := command.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			var stderr bytes.Buffer
			command.Stderr = &stderr
			if err := command.Start(); err != nil {
				t.Fatal(err)
			}
			killed := false
			defer func() {
				if !killed {
					_ = command.Process.Kill()
					_ = command.Wait()
				}
			}()
			ready := make(chan string, 1)
			go func() { line, _ := bufio.NewReader(stdout).ReadString('\n'); ready <- line }()
			select {
			case line := <-ready:
				if line != "staged\n" {
					_ = command.Process.Kill()
					_ = command.Wait()
					killed = true
					t.Fatalf("child did not stage: %q %s", line, stderr.String())
				}
			case <-time.After(10 * time.Second):
				t.Fatal("publisher did not reach stage barrier")
			}
			if _, err := os.Stat(target); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("target had authority before rename: %s %v", target, err)
			}
			if err := command.Process.Kill(); err != nil {
				t.Fatal(err)
			}
			_ = command.Wait()
			killed = true
			if _, err := os.Stat(s.publicationPath()); err != nil {
				t.Fatal("killed publisher left no recoverable stage", err)
			}
			if phase != "journal" {
				if _, err := os.Stat(s.journalPath()); err != nil {
					t.Fatal("target write preceded durable journal", err)
				}
			}
			// Generic unlocked continuation staging is deliberately outside this cleanup.
			unlocked := filepath.Join(s.root, ".thread-store-unlocked")
			if err := os.WriteFile(unlocked, []byte("live continuation writer"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := s.RecoverStoreJournal(); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(s.publicationPath()); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("publication stage retained", err)
			}
			raw, err := os.ReadFile(unlocked)
			if err != nil || string(raw) != "live continuation writer" {
				t.Fatal("cleanup touched generic staging", err)
			}
			if phase == "journal" {
				if _, err := s.GetThread(record.Address); err != nil {
					t.Fatal("unpublished archive changed thread", err)
				}
				if _, err := os.Stat(s.archivePath(record.Address)); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("unpublished archive acquired authority", err)
				}
			} else if phase == "receipt" {
				if _, err := os.Stat(s.archiveReceiptPath("killed-publication")); err != nil {
					t.Fatal("receipt replay missing", err)
				}
				if _, err := os.Stat(s.archivePath(record.Address)); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("archive replay did not detach", err)
				}
			} else {
				if _, err := readTestArchiveGrace(s, record.Address); err != nil {
					t.Fatal("archive replay failed", err)
				}
			}
		})
	}
}

func TestStorePublicationChild(t *testing.T) {
	if os.Getenv("PAIR_STORE_PUBLICATION_CHILD") != "1" {
		return
	}
	ns, err := ExistingCouchNamespace(os.Getenv("PAIR_STORE_PUBLICATION_NAMESPACE"))
	if err != nil {
		t.Fatal(err)
	}
	c, err := storagegc.NewCoordinator(os.Getenv("PAIR_STORE_PUBLICATION_ROOT"))
	if err != nil {
		t.Fatal(err)
	}
	c.Now = func() time.Time { return time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC) }
	s, err := NewCoordinatedThreadStore(ns, c)
	if err != nil {
		t.Fatal(err)
	}
	address := ThreadAddress{RepoScope: "816fc349d3faebf8", Tag: "couch-0000000000000001"}
	phase := os.Getenv("PAIR_STORE_PUBLICATION_PHASE")
	target := s.journalPath()
	switch phase {
	case "archive":
		target = s.archivePath(address)
	case "grace":
		target = s.archiveGracePath(address)
	case "receipt":
		target = s.archiveReceiptPath("killed-publication")
	}
	s.hooks.AfterPublicationWrite = func(path string) error {
		if path == target {
			fmt.Println("staged")
			for {
				time.Sleep(time.Hour)
			}
		}
		return nil
	}
	if phase == "receipt" {
		e, err := readTestArchiveGrace(s, address)
		if err != nil {
			t.Fatal(err)
		}
		err = c.WithLock(context.Background(), func(held *storagegc.Locked) error {
			return s.DetachArchive(held, ArchiveDetachRequest{OperationID: "killed-publication", Address: address, RecordHash: e.RecordHash, ArchivedAt: e.ArchivedAt})
		})
		if err != nil {
			t.Fatal(err)
		}
	} else if err := s.ArchiveThread(address); err != nil {
		t.Fatal(err)
	}
	t.Fatal("publication barrier not reached")
}

func TestStorePublicationRecoveryRejectsUnsafeStage(t *testing.T) {
	for _, kind := range []string{"symlink", "directory"} {
		t.Run(kind, func(t *testing.T) {
			s, _ := retentionStore(t)
			if _, err := s.Snapshot(); err != nil {
				t.Fatal(err)
			}
			outside := filepath.Join(t.TempDir(), "outside")
			if err := os.WriteFile(outside, []byte("keep"), 0600); err != nil {
				t.Fatal(err)
			}
			var err error
			if kind == "symlink" {
				err = os.Symlink(outside, s.publicationPath())
			} else {
				err = os.Mkdir(s.publicationPath(), 0700)
			}
			if err != nil {
				t.Fatal(err)
			}
			if err := s.RecoverStoreJournal(); err == nil {
				t.Fatal("unsafe staging accepted")
			}
			if _, err := os.Lstat(s.publicationPath()); err != nil {
				t.Fatal("unsafe staging removed", err)
			}
			if raw, err := os.ReadFile(outside); err != nil || string(raw) != "keep" {
				t.Fatal("outside target changed")
			}
		})
	}
}

func TestRetentionJournalCancellationPreservesDurableReplay(t *testing.T) {
	s, c, request := archiveDetachFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.hooks.AfterTarget = func(index int) error {
		if index == 0 {
			cancel()
		}
		return nil
	}
	err := c.WithLock(ctx, func(held *storagegc.Locked) error { return s.DetachArchive(held, request) })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation ignored %v", err)
	}
	if _, err := os.Stat(s.archiveReceiptPath(request.OperationID)); err != nil {
		t.Fatal("first durable entry missing", err)
	}
	for _, path := range []string{s.journalPath(), s.archivePath(request.Address), s.archiveGracePath(request.Address)} {
		if _, err := os.Stat(path); err != nil {
			t.Fatal("cancelled journal lost replay authority", path, err)
		}
	}
	s.hooks = threadStoreHooks{}
	if err := c.WithLock(context.Background(), func(held *storagegc.Locked) error { return s.DetachArchive(held, request) }); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(s.archivePath(request.Address)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("resume failed to detach", err)
	}
}

func TestStoreJournalReplayChecksCancellationBetweenEntries(t *testing.T) {
	s, c, request := archiveDetachFixture(t)
	interrupted := errors.New("journal published")
	s.hooks.AfterJournal = func() error { return interrupted }
	if err := c.WithLock(context.Background(), func(held *storagegc.Locked) error { return s.DetachArchive(held, request) }); !errors.Is(err, interrupted) {
		t.Fatal(err)
	}
	s.hooks = threadStoreHooks{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	check := func() error {
		if _, err := os.Stat(s.archiveReceiptPath(request.OperationID)); err == nil {
			cancel()
		}
		return ctx.Err()
	}
	if err := s.withStoreLockChecked(func() error { return nil }, check); !errors.Is(err, context.Canceled) {
		t.Fatalf("replay ignored cancellation %v", err)
	}
	for _, path := range []string{s.journalPath(), s.archivePath(request.Address), s.archiveGracePath(request.Address)} {
		if _, err := os.Stat(path); err != nil {
			t.Fatal("replay proceeded beyond cancellation", path, err)
		}
	}
	if err := s.RecoverStoreJournal(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(s.archivePath(request.Address)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("resume failed", err)
	}
}

func TestStorePublicationCancellationBeforeRenameLeavesJournalAuthority(t *testing.T) {
	s, c, request := archiveDetachFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	target := s.archiveReceiptPath(request.OperationID)
	s.hooks.AfterPublicationWrite = func(path string) error {
		if path == target {
			cancel()
		}
		return nil
	}
	if err := c.WithLock(ctx, func(held *storagegc.Locked) error { return s.DetachArchive(held, request) }); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-rename cancellation ignored %v", err)
	}
	for _, path := range []string{target, s.publicationPath()} {
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("cancelled publication retained stage or installed target", path, err)
		}
	}
	for _, path := range []string{s.journalPath(), s.archivePath(request.Address), s.archiveGracePath(request.Address)} {
		if _, err := os.Stat(path); err != nil {
			t.Fatal("lost replay authority", path, err)
		}
	}
	s.hooks = threadStoreHooks{}
	if err := s.RecoverStoreJournal(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatal("replay did not install receipt", err)
	}
}
