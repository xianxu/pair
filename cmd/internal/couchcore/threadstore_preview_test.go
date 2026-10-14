package couchcore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"time"

	"github.com/xianxu/pair/cmd/internal/storagegc"
	"testing"
)

func TestThreadStorePreviewDoesNotInitialize(t *testing.T) {
	t.Parallel()

	s, _ := newTestThreadStore(t)
	snapshot, err := s.PreviewSnapshot()
	if err != nil || len(snapshot.Records) != 0 {
		t.Fatalf("preview=%+v %v", snapshot, err)
	}
	_, found, err := s.PreviewPathLaunchPreference("repo", "/repo", "")
	if err != nil || found {
		t.Fatalf("preference=%v %v", found, err)
	}
	if _, err := os.Lstat(s.root); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("preview created root: %v", err)
	}
}
func TestThreadStorePreviewDoesNotRecoverJournal(t *testing.T) {
	t.Parallel()

	s, _ := newTestThreadStore(t)
	fail := errors.New("interrupted")
	s.hooks.AfterJournal = func() error { return fail }
	if _, err := s.CreateThread(validThreadRecord(t)); !errors.Is(err, fail) {
		t.Fatal(err)
	}
	before, err := os.ReadFile(s.journalPath())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.PreviewSnapshot(); err == nil {
		t.Fatal("preview accepted pending journal")
	}
	after, err := os.ReadFile(s.journalPath())
	if err != nil || string(before) != string(after) {
		t.Fatalf("preview recovered journal: %v", err)
	}
}

func TestSlotPreferenceRefusesSymlinkedPayload(t *testing.T) {
	t.Parallel()

	s := testLocalThreadStore(t)
	if err := os.MkdirAll(s.root, 0700); err != nil {
		t.Fatal(err)
	}
	external := s.root + "-external-preferences"
	if err := os.WriteFile(external, []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, s.pathLaunchPreferencePath(s.slot.RepoIdentity, s.slot.WorktreeRoot)); err != nil {
		t.Fatal(err)
	}
	_, _, err := s.GetPathLaunchPreference(s.slot.RepoIdentity, s.slot.WorktreeRoot)
	if err == nil {
		t.Fatalf("unsafe preference read: %v", err)
	}
}

func TestPreviewThreadPreservesPendingJournalAcrossBackends(t *testing.T) {
	for _, kind := range []string{"global", "local", "routed-local"} {
		t.Run(kind, func(t *testing.T) {
			store, backend, record := previewThreadFixture(t, kind)
			interrupted := errors.New("preview fixture: pending journal")
			backend.hooks.AfterJournal = func() error { return interrupted }
			if _, err := backend.updateExistingThread(record.Address, record.Revision, func(next *ThreadRecord) error { next.Description = "must remain pending"; return nil }); !errors.Is(err, interrupted) {
				t.Fatalf("interrupt transaction: %v", err)
			}
			before, err := snapshotDirectoryForTest(backend.root)
			if err != nil {
				t.Fatal(err)
			}
			_, readErr := store.PreviewThreadContext(context.Background(), record.Address)
			after, err := snapshotDirectoryForTest(backend.root)
			if err != nil {
				t.Fatal(err)
			}
			if readErr == nil || !reflect.DeepEqual(before, after) {
				t.Fatalf("preview replayed pending journal: err=%v changed=%t", readErr, !reflect.DeepEqual(before, after))
			}
		})
	}
}

func previewThreadFixture(t *testing.T, kind string) (*ThreadStore, *ThreadStore, ThreadRecord) {
	t.Helper()
	if kind == "routed-local" {
		store, repository, record := migrationFixture(t)
		if err := store.EnrollSlotRepository(context.Background(), repository); err != nil {
			t.Fatal(err)
		}
		backend, err := store.storeForAddress(record.Address)
		if err != nil {
			t.Fatal(err)
		}
		return store, backend, record
	}
	var store *ThreadStore
	var record ThreadRecord
	if kind == "local" {
		store = testLocalThreadStore(t)
		record = recordAtCheckout(t, store.slot.WorktreeRoot, store.slot.WorktreeRoot, "couch-0123456789abcdef")
	} else {
		store, _ = newTestThreadStore(t)
		record = validThreadRecord(t)
	}
	created, err := store.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	return store, store, created
}

func TestPreviewThreadDoesNotInitializeOrEnrollRetention(t *testing.T) {
	for _, kind := range []string{"global", "local"} {
		t.Run(kind, func(t *testing.T) {
			var store *ThreadStore
			if kind == "local" {
				store = testLocalThreadStore(t)
			} else {
				store, _ = newTestThreadStore(t)
			}
			coordinator, err := storagegc.NewCoordinator(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			store.coordinator = coordinator
			before, err := snapshotDirectoryForTest(store.namespace.Dir())
			if err != nil {
				t.Fatal(err)
			}
			retentionBefore, err := snapshotDirectoryForTest(coordinator.Root)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := store.PreviewThreadContext(context.Background(), validThreadRecord(t).Address); !errors.Is(err, ErrThreadNotFound) {
				t.Fatalf("missing preview: %v", err)
			}
			after, err := snapshotDirectoryForTest(store.namespace.Dir())
			if err != nil {
				t.Fatal(err)
			}
			retentionAfter, err := snapshotDirectoryForTest(coordinator.Root)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(before, after) || !reflect.DeepEqual(retentionBefore, retentionAfter) {
				t.Fatal("preview initialized or enrolled an absent store")
			}
			if _, err := os.Lstat(store.root); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("preview initialized backend: %v", err)
			}
		})
	}
}

func TestPreviewThreadContextBoundsRoutedLocks(t *testing.T) {
	for _, kind := range []string{"global", "local", "routed-local"} {
		t.Run(kind, func(t *testing.T) {
			store, backend, record := previewThreadFixture(t, kind)
			lock, err := backend.retentionReadLock(context.Background(), 0)
			if err != nil {
				t.Fatal(err)
			}
			defer lock.Close()
			canceled, cancel := context.WithCancel(context.Background())
			cancel()
			if _, err := store.PreviewThreadContext(canceled, record.Address); !errors.Is(err, context.Canceled) {
				t.Fatalf("canceled preview: %v", err)
			}
			ctx, stop := context.WithTimeout(context.Background(), 25*time.Millisecond)
			defer stop()
			started := time.Now()
			_, err = store.PreviewThreadContext(ctx, record.Address)
			if err == nil || time.Since(started) > 500*time.Millisecond {
				t.Fatalf("contended preview escaped caller deadline: %v after %s", err, time.Since(started))
			}
		})
	}
}

func TestPreviewThreadSharesDecoderAndUnlockedReadHook(t *testing.T) {
	for _, kind := range []string{"global", "local", "routed-local"} {
		t.Run(kind, func(t *testing.T) {
			store, backend, record := previewThreadFixture(t, kind)
			calls := 0
			store.hooks.AfterGetThread = func(address ThreadAddress) error {
				if address != record.Address {
					t.Fatalf("read hook address: %+v", address)
				}
				lock, err := backend.retentionReadLock(context.Background(), 0)
				if err != nil {
					return fmt.Errorf("read hook ran under backend lock: %w", err)
				}
				defer lock.Close()
				calls++
				return nil
			}
			ordinary, err := store.GetThread(record.Address)
			if err != nil {
				t.Fatal(err)
			}
			preview, err := store.PreviewThreadContext(context.Background(), record.Address)
			if err != nil || !reflect.DeepEqual(ordinary, preview) || calls != 2 {
				t.Fatalf("shared getter/hook: calls=%d preview=%+v err=%v", calls, preview, err)
			}
		})
	}
}
