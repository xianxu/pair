// Package textwidth measures and pads strings in TERMINAL COLUMNS.
//
// Extracted from launcher/list.go in #132 so the keybind help and `pair list`
// share one implementation. Three units are in play across Pair — bytes (zellij's
// socket-name budget), runes (Go's string ops), and columns (what a terminal
// actually renders) — and each belongs to a different question. Anything aligning
// output belongs here; anything counting a budget does not.
package textwidth

import "strings"

// Width counts terminal columns. Only the wide case that actually occurs in
// Pair's output is modeled: emoji and other East-Asian-Wide runes take two
// columns.
func Width(s string) int {
	w := 0
	for _, r := range s {
		if isWide(r) {
			w += 2
			continue
		}
		w++
	}
	return w
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

// isWide covers the double-width ranges Pair's output can contain — the emoji
// planes plus the CJK/symbol blocks that render wide in a terminal.
func isWide(r rune) bool {
	switch {
	case r >= 0x1100 && r <= 0x115F, // Hangul Jamo
		r >= 0x2E80 && r <= 0xA4CF, // CJK radicals .. Yi
		r >= 0xAC00 && r <= 0xD7A3, // Hangul syllables
		r >= 0xF900 && r <= 0xFAFF, // CJK compatibility ideographs
		r >= 0xFE30 && r <= 0xFE6F, // CJK compatibility forms
		r >= 0xFF00 && r <= 0xFF60, // fullwidth forms
		r >= 0xFFE0 && r <= 0xFFE6,
		r >= 0x1F300 && r <= 0x1F64F, // misc symbols + pictographs, emoticons
		r >= 0x1F900 && r <= 0x1F9FF, // supplemental symbols + pictographs
		r >= 0x20000 && r <= 0x3FFFD:
		return true
	}
	return false
}
