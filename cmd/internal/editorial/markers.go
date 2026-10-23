package editorial

import "strings"

// Marker is one complete review marker, in byte coordinates of the scanned text.
// Reference is the explicit anchor's contents when HasReference is true.
type Marker struct {
	Range        Range
	Reference    Range
	HasReference bool
	Ready        bool
	Pending      bool
}

// ScanMarkers reads the shared review grammar once. A strike is never ready;
// this retains the existing request-approval semantics of agent write paths.
func ScanMarkers(source string) []Marker {
	const robot = "🤖"
	var out []Marker
	for at := 0; at < len(source); {
		relative := strings.Index(source[at:], robot)
		if relative < 0 {
			break
		}
		start := at + relative
		cursor := start + len(robot)
		m := Marker{Range: Range{Start: start}}
		strike := false
		if cursor < len(source) && source[cursor] == '<' {
			if end := MatchBracket(source, cursor, '<', '>'); end >= 0 {
				m.Reference = Range{cursor + 1, end}
				m.HasReference = true
				cursor = end + 1
			}
		}
		if cursor < len(source) && source[cursor] == '~' {
			if end := strings.IndexByte(source[cursor+1:], '~'); end >= 0 && !strings.Contains(source[cursor+1:cursor+1+end], "\n") {
				strike = true
				m.Reference = Range{cursor + 1, cursor + 1 + end}
				m.HasReference = true
				cursor += end + 2
			}
		}
		lastUser, lastText, hasTurn := false, "", false
		for cursor < len(source) && (source[cursor] == '[' || source[cursor] == '{') {
			open, close := source[cursor], byte(']')
			if open == '{' {
				close = '}'
			}
			end := MatchBracket(source, cursor, open, close)
			if end < 0 {
				break
			}
			lastUser, lastText, hasTurn = open == '[', source[cursor+1:end], true
			cursor = end + 1
		}
		m.Range.End = cursor
		m.Ready = !strike && lastUser && strings.TrimSpace(lastText) != ""
		m.Pending = !m.Ready && (strike || m.HasReference || hasTurn)
		if m.Ready || m.Pending {
			out = append(out, m)
		}
		at = cursor
	}
	return out
}

// ReadyMarkerSpan locates the first complete ready request marker.
func ReadyMarkerSpan(source string) (start, end int, ok bool) {
	for _, m := range ScanMarkers(source) {
		if m.Ready {
			return m.Range.Start, m.Range.End, true
		}
	}
	return 0, 0, false
}

// MatchBracket finds a matching nested bracket, ignoring backslash escapes.
func MatchBracket(source string, start int, open, close byte) int {
	depth := 0
	for i := start; i < len(source); i++ {
		if source[i] == '\\' {
			i++
			continue
		}
		switch source[i] {
		case open:
			depth++
		case close:
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}
