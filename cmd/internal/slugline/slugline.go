// Package slugline is the one definition of the orientation slug's line format,
// `=== <left> | <focus> ===`. pair-slug writes it, nvim shows it as the draft's
// first line, and couch's focus view shows it unfenced (pair#372).
package slugline

import (
	"regexp"
	"strings"
)

const (
	open  = "=== "
	close = " ==="
	sep   = " | "
)

// lineRE: two non-empty segments separated by " | ", fenced by "=== " / " ===".
var lineRE = regexp.MustCompile(`^=== .+ \| .+ ===$`)

// Valid reports whether s is a well-formed two-segment slug line.
func Valid(s string) bool { return lineRE.MatchString(s) }

// Format assembles a slug line from its two segments.
func Format(left, focus string) string { return open + left + sep + focus + close }

// Focus extracts the <focus> segment from a valid slug line.
func Focus(line string) string {
	inner := Unfenced(line)
	if i := strings.Index(inner, sep); i >= 0 {
		return inner[i+len(sep):]
	}
	return inner
}

// Unfenced is the line without its "=== " / " ===" fence: `<left> | <focus>`.
func Unfenced(line string) string {
	return strings.TrimSuffix(strings.TrimPrefix(line, open), close)
}
