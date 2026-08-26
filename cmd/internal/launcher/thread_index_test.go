package launcher

import (
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func indexEntry(scope, tag, path, name string) ThreadIndexEntry {
	return ThreadIndexEntry{
		Address:     ThreadIndexAddress{RepoScope: scope, Tag: tag},
		WorkingPath: path, Name: name, CreatedAt: time.Unix(10, 0).UTC(),
	}
}

func TestResolveThreadIndexReferenceUsesScopedExactTagAndReturnsAmbiguity(t *testing.T) {
	entries := []ThreadIndexEntry{
		indexEntry("816fc349d3faebf8", "work", "/repo/one", "compiler"),
		indexEntry("fedcba9876543210", "work", "/other/one", "compiler"),
		indexEntry("816fc349d3faebf8", "other", "/repo/work", "work"),
	}
	got, err := ResolveThreadIndexReference(entries, "816fc349d3faebf8", "work")
	if err != nil || len(got) != 1 || got[0].Address != entries[0].Address {
		t.Fatalf("scoped exact resolution = %+v, %v", got, err)
	}
	got, err = ResolveThreadIndexReference(entries, "", "work")
	var ambiguous *AmbiguousThreadIndexReferenceError
	if !errors.As(err, &ambiguous) || len(got) != 2 {
		t.Fatalf("global repeated tag = %+v, %v", got, err)
	}
	got, err = ResolveThreadIndexReference(entries, "816fc349d3faebf8", "comp")
	if err != nil || len(got) != 1 || got[0].Address.Tag != "work" {
		t.Fatalf("name resolution = %+v, %v", got, err)
	}
}

func TestLoadThreadIndexReadsManifestAddressedRecords(t *testing.T) {
	root := "/global/couch"
	files := map[string]string{
		filepath.Join(root, "threadstore", "manifest.json"): `{
  "schema_version": 1,
  "generation": 2,
  "threads": [{"repo_scope":"816fc349d3faebf8","tag":"couch-0102030405060708"}]
}`,
		filepath.Join(root, "threadstore", "records", "816fc349d3faebf8", "couch-0102030405060708.json"): `{
  "schema_version": 1,
  "address": {"repo_scope":"816fc349d3faebf8","tag":"couch-0102030405060708"},
  "working_path": "/repo/task",
  "created_at": "2026-08-26T12:00:00Z",
  "revision": 3,
  "name": "compiler",
  "description": "operator context",
  "published_summary": "agent context"
}`,
	}
	index, err := LoadThreadIndex(root, func(path string) (string, error) {
		raw, ok := files[path]
		if !ok {
			return "", fmt.Errorf("missing %s", path)
		}
		return raw, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	want := indexEntry("816fc349d3faebf8", "couch-0102030405060708", "/repo/task", "compiler")
	want.CreatedAt = time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	want.Description = "operator context"
	want.PublishedSummary = "agent context"
	if !reflect.DeepEqual(index.Entries, []ThreadIndexEntry{want}) {
		t.Fatalf("index = %#v", index)
	}
}

func TestMergeThreadHistoryAddsParkedRowsAndHumanNames(t *testing.T) {
	index := ThreadIndex{Entries: []ThreadIndexEntry{
		indexEntry("816fc349d3faebf8", "couch-0000000000000001", "/repo", "compiler"),
		indexEntry("816fc349d3faebf8", "couch-0000000000000002", "/repo", ""),
		indexEntry("fedcba9876543210", "couch-0000000000000003", "/other", "foreign"),
	}}
	history := []HistoricalTag{{Tag: "couch-0000000000000001", MTime: time.Unix(20, 0)}}
	got := MergeThreadHistory(history, index, RepoScope{Key: "816fc349d3faebf8", DisplayName: "repo"})
	if len(got) != 2 || got[0].Name != "compiler" || got[1].Tag != "couch-0000000000000002" {
		t.Fatalf("merged history = %+v", got)
	}
}
