package agentdrive

import (
	"strings"
	"unicode/utf8"
)

// SlashCommand reports whether text contains one slash command line.
func SlashCommand(text string) bool {
	if len(text) < 2 || text[0] != '/' {
		return false
	}

	i := 1
	if !asciiLetter(text[i]) {
		return false
	}
	i++
	for i < len(text) && (asciiLetter(text[i]) || asciiDigit(text[i]) || text[i] == '_' || text[i] == '-') {
		i++
	}
	if i == len(text) {
		return true
	}
	if text[i] != ' ' && text[i] != '\t' {
		return false
	}
	i++ // The grammar permits one space or tab as the argument separator.

	for i < len(text) {
		r, size := utf8.DecodeRuneInString(text[i:])
		if r == utf8.RuneError && size == 1 || r == '\r' || r == '\n' || r < 0x20 || r >= 0x7f && r <= 0x9f {
			return false
		}
		i += size
	}
	return true
}

func asciiLetter(b byte) bool {
	return b >= 'A' && b <= 'Z' || b >= 'a' && b <= 'z'
}

func asciiDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

// commandExpectations predicts only append-only text from an admitted empty
// composer, with optional clear-line. A proved command boundary can start a new
// clear-line/text group; unsupported editing invalidates prediction. It scans
// the bounded request once; text
// is joined once per Enter, never by rescanning the preceding command groups.
// Predictions are authored targets, never evidence of what the agent received.
func commandExpectations(steps KeySequence) []string {
	targets := make([]string, len(steps))
	prediction := commandExpectation{known: true}
	for i, step := range steps {
		targets[i] = prediction.advance(step)
	}
	return targets
}

// commandExpectation owns only the current append-only span. Each call consumes
// one key; completed spans are discarded so later groups cannot rescan them.
type commandExpectation struct {
	known   bool
	invalid bool
	text    strings.Builder
}

func (p *commandExpectation) advance(step KeyStep) string {
	switch step.Kind {
	case KeyClearLine:
		p.known = !p.invalid
		p.text.Reset()
	case KeyText:
		if p.known {
			p.text.WriteString(step.Text)
		}
	case KeyWait:
	case KeyEnter:
		var target string
		if p.known && SlashCommand(p.text.String()) {
			// The projector drops terminal right-padding. Validate the full
			// authored line first, then use that same ASCII-space representation.
			target = strings.TrimRight(p.text.String(), " ")
		}
		p.known = false
		p.invalid = p.invalid || target == ""
		p.text.Reset()
		return target
	default:
		p.invalid = true
		p.known = false
		p.text.Reset()
	}
	return ""
}
