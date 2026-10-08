package broadcast

import (
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

// Marks is the remote pointer's state: which cells a helper marked, and when.
// Pure: the caller supplies the clock, and owns locking.
type Marks struct {
	cells map[cell]time.Time
}

func NewMarks() *Marks { return &Marks{cells: make(map[cell]time.Time)} }

// Add marks the cells of one batch of points on a cols×rows grid at now.
// Consecutive points within the batch are joined by a line; separate batches
// never are. The last row (Couch's status row) and points off the grid are
// dropped. At most cols*rows/8 cells stay marked, the oldest dropped first.
func (m *Marks) Add(points [][2]int, cols, rows int, now time.Time) {
	put := func(c cell) {
		if c.col >= 0 && c.col < cols && c.row >= 0 && c.row < rows-1 {
			m.cells[c] = now
		}
	}
	for i, p := range points {
		if i == 0 {
			put(cell{p[0], p[1]})
			continue
		}
		line(points[i-1], p, put)
	}
	m.prune(now)
	limit := max(1, cols*rows/8)
	for len(m.cells) > limit {
		var oldest cell
		var at time.Time
		first := true
		for c, t := range m.cells {
			if first || t.Before(at) || t.Equal(at) && (c.row < oldest.row || c.row == oldest.row && c.col < oldest.col) {
				oldest, at, first = c, t, false
			}
		}
		delete(m.cells, oldest)
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
		if age >= MarkLife || c.col >= cols || c.row >= rows {
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
