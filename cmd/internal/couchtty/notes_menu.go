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
	// notesMenuLink is the Codes menu's action words (pair#495): the
	// actionable-link blue.
	notesMenuLink = ansi.IndexedColor(75)
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
	tip   bool // a tooltip's one plain line, no header marks (pair#495)
	// chipBg is a chip's own color, for items with no author palette (the
	// attention list's kinds, pair#673); nil uses color.
	chipBg color.Color

	accents []bool // source byte positions carrying the bell reference accent
}

// menuItem is one entry of a titled menu, with its color chip or none. Bg,
// when set, is the chip's color instead of the author palette's Color.
type menuItem struct {
	ID    int
	Text  string
	Color int
	Chip  bool
	Bg    color.Color
	refs  bool // bell-only reference accents; Text remains plain
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
	return RenderTitledMenu("Notes", notes, cols, rows)
}

// RenderTitledMenu is the menu box with its title: the notes menu's, or remote
// driving's (#467), whose items are its driver and askers.
func RenderTitledMenu(title string, notes []broadcast.Note, cols, rows int) NotesMenu {
	items := make([]menuItem, len(notes))
	for i, n := range notes {
		items[i] = menuItem{ID: n.ID, Text: n.Text, Color: n.Color, Chip: true}
	}
	return renderMenu(title, items, cols, rows)
}

// renderMenu lays out a titled menu of items over the bottom right of a
// cols×rows screen, above its status row.
func renderMenu(title string, notes []menuItem, cols, rows int) NotesMenu {
	return renderMenuAt(title, notes, cols, rows, false)
}

// renderMenuAt lays the menu out at the bottom right, or with top at the top
// right (the attention list, pair#673); either way above the status row.
func renderMenuAt(title string, notes []menuItem, cols, rows int, top bool) NotesMenu {
	w := min(notesMenuMaxWidth, cols-2)
	maxH := rows - 3
	textW := w - 5 // a space, the two-cell chip, a space … a space
	if len(notes) == 0 || textW < 4 || maxH < 2 {
		return NotesMenu{}
	}
	lines := []notesMenuLine{{text: " " + title}}
	var items []notesMenuItem
	shown := 0
	for _, n := range notes {
		source := sanitizedNote(n.Text)
		wrapped := wrapNote(source, textW)
		var accents []bool
		if n.refs {
			accents = referenceAccents(source, referenceToken)
		}
		offset := 0
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
			lines = append(lines, notesMenuLine{text: l, chip: n.Chip && i == 0, color: n.Color, chipBg: n.Bg, id: n.ID, accents: projectReferenceAccents(source, l, accents, &offset)})
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
	y := rows - 1 - h
	if top {
		y = 0
	}
	m := NotesMenu{X: cols - w, Y: y, W: w, H: h, lines: lines}
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
	for _, para := range strings.Split(sanitizedNote(text), "\n") {
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
	put := func(s string, st uv.Style, accents ...[]bool) {
		offset := 0
		for s != "" && len(cells) < w {
			g, width := ansi.FirstGraphemeCluster(s, ansi.GraphemeWidth)
			s = s[len(g):]
			if width == 0 {
				offset += len(g)
				continue
			}
			if len(cells)+width > w {
				break
			}
			cellStyle := st
			if len(accents) > 0 && offset < len(accents[0]) && accents[0][offset] {
				cellStyle.Fg = notesMenuLink
			}
			offset += len(g)
			cells = append(cells, terminal.Cell{Content: g, Width: width, Style: cellStyle})
			if width == 2 {
				cells = append(cells, terminal.Cell{Style: cellStyle})
			}
		}
	}
	plain := uv.Style{Bg: bg}
	switch {
	case line.tip:
		put(line.text, plain)
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
			if line.chipBg != nil {
				chip.Bg = line.chipBg
			}
			put("  ", chip)
		} else {
			put("  ", plain)
		}
		put(" ", plain)
		put(line.text, plain, line.accents)
	}
	for len(cells) < w {
		put(" ", plain)
	}
	return cells
}
