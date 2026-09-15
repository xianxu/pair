package terminal

import (
	"bytes"
	"fmt"
	"unicode"
	"unicode/utf8"

	uv "github.com/charmbracelet/ultraviolet"
)

// InputEvent separates protocol decoding from product shortcut policy. Raw owns
// the received bytes. Canonical is only for policy; Event is sent to Endpoint.
// A release has no Canonical bytes so a shortcut is not invoked twice.
type InputEvent struct {
	Event          uv.Event
	Raw, Canonical []byte
	Reply          bool
}

// Decoder is a bounded incremental framing adapter, not a keyboard protocol
// implementation. Event semantics belong to ultraviolet.EventDecoder. Its zero
// value is ready; after an error the connection must stop, not silently resync.
type Decoder struct {
	decoder     uv.EventDecoder
	pending     []byte
	scan        int
	pastePrefix []byte
	failure     error
}

func (d *Decoder) PendingEscape() bool {
	return d.failure == nil && len(d.pastePrefix) == 0 && bytes.Equal(d.pending, []byte{27})
}

func (d *Decoder) FlushEscape() ([]InputEvent, error) {
	if d.failure != nil {
		return nil, d.failure
	}
	if !d.PendingEscape() {
		return nil, nil
	}
	raw := d.pending
	d.pending = nil
	d.scan = 0
	_, event := d.decoder.Decode(raw)
	return []InputEvent{makeInputEvent(event, raw)}, nil
}

func (d *Decoder) fail(err error) error {
	d.failure = err
	d.pending = nil
	d.pastePrefix = nil
	d.scan = 0
	return err
}

func (d *Decoder) Feed(data []byte) (events []InputEvent, err error) {
	if d.failure != nil {
		return nil, d.failure
	}
	for len(data) > 0 {
		limit := MaxStringBytes
		if len(d.pastePrefix) > 0 {
			limit = MaxPasteBytes + len("\x1b[201~")
		}
		room := limit - len(d.pending)
		if room <= 0 {
			return events, d.fail(fmt.Errorf("terminal: input frame exceeds %d bytes", limit))
		}
		n := min(room, len(data))
		d.pending = append(d.pending, data[:n]...)
		data = data[n:]
		for len(d.pending) > 0 {
			if len(d.pastePrefix) > 0 {
				start := min(d.scan, len(d.pending))
				marker := "\x1b[201~"
				if d.pastePrefix[0] == 0x9b {
					marker = "\x9b201~"
				}
				end := bytes.Index(d.pending[start:], []byte(marker))
				term := len(marker)
				if end < 0 {
					// Retain only the terminator's overlap in the next search. Payload is
					// never interpreted, so embedded escape sequences stay literal.
					d.scan = max(0, len(d.pending)-5)
					overlap := 0
					for n := 1; n < len(marker); n++ {
						if bytes.HasSuffix(d.pending, []byte(marker[:n])) {
							overlap = max(overlap, n)
						}
					}
					if len(d.pending)-overlap > MaxPasteBytes {
						return events, d.fail(fmt.Errorf("terminal: paste exceeds %d bytes", MaxPasteBytes))
					}
					break
				}
				end += start
				if end > MaxPasteBytes {
					return events, d.fail(fmt.Errorf("terminal: paste exceeds %d bytes", MaxPasteBytes))
				}
				raw := make([]byte, 0, len(d.pastePrefix)+end+term)
				raw = append(raw, d.pastePrefix...)
				raw = append(raw, d.pending[:end+term]...)
				event := uv.PasteEvent{Content: string(d.pending[:end])}
				events = append(events, makeInputEvent(event, raw))
				d.pastePrefix = nil
				d.consume(end + term)
				continue
			}
			n, frameErr := d.frameLength()
			if frameErr != nil {
				return events, d.fail(frameErr)
			}
			if n == 0 {
				break
			}
			raw := append([]byte(nil), d.pending[:n]...)
			consumed, event := d.decoder.Decode(raw)
			// Legacy terminal Alt-arrow aliases already supported by Pair.
			if bytes.Equal(raw, []byte("\x1b[3D")) {
				consumed, event = len(raw), uv.KeyPressEvent{Code: uv.KeyLeft, Mod: uv.ModAlt}
			}
			if bytes.Equal(raw, []byte("\x1b[3C")) {
				consumed, event = len(raw), uv.KeyPressEvent{Code: uv.KeyRight, Mod: uv.ModAlt}
			}
			// ultraviolet currently classifies a valid U+FFFD as malformed.
			// Correct that one decoded-rune case without accepting invalid bytes.
			if bytes.Equal(raw, []byte("�")) {
				consumed, event = len(raw), uv.KeyPressEvent{Code: utf8.RuneError, Text: "�"}
			} else if bytes.Equal(raw, []byte("\x1b�")) {
				consumed, event = len(raw), uv.KeyPressEvent{Code: utf8.RuneError, Mod: uv.ModAlt}
			}
			if consumed != len(raw) {
				// Unsupported complete controls are host reports, never fragments of
				// keyboard input. Keep their entire frame intact for diagnostics.
				event = uv.UnknownEvent(string(raw))
			}
			if _, ok := event.(uv.PasteStartEvent); ok {
				d.pastePrefix = raw
				d.consume(n)
				continue
			}
			if multiple, ok := event.(uv.MultiEvent); ok {
				// UV intentionally returns both F3 and CPR for CSI 1;<m>R.
				// If a frame has any reply interpretation, none of its events may
				// be admitted as a keystroke. Enhanced F3 is unambiguous.
				reply := false
				for _, one := range multiple {
					if makeInputEvent(one, nil).Reply {
						reply = true
					}
				}
				// A platform report can describe several semantic events. Exactly the
				// first owns Raw, keeping stream concatenation lossless without replay.
				for i, one := range multiple {
					var owned []byte
					if i == 0 {
						owned = raw
					}
					decoded := makeInputEvent(one, owned)
					if reply {
						decoded.Reply = true
						decoded.Canonical = nil
					}
					events = append(events, decoded)
				}
			} else {
				events = append(events, makeInputEvent(event, raw))
			}
			d.consume(n)
		}
	}
	return events, nil
}

