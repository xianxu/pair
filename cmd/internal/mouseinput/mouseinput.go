// Package mouseinput decodes SGR (DEC private mode 1006) mouse reports.
//
// It exists because two surfaces need the same decoder and a second copy would
// be a second wire format. termcmd has parsed these since the terminal mux
// learned to route clicks; couch needs the identical parse to decide whether a
// report is its own, the child's, or nobody's (pair#172). The rule that made
// the move worth doing rather than copying: one parser means one answer to
// "where does this sequence end", which is the decision #127's dead keyboard
// came from making twice.
package mouseinput

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

// An SGR (1006) mouse event is "\x1b[<button;col;rowT" where T is 'M' for a
// press and 'm' for a RELEASE. Both terminators must be recognized: treating
// 'm' as "sequence not finished yet" parks the release -- and then every
// keystroke behind it -- in the caller's held buffer, which reads as a dead
// keyboard, and leaves the child app holding an unmatched button-press (nvim
// stays in a mouse drag, i.e. stuck in visual selection).
const Terminators = "Mm"

// MaxReport bounds a well-formed report: "\x1b[<" + three numbers with
// separators + one terminator. Generous -- a real report is under 20 bytes --
// because the bound is a memory and liveness guard, not a plausibility
// judgement. A caller holding a prefix longer than this has something that is
// not a mouse report and must stop waiting for one.
const MaxReport = 32

// Wheel button codes. The wheel reports press-only, which is why a release is
// never a wheel tick.
const (
	WheelUp   = 64
	WheelDown = 65
)

// Modifier bits carried IN the button field. They are part of the button value,
// not a separate field: ctrl+wheel-up is 64|16 = 80, which is why comparing a
// raw Button against WheelUp misses every modified wheel tick.
const (
	ModShift = 4
	ModAlt   = 8
	ModCtrl  = 16
	ModMask  = ModShift | ModAlt | ModCtrl
)

// BaseButton strips the modifier bits, leaving the button a caller means when
// it asks "is this the wheel?".
func BaseButton(button int) int { return button &^ ModMask }

// WithButton returns raw with only its button field replaced — separators,
// coordinates and terminator stay exactly the bytes the terminal sent.
//
// Deliberately a splice, not a re-encoder. Rebuilding the report from Event
// would make this a second source of truth for the wire format, which is the
// thing this package exists to prevent (see the package doc); a splice cannot
// drift on the parts it does not touch. It refuses anything Parse refuses, so a
// caller can never turn a malformed report into a well-formed-looking one, and
// refuses a negative button, which has no encoding.
func WithButton(raw []byte, button int) ([]byte, bool) {
	if button < 0 {
		return nil, false
	}
	if _, ok := Parse(raw); !ok {
		return nil, false
	}
	sep := bytes.IndexByte(raw, ';')
	if sep < 0 {
		return nil, false
	}
	out := make([]byte, 0, len(raw)+4)
	out = append(out, "\x1b[<"...)
	out = strconv.AppendInt(out, int64(button), 10)
	out = append(out, raw[sep:]...)
	return out, true
}

// Event is one decoded report.
type Event struct {
	Button  int
	X       int
	Y       int
	Release bool
}

// Parse decodes a complete report, terminator included.
func Parse(data []byte) (Event, bool) {
	s := string(data)
	if !strings.HasPrefix(s, "\x1b[<") || s == "" {
		return Event{}, false
	}
	term := s[len(s)-1:]
	if !strings.Contains(Terminators, term) {
		return Event{}, false
	}
	var event Event
	if _, err := fmt.Sscanf(s, "\x1b[<%d;%d;%d"+term, &event.Button, &event.X, &event.Y); err != nil {
		return Event{}, false
	}
	event.Release = term == "m"
	return event, true
}

// ParsePrefix decodes a report at the head of data, returning it, its RAW
// bytes, and what follows.
//
// The raw bytes are returned, not reconstructed: a caller that forwards a report
// to a child must write exactly what the terminal sent, and re-encoding from the
// parsed fields would be a second source of truth for the wire format.
func ParsePrefix(data []byte) (event Event, raw []byte, rest []byte, ok bool) {
	if !bytes.HasPrefix(data, []byte("\x1b[<")) {
		return Event{}, nil, data, false
	}
	end := bytes.IndexAny(data, Terminators)
	if end < 0 {
		return Event{}, nil, data, false
	}
	raw = data[:end+1]
	event, ok = Parse(raw)
	if !ok {
		return Event{}, nil, data, false
	}
	return event, raw, data[end+1:], true
}

// Find locates the first report anywhere in data.
func Find(data []byte) (before []byte, event Event, raw []byte, rest []byte, ok bool) {
	start := bytes.Index(data, []byte("\x1b[<"))
	if start < 0 {
		return data, Event{}, nil, nil, false
	}
	event, raw, rest, ok = ParsePrefix(data[start:])
	if !ok {
		return data, Event{}, nil, nil, false
	}
	return data[:start], event, raw, rest, true
}

// IsPrefix reports whether data could still become a report: a genuine prefix of
// the introducer, or an introducer whose terminator has not arrived.
//
// A caller holding on this must ALSO bound the wait -- see MaxReport. Holding
// without a bound is how a stray "\x1b[<" parks every following keystroke.
func IsPrefix(data []byte) bool {
	return bytes.HasPrefix([]byte("\x1b[<"), data) ||
		(bytes.HasPrefix(data, []byte("\x1b[<")) && bytes.IndexAny(data, Terminators) < 0)
}
