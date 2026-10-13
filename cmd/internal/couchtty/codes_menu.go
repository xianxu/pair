package couchtty

import (
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/xianxu/pair/cmd/internal/terminal"
)

// The Codes menu (pair#495): one row per cast link, its label and code plain,
// then its actions -- each word its own click target, drawn in the link blue:
//
//	view 3926 · rotate · copy-link
//	pointer 8538 · rotate · copy-link
//	rotate all
//
// Up/Down pick the row, Left/Right the action, Enter runs it. A link that
// doesn't exist yet (the pointer link before pointing first starts) is a row
// with its label alone and no actions. Pure: layout, hit-testing, selection
// and drawing; the console runs what is chosen.

// codesAction is what a Codes menu word does: copy a link, or rotate a link's
// code ("" rotates every link).
type codesAction struct {
	copy bool
	link string
}

// codesMenuRow is one row as the console asks for it.
type codesMenuRow struct {
	label   string // "view 3926", or "pointer" with no code yet
	actions []codesMenuWord
}

// codesMenuWord is one clickable word and what it does.
type codesMenuWord struct {
	text   string
	action codesAction
}

// codesMenu is the laid-out menu: its box (zero-based, half-open), its rows
// with each word's columns, and the selection. The zero value is no menu.
type codesMenu struct {
	X, Y, W, H int
	title      string
	rows       []codesMenuLaidRow
	sel        codesMenuSel
}

type codesMenuLaidRow struct {
	y     int
	label string
	words []codesMenuLaidWord
}

type codesMenuLaidWord struct {
	codesMenuWord
	x0, x1 int // screen columns [x0, x1)
}

// codesMenuSel is the selected word: row and word index; row -1 is none.
type codesMenuSel struct{ row, word int }

const codesMenuSep = " · "

// layoutCodesMenu lays the menu out over the bottom right of a cols×rows
// screen, above its status row. With no room there is no menu.
func layoutCodesMenu(title string, rows []codesMenuRow, cols, screenRows int) codesMenu {
	cells := func(s string) int { return ansi.StringWidth(s) }
	width := cells(" " + title + " ×")
	for _, r := range rows {
		w := 1 + cells(r.label)
		for _, a := range r.actions {
			w += cells(codesMenuSep) + cells(a.text)
		}
		width = max(width, w+1)
	}
	h := len(rows) + 1
	if width > cols || h > screenRows-1 || len(rows) == 0 {
		return codesMenu{}
	}
	m := codesMenu{X: cols - width, Y: screenRows - 1 - h, W: width, H: h, title: title, sel: codesMenuSel{row: -1}}
	for i, r := range rows {
		laid := codesMenuLaidRow{y: m.Y + 1 + i, label: r.label}
		x := m.X + 1 + cells(r.label)
		for _, a := range r.actions {
			x += cells(codesMenuSep)
			n := cells(a.text)
			laid.words = append(laid.words, codesMenuLaidWord{codesMenuWord: a, x0: x, x1: x + n})
			x += n
		}
		m.rows = append(m.rows, laid)
	}
	return m
}

// Open reports whether there is a menu.
func (m codesMenu) Open() bool { return m.W > 0 && m.H > 0 }

// Contains reports whether a zero-based screen cell is inside the box.
func (m codesMenu) Contains(x, y int) bool {
	return m.Open() && x >= m.X && x < m.X+m.W && y >= m.Y && y < m.Y+m.H
}

// CloseAt reports whether a zero-based screen cell is the close mark.
func (m codesMenu) CloseAt(x, y int) bool {
	return m.Open() && y == m.Y && x >= m.X+m.W-3 && x < m.X+m.W
}

// WordAt is the word under a zero-based screen cell.
func (m codesMenu) WordAt(x, y int) (codesMenuSel, bool) {
	for i, r := range m.rows {
		if r.y != y {
			continue
		}
		for j, w := range r.words {
			if x >= w.x0 && x < w.x1 {
				return codesMenuSel{i, j}, true
			}
		}
	}
	return codesMenuSel{row: -1}, false
}

