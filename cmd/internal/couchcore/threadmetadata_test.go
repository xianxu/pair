package couchcore

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

func metadataThread(scope string, tag ThreadTag, path, name string) ThreadRecord {
	return ThreadRecord{
		SchemaVersion:    ThreadSchemaVersion,
		Address:          ThreadAddress{RepoScope: scope, Tag: tag},
		StartingPath:     path,
		WorkingPath:      path,
		CreatedAt:        time.Unix(1, 0).UTC(),
		Revision:         1,
		ClaimGeneration:  1,
		Name:             name,
		Description:      "operator description",
		PublishedSummary: "agent summary",
	}
}

func metadataValue(value string) *string { return &value }

func TestApplyThreadMetadataPreservesIndependentFields(t *testing.T) {
	record := metadataThread("816fc349d3faebf8", "couch-0102030405060708", "/repo/task", "old name")

	next := ApplyThreadMetadata(record, ThreadMetadataPatch{Name: metadataValue("")})
	if next.Name != "" || next.Description != record.Description || next.PublishedSummary != record.PublishedSummary {
		t.Fatalf("metadata patch crossed fields: %+v", next)
	}
	if record.Name != "old name" {
		t.Fatalf("ApplyThreadMetadata mutated its input: %+v", record)
	}
}

func TestThreadStoreMetadataUpdateUsesRevisionCAS(t *testing.T) {
	store, _ := newTestThreadStore(t)
	record := metadataThread("816fc349d3faebf8", "couch-0102030405060708", "/repo/task", "old name")
	created, err := store.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}

	updated, err := store.ApplyThreadMetadata(created.Address, created.Revision, ThreadMetadataPatch{
		Description:      metadataValue("new description"),
		PublishedSummary: metadataValue("new agent summary"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Revision != created.Revision+1 || updated.Name != "old name" || updated.Description != "new description" || updated.PublishedSummary != "new agent summary" {
		t.Fatalf("updated metadata = %+v", updated)
	}

	_, err = store.ApplyThreadMetadata(created.Address, created.Revision, ThreadMetadataPatch{Name: metadataValue("stale overwrite")})
	var stale *ThreadRevisionError
	if !errors.As(err, &stale) {
		t.Fatalf("stale update error = %v, want ThreadRevisionError", err)
	}
	got, err := store.GetThread(created.Address)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "old name" || got.Description != "new description" || got.PublishedSummary != "new agent summary" {
		t.Fatalf("stale update changed record: %+v", got)
	}
}

func TestResolveThreadReferencePrefersExactScopedTag(t *testing.T) {
	records := []ThreadRecord{
		metadataThread("816fc349d3faebf8", "work", "/repo/one", "other"),
		metadataThread("fedcba9876543210", "work", "/other/one", "other"),
		metadataThread("816fc349d3faebf8", "other", "/repo/work", "work"),
	}

	got, err := ResolveThreadReference(records, "816fc349d3faebf8", "work")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Address != records[0].Address {
		t.Fatalf("exact scoped tag resolution = %+v", got)
	}
}

func TestResolveThreadReferenceRefusesRepeatedGlobalTag(t *testing.T) {
	records := []ThreadRecord{
		metadataThread("816fc349d3faebf8", "work", "/repo/one", "first"),
		metadataThread("fedcba9876543210", "work", "/other/one", "second"),
	}
	got, err := ResolveThreadReference(records, "", "work")
	var ambiguous *AmbiguousThreadReferenceError
	if !errors.As(err, &ambiguous) || len(got) != 2 {
		t.Fatalf("repeated global tag resolution = %+v, %v", got, err)
	}
}

func TestResolveThreadReferenceReturnsEveryAmbiguousCandidate(t *testing.T) {
	records := []ThreadRecord{
		metadataThread("816fc349d3faebf8", "couch-0000000000000001", "/repo/a", "compiler"),
		metadataThread("816fc349d3faebf8", "couch-0000000000000002", "/repo/b", "compiler"),
		metadataThread("fedcba9876543210", "couch-0000000000000003", "/other/compiler", "compiler"),
	}

	got, err := ResolveThreadReference(records, "816fc349d3faebf8", "comp")
	var ambiguous *AmbiguousThreadReferenceError
	if !errors.As(err, &ambiguous) {
		t.Fatalf("ambiguous resolution error = %v", err)
	}
	want := []ThreadAddress{records[0].Address, records[1].Address}
	if len(got) != 2 || !reflect.DeepEqual(ambiguous.Candidates, want) {
		t.Fatalf("ambiguous candidates = records:%+v error:%+v", got, ambiguous.Candidates)
	}
}

func TestResolveThreadReferenceMatchesCanonicalPathAndRefusesEmpty(t *testing.T) {
	record := metadataThread("816fc349d3faebf8", "couch-0102030405060708", "/repo/Arc-AGI-3", "competition")
	got, err := ResolveThreadReference([]ThreadRecord{record}, "816fc349d3faebf8", "arc-agi")
	if err != nil || len(got) != 1 || got[0].Address != record.Address {
		t.Fatalf("path resolution = %+v, %v", got, err)
	}
	if _, err := ResolveThreadReference([]ThreadRecord{record}, "816fc349d3faebf8", "  "); !errors.Is(err, ErrThreadReferenceNotFound) {
		t.Fatalf("empty resolution error = %v", err)
	}
}
