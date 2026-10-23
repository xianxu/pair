package wrapcmd

// composerRecognition preserves activity independently of complete text proof.
// An active composer with no region keeps Return's existing newline mapping.
type composerRecognition struct {
	Active bool
	region composerRegion
}

type composerInputRecognizer func(terminalSnapshot) composerRecognition

type composerRegion struct {
	top, bottom, left int // bottom is exclusive
	agent             string
	valid             bool
}

func ruledComposerInput(agent string, s terminalSnapshot, spec ruledBoxComposerSpec) composerRecognition {
	top, bottom, active := ruledBoxComposerBounds(s, spec)
	result := composerRecognition{Active: active}
	if !active || s.Cursor.Y < top || s.Cursor.Y >= bottom {
		return result
	}
	if agent == "claude" && s.CellAt(0, top).Content != "❯" {
		return result
	}
	result.region = composerRegion{top: top, bottom: bottom, left: peerInputColumn(agent), agent: agent, valid: true}
	return result
}

func claudeComposerInput(s terminalSnapshot) composerRecognition {
	return ruledComposerInput("claude", s, claudeComposerSpec())
}
func qoderComposerInput(s terminalSnapshot) composerRecognition {
	return ruledComposerInput("qoder", s, qoderComposerSpec())
}
func museComposerInput(s terminalSnapshot) composerRecognition {
	return ruledComposerInput("muse", s, museComposerSpec())
}

func codexComposerInput(s terminalSnapshot) composerRecognition {
	top, footer, active := codexComposerBounds(s)
	result := composerRecognition{Active: active}
	if !active {
		return result
	}
	if card, pending := codexStartupStatus(s); card && pending && !peerCodexModernStartup(s) {
		return result
	}
	bottom, ok := codexComposerBottom(s, top, footer)
	if ok {
		result.region = composerRegion{top: top, bottom: bottom, left: codexComposerTextColumn, agent: "codex", valid: true}
	}
	return result
}

// codexComposerBottom shares the complete text extent with ordinary peer input.
func codexComposerBottom(s terminalSnapshot, top, footer int) (int, bool) {
	if footer >= 0 {
		return footer - 1, true
	}
	bottom := top + 1
	for bottom < s.Height && !codexComposerRowPaintsLeftEdge(s, bottom) {
		if bottom > s.Cursor.Y && bottom > top+1 && !snapshotRowPainted(s, bottom-1) && codexComposerFooter(s, bottom) && rowPaintedBetween(s, top, 2, s.Width) {
			bottom--
			break
		}
		bottom++
	}
	if bottom < s.Height && codexComposerRowPaintsLeftEdge(s, bottom) {
		return 0, false
	}
	return bottom, true
}

func grokComposerInput(s terminalSnapshot) composerRecognition {
	top, bottom, active := grokComposerBounds(s)
	return composerRecognition{Active: active, region: composerRegion{top: top, bottom: bottom, left: grokPromptCol + 2, agent: "grok", valid: active}}
}

func agyComposerInput(s terminalSnapshot) composerRecognition {
	x, top, bottom, active := agyComposerBounds(s)
	result := composerRecognition{Active: active}
	if !active || s.Cursor.X < x+2 {
		return result
	}
	// Activity tolerates multiple blue markers; text authority requires exactly
	// one prompt and a cursor in that prompt's input region.
	prompt := -1
	for y := top + 1; y < bottom; y++ {
		c := s.CellAt(x, y)
		if c != nil && c.Content == ">" && colorIsANSI(c.Style.Fg, agyPromptColor) {
			if prompt >= 0 {
				return result
			}
			prompt = y
		}
	}
	if prompt < 0 || s.Cursor.Y < prompt {
		return result
	}
	result.region = composerRegion{top: prompt, bottom: bottom, left: x + 2, agent: "agy", valid: true}
	return result
}
