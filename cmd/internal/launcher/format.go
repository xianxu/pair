package launcher

import "strings"

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
		return itoa64(days) + "d ago"
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
		return cwd
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
// 'book' → 📗, 'pair' → ♋ anywhere in the title — uniformly on every rename.
func EmojiTitle(title string) string {
	title = strings.ReplaceAll(title, "brain", "🧠")
	title = strings.ReplaceAll(title, "book", "📗")
	title = strings.ReplaceAll(title, "pair", "♋")
	return title
}

func itoa64(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
