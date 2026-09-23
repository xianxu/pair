package couchcore

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestArchivedListingRecoversLocalJournalThroughGlobalFacade(t *testing.T) {
	store, repository, record := migrationFixture(t)
	if err := store.EnrollSlotRepository(context.Background(), repository); err != nil {
		t.Fatal(err)
	}
	local, err := store.storeForAddress(record.Address)
	if err != nil {
		t.Fatal(err)
	}
	interrupted := errors.New("archive interrupted after journal")
	local.hooks.AfterJournal = func() error { return interrupted }
	if err := local.ArchiveThread(record.Address); !errors.Is(err, interrupted) {
		t.Fatalf("archive interruption: %v", err)
	}
	records, err := store.ArchivedThreads()
	if err != nil || len(records) != 1 || records[0].Address != record.Address {
		t.Fatalf("durable local archive omitted: %+v %v", records, err)
	}
}

func TestArchivedListingRejectsLocalSymlinksAndUnexpectedLayout(t *testing.T) {
	for _, name := range []string{"symlink", "nested"} {
		t.Run(name, func(t *testing.T) {
			local := testLocalThreadStore(t)
			path := filepath.Join(local.root, "archive", "scope", "tag.json")
			if name == "nested" {
				path = filepath.Join(local.root, "archive", "extra", "scope", "tag.json")
			}
			if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				t.Fatal(err)
			}
			if name == "symlink" {
				outside := filepath.Join(t.TempDir(), "record.json")
				if err := os.WriteFile(outside, []byte("{}"), 0600); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(outside, path); err != nil {
					t.Fatal(err)
				}
			} else if err := os.WriteFile(path, []byte("{}"), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := local.ArchivedThreads(); err == nil {
				t.Fatal("unsafe local archive accepted")
			}
		})
	}
}
