package sessionsync

import "strings"

// readyMarkerIn reports whether text holds a complete operator request marker
// (pair#661): a 🤖 marker with no strike whose last section is a non-empty
// `[…]` operator turn — the review grammar's "ready" (nvim/review/markers.lua:
// 🤖<quoted>?(~strike~)?([user]|{agent})*, brackets nest, backslash escapes).
// Under presentation: propose, a record whose old text holds one answers that
// request, which the operator approved by writing it, so it applies directly.
func readyMarkerIn(text string) bool {
	const robot = "🤖"
	for at := 0; ; {
		i := strings.Index(text[at:], robot)
		if i < 0 {
			return false
		}
		cursor := at + i + len(robot)
		strike := false
		if cursor < len(text) {
			switch text[cursor] {
			case '<':
				if end := matchBracket(text, cursor, '<', '>'); end >= 0 {
					cursor = end + 1
				}
			case '~':
				if end := strings.IndexByte(text[cursor+1:], '~'); end >= 0 && !strings.Contains(text[cursor+1:cursor+1+end], "\n") {
					strike, cursor = true, cursor+1+end+1
				}
			}
		}
		lastUser, lastText := false, ""
		for cursor < len(text) && (text[cursor] == '[' || text[cursor] == '{') {
			open, close := text[cursor], byte(']')
			if open == '{' {
				close = '}'
			}
			end := matchBracket(text, cursor, open, close)
			if end < 0 {
				break
			}
			lastUser, lastText = open == '[', text[cursor+1:end]
			cursor = end + 1
		}
		if !strike && lastUser && strings.TrimSpace(lastText) != "" {
			return true
		}
		at = cursor
	}
}

// matchBracket is the index of the bracket closing the one at start, counting
// nesting and skipping backslash-escaped brackets, or -1.
func matchBracket(text string, start int, open, close byte) int {
	depth := 0
	for i := start; i < len(text); i++ {
		c := text[i]
		if c != open && c != close {
			continue
		}
		escapes := 0
		for j := i - 1; j >= start && text[j] == '\\'; j-- {
			escapes++
		}
		if escapes%2 == 1 {
			continue
		}
		if c == open {
			depth++
		} else if depth--; depth == 0 {
			return i
		}
	}
	return -1
}
