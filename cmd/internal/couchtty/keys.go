// Package couchtty is couch's console: the operator's terminal routed to one
// agent child at a time.
//
// The pure model lives here -- what a keystroke means, what the reserved row
// says, where "up one level" goes -- and the IO shell (console.go) does nothing
// but drive it against hostty.Host and ptychild.Child. Nothing in couchcore
// learns that a terminal exists.
package couchtty

import (
	"bytes"

	"github.com/xianxu/pair/cmd/internal/mouseinput"
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
)

// hotkeyByte is ctrl-space in the LEGACY encoding: ctrl-@ is NUL.
const hotkeyByte = 0x00

// previousByte is ctrl+backspace in the LEGACY encoding. Like hotkeyByte it
// is a bare-byte navigation encoding rather than a multi-byte known sequence.
//
// Accepted cost, deliberate and not a discovery: in legacy encoding 0x08 IS
// ^H, so intercepting ctrl+backspace also takes ctrl-h from the child (readline
// and nvim insert-mode treat it as backspace). Under the Kitty protocol the two
// separate cleanly -- \x1b[104;5u vs \x1b[127;5u -- and zellij pushes the
// protocol, so this only bites with the protocol off.
const previousByte = 0x08

// newestPageSequence is ctrl+return under the Kitty protocol: codepoint 13 with
// modifier bitmask 4 encoded as 4+1, the same construction as ctrl-space's row.
//
// It has no legacy form: CR is also ordinary Return. Couch maintains the
// disambiguation flag while it owns a supporting terminal; an unsupported host
// retains Ctrl+Space then Return as the notification-jump fallback.
//
// Named because two sites need the same bytes: the knownSequences row, and the
// panel arm of onNewestPageHotkey, which hands them to the panel's decoder.
const newestPageSequence = "\x1b[13;5u"

type seqKind uint8

const (
	seqNone seqKind = iota
	seqPartial
	seqPasteStart
	seqPasteEnd
	seqSwitch
	seqPark
	seqPrevious
	seqDetach
	seqRelaunch
	seqNewestPage
	// Every new kind goes ABOVE this line. An omitted expression in a const block
	// repeats the previous one, so a kind appended below would EQUAL seqSwitch.
	// With a hit() case that is a duplicate-case compile error; without one -- a
	// marker rather than a chord -- its sequence would silently open the switcher.
	seqHotkey = seqSwitch // compatibility name for the switch-sequence tests
)

// hit maps a sequence to what the console should do about it, and is the ONE
// place that knows. HitNone means the sequence is not couch's.
//
// intercepts() used to be a second switch over the same enum, so a new chord had
// to join both or be silently forwarded -- the failure the Kitty encoding
// already caused once, and the shape that let alt+n ship consumed-and-dropped.
// Two switches over one enum agree until someone edits one.
func (k seqKind) hit() InterceptorHit {
	for _, binding := range couchNavigation {
		if binding.kind == k {
			return binding.Hit
		}
	}
	switch k {
	case seqPark:
		return HitPark
	case seqDetach:
		return HitDetach
	case seqRelaunch:
		return HitRelaunch
	}
	return HitNone
}

// intercepts reports whether a sequence is a Couch candidate. Console decides
// its disposition after routing preceding input. Derived from hit().
func (k seqKind) intercepts() bool { return k.hit() != HitNone }

type InterceptorHit uint8

const (
	HitNone InterceptorHit = iota
	HitSwitch
	HitPark
	// HitPrevious is ctrl+backspace: return to the actor recorded by
	// SwitchTracker. The key labelled `delete` on an Apple keyboard, not
	// forward-delete -- no fn in the chord.
	HitPrevious
	// HitNewestPage is ctrl+return: jump straight to the thread ctrl-space would
	// have opened the switcher on, with no switcher in between. HitPrevious's
	// mirror image -- that one goes home, this one goes to the page.
	HitNewestPage
	// HitDetach is alt+d: stop this thread's pair client without tearing down
	// its zellij session.
	HitDetach
	// HitRelaunch is alt+n (or ctrl+alt+n): replace this thread's Pair process
	// with the current binary, keeping the agent conversation.
	HitRelaunch
	// HitMouse is an SGR mouse report. It has NO seqKind: a report's shape is
	// `\x1b[<button;col;rowM|m` with variable digits, so knownSequences -- which
	// holds fixed strings -- cannot express it, and FeedHit matches it against
	// mouseinput's predicate BEFORE consulting that table. A seqKind for it
	// existed briefly and was dead: sequenceAt can only return kinds the table
	// carries.
	//
	// Its additional payload is the decoded event and raw wire bytes, read
	// with Mouse() --
	// because a coordinate cannot be recovered from the hit alone and a
	// forwarded report must be the bytes the terminal sent, not a re-encoding.
	HitMouse
)

