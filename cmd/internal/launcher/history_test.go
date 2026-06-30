package launcher

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestHistorySourceScansCwdPrefixedDraftAndLogSidecars(t *testing.T) {
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
	if len(got) != 2 {
		t.Fatalf("Scan returned %#v, want 2 pair-prefixed tags", got)
	}
	if got[0].Tag != "pair" || got[1].Tag != "pair-old" {
		t.Fatalf("Scan returned %#v, want sorted pair tags", got)
	}
}
