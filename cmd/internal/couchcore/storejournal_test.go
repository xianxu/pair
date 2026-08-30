package couchcore

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var errInjectedStoreCrash = errors.New("injected store crash")

func journalTestRecord(t *testing.T, ns CouchNamespace) ThreadRecord {
	t.Helper()
	record := validThreadRecord(t)
	record.StartingPath, record.WorkingPath = ns.Dir(), ns.Dir()
	return record
}

func TestStoreJournalRecoversCrashAfterDurableIntent(t *testing.T) {
	_, ns := newTestThreadStore(t)
	crashing := newThreadStoreWithHooks(ns, threadStoreHooks{
		AfterJournal: func() error { return errInjectedStoreCrash },
	})
	record := journalTestRecord(t, ns)
	if _, err := crashing.CreateThread(record); !errors.Is(err, errInjectedStoreCrash) {
		t.Fatalf("CreateThread crash err = %v", err)
	}
	if _, err := os.Stat(crashing.journalPath()); err != nil {
		t.Fatalf("durable intent missing after crash: %v", err)
	}

	restarted := NewThreadStore(ns)
	if err := restarted.RecoverStoreJournal(); err != nil {
		t.Fatalf("RecoverStoreJournal: %v", err)
	}
	got, err := restarted.GetThread(record.Address)
	if err != nil {
		t.Fatalf("GetThread after recovery: %v", err)
	}
	if got.ClaimGeneration != 1 || got.Revision != 1 {
		t.Fatalf("recovered record = %+v", got)
	}
	if generation, _ := restarted.ManifestGeneration(); generation != 1 {
		t.Fatalf("recovered manifest generation = %d", generation)
	}
	if err := restarted.RecoverStoreJournal(); err != nil {
		t.Fatalf("second recovery was not idempotent: %v", err)
	}
}

func TestStoreJournalRollsForwardCrashBetweenRecordAndManifest(t *testing.T) {
	_, ns := newTestThreadStore(t)
	crashing := newThreadStoreWithHooks(ns, threadStoreHooks{
		AfterTarget: func(index int) error {
			if index == 0 {
				return errInjectedStoreCrash
			}
			return nil
		},
	})
	record := journalTestRecord(t, ns)
	if _, err := crashing.CreateThread(record); !errors.Is(err, errInjectedStoreCrash) {
		t.Fatalf("CreateThread crash err = %v", err)
	}
	if _, err := os.Stat(crashing.recordPath(record.Address)); err != nil {
		t.Fatalf("record after-image missing before manifest crash: %v", err)
	}
	if _, err := os.Stat(crashing.manifestPath()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("manifest unexpectedly advanced before recovery: %v", err)
	}

	restarted := NewThreadStore(ns)
	if err := restarted.RecoverStoreJournal(); err != nil {
		t.Fatalf("RecoverStoreJournal: %v", err)
	}
	if _, err := restarted.GetThread(record.Address); err != nil {
		t.Fatalf("recovered thread missing: %v", err)
	}
}

