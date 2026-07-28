package layoutcmd

// Pure planning for the Alt+Shift+Enter terminal-width toggle. The right
// terminal column re-tiles between half the screen (normal, 50/50) and about
// two thirds (expanded, 65/35 — the 1/3–2/3 arrangement). Zellij 0.44.3's
// tiled resize step is a stable fraction of the screen (RESIZE_PERCENT = 5%
// per `resize increase|decrease left`; measured live: 7–8 cols on a 150-col
// screen, and back-to-back actions all apply), so the toggle delta is always
// exactly three steps — fired blind as one burst (#124), replacing the #123
// converge loop whose settle pauses made the toggle feel slow. If zellij
// refuses a step (minimum sizes) the extra action is a harmless no-op; a
// future zellij step-size change degrades to a different stable pair of
// widths, still classified by the 60% threshold.

// terminalToggleSteps is the burst length: the 1/2 ↔ ~2/3 delta (≈17% of
// the screen) at zellij's 5%-per-step resize → 3 steps = 15%, landing at
// 65/35 expanded and exactly 50/50 collapsed.
const terminalToggleSteps = 3

// terminalToggleBurst returns the fixed action burst toggling the right
// terminal column: at ≥60% of the screen the terminal reads as expanded and
// collapses; otherwise it expands. The terminal owns the column's left edge,
// so growing pushes that edge left and shrinking pulls it right. Returns
// false when the geometry is unusable.
func terminalToggleBurst(terminalCols, screenCols int) ([][]string, bool) {
	if terminalCols <= 0 || screenCols <= 0 || terminalCols > screenCols {
		return nil, false
	}
	verb := "increase"
	if terminalCols*100 >= screenCols*60 {
		verb = "decrease"
	}
	burst := make([][]string, terminalToggleSteps)
	for i := range burst {
		burst[i] = []string{"resize", verb, "left"}
	}
	return burst, true
}
