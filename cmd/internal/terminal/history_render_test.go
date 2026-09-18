package terminal

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/ttyio"
)

func historyFixture(t *testing.T, wire string) (Publication, Frame) {
	t.Helper()
	e, err := NewEndpoint("history", Geometry{4, 4}, ttyio.NewFake())
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	e.Feed([]byte(wire), time.Time{})
	p, err := e.Publication(time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	chrome, _ := StyledRows("CHR", 4, 1)
	f, err := Compose(p.Frame, Geometry{4, 5}, chrome)
	if err != nil {
		t.Fatal(err)
	}
	return p, f
}
func renderHistoryBytes(t *testing.T, prev, next Frame, h HistoryWindow, state HistoryState) ([]byte, HistoryState) {
	t.Helper()
	plan, err := RenderWithHistory(prev, next, h, state)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := plan.Emit(func(p []byte) error {
		if len(p) > 64<<10 {
			t.Fatalf("oversize chunk %d", len(p))
		}
		_, err := out.Write(p)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	return out.Bytes(), plan.NextState()
}
func TestHistoryRendererStateAndCancellation(t *testing.T) {
	p, f := historyFixture(t, "AAA界Z\r\nAA  BZ\r\nHARD\r\nONE\r\nTWO\r\nTHR\r\nFOUR")
	wire, state := renderHistoryBytes(t, Frame{}, f, p.History, HistoryState{})
	if !bytes.Contains(wire, []byte("\x1b[3J")) || bytes.Contains(wire, []byte("\x1b[2J")) {
		t.Fatal("rebuild must clear history without ED2")
	}
	if state.Cursor != p.History.Cursor || state.EndpointID != "history" {
		t.Fatal("wrong next state")
	}
	idle, _ := renderHistoryBytes(t, f, f, p.History, state)
	if len(idle) != 0 {
		t.Fatal("idle repainted")
	}
	plan, err := RenderWithHistory(Frame{}, f, p.History, HistoryState{})
	if err != nil {
		t.Fatal(err)
	}
	stop := errors.New("stop")
	calls := 0
	if err := plan.Emit(func([]byte) error { calls++; return stop }); !errors.Is(err, stop) || calls != 1 {
		t.Fatalf("cancellation %v %d", err, calls)
	}
}
func TestHistoryRendererRejectsUnsafeWindowBeforeWriting(t *testing.T) {
	p, f := historyFixture(t, "A\r\nB\r\nC\r\nD\r\nE")
	p.History.Rows[0].Cells[0].Content = "\x1b[2J"
	if _, err := RenderWithHistory(Frame{}, f, p.History, HistoryState{}); err == nil {
		t.Fatal("unsafe history admitted")
	}
}
func TestHistoryRendererAltTransitionsAreWholePackets(t *testing.T) {
	p, f := historyFixture(t, "A")
	f.AltScreen = true
	plan, err := RenderWithHistory(Frame{}, f, p.History, HistoryState{})
	if err != nil {
		t.Fatal(err)
	}
	var packets []string
	plan.Emit(func(p []byte) error { packets = append(packets, string(p)); return nil })
	// The ALT packet stays whole (Presenter tracks ownership by exact match) and
	// sits inside the frame's synchronized-output bracket (#262).
	if len(packets) < 3 || packets[0] != syncBegin || packets[1] != "\x1b[?1049h" || !strings.HasSuffix(packets[len(packets)-1], syncEnd) {
		t.Fatalf("ALT enter not a whole packet inside the bracket: %q", packets)
	}
	normal := f
	normal.AltScreen = false
	plan, err = RenderWithHistory(f, normal, p.History, plan.NextState())
	if err != nil {
		t.Fatal(err)
	}
	packets = nil
	plan.Emit(func(p []byte) error { packets = append(packets, string(p)); return nil })
	if len(packets) < 3 || packets[0] != syncBegin || packets[1] != "\x1b[?1049l" || !strings.HasSuffix(packets[len(packets)-1], syncEnd) {
		t.Fatalf("ALT leave not a whole packet inside the bracket: %q", packets)
	}
	if !strings.Contains(strings.Join(packets, ""), "\x1b[3J") {
		t.Fatalf("exit did not restore primary: %q", packets)
	}
}

func TestHistoryRendererClipsUnrepresentableGlyphWithoutLosingSource(t *testing.T) {
	e, err := NewEndpoint("narrow", Geometry{4, 2}, ttyio.NewFake())
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	e.Feed([]byte("界\r\nB\r\nC"), time.Time{})
	if err := e.Resize(Geometry{1, 2}, func(Geometry) error { return nil }); err != nil {
		t.Fatal(err)
	}
	p, err := e.Publication(time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	wire, state := renderHistoryBytes(t, Frame{}, p.Frame, p.History, HistoryState{})
	if bytes.Contains(wire, []byte("界")) {
		t.Fatal("oversized glyph emitted into one column")
	}
	if p.History.Rows[0].Cells[0].Content != "界" {
		t.Fatal("source history lost")
	}
	if err := e.Resize(Geometry{4, 2}, func(Geometry) error { return nil }); err != nil {
		t.Fatal(err)
	}
	wide, err := e.Publication(time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	wire, _ = renderHistoryBytes(t, p.Frame, wide.Frame, wide.History, state)
	if !bytes.Contains(wire, []byte("界")) {
		t.Fatal("widening did not restore source history")
	}
}

func TestHistoryEmitBracketsEveryFrame(t *testing.T) {
	source := "AAA界Z\r\nAA  BZ\r\nHARD\r\nONE\r\nTWO\r\nTHR\r\nFOUR"
	p, f := historyFixture(t, source)
	wire, state := renderHistoryBytes(t, Frame{}, f, p.History, HistoryState{})
	assertOneBracket(t, "reset", wire)

	p2, f2 := historyFixture(t, source+"\r\nNEXT")
	wire, state2 := renderHistoryBytes(t, f, f2, p2.History, state)
	assertOneBracket(t, "history append", wire)

	edited := f2.Clone()
	edited.Cells[0] = Cell{Content: "Q", Width: 1}
	edited.Rows = nil // derive row metadata from the edited cells
	wire, _ = renderHistoryBytes(t, f2, edited, p2.History, state2)
	assertOneBracket(t, "steady one-cell change", wire)

	if idle, _ := renderHistoryBytes(t, f2, f2, p2.History, state2); len(idle) != 0 {
		t.Fatalf("an idle frame must write nothing, got %q", idle)
	}
}

// bigAltEndpoint holds an alt-screen frame whose serialization spans several
// 64 KiB chunks: every cell carries its own SGR colour.
func bigAltEndpoint(t *testing.T, id string) *Endpoint {
	t.Helper()
	e, err := NewEndpoint(id, Geometry{200, 100}, ttyio.NewFake())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { e.Close() })
	var b strings.Builder
	b.WriteString("\x1b[?1049h")
	for y := 0; y < 100; y++ {
		for x := 0; x < 200; x++ {
			fmt.Fprintf(&b, "\x1b[38;5;%dm#", (x+y)%256)
		}
	}
	e.Feed([]byte(b.String()), time.Time{})
	return e
}

// A frame larger than one chunk is still ONE bracket: opened by the first write,
// closed by the last, never re-opened or closed in between.
func TestHistoryEmitBracketSpansChunks(t *testing.T) {
	pub, err := bigAltEndpoint(t, "big").Publication(time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := RenderWithHistory(Frame{}, pub.Frame, pub.History, HistoryState{})
	if err != nil {
		t.Fatal(err)
	}
	var chunks []string
	if err := plan.Emit(func(p []byte) error { chunks = append(chunks, string(p)); return nil }); err != nil {
		t.Fatal(err)
	}
	if len(chunks) < 5 { // syncBegin, ?1049h, and at least three body chunks
		t.Fatalf("fixture must span several body chunks, got %d writes", len(chunks))
	}
	assertOneBracket(t, "multi-chunk", []byte(strings.Join(chunks, "")))
	if chunks[0] != syncBegin || !strings.HasSuffix(chunks[len(chunks)-1], syncEnd) {
		t.Fatal("the bracket must open in the first write and close in the last")
	}
}
