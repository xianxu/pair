package couchcore

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func newTestThreadStore(t *testing.T) (*ThreadStore, CouchNamespace) {
	t.Helper()
	ns := testCouchNamespace(t)
	return NewThreadStore(ns), ns
}

func TestThreadStoreRejectsDuplicatePersistedFields(t *testing.T) {
	store, ns := newTestThreadStore(t)
	record := validThreadRecord(t)
	record.StartingPath, record.WorkingPath = ns.Dir(), ns.Dir()
	created, err := store.CreateThread(record)
	if err != nil {
		t.Fatalf("CreateThread: %v", err)
	}
	path := store.recordPath(created.Address)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	corrupt := strings.Replace(string(raw), `"revision": 1`, `"revision": 1, "revision": 2`, 1)
	if err := os.WriteFile(path, []byte(corrupt), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetThread(created.Address); err == nil || !strings.Contains(err.Error(), "duplicate JSON object key") {
		t.Fatalf("GetThread duplicate-field err = %v", err)
	}
}

func TestThreadStoreRejectsCompositeTraversalBeforeAnyPathLookup(t *testing.T) {
	store, ns := newTestThreadStore(t)
	record := validThreadRecord(t)
	record.Address.RepoScope = "../outside"
	record.StartingPath, record.WorkingPath = ns.Dir(), ns.Dir()
	_, err := store.CreateThread(record)
	if err == nil || !strings.Contains(err.Error(), "invalid thread repo scope") {
		t.Fatalf("CreateThread traversal err = %v", err)
	}
	var exists *ThreadExistsError
	if errors.As(err, &exists) {
		t.Fatalf("invalid address reached filesystem collision lookup: %v", err)
	}
}

func TestThreadStoreUpdateExistingThreadUsesRevisionWithoutChangingManifest(t *testing.T) {
	store, ns := newTestThreadStore(t)
	record := validThreadRecord(t)
	record.StartingPath, record.WorkingPath = ns.Dir(), ns.Dir()
	created, err := store.CreateThread(record)
	if err != nil {
		t.Fatalf("CreateThread: %v", err)
	}
	beforeGeneration, err := store.ManifestGeneration()
	if err != nil {
		t.Fatalf("ManifestGeneration: %v", err)
	}

	other := NewThreadStore(ns)
	updated, err := other.UpdateExistingThread(created.Address, created.Revision, func(next *ThreadRecord) error {
		next.Description = "first description"
		return nil
	})
	if err != nil {
		t.Fatalf("UpdateExistingThread: %v", err)
	}
	if updated.Revision != created.Revision+1 || updated.Description != "first description" {
		t.Fatalf("updated = %+v", updated)
	}
	afterGeneration, err := store.ManifestGeneration()
	if err != nil {
		t.Fatalf("ManifestGeneration(after): %v", err)
	}
	if afterGeneration != beforeGeneration {
		t.Fatalf("single-record update changed manifest generation: %d -> %d", beforeGeneration, afterGeneration)
	}

	_, err = store.UpdateExistingThread(created.Address, created.Revision, func(next *ThreadRecord) error {
		next.Description = "stale overwrite"
		return nil
	})
	var conflict *ThreadRevisionError
	if !errors.As(err, &conflict) {
		t.Fatalf("stale update err = %v, want *ThreadRevisionError", err)
	}
	got, err := store.GetThread(created.Address)
	if err != nil {
		t.Fatalf("GetThread: %v", err)
	}
	if got.Description != "first description" {
		t.Fatalf("stale writer changed record: %+v", got)
	}
}

func TestThreadStoreReturnsDefensiveIncarnationCopies(t *testing.T) {
	store, ns := newTestThreadStore(t)
	record := validThreadRecord(t)
	record.StartingPath, record.WorkingPath = ns.Dir(), ns.Dir()
	record.Incarnations = []ThreadIncarnation{{PID: 42, Identity: "original", State: IncarnationUnknown}}
	created, err := store.CreateThread(record)
	if err != nil {
		t.Fatalf("CreateThread: %v", err)
	}
	created.Incarnations[0].Identity = "caller mutation"
	got, err := store.GetThread(record.Address)
	if err != nil {
		t.Fatalf("GetThread: %v", err)
	}
	if got.Incarnations[0].Identity != "original" {
		t.Fatalf("stored incarnation aliased caller: %+v", got.Incarnations)
	}
	got.Incarnations[0].Identity = "read mutation"
	again, _ := store.GetThread(record.Address)
	if again.Incarnations[0].Identity != "original" {
		t.Fatalf("read result aliased store: %+v", again.Incarnations)
	}
}

func TestThreadStoreIndependentInstancesSerializeRevisionUpdates(t *testing.T) {
	store, ns := newTestThreadStore(t)
	record := validThreadRecord(t)
	record.StartingPath, record.WorkingPath = ns.Dir(), ns.Dir()
	created, err := store.CreateThread(record)
	if err != nil {
		t.Fatalf("CreateThread: %v", err)
	}

	stores := []*ThreadStore{NewThreadStore(ns), NewThreadStore(ns)}
	var wg sync.WaitGroup
	errs := make([]error, len(stores))
	for i := range stores {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = stores[i].UpdateExistingThread(created.Address, created.Revision, func(next *ThreadRecord) error {
				next.Description = string(rune('a' + i))
				return nil
			})
		}(i)
	}
	wg.Wait()
	var successes, conflicts int
	for _, err := range errs {
		if err == nil {
			successes++
			continue
		}
		var conflict *ThreadRevisionError
		if errors.As(err, &conflict) {
			conflicts++
		}
	}
	if !reflect.DeepEqual([]int{successes, conflicts}, []int{1, 1}) {
		t.Fatalf("successes/conflicts = %d/%d, errors=%v", successes, conflicts, errs)
	}
}
