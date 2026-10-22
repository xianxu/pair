package broadcast

import (
	"fmt"
	"image/color"
	"sort"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/xianxu/pair/cmd/internal/terminal"
)

// A mark holds at its full color, then fades fast (#412: the operator found a
// slow, stepped fade clumsy).
const (
	MarkHold      = 1500 * time.Millisecond
	MarkFade      = 500 * time.Millisecond
	MarkLife      = MarkHold + MarkFade
	markFadeSteps = 10
	markFadeStep  = MarkFade / markFadeSteps
)

// markColor is one pointer color (#446): its truecolor value, its xterm-256
// index, and its fade without truecolor, darker steps toward a dark
// background.
type markColor struct {
	rgb    color.RGBA
	index  ansi.IndexedColor
	ladder [5]ansi.IndexedColor
}

// markPalette is the pointer's six colors, Okabe–Ito based so they stay
// apart for colorblind viewers. Index 0, orange, is the default and keeps
// the amber the pointer always drew (xterm 214). A batch names its color by
// index, so nothing a viewer sends reaches the terminal as a raw color.
var markPalette = [...]markColor{
	{color.RGBA{0xff, 0xaf, 0x00, 0xff}, 214, [5]ansi.IndexedColor{178, 172, 136, 94, 58}},  // orange
	{color.RGBA{0x56, 0xb4, 0xe9, 0xff}, 74, [5]ansi.IndexedColor{67, 31, 24, 23, 17}},      // sky blue
	{color.RGBA{0x00, 0x9e, 0x73, 0xff}, 36, [5]ansi.IndexedColor{35, 29, 23, 22, 234}},     // bluish green
	{color.RGBA{0xf0, 0xe4, 0x42, 0xff}, 221, [5]ansi.IndexedColor{185, 142, 100, 58, 236}}, // yellow
	{color.RGBA{0xd5, 0x5e, 0x00, 0xff}, 166, [5]ansi.IndexedColor{130, 94, 88, 52, 233}},   // vermillion
	{color.RGBA{0xcc, 0x79, 0xa7, 0xff}, 175, [5]ansi.IndexedColor{139, 132, 96, 89, 53}},   // reddish purple
}

// PointerColors is how many colors a batch can name: 0..PointerColors-1.
const PointerColors = len(markPalette)

// PointerPalette is the colors as #rrggbb, in index order, for the pointer
// page's swatches.
func PointerPalette() []string {
	out := make([]string, PointerColors)
	for i, c := range markPalette {
		out[i] = fmt.Sprintf("#%02x%02x%02x", c.rgb.R, c.rgb.G, c.rgb.B)
	}
	return out
}

type cell struct{ col, row int }

// guarded reports whether c is one of the broadcast's controls (the first
// StatusGuardCols columns of the first chrome row), which marks must never cover.
func guarded(c cell, rows, chromeRows int) bool {
	return chromeRows <= 0 || chromeRows >= rows || (c.row == rows-chromeRows && c.col < StatusGuardCols)
}

// mark is when a cell was marked, and in which palette color.
type mark struct {
	at    time.Time
	color int
}

// Marks is the remote pointer's state: which cells a helper marked, when,
// and in which color. Pure: the caller supplies the clock, and owns locking.
type Marks struct {
	cells map[cell]mark
	// With truecolor and a known background, the fade blends the mark's
	// color into the operator's real background; otherwise it walks the
	// color's ladder.
	blend bool
	bg    color.RGBA
}

// SetBlend gives the fade the operator's background, for terminals with
// truecolor.
func (m *Marks) SetBlend(bg color.RGBA, truecolor bool) { m.bg, m.blend = bg, truecolor }

// tint is the background of a mark in palette color c at age: full color
// while it holds, then each fade step a little further toward the
// background.
func (m *Marks) tint(c int, age time.Duration) color.Color {
	if age < MarkHold {
		return paletteTint(c, 0, m.bg, m.blend)
	}
	k := int((age - MarkHold) / markFadeStep) // 0..markFadeSteps-1
	return paletteTint(c, float64(k+1)/float64(markFadeSteps+1), m.bg, m.blend)
}

// paletteTint is palette color c weakened by t, from 0 (the full color) to
// 1 (the background): blended into bg with truecolor, or else a step down
// the color's ladder. It is the one place marks and notes (#447) fade.
func paletteTint(c int, t float64, bg color.RGBA, blend bool) color.Color {
	p := markPalette[c]
	if t <= 0 {
		if blend {
			return p.rgb
		}
		return p.index
	}
	if !blend {
		return p.ladder[min(int(t*float64(len(p.ladder))), len(p.ladder)-1)]
	}
	mix := func(a, b uint8) uint8 { return uint8(float64(a) + (float64(b)-float64(a))*t + 0.5) }
	return color.RGBA{mix(p.rgb.R, bg.R), mix(p.rgb.G, bg.G), mix(p.rgb.B, bg.B), 0xff}
}

func NewMarks() *Marks { return &Marks{cells: make(map[cell]mark)} }

