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
			if len(in)-i >= 2 && in[i+1] == 'O' {
				if len(in)-i == 2 {
					return keys, append([]byte(nil), in[i:]...)
				}
				if k, ok := decodeSequence(in[i : i+3]); ok {
					keys = append(keys, k)
				}
				i += 3
				continue
			}
			// A bare ESC is ambiguous: it may be the key or the first byte of a
			// sequence split by Read. Hold it; the Console's IO loop resolves
			// the ambiguity with a short timeout.
			if len(in)-i == 1 {
				return keys, append([]byte(nil), in[i:]...)
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

// decodeSequence maps the escape sequences the panel acts on.
//
// EVERY key has two encodings and both are handled, because which one arrives
// depends on the keyboard mode the previous child left the terminal in -- and
// couch does not get to assume. zellij enables the Kitty keyboard protocol, so
// a real session's Escape is `\x1b[27u`, not `\x1b`.
//
// This generalises a fix that was applied to ONE key in M2: ctrl-space had the
// same problem, and handling only that one left Escape, Enter and the arrows
// dead in the panel. pair's own chord table carries both encodings for every
// chord for exactly this reason.
func decodeSequence(seq []byte) (PanelKey, bool) {
	switch {
	case bytes.Equal(seq, []byte("\x1b\x1b")):
		// ESC ESC: a pressed Escape while an app mode is on.
		return PanelKey{Kind: KeyEscape}, true
	case bytes.HasSuffix(seq, []byte("A")):
		if isCSI(seq) {
			return PanelKey{Kind: KeyUp}, true
		}
	case bytes.HasSuffix(seq, []byte("B")):
		if isCSI(seq) {
			return PanelKey{Kind: KeyDown}, true
		}
	case bytes.HasSuffix(seq, []byte("u")):
		return decodeCSIu(seq)
	}
	if bytes.Equal(seq, []byte("\x1bOA")) {
		return PanelKey{Kind: KeyUp}, true
	}
	if bytes.Equal(seq, []byte("\x1bOB")) {
		return PanelKey{Kind: KeyDown}, true
	}
	return PanelKey{}, false
}

// isCSI reports whether seq is `ESC [ <params> <final>`. Params are ignored:
// an arrow with a modifier is still an arrow, and the panel has no use for the
// modifier.
func isCSI(seq []byte) bool {
	return len(seq) >= 3 && seq[0] == 0x1b && seq[1] == '['
}

// decodeCSIu reads the Kitty protocol's `CSI <codepoint> [;<modifiers>] u`.
//
// The codepoint is the key; the modifiers are deliberately dropped except to
// refuse a MODIFIED printable, which is a chord rather than a character --
// typing `a` and pressing ctrl+a must not both insert an `a`.
func decodeCSIu(seq []byte) (PanelKey, bool) {
	if !isCSI(seq) {
		return PanelKey{}, false
	}
	body := seq[2 : len(seq)-1]
	code, mods := body, []byte(nil)
	if i := bytes.IndexByte(body, ';'); i >= 0 {
		code, mods = body[:i], body[i+1:]
	}
	n, ok := atoiBytes(code)
	if !ok {
		return PanelKey{}, false
	}
	// Modifier bitmask 1 means "none" in this protocol; anything else is a
	// chord.
	modified := len(mods) > 0 && !bytes.Equal(mods, []byte("1"))

	switch n {
	case 27:
		return PanelKey{Kind: KeyEscape}, true
	case 13:
		return PanelKey{Kind: KeyEnter}, true
	case 127, 8:
		return PanelKey{Kind: KeyBackspace}, true
	}
	if !modified && n >= 0x20 && n < 0x7f {
		return PanelKey{Kind: KeyRune, Rune: byte(n)}, true
	}
	return PanelKey{}, false
}

func atoiBytes(b []byte) (int, bool) {
	if len(b) == 0 {
		return 0, false
	}
	n := 0
	for _, c := range b {
		if c < '0' || c > '9' {
			return 0, false
		}
		n = n*10 + int(c-'0')
		if n > 0x10FFFF {
			return 0, false
		}
	}
	return n, true
}