func TestStoreJournalFailsClosedOnThirdTargetState(t *testing.T) {
	_, ns := newTestThreadStore(t)
	crashing := newThreadStoreWithHooks(ns, threadStoreHooks{
		AfterJournal: func() error { return errInjectedStoreCrash },
	})
	record := journalTestRecord(t, ns)
	if _, err := crashing.CreateThread(record); !errors.Is(err, errInjectedStoreCrash) {
		t.Fatalf("CreateThread crash err = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(crashing.recordPath(record.Address)), 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(crashing.recordPath(record.Address), []byte("third state\n"), 0o600); err != nil {
		t.Fatalf("corrupt target: %v", err)
	}

	restarted := NewThreadStore(ns)
	err := restarted.RecoverStoreJournal()
	if err == nil || !strings.Contains(err.Error(), "neither expected-before nor exact after-image") {
		t.Fatalf("recovery err = %v", err)
	}
	if _, statErr := os.Stat(restarted.journalPath()); statErr != nil {
		t.Fatalf("failed recovery cleared evidence: %v", statErr)
	}
}

func TestStoreJournalLegacyCutoverKeepsCoTenantsAndSeparatesRepoScopes(t *testing.T) {
	store, _ := newTestThreadStore(t)
	started := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	actors := []ActorRecord{
		{ID: "couch-a", Args: StartArgs{Worktree: "/one/repo", Cwd: "/one/repo"}, PID: 10, Identity: "a", StartedAt: started},
		{ID: "couch-b", Args: StartArgs{Worktree: "/one/repo", Cwd: "/one/repo/sub"}, PID: 11, Identity: "b", StartedAt: started.Add(time.Minute)},
		{ID: "couch-c", Args: StartArgs{Worktree: "/two/repo", Cwd: "/two/repo"}, PID: 12, Identity: "c", StartedAt: started},
	}
	if err := store.CutoverLegacyActors(actors); err != nil {
		t.Fatalf("CutoverLegacyActors: %v", err)
	}
	firstAddress, err := legacyThreadAddress(actors[0])
	if err != nil {
		t.Fatal(err)
	}
	secondAddress, err := legacyThreadAddress(actors[2])
	if err != nil {
		t.Fatal(err)
	}
	if firstAddress.Tag != secondAddress.Tag || firstAddress.RepoScope == secondAddress.RepoScope {
		t.Fatalf("legacy composite addresses = %+v and %+v", firstAddress, secondAddress)
	}
	first, err := store.GetThread(firstAddress)
	if err != nil {
		t.Fatalf("GetThread(first): %v", err)
	}
	if len(first.Incarnations) != 2 || first.Incarnations[0].State != IncarnationUnknown || first.Incarnations[1].State != IncarnationUnknown {
		t.Fatalf("same-tree co-tenants = %+v", first.Incarnations)
	}
	if _, err := store.GetThread(secondAddress); err != nil {
		t.Fatalf("GetThread(second): %v", err)
	}
	generation, _ := store.ManifestGeneration()
	if err := store.CutoverLegacyActors(actors); err != nil {
		t.Fatalf("repeated cutover: %v", err)
	}
	after, _ := store.ManifestGeneration()
	if after != generation {
		t.Fatalf("repeated cutover changed generation: %d -> %d", generation, after)
	}
}

func TestStoreJournalLegacyCutoverRecoversAsOneMutation(t *testing.T) {
	_, ns := newTestThreadStore(t)
	crashing := newThreadStoreWithHooks(ns, threadStoreHooks{
		AfterTarget: func(index int) error {
			if index == 0 {
				return errInjectedStoreCrash
			}
			return nil
		},
	})
	actor := ActorRecord{ID: "couch-a", Args: StartArgs{Worktree: "/one/repo", Cwd: "/one/repo"}, PID: 10, Identity: "a", StartedAt: time.Now()}
	if err := crashing.CutoverLegacyActors([]ActorRecord{actor}); !errors.Is(err, errInjectedStoreCrash) {
		t.Fatalf("cutover crash err = %v", err)
	}
	restarted := NewThreadStore(ns)
	if err := restarted.CutoverLegacyActors([]ActorRecord{actor}); err != nil {
		t.Fatalf("recover/retry cutover: %v", err)
	}
	address, _ := legacyThreadAddress(actor)
	if _, err := restarted.GetThread(address); err != nil {
		t.Fatalf("legacy record missing after recovery: %v", err)
	}
}

func TestStoreJournalReplaysParkFinalizationAndTombstone(t *testing.T) {
	for _, operation := range []string{"finalize", "abandon"} {
		for _, boundary := range []string{"after journal", "after target"} {
			t.Run(operation+"/"+boundary, func(t *testing.T) {
				store, ns := newTestThreadStore(t)
				created, identity, _ := createParkableThread(t, store, ns, "park-3333333333333333")
				begun, err := store.BeginPark(created.Address, created.Revision, identity)
				if err != nil {
					t.Fatal(err)
				}
				hooks := threadStoreHooks{}
				if boundary == "after journal" {
					hooks.AfterJournal = func() error { return errInjectedStoreCrash }
				} else {
					hooks.AfterTarget = func(index int) error {
						if index == 0 {
							return errInjectedStoreCrash
						}
						return nil
					}
				}
				crashing := newThreadStoreWithHooks(ns, hooks)
				if operation == "finalize" {
					_, err = crashing.FinalizePark(begun.Address, begun.Revision, identity, 1, time.Unix(70, 0).UTC())
				} else {
					_, err = crashing.AbandonPark(begun.Address, begun.Revision, identity)
				}
				if !errors.Is(err, errInjectedStoreCrash) {
					t.Fatalf("injected boundary err = %v", err)
				}

				restarted := NewThreadStore(ns)
				if err := restarted.RecoverStoreJournal(); err != nil {
					t.Fatalf("RecoverStoreJournal: %v", err)
				}
				recovered, err := restarted.GetThread(begun.Address)
				if err != nil {
					t.Fatal(err)
				}
				if recovered.Park != nil || len(recovered.ParkHistory) != 1 || !recovered.ParkHistory[0].Closed {
					t.Fatalf("recovered lifecycle = %+v", recovered)
				}
				if operation == "finalize" {
					if recovered.VerifiedPark == nil || len(recovered.Incarnations) != 0 || recovered.ParkHistory[0].Tombstoned {
						t.Fatalf("finalization replay = %+v", recovered)
					}
				} else if recovered.VerifiedPark != nil || len(recovered.Incarnations) != 1 || !recovered.ParkHistory[0].Tombstoned {
					t.Fatalf("tombstone replay = %+v", recovered)
				}
				if err := restarted.RecoverStoreJournal(); err != nil {
					t.Fatalf("second recovery: %v", err)
				}
			})
		}
	}
}
