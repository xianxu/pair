package launcher

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestHistorySourceScansAllTagsInScopeDir(t *testing.T) {
	dir := t.TempDir()
	now := time.Unix(2000, 0)
	for _, name := range []string{"draft-pair.md", "log-pair-old.md", "draft-other.md"} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(path, now, now); err != nil {
			t.Fatal(err)
		}
	}

	got, err := HistorySource{DataDir: dir}.Scan("pair", now.Add(-24*time.Hour))
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("Scan returned %#v, want all 3 scoped tags", got)
	}
	if got[0].Tag != "other" || got[1].Tag != "pair" || got[2].Tag != "pair-old" {
		t.Fatalf("Scan returned %#v, want sorted scoped tags", got)
	}
}

func TestHistorySourceAddsAmbiguousLegacyRowsForBaseFamily(t *testing.T) {
	scoped := t.TempDir()
	flat := t.TempDir()
	now := time.Unix(3000, 0)
	for _, name := range []string{"draft-pair.md", "log-pair-old.md", "draft-other.md"} {
		path := filepath.Join(flat, name)
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(path, now, now); err != nil {
			t.Fatal(err)
		}
	}

	got, err := HistorySource{DataDir: scoped, LegacyDataDir: flat}.Scan("pair", now.Add(-24*time.Hour))
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("Scan returned %#v, want pair + pair-old legacy rows", got)
	}
	for _, row := range got {
		if !row.LegacyUnscoped {
			t.Fatalf("row %#v should be marked legacy", row)
		}
		if row.Tag != "pair" && row.Tag != "pair-old" {
			t.Fatalf("unexpected legacy row %#v", row)
		}
	}
}

func TestHistorySourceScopedRowsWinOverLegacyRows(t *testing.T) {
	scoped := t.TempDir()
	flat := t.TempDir()
	now := time.Unix(3000, 0)
	if err := os.WriteFile(filepath.Join(scoped, "draft-pair.md"), []byte("scoped"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(flat, "draft-pair.md"), []byte("flat"), 0o644); err != nil {
		t.Fatal(err)
	}
	_ = os.Chtimes(filepath.Join(scoped, "draft-pair.md"), now, now)
	_ = os.Chtimes(filepath.Join(flat, "draft-pair.md"), now, now)

	got, err := HistorySource{DataDir: scoped, LegacyDataDir: flat}.Scan("pair", now.Add(-24*time.Hour))
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}
	if len(got) != 1 || got[0].Tag != "pair" || got[0].LegacyUnscoped {
		t.Fatalf("Scan returned %#v, want one normal scoped pair row", got)
	}
}
