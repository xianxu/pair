package couchtty

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/terminal"
)

// A terminal note's anchor (#488): Couch knows its own screen exactly, so a
// note's box resolves to where it is on it and what it covers, the
// terminal's counterpart of the web cast's element anchor. Pure: the screen
// is a frame and a ScreenLayout; noteanchor_resolve.go is the shell that
// gathers the pane layout from zellij.

// maxAnchorText bounds the text under a box an anchor carries, in bytes.
const maxAnchorText = 1024

// ScreenLayout is what Couch knows of its screen when a note is placed.
type ScreenLayout struct {
	// Chips are the tab bar's chips on the last row.
	Chips []ColumnSpan
	// Slot is the label of the slot in view (its chip's, without the
	// status glyph), "" when there is none.
	Slot string
	// Agent is the agent the slot in view runs (e.g. "claude").
	Agent string
	// Panes are the slot's panes, in frame cells; nil while unknown (the
	// pane layout is resolved after the note opens).
	Panes []PaneRect
}

// PaneRect is one pane of the slot in view, in frame cells.
type PaneRect struct {
	Role       string // PaneAgent, PaneDraft, PaneReview, PaneTerminal, or anything else
	X, Y, W, H int
	// File is the file a draft or review nvim was opened on, and Lines its
	// lines on disk (bounded by the reader), where the text under a box is
	// found; both empty when unknown.
	File  string
	Lines []string
}

// Pane roles.
const (
	PaneAgent    = "agent"
	PaneDraft    = "draft"
	PaneReview   = "review"
	PaneTerminal = "terminal"
)

// AnchorFor is the anchor of a note's box on frame f: the part of Couch's
// screen holding the box's centre, the steps down to it, and the text
// under the box.
//   - The last row: the tab bar (its chips), or the status line (the
//     broadcast's and Chrome's controls).
//   - Above it: the slot in view's pane, by role. An agent pane names the
//     agent; a draft or review nvim names its file and the lines under the
//     box, or its statusline on the pane's last row. With the panes unknown
//     the kind is AnchorScreen.
func AnchorFor(box broadcast.CellBox, f terminal.Frame, l ScreenLayout) broadcast.NoteAnchor {
	cols, rows := f.Geometry.Cols, f.Geometry.Rows
	a := broadcast.NoteAnchor{Kind: broadcast.AnchorScreen, Text: TextUnder(f, box)}
	if cols <= 0 || rows <= 0 {
		return a
	}
	cx, cy := box.Col+box.W/2, box.Row+box.H/2
	if cy >= rows-1 {
		var tabs []string
		for _, s := range l.Chips {
			if s.Start < box.Col+box.W && box.Col < s.End {
				tabs = append(tabs, strings.TrimSpace(rowText(f, rows-1, s.Start, s.End)))
			}
		}
		if len(tabs) > 0 && inChips(cx, l.Chips) {
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
	if l.Panes == nil {
		return a
	}
	for _, p := range l.Panes {
		if cx < p.X || cx >= p.X+p.W || cy < p.Y || cy >= p.Y+p.H {
			continue
		}
		a.Kind = broadcast.AnchorPane
		switch p.Role {
		case PaneAgent:
			step := "agent"
			if l.Agent != "" {
				step = "agent (" + l.Agent + ")"
			}
			a.Path = append(a.Path, step)
		case PaneDraft, PaneReview:
			step := "draft nvim"
			if p.Role == PaneReview {
				step = "review nvim"
			}
			a.Path = append(a.Path, step)
			if cy == p.Y+p.H-1 {
				a.Path = append(a.Path, "statusline")
			} else if p.File != "" {
				a.Path = append(a.Path, fileSteps(p, f, box)...)
			}
		case PaneTerminal:
			a.Path = append(a.Path, "terminal")
		default:
			a.Path = append(a.Path, "pane")
		}
		return a
	}
	return a
}

func inChips(col int, chips []ColumnSpan) bool {
	return len(chips) > 0 && col >= chips[0].Start && col < chips[len(chips)-1].End
}

// fileSteps names a nvim pane's file and, when the text under the box is
// found in it, the lines it is on and the Markdown section they are in:
// "notes.md, lines 12–18", "section: Plan". The pane's statusline (its
// last row) isn't file text.
func fileSteps(p PaneRect, f terminal.Frame, box broadcast.CellBox) []string {
	name := p.File[strings.LastIndexByte(p.File, '/')+1:]
	var rows []string
	for row := max(box.Row, p.Y); row <= min(box.Row+box.H-1, p.Y+p.H-2); row++ {
		if t := strings.TrimSpace(rowText(f, row, max(box.Col, p.X), min(box.Col+box.W, p.X+p.W))); t != "" {
			rows = append(rows, t)
		}
	}
	first, last, ok := locate(rows, p.Lines)
	if !ok {
		return []string{name}
	}
	steps := []string{fmt.Sprintf("%s, lines %d–%d", name, first+1, last+1)}
	if first == last {
		steps[0] = fmt.Sprintf("%s, line %d", name, first+1)
	}
	if h := sectionOf(p.Lines, first); h != "" {
		steps = append(steps, "section: "+h)
	}
	return steps
}

// maxLocate bounds locate's work, in line comparisons.
const maxLocate = 1 << 20

// locate finds rows (screen rows' text, in order) in lines: the first line
// holding rows[0] from which the rest follow, each on the same line (a
// wrapped line) or a later one. It returns the 0-based first and last
// lines, false when the rows aren't there (an unsaved buffer, a fold).
func locate(rows, lines []string) (first, last int, ok bool) {
	if len(rows) == 0 {
		return 0, 0, false
	}
	work := 0
	for i := range lines {
		if !strings.Contains(lines[i], rows[0]) {
			continue
		}
		j, matched := i, true
		for _, r := range rows[1:] {
			for j < len(lines) && !strings.Contains(lines[j], r) {
				j++
				if work++; work > maxLocate {
					return 0, 0, false
				}
			}
			if j == len(lines) {
				matched = false
				break
			}
		}
		if matched {
			return i, j, true
		}
	}
	return 0, 0, false
}

// sectionOf is the Markdown heading at or above line i, without its #s.
func sectionOf(lines []string, i int) string {
	for ; i >= 0; i-- {
		if t := strings.TrimSpace(lines[i]); strings.HasPrefix(t, "#") {
			return strings.TrimSpace(strings.TrimLeft(t, "#"))
		}
	}
	return ""
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
