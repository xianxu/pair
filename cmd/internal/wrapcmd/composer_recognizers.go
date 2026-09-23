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

// musePromptGlyphs is the ONE authority for what may sit at column 0 of a Muse
// composer prompt row. Both consumers read it — museComposerActive (the Return
// remap's gate) and orientationPromptOK (the orientation auto-submit gate) —
// because a glyph accepted by one and rejected by the other is a state where
// Return inserts a newline in a composer orientation refuses to submit into.
//
// The set is the chevron family Muse's own prompt belongs to. `⟩` is the
// captured glyph (see testdata/tty/muse/*); the other three are the shapes a UI
// refresh plausibly swaps it for, admitted because for Muse a false NEGATIVE is
// the expensive direction: plain Return then falls through as bare CR, which
// Muse reads as submit, so a half-written draft ships.
//
// Selection markers are deliberately NOT here. `!`, `●`, `▶` and `▸` were
// admitted speculatively and removed (#266 close BR-3): they are how TUIs mark
// a highlighted menu row, so accepting them invites the opposite failure — the
// gate calling a picker a composer — and `!` also contradicts orientation's own
// non-coding-mode guard, which rejects that glyph as row content.
var musePromptGlyphs = map[string]bool{"⟩": true, "›": true, "❯": true, ">": true}

// ruledBoxComposerSpec parameterises the composer shape Claude and Muse share:
// a prompt glyph at column 0 forming the first row inside a pair of rule rows.
// Anchoring on the *enclosing* rules rather than on rules directly above and
// below the prompt is what lets the composer grow past one line.
type ruledBoxComposerSpec struct {
	// promptOK qualifies the cell at promptCol of a candidate prompt row.
	promptOK func(uv.Cell) bool
	// ruleAt reports whether row y is one of the box's rule rows.
	ruleAt func(terminalSnapshot, int) bool
	// rulesMatch, when set, additionally requires the box's two rules to agree
	// with each other, so unrelated chrome cannot pair up into a box.
	rulesMatch func(top, bottom uv.Cell) bool
	// maxRows bounds how far the prompt may sit above the cursor. Zero means
	// unbounded, which is safe for a spec that anchors on an immediate top rule
	// and takes the first painted column-0 row below as the closing rule: the
	// box cannot absorb distant chrome regardless of height.
	maxRows int
	// promptCol is the column the harness paints its prompt glyph in. Zero for
	// Claude and Muse; Qoder indents by one column (see qoderPromptCol).
	promptCol int
	// minCursorX is the first column the harness leaves for composer text.
	minCursorX int
	// allowHiddenCursor is true only for a harness that hides the system cursor
	// behind one it paints itself (Qoder), so hidden-cursor snapshots may still
	// qualify. The zero value requires the visible cursor — the prior behaviour
	// of every spec — so a spec that forgets the field fails closed.
	allowHiddenCursor bool
}

// ruledBoxComposerActive reports whether the cursor rests inside a ruled box.
// The prompt sits at or above the cursor — except when the cursor rests on the
// box's own top rule — and the closing rule must sit at or below the cursor.
// This is the only function that owns the ruled-box scan: a harness with a
// ruled box registers a spec, however its prompt column and glyphs differ.
func ruledBoxComposerActive(snapshot terminalSnapshot, spec ruledBoxComposerSpec) bool {
	if !snapshotCoordinatesValid(snapshot) ||
		snapshot.Cursor.X < spec.minCursorX ||
		(!spec.allowHiddenCursor && !snapshot.CursorVisible) {
		return false
	}

	for promptY := snapshot.Cursor.Y + 1; promptY >= 0; promptY-- {
		if spec.maxRows > 0 && snapshot.Cursor.Y-promptY >= spec.maxRows {
			break
		}
		if promptY >= snapshot.Height || promptY-1 < 0 {
			continue
		}
		prompt := snapshot.CellAt(spec.promptCol, promptY)
		if prompt == nil || !spec.promptOK(*prompt) || !spec.ruleAt(snapshot, promptY-1) {
			continue
		}
		bottom, ok := ruledBoxBottomRule(snapshot, spec, promptY)
		if !ok || bottom < snapshot.Cursor.Y {
			continue
		}
		if spec.rulesMatch != nil {
			top, closing := snapshot.CellAt(0, promptY-1), snapshot.CellAt(0, bottom)
			if top == nil || closing == nil || !spec.rulesMatch(*top, *closing) {
				continue
			}
		}
		return true
	}
	return false
}

// ruledBoxBottomRule finds the first row below the prompt that paints column 0
// and reports whether it is the box's closing rule.
func ruledBoxBottomRule(snapshot terminalSnapshot, spec ruledBoxComposerSpec, promptY int) (int, bool) {
	for y := promptY + 1; y < snapshot.Height; y++ {
		if spec.maxRows > 0 && y-promptY > spec.maxRows {
			break
		}
		cell := snapshot.CellAt(0, y)
		if cell == nil || strings.TrimSpace(cell.Content) == "" {
			continue
		}
		return y, spec.ruleAt(snapshot, y)
	}
	return 0, false
}

