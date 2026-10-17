package pairlog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// pair#618: a recorded peer turn is one submitted entry under the peer prefix,
// so correlation reads it; recording the same delivery again adds nothing.
func TestPersistPeerTurnRecordsOneSubmittedEntry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "log.md")
	envelope := []byte("[Couch peer from ops:0; delivery d-1]\nbody")
	unmanaged := func(string) string { return "" }
	if err := PersistPeerTurn(unmanaged, path, envelope, time.Now(), "d-1"); err != nil {
		t.Fatal(err)
	}
	if err := PersistPeerTurn(unmanaged, path, envelope, time.Now(), "d-1"); err == nil || !strings.Contains(err.Error(), "already submitted") {
		t.Fatalf("repeat err=%v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	parsed := sessioninventory.ParsePairLog(raw, 0)
	if len(parsed.Entries) != 1 || !parsed.Entries[0].Submitted || parsed.Entries[0].AppendID != PeerAppendIDPrefix+"d-1" || parsed.Entries[0].AuthoredText != string(envelope) {
		t.Fatalf("entries=%#v", parsed.Entries)
	}
}
