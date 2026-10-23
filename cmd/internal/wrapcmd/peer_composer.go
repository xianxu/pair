package wrapcmd

import (
	"strconv"
	"strings"

	uv "github.com/charmbracelet/ultraviolet"
)

type PeerComposerState uint8

const (
	PeerComposerUnknown PeerComposerState = iota
	PeerComposerOccupied
	PeerComposerEmpty
)

// backgroundWorkState is display evidence, independent of composer admission.
func backgroundWorkState(agent string, s terminalSnapshot) *bool {
	if agent != "claude" || !s.Complete {
		return nil
	}
	_, bottom, ok := ruledBoxComposerBounds(s, claudeComposerSpec())
	if !ok {
		return nil
	}
	for y := bottom + 1; y < s.Height; y++ {
		line := strings.TrimSpace(terminalRowText(s, y))
		if !strings.HasPrefix(line, "⏵⏵ ") {
			continue
		}
		fields := strings.Split(line, " · ")
		const permission = "⏵⏵ bypass permissions on"
		cycle := fields[0] == permission+" (shift+tab to cycle)"
		if fields[0] != permission && !cycle {
			return nil
		}
		count, seen, manage, pr := 0, false, false, false
		for i, field := range fields[1:] {
			// The captured PR link is the first optional field, never an
			// arbitrary decoration accepted elsewhere in the footer.
			if number, ok := strings.CutPrefix(field, "PR #"); ok {
				n, err := strconv.Atoi(number)
				if i != 0 || len(number) > 9 || err != nil || n <= 0 || strconv.Itoa(n) != number {
					return nil
				}
				pr = true
				continue
			}
			switch field {
			case "esc to interrupt", "← for agents":
				continue
			case "↓ to manage":
				if manage {
					return nil
				}
				manage = true
				continue
			}
			parts := strings.Fields(field)
			if seen || len(parts) != 2 || len(parts[0]) > 9 {
				return nil
			}
			n, err := strconv.Atoi(parts[0])
			if err != nil || n < 0 || strconv.Itoa(n) != parts[0] ||
				(n == 1 && parts[1] != "shell") || (n != 1 && parts[1] != "shells") {
				return nil
			}
			count, seen = n, true
		}
		// Only the complete captured no-shell chrome proves absence. A
		// missing/partial counter or an unfamiliar field remains unknown.
		if (seen && !manage) || (!seen && (!cycle || manage || pr)) {
			return nil
		}
		active := count > 0
		return &active
	}
	return nil
}

func peerComposerState(agent string, s terminalSnapshot) PeerComposerState {
	text, ok := peerComposerText(agent, s)
	if !ok {
		return PeerComposerUnknown
	}
	if text != "" || s.Cursor.X != peerInputColumn(agent) {
		return PeerComposerOccupied
	}
	return PeerComposerEmpty
}

// peerComposerWaitDetail explains missing evidence without retaining screen or
// draft contents. It adds only constant-time checks to an unknown observation.
func peerComposerWaitDetail(agent string, s terminalSnapshot) string {
	reason := "composer geometry or input is unrecognized"
	switch {
	case !s.Complete:
		reason = "terminal frame is incomplete"
	case !s.CursorVisible && agent != "qoder":
		reason = "cursor is hidden"
	}
	return "waiting for recognized composer: " + reason
}

func peerInputColumn(agent string) int {
	if agent == "qoder" {
		return qoderPromptCol + 2
	}
	return 2
}

// peerComposerText reads the complete qualified composer. Attachments, admitted
// input and overlays additionally gate delivery in the input owner. No model
// execution-idle observation is required. Unknown screens decline.
func peerComposerText(agent string, s terminalSnapshot) (string, bool) {
	text, ok := qualifiedComposerText(agent, s)
	if !ok || strings.HasPrefix(strings.TrimSpace(text), "/") || strings.HasPrefix(strings.TrimSpace(text), "!") {
		return "", false
	}
	return text, true
}

// qualifiedComposerText recognizes the composer geometry without granting
// ordinary peer delivery permission to overwrite command or shell drafts.
func qualifiedComposerText(agent string, s terminalSnapshot) (string, bool) {
	var top, bottom int
	switch agent {
	case "claude":
		var ok bool
		top, bottom, ok = ruledBoxComposerBounds(s, claudeComposerSpec())
		if !ok || s.CellAt(0, top).Content != "❯" {
			return "", false
		}
	case "qoder":
		var ok bool
		top, bottom, ok = ruledBoxComposerBounds(s, qoderComposerSpec())
		if !ok {
			return "", false
		}
	case "codex":
		if card, pending := codexStartupStatus(s); card && pending && !peerCodexModernStartup(s) {
			return "", false
		}
		var footer int
		var ok bool
		top, footer, ok = codexComposerBounds(s)
		if !ok {
			return "", false
		}
		var extentOK bool
		bottom, extentOK = codexComposerBottom(s, top, footer)
		if !extentOK {
			return "", false
		}
	default:
		return "", false
	}
	return projectComposerRegion(s, composerRegion{top: top, bottom: bottom, left: peerInputColumn(agent), agent: agent, valid: true})
}

