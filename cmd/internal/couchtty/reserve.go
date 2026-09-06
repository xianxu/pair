package couchtty

import (
	"strings"

	"github.com/xianxu/pair/cmd/internal/ansi"
	"github.com/xianxu/pair/cmd/internal/couchcore"

	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/textwidth"
)

// This file COMPOSES sequences from hostty's constants; it does not spell them.
// One escape, one definition, per the paired-terminator lesson.

// ChildRows is how tall a child is on a host of that many rows: one shorter,
// because couch owns the last row.
//
// It never returns zero. A terminal too short to reserve from gives the child
// the whole thing and the row is simply not drawn -- a zero-row pty is not a
// thing, and clamping here keeps every caller from re-deciding it.
func ChildRows(hostRows uint16) uint16 {
	if hostRows == 0 {
		return 1
	}
	if hostRows == 1 {
		return hostRows
	}
	return hostRows - 1
}

// Reserve pins the scrolling region above the reserved row.
//
// This is what makes the row a RESERVATION rather than compositing: a child
// scrolling at the bottom of its own screen scrolls inside the region and
// cannot walk onto the row below it. The child is never told; from its side
// this is just a smaller terminal (Decision 4).
func Reserve(hostRows uint16) string {
	if hostRows <= 1 {
		return ""
	}
	return hostty.SetRegion(1, int(hostRows)-1)
}

// Release resets the region. Written on teardown, or a child that set margins
// and died would leave the operator's shell scrolling inside a box.
func Release() string { return hostty.ResetRegion }

// PaintRow draws the reserved row without disturbing the child.
//
// Save and restore BRACKET the paint. Without them the child's cursor is left
// on the status row, which the operator sees as the caret jumping to the bottom
// line every time anything is notified.
func PaintRow(hostRows uint16, text string) string {
	if hostRows == 0 {
		return ""
	}
	return hostty.SaveCursor +
		hostty.MoveTo(int(hostRows), 1) +
		hostty.ClearLine +
		text +
		hostty.RestoreCursor
}

// StatusActor is one chip on the row.
type StatusActor struct {
	Label string
	// Thread is who a click on this chip lands on. The THREAD address, not the
	// pane handle or the actor id: the declared `switch` operation is addressed
	// by thread, so carrying anything else here would mean translating at the
	// call site and having two answers to "which actor is this" (pair#172).
	Thread couchcore.ThreadAddress
	Active bool
	// Bell means this actor has asked for attention since the operator last
	// looked at it. Before #147's transport it is the only real activity
	// signal available, which is why the row carries it at all.
	Bell bool
}

// StatusModel is everything the row shows.
type StatusModel struct {
	Actors []StatusActor
	Notice string
}

// The untrusted-text rationale below belongs to RenderStatusRow, and sat above
// ChipSpan until a review pointed out that `go doc ChipSpan` printed it: a
// comment separated from its subject by an intervening declaration documents the
// wrong thing to every reader who arrives through the tool rather than the file.
//
// RenderStatusRow lays the model out in width columns.
//
// Labels and notices carry UNTRUSTED text: couchcore.Describe prefers a sidecar
// the agent session writes, so a description is whatever a child chose to put
// there. Control bytes are stripped rather than escaped-around, because the
// hazard is not a mangled row -- it is `\x1b[2J` from a description clearing the
// operator's screen. Stripping also makes truncation honest, since after it
// every remaining byte occupies the columns textwidth says it does.
// ChipSpan is the column range one actor occupies on the drawn row, and the
// actor a click there lands on. Half-open: [Start, End).
//
// ZERO-BASED, like the string it indexes. An SGR mouse report is ONE-based, so a
// caller converts once at the boundary -- see RenderedStatusRow.ColumnToActor's
// contract. Stated because the two bases meet in this feature and an unstated
// one is an off-by-one waiting for a narrow terminal.
type ChipSpan struct {
	Thread couchcore.ThreadAddress
	Start  int
	End    int
}

// RenderedStatusRow is the drawn row and where its chips are.
//
// One value, because the spans must come from the pass that already CLIPS chips
// to width. A caller re-deriving them from StatusModel would agree at
// comfortable widths and disagree at exactly the narrow ones where clipping
// happens -- the case a mis-mapped click is least catchable by eye (ARCH-DRY).
type RenderedStatusRow struct {
	Body  string
	Chips []ChipSpan
}

// ColumnToActor maps a ZERO-BASED column on the drawn row to the actor whose
// chip covers it. Total: a column in a gap, past the last chip, or negative is
// nobody, which is how "clicking bare row does nothing" is expressed as a value
// rather than as a branch at the call site.
//
// The caller converts from the report's 1-based X.
func (r RenderedStatusRow) ColumnToActor(column int) (couchcore.ThreadAddress, bool) {
	for _, chip := range r.Chips {
		if column >= chip.Start && column < chip.End {
			return chip.Thread, true
		}
	}
	return couchcore.ThreadAddress{}, false
}

func RenderStatusRow(width int, m StatusModel) RenderedStatusRow {
	if width <= 0 {
		return RenderedStatusRow{}
	}
	var row strings.Builder
	used := 0
	appendText := func(text string, attention bool) {
		if used >= width || text == "" {
			return
		}
		clipped := truncate(text, width-used)
		if clipped == "" {
			return
		}
		if attention {
			row.WriteString("\x1b[38;5;220m")
		}
		row.WriteString(clipped)
		if attention {
			row.WriteString("\x1b[0m")
		}
		used += textwidth.Width(clipped)
	}
	var chips []ChipSpan
	for _, a := range m.Actors {
		label := sanitize(a.Label)
		if a.Active {
			label = "[" + label + "]"
		}
		if used > 0 {
			appendText("  ", false)
		}
		// Recorded from the SAME appendText that clips, so a chip the width
		// dropped contributes no span and a chip the width truncated contributes
		// the columns it actually drew.
		start := used
		appendText(label, a.Bell && !a.Active)
		if used > start && a.Thread != (couchcore.ThreadAddress{}) {
			chips = append(chips, ChipSpan{Thread: a.Thread, Start: start, End: used})
		}
	}
	if n := sanitize(m.Notice); n != "" {
		if used > 0 {
			appendText("  · ", false)
		}
		appendText(n, false)
	}
	return RenderedStatusRow{Body: row.String(), Chips: chips}
}

// sanitize removes escape SEQUENCES first, then any remaining C0 control or
// DEL. Two passes, and the order matters: dropping the lone ESC byte would
// leave `[2J` sitting in the row as visible junk -- safe, but garbage the
// operator cannot explain. ansi.Strip is the repo's existing answer to "remove
// complete escape sequences", so the sequence framing is not re-decided here.
func sanitize(s string) string {
	stripped := string(ansi.Strip([]byte(s)))
	return strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, stripped)
}

// truncate cuts to width in terminal COLUMNS, not bytes or runes -- an emoji in
// an agent's description is one rune and two columns, and the row must not wrap
// onto the child's area.
func truncate(s string, width int) string {
	if textwidth.Width(s) <= width {
		return s
	}
	var b strings.Builder
	used := 0
	for _, r := range s {
		w := textwidth.Width(string(r))
		if used+w > width {
			break
		}
		b.WriteRune(r)
		used += w
	}
	return b.String()
}
