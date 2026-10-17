package wrapcmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/pairlog"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// pair#618 BR-11: one launch's peer turns grow the Pair log by a stated bound.
func TestPeerTurnRecorderIsBoundedPerLaunch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "log.md")
	var workers sync.WaitGroup
	var failures []string
	record := peerTurnRecorder(path, func(string) string { return "" }, func(_, detail string) { failures = append(failures, detail) }, &workers)
	for i := range peerTurnRecordLimit + 3 {
		m := couchmessage.Message{ID: fmt.Sprintf("d-%d", i), From: couchmessage.Binding{Slot: "ops:0"}, Body: strings.Repeat("x", 100)}
		record(m)
		workers.Wait() // in order, as deliveries are serial
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	entries := sessioninventory.ParsePairLog(raw, 0).Entries
	if len(entries) != peerTurnRecordLimit || len(failures) != 0 {
		t.Fatalf("%d entries (want %d), failures %v", len(entries), peerTurnRecordLimit, failures)
	}
	if first := entries[0]; first.AppendID != pairlog.PeerAppendIDPrefix+"d-0" || !strings.HasPrefix(first.AuthoredText, "[Couch peer from ops:0; delivery d-0]\n") {
		t.Fatalf("first entry %#v", first)
	}
}

// The exact combined turn is the transcript-correlation unit; splitting it
// into member turns would change every downstream hash (pair#618).
func TestPeerBatchTurnRecorderExactCombinedTurnAndSharedBudget(t *testing.T) {
	path := filepath.Join(t.TempDir(), "batch-log.md")
	var workers sync.WaitGroup
	single, batch := peerTurnRecorders(path, func(string) string { return "" }, func(_, detail string) { t.Error(detail) }, &workers)
	messages := []couchmessage.Message{{ID: "head", From: couchmessage.Binding{Slot: "ops:0"}, Body: "first\nline"}, {ID: "tail", From: couchmessage.Binding{Slot: "pair:2"}, Body: "second 🦉"}}
	batch(messages)
	workers.Wait()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	entries := sessioninventory.ParsePairLog(raw, 0).Entries
	if len(entries) != 1 || entries[0].AuthoredText != couchmessage.BatchEnvelope(messages) || entries[0].AppendID != pairlog.PeerAppendIDPrefix+"head" {
		t.Fatalf("combined turn mismatch: %+v", entries)
	}
	// A few large, variable-width payloads exhaust bytes before eight turns.
	for i := range 5 {
		large := messages[0]
		large.ID = fmt.Sprintf("large-%d", i)
		large.Body = strings.Repeat("界", 30000)
		batch([]couchmessage.Message{large})
		workers.Wait()
	}
	// A rejected oversized turn consumes neither budget; the remaining budget
	// can still accept a small singleton, and both callbacks share the count.
	single(couchmessage.Message{ID: "small", From: messages[0].From, Body: "small"})
	workers.Wait()
	raw, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	entries = sessioninventory.ParsePairLog(raw, 0).Entries
	total := 0
	for _, entry := range entries {
		total += len(entry.AuthoredText)
	}
	if len(entries) != 4 || total > peerTurnRecordBytes || entries[len(entries)-1].AppendID != pairlog.PeerAppendIDPrefix+"small" {
		t.Fatalf("unbounded or split budget: entries=%d bytes=%d", len(entries), total)
	}
}
