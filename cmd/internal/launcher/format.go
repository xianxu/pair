package launcher

import "strconv"

// Display formatting for the launcher's history/picker rows (#99 M1, ported from
// bin/pair-shell). Pure string derivations. Pane-title formatting used to live
// here too; #133 removed it — the startup title is now just the agent name, set
// inline at its one call site.

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
