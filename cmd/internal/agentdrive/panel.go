package agentdrive

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/tailmarkup"
)

// claudePanelFooters are the closing lines a Claude panel paints to declare
// that Esc dismisses it. PROVISIONAL (pair#691): shaped from the footer of
// Claude's own dialogs until the hub's /usage capture replaces the fixture;
// add a wording here only from a captured screen.
var claudePanelFooters = map[string]bool{
	"Esc to cancel": true,
}

// panelChoiceRow is any numbered row. A screen offering choices is a picker or
// a permission prompt, where Esc is a decision, not a dismissal.
var panelChoiceRow = regexp.MustCompile(`^[❯›↓]?\s*\d+\.\s`)

// DismissablePanel reports whether the current viewport is a panel whose own
// footer declares Esc as its dismissal: the last nonblank line is a known
// footer, and no line is a composer prompt or a numbered choice. lines must be
// the viewport only, never scrollback, so old text cannot authorize a key.
func DismissablePanel(agent string, lines []string) bool {
	if agent != "claude" || len(lines) > 200 {
		return false
	}
	footer := ""
	for _, line := range lines {
		if !utf8.ValidString(line) || len(line) > MaxResultTailBytes {
			return false
		}
		if _, ok := checkedMarkup(line); !ok {
			return false
		}
		plain := strings.TrimSpace(tailmarkup.Strip(line))
		if strings.HasPrefix(plain, "❯") || panelChoiceRow.MatchString(plain) {
			return false
		}
		if plain != "" {
			footer = plain
		}
	}
	return claudePanelFooters[footer]
}
