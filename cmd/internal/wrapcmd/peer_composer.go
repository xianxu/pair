package wrapcmd

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"

	uv "github.com/charmbracelet/ultraviolet"
	ansi "github.com/charmbracelet/x/ansi"
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

// peerComposerMatches projects the known composer width without collapsing
// arbitrary whitespace. Captured harness word wrapping additionally permits a
// single separating space to become a line break. Ambiguous whitespace and other
// layouts remain unsupported. Claude's collapsed paste marker is accepted only
// in its strict form (peerClaudeCollapsedPaste).
func peerComposerMatches(agent string, s terminalSnapshot, expected string) bool {
	actual, ok := peerComposerText(agent, s)
	if !ok || expected == "" {
		return false
	}
	if actual == expected {
		return true
	}
	if agent == "claude" && peerClaudeCollapsedPaste(s, actual, expected) {
		return true
	}
	width := s.Width - 2
	if width < 1 {
		return false
	}
	wordWidth := width
	if agent == "claude" {
		wordWidth = s.Width - 4
	}
	if wordWidth > 0 && peerWordwrapSafe(expected) {
		switch agent {
		case "codex":
			if actual == ansi.Wordwrap(expected, wordWidth, "") {
				return true
			}
		case "claude":
			// Claude Code breaks only at spaces; ansi.Wordwrap also breaks
			// after hyphens, which split `ariadne-robustness-1.md` where
			// Claude moved it whole (pair#418, captured on 2.1.295/2.1.296).
			if projected, ok := peerSpaceWordwrap(expected, wordWidth); ok && actual == projected {
				return true
			}
		}
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

// peerSpaceWordwrap projects Claude Code's composer wrapping, which is
// wrap-ansi's hard mode: break only at single spaces; a word wider than the
// line is hard-broken, starting on the current line unless starting on the
// next one needs fewer breaks (pair#418, captured on 2.1.296). The bool is false
// only for a width that cannot hold one grapheme.
func peerSpaceWordwrap(text string, width int) (string, bool) {
	var out []string
	for _, line := range strings.Split(text, "\n") {
		row, column := "", 0
		// The composer trims a row's trailing space, so a break drops it.
		breakRow := func() {
			out = append(out, strings.TrimSuffix(row, " "))
			row, column = "", 0
		}
		for j, word := range strings.Split(line, " ") {
			n := ansi.StringWidth(word)
			if j > 0 && column > 0 {
				if column >= width {
					breakRow()
				} else {
					row += " "
					column++
				}
			}
			if n > width {
				thisLine := 1 + (n-(width-column)-1)/width
				nextLine := (n - 1) / width
				if nextLine < thisLine && column > 0 {
					breakRow()
				}
				for rest := word; rest != ""; {
					cluster, w := ansi.FirstGraphemeCluster(rest, ansi.GraphemeWidth)
					if w > width {
						return "", false
					}
					if column+w > width {
						breakRow()
					}
					row += cluster
					column += w
					rest = rest[len(cluster):]
				}
				continue
			}
			if column > 0 && column+n > width {
				breakRow()
			}
			row += word
			column += n
		}
		out = append(out, row)
	}
	return strings.Join(out, "\n"), true
}

// peerClaudeCollapsedMarker is Claude Code's summary of a long or multi-line
// paste, the composer's whole content in place of the text.
var peerClaudeCollapsedMarker = regexp.MustCompile(`^\[Pasted text #[1-9][0-9]* \+([1-9][0-9]*) lines\]$`)

// peerClaudeCollapsedPaste accepts Claude's collapsed marker as evidence that
// our paste rendered (pair#418, operator decision). It cannot check the body,
// so it demands everything else: the marker is the composer's only content,
// the cursor sits right after it, and its line count is the envelope's. The
// delivery separately required an empty composer before pasting and a render
// after it, so the marker cannot be a human draft's.
func peerClaudeCollapsedPaste(s terminalSnapshot, actual, expected string) bool {
	m := peerClaudeCollapsedMarker.FindStringSubmatch(actual)
	if m == nil || m[1] != strconv.Itoa(strings.Count(expected, "\n")) {
		return false
	}
	prompt := s.CellAt(0, s.Cursor.Y)
	return s.Cursor.X == 2+ansi.StringWidth(actual) && prompt != nil && prompt.Content == "❯"
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

// Word wrapping hides boundary spaces. Only a single ordinary inter-word space
// is eligible; indentation, trailing spaces, tabs and repeated whitespace must
// instead match literally, so projecting them cannot erase semantic content.
func peerWordwrapSafe(text string) bool {
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) != line || strings.Contains(line, "  ") {
			return false
		}
		for _, r := range line {
			if unicode.IsSpace(r) && r != ' ' {
				return false
			}
		}
	}
	return true
}
