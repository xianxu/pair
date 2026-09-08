package termcmd

import (
	"strings"

	"github.com/xianxu/pair/cmd/internal/rowtext"
	"github.com/xianxu/pair/cmd/internal/textwidth"
)

// The tab strip: what `pair term` draws in the row it reserves for itself.
//
// This is the POLICY half of the reserved row, the counterpart to couch's
// RenderStatusRow. The mechanism -- reserving the row, painting it without
// moving the child's cursor -- is hostty.Reservation, shared by both (#199 M1).
//
// Pure: no IO, no mux, no terminal. Everything that makes the row correct under
// a hostile tab name or a narrow pane is decidable from (width, model), which is
// what lets the hard cases be unit tests rather than manual observation.

// TabChip is one tab as the strip sees it.
type TabChip struct {
	// Name is OPERATOR-SUPPLIED and therefore untrusted: it reaches a live
	// terminal, so it goes through rowtext before it is drawn.
	Name string
}

// RenameField is a rename in progress as the strip draws it.
//
// It sits on the model rather than on TabChip because it is a MODE the row is
// in: at most one tab is being renamed, and the row says so in that tab's place.
type RenameField struct {
	// Tab indexes Tabs, with the same out-of-range contract Active carries and
	// for the same reason -- a background tab exiting reindexes the slice while
	// a rename is open.
	Tab int
	// Text is the field with its caret already composed, by RenameEditor.Field.
	// OPERATOR-SUPPLIED, so it goes through rowtext exactly as a name does.
	Text string
}

// StripModel is everything the row shows.
type StripModel struct {
	Tabs []TabChip
	// Active indexes Tabs. It may be OUT OF RANGE: the mux's tabs can close
	// between building this model and drawing it, and a renderer that panics
	// there takes the pane down with it. Out of range marks nothing.
	Active int
	// Rename is the tab being renamed and what the operator has typed, or nil.
	//
	// The strip owns this because the row is where tab state lives now. It used
	// to be drawn ONLY into the zellij pane title -- i.e. into the pane FRAME,
	// which M4 removes -- so the field had no surface at all once the frame came
	// off (found by cmd/probes/couchnestedrows).
	Rename *RenameField
}

// TabSpan is where a tab was drawn, in DISPLAY COLUMNS.
//
// Columns, not rune offsets: a mouse click arrives as a column, and a rune count
// puts every span after a wide character one column left of what was drawn.
// Emitted here so #200 can map a click to a tab without re-deriving the layout
// -- and, per #172's BR-32, recorded by the same pass that CLIPS, so a tab the
// width dropped contributes no span and a truncated one contributes the columns
// it actually drew.
type TabSpan struct {
	Index int
	Start int
	End   int
}

// RenderedStrip is the drawn row and where its tabs are.
type RenderedStrip struct {
	Body  string
	Spans []TabSpan
}

// RenderStrip draws the tab strip for a pane `width` columns wide.
func RenderStrip(width int, m StripModel) RenderedStrip {
	if width <= 0 || len(m.Tabs) == 0 {
		return RenderedStrip{}
	}

	// The tab being RENAMED and then the ACTIVE tab are placed first in the
	// budget, and the others fill what remains. A narrow pane must never drop
	// the tab the operator is looking at, still less the field they are typing
	// into -- and left-to-right truncation loses them precisely when tabs are
	// numerous.
	order := make([]int, 0, len(m.Tabs))
	placed := make(map[int]bool, 2)
	first := func(i int) {
		if i >= 0 && i < len(m.Tabs) && !placed[i] {
			placed[i] = true
			order = append(order, i)
		}
	}
	if m.Rename != nil {
		first(m.Rename.Tab)
	}
	first(m.Active)
	for i := range m.Tabs {
		if !placed[i] {
			order = append(order, i)
		}
	}

	drawn := make(map[int]string, len(m.Tabs))
	used := 0
	for _, i := range order {
		label := chipLabel(m, i)
		sep := 0
		if len(drawn) > 0 {
			sep = 1 // one space between chips
		}
		if used+sep+textwidth.Width(label) > width {
			// Truncate rather than drop, but only if a useful amount survives.
			room := width - used - sep
			if room < 3 {
				continue
			}
			label = rowtext.Fit(label, room)
			if textwidth.Width(label) < 3 {
				continue
			}
		}
		drawn[i] = label
		used += sep + textwidth.Width(label)
	}

	// Emitted in TAB order, whatever order the budget admitted them in: the
	// strip reads left to right as the tabs are numbered, and Alt+Left/Right
	// move through that order.
	var row strings.Builder
	var spans []TabSpan
	col := 0
	for i := range m.Tabs {
		label, ok := drawn[i]
		if !ok {
			continue
		}
		if col > 0 {
			row.WriteString(" ")
			col++
		}
		start := col
		row.WriteString(label)
		col += textwidth.Width(label)
		spans = append(spans, TabSpan{Index: i, Start: start, End: col})
	}
	return RenderedStrip{Body: row.String(), Spans: spans}
}

// chipLabel is the drawn form of one tab.
//
// The active tab is bracketed, and that is deliberately a TEXT difference rather
// than only a colour one: colour is lost to a mono terminal, to ansi.Strip in a
// log, and to a colour-blind operator, and "which tab am I in" is the whole
// point of the row.
func chipLabel(m StripModel, i int) string {
	if m.Rename != nil && m.Rename.Tab == i {
		return "[rename: " + rowtext.Sanitize(m.Rename.Text) + "]"
	}
	name := rowtext.Sanitize(m.Tabs[i].Name)
	if i == m.Active {
		return "[" + name + "]"
	}
	return name
}
