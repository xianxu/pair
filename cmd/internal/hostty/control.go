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

	// ResetSGR clears colour and attributes.
	//
	// A reserved row MUST emit this before it erases and draws. An ERASE paints
	// with the CURRENT background, and text inherits the current foreground, so
	// a row drawn straight after a child's output wears whatever colour that
	// child last set -- measured 2026-09-08: `pair term`'s tab strip came out
	// in nvim's lualine colours, because lualine is the last thing that sets
	// SGR before the row is painted.
	ResetSGR = "\x1b[0m"

	// HomeAndClear is the prelude to a repaint: reset, home, erase.
	//
	// THE RESET IS PART OF THE ERASE, not decoration in front of it. `\x1b[J`
	// paints the cleared region with the CURRENT background, so a clear issued
	// while a child's colour is active tints the whole screen -- and the next
	// child to write inherits it. Measured 2026-09-08: opening a tab after nvim
	// gave the new shell nvim's lualine blue behind its first lines, because
	// `pair term`'s tab takeover cleared while lualine's SGR was still set.
	//
	// Folded into the constant rather than left to each caller: there are two
	// (termcmd's takeover, couch's takeOverScreen) and both want the same thing,
	// which is what makes "remember to reset first" the wrong shape.
	HomeAndClear = "\x1b[0m\x1b[1;1H\x1b[J"

	// LeaveAltScreen and ShowCursor are unconditional teardown guards. A child
	// may die or couch may be signalled before it emits its own paired restore.
	LeaveAltScreen = "\x1b[?1049l"
	ShowCursor     = "\x1b[?25h"
	HideCursor     = "\x1b[?25l"

	// ResetInteractiveModes returns input/display handling to a shell-safe
	// baseline after Couch has replayed a child's terminal modes. Raw termios
	// restoration does not revoke DEC private modes: without this, any-event
	// mouse tracking makes ordinary pointer movement type SGR escape bytes into
	// the returned shell. The grouped DECRST also disables legacy/SGR/pixel
	// mouse encodings, focus events, alternate scroll, bracketed paste, and
	// synchronized output. CSI = 0 u disables Kitty extended-key reporting.
	ResetInteractiveModes = "\x1b[?9;1000;1001;1002;1003;1004;1005;1006;1007;1015;1016;2004;2026l\x1b[=0u"

	// EnableMouseClicks asks for CLICK reporting (?1000) in SGR encoding
	// (?1006), and nothing else.
	//
	// Not ?1002/?1003: motion reports arrive at pointer-movement rates, and the
	// feature that needs this wants human click rates. Not the legacy encoding:
	// it caps coordinates at 223 and fails SILENTLY on a wide or tall terminal,
	// which is a wrong answer rather than a refused one.
	EnableMouseClicks = "\x1b[?1000;1006h"
)

// SetRegion pins the scrolling region to rows top..bottom (1-based, inclusive).
// This is how a row is RESERVED without compositing: a child scrolling at the
// bottom of its own screen scrolls inside the region and cannot reach the row
// below it.
func SetRegion(top, bottom int) string { return fmt.Sprintf("\x1b[%d;%dr", top, bottom) }

// MoveTo positions the cursor (1-based).
func MoveTo(row, col int) string { return fmt.Sprintf("\x1b[%d;%dH", row, col) }
