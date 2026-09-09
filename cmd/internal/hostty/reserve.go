package hostty

import "fmt"

// Row reservation: how a process keeps one row of its terminal for itself while
// a child scrolls freely above it.
//
// This is a RESERVATION, not compositing. The scrolling region is pinned to
// stop one row short, so a child scrolling at the bottom of its own screen
// scrolls inside the region and cannot walk onto the row below. The child is
// never told; from its side this is simply a shorter terminal.
//
// It lives in hostty because it is host-half MECHANISM shared by two consumers
// -- couch reserves the host's bottom row for its actor strip, `pair term`
// reserves its pane's bottom row for a tab strip (pair#199) -- while what the
// row SAYS stays with each consumer as policy. Same split the atlas already
// records for ptychild/hostty: "what is shared is structure; what stays is
// policy. `\x1b[r` lives here and only here."
//
// That `pair term` can do this at all is measured, not assumed: zellij honors
// DECSTBM from a pane process (pair#199 finding 5 -- 200 lines scrolled in the
// region while the reserved row held its paint).

// Edge is which end of the terminal a reservation takes.
type Edge int

const (
	// EdgeBottom is the only implemented edge, and the zero value so that a
	// zero Reservation is the safe one.
	EdgeBottom Edge = iota
	// EdgeTop is representable and REFUSED. A top reservation is not the mirror
	// of a bottom one: with the region set to 2..N the child still addresses
	// absolute rows, so its row 1 IS the strip and any absolute positioning it
	// does lands on top of us. Making it correct needs origin mode (DECOM,
	// `\x1b[?6h`) arbitrated with children that set it themselves, which
	// nothing here tracks. Named rather than omitted so the asymmetry is
	// discoverable instead of being rediscovered.
	EdgeTop
)

// Reservation is a terminal of Rows rows with one row held at Edge.
//
// It answers ChildRows, ReserveAndPaint, Paint and Release. There is no bare
// `Reserve()`: both consumers assert the region and draw the row together, in
// that order, because DECSTBM homes the cursor -- so a caller that could reserve
// WITHOUT painting is a caller that can compose the two in the order that was
// the bug.
type Reservation struct {
	Rows uint16
	Edge Edge
}

// NewReservation is the validating constructor: it refuses an edge that is not
// implemented rather than letting a caller find out by seeing a corrupt screen.
func NewReservation(rows uint16, edge Edge) (Reservation, error) {
	// rows too, not just the edge: a "validating door" that admits rows: 0
	// hands back a Reservation whose every method silently no-ops, which reads
	// to the caller as a working reservation that never draws.
	if rows <= 1 {
		return Reservation{}, fmt.Errorf(
			"hostty: %d rows cannot be reserved from; a terminal needs at least 2 "+
				"so the child keeps one", rows)
	}
	if edge != EdgeBottom {
		return Reservation{}, fmt.Errorf(
			"hostty: edge %d is not implemented; only EdgeBottom reserves correctly "+
				"(a top strip needs DECOM arbitration)", edge)
	}
	return Reservation{Rows: rows, Edge: edge}, nil
}

// usable reports whether this reservation can actually hold a row.
//
// An unsupported edge fails CLOSED -- no region, no paint, child gets the whole
// terminal. The struct is exported so a caller can build one without
// NewReservation, and the alternative to failing closed is emitting a region
// computed for an edge we do not implement: a silently corrupted screen, where
// failing closed costs only the strip.
func (r Reservation) usable() bool {
	return r.Edge == EdgeBottom && r.Rows > 1
}

// ChildRows is how tall the child is: one row shorter, unless there is no room
// to reserve from.
//
// It never returns zero. A terminal too short gives the child the whole thing
// and the row is simply not drawn -- a zero-row pty is not a thing, and
// clamping here keeps every caller from re-deciding it.
func (r Reservation) ChildRows() uint16 {
	if r.Rows == 0 {
		return 1
	}
	if !r.usable() {
		return r.Rows
	}
	return r.Rows - 1
}

// Release resets the region. Written on teardown, or a child that set margins
// and died would leave the operator's shell scrolling inside a box.
func (r Reservation) Release() string { return ResetRegion }

// ReserveAndPaint asserts the region AND draws the row, in the one order that
// leaves the child's cursor where it was.
//
// ORDER IS THE WHOLE POINT. `SetRegion` (DECSTBM) HOMES THE CURSOR as a
// documented side effect, so `Reserve() + Paint(text)` saves a cursor that is
// already at 1,1 and faithfully restores it there. Measured 2026-09-08: the
// operator's shell prompt sat at the bottom of the pane while the caret blinked
// on row 1. Save first, then set the region, then draw.
//
// couch had the same latent bug at console.go and never saw it: its child is a
// full-screen TUI that repositions the cursor on every frame, so the damage was
// overwritten before anyone could look at it. A shell does not.
func (r Reservation) ReserveAndPaint(text string) string {
	if !r.usable() {
		return ""
	}
	return SaveCursor + SetRegion(1, int(r.Rows)-1) + r.drawRow(text) + RestoreCursor
}

// drawRow is the shared tail of both painters: position, reset, erase, draw,
// reset. ONE spelling, so the two differ only in what they say they differ in --
// whether the region is re-asserted first.
//
// They were two copies of these five sequences, and the test comparing them
// checked only that the region substring was present, so a change to one (a
// different erase, a hide-cursor) would not have reached the other.
//
// Reset BEFORE the erase: ClearLine paints with the CURRENT background, so
// without this the row is erased in the child's colour and the text drawn in its
// foreground. RestoreCursor (DECRC) puts the child's attributes back afterwards,
// so this costs the child nothing.
func (r Reservation) drawRow(text string) string {
	return MoveTo(int(r.Rows), 1) + ResetSGR + ClearLine + text + ResetSGR
}

// Paint draws the reserved row without disturbing the child.
//
// Save and restore BRACKET the paint. Without them the child's cursor is left
// on the reserved row, which the operator sees as the caret jumping to the
// bottom line every time anything is drawn.
//
// DELIBERATE CHANGE from the couchtty.PaintRow this replaces: on a ONE-ROW
// terminal the old function still painted, because it guarded only rows == 0.
// But a one-row terminal cannot be reserved from -- ChildRows gives the child
// all of it -- so that paint landed on a row the child fully owns, overwriting
// its content and leaving no strip anyway. Reserve and Paint now agree: if the
// row was never reserved, nothing is drawn on it.
func (r Reservation) Paint(text string) string {
	if !r.usable() {
		return ""
	}
	return SaveCursor + r.drawRow(text) + RestoreCursor
}
