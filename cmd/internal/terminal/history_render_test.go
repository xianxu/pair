package terminal

import (
	"bytes"
	"errors"
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
	if len(packets) == 0 || packets[0] != "\x1b[?1049h" {
		t.Fatalf("ALT transition not isolated: %q", packets)
	}
	normal := f
	normal.AltScreen = false
	plan, err = RenderWithHistory(f, normal, p.History, plan.NextState())
	if err != nil {
		t.Fatal(err)
	}
	packets = nil
	plan.Emit(func(p []byte) error { packets = append(packets, string(p)); return nil })
	if packets[0] != "\x1b[?1049l" || !strings.Contains(strings.Join(packets, ""), "\x1b[3J") {
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