func (d *Decoder) consume(n int) {
	d.pending = d.pending[n:]
	d.scan = 0
	if len(d.pending) == 0 {
		d.pending = nil
	}
}

// frameLength recognizes only framing. No key, modifier, mouse or response
// semantics are decoded here. scan prevents quadratic work on split strings.
func (d *Decoder) frameLength() (int, error) {
	b := d.pending
	if len(b) == 0 {
		return 0, nil
	}
	intro, start := b[0], 1
	if intro == 27 {
		if len(b) < 2 {
			return 0, nil
		}
		intro = b[1]
		start = 2
		switch intro {
		case '[':
			intro = 0x9b
		case 'O':
			intro = 0x8f
		case ']':
			intro = 0x9d
		case 'P':
			intro = 0x90
		case '_':
			intro = 0x9f
		case '^':
			intro = 0x9e
		case 'X':
			intro = 0x98
		default:
			if !utf8.FullRune(b[1:]) {
				return 0, nil
			}
			r, n := utf8.DecodeRune(b[1:])
			if r == utf8.RuneError && n == 1 {
				return 0, fmt.Errorf("terminal: invalid UTF-8 in Alt key")
			}
			return 1 + n, nil
		}
	}
	switch intro {
	case 0x9b, 0x8f:
		for i := max(start, d.scan); i < len(b); i++ {
			c := b[i]
			if c >= 0x40 && c <= 0x7e {
				if intro == 0x9b && i == start && c == 'M' {
					if len(b) < i+4 {
						return 0, nil
					}
					return i + 4, nil
				}
				return i + 1, nil
			}
			if c < 0x20 || c > 0x3f {
				return 0, fmt.Errorf("terminal: malformed input control")
			}
		}
		d.scan = len(b)
		return 0, nil
	case 0x9d, 0x90, 0x9f, 0x9e, 0x98:
		for i := max(start, d.scan); i < len(b); {
			if b[i] == 0x18 || b[i] == 0x1a {
				return 0, fmt.Errorf("terminal: cancelled input string")
			}
			if intro == 0x9d && b[i] == 7 {
				return i + 1, nil
			}
			if b[i] == 0x9c {
				return i + 1, nil
			}
			if b[i] == 27 && i+1 == len(b) {
				d.scan = i
				return 0, nil
			}
			if b[i] == 27 && i+1 < len(b) && b[i+1] == '\\' {
				return i + 2, nil
			}
			if b[i] >= utf8.RuneSelf {
				if !utf8.FullRune(b[i:]) {
					d.scan = i
					return 0, nil
				}
				_, n := utf8.DecodeRune(b[i:])
				i += n
				continue
			}
			i++
		}
		d.scan = len(b)
		return 0, nil
	default:
		if b[0] < utf8.RuneSelf {
			return 1, nil
		}
		if !utf8.FullRune(b) {
			return 0, nil
		}
		r, n := utf8.DecodeRune(b)
		if r == utf8.RuneError && n == 1 {
			return 0, fmt.Errorf("terminal: invalid UTF-8 input")
		}
		return n, nil
	}
}

