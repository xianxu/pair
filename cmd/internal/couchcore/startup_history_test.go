package couchcore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestStartupObservesUnenrolledConfiguredHistoryWithoutMutation(t *testing.T) {
	for _, family := range []string{"local-current", "local-archive", "local-recovery", "global-archive", "global-legacy"} {
		t.Run(family, func(t *testing.T) {
			f := newProvisionFixture(t)
			f.git(f.Primary, "worktree", "add", "-b", "main-slot1", f.host(1), "main")
			env := newTestEnv(t, f.Primary, f.host(1))
			env.Couch.Slots = NewOSSlotCatalog(f)
			repository, err := env.Couch.Slots.Discover(context.Background(), f.Primary)
			if err != nil {
				t.Fatal(err)
			}
			local := newSlotThreadStore(env.Couch.Namespace, repository.Slots[0].Identity)
			var path string
			switch family {
			case "local-current":
				path = filepath.Join(local.root, "thread.json")
			case "local-archive":
				path = filepath.Join(local.root, "archive", "retained-scope", "retained.json")
			case "local-recovery":
				path = filepath.Join(local.root, "recovery", "retained.json")
			case "global-archive":
				path = filepath.Join(env.Couch.Threads.root, "archive", "retained-scope", "retained.json")
			case "global-legacy":
				path = filepath.Join(env.Couch.Threads.root, "records", "retained-scope", "retained.json")
			}
			if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				t.Fatal(err)
			}
			// Even unreadable history is evidence, never permission to replace it.
			if err := os.WriteFile(path, []byte("retained history: do not replace"), 0600); err != nil {
				t.Fatal(err)
			}
			before, err := snapshotDirectoryForTest(filepath.Dir(f.Remote))
			if err != nil {
				t.Fatal(err)
			}
			observed, err := env.Couch.ObserveStartupContext(context.Background(), []string{f.Primary, f.host(1)})
			if err != nil {
				t.Fatal(err)
			}
			if observed.Kind == StartupFresh {
				t.Errorf("retained %s classified fresh: %+v", family, observed)
			}
			after, err := snapshotDirectoryForTest(filepath.Dir(f.Remote))
			if err != nil || !reflect.DeepEqual(before, after) {
				t.Fatalf("read-only startup changed configured storage: %v", err)
			}
		})
	}
}

func TestStartupHistoryLockedPresenceAndScope(t *testing.T) {
	for _, family := range []string{"records", "archive"} {
		t.Run(family, func(t *testing.T) {
			store, _ := newTestThreadStore(t)
			path := filepath.Join(store.root, family, "kept", "note.json")
			if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte("not decoded"), 0600); err != nil {
				t.Fatal(err)
			}
			for _, scope := range []string{"", "kept", "other"} {
				err := store.withLock(func() error {
					present, err := store.startupHistoryLocked(t.Context(), scope)
					if err != nil {
						return err
					}
					if present != (scope != "other") {
						t.Errorf("scope %q present=%v", scope, present)
					}
					return nil
				})
				if err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestStartupHistoryPreviewRefusesUnsafeAndPending(t *testing.T) {
	for _, damage := range []string{"symlink", "journal"} {
		t.Run(damage, func(t *testing.T) {
			store, _ := newTestThreadStore(t)
			if err := store.withLock(func() error { return nil }); err != nil {
				t.Fatal(err)
			}
			if damage == "journal" {
				if err := os.WriteFile(store.journalPath(), []byte("pending"), 0600); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := os.Symlink(t.TempDir(), filepath.Join(store.root, "archive")); err != nil {
					t.Fatal(err)
				}
			}
			view := *store
			view.readOnly = true
			if err := view.withPreviewLockContext(t.Context(), func() error { _, err := view.startupHistoryLocked(t.Context(), ""); return err }); err == nil {
				t.Fatal("unsafe history accepted")
			}
		})
	}
}

func TestStartupHistoryDirectoryWorkBudget(t *testing.T) {
	for _, entries := range []int{8, 16} {
		t.Run(fmt.Sprint(entries), func(t *testing.T) {
			root := t.TempDir()
			for i := 0; i < entries; i++ {
				if err := os.Mkdir(filepath.Join(root, fmt.Sprintf("scope-%02d", i)), 0700); err != nil {
					t.Fatal(err)
				}
			}
			present, err := startupHistoryTreeWithin(t.Context(), root, "", entries)
			if err != nil || present {
				t.Fatalf("exact entry budget: present=%v err=%v", present, err)
			}
			if _, err := startupHistoryTreeWithin(t.Context(), root, "", entries-1); err == nil {
				t.Fatal("under-budget control accepted")
			}
		})
	}
}

func TestStartupHistoryMissingConfiguredLocalStaysMissing(t *testing.T) {
	f := newProvisionFixture(t)
	f.git(f.Primary, "worktree", "add", "-b", "main-slot1", f.host(1), "main")
	env := newTestEnv(t, f.Primary, f.host(1))
	env.Couch.Slots = NewOSSlotCatalog(f)
	repository, err := env.Couch.Slots.Discover(t.Context(), f.Primary)
	if err != nil {
		t.Fatal(err)
	}
	local := newSlotThreadStore(env.Couch.Namespace, repository.Slots[0].Identity)
	present, err := env.Couch.observeStartupHistory(t.Context(), []string{f.Primary, f.host(1)})
	if err != nil || present {
		t.Fatalf("missing history: present=%v err=%v", present, err)
	}
	if _, err := os.Lstat(local.root); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("observation initialized local store: %v", err)
	}
}

func TestStartupHistoryLockedRetainsRecoveryOnlyLocalStore(t *testing.T) {
	store := testLocalThreadStore(t)
	if err := store.withLock(func() error {
		path := filepath.Join(store.root, "recovery", "retained.json")
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte("unreadable retained original"), 0600); err != nil {
			return err
		}
		present, err := store.startupHistoryLocked(t.Context(), "")
		if err != nil {
			return err
		}
		if !present {
			t.Error("recovery-only local store classified empty")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
