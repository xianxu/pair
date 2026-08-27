package launcher

import (
	"errors"
	"fmt"
	"os"
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
  "starting_path": "/repo/task",
  "working_path": "/repo/task",
  "created_at": "2026-08-26T12:00:00Z",
  "revision": 3,
  "claim_generation": 1,
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

func TestLoadThreadIndexTreatsCorruptStateAsAuthoritative(t *testing.T) {
	_, err := LoadThreadIndex("/global/couch", func(string) (string, error) {
		return "{not-json", nil
	})
	if err == nil || errors.Is(err, ErrThreadIndexAbsent) {
		t.Fatalf("corrupt manifest error = %v, want authoritative corruption", err)
	}
}

func TestOSRuntimeThreadIndexDistinguishesAbsentStoreFromIncompleteStore(t *testing.T) {
	root := t.TempDir()
	rt := OSRuntime{CouchStoreDir: root}
	_, err := rt.ReadThreadIndex()
	if !errors.Is(err, ErrThreadIndexAbsent) {
		t.Fatalf("absent store error = %v, want ErrThreadIndexAbsent", err)
	}

	if err := os.Mkdir(filepath.Join(root, "threadstore"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err = rt.ReadThreadIndex()
	if err == nil || errors.Is(err, ErrThreadIndexAbsent) {
		t.Fatalf("incomplete store error = %v, want authoritative failure", err)
	}
}

func TestLoadThreadIndexRejectsMissingAddressedRecord(t *testing.T) {
	root := "/global/couch"
	manifestPath := filepath.Join(root, "threadstore", "manifest.json")
	_, err := LoadThreadIndex(root, func(path string) (string, error) {
		if path == manifestPath {
			return `{"schema_version":1,"threads":[{"repo_scope":"816fc349d3faebf8","tag":"couch-0102030405060708"}]}`, nil
		}
		return "", os.ErrNotExist
	})
	if err == nil || errors.Is(err, ErrThreadIndexAbsent) {
		t.Fatalf("missing addressed record error = %v, want authoritative failure", err)
	}
}

func TestLoadThreadIndexRejectsRecordMissingAuthoritativeRequiredFields(t *testing.T) {
	root := "/global/couch"
	files := map[string]string{
		filepath.Join(root, "threadstore", "manifest.json"): `{"schema_version":1,"generation":1,"threads":[{"repo_scope":"816fc349d3faebf8","tag":"couch-0102030405060708"}]}`,
		filepath.Join(root, "threadstore", "records", "816fc349d3faebf8", "couch-0102030405060708.json"): `{
  "schema_version": 1,
  "address": {"repo_scope":"816fc349d3faebf8","tag":"couch-0102030405060708"},
  "working_path": "/repo",
  "created_at": "2026-08-26T12:00:00Z",
  "revision": 1
}`,
	}
	_, err := LoadThreadIndex(root, func(path string) (string, error) {
		return files[path], nil
	})
	if err == nil {
		t.Fatal("portable reader accepted missing starting_path and claim_generation")
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
