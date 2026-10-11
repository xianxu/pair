package couchtty

import (
	"image/color"
	"strconv"
	"strings"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/rowtext"
	"github.com/xianxu/pair/cmd/internal/terminal"
)

// The terminal's notes menu (#447 M4): clicking 💭N in the status row opens a
// box over the bottom right of the screen listing the broadcast's sticky
// notes, each with a chip of its author's color. Hovering an item highlights
// it (and the console makes its 💭 glow); clicking it copies its text.
const (
	notesMenuMaxWidth = 48
	notesMenuItemRows = 3
	notesMenuClose    = "×"
)

var (
	notesMenuBg    = ansi.IndexedColor(236)
	notesMenuHover = ansi.IndexedColor(239)
	notesMenuDim   = ansi.IndexedColor(245)
)

// notesMenuItem is one note's rows in the menu: [Top, Bottom) on the screen.
type notesMenuItem struct {
	ID          int
	Top, Bottom int
}

// NotesMenu is the laid-out menu: its box on the screen (zero-based,
// half-open), its items, and the rows it draws. The zero value is no menu.
type NotesMenu struct {
	X, Y, W, H int
	items      []notesMenuItem
	lines      []notesMenuLine
}

type notesMenuLine struct {
	text  string
	chip  bool // the line starts an item: two cells of the author's color
	color int
	id    int // 0 for the header and the "+k more" line
	dim   bool
}

// Open reports whether there is a menu.
func (m NotesMenu) Open() bool { return m.W > 0 && m.H > 0 }

// Contains reports whether a zero-based screen cell is inside the box.
func (m NotesMenu) Contains(x, y int) bool {
	return m.Open() && x >= m.X && x < m.X+m.W && y >= m.Y && y < m.Y+m.H
}

// ItemAt is the note whose rows cover a zero-based screen cell, or 0.
func (m NotesMenu) ItemAt(x, y int) int {
	if !m.Contains(x, y) {
		return 0
	}
	for _, it := range m.items {
		if y >= it.Top && y < it.Bottom {
			return it.ID
		}
	}
	return 0
}

// CloseAt reports whether a zero-based screen cell is the close mark.
func (m NotesMenu) CloseAt(x, y int) bool {
	return m.Open() && y == m.Y && x >= m.X+m.W-3 && x < m.X+m.W
}

// RenderNotesMenu lays the menu out over a cols×rows screen whose last row is
// the status row, which it never covers. It shows as many notes as fit, each
// on at most notesMenuItemRows rows (the last cut with …; a copy still takes
// the whole text), and says how many more there are. With no notes, or no
// room, there is no menu.
func RenderNotesMenu(notes []broadcast.Note, cols, rows int) NotesMenu {
	w := min(notesMenuMaxWidth, cols-2)
	maxH := rows - 3
	textW := w - 5 // a space, the two-cell chip, a space … a space
	if len(notes) == 0 || textW < 4 || maxH < 2 {
		return NotesMenu{}
	}
	lines := []notesMenuLine{{text: " Notes"}}
	var items []notesMenuItem
	shown := 0
	for _, n := range notes {
		wrapped := wrapNote(n.Text, textW)
		need := len(wrapped)
		more := len(notes) - shown - 1
		reserve := 0
		if more > 0 {
			reserve = 1
		}
		if len(lines)+need+reserve > maxH {
			break
		}
		top := len(lines)
		for i, l := range wrapped {
			lines = append(lines, notesMenuLine{text: l, chip: i == 0, color: n.Color, id: n.ID})
		}
		items = append(items, notesMenuItem{ID: n.ID, Top: top, Bottom: len(lines)})
		shown++
	}
	if shown == 0 {
		return NotesMenu{}
	}
	if rest := len(notes) - shown; rest > 0 {
		lines = append(lines, notesMenuLine{text: " +" + strconv.Itoa(rest) + " more", dim: true})
	}
	h := len(lines)
	m := NotesMenu{X: cols - w, Y: rows - 1 - h, W: w, H: h, lines: lines}
	for _, it := range items {
		m.items = append(m.items, notesMenuItem{ID: it.ID, Top: m.Y + it.Top, Bottom: m.Y + it.Bottom})
	}
	return m
}

// wrapNote wraps a note's text to width, at most notesMenuItemRows lines,
// the last ending in … when the text goes on. The text is sanitized again:
// it is already plain, but it is drawn into the operator's terminal.
func wrapNote(text string, width int) []string {
	var out []string
	for _, para := range strings.Split(text, "\n") {
		para = rowtext.Sanitize(strings.ReplaceAll(para, "\t", " "))
		wrapped := ansi.Wrap(para, width, " -")
		out = append(out, strings.Split(wrapped, "\n")...)
	}
	if len(out) > notesMenuItemRows {
		last := rowtext.Fit(out[notesMenuItemRows-1], width-1)
		out = append(out[:notesMenuItemRows-1], last+"…")
	}
	return out
}

// Draw returns f with the menu over it; hover is the highlighted note.
// Wide characters the box's left edge cuts in half lose their other half.
func (m NotesMenu) Draw(f terminal.Frame, hover int, style broadcast.NoteStyle) terminal.Frame {
	cols, rows := f.Geometry.Cols, f.Geometry.Rows
	if !m.Open() || m.X+m.W > cols || m.Y+m.H > rows-1 || m.Y < 0 || len(f.Cells) != cols*rows {
		return f
	}
	out := f.Clone()
	for r, line := range m.lines {
		y := m.Y + r
		row := out.Cells[y*cols : (y+1)*cols]
		if m.X > 0 && row[m.X].Width == 0 && row[m.X-1].Width == 2 {
			row[m.X-1].Content, row[m.X-1].Width = " ", 1
		}
		var bg color.Color = notesMenuBg
		if line.id != 0 && line.id == hover {
			bg = notesMenuHover
		}
		cells := menuLineCells(line, m.W, bg, style)
		copy(row[m.X:m.X+m.W], cells)
		if y < len(out.Rows) {
			out.Rows[y].UsedColumns = cols
		}
	}
	return out
}

// menuLineCells is one line of the box, exactly w cells wide.
func menuLineCells(line notesMenuLine, w int, bg color.Color, style broadcast.NoteStyle) []terminal.Cell {
	cells := make([]terminal.Cell, 0, w)
	put := func(s string, st uv.Style) {
		for s != "" && len(cells) < w {
			g, width := ansi.FirstGraphemeCluster(s, ansi.GraphemeWidth)
			s = s[len(g):]
			if width == 0 {
				continue
			}
			if len(cells)+width > w {
				break
			}
			cells = append(cells, terminal.Cell{Content: g, Width: width, Style: st})
			if width == 2 {
				cells = append(cells, terminal.Cell{Style: st})
			}
		}
	}
	plain := uv.Style{Bg: bg}
	switch {
	case line.id == 0 && !line.dim: // the header, with the close mark at the right
		put(line.text, uv.Style{Bg: bg, Attrs: uv.AttrBold})
		for len(cells) < w-2 {
			put(" ", plain)
		}
		put(notesMenuClose, plain)
	case line.dim:
		put(line.text, uv.Style{Bg: bg, Fg: notesMenuDim})
	default:
		put(" ", plain)
		if line.chip {
			chip := uv.Style{Bg: broadcast.NoteChipColor(line.color, style)}
			put("  ", chip)
		} else {
			put("  ", plain)
		}
		put(" ", plain)
		put(line.text, plain)
	}
	for len(cells) < w {
		put(" ", plain)
	}
	return cells
}
