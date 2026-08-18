package wrapcmd

import (
	"image/color"

	uv "github.com/charmbracelet/ultraviolet"
)

const codexComposerMinRows = 2

func codexComposerActive(snapshot terminalSnapshot) bool {
	if !snapshot.CursorVisible || !snapshotCoordinatesValid(snapshot) {
		return false
	}

	paintedRows := 0
	for y := max(0, snapshot.Cursor.Y-1); y <= min(snapshot.Height-1, snapshot.Cursor.Y+1); y++ {
		if rowHasBackground(snapshot, y, 57, 57, 57) {
			paintedRows++
		}
	}
	return paintedRows >= codexComposerMinRows
}

func museComposerActive(snapshot terminalSnapshot) bool {
	if !snapshot.CursorVisible || !snapshotCoordinatesValid(snapshot) || snapshot.Cursor.X < 2 {
		return false
	}

	for promptY := max(1, snapshot.Cursor.Y-1); promptY <= min(snapshot.Height-2, snapshot.Cursor.Y+1); promptY++ {
		prompt := snapshot.CellAt(0, promptY)
		if prompt != nil && prompt.Content == "⟩" && prompt.Style.Attrs&uv.AttrFaint == 0 &&
			faintRuleAt(snapshot, 0, promptY-1) && faintRuleAt(snapshot, 0, promptY+1) {
			return true
		}
	}
	return false
}

func agyComposerActive(snapshot terminalSnapshot) bool {
	if !snapshot.CursorVisible || !snapshotCoordinatesValid(snapshot) {
		return false
	}

	const (
		minBorderLength = 5
		maxBoxHeight    = 25
		promptColumns   = 6
	)
	borderCovers := make([][promptColumns]bool, snapshot.Height)
	for y := 0; y < snapshot.Height; y++ {
		for x := 0; x < snapshot.Width; {
			if cell := snapshot.CellAt(x, y); cell == nil || cell.Content != "─" {
				x++
				continue
			}
			start := x
			for x < snapshot.Width {
				cell := snapshot.CellAt(x, y)
				if cell == nil || cell.Content != "─" {
					break
				}
				x++
			}
			end := x - 1
			if x-start >= minBorderLength && snapshot.Cursor.X >= start && snapshot.Cursor.X <= end {
				for promptX := max(0, start); promptX <= min(promptColumns-1, end); promptX++ {
					borderCovers[y][promptX] = true
				}
			}
		}
	}

	for promptX := 0; promptX < promptColumns && promptX < snapshot.Width; promptX++ {
		promptPrefix := make([]int, snapshot.Height+1)
		for y := 0; y < snapshot.Height; y++ {
			promptPrefix[y+1] = promptPrefix[y]
			if cell := snapshot.CellAt(promptX, y); cell != nil && cell.Content == ">" {
				promptPrefix[y+1]++
			}
		}
		for top := 0; top < snapshot.Cursor.Y; top++ {
			if !borderCovers[top][promptX] {
				continue
			}
			for bottom := snapshot.Cursor.Y + 1; bottom < snapshot.Height && bottom-top <= maxBoxHeight; bottom++ {
				if borderCovers[bottom][promptX] && promptPrefix[bottom]-promptPrefix[top+1] > 0 {
					return true
				}
			}
		}
	}
	return false
}

func snapshotCoordinatesValid(snapshot terminalSnapshot) bool {
	if validateTerminalDimensions(snapshot.Width, snapshot.Height) != nil ||
		snapshot.Height > len(snapshot.Cells)/snapshot.Width {
		return false
	}
	return snapshot.Cursor.X >= 0 && snapshot.Cursor.X < snapshot.Width &&
		snapshot.Cursor.Y >= 0 && snapshot.Cursor.Y < snapshot.Height
}

func rowHasBackground(snapshot terminalSnapshot, y int, r, g, b uint8) bool {
	for x := 0; x < snapshot.Width; x++ {
		cell := snapshot.CellAt(x, y)
		if cell != nil && colorMatches(cell.Style.Bg, r, g, b) {
			return true
		}
	}
	return false
}

func colorMatches(value color.Color, wantR, wantG, wantB uint8) bool {
	if value == nil {
		return false
	}
	r, g, b, _ := value.RGBA()
	return uint8(r>>8) == wantR && uint8(g>>8) == wantG && uint8(b>>8) == wantB
}

func faintRuleAt(snapshot terminalSnapshot, x, y int) bool {
	cell := snapshot.CellAt(x, y)
	return cell != nil && cell.Content == "─" && cell.Style.Attrs&uv.AttrFaint != 0
}
