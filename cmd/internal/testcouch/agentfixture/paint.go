package agentfixture

import (
	"embed"
	"fmt"
	"strings"
	"unicode"

	"github.com/xianxu/pair/cmd/internal/textwidth"
)

// Captures are exact pinned copies of wrapcmd/testdata/tty/{claude/2.1.237,
// codex/0.152.0}/composer.raw, including their original terminal negotiation.
// The deterministic repaint removes machine-specific suggestions/startup cards.
//
//go:embed testdata/*-composer.raw
var captures embed.FS

func Version(agent string) string {
	switch agent {
	case "claude":
		return "2.1.237 (Claude Code)"
	case "codex":
		return "codex-cli 0.152.0"
	}
	return ""
}
func CapturedComposer(agent string) ([]byte, error) {
	return captures.ReadFile("testdata/" + agent + "-composer.raw")
}

// Paint uses the profile shapes admitted by production composer_recognizers.go:
// Claude's matching ruled box/❯ and Codex's bold ›. The input starts at column2.
func Paint(s AgentState, cols, rows int) string {
	if cols < 20 {
		cols = 20
	}
	if rows < 10 {
		rows = 10
	}
	var b strings.Builder
	b.WriteString("\x1b[?2004h\x1b[?2026h\x1b[2J\x1b[H")
	if s.Phase == Picker {
		if s.Agent == "claude" {
			b.WriteString("\x1b]777;notify;Claude Code;Claude needs your permission\a")
		}
		b.WriteString("Fixture permission picker\r\nPress enter to confirm or esc to cancel\x1b[?25l\x1b[?2026l")
		return b.String()
	}
	line := 2
	if s.Phase == Working {
		b.WriteString("• Working (1s • esc to interrupt)\r\n")
		line = 3
	}
	rule := strings.Repeat("─", cols-1)
	if s.Agent == "claude" {
		fmt.Fprintf(&b, "\x1b[%d;1H\x1b[38;2;136;136;136m%s\x1b[39m", line, rule)
		line++
	}
	promptRow := line
	safe := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return -1
		}
		return r
	}, s.Draft)
	lines := strings.Split(safe, "\n")
	rendered := make([]string, 0, len(lines))
	for _, text := range lines {
		var segment strings.Builder
		count := 0
		for _, r := range text {
			if count >= cols-3 {
				rendered = append(rendered, segment.String())
				segment.Reset()
				count = 0
			}
			segment.WriteRune(r)
			count += textwidth.Width(string(r))
		}
		rendered = append(rendered, segment.String())
	}
	if len(rendered) > rows-line-3 {
		rendered = rendered[len(rendered)-(rows-line-3):]
	}
	for i, text := range rendered {
		fmt.Fprintf(&b, "\x1b[%d;1H", line+i)
		if i == 0 {
			if s.Agent == "codex" {
				b.WriteString("\x1b[1m›\x1b[22m ")
			} else {
				b.WriteString("❯ ")
			}
		} else {
			b.WriteString("  ")
		}
		b.WriteString(text)
	}
	bottom := line + len(rendered)
	if s.Agent == "claude" {
		fmt.Fprintf(&b, "\x1b[%d;1H\x1b[38;2;136;136;136m%s\x1b[39m\x1b[%d;1H⏵⏵ bypass permissions on (shift+tab to cycle)", bottom, rule, bottom+1)
	}
	// Codex's optional native footer is omitted. A synthetic footer at the
	// left edge looks like another input surface to the real wrapper; its
	// minimal qualified composer is the bold prompt with blank rows below.
	cursorRow := promptRow + len(rendered) - 1
	cursorCol := 2 + textwidth.Width(rendered[len(rendered)-1])
	fmt.Fprintf(&b, "\x1b[%d;%dH\x1b[?25h\x1b[?2026l", cursorRow, cursorCol+1)
	return b.String()
}