// CouchNavigationBinding declares one actor-wide navigation reservation. Help
// consumers use the same declarations as input matching and focus admission.
type CouchNavigationBinding struct {
	Hit       InterceptorHit
	Key, Help string
	Encodings [][]byte
}

var couchNavigation = []struct {
	CouchNavigationBinding
	kind seqKind
}{
	{CouchNavigationBinding{HitSwitch, "Ctrl+Space", "open the Couch switcher", [][]byte{{hotkeyByte}, []byte("\x1b[32;5u")}}, seqSwitch},
	{CouchNavigationBinding{HitPrevious, "Ctrl+Backspace", "return to the previous Couch thread", [][]byte{{previousByte}, []byte("\x1b[127;5u")}}, seqPrevious},
	{CouchNavigationBinding{HitNewestPage, "Ctrl+Return", "jump to the newest notification thread", [][]byte{[]byte(newestPageSequence), []byte("\x1b[13;5:1u"), []byte("\x1b[13;5:2u")}}, seqNewestPage},
}

func CouchNavigationBindings() []CouchNavigationBinding {
	out := make([]CouchNavigationBinding, 0, len(couchNavigation))
	for _, entry := range couchNavigation {
		binding := entry.CouchNavigationBinding
		binding.Encodings = make([][]byte, len(entry.Encodings))
		for n, encoding := range entry.Encodings {
			binding.Encodings[n] = append([]byte(nil), encoding...)
		}
		out = append(out, binding)
	}
	return out
}

func (hit InterceptorHit) actorReserved() bool {
	for _, entry := range couchNavigation {
		if entry.Hit == hit {
			return true
		}
	}
	return false
}

// MouseHit is the payload of a HitMouse.
type MouseHit struct {
	Event mouseinput.Event
	// Raw is the wire form, kept because the forward disposition writes the
	// child exactly what the terminal sent. Re-encoding from Event would be a
	// second source of truth for the format (ARCH-DRY).
	Raw []byte
}

// AllInterceptorHits is every hit the console must be able to act on.
//
// It exists so "the dispatch switch is exhaustive" can be CHECKED rather than
// remembered. alt+n shipped once intercepted with no arm in that switch: the
// bytes were consumed and the operation never ran, silently, and every
// interceptor test stayed green because the Interceptor's half was correct. A
// hand-written switch cannot report the case it forgot; this list is what a test
// walks to prove one exists for each.
//
// HitNone is deliberately absent: it is the ABSENCE of a hit, and giving it a
// handler would be inventing an action for "nothing happened".
func AllInterceptorHits() []InterceptorHit {
	return []InterceptorHit{HitSwitch, HitPark, HitPrevious, HitNewestPage, HitDetach, HitRelaunch, HitMouse}
}

