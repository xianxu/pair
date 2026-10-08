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

	// PointerLabel is the pointer-link control, drawn right after
	// LiveLabel + " " (#412); ControlLabel is reserved for remote control
	// (#407). Both are emoji, two columns wide.
	PointerLabel = "👆"
	ControlLabel = "👽"
	// PointerSGR marks pointing as on: an amber background.
	PointerSGR = "\x1b[48;5;214m"
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

// pointerBackground is the cell background PointerSGR produces.
var pointerBackground = ansi.IndexedColor(214)

// pointerColumn is where Couch draws PointerLabel: right after LiveLabel and
// one space.
var pointerColumn = textwidthOf(LiveLabel) + 1

// PointerShown reports whether f's last row shows the active pointer marker:
// PointerLabel on the pointer background at pointerColumn, after LIVE. It is
// the pointer's visible-capability check, as IndicatorShown is LIVE's (#412).
func PointerShown(f terminal.Frame) bool {
	cols, rows := f.Geometry.Cols, f.Geometry.Rows
	if !IndicatorShown(f) || pointerColumn+2 > cols || len(f.Cells) != cols*rows {
		return false
	}
	c := f.Cells[(rows-1)*cols+pointerColumn]
	return c.Content == PointerLabel && c.Width == 2 && c.Style.Bg == pointerBackground
}

func textwidthOf(s string) int { return ansi.StringWidth(s) }

// StatusGuardCols is the width of the broadcast's controls at the left of
// Couch's status row, `LIVE ⏸ 👆 👽`. Marks never cover these columns of the
// last row (#412); the rest of the tab bar can be pointed at.
var StatusGuardCols = textwidthOf(LiveLabel + " " + PointerLabel + " " + ControlLabel)
