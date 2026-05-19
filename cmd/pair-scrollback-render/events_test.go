package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/x/vt"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestParseEvents_MissingFileIsNotAnError(t *testing.T) {
	got, err := parseEvents(filepath.Join(t.TempDir(), "nope.jsonl"))
	if err != nil {
		t.Fatalf("missing-file: unexpected err %v", err)
	}
	if got != nil {
		t.Errorf("missing-file: got %v, want nil", got)
	}
}

func TestParseEvents_EmptyFile(t *testing.T) {
	p := writeFile(t, t.TempDir(), "events.jsonl", "")
	got, err := parseEvents(p)
	if err != nil {
		t.Fatalf("empty: unexpected err %v", err)
	}
	if got != nil {
		t.Errorf("empty: got %v, want nil", got)
	}
}

func TestParseEvents_SkipsMalformedLines(t *testing.T) {
	// Mix of valid + malformed lines. The malformed ones must be
	// silently skipped — pair-wrap pads the sidecar with one JSON
	// object per write, and a half-written tail line after a hard kill
	// must not abort the entire render.
	content := `{"type":"resize","offset":0,"cols":80,"rows":24}
not-json
{"type":"resize","offset":4096,"cols":120,"rows":40}
{"type":"resize","offset"`
	p := writeFile(t, t.TempDir(), "events.jsonl", content)
	got, err := parseEvents(p)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d events, want 2 (skipping malformed)", len(got))
	}
	if got[0].Cols != 80 || got[0].Rows != 24 {
		t.Errorf("event[0]: %+v, want cols=80 rows=24", got[0])
	}
	if got[1].Offset != 4096 || got[1].Cols != 120 || got[1].Rows != 40 {
		t.Errorf("event[1]: %+v, want offset=4096 cols=120 rows=40", got[1])
	}
}

func TestParseEvents_FiltersNonResize(t *testing.T) {
	// A non-"resize" event type should not show up in the returned
	// slice — feedSegments would otherwise try to treat it as a
	// boundary and call em.Resize with zero cols/rows.
	content := `{"type":"resize","offset":0,"cols":80,"rows":24}
{"type":"capture","offset":1024,"data":"foo"}
{"type":"resize","offset":2048,"cols":100,"rows":30}
`
	p := writeFile(t, t.TempDir(), "events.jsonl", content)
	got, err := parseEvents(p)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d events, want 2 (non-resize filtered)", len(got))
	}
	for _, e := range got {
		if e.Type != "resize" {
			t.Errorf("unexpected non-resize event: %+v", e)
		}
	}
}

func TestInitialSize_DefaultsWhenEmpty(t *testing.T) {
	cols, rows := initialSize(nil)
	if cols != defaultCols || rows != defaultRows {
		t.Errorf("empty: got %dx%d, want %dx%d",
			cols, rows, defaultCols, defaultRows)
	}
}

func TestInitialSize_PicksFirstUsable(t *testing.T) {
	// First entry has zero cols (truncated) → fall through to next.
	events := []resizeEvent{
		{Type: "resize", Offset: 0, Cols: 0, Rows: 0},
		{Type: "resize", Offset: 0, Cols: 120, Rows: 40},
	}
	cols, rows := initialSize(events)
	if cols != 120 || rows != 40 {
		t.Errorf("got %dx%d, want 120x40 (skipping malformed first entry)", cols, rows)
	}
}

func TestFeedSegments_ClampsOffsetBeyondEOF(t *testing.T) {
	// Synthesized corrupted sidecar case: events claim a resize at an
	// offset well past the end of the raw byte stream. Without the
	// clamp in feedSegments, raw[cursor:off] would panic.
	em := vt.NewEmulator(80, 24)
	defer em.Close()
	go consumeAll(em) // drain emulator replies, see render() comment

	raw := []byte("hello\r\nworld\r\n") // 14 bytes
	events := []resizeEvent{
		{Type: "resize", Offset: 0, Cols: 80, Rows: 24},
		{Type: "resize", Offset: 9999, Cols: 100, Rows: 30}, // way past EOF
	}

	// Should not panic. We can't easily inspect what's in the
	// emulator, but the absence of a panic + the resize having
	// landed (em.Width == 100 after) is the observable contract.
	feedSegments(em, raw, events)
	if w, h := em.Width(), em.Height(); w != 100 || h != 30 {
		t.Errorf("post-feed size: got %dx%d, want 100x30 (resize event applied)",
			w, h)
	}
}

func TestFeedSegments_WritesAllRawWhenNoMidStreamResizes(t *testing.T) {
	em := vt.NewEmulator(80, 24)
	defer em.Close()
	go consumeAll(em)

	raw := []byte("alpha\r\nbeta\r\ngamma")
	events := []resizeEvent{
		{Type: "resize", Offset: 0, Cols: 80, Rows: 24},
	}
	feedSegments(em, raw, events) // must not panic; trailing tail flushed
	// Size should be unchanged.
	if w, h := em.Width(), em.Height(); w != 80 || h != 24 {
		t.Errorf("size drifted: got %dx%d, want 80x24", w, h)
	}
}

func TestFeedSegments_AppliesMidStreamResize(t *testing.T) {
	em := vt.NewEmulator(80, 24)
	defer em.Close()
	go consumeAll(em)

	raw := []byte("first chunk\r\nsecond chunk\r\n")
	// Mid-stream resize after the first line.
	events := []resizeEvent{
		{Type: "resize", Offset: 0, Cols: 80, Rows: 24},
		{Type: "resize", Offset: 13, Cols: 120, Rows: 40},
	}
	feedSegments(em, raw, events)
	if w, h := em.Width(), em.Height(); w != 120 || h != 40 {
		t.Errorf("post-feed size: got %dx%d, want 120x40", w, h)
	}
}

// consumeAll drains an emulator's response pipe in the background.
// The emulator writes replies to status queries (DSR, DA) back into
// its own reader, and offscreen tests with no consumer would
// deadlock on those writes. Mirrors what render() does in main.go.
func consumeAll(em *vt.Emulator) {
	buf := make([]byte, 256)
	for {
		_, err := em.Read(buf)
		if err != nil {
			return
		}
	}
}