// projectComposerRegion reads the full located input, including blank cursor
// lines and text after the cursor. It never trims user leading whitespace.
func projectComposerRegion(s terminalSnapshot, region composerRegion) (string, bool) {
	return projectComposerCells(s, region, s.CellAt)
}

// projectComposerCells projects immutable cells through one bounded row pass.
func projectComposerCells(s terminalSnapshot, region composerRegion, cellAt func(int, int) *uv.Cell) (string, bool) {
	if !region.valid || region.top < 0 || region.bottom > s.Height || region.bottom <= region.top || region.left < 0 || region.left >= s.Width {
		return "", false
	}
	top, bottom, agent := region.top, region.bottom, region.agent
	lines := make([]string, 0, bottom-top)
	for y := top; y < bottom; y++ {
		var b strings.Builder
		for x := region.left; x < s.Width; x++ {
			c := cellAt(x, y)
			if c == nil {
				return "", false
			}
			if c.Content == "" {
				b.WriteByte(' ')
				continue
			}
			b.WriteString(c.Content)
			if c.Width > 1 {
				x += c.Width - 1
			}
		}
		lines = append(lines, strings.TrimRight(b.String(), " "))
	}
	for len(lines) > 1 && lines[len(lines)-1] == "" && top+len(lines)-1 > s.Cursor.Y {
		lines = lines[:len(lines)-1]
	}
	text := strings.Join(lines, "\n")
	if (agent == "qoder" && peerQoderPlaceholder(s, top, text)) || (agent != "qoder" && composerPlaceholder(s, top, region.left, text)) {
		return "", true
	}
	return text, true
}

// Recognized composers share a default suggestion convention: single-line
// ANSI faint text at the empty input origin, independent of wording or color.
// The wrapper separately retains human draft ownership even if text looks blank.
func peerPlaceholder(s terminalSnapshot, row int, text string) bool {
	return composerPlaceholder(s, row, 2, text)
}

func composerPlaceholder(s terminalSnapshot, row, left int, text string) bool {
	if strings.TrimSpace(text) == "" || strings.ContainsAny(text, "\r\n") || s.Cursor.X != left || s.Cursor.Y != row {
		return false
	}
	for x := left; x < s.Width; x++ {
		c := s.CellAt(x, row)
		if c == nil {
			return false
		}
		if strings.TrimSpace(c.Content) != "" && c.Style.Attrs&uv.AttrFaint == 0 {
			return false
		}
	}
	return true
}

// Newer Codex paints a bare startup title and working path instead of the old
// boxed model/directory fields. A resolved model footer remains mandatory;
// merely seeing a prompt while startup is loading does not authorize input.
func peerCodexModernStartup(s terminalSnapshot) bool {
	title, path := false, false
	for y := 0; y < s.Cursor.Y; y++ {
		line := strings.TrimSpace(terminalRowText(s, y))
		if strings.HasPrefix(line, ">_ OpenAI Codex (v") {
			title = true
			continue
		}
		if title && line != "" {
			path = strings.HasPrefix(line, "/") || strings.HasPrefix(line, "~/")
			break
		}
	}
	if !title || !path {
		return false
	}
	for y := s.Cursor.Y + 1; y < s.Height; y++ {
		if codexComposerFooter(s, y) {
			return true
		}
	}
	return false
}

// Qoder's captured placeholder is plain grey, preceded by a painted cursor
// and a space. Wording alone cannot authorize overwriting an operator draft.
func peerQoderPlaceholder(s terminalSnapshot, row int, text string) bool {
	if s.Cursor.X != peerInputColumn("qoder") || s.Cursor.Y != row || text != "  Type your message or @path/to/file" {
		return false
	}
	cursor := s.CellAt(s.Cursor.X, row)
	if cursor == nil || cursor.Content != " " || cursor.Style.Bg == nil {
		return false
	}
	for x := s.Cursor.X + 1; x < s.Width; x++ {
		c := s.CellAt(x, row)
		if c == nil {
			return false
		}
		if strings.TrimSpace(c.Content) == "" {
			continue
		}
		if !sameForeground(c.Style.Fg, cursor.Style.Bg) {
			return false
		}
	}
	return true
}
