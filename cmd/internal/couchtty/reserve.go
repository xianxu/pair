package couchtty

import (
	"strings"

	"github.com/xianxu/pair/cmd/internal/ansi"

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
	if hostRows <= 1 {
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
	Label  string
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

// RenderStatusRow lays the model out in width columns.
//
// Labels and notices carry UNTRUSTED text: couchcore.Describe prefers a sidecar
// the agent session writes, so a description is whatever a child chose to put
// there. Control bytes are stripped rather than escaped-around, because the
// hazard is not a mangled row -- it is `\x1b[2J` from a description clearing the
// operator's screen. Stripping also makes truncation honest, since after it
// every remaining byte occupies the columns textwidth says it does.
func RenderStatusRow(width int, m StatusModel) string {
	if width <= 0 {
		return ""
	}
	var chips []string
	for _, a := range m.Actors {
		label := sanitize(a.Label)
		switch {
		case a.Active:
			label = "[" + label + "]"
		case a.Bell:
			label += "*"
		}
		chips = append(chips, label)
	}

	row := strings.Join(chips, "  ")
	if n := sanitize(m.Notice); n != "" {
		if row != "" {
			row += "  · "
		}
		row += n
	}
	return truncate(row, width)
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