func makeInputEvent(event uv.Event, raw []byte) InputEvent {
	out := InputEvent{Event: event, Raw: append([]byte(nil), raw...)}
	switch e := event.(type) {
	case uv.KeyPressEvent:
		out.Canonical = canonicalKey(uv.Key(e), raw)
	case uv.KeyReleaseEvent:
	case uv.MouseClickEvent, uv.MouseReleaseEvent, uv.MouseMotionEvent, uv.MouseWheelEvent, uv.FocusEvent, uv.BlurEvent, uv.PasteEvent:
		out.Canonical = append([]byte(nil), raw...)
	default:
		out.Reply = true
	}
	return out
}

func canonicalKey(k uv.Key, raw []byte) []byte {
	mod := k.Mod &^ (uv.ModCapsLock | uv.ModNumLock | uv.ModScrollLock)
	if k.Code == uv.KeySpace && mod == uv.ModCtrl {
		return []byte{0}
	}
	if mod == 0 {
		switch k.Code {
		case uv.KeyEscape:
			return []byte{27}
		case uv.KeyEnter:
			return []byte{'\r'}
		case uv.KeyTab:
			return []byte{'\t'}
		case uv.KeyBackspace:
			return []byte{127}
		case uv.KeyUp:
			return []byte("\x1b[A")
		case uv.KeyDown:
			return []byte("\x1b[B")
		case uv.KeyRight:
			return []byte("\x1b[C")
		case uv.KeyLeft:
			return []byte("\x1b[D")
		case uv.KeyHome:
			return []byte("\x1b[H")
		case uv.KeyEnd:
			return []byte("\x1b[F")
		}
	}
	if k.Code >= 32 && k.Code <= unicode.MaxRune && mod&^(uv.ModShift|uv.ModAlt) == 0 {
		text := k.Text
		if text == "" {
			code := k.Code
			if mod&uv.ModShift != 0 {
				if k.ShiftedCode != 0 {
					code = k.ShiftedCode
				} else {
					code = unicode.ToUpper(code)
				}
			}
			text = string(code)
		}
		if mod&uv.ModAlt != 0 {
			return append([]byte{27}, []byte(text)...)
		}
		return []byte(text)
	}
	// Keep modified special keys and Ctrl-letter disambiguation. Canonical is a
	// policy representation, not an attempt to encode the child's protocol.
	if k.Code >= 0 && k.Code <= unicode.MaxRune && mod&^(uv.ModShift|uv.ModAlt|uv.ModCtrl) == 0 && bytes.HasPrefix(raw, []byte("\x1b[")) && bytes.HasSuffix(raw, []byte("u")) {
		return []byte(fmt.Sprintf("\x1b[%d;%du", k.Code, int(mod)+1))
	}
	return append([]byte(nil), raw...)
}
