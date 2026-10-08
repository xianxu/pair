package broadcast

import (
	"sort"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/xianxu/pair/cmd/internal/terminal"
)

const (
	// MarkLife is how long a mark stays after its last input (#412).
	MarkLife = 3 * time.Second
	markStep = MarkLife / 3
)

// markTints fade a mark from strong to faint amber, one per markStep.
var markTints = [3]ansi.IndexedColor{214, 172, 94}

type cell struct{ col, row int }

// guarded reports whether c is one of the broadcast's controls (the first
// StatusGuardCols columns of the last row), which marks must never cover.
func guarded(c cell, rows int) bool { return c.row == rows-1 && c.col < StatusGuardCols }

// Marks is the remote pointer's state: which cells a helper marked, and when.
// Pure: the caller supplies the clock, and owns locking.
type Marks struct {
	cells map[cell]time.Time
}

func NewMarks() *Marks { return &Marks{cells: make(map[cell]time.Time)} }

// Add marks the cells of one batch of points on a cols×rows grid at now.
// Consecutive points within the batch are joined by a line; separate batches
// never are. Points off the grid and on the broadcast's controls (the left of
// the last row) are dropped; the rest of the tab bar can be marked. At most cols*rows/8 cells stay marked, the oldest dropped first.
//
// Its cost is bounded (it runs under the lock the paint path takes): a batch
// contributes at most the cap's worth of cells, its last ones, and eviction
// is one sort per call.
func (m *Marks) Add(points [][2]int, cols, rows int, now time.Time) {
	limit := max(1, cols*rows/8)
	var batch []cell
	put := func(c cell) {
		if c.col >= 0 && c.col < cols && c.row >= 0 && c.row < rows && !guarded(c, rows) {
			batch = append(batch, c)
		}
	}
	// An off-grid point is dropped before any line reaches it, so a line is
	// never longer than the grid, whatever the caller passed.
	onGrid := func(p [2]int) bool { return p[0] >= 0 && p[0] < cols && p[1] >= 0 && p[1] < rows }
	var prev *[2]int
	for i := range points {
		p := points[i]
		if !onGrid(p) {
			prev = nil
			continue
		}
		if prev == nil {
			put(cell{p[0], p[1]})
		} else {
			line(*prev, p, put)
		}
		// Keep only the cap's worth, the latest, as the batch grows.
		if len(batch) > 2*limit {
			batch = append(batch[:0], batch[len(batch)-limit:]...)
		}
		prev = &points[i]
	}
	if len(batch) > limit {
		batch = batch[len(batch)-limit:]
	}
	m.prune(now)
	for _, c := range batch {
		m.cells[c] = now
	}
	if excess := len(m.cells) - limit; excess > 0 {
		type aged struct {
			c  cell
			at time.Time
		}
		all := make([]aged, 0, len(m.cells))
		for c, t := range m.cells {
			all = append(all, aged{c, t})
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
	for c, t := range m.cells {
		if now.Sub(t) >= MarkLife {
			delete(m.cells, c)
		}
	}
}

// Clear removes every mark.
func (m *Marks) Clear() { clear(m.cells) }

// Live reports whether any mark is still drawn at now.
func (m *Marks) Live(now time.Time) bool {
	for _, t := range m.cells {
		if now.Sub(t) < MarkLife {
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
	for _, t := range m.cells {
		age := now.Sub(t)
		if age >= MarkLife {
			continue
		}
		d := markStep - age%markStep
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
	out := f.Clone()
	for c, t := range m.cells {
		age := now.Sub(t)
		// Never the broadcast's controls on the last row, whose LIVE and
		// pointer indicators the fail-safes read (a resize can move a mark
		// there).
		if age >= MarkLife || c.col >= cols || c.row >= rows || guarded(c, rows) {
			continue
		}
		tint := markTints[min(int(age/markStep), len(markTints)-1)]
		i := c.row*cols + c.col
		// A continuation cell belongs to the wide character before it.
		if out.Cells[i].Width == 0 && c.col > 0 && out.Cells[i-1].Width == 2 {
			i--
		}
		out.Cells[i].Style.Bg = tint
		if out.Cells[i].Width == 2 && c.col+1 < cols {
			out.Cells[i+1].Style.Bg = tint
		}
	}
	return out
}
