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
	"time"

	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
)

// hotkeyByte is ctrl-space in the LEGACY encoding: ctrl-@ is NUL.
const hotkeyByte = 0x00

// previousByte is ctrl+backspace in the LEGACY encoding. Unlike every other
// chord couch intercepts it is a bare byte, not an escape sequence, so it needs
// a branch beside hotkeyByte rather than a knownSequences row.
//
// Accepted cost, deliberate and not a discovery: in legacy encoding 0x08 IS
// ^H, so intercepting ctrl+backspace also takes ctrl-h from the child (readline
// and nvim insert-mode treat it as backspace). Under the Kitty protocol the two
// separate cleanly -- \x1b[104;5u vs \x1b[127;5u -- and zellij pushes the
// protocol, so this only bites with the protocol off.
const previousByte = 0x08

// escapeAmbiguity is the one deadline used by both terminal-input framers to
// distinguish an ESC key from the first byte of a split escape sequence.
const escapeAmbiguity = 35 * time.Millisecond

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
	seqHotkey = seqSwitch // compatibility name for the switch-sequence tests
)

// intercepts reports whether a sequence is CONSUMED by couch rather than
// forwarded to the child. Derived rather than enumerated at each site: a new
// chord that forgot to update a hand-written list would be silently forwarded,
// which is exactly the failure the Kitty encoding already caused once.
func (k seqKind) intercepts() bool {
	switch k {
	case seqSwitch, seqPark, seqPrevious, seqDetach, seqRelaunch:
		return true
	}
	return false
}

// hit maps an intercepted sequence to what the console should do about it.
func (k seqKind) hit() InterceptorHit {
	switch k {
	case seqSwitch:
		return HitSwitch
	case seqPark:
		return HitPark
	case seqPrevious:
		return HitPrevious
	case seqDetach:
		return HitDetach
	case seqRelaunch:
		return HitRelaunch
	}
	return HitNone
}

type InterceptorHit uint8

const (
	HitNone InterceptorHit = iota
	HitSwitch
	HitPark
	// HitPrevious is ctrl+backspace: return to the actor recorded by
	// SwitchTracker. The key labelled `delete` on an Apple keyboard, not
	// forward-delete -- no fn in the chord.
	HitPrevious
	// HitDetach is alt+d: stop this thread's pair client without tearing down
	// its zellij session.
	HitDetach
	// HitRelaunch is alt+n (or ctrl+alt+n): replace this thread's Pair process
	// with the current binary, keeping the agent conversation.
	HitRelaunch
)

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
	return []InterceptorHit{HitSwitch, HitPark, HitPrevious, HitDetach, HitRelaunch}
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
		{[]byte("\x1b[32;5u"), seqSwitch},
		// ctrl+backspace under the Kitty protocol: codepoint 127 with modifier
		// bitmask 4 encoded as 4+1. Its legacy form is the bare byte handled
		// above, not a sequence.
		{[]byte("\x1b[127;5u"), seqPrevious},
	}
	for _, chord := range []struct {
		chord workbenchshortcut.Chord
		kind  seqKind
	}{
		{workbenchshortcut.ChordAltX, seqPark},
		// alt+d is Pair's own detach chord, intercepted for the same reason
		// alt+x is: un-intercepted, PairConfirmDetach runs `zellij action
		// detach` from inside the session, leaving couch with a dead child and
		// a stale live incarnation -- which the fail-closed projector hides. The
		// operator's safest gesture would make the thread vanish from the
		// switcher. Intercepting costs the hosted Pair its own chord and buys
		// the durable retirement that makes the thread reattachable.
		//
		// ChordAltD has exactly one encoding: the table declares no legacy
		// "\x1bd", so with the Kitty protocol off alt+d passes through to the
		// child. zellij pushes the protocol, so this is a documented edge.
		{workbenchshortcut.ChordAltD, seqDetach},
		// alt+n and ctrl+alt+n are Pair's own reload chords, intercepted for a
		// reason that is sharper than alt+d's. Un-intercepted, Pair handles them
		// INSIDE the process couch spawned: `pair restart` writes a marker and
		// execs kill-session, the outer process unblocks, and createflow's loop
		// re-enters runOnce in the same process image. There is no re-exec, so
		// the binary in memory is the old one -- a rebuilt Pair is not what
		// comes back. For pair development that is worse than not working,
		// because it looks like it worked.
		//
		// BOTH aliases, and the second is not a nicety: on newer macOS
		// Option+n is a dead-tilde composer, which is why Pair carries
		// ctrl+alt+n at all. Taking only one would leave the operator's actual
		// keystroke reaching the child.
		//
		// alt+SHIFT+n is deliberately NOT taken (\x1b[78;4u, a distinct
		// sequence). It restarts only the agent conversation, and it is the
		// cheap in-session escape hatch that has to survive couch claiming the
		// heavier chord: alt+n means new code, same conversation;
		// alt+shift+n means same code, new conversation.
		//
		// Kitty-protocol edge, inherited from ChordAltD: neither declares a
		// legacy encoding, so with the protocol off both pass through to Pair
		// and do its old in-place reload. zellij pushes the protocol, so this is
		// a documented degradation -- but it means the behaviour differs
		// silently by protocol state, which is why it is written here.
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
	inPaste bool

	// held is a partial paste marker straddling a read boundary. Bounded by
	// construction: a marker is six bytes.
	held []byte
}

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

// FeedHit is Feed's typed form, distinguishing Couch switching from Pair's
// Alt+x full-quit chord intercepted as Couch Park.
func (i *Interceptor) FeedHit(in []byte) (before []byte, hit InterceptorHit, rest []byte) {
	buf := in
	if len(i.held) > 0 {
		buf = append(i.held, in...)
		i.held = nil
	}

	out := make([]byte, 0, len(buf))
	for idx := 0; idx < len(buf); {
		if !i.inPaste && buf[idx] == hotkeyByte {
			return out, HitSwitch, buf[idx+1:]
		}
		if !i.inPaste && buf[idx] == previousByte {
			return out, HitPrevious, buf[idx+1:]
		}
		if buf[idx] == 0x1b {
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
