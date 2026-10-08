package broadcast

import (
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/terminal"
)

func TestStreamJoinThenDiffsReproduceScreen(t *testing.T) {
	a := textFrame(t, 24, 5, "alpha\nline two", liveChrome("x"))
	b := textFrame(t, 24, 5, "beta\nline two\nthree", liveChrome("y"))
	c := textFrame(t, 24, 5, "gamma", liveChrome("z"))

	var all Stream
	var full screen
	for _, f := range []terminal.Frame{a, b} {
		m, ok, err := all.Next(f)
		if err != nil || !ok {
			t.Fatalf("next: %v %v", ok, err)
		}
		full.apply(t, m)
	}
	var late screen
	join, ok, err := all.Join()
	if err != nil || !ok {
		t.Fatalf("join: %v %v", ok, err)
	}
	late.apply(t, join)
	m, ok, err := all.Next(c)
	if err != nil || !ok {
		t.Fatalf("next c: %v %v", ok, err)
	}
	full.apply(t, m)
	late.apply(t, m)
	want := frameText(t, c)
	if full.text() != want || late.text() != want {
		t.Fatalf("screens differ\nfull: %q\nlate: %q\nwant: %q", full.text(), late.text(), want)
	}
}

func TestStreamJoinBeforeAnyFrame(t *testing.T) {
	var s Stream
	if _, ok, err := s.Join(); ok || err != nil {
		t.Fatalf("join before any frame: %v %v", ok, err)
	}
}

func TestStreamUnchangedFrameYieldsNoMessage(t *testing.T) {
	a := textFrame(t, 24, 3, "same", liveChrome(""))
	var s Stream
	if _, ok, _ := s.Next(a); !ok {
		t.Fatal("first frame produced nothing")
	}
	if _, ok, err := s.Next(a.Clone()); ok || err != nil {
		t.Fatalf("unchanged frame produced a message: %v", err)
	}
}

func TestStreamClearClearsViewer(t *testing.T) {
	a := textFrame(t, 24, 4, "BROADCAST-MARKER", liveChrome(""))
	b := textFrame(t, 24, 4, "", liveChrome(""))
	var s Stream
	var v screen
	m1, _, _ := s.Next(a)
	m2, ok, err := s.Next(b)
	if err != nil || !ok {
		t.Fatalf("clear frame: %v %v", ok, err)
	}
	v.apply(t, m1, m2)
	if strings.Contains(v.text(), "BROADCAST-MARKER") {
		t.Fatalf("clear did not clear the viewer: %q", v.text())
	}
}

func TestStreamGeometryChange(t *testing.T) {
	a := textFrame(t, 24, 4, "small", liveChrome(""))
	b := textFrame(t, 40, 6, "bigger", liveChrome(""))
	var s Stream
	s.Next(a)
	m, ok, err := s.Next(b)
	if err != nil || !ok {
		t.Fatalf("resize: %v %v", ok, err)
	}
	if m.Cols != 40 || m.Rows != 6 {
		t.Fatalf("message geometry %dx%d", m.Cols, m.Rows)
	}
	if !strings.Contains(string(m.Data), "\x1b[2J") {
		t.Fatal("geometry change did not redraw fully")
	}
	var v screen
	v.apply(t, m)
	if v.text() != frameText(t, b) {
		t.Fatalf("fresh viewer of the resize message: %q", v.text())
	}
}