// knownSequences is every multi-byte sequence the console must recognise in the
// operator's input. Everything else is forwarded untouched -- couch does not
// frame the child's keyboard.
//
// The Kitty row is the one an M2 operator smoke had to teach us. zellij enables
// the Kitty keyboard protocol, so the terminal stops sending NUL for ctrl-space
// and sends CSI-u instead: space is codepoint 32, ctrl is modifier bitmask 4
// encoded as 4+1. Knowing only the legacy byte meant ctrl-space sailed through
// to the child and landed in draft nvim. pair's own chord table carries both
// encodings for every chord (workbenchshortcut/shortcut.go:294-312); this is the
// same lesson arriving one layer up.
//
// Exact strings, matching how workbenchshortcut does it. A tolerant parser for
// CSI-u variants would also have to decide what `\x1b[32;5:3u` (key RELEASE)
// means, and guessing there is how a switcher fires twice per keypress.
var knownSequences = func() []struct {
	bytes []byte
	kind  seqKind
} {
	sequences := []struct {
		bytes []byte
		kind  seqKind
	}{
		{[]byte("\x1b[200~"), seqPasteStart},
		{[]byte("\x1b[201~"), seqPasteEnd},
	}
	for _, binding := range couchNavigation {
		for _, encoding := range binding.Encodings {
			if len(encoding) > 1 {
				sequences = append(sequences, struct {
					bytes []byte
					kind  seqKind
				}{encoding, binding.kind})
			}
		}
	}

	for _, chord := range []struct {
		chord workbenchshortcut.Chord
		kind  seqKind
	}{
		{workbenchshortcut.ChordAltX, seqPark},
		// Lifecycle candidates remain recognized for the switcher. Console
		// forwards their exact bytes when prefix routing leaves actor focus.
		{workbenchshortcut.ChordAltD, seqDetach},
		{workbenchshortcut.ChordAltN, seqRelaunch},
		{workbenchshortcut.ChordCtrlAltN, seqRelaunch},
	} {
		for _, encoding := range workbenchshortcut.ChordEncodings(chord.chord) {
			sequences = append(sequences, struct {
				bytes []byte
				kind  seqKind
			}{encoding, chord.kind})
		}
	}
	return sequences
}()

// Interceptor splits the operator's keystrokes around the hotkey.
//
// It returns a SPLIT rather than a filtered buffer because the bytes either
// side of the hotkey belong to different children: in `x<ctrl-space>y`, x goes
// to the child being left and y to the one landed on. The shape is
// workbenchshortcut.FindChord's, deliberately -- that is the repo's existing
// answer to "find a key in a stream and split around it". The chord TABLE is
// not shared: couch claims a handful of keys, the workbench has a dozen, and
// merging opposed tables is the bug rather than the cleanup.
//
// One piece of state, and it earns its place: a bracketed paste can carry
// arbitrary bytes, and a pasted NUL that silently switches actors while eating a
// byte is data loss the operator would never trace back.
type Interceptor struct {
	rawHit  []byte
	inPaste bool

	// mouse is the payload of the hit just returned. Read with Mouse()
	// immediately after a HitMouse; overwritten by the next one.
	mouse MouseHit

	// held is a partial paste marker straddling a read boundary. Bounded by
	// construction: a marker is six bytes.
	held []byte
}

// Mouse is the payload of the HitMouse just returned.
func (i *Interceptor) Mouse() MouseHit { return i.mouse }

// RawHit is the exact wire payload of the latest FeedHit candidate. Read it
// before the next FeedHit; nil means the latest feed found no candidate.
func (i *Interceptor) RawHit() []byte { return i.rawHit }

// Flush resolves an ambiguous partial as literal child input. The IO owner
// calls it only after its short escape-key timeout expires.
func (i *Interceptor) Flush() []byte {
	out := append([]byte(nil), i.held...)
	i.held = nil
	return out
}

// Feed consumes a chunk of operator input.
//
// before is for the current focus; hit says the hotkey fired; rest is for the
// focus landed on and is fed back in by the caller after switching. With no
// hotkey, before is everything and rest is empty -- one place to look.
func (i *Interceptor) Feed(in []byte) (before []byte, hit bool, rest []byte) {
	before, typed, rest := i.FeedHit(in)
	return before, typed != HitNone, rest
}

