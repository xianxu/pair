// Package textwidth measures and pads strings in TERMINAL COLUMNS.
//
// Extracted from launcher/list.go in #132 so the keybind help and `pair list`
// share one implementation. Three units are in play across Pair — bytes (zellij's
// socket-name budget), runes (Go's string ops), and columns (what a terminal
// actually renders) — and each belongs to a different question. Anything aligning
// output belongs here; anything counting a budget does not.
package textwidth

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"
)

// Width counts plain text in terminal columns using the same grapheme metric as
// terminal.StyledRows. Leading zero-width scalars are ignored, as in that parser;
// selectors, modifiers and joiners within a printable cluster stay together.
func Width(s string) int {
	if printableASCII(s) {
		return len(s)
	}
	w := 0
	for s != "" {
		_, width, rest := nextCluster(s)
		w += width
		s = rest
	}
	return w
}

// Fit returns the longest prefix fitting the column budget without splitting a
// grapheme. Unsupported leading zero-width scalars are dropped, matching chrome.
func Fit(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if printableASCII(s) {
		return s[:min(len(s), width)]
	}
	original := s
	var out strings.Builder
	used, end := 0, 0
	rewritten := false
	for s != "" {
		cluster, w, rest := nextCluster(s)
		if used+w > width {
			break
		}
		consumed := len(s) - len(rest)
		// Most text needs only a prefix slice, including complete Unicode
		// clusters. Allocate only when ignored leading scalars create a gap.
		if consumed != len(cluster) && !rewritten {
			out.WriteString(original[:end])
			rewritten = true
		}
		if rewritten {
			out.WriteString(cluster)
		}
		used += w
		end += consumed
		s = rest
	}
	if rewritten {
		return out.String()
	}
	return original[:end]
}

func printableASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < ' ' || s[i] > '~' {
			return false
		}
	}
	return true
}

func nextCluster(s string) (cluster string, width int, rest string) {
	for s != "" {
		_, n := utf8.DecodeRuneInString(s)
		_, leadWidth := ansi.FirstGraphemeCluster(s[:n], ansi.GraphemeWidth)
		if leadWidth == 0 {
			s = s[n:]
			continue
		}
		cluster, width = ansi.FirstGraphemeCluster(s, ansi.GraphemeWidth)
		return cluster, width, s[len(cluster):]
	}
	return "", 0, ""
}

// Pad left-aligns s in a field of w terminal columns.
//
// Go's %-30s pads by rune count, which a 📁 prefix breaks (#130): one rune, two
// columns, so every row sits a column left of its header. The keybind help hits
// the same thing with glyph keys like Alt+⏎ (#132).
func Pad(s string, w int) string {
	if pad := w - Width(s); pad > 0 {
		return s + strings.Repeat(" ", pad)
	}
	return s
}
