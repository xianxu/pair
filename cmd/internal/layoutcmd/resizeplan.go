package layoutcmd

// Pure planning for the Alt+Shift+Enter terminal-width toggle. The right
// terminal column re-tiles between half the screen (normal) and two thirds
// (expanded). Zellij's per-step resize amount is a runtime detail we never
// hardcode: the executor loops read-geometry → step → act until the width
// converges, and these functions stay free of IO.

// terminalResizeTolerance is how close (in columns) the terminal width must
// get to its target before the toggle stops stepping.
const terminalResizeTolerance = 2

// terminalResizeMaxSteps caps the resize loop so a misbehaving zellij can
// never spin it; convergence normally takes a handful of steps.
const terminalResizeMaxSteps = 12

// terminalResizeTarget picks the toggle's target width for the right terminal
// column: at ≥60% of the screen the terminal reads as expanded and collapses
// to half; otherwise it expands to two thirds. Returns false when the
// geometry is unusable.
func terminalResizeTarget(terminalCols, screenCols int) (int, bool) {
	if terminalCols <= 0 || screenCols <= 0 || terminalCols > screenCols {
		return 0, false
	}
	if terminalCols*100 >= screenCols*60 {
		return screenCols / 2, true
	}
	return screenCols * 2 / 3, true
}

// terminalResizeStep returns the next zellij action moving the terminal
// toward targetCols, or done=true once the width is within tolerance. The
// terminal owns the column's left edge, so growing means pushing that edge
// left and shrinking means pulling it right.
func terminalResizeStep(currentCols, targetCols int) ([]string, bool) {
	diff := targetCols - currentCols
	if abs(diff) <= terminalResizeTolerance {
		return nil, true
	}
	if diff > 0 {
		return []string{"resize", "increase", "left"}, false
	}
	return []string{"resize", "decrease", "left"}, false
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
