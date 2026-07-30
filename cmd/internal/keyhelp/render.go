package keyhelp

import (
	"strings"

	"github.com/xianxu/pair/cmd/internal/textwidth"
)

// Render lays the sections out as plain text, aligning each section's
// descriptions against that section's widest key.
//
// Alignment is PER SECTION: a long key like Ctrl+Alt+n in one group must not
// indent every other group's descriptions off to the right.
func Render(sections []Section) string {
	var b strings.Builder
	for i, s := range sections {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(s.Title)
		b.WriteString("\n")
		w := 0
		for _, bind := range s.Bindings {
			if n := textwidth.Width(bind.Key); n > w {
				w = n
			}
		}
		for _, bind := range s.Bindings {
			b.WriteString("  ")
			b.WriteString(textwidth.Pad(bind.Key, w+2))
			b.WriteString(bind.Desc)
			b.WriteString("\n")
		}
	}
	return b.String()
}

// Center indents every line so the block sits centred in cols terminal columns.
//
// This lives in Go because bin/pair-help measured width with `awk '{ ...length... }'`,
// which counts BYTES — so glyph keys (Alt+⏎, Alt+←) inflated the measurement and
// skewed the centring. Doing it here reuses the one column-measuring rule
// (textwidth) instead of adding a second, wrong one in shell (#132).
//
// Blank lines are left empty rather than padded, so the output carries no trailing
// whitespace on the section separators.
func Center(s string, cols int) string {
	lines := strings.Split(s, "\n")
	widest := 0
	for _, ln := range lines {
		if n := textwidth.Width(ln); n > widest {
			widest = n
		}
	}
	pad := (cols - widest) / 2
	if pad <= 0 {
		return s
	}
	prefix := strings.Repeat(" ", pad)
	for i, ln := range lines {
		if ln == "" {
			continue
		}
		lines[i] = prefix + ln
	}
	return strings.Join(lines, "\n")
}
