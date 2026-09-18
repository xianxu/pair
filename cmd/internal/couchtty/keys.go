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

	"github.com/xianxu/pair/cmd/internal/couchkeys"
	"github.com/xianxu/pair/cmd/internal/mouseinput"
)

// Couch's chords -- their encodings, scope and wording -- are declared in
// couchkeys (#282). This file joins them to the console's dispatch.

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
//
// It reads couchChords, so a chord declared in couchkeys is recognised here with
// no second edit.
func (k seqKind) hit() InterceptorHit {
	for _, chord := range couchChords {
		if chord.kind == k {
			return chord.hit
		}
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

// couchChord joins one declared chord (couchkeys) to the console's dispatch
// vocabulary: couchkeys says what a chord means and where; this adds how the
// console acts on it.
type couchChord struct {
	couchkeys.Binding
	hit  InterceptorHit
	kind seqKind
}

var couchChords = func() []couchChord {
	declared := couchkeys.Bindings()
	out := make([]couchChord, 0, len(declared))
	for _, binding := range declared {
		hit, kind := dispatchFor(binding.Action)
		out = append(out, couchChord{Binding: binding, hit: hit, kind: kind})
	}
	return out
}()

// dispatchFor is the one mapping from a declared action to the console's hit
// and sequence kind. An action with no arm returns HitNone and would be
// forwarded to the child; TestEveryCouchActionDispatches makes that a failure.
func dispatchFor(action couchkeys.Action) (InterceptorHit, seqKind) {
	switch action {
	case couchkeys.ActionSwitch:
		return HitSwitch, seqSwitch
	case couchkeys.ActionPrevious:
		return HitPrevious, seqPrevious
	case couchkeys.ActionNewestPage:
		return HitNewestPage, seqNewestPage
	case couchkeys.ActionDetach:
		return HitDetach, seqDetach
	case couchkeys.ActionPark:
		return HitPark, seqPark
	case couchkeys.ActionRelaunch:
		return HitRelaunch, seqRelaunch
	}
	return HitNone, seqNone
}

// actorReserved reports whether Couch takes this hit from a displayed Pair
// pane. It reads the declared scope, so routing and the help's context cannot
// disagree (#282).
func (hit InterceptorHit) actorReserved() bool {
	for _, chord := range couchChords {
		if chord.hit == hit {
			return chord.Scope == couchkeys.ScopeEveryPane
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
	// Switcher-scope chords are framed here too; Console forwards them while a
	// Pair pane has focus (couchkeys ScopeSwitcher, #245).
	for _, chord := range couchChords {
		for _, encoding := range chord.Encodings {
			if len(encoding) > 1 {
				sequences = append(sequences, struct {
					bytes []byte
					kind  seqKind
				}{encoding, chord.kind})
			}
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
// answer to "find a key in a stream and split around it". The two chord
// tables stay separate: Couch's is declared in couchkeys, its every-pane chords
// are Couch's alone, and its switcher chords borrow Pair's encodings for the
// same keys rather than merging the tables -- merging opposed tables is the bug
// rather than the cleanup (#282).
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
			for _, chord := range couchChords {
				for _, encoding := range chord.Encodings {
					if len(encoding) == 1 && encoding[0] == buf[idx] {
						i.rawHit = append([]byte(nil), encoding...)
						return out, chord.hit, buf[idx+1:]
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
