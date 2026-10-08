package broadcast

import (
	"github.com/charmbracelet/x/ansi"
	"github.com/xianxu/pair/cmd/internal/terminal"
)

// The status row draws these and IndicatorShown checks them: one definition,
// so the drawer and the checker can't disagree.
const (
	LiveLabel     = "LIVE ⏸"
	StartingLabel = "LIVE …"
	// LiveSGR is the style both labels are drawn in: a red background and the
	// terminal's own foreground.
	LiveSGR = "\x1b[41m"
)

// liveBackground is the cell background LiveSGR produces.
var liveBackground = ansi.Red

// IndicatorShown reports whether f's last row begins, at column zero, with
// LiveLabel drawn on the live background. A broadcast forwards only frames for
// which this holds: what viewers get, the operator saw marked as live.
func IndicatorShown(f terminal.Frame) bool {
	cols, rows := f.Geometry.Cols, f.Geometry.Rows
	if cols <= 0 || rows <= 0 || len(f.Cells) != cols*rows {
		return false
	}
	row := f.Cells[(rows-1)*cols:]
	x := 0
	for rest := LiveLabel; rest != ""; {
		g, _ := ansi.FirstGraphemeCluster(rest, ansi.GraphemeWidth)
		rest = rest[len(g):]
		if x >= cols {
			return false
		}
		c := row[x]
		if c.Content != g || c.Style.Bg != liveBackground {
			return false
		}
		x += max(1, c.Width)
	}
	return true
}
