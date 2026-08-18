package wrapcmd

import (
	"image/color"

	uv "github.com/charmbracelet/ultraviolet"
)

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
	if !snapshot.CursorVisible || !snapshotCoordinatesValid(snapshot) || snapshot.Cursor.X != 2 {
		return false
	}
	promptY := snapshot.Cursor.Y
	if promptY <= 0 || promptY >= snapshot.Height-1 {
		return false
	}

	prompt := snapshot.CellAt(0, promptY)
	if prompt == nil || prompt.Content != "⟩" || prompt.Style.Attrs&uv.AttrFaint != 0 {
		return false
	}
	return faintRuleAt(snapshot, 0, promptY-1) && faintRuleAt(snapshot, 0, promptY+1)
}

func snapshotCoordinatesValid(snapshot terminalSnapshot) bool {
	return snapshot.Width > 0 && snapshot.Height > 0 &&
		len(snapshot.Cells) >= snapshot.Width*snapshot.Height &&
		snapshot.Cursor.X >= 0 && snapshot.Cursor.X < snapshot.Width &&
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
