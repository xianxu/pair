package couchcore

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"testing"
	"time"
)

func legacyMigrationRecord(t *testing.T) ThreadRecord {
	t.Helper()
	record := ThreadRecord{
		SchemaVersion: ThreadSchemaVersion,
		Address:       ThreadAddress{RepoScope: "816fc349d3faebf8", Tag: "pair"},
		StartingPath:  "/repo",
		WorkingPath:   "/repo",
		CreatedAt:     time.Unix(1, 0).UTC(),
		Revision:      2,
		Incarnations: []ThreadIncarnation{
			{LegacyActorID: "actor-1", PID: 41, Identity: "one", State: IncarnationUnknown, StartedAt: time.Unix(1, 0).UTC()},
			{LegacyActorID: "actor-2", PID: 42, Identity: "two", State: IncarnationUnknown, StartedAt: time.Unix(2, 0).UTC()},
		},
	}
	if err := ValidateThreadRecord(record); err != nil {
		t.Fatal(err)
	}
	return record
}

func TestThreadStoreMigratesLegacyMetadataAtomicallyAndRerunsByteStable(t *testing.T) {
	store, ns := newTestThreadStore(t)
	actor1 := ActorRecord{ID: "actor-1", Args: StartArgs{Worktree: Worktree(ns.Dir())}, PID: 41, Identity: "one", StartedAt: time.Unix(1, 0).UTC()}
	actor2 := ActorRecord{ID: "actor-2", Args: StartArgs{Worktree: Worktree(ns.Dir()), SameTree: true}, PID: 42, Identity: "two", StartedAt: time.Unix(2, 0).UTC()}
	legacyStore := NewStore(ns.Dir())
	registry := NewRegistry().Insert(actor1).Insert(actor2)
	names := NewNamingTable().SetName(Worktree(ns.Dir()), "compiler").SetDescription(Worktree(ns.Dir()), "fix parser")
	if err := legacyStore.Save(registry, names); err != nil {
		t.Fatal(err)
	}
	legacyRaw, err := os.ReadFile(legacyStore.registryPath())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CutoverLegacyActors(registry.Records()); err != nil {
		t.Fatal(err)
	}
	if err := store.MigrateLegacyRecords(names); err != nil {
		t.Fatal(err)
	}
	address, err := legacyThreadAddress(actor1)
	if err != nil {
		t.Fatal(err)
	}
	record, err := store.GetThread(address)
	if err != nil {
		t.Fatal(err)
	}
	if record.Name != "compiler" || record.Description != "fix parser" || len(record.Incarnations) != 2 {
		t.Fatalf("migrated record = %+v", record)
	}
	if raw, err := os.ReadFile(legacyStore.registryPath()); err != nil || !bytes.Equal(raw, legacyRaw) {
		t.Fatalf("legacy registry changed: equal=%v err=%v", bytes.Equal(raw, legacyRaw), err)
	}
	recordRaw, err := os.ReadFile(store.recordPath(address))
	if err != nil {
		t.Fatal(err)
	}
	manifestRaw, err := os.ReadFile(store.manifestPath())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.MigrateLegacyRecords(names); err != nil {
		t.Fatal(err)
	}
	againRecord, _ := os.ReadFile(store.recordPath(address))
	againManifest, _ := os.ReadFile(store.manifestPath())
	if !bytes.Equal(recordRaw, againRecord) || !bytes.Equal(manifestRaw, againManifest) {
		t.Fatal("rerunning completed migration changed durable bytes")
	}
}

func TestThreadStoreLegacyMigrationPreservesUnreadableRecordAndIncompleteMarker(t *testing.T) {
	store, ns := newTestThreadStore(t)
	actor := ActorRecord{ID: "actor-1", Args: StartArgs{Worktree: Worktree(ns.Dir())}, PID: 41, Identity: "one", StartedAt: time.Unix(1, 0).UTC()}
	if err := store.CutoverLegacyActors([]ActorRecord{actor}); err != nil {
		t.Fatal(err)
	}
	address, err := legacyThreadAddress(actor)
	if err != nil {
		t.Fatal(err)
	}
	corrupt := []byte("{not-json\n")
	if err := os.WriteFile(store.recordPath(address), corrupt, 0o600); err != nil {
		t.Fatal(err)
	}
	manifestBefore, err := os.ReadFile(store.manifestPath())
	if err != nil {
		t.Fatal(err)
	}
	err = store.MigrateLegacyRecords(NewNamingTable().SetName(Worktree(ns.Dir()), "compiler"))
	if err == nil {
		t.Fatal("migration accepted unreadable legacy cutover record")
	}
	recordAfter, _ := os.ReadFile(store.recordPath(address))
	manifestAfter, _ := os.ReadFile(store.manifestPath())
	if !bytes.Equal(recordAfter, corrupt) || !bytes.Equal(manifestAfter, manifestBefore) {
		t.Fatal("failed migration changed unreadable input or completion marker")
	}
}

