package wrapcmd

import (
	uv "github.com/charmbracelet/ultraviolet"
	ansi "github.com/charmbracelet/x/ansi"
	"strings"
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
		if card, pending := codexOrientationStartupStatus(s); card && pending {
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
	if peerPlaceholder(agent, s, top, text) {
		return "", true
	}
	if strings.HasPrefix(strings.TrimSpace(text), "/") || strings.HasPrefix(strings.TrimSpace(text), "!") {
		return "", false
	}
	return text, true
}

// Only literal captured ghost signatures qualify. Different suggestions remain
// occupied until backed by a fixture; faint arbitrary text is not a placeholder.
func peerPlaceholder(agent string, s terminalSnapshot, row int, text string) bool {
	expected := map[string]string{"claude": `Try "refactor Makefile.local"`, "codex": "Ask Codex to do anything"}[agent]
	if text != expected || s.Cursor.X != 2 || s.Cursor.Y != row {
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
	if !strings.HasPrefix(line, "gpt-") || !strings.Contains(line, " · ") {
		return false
	}
	for y := row + 1; y < s.Height; y++ {
		if snapshotRowPainted(s, y) {
			return false
		}
	}
	return true
}

// peerComposerMatches accounts only for hard wrapping at the known composer
// width. It never collapses whitespace or accepts a summarized paste marker.
// Layouts that wrap differently remain pending until the delivery deadline.
func peerComposerMatches(agent string, s terminalSnapshot, expected string) bool {
	actual, ok := peerComposerText(agent, s)
	if !ok || expected == "" {
		return false
	}
	if actual == expected {
		return true
	}
	width := s.Width - 2
	if width < 1 {
		return false
	}
	var b strings.Builder
	column := 0
	for len(expected) > 0 {
		if expected[0] == '\n' {
			b.WriteByte('\n')
			column = 0
			expected = expected[1:]
			continue
		}
		if expected[0] == '\t' {
			return false
		} // tab expansion is harness-specific
		cluster, n := ansi.FirstGraphemeCluster(expected, ansi.GraphemeWidth)
		if cluster == "" || n > width {
			return false
		}
		if column+n > width {
			b.WriteByte('\n')
			column = 0
		}
		b.WriteString(cluster)
		column += n
		expected = expected[len(cluster):]
	}
	return actual == b.String()
}
