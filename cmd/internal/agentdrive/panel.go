package agentdrive

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/tailmarkup"
)

// claudePanelFooters are the closing lines a Claude panel paints to declare
// that Esc dismisses it, from captured screens only
// (wrapcmd/testdata/drive/claude/2.1.304/usage-panel.screen).
var claudePanelFooters = map[string]bool{
	"Esc to cancel": true,
}

// panelChoiceRow is any numbered row. A screen offering choices is a picker or
// a permission prompt, where Esc is a decision, not a dismissal.
var panelChoiceRow = regexp.MustCompile(`^[❯›↓]?\s*\d+\.\s`)

// panelRule is the full-width rule that opens a panel and bounds a composer.
var panelRule = regexp.MustCompile(`^─{20,}$`)

// DismissablePanel reports whether the current viewport ends in a panel whose
// own footer declares Esc as its dismissal. The panel is the rows below the
// last full-width rule: earlier prompt rows stay visible above it. Within it
// the last nonblank row is a known footer, and no row is a composer prompt or
// a numbered choice. A composer's closing rule leaves only its status footer
// below, so a composer never qualifies. lines must be the viewport only, never
// scrollback, so old text cannot authorize a key.
func DismissablePanel(agent string, lines []string) bool {
	if agent != "claude" || len(lines) > 200 {
		return false
	}
	panel := -1
	plain := make([]string, len(lines))
	for i, line := range lines {
		if !utf8.ValidString(line) || len(line) > MaxResultTailBytes {
			return false
		}
		if _, ok := checkedMarkup(line); !ok {
			return false
		}
		plain[i] = strings.TrimSpace(tailmarkup.Strip(line))
		if panelRule.MatchString(plain[i]) {
			panel = i
		}
	}
	if panel < 0 {
		return false
	}
	footer := ""
	for _, line := range plain[panel+1:] {
		if strings.HasPrefix(line, "❯") || panelChoiceRow.MatchString(line) {
			return false
		}
		if line != "" {
			footer = line
		}
	}
	return claudePanelFooters[footer]
}
