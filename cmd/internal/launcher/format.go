package launcher

import (
	"strconv"
	"strings"

	"github.com/xianxu/pair/cmd/internal/titlefmt"
)

// Display formatting for the launcher's history/picker rows + pane titles (#99
// M1, ported from bin/pair-shell). Pure string derivations.

const secondsPerDay = 86400

// FormatAge renders a "last touched" age from two epoch seconds (now, then):
// same day → "today", one day → "yesterday", else "<n>d ago".
func FormatAge(nowEpoch, thenEpoch int64) string {
	days := (nowEpoch - thenEpoch) / secondsPerDay
	switch days {
	case 0:
		return "today"
	case 1:
		return "yesterday"
	default:
		return strconv.FormatInt(days, 10) + "d ago"
	}
}

// AgeColor is the greyscale ANSI (xterm 256-color) escape for a historical row,
// brighter the more recently the tag was touched; oldest fades toward the dark bg
// without disappearing. fzf --ansi honors these.
func AgeColor(days int) string {
	switch {
	case days <= 0:
		return "\033[38;5;250m"
	case days <= 1:
		return "\033[38;5;245m"
	case days <= 3:
		return "\033[38;5;242m"
	case days <= 6:
		return "\033[38;5;240m"
	default:
		return "\033[38;5;238m"
	}
}

// TildeAbbrev abbreviates $HOME to ~ only on a real path boundary — exactly $HOME
// or $HOME/*, so a sibling like /Users/xianxu-other is never mangled to ~-other.
func TildeAbbrev(cwd, home string) string {
	if home == "" {
		return cwd // defensive extension (the shell assumes $HOME is always set)
	}
	if cwd == home {
		return "~"
	}
	if strings.HasPrefix(cwd, home+"/") {
		return "~" + cwd[len(home):]
	}
	return cwd
}

// PaneTitle is the "<agent> [<tilde-cwd>]" string exported as PAIR_PANE_TITLE.
func PaneTitle(agent, cwd, home string) string {
	return agent + " [" + TildeAbbrev(cwd, home) + "]"
}

// EmojiTitle applies the personal cmux display convention — 'brain' → 🧠,
// 'book' → 📗, 'pair' → ♋ in compound titles — uniformly on every rename.
func EmojiTitle(title string) string {
	return titlefmt.EmojiTitle(title)
}
