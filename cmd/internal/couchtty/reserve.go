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

// StatusGroupID names one presentation group under an owning hub.
type StatusGroupID struct{ OwnerHub, GroupKey string }

// GroupCountSpan is a complete drawn count and its exact switcher scope.
// Empty Group means the ordinary unscoped switcher (global overflow).
type GroupCountSpan struct {
	Row, Start, End int
	Group           StatusGroupID
}

// StatusActor is one chip on the row.
type StatusActor struct {
	// Folded is nonzero only for a per-frame synthetic group count.
	Folded int
	// FoldedStateStrip is one checkout marker per hidden live member.
	FoldedStateStrip string
	// CollapseAnchor is derived from the launching hub, independently of focus.
	CollapseAnchor bool
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
	// Blocked: the thread's agent is behind an input-holding picker (pair#465).
	// It draws red even on the active chip, so the operator sees it at a glance.
	Blocked bool
	// Hub: the hub slot, which leads its section and reverses its colours
	// (pair#521); a later hub starts a new section after a separator.
	Hub      bool
	OwnerHub string
	OtherHub bool
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
	TabStyle  TabStyle
	Broadcast BroadcastCell
	Pointer   PointerCell
	// BroadcastSend and WebSend reflect eligible feedback, not note counts.
	BroadcastSend, WebSend bool
	// Notes is the broadcast's open sticky notes (#447), drawn as 💭N after
	// the capability controls; not a click target.
	Notes   int
	Capture terminalcapture.Status
	Actors  []StatusActor
	Notice  string
	// Webcast is the cast Chrome's cell (#454), drawn after the screen
	// broadcast's cells so their fixed columns never move.
	Webcast                       WebcastCell
	WebAnnotationOn, WebDrivingOn bool
	CapabilityNotice              string
	// WebNotes is the web cast's open notes (#470), drawn as 💭N after the
	// Chrome cell; a click opens the notes menu with them.
	WebNotes int
	// WebDriverName belongs to the enabled 👽 capability control.
	WebDriverName string
	// WebDrive is the separate pending-request span (✋Bo or ✋N).
	// A click opens the drive menu.
	WebDrive string
	// Spinner is the loading placeholder's spinner frame (pair#206).
	Spinner uint8
	// Palette is what the host terminal told couch about its colours, which
	// idle fading blends toward (pair#247).
	Palette Palette
	// Attention is the attention list's length (pair#673), drawn as 🔔N at the
	// row's right end; AttentionPrompt colors it as a blocked slot's chip.
	Attention       int
	AttentionPrompt bool
}

const (
	reverseSGR     = "\x1b[7m"
	attentionSGR   = "\x1b[38;5;220m"
	placeholderSGR = "\x1b[38;5;240m"
	// blockedSGR is the terminal's own red: its theme decides the shade. Not
	// the screen broadcast's LIVE red, which its indicator check reads.
	blockedSGR = "\x1b[31m"
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
	// Row is zero-based within the tab bar; the one-row renderer leaves it zero.
	Row    int
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
	Body        string
	Chips       []ChipSpan
	GroupCounts []GroupCountSpan
	// Control is the broadcast cell's red span, from the same clipping pass;
	// empty when no broadcast is running.
	Control ColumnSpan
	// Pointer is the 👆 control's span while live (#412); empty otherwise
	// or when clipped.
	Pointer ColumnSpan
	// Notes is the 💭N count's span (#447 M4): a click opens the notes menu.
	Notes ColumnSpan
	// BroadcastSend and WebSend cover only complete drawn send glyphs.
	BroadcastSend ColumnSpan
	WebSend       ColumnSpan
	// Webcast is the Chrome cell's span (#454); empty when Chrome isn't
	// running or the cell was clipped.
	Webcast ColumnSpan
	// WebFocus is the go-to-Chrome button's span (#501).
	WebFocus ColumnSpan
	// WebPointerLink and WebDriveLink are the 👆 and 👽 controls for pointer and
	// drive links (#501, pair#506), in the screen broadcast's `LIVE ⏸ 👆` family.
	// The drive span includes the driver name while driving is enabled.
	WebPointerLink ColumnSpan
	WebDriveLink   ColumnSpan
	// WebNotes is the web cast's 💭N span (#470).
	WebNotes ColumnSpan
	// Attention is the 🔔N span (pair#673): a click opens the attention list.
	Attention ColumnSpan
	// WebDrive is the pending drive requests' popup span (#467).
	WebDrive ColumnSpan
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
	r, _, _ := renderStatusCells(width, m, nil, false, -1)
	return r
}

