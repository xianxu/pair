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
	if got := r.Reserve(); got != "\x1b[1;23r" {
		t.Fatalf("Reserve = %q; want the region to stop at row 23", got)
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
		if r.Reserve() != "" {
			t.Fatalf("rows=%d reserved from a terminal with no room", rows)
		}
	}
}

// EdgeTop is representable but refused, because a top reservation is NOT the
// mirror of a bottom one: the child addresses rows 1..N-1, so its row 1 IS the
// strip. Correct only under origin mode (DECOM), which nothing here tracks.
func TestTopEdgeIsRefusedUntilOriginModeExists(t *testing.T) {
	if _, err := hostty.NewReservation(24, hostty.EdgeTop); err == nil {
		t.Fatal("EdgeTop accepted; a top strip needs DECOM arbitration first")
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
	if got := r.Reserve(); got != "" {
		t.Fatalf("Reserve on an unsupported edge = %q; want no region at all", got)
	}
	if got := r.ChildRows(); got != 24 {
		t.Fatalf("ChildRows = %d; an unreserved terminal gives the child all %d", got, 24)
	}
	if got := r.Paint("x"); got != "" {
		t.Fatalf("Paint on an unsupported edge = %q; want nothing drawn", got)
	}
}

// Without save/restore the child's cursor is left on the reserved row, which the
// operator sees as the caret jumping to the bottom line on every repaint.
func TestPaintBracketsWithCursorSaveRestore(t *testing.T) {
	got := hostty.Reservation{Rows: 24, Edge: hostty.EdgeBottom}.Paint("hello")

	save := strings.Index(got, "\x1b7")
	restore := strings.Index(got, "\x1b8")
	text := strings.Index(got, "hello")
	if save < 0 || restore < 0 {
		t.Fatalf("Paint lacks save/restore: %q", got)
	}
	if !(save < text && text < restore) {
		t.Fatalf("the paint is not bracketed by save/restore: %q", got)
	}
	if !strings.Contains(got, "\x1b[24;1H") {
		t.Fatalf("Paint did not move to the reserved row: %q", got)
	}
}

func TestReleaseResetsTheRegion(t *testing.T) {
	r := hostty.Reservation{Rows: 24, Edge: hostty.EdgeBottom}
	if !strings.Contains(r.Release(), "\x1b[r") {
		t.Fatalf("Release = %q; want a region reset", r.Release())
	}
}
