// Package rowtext makes untrusted text safe to put on one row of a terminal.
//
// Two consumers need exactly this pair and neither may have its own copy:
// couch's status row renders agent-published labels and notices, and `pair
// term`'s tab strip renders operator-typed tab names plus, since #199 M2,
// whatever a failing `zellij action` wrote to stderr. All three sources are
// outside our control, and all three land on a live terminal.
//
// The hazard is not hypothetical. A label containing \x1b[2J clears the
// operator's screen from the status row; a `ps`-derived name carrying ^N (SO)
// switches the terminal to the alternate character set and garbles every line
// after it (measured in #208); a newline splits a row and can forge a line of
// someone else's output inside it.
//
// It lives in its own package because couchtty's versions were UNEXPORTED, so
// "reuse them" was not implementable from termcmd (#199 PQ-7) -- and a second
// copy of a security-relevant strip, in a package whose tests do not cover it,
// is the outcome that rule exists to prevent.
package rowtext

import (
	"strings"

	"github.com/xianxu/pair/cmd/internal/ansi"
	"github.com/xianxu/pair/cmd/internal/textwidth"
)

// Sanitize removes complete escape sequences and every remaining control byte.
//
// ansi.Strip is the repo's existing answer to "remove complete escape
// sequences", so the sequence framing is not re-decided here; the rune pass
// then catches the leftovers a partial sequence leaves behind, plus the bare
// control bytes (^N, \r, \n, DEL) that were never part of a sequence at all.
func Sanitize(s string) string {
	stripped := string(ansi.Strip([]byte(s)))
	return strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, stripped)
}

// Fit truncates to a display WIDTH, not a byte or rune count: an emoji in an
// agent's description is one rune and two columns, and a row that overflows
// wraps onto the child's area.
func Fit(s string, width int) string {
	if width <= 0 {
		return ""
	}
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

// SanitizeAndFit is the pair, in the order that matters: sanitizing first can
// only shorten, so fitting afterwards is what actually bounds the width.
func SanitizeAndFit(s string, width int) string { return Fit(Sanitize(s), width) }
