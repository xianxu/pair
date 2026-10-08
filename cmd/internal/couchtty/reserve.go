package couchtty

import (
	"github.com/xianxu/pair/cmd/internal/broadcast"
	"strconv"
	"strings"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/rowtext"
	"github.com/xianxu/pair/cmd/internal/terminalcapture"

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
	// Loading marks a placeholder currently starting; it carries the spinner.
	// Several can load at once (pair#205).
	Loading bool
	// Idle is how long this thread has gone without activity (pair#247). It
	// fades the chip -- label and amber glyphs alike -- unless the chip is
	// selected, has a pending notification, or is a placeholder, which all
	// keep their own emphasis.
	Idle IdleLevel
}

// BroadcastCell is the state of the tab bar's broadcast control (#395).
type BroadcastCell uint8

const (
	// BroadcastOff draws nothing: no cell, no click target.
	BroadcastOff BroadcastCell = iota
	// BroadcastStarting is the tunnel opening; nothing streams yet.
	BroadcastStarting
	// BroadcastLive draws broadcast.LiveLabel, the indicator every streamed
	// frame must show.
	BroadcastLive
)

// PointerCell is the pointer control's state (#412), drawn only while live.
type PointerCell uint8

const (
	// PointerOff draws a dim 👆: no pointer link yet, or pointing turned off.
	PointerOff PointerCell = iota
	// PointerOn draws 👆 on broadcast.PointerSGR, the marker the broadcast's
	// pointer check reads.
	PointerOn
)

// dimSGR draws an inactive control.
const dimSGR = "\x1b[2m"

// StatusModel is everything the row shows.
type StatusModel struct {
	Broadcast BroadcastCell
	Pointer   PointerCell
	Capture   terminalcapture.Status
	Actors    []StatusActor
	Notice    string
	// Spinner is the loading placeholder's spinner frame (pair#206).
	Spinner uint8
	// Palette is what the host terminal told couch about its colours, which
	// idle fading blends toward (pair#247).
	Palette Palette
}

const (
	attentionSGR   = "\x1b[38;5;220m"
	placeholderSGR = "\x1b[38;5;240m"
)

// slotGlyphBase is the one styling decision for slot glyphs, shared by the tab
// bar and the switcher, per glyph character: a diverged resting branch (±) and a
// dirty tree (*) ask for attention in the one amber (pair#321), and every other
// glyph keeps its row's style (false). It names the glyph's BASE colour rather
// than an escape, so idle fading (pair#247) fades each glyph from its own
// colour: FadeStyle at IdleFresh is exactly that colour.
func slotGlyphBase(glyph rune) (styleBase, bool) {
	switch string(glyph) {
	case couchcore.SlotGlyphDiverged, couchcore.SlotGlyphDirty:
		return baseAmber, true
	}
	return baseDefault, false
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

// ColumnSpan is a zero-based, half-open column range on the drawn row.
type ColumnSpan struct{ Start, End int }

func (s ColumnSpan) Contains(column int) bool { return column >= s.Start && column < s.End }

// RenderedStatusRow is the drawn row and where its chips are.
//
// One value, because the spans must come from the pass that already CLIPS chips
// to width. A caller re-deriving them from StatusModel would agree at
// comfortable widths and disagree at exactly the narrow ones where clipping
// happens -- the case a mis-mapped click is least catchable by eye (ARCH-DRY).
type RenderedStatusRow struct {
	Body  string
	Chips []ChipSpan
	// Control is the broadcast cell's red span, from the same clipping pass;
	// empty when no broadcast is running.
	Control ColumnSpan
	// Pointer and Remote are the 👆 and 👽 controls' spans while live
	// (#412); empty otherwise or when clipped.
	Pointer ColumnSpan
	Remote  ColumnSpan
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
	// The broadcast cell leads (#395): it is the indicator the broadcast
	// checks every frame for, so nothing may push it off the row.
	var control, pointer, remote ColumnSpan
	if label := broadcastLabel(m.Broadcast); label != "" {
		appendText(label, broadcast.LiveSGR)
		control = ColumnSpan{Start: 0, End: used}
	}
	// While live, the capability controls follow LIVE at the fixed column the
	// broadcast's pointer check reads: `LIVE ⏸ 👆 👽` (#412).
	if m.Broadcast == BroadcastLive {
		capability := func(label, sgr string) ColumnSpan {
			appendText(" ", "")
			start := used
			appendText(label, sgr)
			if used == start {
				return ColumnSpan{}
			}
			return ColumnSpan{Start: start, End: used}
		}
		pointerSGR := dimSGR
		if m.Pointer == PointerOn {
			pointerSGR = broadcast.PointerSGR
		}
		pointer = capability(broadcast.PointerLabel, pointerSGR)
		remote = capability(broadcast.ControlLabel, dimSGR)
	}
	// Capture follows so actor chips and transient notices cannot hide a
	// stopped recorder. It never receives an actor click target.
	if badge := captureBadge(m.Capture); badge != "" {
		if used > 0 {
			appendText(" ", "")
		}
		appendText(badge, "\x1b[1;7m")
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
		// Fading is the fallback: every stronger cue wins over idleness. The
		// switcher's live-row branch in renderRootMenuFrame states the same
		// precedence from its own inputs (selection, attention); keep the two
		// in step (pair#247).
		faded := !a.Placeholder && !a.Active && !a.Bell
		idle := IdleFresh
		if faded {
			idle = a.Idle
		}
		switch {
		case a.Placeholder:
			style = placeholderSGR
		case a.Bell && !a.Active:
			style = attentionSGR
		case faded:
			style = FadeStyle(m.Palette, idle, baseDefault)
		}
		appendText(label, style)
		for _, r := range a.Glyph {
			glyphStyle := style
			if base, own := slotGlyphBase(r); own && !a.Placeholder {
				glyphStyle = FadeStyle(m.Palette, idle, base)
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
	return RenderedStatusRow{Body: row.String(), Chips: chips, Control: control, Pointer: pointer, Remote: remote}
}

func broadcastLabel(c BroadcastCell) string {
	switch c {
	case BroadcastStarting:
		return broadcast.StartingLabel
	case BroadcastLive:
		return broadcast.LiveLabel
	}
	return ""
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
