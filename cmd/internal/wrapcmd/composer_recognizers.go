package wrapcmd

import (
	"image/color"
	"strings"

	uv "github.com/charmbracelet/ultraviolet"
)

const (
	// codexComposerPrompt is the marker Codex paints at column 0 of the
	// composer's first row. Codex reuses the same glyph as a menu selection
	// marker, so the glyph alone never qualifies a composer.
	codexComposerPrompt = "\u203a"
	// codexComposerTextColumn is the first column Codex leaves for composer
	// text: the prompt owns column 0 and column 1 is its trailing space.
	// Continuation rows keep both columns blank and align text here.
	codexComposerTextColumn = 2
	// codexComposerMaxRows bounds how far above the cursor a prompt may sit
	// before the block stops reading as one composer.
	codexComposerMaxRows = 20
)

// codexComposerActive reports whether the cursor rests inside Codex's live
// composer. The composer is one block whose left edge is a bold prompt glyph at
// column 0; rows below it keep columns 0 and 1 blank and align text at
// codexComposerTextColumn. Codex paints the same glyph unemphasized as a menu
// selection marker and parks the cursor on unrelated rows mid-paint, so a
// qualifying cursor must reach its prompt through painted continuation rows
// only. Its own row may be an empty composer line; a gap anywhere above it
// proves the cursor sits below the composer rather than inside it.
func codexComposerActive(snapshot terminalSnapshot) bool {
	if !snapshot.CursorVisible || !snapshotCoordinatesValid(snapshot) ||
		snapshot.Cursor.X < codexComposerTextColumn {
		return false
	}

	for promptY := snapshot.Cursor.Y; promptY >= 0 && snapshot.Cursor.Y-promptY < codexComposerMaxRows; promptY-- {
		if codexComposerRowPaintsLeftEdge(snapshot, promptY) {
			prompt := snapshot.CellAt(0, promptY)
			return prompt != nil && prompt.Content == codexComposerPrompt &&
				prompt.Style.Attrs&uv.AttrBold != 0
		}
		if promptY != snapshot.Cursor.Y && !codexComposerContinuationRow(snapshot, promptY) {
			return false
		}
	}
	return false
}

// codexComposerRowPaintsLeftEdge reports whether row y paints anything in the
// columns Codex reserves for the composer prompt. Such a row is the block's
// left edge: either the composer's own prompt row or unrelated surface.
func codexComposerRowPaintsLeftEdge(snapshot terminalSnapshot, y int) bool {
	for x := 0; x < codexComposerTextColumn && x < snapshot.Width; x++ {
		if cell := snapshot.CellAt(x, y); cell != nil && strings.TrimSpace(cell.Content) != "" {
			return true
		}
	}
	return false
}

// codexComposerContinuationRow reports whether row y carries composer text: it
// paints nothing before codexComposerTextColumn and something at or after it.
func codexComposerContinuationRow(snapshot terminalSnapshot, y int) bool {
	if codexComposerRowPaintsLeftEdge(snapshot, y) {
		return false
	}
	for x := codexComposerTextColumn; x < snapshot.Width; x++ {
		if cell := snapshot.CellAt(x, y); cell != nil && strings.TrimSpace(cell.Content) != "" {
			return true
		}
	}
	return false
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
