package historyarchive

import (
	"testing"
	"time"
)

func TestArchivedIndexLatestVerifiedTracksMaximumBySourceAndPath(t *testing.T) {
	x := NewArchivedIndex()
	source := SourceIdentity{Host: "host", Root: "/root", Repository: "repo", Worktree: "main"}
	other := source
	other.Worktree = "other"
	old := time.Date(2026, 10, 17, 0, 0, 0, 0, time.UTC)
	latest := time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)
	if got := x.LatestVerified(source, "state.sqlite"); !got.IsZero() {
		t.Fatalf("empty index verified at %v", got)
	}
	x.AddRecord(CatalogRecord{Manifest: Manifest{Source: source, Files: []FileEntry{{Path: "state.sqlite", Size: 11, Mtime: old, SHA256: "old"}, {Path: "logs.sqlite", SHA256: "logs"}}}, VerifiedAt: old})
	x.AddRecord(CatalogRecord{Manifest: Manifest{Source: source, Files: []FileEntry{{Path: "state.sqlite", Size: 12, Mtime: latest, SHA256: "new"}}}, VerifiedAt: latest})
	x.AddRecord(CatalogRecord{Manifest: Manifest{Source: source, Files: []FileEntry{{Path: "state.sqlite", SHA256: "older"}}}, VerifiedAt: old})
	x.AddRecord(CatalogRecord{Manifest: Manifest{Source: other, Files: []FileEntry{{Path: "state.sqlite", SHA256: "other"}}}, VerifiedAt: latest.Add(time.Hour)})
	if got := x.LatestVerified(source, "state.sqlite"); !got.Equal(latest) {
		t.Fatalf("latest = %v, want %v", got, latest)
	}
	if got := x.LatestVerified(source, "logs.sqlite"); !got.Equal(old) {
		t.Fatalf("logs = %v, want %v", got, old)
	}
	if got := x.LatestVerified(other, "state.sqlite"); !got.Equal(latest.Add(time.Hour)) {
		t.Fatalf("other source = %v", got)
	}
	if got := x.LatestVerified(source, "absent.sqlite"); !got.IsZero() {
		t.Fatalf("absent path = %v", got)
	}
	if !x.SeenHash(source, "state.sqlite", "old") || !x.SeenHash(source, "state.sqlite", "new") || !x.Seen(source, Candidate{Path: "state.sqlite", Size: 12, Mtime: latest}) {
		t.Fatal("AddRecord lost manifest content/stamps")
	}
}

func TestArchivedIndexAddDoesNotClaimVerifiedTime(t *testing.T) {
	x := NewArchivedIndex()
	source := SourceIdentity{Host: "host", Root: "/root"}
	x.Add(Manifest{Source: source, Files: []FileEntry{{Path: "state.sqlite", SHA256: "hash"}}})
	if !x.SeenHash(source, "state.sqlite", "hash") {
		t.Fatal("Add lost content hash")
	}
	if got := x.LatestVerified(source, "state.sqlite"); !got.IsZero() {
		t.Fatalf("manifest alone claims verification: %v", got)
	}
}