// Add marks the cells of one batch of points on a cols×rows grid at now, in
// palette color c (out of range draws the default).
// Consecutive points within the batch are joined by a line; separate batches
// never are. Points off the grid and on the broadcast's controls (the left of
// the first chrome row) are dropped; the rest of the tab bar can be marked. At most cols*rows/8 cells stay marked, the oldest dropped first.
//
// Its cost is bounded (it runs under the lock the paint path takes): a batch
// contributes at most the cap's worth of cells, its last ones, and eviction
// is one sort per call.
func (m *Marks) Add(points [][2]int, cols, rows, chromeRows, c int, now time.Time) {
	if c < 0 || c >= PointerColors {
		c = 0
	}
	limit := max(1, cols*rows/8)
	var batch []cell
	walkStrokes(points, cols, rows, func(c cell) {
		if c.col >= 0 && c.col < cols && c.row >= 0 && c.row < rows && !guarded(c, rows, chromeRows) {
			batch = append(batch, c)
		}
		// Keep only the cap's worth, the latest, as the batch grows.
		if len(batch) > 2*limit {
			batch = append(batch[:0], batch[len(batch)-limit:]...)
		}
	})
	if len(batch) > limit {
		batch = batch[len(batch)-limit:]
	}
	m.prune(now)
	for _, at := range batch {
		m.cells[at] = mark{now, c}
	}
	if excess := len(m.cells) - limit; excess > 0 {
		type aged struct {
			c  cell
			at time.Time
		}
		all := make([]aged, 0, len(m.cells))
		for c, mk := range m.cells {
			all = append(all, aged{c, mk.at})
		}
		sort.Slice(all, func(i, j int) bool {
			a, b := all[i], all[j]
			if !a.at.Equal(b.at) {
				return a.at.Before(b.at)
			}
			if a.c.row != b.c.row {
				return a.c.row < b.c.row
			}
			return a.c.col < b.c.col
		})
		for _, a := range all[:excess] {
			delete(m.cells, a.c)
		}
	}
}

// walkStrokes visits the cells a batch's points draw: each on-grid point,
// joined to the previous one by a line. An off-grid point is dropped before
// any line reaches it and breaks the stroke, so a line is never longer than
// the grid, whatever the caller passed: the walk is at most one grid span
// per point.
func walkStrokes(points [][2]int, cols, rows int, visit func(cell)) {
	onGrid := func(p [2]int) bool { return p[0] >= 0 && p[0] < cols && p[1] >= 0 && p[1] < rows }
	var prev *[2]int
	for i := range points {
		p := points[i]
		if !onGrid(p) {
			prev = nil
			continue
		}
		if prev == nil {
			visit(cell{p[0], p[1]})
		} else {
			line(*prev, p, visit)
		}
		prev = &points[i]
	}
}

// line visits the cells from a to b (Bresenham).
func line(a, b [2]int, visit func(cell)) {
	x0, y0, x1, y1 := a[0], a[1], b[0], b[1]
	dx, dy := abs(x1-x0), -abs(y1-y0)
	sx, sy := 1, 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}
	err := dx + dy
	for {
		visit(cell{x0, y0})
		if x0 == x1 && y0 == y1 {
			return
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func (m *Marks) prune(now time.Time) {
	for c, mk := range m.cells {
		if now.Sub(mk.at) >= MarkLife {
			delete(m.cells, c)
		}
	}
}

// Clear removes every mark.
func (m *Marks) Clear() { clear(m.cells) }

// Live reports whether any mark is still drawn at now.
func (m *Marks) Live(now time.Time) bool {
	for _, mk := range m.cells {
		if now.Sub(mk.at) < MarkLife {
			return true
		}
	}
	return false
}

// NextChange is how long until the drawing next changes (a fade step or an
// expiry); ok is false when nothing is drawn.
func (m *Marks) NextChange(now time.Time) (time.Duration, bool) {
	var next time.Duration
	ok := false
	for _, mk := range m.cells {
		age := now.Sub(mk.at)
		if age >= MarkLife {
			continue
		}
		var d time.Duration
		if age < MarkHold {
			d = MarkHold - age
		} else {
			d = markFadeStep - (age-MarkHold)%markFadeStep
		}
		if !ok || d < next {
			next, ok = d, true
		}
	}
	return next, ok
}

// Overlay returns f with marked cells' backgrounds tinted by age; text and
// foreground are untouched, and both cells of a wide character are tinted
// alike. With nothing to draw it returns f itself, uncopied.
func (m *Marks) Overlay(f terminal.Frame, now time.Time) terminal.Frame {
	if !m.Live(now) {
		return f
	}
	cols, rows := f.Geometry.Cols, f.Geometry.Rows
	chromeRows := f.ChromeRows
	out := f.Clone()
	for c, mk := range m.cells {
		age := now.Sub(mk.at)
		// Never the broadcast's controls on the first chrome row, whose LIVE and
		// pointer indicators the fail-safes read (a resize can move a mark
		// there).
		if age >= MarkLife || c.col >= cols || c.row >= rows || guarded(c, rows, chromeRows) {
			continue
		}
		tintCell(out, c, m.tint(mk.color, age))
	}
	return out
}

// tintCell sets one cell's background; both cells of a wide character are
// tinted alike, a continuation cell belonging to the character before it.
func tintCell(f terminal.Frame, c cell, bg color.Color) {
	cols := f.Geometry.Cols
	i := c.row*cols + c.col
	if f.Cells[i].Width == 0 && c.col > 0 && f.Cells[i-1].Width == 2 {
		i--
	}
	if guardedGlyph(f, cell{i % cols, c.row}) {
		return
	}
	f.Cells[i].Style.Bg = bg
	if f.Cells[i].Width == 2 && i%cols+1 < cols {
		f.Cells[i+1].Style.Bg = bg
	}
}

// guardedGlyph protects the whole glyph touched by a tint or replacement,
// including a wide character whose leading cell is inside the controls.
func guardedGlyph(f terminal.Frame, c cell) bool {
	cols := f.Geometry.Cols
	i := c.row*cols + c.col
	if f.Cells[i].Width == 0 && c.col > 0 && f.Cells[i-1].Width == 2 {
		c.col--
		i--
	}
	return guarded(c, f.Geometry.Rows, f.ChromeRows) ||
		(f.Cells[i].Width == 2 && guarded(cell{c.col + 1, c.row}, f.Geometry.Rows, f.ChromeRows))
}
