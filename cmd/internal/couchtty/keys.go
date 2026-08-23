// Package couchtty is couch's console: the operator's terminal routed to one
// agent child at a time.
//
// The pure model lives here -- what a keystroke means, what the reserved row
// says, where "up one level" goes -- and the IO shell (console.go) does nothing
// but drive it against hostty.Host and ptychild.Child. Nothing in couchcore
// learns that a terminal exists.
package couchtty

import "bytes"

// The bracketed-paste markers, defined ONCE. A protocol with paired delimiters
// gets one constant pair and every site derives from it; two sites framing the
// same delimiter independently is the bug this repo paid for in #127.
var (
	pasteStart = []byte("\x1b[200~")
	pasteEnd   = []byte("\x1b[201~")
)

// hotkey is ctrl-space. A bare key, not a chord, so there is no timing window
// and no prefix table -- which is precisely why the Spec chose it over
// double-ESC.
const hotkey = 0x00

// Interceptor splits the operator's keystrokes around the hotkey.
//
// It returns a SPLIT rather than a filtered buffer because the bytes either
// side of the hotkey belong to different children: in `x<ctrl-space>y`, x goes
// to the child being left and y to the one landed on. The shape is
// workbenchshortcut.FindChord's, deliberately -- that is the repo's existing
// answer to "find a key in a stream and split around it". The chord TABLE is
// not shared: couch has one key, the workbench has a dozen, and merging opposed
// tables is the bug rather than the cleanup.
//
// One piece of state, and it earns its place: a bracketed paste can carry
// arbitrary bytes, and a pasted NUL that silently switches actors while eating a
// byte is data loss the operator would never trace back.
type Interceptor struct {
	inPaste bool

	// held is a partial paste marker straddling a read boundary. Bounded by
	// construction: a marker is six bytes.
	held []byte
}

// Feed consumes a chunk of operator input.
//
// before is for the current focus; hit says the hotkey fired; rest is for the
// focus landed on and is fed back in by the caller after switching. With no
// hotkey, before is everything and rest is empty -- one place to look.
func (i *Interceptor) Feed(in []byte) (before []byte, hit bool, rest []byte) {
	buf := in
	if len(i.held) > 0 {
		buf = append(i.held, in...)
		i.held = nil
	}

	out := make([]byte, 0, len(buf))
	for idx := 0; idx < len(buf); {
		if !i.inPaste && buf[idx] == hotkey {
			return out, true, buf[idx+1:]
		}
		if buf[idx] == 0x1b {
			n, kind := markerAt(buf[idx:])
			switch kind {
			case markerPartial:
				// A REAL prefix -- hold it. Anything already scanned still
				// goes to the current focus.
				i.held = append([]byte(nil), buf[idx:]...)
				return out, false, nil
			case markerStart, markerEnd:
				i.inPaste = kind == markerStart
				out = append(out, buf[idx:idx+n]...)
				idx += n
				continue
			}
			// markerNone: an ordinary escape sequence. Fall through and copy
			// its bytes one at a time -- couch does not frame the operator's
			// input beyond the two markers it must know about.
		}
		out = append(out, buf[idx])
		idx++
	}
	return out, false, nil
}

type markerKind uint8

const (
	markerNone markerKind = iota
	markerPartial
	markerStart
	markerEnd
)

// markerAt classifies the bytes at buf[0] against the two paste markers.
//
// The distinction that matters is PARTIAL versus NONE. `\x1b[2~` is the Insert
// key and shares three bytes with `\x1b[200~`; treating it as an unfinished
// marker would park it, and every keystroke behind it, exactly as #127's dead
// keyboard did. Once a byte diverges from BOTH markers the run is not a marker
// and is emitted as ordinary input.
func markerAt(buf []byte) (int, markerKind) {
	switch {
	case bytes.HasPrefix(buf, pasteStart):
		return len(pasteStart), markerStart
	case bytes.HasPrefix(buf, pasteEnd):
		return len(pasteEnd), markerEnd
	}
	if len(buf) >= len(pasteStart) {
		return 0, markerNone // long enough to have matched, and did not
	}
	if bytes.HasPrefix(pasteStart, buf) || bytes.HasPrefix(pasteEnd, buf) {
		return 0, markerPartial
	}
	return 0, markerNone
}
