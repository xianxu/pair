// Package hostty owns the OPERATOR's terminal: how big it is, when it changes
// size, putting it in raw mode and reliably putting it back, and the escape
// sequences that address it as a whole.
//
// It is the host half of the terminal plumbing `pair term` and `couch` share;
// cmd/internal/ptychild is the child half. Splitting them this way is what makes
// a console testable without a real tty -- FakeHost is scriptable, so the
// SIGWINCH path and the restore-on-signal path are covered by tests rather than
// only by an operator smoke.
//
// The control sequences live here, one constant per sequence, because two sites
// framing the same sequence independently is a bug this repo has already paid
// for (#127's dead keyboard). `\x1b[r` in particular was about to exist in both
// termcmd and couch.
package hostty

import "fmt"

const (
	// ResetRegion clears DECSTBM, restoring the full-screen scrolling region.
	// Written on teardown: a child that set margins and died would otherwise
	// leave the operator's shell scrolling inside a box.
	ResetRegion = "\x1b[r"

	// SaveCursor / RestoreCursor bracket anything drawn outside the child's
	// area, so the child's cursor is where it left it.
	SaveCursor    = "\x1b7"
	RestoreCursor = "\x1b8"

	// ClearLine erases the row the cursor is on.
	ClearLine = "\x1b[2K"

	// HomeAndClear is the prelude to a repaint.
	HomeAndClear = "\x1b[1;1H\x1b[J"
)

// SetRegion pins the scrolling region to rows top..bottom (1-based, inclusive).
// This is how a row is RESERVED without compositing: a child scrolling at the
// bottom of its own screen scrolls inside the region and cannot reach the row
// below it.
func SetRegion(top, bottom int) string { return fmt.Sprintf("\x1b[%d;%dr", top, bottom) }

// MoveTo positions the cursor (1-based).
func MoveTo(row, col int) string { return fmt.Sprintf("\x1b[%d;%dH", row, col) }
