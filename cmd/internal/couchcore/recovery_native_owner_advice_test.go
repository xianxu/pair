package couchcore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/launcher"
)

type legacyNativeOwnerChecker struct {
	NoThreadArtifactCollisions
	err error
}

func (c legacyNativeOwnerChecker) CheckRecoveryNativeOwner(context.Context, launcher.RecoveryNativeOwnerCheck) error {
	return c.err
}

func TestRecoveryNativeOwnerLegacySlotAdvice(t *testing.T) {
	for _, mode := range []string{"exact", "wrong-scope", "wrong-tag", "no-store", "absent-store", "unreadable-current", "failed-preview", "ambiguous", "global-journal", "local-journal", "ordinary-error"} {
		t.Run(mode, func(t *testing.T) {
			store, repository, record := migrationFixture(t)
			if err := store.EnrollSlotRepository(context.Background(), repository); err != nil {
				t.Fatal(err)
			}
			backend, err := store.storeForAddress(record.Address)
			if err != nil {
				t.Fatal(err)
			}
			owner := artifactpath.StorageOwner{RepoScope: record.Address.RepoScope, Tag: string(record.Address.Tag)}
			switch mode {
			case "wrong-scope":
				owner.RepoScope += "-other"
			case "wrong-tag":
				owner.Tag += "-other"
			case "unreadable-current":
				if err := os.WriteFile(backend.recordPath(record.Address), []byte("{broken"), 0600); err != nil {
					t.Fatal(err)
				}
			case "failed-preview":
				if err := os.WriteFile(store.manifestPath(), []byte("{broken"), 0600); err != nil {
					t.Fatal(err)
				}
			case "ambiguous":
				raw, err := os.ReadFile(store.manifestPath())
				if err != nil {
					t.Fatal(err)
				}
				var manifest threadManifest
				if err := json.Unmarshal(raw, &manifest); err != nil {
					t.Fatal(err)
				}
				manifest.SlotRepositories = append(manifest.SlotRepositories, manifest.SlotRepositories[0])
				raw, err = json.Marshal(manifest)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(store.manifestPath(), raw, 0600); err != nil {
					t.Fatal(err)
				}
			case "global-journal", "local-journal":
				pending := store
				if mode == "local-journal" {
					pending = backend
				}
				interrupted := errors.New("pending legacy advice fixture")
				pending.hooks.AfterJournal = func() error { return interrupted }
				if mode == "local-journal" {
					_, err = backend.updateExistingThread(record.Address, record.Revision, func(next *ThreadRecord) error { next.Description = "must remain pending"; return nil })
				} else {
					_, err = store.CreateThread(validThreadRecord(t))
				}
				if !errors.Is(err, interrupted) {
					t.Fatalf("pending journal fixture: %v", err)
				}
			}
			beforeGlobal, err := snapshotDirectoryForTest(store.namespace.Dir())
			if err != nil {
				t.Fatal(err)
			}
			beforeLocal, err := snapshotDirectoryForTest(backend.slot.EnvironmentRoot)
			if err != nil {
				t.Fatal(err)
			}
			legacy := &launcher.LegacyNativeOwnerError{Owner: owner, Agent: "codex", PID: 2048}
			original := fmt.Errorf("native check: %w", legacy)
			if mode == "ordinary-error" {
				original = errors.New("ordinary ownership failure")
			}
			couch := &Couch{Threads: store, Artifacts: legacyNativeOwnerChecker{err: original}}
			if mode == "no-store" {
				couch.Threads = nil
			}
			if mode == "absent-store" {
				couch.Threads, _ = newTestThreadStore(t)
			}
			got := couch.checkRecoveryNativeOwner(context.Background(), record, RecoveryChoice{Agent: "codex", NativeID: "native-id"}, false)
			if mode == "exact" {
				slot := repository.Slots[0].Identity
				friendly := fmt.Sprintf("%s:%d", slot.Repo, slot.Number)
				for _, advice := range []string{original.Error(), "relaunch " + friendly + " to migrate", "couch --relaunch " + friendly + " --confirm"} {
					if got == nil || !strings.Contains(got.Error(), advice) {
						t.Fatalf("missing %q in %v", advice, got)
					}
				}
				var cause *launcher.LegacyNativeOwnerError
				if !errors.As(got, &cause) || cause != legacy {
					t.Fatalf("lost typed cause: %v", got)
				}
			} else if got != original {
				t.Fatalf("fallback changed error: got %v, want exact %v", got, original)
			}
			afterGlobal, err := snapshotDirectoryForTest(store.namespace.Dir())
			if err != nil {
				t.Fatal(err)
			}
			afterLocal, err := snapshotDirectoryForTest(backend.slot.EnvironmentRoot)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(beforeGlobal, afterGlobal) || !reflect.DeepEqual(beforeLocal, afterLocal) {
				t.Fatal("legacy advice mutated storage or replayed journal")
			}
			if mode == "absent-store" {
				if _, err := os.Lstat(couch.Threads.root); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("advice initialized absent store: %v", err)
				}
			}
		})
	}
}
