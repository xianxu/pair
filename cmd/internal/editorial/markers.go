package editorial

import "strings"

// Marker is one complete review marker, in byte coordinates of the scanned text.
// Reference is the explicit anchor's contents when HasReference is true.
type Marker struct {
	Range         Range
	Reference     Range
	HasReference  bool
	Ready         bool
	Pending       bool
	LastTurn      Range
	LastTurnAgent bool
}

// ScanMarkers reads the shared review grammar once. A strike is never ready;
// this retains the existing request-approval semantics of agent write paths.
func ScanMarkers(source string) []Marker { out, _ := scanMarkers(source); return out }

type markerScanStats struct{ Bytes, Robots, BracketLookups int }

func scanMarkers(source string) ([]Marker, markerScanStats) {
	stats := markerScanStats{}
	pairs := map[int]int{}
	var square, brace, angle []int
	for i := 0; i < len(source); i++ {
		stats.Bytes++
		if source[i] == '\\' {
			if i+1 < len(source) {
				i++
				stats.Bytes++
			}
			continue
		}
		var stack *[]int
		closing := false
		switch source[i] {
		case '[':
			stack = &square
		case ']':
			stack = &square
			closing = true
		case '{':
			stack = &brace
		case '}':
			stack = &brace
			closing = true
		case '<':
			stack = &angle
		case '>':
			stack = &angle
			closing = true
		default:
			continue
		}
		if !closing {
			*stack = append(*stack, i)
		} else if len(*stack) > 0 {
			last := len(*stack) - 1
			pairs[(*stack)[last]] = i
			*stack = (*stack)[:last]
		}
	}
	lookup := func(start int) int {
		stats.BracketLookups++
		if end, ok := pairs[start]; ok {
			return end
		}
		return -1
	}

	const robot = "🤖"
	var out []Marker
	for at := 0; at < len(source); {
		relative := strings.Index(source[at:], robot)
		if relative < 0 {
			break
		}
		stats.Robots++
		start := at + relative
		cursor := start + len(robot)
		m := Marker{Range: Range{Start: start}}
		strike := false
		if cursor < len(source) && source[cursor] == '<' {
			if end := lookup(cursor); end >= 0 {
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
			open := source[cursor]
			end := lookup(cursor)
			if end < 0 {
				break
			}
			m.LastTurn = Range{cursor + 1, end}
			m.LastTurnAgent = open == '{'
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
	return out, stats
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

// UnescapeMarkerText decodes only the delimiters escaped by the marker codec.
func UnescapeMarkerText(source string) string {
	var out strings.Builder
	out.Grow(len(source))
	for i := 0; i < len(source); i++ {
		if source[i] == '\\' && i+1 < len(source) && strings.ContainsRune("\\<>[]{}", rune(source[i+1])) {
			i++
		}
		out.WriteByte(source[i])
	}
	return out.String()
}
