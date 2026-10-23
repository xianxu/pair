package sessionsync

import "github.com/xianxu/pair/cmd/internal/editorial"

// readyMarkerIn reports whether text contains an operator-approved request.
func readyMarkerIn(text string) bool { _, _, ok := readyMarkerSpan(text); return ok }

// readyMarkerSpan delegates to the single review-marker grammar owner.
func readyMarkerSpan(text string) (start, end int, ok bool) { return editorial.ReadyMarkerSpan(text) }

// matchBracket retains the package's existing helper seam.
func matchBracket(text string, start int, open, close byte) int {
	return editorial.MatchBracket(text, start, open, close)
}
