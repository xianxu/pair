package launcher

import (
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/sessionledger"
)

func TestSessionLedgerRoundTripAndLatest(t *testing.T) {
	first := LedgerEntry{
		Agent:      "claude",
		Args:       []string{"--old"},
		SessionID:  "A",
		Started:    time.Unix(10, 0).UTC(),
		LastActive: time.Unix(20, 0).UTC(),
		RepoRoot:   "/repo",
		RepoName:   "pair",
	}
	second := LedgerEntry{
		Agent:      "codex",
		Args:       []string{"--search"},
		SessionID:  "B",
		Started:    time.Unix(30, 0).UTC(),
		LastActive: time.Unix(40, 0).UTC(),
		RepoRoot:   "/repo",
		RepoName:   "pair",
	}
	line1, err := BuildLedgerLine(first)
	if err != nil {
		t.Fatalf("BuildLedgerLine(first): %v", err)
	}
	line2, err := BuildLedgerLine(second)
	if err != nil {
		t.Fatalf("BuildLedgerLine(second): %v", err)
	}

	entries := ParseLedger(line1 + "\nnot-json\n" + line2 + "\n")
	if len(entries) != 2 {
		t.Fatalf("entries len = %d, want 2: %#v", len(entries), entries)
	}
	latest, ok := LatestLedgerEntry(entries)
	if !ok || latest.Agent != "codex" || latest.SessionID != "B" {
		t.Fatalf("latest = %#v ok=%v, want codex/B", latest, ok)
	}
}

func TestLauncherParseLedgerPrefersTypedCurrentGeneration(t *testing.T) {
	t.Parallel()
	legacy, err := BuildLedgerLine(LedgerEntry{Agent: "claude", SessionID: "stale", LastActive: time.Unix(999, 0).UTC()})
	if err != nil {
		t.Fatal(err)
	}
	launch, err := sessionledger.EncodeRecord(sessionledger.Record{Version: 1, Kind: sessionledger.RecordLaunch, ScopeKey: "scope", Tag: "work", Agent: "claude"})
	if err != nil {
		t.Fatal(err)
	}
	binding, err := sessionledger.EncodeRecord(sessionledger.Record{Version: 1, Kind: sessionledger.RecordBinding, ScopeKey: "scope", Tag: "work", Agent: "claude", LaunchOrdinal: 2, RootNativeID: "current"})
	if err != nil {
		t.Fatal(err)
	}
	entries := ParseLedger(legacy + "\n" + string(launch) + "\n" + string(binding) + "\n")
	latest, ok := LatestLedgerEntryForAgent(entries, "claude")
	if !ok || latest.SessionID != "current" || !latest.Typed || latest.SourceOrdinal != 2 {
		t.Fatalf("latest=%#v ok=%v entries=%#v", latest, ok, entries)
	}
}

func TestLatestLedgerEntryForAgent(t *testing.T) {
	entries := []LedgerEntry{
		{Agent: "claude", SessionID: "old", LastActive: time.Unix(10, 0).UTC()},
		{Agent: "codex", SessionID: "cx", LastActive: time.Unix(30, 0).UTC()},
		{Agent: "claude", SessionID: "new", LastActive: time.Unix(20, 0).UTC()},
	}

	got, ok := LatestLedgerEntryForAgent(entries, "claude")
	if !ok || got.SessionID != "new" {
		t.Fatalf("latest claude = %#v ok=%v, want new", got, ok)
	}
	if _, ok := LatestLedgerEntryForAgent(entries, "agy"); ok {
		t.Fatal("agy unexpectedly found")
	}
}

func TestCompactLedgerKeepsRecentAndLatestPerAgent(t *testing.T) {
	entries := []LedgerEntry{
		{Agent: "claude", SessionID: "c1", LastActive: time.Unix(10, 0).UTC()},
		{Agent: "codex", SessionID: "x1", LastActive: time.Unix(20, 0).UTC()},
		{Agent: "claude", SessionID: "c2", LastActive: time.Unix(30, 0).UTC()},
		{Agent: "agy", SessionID: "a1", LastActive: time.Unix(40, 0).UTC()},
	}

	got := CompactLedger(entries, 1)
	var ids []string
	for _, e := range got {
		ids = append(ids, e.SessionID)
	}
	joined := strings.Join(ids, ",")
	for _, want := range []string{"x1", "c2", "a1"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("CompactLedger ids = %s, missing %s", joined, want)
		}
	}
	if strings.Contains(joined, "c1") {
		t.Fatalf("CompactLedger ids = %s, should drop old claude", joined)
	}
}
