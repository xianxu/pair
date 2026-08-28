package sessioninventory

import "strings"

// NormalizePairText is the canonical identity projection for operator-authored
// Pair text. It removes Pair's sticky comment framing and presentation-only
// whitespace while preserving case and meaningful internal spacing.
// pair:155-concept pure new M2
func NormalizePairText(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimRight(line, " \t")
		if strings.HasPrefix(strings.TrimLeft(line, " \t"), "===") {
			continue
		}
		out = append(out, line)
	}
	for len(out) > 0 && pairBlankLine(out[0]) {
		out = out[1:]
	}
	for len(out) > 0 && pairBlankLine(out[len(out)-1]) {
		out = out[:len(out)-1]
	}
	return strings.Join(out, "\n")
}

func pairBlankLine(line string) bool { return strings.Trim(line, " \t\r\v\f") == "" }
