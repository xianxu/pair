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
type Reservation struct {
	Rows uint16
	Edge Edge
}

// NewReservation is the validating constructor: it refuses an edge that is not
// implemented rather than letting a caller find out by seeing a corrupt screen.
func NewReservation(rows uint16, edge Edge) (Reservation, error) {
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

// Reserve pins the scrolling region above the reserved row.
func (r Reservation) Reserve() string {
	if !r.usable() {
		return ""
	}
	return SetRegion(1, int(r.Rows)-1)
}

// Release resets the region. Written on teardown, or a child that set margins
// and died would leave the operator's shell scrolling inside a box.
func (r Reservation) Release() string { return ResetRegion }

// Paint draws the reserved row without disturbing the child.
//
// Save and restore BRACKET the paint. Without them the child's cursor is left
// on the reserved row, which the operator sees as the caret jumping to the
// bottom line every time anything is drawn.
func (r Reservation) Paint(text string) string {
	if !r.usable() {
		return ""
	}
	return SaveCursor +
		MoveTo(int(r.Rows), 1) +
		ClearLine +
		text +
		RestoreCursor
}
