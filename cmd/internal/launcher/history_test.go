package launcher

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/sessionledger"
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

func TestHistorySourceEnrichesScopedRowsFromLedgerAndSortsByRecency(t *testing.T) {
	dir := t.TempDir()
	now := time.Unix(4000, 0).UTC()
	for _, tag := range []string{"old", "recent"} {
		path := filepath.Join(dir, "draft-"+tag+".md")
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(path, now.Add(-2*time.Hour), now.Add(-2*time.Hour)); err != nil {
			t.Fatal(err)
		}
	}
	line, err := BuildLedgerLine(LedgerEntry{
		Agent:      "codex",
		LastActive: now,
		RepoName:   "pair",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ledger-recent.jsonl"), []byte(line+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := HistorySource{DataDir: dir}.Scan("pair", now.Add(-24*time.Hour))
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("Scan returned %#v, want two rows", got)
	}
	if got[0].Tag != "recent" || got[0].Agent != "codex" || got[0].RepoName != "pair" {
		t.Fatalf("first row = %#v, want recent ledger-enriched row first", got[0])
	}
	if !got[0].MTime.Equal(now) {
		t.Fatalf("recent MTime = %s, want ledger last_active %s", got[0].MTime, now)
	}
}

func TestHistorySourceTypedAuthorityRendersPairSlashOneMetadata(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	now := time.Unix(6000, 0).UTC()
	compatibility, err := BuildLedgerLine(LedgerEntry{Agent: "claude", Args: []string{"--model", "opus"}, SessionID: "stale", Started: now.Add(-time.Hour), LastActive: now, RepoRoot: "/repo/pair", RepoName: "pair"})
	if err != nil {
		t.Fatal(err)
	}
	launch, err := sessionledger.EncodeRecord(sessionledger.Record{Version: 1, Kind: sessionledger.RecordLaunch, ScopeKey: "scope", Tag: "1", Agent: "claude"})
	if err != nil {
		t.Fatal(err)
	}
	binding, err := sessionledger.EncodeRecord(sessionledger.Record{Version: 1, Kind: sessionledger.RecordBinding, ScopeKey: "scope", Tag: "1", Agent: "claude", LaunchOrdinal: 2, RootNativeID: "current"})
	if err != nil {
		t.Fatal(err)
	}
	raw := compatibility + "\n" + string(launch) + "\n" + string(binding) + "\n"
	if err := os.WriteFile(filepath.Join(dir, "ledger-1.jsonl"), []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}

	rows, err := (HistorySource{DataDir: dir}).Scan("pair", now.Add(-24*time.Hour))
	if err != nil || len(rows) != 1 || rows[0].RepoName != "pair" || rows[0].Agent != "claude" {
		t.Fatalf("rows=%#v err=%v", rows, err)
	}
	if label := historicalPickLabel(rows[0], now.Unix()); !strings.HasPrefix(label, "pair/1  claude ") || strings.Contains(label, "?/1") {
		t.Fatalf("label=%q", label)
	}
	latest, ok := LatestLedgerEntry(ParseLedger(raw))
	if !ok || latest.SessionID != "current" {
		t.Fatalf("typed authority=%#v ok=%v", latest, ok)
	}
}

func TestHistorySourceDiscoversLedgerOnlyTags(t *testing.T) {
	dir := t.TempDir()
	now := time.Unix(5000, 0).UTC()
	line, err := BuildLedgerLine(LedgerEntry{
		Agent:      "codex",
		LastActive: now,
		RepoName:   "pair",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ledger-ledgeronly.jsonl"), []byte(line+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := HistorySource{DataDir: dir}.Scan("pair", now.Add(-24*time.Hour))
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}
	if len(got) != 1 || got[0].Tag != "ledgeronly" || got[0].Agent != "codex" || got[0].RepoName != "pair" {
		t.Fatalf("Scan returned %#v, want one ledger-only row", got)
	}
	if !got[0].MTime.Equal(now) {
		t.Fatalf("ledger-only MTime = %s, want %s", got[0].MTime, now)
	}
}
