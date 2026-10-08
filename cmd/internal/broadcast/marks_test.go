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

// The status row (row 9 of 10) can be marked, except the broadcast's own
// controls at its left; off-grid points are dropped.
func TestMarksGuardControlsAndGrid(t *testing.T) {
	m := NewMarks()
	m.Add([][2]int{{0, 8}, {0, 9}, {-1, 2}, {20, 2}, {5, 10}}, 20, 10, t0)
	m.Add([][2]int{{StatusGuardCols + 3, 9}}, 20, 10, t0)
	gotTabBar := false
	for _, c := range markedCells(m) {
		if c[0] < 0 || c[0] >= 20 || c[1] > 9 {
			t.Fatalf("marked %v outside the grid", c)
		}
		if c[1] == 9 && c[0] < StatusGuardCols {
			t.Fatalf("marked %v on the broadcast controls", c)
		}
		if c == [2]int{StatusGuardCols + 3, 9} {
			gotTabBar = true
		}
	}
	if !gotTabBar {
		t.Fatal("a point on the tab bar, right of the controls, was dropped")
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

// BR-3, narrowed by the #412 smoke: the overlay never tints the broadcast's
// controls on the last row, however a mark got there (here, a resize moved
// it), so marks can't hide LIVE or the pointer marker; the rest of the tab
// bar can be marked.
func TestMarksOverlayNeverTintsControls(t *testing.T) {
	m := NewMarks()
	m.Add([][2]int{{0, 7}, {StatusGuardCols - 1, 7}}, 20, 10, t0) // a line across the future controls
	m.Add([][2]int{{StatusGuardCols + 2, 7}}, 20, 10, t0)
	chrome := LiveSGR + LiveLabel + "\x1b[0m " + PointerSGR + PointerLabel + "\x1b[0m " + ControlLabel + " tabs"
	f := textFrame(t, 20, 8, "body", chrome)
	o := m.Overlay(f, t0)
	for i := range StatusGuardCols {
		if !o.Cells[7*20+i].Equal(&f.Cells[7*20+i]) {
			t.Fatalf("control cell %d tinted after a resize", i)
		}
	}
	if o.Cells[7*20+StatusGuardCols+2].Equal(&f.Cells[7*20+StatusGuardCols+2]) {
		t.Fatal("tab-bar cell right of the controls not tinted")
	}
	if !IndicatorShown(o) || !PointerShown(o) {
		t.Fatal("marks hid the LIVE or pointer indicator")
	}
}

// BR-2: Add stays cheap at the worst case, a full batch of far-apart points
// on a big grid with the cap already full, because it runs under the lock the
// paint path takes.
func TestMarksAddWorstCaseIsCheap(t *testing.T) {
	m := NewMarks()
	cols, rows := 300, 100
	var points [][2]int
	for i := range 64 {
		points = append(points, [2]int{(i * 97) % cols, (i * 31) % (rows - 1)})
	}
	for i := range 10 {
		m.Add(points, cols, rows, t0.Add(time.Duration(i)*time.Millisecond))
	}
	start := time.Now()
	for i := range 20 {
		m.Add(points, cols, rows, t0.Add(time.Duration(20+i)*time.Millisecond))
	}
	if per := time.Since(start) / 20; per > 20*time.Millisecond {
		t.Fatalf("Add took %v per batch at the cap", per)
	}
	if n := len(m.cells); n > cols*rows/8 {
		t.Fatalf("%d cells marked, cap %d", n, cols*rows/8)
	}
}

// Add's cost holds without trusting its caller: an off-grid point is dropped
// before any line is drawn to it, so a huge coordinate can't make it walk a
// huge line.
func TestMarksAddIgnoresHugeCoordinates(t *testing.T) {
	m := NewMarks()
	start := time.Now()
	m.Add([][2]int{{0, 0}, {1 << 30, 1 << 30}, {2, 0}}, 20, 10, t0)
	if d := time.Since(start); d > 50*time.Millisecond {
		t.Fatalf("Add took %v on a huge coordinate", d)
	}
	for _, c := range markedCells(m) {
		if c[0] >= 20 || c[1] >= 9 {
			t.Fatalf("marked %v", c)
		}
	}
}
