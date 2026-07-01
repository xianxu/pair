package wrapcmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// dueForTimeEvent is the pure minute-debounce decision (#59): first event
// always fires (zero last), then at most one per minute.
func TestDueForTimeEvent(t *testing.T) {
	base := time.Date(2026, 6, 14, 10, 0, 0, 0, time.UTC)
	if !dueForTimeEvent(time.Time{}, base) {
		t.Fatal("first event (zero last) should be due")
	}
	if dueForTimeEvent(base, base.Add(59*time.Second)) {
		t.Fatal("59s < 1min should not be due")
	}
	if !dueForTimeEvent(base, base.Add(60*time.Second)) {
		t.Fatal("exactly 60s should be due")
	}
}

// maybeLogTime emits a debounced "time" event to the events sidecar: the first
// call emits, a same-instant call is skipped, and a call >1min later emits
// again — so a >1min activity window yields 2 events, a sub-minute burst 1.
func TestMaybeLogTimeDebounced(t *testing.T) {
	dir := t.TempDir()
	f, err := os.Create(filepath.Join(dir, "e.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	clock := time.Date(2026, 6, 14, 10, 0, 0, 0, time.UTC)
	p := &proxy{eventsFD: f, now: func() time.Time { return clock }}

	p.maybeLogTime() // first → emit
	p.maybeLogTime() // same instant → skip
	clock = clock.Add(61 * time.Second)
	p.maybeLogTime() // >1min → emit

	if err := f.Sync(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "e.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(data), `"type":"time"`); n != 2 {
		t.Fatalf("got %d time events, want 2:\n%s", n, data)
	}
	// The offset field is present (anchors the event to the byte stream).
	if !strings.Contains(string(data), `"offset"`) {
		t.Fatalf("time event missing offset:\n%s", data)
	}
}
