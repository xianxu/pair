package couchcore

import (
	"strings"
	"testing"
	"time"
)

func validThreadRecord(t *testing.T) ThreadRecord {
	t.Helper()
	return ThreadRecord{
		SchemaVersion: ThreadSchemaVersion,
		Address:       ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-0123456789abcdef"},
		StartingPath:  testCouchNamespace(t).Dir(),
		WorkingPath:   testCouchNamespace(t).Dir(),
		CreatedAt:     time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC),
		Revision:      1,
	}
}

func TestThreadStoreValidateThreadRecordRejectsInvalidBoundaries(t *testing.T) {
	base := validThreadRecord(t)
	tests := []struct {
		name   string
		mutate func(*ThreadRecord)
	}{
		{name: "zero schema", mutate: func(r *ThreadRecord) { r.SchemaVersion = 0 }},
		{name: "unknown schema", mutate: func(r *ThreadRecord) { r.SchemaVersion++ }},
		{name: "scope traversal", mutate: func(r *ThreadRecord) { r.Address.RepoScope = "../other" }},
		{name: "tag traversal", mutate: func(r *ThreadRecord) { r.Address.Tag = "../other" }},
		{name: "relative starting path", mutate: func(r *ThreadRecord) { r.StartingPath = "relative" }},
		{name: "relative working path", mutate: func(r *ThreadRecord) { r.WorkingPath = "relative" }},
		{name: "zero creation", mutate: func(r *ThreadRecord) { r.CreatedAt = time.Time{} }},
		{name: "zero revision", mutate: func(r *ThreadRecord) { r.Revision = 0 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := cloneThreadRecord(base)
			tt.mutate(&record)
			if err := ValidateThreadRecord(record); err == nil {
				t.Fatalf("accepted invalid record: %+v", record)
			}
		})
	}
}

func TestThreadStoreValidateThreadRecordAcceptsLegacyTag(t *testing.T) {
	record := validThreadRecord(t)
	record.Address.Tag = "pair-work_2"
	if err := ValidateThreadRecord(record); err != nil {
		t.Fatalf("legacy tag rejected: %v", err)
	}
	if strings.Contains(string(record.Address.Tag), "/") {
		t.Fatal("test setup contains traversal")
	}
}

func FuzzThreadStoreValidateThreadRecordNeverPanics(f *testing.F) {
	f.Add("0123456789abcdef", "couch-0123456789abcdef", "/repo", 1, uint64(1))
	f.Add("../scope", "bad/tag", "relative", 0, uint64(0))
	f.Fuzz(func(t *testing.T, scope, tag, path string, schema int, revision uint64) {
		record := ThreadRecord{
			SchemaVersion: schema,
			Address:       ThreadAddress{RepoScope: scope, Tag: ThreadTag(tag)},
			StartingPath:  path,
			WorkingPath:   path,
			CreatedAt:     time.Unix(1, 0),
			Revision:      revision,
		}
		_ = ValidateThreadRecord(record)
	})
}