// FeedHit frames candidates without assigning focus authority. Console routes
// preceding input before deciding whether the candidate belongs to Couch.
func (i *Interceptor) FeedHit(in []byte) (before []byte, hit InterceptorHit, rest []byte) {
	i.rawHit = nil
	buf := in
	if len(i.held) > 0 {
		buf = append(i.held, in...)
		i.held = nil
	}

	out := make([]byte, 0, len(buf))
	for idx := 0; idx < len(buf); {
		if !i.inPaste {
			for _, binding := range couchNavigation {
				for _, encoding := range binding.Encodings {
					if len(encoding) == 1 && encoding[0] == buf[idx] {
						i.rawHit = append([]byte(nil), encoding...)
						return out, binding.Hit, buf[idx+1:]
					}
				}
			}
		}
		if buf[idx] == 0x1b {
			// A mouse report BEFORE the fixed-string table: its shape is
			// variable so knownSequences cannot hold it, and couch cannot
			// WITHHOLD a report it does not recognise -- the interceptor
			// forwards anything unknown, which is why a child that never
			// enabled tracking used to receive reports anyway.
			if event, raw, rest, ok := mouseinput.ParsePrefix(buf[idx:]); ok {
				if !i.inPaste {
					i.mouse = MouseHit{Event: event, Raw: append([]byte(nil), raw...)}
					i.rawHit = i.mouse.Raw
					return out, HitMouse, rest
				}
				// Inside a paste it is content, like any other byte.
				out = append(out, raw...)
				idx += len(raw)
				continue
			}
			if mouseinput.IsPrefix(buf[idx:]) && len(buf)-idx <= mouseinput.MaxReport {
				// Bounded: an unterminated introducer held forever parks every
				// following keystroke -- #127's dead keyboard, which shipped
				// once already. Past the bound this is not a report, so it
				// falls through and becomes ordinary input.
				i.held = append([]byte(nil), buf[idx:]...)
				return out, HitNone, nil
			}
			n, kind := sequenceAt(buf[idx:])
			switch kind {
			case seqPartial:
				// Every genuine prefix is held, including a bare ESC. A read
				// boundary is not a keystroke boundary: forwarding that first
				// byte loses every Kitty key and paste marker split there. The
				// IO owner resolves an actual ESC key through Flush after a
				// short ambiguity timeout, matching the panel decoder's rule.
				i.held = append([]byte(nil), buf[idx:]...)
				return out, HitNone, nil
			case seqPasteStart, seqPasteEnd:
				i.inPaste = kind == seqPasteStart
				out = append(out, buf[idx:idx+n]...)
				idx += n
				continue
			}
			// intercepts(), not a hand-written list of kinds. This site used to
			// enumerate them, which is the precise failure intercepts() exists
			// to prevent -- its own comment says so: "a new chord that forgot to
			// update a hand-written list would be silently forwarded". Adding
			// seqRelaunch updated intercepts() and hit() and was still silently
			// forwarded here, which is the third copy proving the point.
			if kind.intercepts() {
				if !i.inPaste {
					i.rawHit = append([]byte(nil), buf[idx:idx+n]...)
					return out, kind.hit(), buf[idx+n:]
				}
				// Inside a paste it is content, like any other byte.
				out = append(out, buf[idx:idx+n]...)
				idx += n
				continue
			}
			// seqNone: an ordinary escape sequence -- one of the workbench's
			// own chords, an arrow key, anything. Fall through and copy its
			// bytes; couch does not frame the child's keyboard beyond the
			// sequences it must recognise.
		}
		out = append(out, buf[idx])
		idx++
	}
	return out, HitNone, nil
}

// sequenceAt classifies the bytes at buf[0] against knownSequences.
//
// The distinction that matters is PARTIAL versus NONE. `\x1b[2~` is the Insert
// key and shares three bytes with `\x1b[200~`; treating it as an unfinished
// sequence would park it, and every keystroke behind it, exactly as #127's dead
// keyboard did. A run is partial only while it is a genuine PREFIX of something
// known; once it diverges from all of them it is ordinary input.
func sequenceAt(buf []byte) (int, seqKind) {
	for _, s := range knownSequences {
		if bytes.HasPrefix(buf, s.bytes) {
			return len(s.bytes), s.kind
		}
	}
	for _, s := range knownSequences {
		// buf shorter than s and matching so far: still a real prefix.
		if len(buf) < len(s.bytes) && bytes.HasPrefix(s.bytes, buf) {
			return 0, seqPartial
		}
	}
	return 0, seqNone
}
