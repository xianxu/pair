package broadcast

import (
	"image/color"
	"sort"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/terminal"
)

var t0 = time.Unix(1000, 0)

func markedCells(m *Marks) [][2]int {
	var out [][2]int
	for c := range m.cells {
		out = append(out, [2]int{c.col, c.row})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i][1] != out[j][1] {
			return out[i][1] < out[j][1]
		}
		return out[i][0] < out[j][0]
	})
	return out
}

func TestMarksAddFillsLinesWithinABatch(t *testing.T) {
	cases := []struct {
		name   string
		points [][2]int
		want   [][2]int
	}{
		{"tap", [][2]int{{3, 2}}, [][2]int{{3, 2}}},
		{"horizontal", [][2]int{{1, 1}, {4, 1}}, [][2]int{{1, 1}, {2, 1}, {3, 1}, {4, 1}}},
		{"diagonal", [][2]int{{0, 0}, {3, 3}}, [][2]int{{0, 0}, {1, 1}, {2, 2}, {3, 3}}},
		{"repeated point", [][2]int{{2, 2}, {2, 2}}, [][2]int{{2, 2}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := NewMarks()
			m.Add(c.points, 20, 10, t0)
			got := markedCells(m)
			if len(got) != len(c.want) {
				t.Fatalf("marked %v, want %v", got, c.want)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Fatalf("marked %v, want %v", got, c.want)
				}
			}
		})
	}
}

// Two batches are never joined: a helper's next batch, or another helper's,
// doesn't draw a line from the last batch's end.
func TestMarksBatchesAreNotJoined(t *testing.T) {
	m := NewMarks()
	m.Add([][2]int{{0, 0}}, 20, 10, t0)
	m.Add([][2]int{{9, 5}}, 20, 10, t0)
	if got := markedCells(m); len(got) != 2 {
		t.Fatalf("separate batches were joined: %v", got)
	}
}

func TestMarksDropStatusRowAndOutOfGrid(t *testing.T) {
	m := NewMarks()
	// Row 9 is the status row of a 10-row grid; a line through it keeps
	// only the cells above it.
	m.Add([][2]int{{0, 8}, {0, 9}, {-1, 2}, {20, 2}, {5, 10}}, 20, 10, t0)
	for _, c := range markedCells(m) {
		if c[1] >= 9 || c[0] < 0 || c[0] >= 20 {
			t.Fatalf("marked %v outside the drawable grid", c)
		}
	}
}

func TestMarksCoverageCapDropsOldest(t *testing.T) {
	m := NewMarks()
	cols, rows := 16, 8 // cap: 16*8/8 = 16 cells
	for i := range 20 {
		m.Add([][2]int{{i % cols, i / cols}}, cols, rows, t0.Add(time.Duration(i)*time.Millisecond))
	}
	got := markedCells(m)
	if len(got) != 16 {
		t.Fatalf("%d cells marked, cap is 16", len(got))
	}
	for _, c := range got {
		if c[1] == 0 && c[0] < 4 {
			t.Fatalf("oldest cell %v survived past the cap", c)
		}
	}
}

func TestMarksOverlayTintsBackgroundsAndFades(t *testing.T) {
	f := textFrame(t, 10, 4, "abcdefghij\nklmnopqrst", liveChrome(""))
	m := NewMarks()
	m.Add([][2]int{{2, 1}}, 10, 4, t0)
	idx := 1*10 + 2
	var seen []color.Color
	for _, age := range []time.Duration{0, 1500 * time.Millisecond, 2500 * time.Millisecond} {
		o := m.Overlay(f, t0.Add(age))
		c := o.Cells[idx]
		if c.Content != f.Cells[idx].Content || c.Style.Fg != f.Cells[idx].Style.Fg {
			t.Fatalf("age %v: overlay changed text or foreground", age)
		}
		if c.Style.Bg == f.Cells[idx].Style.Bg {
			t.Fatalf("age %v: no tint", age)
		}
		seen = append(seen, c.Style.Bg)
		for i := range o.Cells {
			if i != idx && !o.Cells[i].Equal(&f.Cells[i]) {
				t.Fatalf("age %v: overlay touched cell %d", age, i)
			}
		}
	}
	if seen[0] == seen[1] || seen[1] == seen[2] {
		t.Fatalf("no fade steps: %v", seen)
	}
	if o := m.Overlay(f, t0.Add(3*time.Second)); !o.Cells[idx].Equal(&f.Cells[idx]) {
		t.Fatal("mark still drawn after 3s")
	}
}

// With no live marks the overlay returns the frame itself, no copy.
func TestMarksOverlayWithoutMarksIsIdentity(t *testing.T) {
	f := textFrame(t, 10, 4, "abc", liveChrome(""))
	m := NewMarks()
	o := m.Overlay(f, t0)
	if &o.Cells[0] != &f.Cells[0] {
		t.Fatal("overlay copied a frame it didn't change")
	}
	m.Add([][2]int{{0, 0}}, 10, 4, t0)
	if o := m.Overlay(f, t0.Add(5*time.Second)); &o.Cells[0] != &f.Cells[0] {
		t.Fatal("overlay copied a frame whose marks had all expired")
	}
	if o := m.Overlay(f, t0); &o.Cells[0] == &f.Cells[0] {
		t.Fatal("overlay tinted the caller's frame in place")
	}
}

func TestMarksOverlayCoversWideCellContinuation(t *testing.T) {
	f := textFrame(t, 10, 3, "a界b", liveChrome(""))
	m := NewMarks()
	m.Add([][2]int{{1, 0}}, 10, 3, t0)
	o := m.Overlay(f, t0)
	if o.Cells[1].Style.Bg == f.Cells[1].Style.Bg || o.Cells[2].Style.Bg == f.Cells[2].Style.Bg {
		t.Fatal("wide character's two cells not tinted alike")
	}
	if o.Cells[1].Style.Bg != o.Cells[2].Style.Bg {
		t.Fatal("wide character's cells tinted differently")
	}
}

func TestMarksLiveAndNextChange(t *testing.T) {
	m := NewMarks()
	if m.Live(t0) {
		t.Fatal("empty marks are live")
	}
	if _, ok := m.NextChange(t0); ok {
		t.Fatal("empty marks schedule a repaint")
	}
	m.Add([][2]int{{0, 0}}, 10, 4, t0)
	if !m.Live(t0.Add(2900*time.Millisecond)) || m.Live(t0.Add(3*time.Second)) {
		t.Fatal("liveness doesn't match the 3s fade")
	}
	if d, ok := m.NextChange(t0.Add(300 * time.Millisecond)); !ok || d != 700*time.Millisecond {
		t.Fatalf("next fade step in %v %v, want 700ms", d, ok)
	}
	m.Clear()
	if m.Live(t0) {
		t.Fatal("Clear left marks")
	}
}

// Overlay leaves a private-class decision to its caller; it only tints.
var _ = terminal.FramePrivate
