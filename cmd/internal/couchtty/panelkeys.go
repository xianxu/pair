package couchtty

import (
	"bytes"

	"github.com/xianxu/pair/cmd/internal/ansi"
)

// PanelKeyKind is what a keystroke MEANS to the panel.
type PanelKeyKind uint8

const (
	KeyRune PanelKeyKind = iota
	KeyUp
	KeyDown
	KeyEnter
	KeyEscape
	KeyBackspace
)

// PanelKey is one decoded keystroke.
type PanelKey struct {
	Kind PanelKeyKind
	Rune byte // set when Kind == KeyRune
}

// DecodePanelKeys turns raw terminal input into keystrokes the panel
// understands, returning any trailing PARTIAL sequence for the next read.
//
// Framing is the whole point. The first version of the panel took any printable
// byte as typeahead -- and an SGR mouse report is `\x1b[<0;12;4M`, whose bytes
// after the ESC are all printable. Moving the mouse over the panel typed
// `[<;0;M[<;;M...` into the filter, which then matched nothing and showed
// "(nothing running)" with no way back. Sequences are consumed WHOLE and the
// ones the panel does not use are DROPPED, rather than decaying into text.
//
// Framing goes through cmd/internal/ansi -- one scanner per package, and this
// is that package's second caller after Screen.
func DecodePanelKeys(in []byte) (keys []PanelKey, held []byte) {
	for i := 0; i < len(in); {
		b := in[i]
		if b == 0x1b {
			// SS3 first: ansi.Frame follows the regex order it replaced, where
			// `O` (0x4f) falls in the two-byte class -- so `\x1bOA` frames as
			// `\x1bO` and leaks the `A` as a typed rune. Application-cursor
			// mode is not exotic: it is whatever mode the previous child left
			// the terminal in, and couch does not get to assume.
			if len(in)-i >= 3 && in[i+1] == 'O' {
				if k, ok := decodeSequence(in[i : i+3]); ok {
					keys = append(keys, k)
				}
				i += 3
				continue
			}
			// A BARE ESC that is the whole remainder is the Escape KEY, not a
			// truncated sequence. Same discriminator the Interceptor uses: a
			// keystroke arrives as its own read, and holding it would make
			// Escape do nothing until the operator pressed something else.
			if len(in)-i == 1 {
				keys = append(keys, PanelKey{Kind: KeyEscape})
				i++
				continue
			}
			size, status := ansi.Frame(in[i:])
			switch status {
			case ansi.Incomplete:
				// A real prefix: carry it. Bounded by construction -- an
				// escape sequence is short, and a stream of them is consumed
				// as it completes.
				return keys, append([]byte(nil), in[i:]...)
			case ansi.Complete:
				if k, ok := decodeSequence(in[i : i+size]); ok {
					keys = append(keys, k)
				}
				// An unrecognised sequence (mouse, focus event, a chord the
				// workbench owns) is DROPPED. The panel is not a child; input
				// it has no meaning for is noise, not text.
				i += size
				continue
			}
			// ansi.None on an ESC: not a sequence this package frames. Drop
			// the ESC and carry on rather than typing it in.
			i++
			continue
		}
		switch {
		case b == '\r' || b == '\n':
			keys = append(keys, PanelKey{Kind: KeyEnter})
		case b == 0x7f || b == 0x08:
			keys = append(keys, PanelKey{Kind: KeyBackspace})
		case b >= 0x20 && b < 0x7f:
			keys = append(keys, PanelKey{Kind: KeyRune, Rune: b})
		default:
			// Other control bytes are ignored rather than filtered on.
		}
		i++
	}
	return keys, nil
}

// decodeSequence maps the escape sequences the panel acts on. Both the legacy
// and application-cursor forms of the arrows, because which one arrives depends
// on the mode the previous child left the terminal in -- and couch does not get
// to assume.
func decodeSequence(seq []byte) (PanelKey, bool) {
	switch {
	case bytes.Equal(seq, []byte("\x1b[A")), bytes.Equal(seq, []byte("\x1bOA")):
		return PanelKey{Kind: KeyUp}, true
	case bytes.Equal(seq, []byte("\x1b[B")), bytes.Equal(seq, []byte("\x1bOB")):
		return PanelKey{Kind: KeyDown}, true
	// Some terminals send ESC ESC for a pressed Escape while an app-mode is on.
	case bytes.Equal(seq, []byte("\x1b\x1b")):
		return PanelKey{Kind: KeyEscape}, true
	}
	return PanelKey{}, false
}
