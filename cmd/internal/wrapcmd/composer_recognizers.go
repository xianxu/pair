package wrapcmd

import (
	"image/color"
	"strings"

	xansi "github.com/charmbracelet/x/ansi"

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
// composer. The composer's left edge is a bold prompt glyph at column 0; rows
// below it keep columns 0 and 1 blank, carry text at codexComposerTextColumn,
// and may be entirely blank when the user has opened an empty line. Codex
// paints the same glyph unemphasized as a menu selection marker, so only a bold
// prompt qualifies.
//
// Codex also paints a status line below the composer, separated from it by one
// blank row, and parks the cursor there while repainting. That row is
// cell-identical to a composer continuation row, so it is excluded by position
// rather than by style: the status line is the last painted row on the screen
// and has a blank row above it, which no composer row inside a block can be.
func codexComposerActive(snapshot terminalSnapshot) bool {
	if !snapshot.CursorVisible || !snapshotCoordinatesValid(snapshot) ||
		snapshot.Cursor.X < codexComposerTextColumn ||
		codexCursorOnTrailingStatusRow(snapshot) {
		return false
	}

	for promptY := snapshot.Cursor.Y; promptY >= 0 && snapshot.Cursor.Y-promptY < codexComposerMaxRows; promptY-- {
		if codexComposerRowPaintsLeftEdge(snapshot, promptY) {
			prompt := snapshot.CellAt(0, promptY)
			return prompt != nil && prompt.Content == codexComposerPrompt &&
				prompt.Style.Attrs&uv.AttrBold != 0
		}
	}
	return false
}

// codexCursorOnTrailingStatusRow reports whether the cursor sits on Codex's
// status line: a painted row with a blank row directly above it and nothing
// painted below it anywhere on screen.
func codexCursorOnTrailingStatusRow(snapshot terminalSnapshot) bool {
	y := snapshot.Cursor.Y
	if y == 0 || codexComposerRowPaintsLeftEdge(snapshot, y) ||
		!snapshotRowPainted(snapshot, y) || snapshotRowPainted(snapshot, y-1) {
		return false
	}
	for below := y + 1; below < snapshot.Height; below++ {
		if snapshotRowPainted(snapshot, below) {
			return false
		}
	}
	return true
}

// codexComposerRowPaintsLeftEdge reports whether row y paints anything in the
// columns Codex reserves for the composer prompt. Such a row is the block's
// left edge: either the composer's own prompt row or unrelated surface.
func codexComposerRowPaintsLeftEdge(snapshot terminalSnapshot, y int) bool {
	return rowPaintedBetween(snapshot, y, 0, codexComposerTextColumn)
}

// snapshotRowPainted reports whether row y paints any non-blank cell.
func snapshotRowPainted(snapshot terminalSnapshot, y int) bool {
	return rowPaintedBetween(snapshot, y, 0, snapshot.Width)
}

// rowPaintedBetween reports whether row y paints a non-blank cell in [x0, x1).
func rowPaintedBetween(snapshot terminalSnapshot, y, x0, x1 int) bool {
	if x1 > snapshot.Width {
		x1 = snapshot.Width
	}
	for x := x0; x < x1; x++ {
		if cell := snapshot.CellAt(x, y); cell != nil && strings.TrimSpace(cell.Content) != "" {
			return true
		}
	}
	return false
}

// museComposerMaxRows bounds how tall a Muse composer box may be.
const museComposerMaxRows = 20

// museComposerActive reports whether the cursor rests inside Muse's live
// composer: a non-faint prompt glyph at column 0 on the first row inside a pair
// of faint rule rows, with the cursor within that box. Anchoring on the
// enclosing rules rather than on rules directly above and below the prompt is
// what lets the composer grow past one line.
func museComposerActive(snapshot terminalSnapshot) bool {
	if !snapshot.CursorVisible || !snapshotCoordinatesValid(snapshot) || snapshot.Cursor.X < 2 {
		return false
	}

	// The prompt sits at or above the cursor, except when the cursor rests on
	// the box's own top rule.
	for promptY := snapshot.Cursor.Y + 1; promptY >= 0 && snapshot.Cursor.Y-promptY < museComposerMaxRows; promptY-- {
		if promptY >= snapshot.Height {
			continue
		}
		prompt := snapshot.CellAt(0, promptY)
		if prompt == nil || prompt.Content != "⟩" || prompt.Style.Attrs&uv.AttrFaint != 0 {
			continue
		}
		if !faintRuleAt(snapshot, 0, promptY-1) {
			continue
		}
		if bottom, ok := museComposerBottomRule(snapshot, promptY); ok && bottom >= snapshot.Cursor.Y {
			return true
		}
	}
	return false
}

// museComposerBottomRule finds the first row below the prompt that paints
// column 0 and reports whether it is the box's closing faint rule.
func museComposerBottomRule(snapshot terminalSnapshot, promptY int) (int, bool) {
	for y := promptY + 1; y < snapshot.Height && y-promptY <= museComposerMaxRows; y++ {
		cell := snapshot.CellAt(0, y)
		if cell == nil || strings.TrimSpace(cell.Content) == "" {
			continue
		}
		return y, faintRuleAt(snapshot, 0, y)
	}
	return 0, false
}

func agyComposerActive(snapshot terminalSnapshot) bool {
	if !snapshot.CursorVisible || !snapshotCoordinatesValid(snapshot) {
		return false
	}

	const (
		// agyPromptColor is the bright blue Agy paints its composer prompt
		// with, so an unstyled ">" never qualifies. This is a necessary
		// condition, NOT a picker discriminator: agy/1.1.15/menu.raw captures
		// Agy painting a slash-menu selection marker in this same bright blue.
		// Tolerable because Agy inserts a newline on LF there rather than
		// selecting; a real permission-picker capture is still outstanding.
		agyPromptColor  = xansi.BrightBlue
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
			if cell := snapshot.CellAt(promptX, y); cell != nil && cell.Content == ">" &&
				colorIsANSI(cell.Style.Fg, agyPromptColor) {
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

func faintRuleAt(snapshot terminalSnapshot, x, y int) bool {
	cell := snapshot.CellAt(x, y)
	return cell != nil && cell.Content == "─" && cell.Style.Attrs&uv.AttrFaint != 0
}

// colorIsANSI reports whether value is the given basic ANSI palette index.
func colorIsANSI(value color.Color, index xansi.BasicColor) bool {
	basic, ok := value.(xansi.BasicColor)
	return ok && basic == index
}