// Selected is the selected word's action.
func (m codesMenu) Selected() (codesAction, bool) {
	if m.sel.row < 0 || m.sel.row >= len(m.rows) || m.sel.word >= len(m.rows[m.sel.row].words) {
		return codesAction{}, false
	}
	return m.rows[m.sel.row].words[m.sel.word].action, true
}

// Select sets the selection; an unknown one clears it.
func (m codesMenu) Select(s codesMenuSel) codesMenu {
	if s.row < 0 || s.row >= len(m.rows) || s.word < 0 || s.word >= len(m.rows[s.row].words) {
		s = codesMenuSel{row: -1}
	}
	m.sel = s
	return m
}

// Move steps the selection: dy picks the row (skipping rows with no actions),
// dx the word on it. With nothing selected, any move selects the first word.
func (m codesMenu) Move(dy, dx int) codesMenu {
	if m.sel.row < 0 {
		for i, r := range m.rows {
			if len(r.words) > 0 {
				return m.Select(codesMenuSel{i, 0})
			}
		}
		return m
	}
	if dx != 0 {
		words := len(m.rows[m.sel.row].words)
		m.sel.word = min(max(m.sel.word+dx, 0), words-1)
		return m
	}
	for row := m.sel.row + dy; row >= 0 && row < len(m.rows); row += dy {
		if n := len(m.rows[row].words); n > 0 {
			return m.Select(codesMenuSel{row, min(m.sel.word, n-1)})
		}
	}
	return m
}

// Draw returns f with the menu over it; pressed is a word just run, drawn
// inverted (its click's feedback). Wide characters the box's left edge cuts
// in half lose their other half.
func (m codesMenu) Draw(f terminal.Frame, pressed codesMenuSel) terminal.Frame {
	cols, rows := f.Geometry.Cols, f.Geometry.Rows
	if !m.Open() || m.X < 0 || m.X+m.W > cols || m.Y < 0 || m.Y+m.H > rows-1 || len(f.Cells) != cols*rows {
		return f
	}
	out := f.Clone()
	plain := uv.Style{Bg: notesMenuBg}
	paint := func(y int, cells []terminal.Cell) {
		row := out.Cells[y*cols : (y+1)*cols]
		if m.X > 0 && row[m.X].Width == 0 && row[m.X-1].Width == 2 {
			row[m.X-1].Content, row[m.X-1].Width = " ", 1
		}
		copy(row[m.X:m.X+m.W], cells)
		if y < len(out.Rows) {
			out.Rows[y].UsedColumns = cols
		}
	}
	line := func() []terminal.Cell {
		cells := make([]terminal.Cell, m.W)
		for i := range cells {
			cells[i] = terminal.Cell{Content: " ", Width: 1, Style: plain}
		}
		return cells
	}
	// put writes text from column x by grapheme, a wide one taking its two
	// cells (the second empty), and stops at the box's edge.
	put := func(cells []terminal.Cell, x int, text string, st uv.Style) int {
		for text != "" {
			g, w := ansi.FirstGraphemeCluster(text, ansi.GraphemeWidth)
			text = text[len(g):]
			if w == 0 {
				continue
			}
			if x+w > len(cells) {
				break
			}
			cells[x] = terminal.Cell{Content: g, Width: w, Style: st}
			if w == 2 {
				cells[x+1] = terminal.Cell{Style: st}
			}
			x += w
		}
		return x
	}
	head := line()
	put(head, 1, m.title, uv.Style{Bg: notesMenuBg, Attrs: uv.AttrBold})
	put(head, m.W-2, notesMenuClose, plain)
	paint(m.Y, head)
	for i, r := range m.rows {
		cells := line()
		x := put(cells, 1, r.label, plain)
		for j, w := range r.words {
			x = put(cells, x, codesMenuSep, plain)
			st := uv.Style{Bg: notesMenuBg, Fg: notesMenuLink}
			switch {
			case pressed == (codesMenuSel{i, j}):
				st = uv.Style{Bg: notesMenuLink, Fg: notesMenuBg, Attrs: uv.AttrBold}
			case m.sel == (codesMenuSel{i, j}):
				st = uv.Style{Bg: notesMenuHover, Fg: notesMenuLink, Attrs: uv.AttrBold}
			}
			x = put(cells, x, w.text, st)
		}
		paint(r.y, cells)
	}
	return out
}