// renderStatusCells shares the control, styling and hit-span pass with both APIs.
// prefixLimit bounds controls only; -1 preserves the normal control budget.
func renderStatusCells(width int, m StatusModel, cells []chipCell, planned bool, prefixLimit int) (RenderedStatusRow, int, int) {
	if width <= 0 {
		return RenderedStatusRow{}, 0, 0
	}
	var row strings.Builder
	used := 0
	// The attention badge is laid out first and drawn last, at the right end,
	// so everything else clips short of it.
	full, badge, badgeW := width, "", 0
	if m.Attention > 0 {
		badge = AttentionGlyph + strconv.Itoa(m.Attention)
		if badgeW = textwidth.Width(badge); badgeW+1 < width {
			width -= badgeW + 1
		} else {
			badge = ""
		}
	}
	actorBudget := width
	if prefixLimit >= 0 && prefixLimit < width {
		width = prefixLimit
	}
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
	// A send target is indivisible: reserve its normal leading gap and glyph
	// together so a clipped control never leaves an active hit target.
	sendButton := func() ColumnSpan {
		if used+textwidth.Width(" ⏎") > width {
			return ColumnSpan{}
		}
		appendText(" ", "")
		start := used
		appendText("⏎", "")
		return ColumnSpan{Start: start, End: used}
	}
	var control, pointer, notes, broadcastSend, webSend ColumnSpan
	if label := broadcastLabel(m.Broadcast); label != "" {
		appendText(label, broadcast.LiveSGR)
		control = ColumnSpan{Start: 0, End: used}
	}
	// While live, the capability controls follow LIVE at the fixed column the
	// broadcast's pointer check reads: `LIVE ⏸ 👆` (#412).
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
		if m.Notes > 0 {
			appendText(" ", "")
			start := used
			appendText(broadcast.NoteGlyph+strconv.Itoa(m.Notes), "")
			if used > start {
				notes = ColumnSpan{Start: start, End: used}
			}
		}
		if m.BroadcastSend {
			broadcastSend = sendButton()
		}
	}
	var webcastSpan, webFocusSpan, webPointerLinkSpan, webDriveLinkSpan, webNotesSpan, webDriveSpan ColumnSpan
	if label, sgr := webcastLabel(m.Webcast); label != "" {
		if used > 0 {
			appendText(" ", "")
		}
		start := used
		appendText(label, sgr)
		if used > start {
			webcastSpan = ColumnSpan{Start: start, End: used}
		}
		// The go-to-Chrome button and the two link glyphs (#501) sit right
		// after the cell, at columns that don't move with the note count.
		button := func(label, sgr string) ColumnSpan {
			appendText(" ", "")
			start := used
			appendText(label, sgr)
			if used == start {
				return ColumnSpan{}
			}
			return ColumnSpan{Start: start, End: used}
		}
		webFocusSpan = button(WebcastFocusLabel, "")
		webPointerLinkSpan = button(broadcast.PointerLabel, capabilitySGR(m.WebAnnotationOn))
		driveLabel, driveStyle := WebcastDriveLinkLabel, ""
		if m.WebDrivingOn {
			driveStyle = broadcast.PointerSGR
			if name := rowtext.Sanitize(m.WebDriverName); name != "" {
				driveLabel += " " + name
			}
		}
		webDriveLinkSpan = button(driveLabel, driveStyle)
		if m.WebNotes > 0 {
			appendText(" ", "")
			start := used
			appendText(broadcast.NoteGlyph+strconv.Itoa(m.WebNotes), "")
			if used > start {
				webNotesSpan = ColumnSpan{Start: start, End: used}
			}
		}
		if m.WebDrive != "" {
			appendText(" ", "")
			start := used
			appendText(rowtext.Sanitize(m.WebDrive), webDriveSGR)
			if used > start {
				webDriveSpan = ColumnSpan{Start: start, End: used}
			}
		}
		if m.WebSend {
			webSend = sendButton()
		}
	}
	// Capture follows so actor chips and transient notices cannot hide a
	// stopped recorder. It never receives an actor click target.
	if badge := captureBadge(m.Capture); badge != "" {
		if used > 0 {
			appendText(" ", "")
		}
		appendText(badge, "\x1b[1;7m")
	}
	if m.CapabilityNotice != "" {
		if used > 0 {
			appendText(" ", "")
		}
		appendText(ChromeStatusText(m.CapabilityNotice), "")
	}
	prefixUsed := used
	width = actorBudget
	var chips []ChipSpan
	var counts []GroupCountSpan
	lead := 0
	if used > 0 {
		lead = 2
	}
	if !planned {
		cells = fitChips(m.Actors, width-used-lead)
	}
	appendNotice := func() {
		if n := ChromeStatusText(m.Notice); n != "" {
			if used > 0 {
				appendText("  · ", "")
			}
			appendText(n, "")
		}
	}
	noticeDrawn := false
	for _, cell := range cells {
		gap := ""
		if used > 0 {
			gap = cellGap(cell)
		}
		if cell.actor < 0 || m.Actors[cell.actor].Folded > 0 {
			// A count is drawn whole or not at all: `+` alone reads as nothing.
			if used+textwidth.Width(gap+cell.label)+stripWidth(cell.strip) > width {
				break
			}
			if planned && cell.actor < 0 && m.Notice != "" {
				// A planned overflow marker belongs at the final row's end.
				// Let a transient notice use only the space before that marker.
				savedWidth := width
				width -= textwidth.Width(cell.label) + 2
				appendNotice()
				width = savedWidth
				noticeDrawn = true
				gap = ""
				if used > 0 {
					gap = "  "
				}
			}
			appendText(gap, "")
			start := used
			countStyle := ""
			if cell.actor >= 0 && m.Actors[cell.actor].OtherHub {
				countStyle = inactiveOwnerStyle(m.Palette, IdleFresh, baseDefault)
			}
			appendText(cell.label, countStyle)
			if cell.strip != "" {
				appendText(" ", countStyle)
				for _, marker := range cell.strip {
					style := countStyle
					if base, own := slotGlyphBase(marker); own {
						style = FadeStyle(m.Palette, IdleFresh, base)
						if cell.actor >= 0 && m.Actors[cell.actor].OtherHub {
							style += inactiveOwnerStyle(m.Palette, IdleFresh, base)
						}
					}
					appendText(string(marker), style)
				}
			}
			if used > start && m.TabStyle == TabStyleCollapse {
				group := StatusGroupID{}
				if cell.actor >= 0 {
					a := m.Actors[cell.actor]
					group = StatusGroupID{OwnerHub: a.OwnerHub, GroupKey: a.GroupKey}
				}
				counts = append(counts, GroupCountSpan{Start: start, End: used, Group: group})
			}
			continue
		}
		appendText(gap, "")
		a := m.Actors[cell.actor]
		label, tail := cell.label, chipTail(a, m.Spinner)
		if a.Active {
			label = "[" + label
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
		faded := !a.Placeholder && !a.Active && !a.Bell && !a.Blocked
		idle := IdleFresh
		if faded {
			idle = a.Idle
		}
		switch {
		case a.Placeholder:
			style = placeholderSGR
		case a.Blocked:
			style = blockedSGR
		case a.Bell && !a.Active:
			style = attentionSGR
		case faded:
			style = FadeStyle(m.Palette, idle, baseDefault)
		}
		if a.Hub {
			style += reverseSGR
		}
		if a.OtherHub && !m.Palette.NoColor {
			if a.Placeholder || a.Blocked || a.Bell {
				style += "\x1b[2m"
			} else {
				style += inactiveOwnerStyle(m.Palette, idle, baseDefault)
			}
		}
		appendText(label, style)
		for _, r := range a.Glyph {
			glyphStyle := style
			if base, own := slotGlyphBase(r); own && !a.Placeholder {
				glyphStyle = FadeStyle(m.Palette, idle, base)
				if a.Hub {
					glyphStyle += reverseSGR
				}
				if a.OtherHub {
					glyphStyle += inactiveOwnerStyle(m.Palette, idle, base)
				}
			}
			appendText(string(r), glyphStyle)
		}
		appendText(tail, style)
		if used > start && !a.Placeholder && a.Thread != (couchcore.ThreadAddress{}) {
			chips = append(chips, ChipSpan{Thread: a.Thread, Start: start, End: used})
		}
	}
	if !noticeDrawn {
		appendNotice()
	}
	var attention ColumnSpan
	if badge != "" {
		width = full
		appendText(strings.Repeat(" ", full-badgeW-used), "")
		sgr := attentionSGR
		if m.AttentionPrompt {
			sgr = blockedSGR
		}
		start := used
		appendText(badge, sgr)
		attention = ColumnSpan{Start: start, End: used}
	}
	return RenderedStatusRow{Body: row.String(), Chips: chips, GroupCounts: counts, Control: control, Pointer: pointer, Notes: notes, BroadcastSend: broadcastSend, WebSend: webSend, Webcast: webcastSpan, WebFocus: webFocusSpan, WebPointerLink: webPointerLinkSpan, WebDriveLink: webDriveLinkSpan, WebNotes: webNotesSpan, WebDrive: webDriveSpan, Attention: attention}, prefixUsed, actorBudget
}

