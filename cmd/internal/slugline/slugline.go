// Package slugline is the Go definition of the orientation slug's line format,
// `=== <left> | <focus> ===`. pair-slug writes it and couch's focus view shows
// it unfenced (pair#372); nvim/slug.lua mirrors the recognition for the
// draft's first line, so a format change touches both.
package slugline

import (
	"regexp"
	"strings"
)

const (
	fenceOpen  = "=== "
	fenceClose = " ==="
	sep        = " | "
)

// lineRE: two non-empty segments separated by " | ", fenced by "=== " / " ===".
var lineRE = regexp.MustCompile(`^=== .+ \| .+ ===$`)

// Valid reports whether s is a well-formed two-segment slug line.
func Valid(s string) bool { return lineRE.MatchString(s) }

// Format assembles a slug line from its two segments.
func Format(left, focus string) string { return fenceOpen + left + sep + focus + fenceClose }

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
	return strings.TrimSuffix(strings.TrimPrefix(line, fenceOpen), fenceClose)
}
