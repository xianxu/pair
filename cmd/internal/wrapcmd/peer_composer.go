package wrapcmd

import (
	"strings"

	uv "github.com/charmbracelet/ultraviolet"
)

type PeerComposerState uint8

const (
	PeerComposerUnknown PeerComposerState = iota
	PeerComposerOccupied
	PeerComposerEmpty
)

func peerComposerState(agent string, s terminalSnapshot) PeerComposerState {
	text, ok := peerComposerText(agent, s)
	if !ok {
		return PeerComposerUnknown
	}
	if text != "" || s.Cursor.X != 2 {
		return PeerComposerOccupied
	}
	return PeerComposerEmpty
}

// peerComposerText reads the complete qualified composer. Attachments, admitted
// input and overlays additionally gate delivery in the input owner. No model
// execution-idle observation is required. Unknown screens decline.
func peerComposerText(agent string, s terminalSnapshot) (string, bool) {
	var top, bottom int
	switch agent {
	case "claude":
		var ok bool
		top, bottom, ok = ruledBoxComposerBounds(s, claudeComposerSpec())
		if !ok || s.CellAt(0, top).Content != "❯" {
			return "", false
		}
	case "codex":
		if card, pending := codexOrientationStartupStatus(s); card && pending && !peerCodexModernStartup(s) {
			return "", false
		}
		if !codexComposerActive(s) {
			return "", false
		}
		top = s.Cursor.Y
		for top >= 0 && !codexComposerRowPaintsLeftEdge(s, top) {
			top--
		}
		if top < 0 {
			return "", false
		}
		bottom = top + 1
		for bottom < s.Height && !codexComposerRowPaintsLeftEdge(s, bottom) {
			if bottom > s.Cursor.Y && bottom > top+1 && !snapshotRowPainted(s, bottom-1) && peerCodexFooter(s, bottom) && rowPaintedBetween(s, top, 2, s.Width) {
				bottom--
				break
			}
			bottom++
		}
		if bottom < s.Height && codexComposerRowPaintsLeftEdge(s, bottom) {
			return "", false
		}
	default:
		return "", false
	}
	lines := make([]string, 0, bottom-top)
	for y := top; y < bottom; y++ {
		var b strings.Builder
		for x := 2; x < s.Width; x++ {
			c := s.CellAt(x, y)
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
	if peerPlaceholder(s, top, text) {
		return "", true
	}
	if strings.HasPrefix(strings.TrimSpace(text), "/") || strings.HasPrefix(strings.TrimSpace(text), "!") {
		return "", false
	}
	return text, true
}

// Recognized composers share a default suggestion convention: single-line
// ANSI faint text at the empty input origin, independent of wording or color.
// The wrapper separately retains human draft ownership even if text looks blank.
func peerPlaceholder(s terminalSnapshot, row int, text string) bool {
	if strings.TrimSpace(text) == "" || strings.ContainsAny(text, "\r\n") || s.Cursor.X != 2 || s.Cursor.Y != row {
		return false
	}
	for x := 2; x < s.Width; x++ {
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
func peerCodexFooter(s terminalSnapshot, row int) bool {
	var b strings.Builder
	for x := 2; x < s.Width; x++ {
		c := s.CellAt(x, row)
		if c == nil {
			return false
		}
		if c.Content == "" {
			b.WriteByte(' ')
		} else {
			b.WriteString(c.Content)
		}
	}
	line := strings.TrimSpace(b.String())
	if !strings.HasPrefix(strings.ToLower(line), "gpt-") || !strings.Contains(line, " · ") {
		return false
	}
	for y := row + 1; y < s.Height; y++ {
		if snapshotRowPainted(s, y) {
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
		line := strings.TrimSpace(orientationRowText(s, y))
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
		if peerCodexFooter(s, y) {
			return true
		}
	}
	return false
}
