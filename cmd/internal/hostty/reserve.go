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
// Since #255 M3 production no longer paints through it. Couch and `pair term`
// compose their strips into frames via terminal.Presenter.UpdateChrome, and
// the presenter is the parent's sole production writer. In production the
// region reset (`\x1b[r`) is written by its renderers and release controls
// (#262). Release here still writes it, but only for the probe. Production uses
// a Reservation only for ChildRows arithmetic; ReserveAndPaint and Release
// serve cmd/probes/couchnestedrows. Whether they survive is pair#281.
//
// That `pair term` can do this at all is measured, not assumed: zellij honors
// DECSTBM from a pane process (pair#199 finding 5 -- 200 lines scrolled in the
// region while the reserved row held its paint).

// The sequences ReserveAndPaint and Release write. Unexported, and here rather
// than in a shared control file, because nothing else uses them: the presenter
// spells its own (#289).
const (
	saveCursor    = "\x1b7"
	restoreCursor = "\x1b8"
	resetRegion   = "\x1b[r"
	resetSGR      = "\x1b[0m"
	clearLine     = "\x1b[2K"
)

// setRegion pins the scrolling region to rows top..bottom (1-based, inclusive).
func setRegion(top, bottom int) string { return fmt.Sprintf("\x1b[%d;%dr", top, bottom) }

// moveTo positions the cursor (1-based).
func moveTo(row, col int) string { return fmt.Sprintf("\x1b[%d;%dH", row, col) }

// Edge is which end of the terminal a reservation takes.
type Edge int

const (
	// EdgeBottom is the only implemented edge, and the zero value so that a
	// zero Reservation is the safe one.
	EdgeBottom Edge = iota
	// EdgeTop is representable and REFUSED. A top reservation is not the mirror
	// of a bottom one: with the region set to 2..N the child still addresses
	// absolute rows, so its row 1 IS the strip and any absolute positioning it
	// does lands on top of us.
	//
	// Origin mode (DECOM, `\x1b[?6h`) is NOT enough, and that is measured
	// (#223, probes/zellijwrapmargin top:*): it moves the child's CUP, but
	// DECSTBM parameters stay ABSOLUTE under it, so a full-screen child's own
	// scroll region — nvim's, when it scrolls — includes a top strip; real nvim,
	// after a few half-page scrolls and a jump, left its own buffer line on the
	// strip row. A top edge would have to
	// rewrite the child's DECSTBM parameters in flight, and police its DECOM,
	// RIS and buffer switches too; zellij's DECRC does not even restore DECOM.
	// Named rather than omitted so the asymmetry is discoverable instead of
	// being rediscovered.
	EdgeTop
)

// Reservation is a terminal of Rows rows with one row held at Edge.
//
// It answers ChildRows, ReserveAndPaint and Release. There is no bare
// `Reserve()`: a painting caller (today only cmd/probes/couchnestedrows; before
// #255 M3, couch and `pair term`) asserts the region and draws the row together,
// in that order, because DECSTBM homes the cursor -- so a caller that could
// reserve WITHOUT painting is a caller that can compose the two in the order
// that was the bug.
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
				"(a top strip would need the child's DECSTBM rewritten in flight — its "+
				"parameters are absolute even under origin mode)", edge)
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
func (r Reservation) Release() string { return resetRegion }

// ReserveAndPaint asserts the region AND draws the row, in the one order that
// leaves the child's cursor where it was.
//
// ORDER IS THE WHOLE POINT. `setRegion` (DECSTBM) HOMES THE CURSOR as a
// documented side effect, so setting the region before saving saves a cursor
// already at 1,1 and faithfully restores it there. Measured 2026-09-08: the
// operator's shell prompt sat at the bottom of the pane while the caret blinked
// on row 1. Save first, then set the region, then draw.
//
// couch had the same latent bug at console.go and never saw it: its child is a
// full-screen TUI that repositions the cursor on every frame, so the damage was
// overwritten before anyone could look at it. A shell does not.
//
// Reset BEFORE the erase: clearLine paints with the CURRENT background, so
// without it the row is erased in the child's colour and the text drawn in its
// foreground. Measured 2026-09-08: `pair term`'s tab strip came out in nvim's
// lualine colours. restoreCursor (DECRC) puts the child's attributes back
// afterwards, so this costs the child nothing.
func (r Reservation) ReserveAndPaint(text string) string {
	if !r.usable() {
		return ""
	}
	return saveCursor + setRegion(1, int(r.Rows)-1) +
		moveTo(int(r.Rows), 1) + resetSGR + clearLine + text + resetSGR +
		restoreCursor
}
