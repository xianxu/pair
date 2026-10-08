package ansiparser

import "github.com/charmbracelet/x/ansi/parser"

// stringUTF8 recognizes valid UTF-8 prefixes without retaining payload bytes.
// It survives transport boundaries and buffer overflow; the parser still owns
// payload retention and control-sequence dispatch.
type stringUTF8 struct{ remaining, min, max byte }

func isStringState(s parser.State) bool {
	switch s {
	case parser.OscStringState, parser.DcsStringState, parser.SosStringState, parser.PmStringState, parser.ApcStringState:
		return true
	}
	return false
}

// payload consumes valid continuation bytes before C1 controls are considered.
// A broken prefix falls back to ordinary control handling for the current byte.
func (u *stringUTF8) payload(b byte) bool {
	if u.remaining > 0 && b >= u.min && b <= u.max {
		u.remaining--
		u.min, u.max = 0x80, 0xbf
		return true
	}
	*u = stringUTF8{min: 0x80, max: 0xbf}
	switch {
	case b >= 0xc2 && b <= 0xdf:
		u.remaining = 1
	case b >= 0xe0 && b <= 0xef:
		u.remaining = 2
		if b == 0xe0 {
			u.min = 0xa0
		}
		if b == 0xed {
			u.max = 0x9f
		}
	case b >= 0xf0 && b <= 0xf4:
		u.remaining = 3
		if b == 0xf0 {
			u.min = 0x90
		}
		if b == 0xf4 {
			u.max = 0x8f
		}
	}
	// Non-control high bytes are opaque string data, even if malformed. In
	// particular, SOS/PM/APC must not enter the screen's UTF-8 printing state.
	return b >= 0xa0
}
