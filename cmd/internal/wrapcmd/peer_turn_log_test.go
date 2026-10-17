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
