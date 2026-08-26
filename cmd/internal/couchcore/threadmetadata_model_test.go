package couchcore

import (
	"testing"
	"time"
)

func TestApplyThreadMetadataPreservesIndependentFields(t *testing.T) {
	empty := ""
	record := ThreadRecord{
		SchemaVersion: ThreadSchemaVersion,
		Address: ThreadAddress{
			RepoScope: "816fc349d3faebf8",
			Tag:       "couch-0102030405060708",
		},
		StartingPath:     "/repo/task",
		WorkingPath:      "/repo/task",
		CreatedAt:        time.Unix(1, 0).UTC(),
		Revision:         1,
		ClaimGeneration:  1,
		Name:             "old name",
		Description:      "operator description",
		PublishedSummary: "agent summary",
	}

	next := ApplyThreadMetadata(record, ThreadMetadataPatch{Name: &empty})
	if next.Name != "" || next.Description != record.Description || next.PublishedSummary != record.PublishedSummary {
		t.Fatalf("metadata patch crossed fields: %+v", next)
	}
	if record.Name != "old name" {
		t.Fatalf("ApplyThreadMetadata mutated its input: %+v", record)
	}
}