func TestThreadStoreLegacyMigrationRecoversWholeJournalAfterInterruption(t *testing.T) {
	_, ns := newTestThreadStore(t)
	base := NewThreadStore(ns)
	actor := ActorRecord{ID: "actor-1", Args: StartArgs{Worktree: Worktree(ns.Dir())}, PID: 41, Identity: "one", StartedAt: time.Unix(1, 0).UTC()}
	if err := base.CutoverLegacyActors([]ActorRecord{actor}); err != nil {
		t.Fatal(err)
	}
	crashing := newThreadStoreWithHooks(ns, threadStoreHooks{AfterTarget: func(index int) error {
		if index == 0 {
			return errInjectedStoreCrash
		}
		return nil
	}})
	err := crashing.MigrateLegacyRecords(NewNamingTable().SetName(Worktree(ns.Dir()), "compiler"))
	if !errors.Is(err, errInjectedStoreCrash) {
		t.Fatalf("migration interruption = %v", err)
	}
	journalRaw, err := os.ReadFile(crashing.journalPath())
	if err != nil {
		t.Fatal(err)
	}
	var journal map[string]any
	if err := json.Unmarshal(journalRaw, &journal); err != nil {
		t.Fatal(err)
	}
	if nonce, ok := journal["nonce"].(string); !ok || nonce == "" {
		t.Fatalf("migration journal nonce = %#v", journal["nonce"])
	}
	restarted := NewThreadStore(ns)
	if err := restarted.RecoverStoreJournal(); err != nil {
		t.Fatal(err)
	}
	address, _ := legacyThreadAddress(actor)
	record, err := restarted.GetThread(address)
	if err != nil || record.Name != "compiler" {
		t.Fatalf("recovered record = %+v err=%v", record, err)
	}
	if err := restarted.MigrateLegacyRecords(NewNamingTable().SetName(Worktree(ns.Dir()), "compiler")); err != nil {
		t.Fatal(err)
	}
}

func TestMigrateLegacyRecordEnrichesExactCutoverThreadWithoutChangingOccupancy(t *testing.T) {
	record := legacyMigrationRecord(t)
	beforeIncarnations := cloneThreadRecord(record).Incarnations
	got, changed, err := MigrateLegacyRecord(record, NameEntry{Tree: "/repo", Name: "compiler", Description: "fix parser"})
	if err != nil {
		t.Fatal(err)
	}
	if !changed || got.Name != "compiler" || got.Description != "fix parser" {
		t.Fatalf("migration = %+v changed=%v", got, changed)
	}
	if got.Revision != record.Revision || !reflect.DeepEqual(got.Incarnations, beforeIncarnations) {
		t.Fatalf("migration changed revision or occupancy: before=%+v after=%+v", record, got)
	}
	if !reflect.DeepEqual(record, legacyMigrationRecord(t)) {
		t.Fatal("migration mutated its input")
	}
}

func TestMigrateLegacyRecordIsIdempotentAndDoesNotOverwriteThreadMetadata(t *testing.T) {
	record := legacyMigrationRecord(t)
	record.Name = "thread name"
	record.Description = "thread description"
	got, changed, err := MigrateLegacyRecord(record, NameEntry{Tree: "/repo", Name: "legacy name", Description: "legacy description"})
	if err != nil || changed || !reflect.DeepEqual(got, record) {
		t.Fatalf("migration = %+v changed=%v err=%v", got, changed, err)
	}
}

func TestMigrateLegacyRecordRejectsWrongTreeAndIgnoresNonLegacyThread(t *testing.T) {
	record := legacyMigrationRecord(t)
	if _, _, err := MigrateLegacyRecord(record, NameEntry{Tree: "/other", Name: "wrong"}); err == nil {
		t.Fatal("wrong-tree metadata accepted")
	}
	record.Incarnations = []ThreadIncarnation{{State: IncarnationLive, PID: 7, Identity: "native"}}
	got, changed, err := MigrateLegacyRecord(record, NameEntry{Tree: "/repo", Name: "legacy"})
	if err != nil || changed || !reflect.DeepEqual(got, record) {
		t.Fatalf("non-legacy migration = %+v changed=%v err=%v", got, changed, err)
	}
}