// AttentionGlyph leads the attention badge (pair#673).
const AttentionGlyph = "🔔"

// WebcastCell is the cast Chrome's state on the tab bar (#454).
type WebcastCell uint8

const (
	// WebcastOff: no cast Chrome; nothing drawn.
	WebcastOff WebcastCell = iota
	// WebcastIdle: Chrome runs, not cast: `Chrome ▶` (click to go live).
	WebcastIdle
	// WebcastLive: the active tab is cast: `Chrome ⏸` (click to pause).
	WebcastLive
)

// Chrome cell labels and styles. Live, it is drawn as the screen
// broadcast's LIVE cell is (#501), so the two cast indicators read as one
// family; the broadcast's indicator check matches LIVE's text, not this.
const (
	WebcastIdleLabel = "Chrome ▶"
	WebcastLiveLabel = "Chrome ⏸"
	// WebcastFocusLabel is the button that brings the cast Chrome to the
	// front (#501).
	WebcastFocusLabel = "↗"
	// WebcastDriveLinkLabel toggles driving or copies its link. The 👆 before it
	// is the screen broadcast's own PointerLabel; the broadcast has no 👽
	// since #503 removed its remote-control placeholder.
	WebcastDriveLinkLabel = "👽"
	// webDriveSGR: someone else's hands on the operator's tab, in bold.
	webDriveSGR = "\x1b[1m"
)