// museComposerActive reports whether the cursor rests inside Muse's live
// composer: a prompt glyph at column 0 on the first row inside a pair
// of faint rule rows, with the cursor within that box. Relaxed from the
// original strict "⟩" + faint-only check so a Muse UI refresh that changes
// the prompt glyph or rule faintness does not silently break the Return
// remap; the admitted glyphs are musePromptGlyphs. The box shape (prompt row enclosed by two "─" rules) remains the
// discriminator; faint agreement is checked via rulesMatch rather than
// requiring every rule to be faint.
func museComposerActive(snapshot terminalSnapshot) bool {
	return ruledBoxComposerActive(snapshot, ruledBoxComposerSpec{
		promptOK: func(c uv.Cell) bool {
			if c.Style.Attrs&uv.AttrFaint != 0 {
				return false
			}
			return musePromptGlyphs[c.Content]
		},
		ruleAt: func(s terminalSnapshot, y int) bool {
			cell := s.CellAt(0, y)
			return cell != nil && cell.Content == "─"
		},
		rulesMatch: func(top, bottom uv.Cell) bool {
			return (top.Style.Attrs&uv.AttrFaint != 0) == (bottom.Style.Attrs&uv.AttrFaint != 0)
		},
		maxRows: museComposerMaxRows,
		// Column 0 is the prompt and column 1 its trailing space.
		minCursorX: 2,
	})
}

// claudeComposerRule is the glyph Claude draws its composer rules with.
const claudeComposerRule = "─"

// claudeComposerMaxRows is deliberately unbounded. A height ceiling here would
// make plain Return submit any draft taller than it — the lost-draft failure
// this issue's blast radius names as the expensive direction, on Pair's default
// agent. The bound also buys nothing: the top rule must sit immediately above
// the prompt and the closing rule is the first painted column-0 row below it,
// so no distant chrome can pair into a box however tall the draft grows.
const claudeComposerMaxRows = 0

// claudeComposerActive reports whether the cursor rests inside Claude's live
// composer: a single glyph at column 0 between two matching rule rows.
//
// Deliberately glyph- and colour-agnostic. Claude repaints both the prompt
// glyph and the rule colour per input mode — bash mode is "!" with pink rules
// where the default is "❯" with grey — so an allowlist would decline in any
// mode it had not enumerated. Claude is Pair's default agent and a decline
// makes the next Return submit a half-written draft, so false negatives are the
// expensive direction here. Requiring the box's two rules to share a foreground
// is what keeps unrelated chrome from pairing into a composer.
func claudeComposerActive(snapshot terminalSnapshot) bool {
	return ruledBoxComposerActive(snapshot, ruledBoxComposerSpec{
		promptOK: func(c uv.Cell) bool {
			return strings.TrimSpace(c.Content) != "" && c.Content != claudeComposerRule
		},
		ruleAt: func(s terminalSnapshot, y int) bool {
			cell := s.CellAt(0, y)
			return cell != nil && cell.Content == claudeComposerRule
		},
		rulesMatch: func(top, bottom uv.Cell) bool {
			return sameForeground(top.Style.Fg, bottom.Style.Fg)
		},
		maxRows: claudeComposerMaxRows,
		// Column 0 is the prompt and column 1 its trailing space.
		minCursorX: 2,
	})
}

// sameForeground reports whether two cells carry the same foreground, treating
// two unset foregrounds as equal.
func sameForeground(a, b color.Color) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, ba := b.RGBA()
	return ar == br && ag == bg && ab == bb && aa == ba
}

// qoderPromptCol is the ONE authority for the column Qoder indents its
// composer prompt to. Qoder reserves column 0 for a leading space (and paints
// its rules across the full width from column 0), so its prompt sits one
// column right of Claude's and Muse's.
const qoderPromptCol = 1

// qoderPromptGlyphs is the ONE authority for what may sit at qoderPromptCol
// of a Qoder composer prompt row. Both consumers read it — qoderComposerActive
// (the Return remap's gate) and orientationPromptOK (the orientation
// auto-submit gate) — because a glyph accepted by one and rejected by the
// other is a state where Return inserts a newline in a composer orientation
// refuses to submit into. `>` is the captured default-mode glyph and `*` the
// captured yolo-mode glyph (see testdata/tty/qoder/1.1.60/*). Menu selection
// markers are deliberately NOT here: Qoder marks a highlighted picker row with
// `❯`, which must keep reading as a picker rather than a composer.
var qoderPromptGlyphs = map[string]bool{">": true, "*": true}

// qoderComposerActive reports whether the cursor rests inside Qoder's live
// composer: a prompt glyph at qoderPromptCol between two rule rows. Qoder
// reuses Claude's ruled-box shape but indents its prompt by one column and
// paints its top and bottom rules in different greys, so colour agreement is
// not required — the enclosing shape plus the prompt glyph discriminates
// against Qoder's permission pickers (whose markers are not
// qoderPromptGlyphs) and the startup screen's separator lines (which have no
// prompt glyph at qoderPromptCol).
//
// Qoder hides the system cursor and renders its own visual cursor, which is
// why its spec — and only its spec — sets allowHiddenCursor. Heights are
// unbounded like Claude's: the box cannot absorb distant chrome because its
// closing rule is the first painted column-0 row below the prompt.
func qoderComposerActive(snapshot terminalSnapshot) bool {
	return ruledBoxComposerActive(snapshot, ruledBoxComposerSpec{
		promptOK: func(c uv.Cell) bool {
			return qoderPromptGlyphs[c.Content]
		},
		ruleAt: func(s terminalSnapshot, y int) bool {
			cell := s.CellAt(0, y)
			return cell != nil && cell.Content == claudeComposerRule
		},
		maxRows:           0,
		promptCol:         qoderPromptCol,
		minCursorX:        2,
		allowHiddenCursor: true,
	})
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
		// Tolerable IF Agy inserts a newline on LF there rather than
		// selecting — see ttyFixtureReactionGaps, which records that as
		// undriven; a real permission-picker capture is still outstanding.
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
