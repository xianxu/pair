package historyarchive

import (
	"testing"
	"time"
)

func TestSelectQuietKeepsOnlyFilesIdleForTheWholePeriod(t *testing.T) {
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	quiet := 72 * time.Hour
	cs := []Candidate{
		{Path: "old", Mtime: now.Add(-quiet - time.Second), Regular: true},
		{Path: "edge", Mtime: now.Add(-quiet), Regular: true},
		{Path: "fresh", Mtime: now.Add(-quiet + time.Second), Regular: true},
		{Path: "dir", Mtime: now.Add(-quiet - time.Hour)},
	}
	got := SelectQuiet(cs, now, quiet)
	if len(got) != 2 || got[0].Path != "old" || got[1].Path != "edge" {
		t.Fatalf("selected %+v, want old and edge", got)
	}
}

func TestArchivedIndexDedupsByIdentityPathAndHash(t *testing.T) {
	id := SourceIdentity{Host: "h", Root: "/r"}
	other := SourceIdentity{Host: "h", Root: "/other"}
	mt := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	sha := "aa"
	idx := NewArchivedIndex()
	idx.Add(Manifest{Source: id, Files: []FileEntry{{Path: "a", Size: 3, Mtime: mt, SHA256: sha}}})
	if !idx.Seen(id, Candidate{Path: "a", Size: 3, Mtime: mt}) {
		t.Fatal("same path, size and mtime must be seen without hashing")
	}
	if idx.Seen(id, Candidate{Path: "a", Size: 4, Mtime: mt}) {
		t.Fatal("a changed size must be hashed, not assumed archived")
	}
	if !idx.SeenHash(id, "a", sha) {
		t.Fatal("same content under the same path is archived")
	}
	if idx.SeenHash(id, "a", "bb") {
		t.Fatal("new content is a new version")
	}
	if idx.Seen(other, Candidate{Path: "a", Size: 3, Mtime: mt}) || idx.SeenHash(other, "a", sha) {
		t.Fatal("another source identity has its own history")
	}
}