func webcastLabel(c WebcastCell) (string, string) {
	switch c {
	case WebcastIdle:
		return WebcastIdleLabel, ""
	case WebcastLive:
		return WebcastLiveLabel, broadcast.LiveSGR
	}
	return "", ""
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

// sectionGap separates one hub's section from the next (pair#491).
const sectionGap = " │ "

// cellGap is what goes before a cell that follows another: the drawer and the
// width estimate both read it, so they can't disagree.
func cellGap(cell chipCell) string {
	if cell.section {
		return sectionGap
	}
	return "  "
}

// chipCell is one cell of the actor run: an actor drawn with label, or, with
// actor -1, a count of slots folded away for width (pair#491).
type chipCell struct {
	actor   int
	label   string
	strip   string // admitted whole after base packing; never affects selection
	section bool
}

// chipTail is what follows a chip's label and glyph: the loading spinner, and
// the active chip's closing bracket.
func chipTail(a StatusActor, spinner uint8) string {
	tail := ""
	if a.Placeholder && a.Loading {
		tail = " " + spinnerGlyph(spinner)
	}
	if a.Active {
		tail += "]"
	}
	return tail
}

// chipKept is a chip that width never folds away: the hub, which leads its
// section, and every chip asking to be seen (active, red, belled).
func chipKept(a StatusActor) bool { return a.Folded == 0 && (a.Hub || a.Active || a.Blocked || a.Bell) }

// fitChips lays the actors out in budget columns (pair#491). In full, a
// group's first drawn member shows its label and later slots show `:N`. When
// that is too wide, groups fold all but their first member into a `+N` count,
// from the right (`pair  :2  +1`, `ops:1  +1`): a count always follows a chip
// of its own repository, so it can't read as the previous one's. If a kept chip still lies past the edge, whole
// groups fold, from the right, into one trailing count until it is in view; a
// kept chip whose group leader folded shows its full label. Otherwise the
// drawer clips as before, so a plain tab cut short stays clickable.
func fitChips(actors []StatusActor, budget int) []chipCell {
	groups := chipGroups(actors)
	collapsed := make([]bool, len(groups))
	folded := make([]bool, len(groups))
	cells := layoutChips(actors, groups, collapsed, folded)
	for g := len(groups) - 1; g >= 0 && chipsWidth(actors, cells, len(cells)) > budget; g-- {
		collapsed[g] = true
		cells = layoutChips(actors, groups, collapsed, folded)
	}
	for g := len(groups) - 1; g >= 0 && keptEnd(actors, cells) > budget; g-- {
		folded[g] = true
		cells = layoutChips(actors, groups, collapsed, folded)
	}
	return cells
}

// keptEnd is the column where the last kept chip ends, 0 with none.
func keptEnd(actors []StatusActor, cells []chipCell) int {
	end := 0
	for i, cell := range cells {
		if cell.actor >= 0 && chipKept(actors[cell.actor]) {
			end = chipsWidth(actors, cells, i+1)
		}
	}
	return end
}

// chipGroups splits the actors into runs of one repository. A primary hub
// anchors its own slots; a numbered hub remains a run of its own.
func chipGroups(actors []StatusActor) [][2]int {
	var groups [][2]int
	for i, a := range actors {
		if i == 0 || a.Hub || actors[i-1].Hub && actors[i-1].SlotNumber > 0 || a.GroupKey == "" || a.GroupKey != actors[i-1].GroupKey {
			groups = append(groups, [2]int{i, i})
		}
		groups[len(groups)-1][1] = i + 1
	}
	return groups
}

func layoutChips(actors []StatusActor, groups [][2]int, collapsed, folded []bool) []chipCell {
	var cells []chipCell
	hidden := 0
	for g, span := range groups {
		count, previous := 0, false
		for i := span[0]; i < span[1]; i++ {
			a := actors[i]
			if !chipKept(a) && (folded[g] || collapsed[g] && i > span[0]) {
				if folded[g] {
					hidden++
				} else {
					count++
				}
				continue
			}
			label := rowtext.Sanitize(a.Label)
			if previous && a.SlotNumber > 0 {
				label = ":" + strconv.Itoa(a.SlotNumber)
			}
			previous = true
			cells = append(cells, chipCell{actor: i, label: label, section: a.Hub && len(cells) > 0})
		}
		if count > 0 {
			cells = append(cells, chipCell{actor: -1, label: "+" + strconv.Itoa(count)})
		}
	}
	if hidden > 0 {
		cells = append(cells, chipCell{actor: -1, label: "+" + strconv.Itoa(hidden)})
	}
	return cells
}

// chipsWidth is the columns the first n cells take, gaps included, as drawn.
func chipsWidth(actors []StatusActor, cells []chipCell, n int) int {
	return measureChipCells(actors, cells[:n], nil)
}

func stripWidth(strip string) int {
	if strip == "" {
		return 0
	}
	return 1 + textwidth.Width(strip)
}

// measureChipCells is the shared width pass, optionally counted by the planner.
func measureChipCells(actors []StatusActor, cells []chipCell, work *statusLayoutWork) int {
	total := 0
	for i, cell := range cells {
		if work != nil {
			work.Visits++
			work.Bytes += len(cell.label) + len(cell.strip)
		}
		if i > 0 {
			total += textwidth.Width(cellGap(cell))
		}
		total += textwidth.Width(cell.label) + stripWidth(cell.strip)
		if cell.actor >= 0 {
			a := actors[cell.actor]
			if work != nil {
				work.Bytes += len(a.Glyph)
			}
			if a.Active {
				total++
			}
			total += textwidth.Width(a.Glyph) + textwidth.Width(chipTail(a, 0))
		}
	}
	return total
}
