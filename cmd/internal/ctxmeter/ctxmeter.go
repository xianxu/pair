// Package ctxmeter humanizes a context-window token count for display.
package ctxmeter

import (
	"fmt"
	"math"
	"strconv"
)

// Humanize formats a token count per the spec's pinned rule:
// <1000 exact; 1000≤n<1_000_000 → Nk (round half-up); ≥1_000_000 → N.NM (floor).
func Humanize(n int) string {
	switch {
	case n < 1000:
		return strconv.Itoa(n)
	case n < 1_000_000:
		return strconv.Itoa(int(math.Round(float64(n)/1000))) + "k"
	default:
		return fmt.Sprintf("%.1fM", math.Floor(float64(n)/100_000)/10)
	}
}
