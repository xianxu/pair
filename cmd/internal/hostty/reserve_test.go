package hostty_test

import (
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/hostty"
)

// The whole reserved-row design is one off-by-one: the child gets rows-1 and the
// region stops at rows-1, so a child scrolling at ITS bottom line scrolls inside
// the region and cannot reach the row below it.
//
// Measured 2026-09-07 (pair#199 finding 5): zellij honors this from a pane
// process, which is what lets `pair term` reserve a row the same way couch does.
func TestBottomReservationKeepsTheChildOffTheRow(t *testing.T) {
	r := hostty.Reservation{Rows: 24, Edge: hostty.EdgeBottom}
	if got := r.ChildRows(); got != 23 {
		t.Fatalf("ChildRows = %d; want 23", got)
	}
	if got := r.ReserveAndPaint("x"); !strings.Contains(got, "\x1b[1;23r") {
		t.Fatalf("ReserveAndPaint = %q; want the region to stop at row 23", got)
	}
}

// A terminal too short to reserve from gives the child everything and draws
// nothing -- a zero-row pty is not a thing, and clamping here keeps every caller
// from re-deciding it.
func TestDegenerateHeightsNeverProduceAZeroRowChild(t *testing.T) {
	for _, rows := range []uint16{0, 1} {
		r := hostty.Reservation{Rows: rows, Edge: hostty.EdgeBottom}
		if r.ChildRows() == 0 {
			t.Fatalf("rows=%d produced a zero-row child", rows)
		}
		if r.ReserveAndPaint("x") != "" {
			t.Fatalf("rows=%d reserved from a terminal with no room", rows)
		}
	}
}

// EdgeTop is representable but refused, because a top reservation is NOT the
// mirror of a bottom one: the child addresses rows 1..N-1, so its row 1 IS the
// strip. Origin mode alone does not fix it — #223 measured DECSTBM parameters
// staying absolute under DECOM, so a child's own scroll region still includes a
// top strip. Correct only if the child's DECSTBM were rewritten in flight.
func TestTopEdgeIsRefused(t *testing.T) {
	if _, err := hostty.NewReservation(24, hostty.EdgeTop); err == nil {
		t.Fatal("EdgeTop accepted; a top strip needs the child's DECSTBM rewritten in flight, not only DECOM")
	}
	if _, err := hostty.NewReservation(24, hostty.EdgeBottom); err != nil {
		t.Fatalf("EdgeBottom refused: %v", err)
	}
}

// NewReservation is the validating door, but the struct is exported and a caller
// can bypass it. An unsupported edge must then reserve NOTHING rather than emit
// a region computed for the edge it does not implement -- a wrong region is a
// silently corrupted screen, where no region is merely no strip.
func TestAnUnvalidatedTopEdgeFailsClosed(t *testing.T) {
	r := hostty.Reservation{Rows: 24, Edge: hostty.EdgeTop}
	if got := r.ReserveAndPaint("x"); got != "" {
		t.Fatalf("ReserveAndPaint on an unsupported edge = %q; want no region at all", got)
	}
	if got := r.ChildRows(); got != 24 {
		t.Fatalf("ChildRows = %d; an unreserved terminal gives the child all %d", got, 24)
	}
}

// The validating door validates ROWS too. Admitting rows: 0 returns a
// Reservation whose every method no-ops, which reads to the caller as a working
// reservation that simply never draws.
func TestNewReservationRefusesATerminalWithNoRoom(t *testing.T) {
	for _, rows := range []uint16{0, 1} {
		if _, err := hostty.NewReservation(rows, hostty.EdgeBottom); err == nil {
			t.Fatalf("rows=%d accepted; there is no room to reserve from", rows)
		}
	}
	if _, err := hostty.NewReservation(2, hostty.EdgeBottom); err != nil {
		t.Fatalf("rows=2 refused: %v", err)
	}
}

func TestReleaseResetsTheRegion(t *testing.T) {
	r := hostty.Reservation{Rows: 24, Edge: hostty.EdgeBottom}
	if !strings.Contains(r.Release(), "\x1b[r") {
		t.Fatalf("Release = %q; want a region reset", r.Release())
	}
}

// The cursor must survive a repaint, and the ORDER is what decides it.
//
// DECSTBM homes the cursor as a documented side effect, so setting the region
// before saving saves a cursor already moved to 1,1 and restores it there. The
// operator sees their shell prompt at the bottom of the pane and the caret
// blinking on row 1 — measured, not theorised. And the paint must sit inside
// the save/restore bracket, or the child's cursor is left on the reserved row:
// the caret jumps to the bottom line on every repaint.
func TestSaveComesBeforeTheRegionChangeThatHomesTheCursor(t *testing.T) {
	r := hostty.Reservation{Rows: 24, Edge: hostty.EdgeBottom}
	got := r.ReserveAndPaint("strip")

	save := strings.Index(got, "\x1b7")
	region := strings.Index(got, "\x1b[1;23r")
	text := strings.Index(got, "strip")
	restore := strings.Index(got, "\x1b8")
	if save < 0 || region < 0 || restore < 0 {
		t.Fatalf("missing save, region or restore: %q", got)
	}
	if !(save < region) {
		t.Fatalf("the region is set BEFORE the cursor is saved, so the saved "+
			"position is the home DECSTBM moved it to: %q", got)
	}
	if !(region < text && text < restore) {
		t.Fatalf("the paint is not between the region change and the restore: %q", got)
	}
	if !strings.Contains(got, "\x1b[24;1H") {
		t.Fatalf("the row was not addressed: %q", got)
	}
}

// The row must not wear the child's colours.
//
// ERASE paints with the CURRENT background and text inherits the current
// foreground, so a row drawn right after a child's output comes out in whatever
// SGR that child last set. Measured 2026-09-08: `pair term`'s tab strip
// rendered in nvim's lualine colours, lualine being the last thing to set SGR
// before the paint.
func TestTheRowResetsColourBeforeErasingAndDrawing(t *testing.T) {
	got := hostty.Reservation{Rows: 24, Edge: hostty.EdgeBottom}.ReserveAndPaint("strip")
	reset := strings.Index(got, "\x1b[0m")
	clear := strings.Index(got, "\x1b[2K")
	text := strings.Index(got, "strip")
	if reset < 0 {
		t.Fatalf("the row never resets SGR, so it wears the child's colours: %q", got)
	}
	if !(reset < clear) {
		t.Fatalf("the row is erased BEFORE the reset, so it is cleared in the "+
			"child's background colour: %q", got)
	}
	if !(clear < text) {
		t.Fatalf("the row is drawn before it is erased: %q", got)
	}
	// And it must not leak its own attributes back to the child.
	if last := strings.LastIndex(got, "\x1b[0m"); last < text {
		t.Fatalf("the strip's attributes stay active after the text: %q", got)
	}
}
