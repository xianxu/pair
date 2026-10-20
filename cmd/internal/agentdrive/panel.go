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
// last full-width rule, since earlier prompt rows stay visible above it; a
// panel taller than the screen has scrolled its rule away and fills the whole
// viewport. Within it the last nonblank row is a known footer, and no row is a
// composer prompt or a numbered choice. A composer's closing rule leaves only
// its status footer below, so a composer, idle or working, never qualifies. lines must be the viewport only, never
// scrollback, so old text cannot authorize a key. Like ReadModelScreen, more
// than 200 rows is refused rather than scanned.
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

// interruptHintRows bounds how far up AdvertisesInterrupt looks: Claude puts
// the hint in its footer, Codex in the status line above its composer.
const interruptHintRows = 8

// AdvertisesInterrupt reports whether the agent's own UI says it is working
// and interruptible: "esc to interrupt" in one of the bottom nonblank viewport
// rows, as Claude 2.1.304 and Codex 0.152.0 show while they work. lines must be
// the viewport, never scrollback (pair#717).
func AdvertisesInterrupt(lines []string) bool {
	seen := 0
	for i := len(lines) - 1; i >= 0 && seen < interruptHintRows; i-- {
		plain := strings.TrimSpace(tailmarkup.Strip(lines[i]))
		if plain == "" {
			continue
		}
		seen++
		if strings.Contains(strings.ToLower(plain), "esc to interrupt") {
			return true
		}
	}
	return false
}
