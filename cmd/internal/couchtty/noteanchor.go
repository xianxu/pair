package couchtty

import (
	"strings"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/terminal"
)

// A terminal note's anchor (#488): Couch knows its own screen exactly, so a
// note's box resolves to where it is on it and what it covers, the
// terminal's counterpart of the web cast's element anchor. Pure: the screen
// is a frame and a ScreenLayout captured by console_noteanchor.go.

// maxAnchorText bounds the text under a box an anchor carries, in bytes.
const maxAnchorText = 1024

// ScreenLayout is what Couch knows of its screen when a note is placed.
type ScreenLayout struct {
	// Chips carry zero-based rows relative to the declared chrome surface.
	Chips []ChipSpan
	// Slot is the label of the slot in view (its chip's, without the
	// status glyph), "" when there is none.
	Slot string
}

// AnchorFor is the anchor of a note's box on frame f: the part of Couch's
// screen holding the box's centre, the steps down to it, and the text
// under the box.
//   - The declared chrome rows: the tab bar (its chips), or the status line (the
//     broadcast's and Chrome's controls).
//   - Above it: the slot in view, with kind AnchorScreen. Native pane roles,
//     files and line numbers cannot be established from the displayed frame.
func AnchorFor(box broadcast.CellBox, f terminal.Frame, l ScreenLayout) broadcast.NoteAnchor {
	cols, rows := f.Geometry.Cols, f.Geometry.Rows
	a := broadcast.NoteAnchor{Kind: broadcast.AnchorScreen, Text: TextUnder(f, box)}
	if cols <= 0 || rows <= 0 {
		return a
	}
	cx, cy := box.Col+box.W/2, box.Row+box.H/2
	top := rows - f.ChromeRows
	if f.ChromeRows > 0 && f.ChromeRows < rows && cy >= top && cy < rows {
		row := cy - top
		var rowChips []ColumnSpan
		var tabs []string
		for _, s := range l.Chips {
			if s.Row != row {
				continue
			}
			rowChips = append(rowChips, ColumnSpan{Start: s.Start, End: s.End})
			if s.Start < box.Col+box.W && box.Col < s.End {
				tabs = append(tabs, strings.TrimSpace(rowText(f, cy, s.Start, s.End)))
			}
		}
		if len(tabs) > 0 && inChips(cx, rowChips) {
			a.Kind, a.Path = broadcast.AnchorTabBar, []string{"tab bar", "tabs: " + strings.Join(tabs, ", ")}
			return a
		}
		a.Kind, a.Path = broadcast.AnchorStatusLine, []string{"status line"}
		return a
	}
	slot := l.Slot
	if slot == "" {
		slot = "the slot in view"
	}
	a.Path = []string{"pane " + slot}
	return a
}

func inChips(col int, chips []ColumnSpan) bool {
	return len(chips) > 0 && col >= chips[0].Start && col < chips[len(chips)-1].End
}

// TextUnder is the screen's text under box: each row's cells within it,
// trailing spaces trimmed, rows joined by newlines, blank edges dropped,
// cut to maxAnchorText bytes at a character boundary.
func TextUnder(f terminal.Frame, box broadcast.CellBox) string {
	rows := f.Geometry.Rows
	var lines []string
	for row := max(box.Row, 0); row < min(box.Row+box.H, rows); row++ {
		lines = append(lines, rowText(f, row, box.Col, box.Col+box.W))
	}
	s := strings.Trim(strings.Join(lines, "\n"), "\n")
	if len(s) > maxAnchorText {
		cut := maxAnchorText
		for cut > 0 && !utf8.RuneStart(s[cut]) {
			cut--
		}
		s = s[:cut]
	}
	return s
}

// rowText is row's text in columns [start, end), trimmed of trailing
// spaces (leading ones are indentation). A wide character counts where it
// starts.
func rowText(f terminal.Frame, row, start, end int) string {
	cols := f.Geometry.Cols
	if row < 0 || row >= f.Geometry.Rows || len(f.Cells) != cols*f.Geometry.Rows {
		return ""
	}
	var b strings.Builder
	for col := max(start, 0); col < min(end, cols); col++ {
		c := f.Cells[row*cols+col]
		switch {
		case c.Width == 0 && col > 0 && f.Cells[row*cols+col-1].Width == 2:
			// The second half of a wide character.
		case c.Content == "":
			b.WriteByte(' ')
		default:
			b.WriteString(c.Content)
		}
	}
	return strings.TrimRight(b.String(), " ")
}
