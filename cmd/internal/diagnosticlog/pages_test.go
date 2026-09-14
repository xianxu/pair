package diagnosticlog

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func fixtureSegment(t *testing.T, path, name string, last time.Time) {
	t.Helper()
	p := segmentPath(path, name)
	if e := os.MkdirAll(filepath.Dir(p), 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(p, []byte(name), 0600); e != nil {
		t.Fatal(e)
	}
	st, _ := os.Stat(p)
	g := generation{Name: name, Identity: fileIdentity(st), Start: last, LastWrite: last, Size: st.Size(), ModTime: st.ModTime()}
	if e := writeJSON(p+".json", g, true); e != nil {
		t.Fatal(e)
	}
}
func TestDiagnosticPageCursorSurvivesDeletionShrink(t *testing.T) {
	path, now, opts := fixture(t)
	w, e := Open(path, opts)
	if e != nil {
		t.Fatal(e)
	}
	w.Write([]byte("current"))
	w.Close()
	for i := 1; i <= 3; i++ {
		fixtureSegment(t, path, fmt.Sprintf("segment-%032x.log", i), now.Add(-8*24*time.Hour))
	}
	cursor := "current"
	removed := 0
	for i := 0; i < 10; i++ {
		rows, next, done, e := CollectLegacyPage(path, opts, cursor, 1)
		if e != nil {
			t.Fatal(e)
		}
		removed += len(rows)
		if done {
			if removed != 3 {
				t.Fatalf("deleted %d of3", removed)
			}
			return
		}
		if next == cursor {
			t.Fatal("cursor made no progress")
		}
		cursor = next
	}
	t.Fatal("pagination never completed")
}
func TestYoungFirstPageDoesNotHideExpiredLaterSegment(t *testing.T) {
	path, now, opts := fixture(t)
	w, e := Open(path, opts)
	if e != nil {
		t.Fatal(e)
	}
	w.Write([]byte("current"))
	w.Close()
	young := fmt.Sprintf("segment-%032x.log", 1)
	old := fmt.Sprintf("segment-%032x.log", 2)
	fixtureSegment(t, path, young, *now)
	fixtureSegment(t, path, old, now.Add(-8*24*time.Hour))
	cursor := ""
	removed := 0
	for i := 0; i < 10; i++ {
		rows, next, done, e := CollectLegacyPage(path, opts, cursor, 1)
		if e != nil {
			t.Fatal(e)
		}
		removed += len(rows)
		if done {
			break
		}
		cursor = next
	}
	if removed != 1 {
		t.Fatalf("removed %d", removed)
	}
	if _, e = os.Stat(segmentPath(path, young)); e != nil {
		t.Fatal("young segment removed")
	}
	if _, e = os.Stat(segmentPath(path, old)); !os.IsNotExist(e) {
		t.Fatal("old segment starved")
	}
}
func TestPagedPreviewReportsIncompleteHonestly(t *testing.T) {
	path, now, opts := fixture(t)
	w, e := Open(path, opts)
	if e != nil {
		t.Fatal(e)
	}
	w.Write([]byte("current"))
	w.Close()
	for i := 0; i < 20; i++ {
		fixtureSegment(t, path, fmt.Sprintf("segment-%04x%028x.log", i, i), *now)
	}
	cursor := ""
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		rows, next, done, e := PreviewLegacyPage(path, opts, cursor, 3)
		if e != nil {
			t.Fatal(e)
		}
		if len(rows) > 3 {
			t.Fatal("page exceeded limit")
		}
		for _, r := range rows {
			if seen[r.Path] {
				t.Fatal("duplicate", r.Path)
			}
			seen[r.Path] = true
		}
		if done {
			if len(seen) != 21 {
				t.Fatalf("incomplete claimed done: %d", len(seen))
			}
			return
		}
		if next == cursor {
			t.Fatal("cursor stalled")
		}
		cursor = next
	}
	t.Fatal("preview never completed")
}
