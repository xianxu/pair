package hostty

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
// The ORDER is the design, and three earlier versions of it were wrong:
//
//  1. Buffer state FIRST, before the clear. `?1049h`/`?1049l` switch buffers,
//     so a paint belongs to whichever buffer was active when it was written —
//     asserting the buffer afterwards discards everything just painted.
//  2. Clear only when there is something to draw, or when the caller asked for
//     a clear outright.
//  3. The retained tail.
//
// Mouse state is deliberately NOT asserted here. couch re-asserts its own mouse
// mode on every paint and ptychild's replay feeds the child's own bytes back
// through the scanner; a third writer would be two authorities for one terminal
// mode, which is the thing #172 spent its rounds separating. Cursor-save is not
// asserted either — it cannot be: `\x1b7` saves the CURRENT cursor, and no
// sequence injects a previously-saved position.
func Repaint(modes ChildModes, replay []byte, intent RepaintIntent) []byte {
	out := make([]byte, 0, len(replay)+len(HomeAndClear)+len(LeaveAltScreen))

	if modes.AltScreenObserved {
		if modes.AltScreen {
			out = append(out, EnterAltScreen...)
		} else {
			out = append(out, LeaveAltScreen...)
		}
	}
	if intent == RepaintClear || len(replay) > 0 {
		out = append(out, HomeAndClear...)
	}
	return append(out, replay...)
}
