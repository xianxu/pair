package vt

import (
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/parser"
)

// parameterGuard retains evidence the bounded upstream parser discards. It
// observes the same transition table before callbacks run, and survives writes
// and the entire DCS string. No prefix of an overflowing command is dispatched.
// The parser has one extra parameter slot so its final-slot accounting can
// represent all 32 supported parameters without truncating the last one.
type parameterGuard struct {
	separators int
	value      int
	overflow   bool
}

func (g *parameterGuard) advance(p *ansi.Parser, b byte) bool {
	old := p.State()
	if old == parser.Utf8State {
		return true
	}
	next, action := parser.Table.Transition(old, b)
	if action == parser.ClearAction || old == parser.EscapeState && next != old {
		*g = parameterGuard{}
	}
	if action != parser.ParamAction {
		return true
	}
	if g.overflow {
		return false
	}
	switch b {
	case ';', ':':
		g.separators++
		g.value = 0
		g.overflow = g.separators >= parser.MaxParamsSize
	default:
		if b >= '0' && b <= '9' {
			// MissingParam is reserved; HasMoreFlag must never become a value bit.
			const maxValue = parser.ParamMask - 1
			digit := int(b - '0')
			if g.value > (maxValue-digit)/10 {
				g.overflow = true
				return false
			}
			g.value = g.value*10 + digit
		}
	}
	return !g.overflow
}
