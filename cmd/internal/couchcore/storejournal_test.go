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
	if got.Revision != 1 {
		t.Fatalf("recovered record = %+v", got)
	}
	if snapshot, err := restarted.Snapshot(); err != nil || snapshot.Generation != 1 {
		t.Fatalf("recovered manifest generation = %d, %v", snapshot.Generation, err)
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

// Every other recovery test here calls RecoverStoreJournal explicitly. This
// one pins the property an operator actually depends on: a pending journal is
// rolled forward TRANSPARENTLY by the next ordinary operation, because
// withLock recovers before running it. Nobody calls RecoverStoreJournal on the
// real startup path.
//
// It used to pin this through CutoverLegacyActors, deleted in pair#170 M4.
// The cutover was only ever the vehicle; the journal property is the subject,
// so it moved to CreateThread rather than going out with the cutover.
func TestStoreJournalPendingWorkRecoversOnTheNextOrdinaryOperation(t *testing.T) {
	_, ns := newTestThreadStore(t)
	crashing := newThreadStoreWithHooks(ns, threadStoreHooks{
		AfterTarget: func(index int) error {
			if index == 0 {
				return errInjectedStoreCrash
			}
			return nil
		},
	})
	interrupted := journalTestRecord(t, ns)
	if _, err := crashing.CreateThread(interrupted); !errors.Is(err, errInjectedStoreCrash) {
		t.Fatalf("CreateThread crash err = %v", err)
	}
	if _, err := os.Stat(crashing.journalPath()); err != nil {
		t.Fatalf("durable intent missing after crash: %v", err)
	}

	// No RecoverStoreJournal call: the next operation must do it.
	restarted := NewThreadStore(ns)
	next := journalTestRecord(t, ns)
	next.Address.Tag = "couch-1112131415161718"
	if _, err := restarted.CreateThread(next); err != nil {
		t.Fatalf("ordinary operation on a store with a pending journal: %v", err)
	}
	// The manifest is the only evidence that distinguishes recovery from
	// coincidence. The interrupted record's FILE was already on disk when the
	// crash hit -- the hook fires after target 0 -- and the next commit clears
	// the journal whether or not it rolled anything forward. Only manifest
	// MEMBERSHIP requires the pending entry to have actually been replayed.
	snapshot, err := restarted.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot after transparent recovery: %v", err)
	}
	listed := map[ThreadAddress]bool{}
	for _, record := range snapshot.Records {
		listed[record.Address] = true
	}
	if !listed[interrupted.Address] {
		t.Fatalf("interrupted record was not rolled into the manifest: %+v", snapshot.Records)
	}
	if !listed[next.Address] {
		t.Fatalf("the recovering operation did not commit its own work: %+v", snapshot.Records)
	}
	if _, err := os.Stat(restarted.journalPath()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("journal survived recovery: %v", err)
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
