package couchtty

import (
	"strconv"
	"strings"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/rowtext"

	"github.com/xianxu/pair/cmd/internal/textwidth"
)

// This file is the POLICY half of the reserved row: what the row SAYS.
//
// The mechanism is terminal.Presenter: since #255 M3 the row is a chrome row
// composed into each presented frame (UpdateChrome), not a region reserved and
// painted beside the child. hostty.Reservation now supplies only the row count.
// What each consumer draws there stays with the consumer: couch renders actors,
// termcmd renders tabs.

// StatusActor is one chip on the row.
type StatusActor struct {
	// GroupKey identifies the primary checkout; SlotNumber is zero for ordinary
	// tabs. The renderer shortens only after drawing this group's first member.
	GroupKey   string
	SlotNumber int
	Label      string
	// Glyph is the slot quick-status glyph PresentThreads derived (pair#317).
	Glyph string
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
	// Placeholder marks a thread the reattach pass has not attached yet
	// (pair#206). It is drawn greyed, and it records NO chip span, so a click on
	// it resolves to no actor: unclickable by construction, not by a check at
	// the click site.
	Placeholder bool
	// Loading marks the one placeholder currently starting; it carries the
	// spinner.
	Loading bool
}

// StatusModel is everything the row shows.
type StatusModel struct {
	Actors []StatusActor
	Notice string
	// Spinner is the loading placeholder's spinner frame (pair#206).
	Spinner uint8
}

const (
	attentionSGR   = "\x1b[38;5;220m"
	placeholderSGR = "\x1b[38;5;240m"
)

// slotGlyphSGR is the one styling decision for slot glyphs, shared by the tab
// bar and the switcher, per glyph character: a diverged resting branch (±) and a
// dirty tree (*) ask for attention in the one amber (pair#321), and every other
// glyph keeps its row's style (empty).
func slotGlyphSGR(glyph rune) string {
	switch string(glyph) {
	case couchcore.SlotGlyphDiverged, couchcore.SlotGlyphDirty:
		return attentionSGR
	}
	return ""
}

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

// RenderStatusRow lays the model out in width columns.
//
// Labels and notices carry UNTRUSTED text: couchcore.Describe prefers a sidecar
// the agent session writes, so a description is whatever a child chose to put
// there. Control bytes are stripped rather than escaped-around, because the
// hazard is not a mangled row -- it is `\x1b[2J` from a description clearing the
// operator's screen. Stripping also makes truncation honest, since after it
// every remaining byte occupies the columns textwidth says it does.
func RenderStatusRow(width int, m StatusModel) RenderedStatusRow {
	if width <= 0 {
		return RenderedStatusRow{}
	}
	var row strings.Builder
	used := 0
	appendText := func(text, sgr string) {
		if used >= width || text == "" {
			return
		}
		clipped := rowtext.Fit(text, width-used)
		if clipped == "" {
			return
		}
		if sgr != "" {
			row.WriteString(sgr)
		}
		row.WriteString(clipped)
		if sgr != "" {
			row.WriteString("\x1b[0m")
		}
		used += textwidth.Width(clipped)
	}
	var chips []ChipSpan
	previousGroup := ""
	for _, a := range m.Actors {
		label := rowtext.Sanitize(a.Label)
		if a.GroupKey != "" && a.GroupKey == previousGroup && a.SlotNumber > 0 {
			label = ":" + strconv.Itoa(a.SlotNumber)
		}
		previousGroup = a.GroupKey
		// The glyph is its own segment so it can carry its own colour; clipping
		// still runs through the one appendText, segment by segment.
		tail := ""
		if a.Placeholder && a.Loading {
			tail = " " + spinnerGlyph(m.Spinner)
		}
		if a.Active {
			label, tail = "["+label, tail+"]"
		}
		if used > 0 {
			appendText("  ", "")
		}
		// Recorded from the SAME appendText that clips, so a chip the width
		// dropped contributes no span and a chip the width truncated contributes
		// the columns it actually drew.
		start := used
		style := ""
		switch {
		case a.Placeholder:
			style = placeholderSGR
		case a.Bell && !a.Active:
			style = attentionSGR
		}
		appendText(label, style)
		for _, r := range a.Glyph {
			glyphStyle := style
			if own := slotGlyphSGR(r); own != "" && !a.Placeholder {
				glyphStyle = own
			}
			appendText(string(r), glyphStyle)
		}
		appendText(tail, style)
		if used > start && !a.Placeholder && a.Thread != (couchcore.ThreadAddress{}) {
			chips = append(chips, ChipSpan{Thread: a.Thread, Start: start, End: used})
		}
	}
	if n := rowtext.Sanitize(m.Notice); n != "" {
		if used > 0 {
			appendText("  · ", "")
		}
		appendText(n, "")
	}
	return RenderedStatusRow{Body: row.String(), Chips: chips}
}

// sanitize removes escape SEQUENCES first, then any remaining C0 control or
// DEL. Two passes, and the order matters: dropping the lone ESC byte would
// leave `[2J` sitting in the row as visible junk -- safe, but garbage the
// operator cannot explain. ansi.Strip is the repo's existing answer to "remove
// complete escape sequences", so the sequence framing is not re-decided here.

// truncate cuts to width in terminal COLUMNS, not bytes or runes -- an emoji in
// an agent's description is one rune and two columns, and the row must not wrap
// onto the child's area.

// spinnerGlyph is couch's one spinner. The switcher's progress notice and the
// status row's loading placeholder both draw from it, so the same idea never
// shows two different animations.
func spinnerGlyph(phase uint8) string {
	frames := [...]string{"◐", "◓", "◑", "◒"}
	return frames[int(phase)%len(frames)]
}
