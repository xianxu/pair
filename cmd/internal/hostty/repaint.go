package hostty

import "github.com/xianxu/pair/cmd/internal/ptychild"

// Repaint composes the bytes that hand the host screen over to a child.
//
// It lives HERE and not in ptychild (#209): the composition emits host-side
// control sequences, hostty imports ptychild and not the reverse, and putting
// it the other way would invert the child/host split #146 drew.
//
// Pure: it returns bytes and writes nothing, so the ordering below — which is
// the whole difficulty — is unit-testable without a terminal.

// RepaintIntent distinguishes the two things a takeover can mean. They differ
// in exactly one case and it is not inferable from the byte slice.
type RepaintIntent int

const (
	// RepaintReplace repaints from a child's retained output. When there is
	// nothing retained it must NOT blank the screen: an empty replay means the
	// ring could not answer, and a stale frame beats a blank one while the
	// child is asked to repaint.
	RepaintReplace RepaintIntent = iota
	// RepaintClear deliberately blanks. `pair term` does this when registering a
	// new tab, to clear before releasing startup output so the queued live copy
	// is not duplicated. An empty replay HERE means "blank it", which is why
	// intent is carried rather than derived from len(replay).
	RepaintClear
)

// ChildModes is what the composer needs to know about the child's terminal
// state. Taken as a value rather than a *ptychild.Screen so the composition
// stays pure and a test can state a mode directly.
//
// Each mode carries whether it was OBSERVED. Absence of evidence is not
// evidence of absence: asserting a buffer state never witnessed would drop a
// child out of an alt screen it is really in — #196's shape, one field over.
type ChildModes struct {
	AltScreen         bool
	AltScreenObserved bool
}

// Repaint returns the takeover byte sequence.
//
// As SHIPPED it is two steps — clear only when there is something to draw or
// the caller asked for one, then the retained tail. The buffer assertion that
// used to lead is WITHDRAWN; see the body for why, and note that the ORDER is
// still the design when it returns: buffer state must go FIRST, before the
// clear, because `?1049`/`?1047` switch buffers and a paint belongs to
// whichever buffer was active when it was written, so asserting afterwards
// discards everything just painted. Three earlier versions got that wrong.
//
// Mouse state is deliberately NOT asserted here. couch re-asserts its own mouse
// mode on every paint and ptychild's replay feeds the child's own bytes back
// through the scanner; a third writer would be two authorities for one terminal
// mode, which is the thing #172 spent its rounds separating. Cursor-save is not
// asserted either — it cannot be: `\x1b7` saves the CURRENT cursor, and no
// sequence injects a previously-saved position.
func Repaint(modes ChildModes, replay []byte, intent RepaintIntent) []byte {
	out := make([]byte, 0, len(replay)+len(HomeAndClear)+len(LeaveAltScreen))

	// BUFFER ASSERTION TEMPORARILY WITHDRAWN (#209).
	//
	// `?1049h`/`?1049l` are not pure buffer switches: the `1049` pair SAVES and
	// RESTORES the cursor as part of switching. Emitting `?1049l` on every
	// switch therefore performs a cursor restore from a slot that, per this
	// repo's own probes/cursorsaveslots question, may ALIAS DECSC (`\x1b7`) —
	// which is exactly what pair's tab strip paints with. Observed live: typed
	// characters landed mid-screen after a repaint, because the strip's
	// save/restore pairing had been consumed.
	//
	// Mode 4 is therefore unfixed for now. The candidate is `?1047h`/`?1047l`,
	// which switch buffers WITHOUT touching the cursor — but that is exactly the
	// kind of terminal-behaviour assumption #209 already had to measure once
	// (cmd/probes/zellijrepaint), so it gets a probe before it ships, not a guess.
	if intent == RepaintClear || len(replay) > 0 {
		out = append(out, HomeAndClear...)
	}
	return append(out, replay...)
}

// RepaintFor is Repaint for a child: read the modes the composition needs, then
// compose. One entry point, because the read was copy-pasted at both consumers
// and NEITHER copy was pinned — `ChildModes` has no observable effect while the
// buffer assertion is withdrawn, so a test at a consumer could not tell a
// correct literal from the zero value (#209 BR-2, BR-12).
//
// Reading here rather than at each caller also means the modes are read at the
// moment the bytes are composed, not at whatever earlier point a caller
// happened to sample them.
//
// A nil child is not a child at all — couch's panel takes the screen over with
// its OWN surface — so it contributes no modes rather than a zero-valued
// assertion.
func RepaintFor(child *ptychild.Child, replay []byte, intent RepaintIntent) []byte {
	var modes ChildModes
	if child != nil {
		modes.AltScreen, modes.AltScreenObserved = child.RepaintModes()
	}
	return Repaint(modes, replay, intent)
}
