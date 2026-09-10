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

// repaint returns the takeover byte sequence: blank the screen, then draw
// whatever the child has retained.
//
// IT ALWAYS BLANKS, and that is the third and final answer to a question this
// issue got wrong twice (#209 C-1, family `absent-data-is-not-intent`).
//
// The middle version emitted NOTHING when the replay was empty, reasoning that
// "a stale frame beats a blank one while the child is asked to repaint". That
// reasoning has a hidden premise: it holds only when the stale frame belongs to
// the CHILD BEING REPAINTED. Enumerate the five takeover sites and not one is —
// couch's switchTo shows the previous thread or the panel, termcmd's
// switchRelative shows the previous tab, and the three deliberate-clear sites
// were carve-outs invented to escape the rule rather than instances of it. So
// the branch had zero correct callers, and its cost was worse than the blank it
// avoided: starting a thread from the panel left the PANEL's body on screen
// under the new thread's label, which is the misleading version of the same
// wrong.
//
// What actually answers "the ring could not tell us what to draw" is asking the
// CHILD, which is measured to work (cmd/probes/zellijrepaint). A blank frame
// for one settle while the authority redraws is honest; a foreign frame under
// the wrong label is not.
//
// The RepaintIntent enum went with the branch. It existed only to let three
// sites opt out of the keep-stale rule, so with the rule gone it distinguished
// nothing — an intent parameter that changes no bytes is a trap, because a site
// can pass the wrong one and nothing says so.
//
// Buffer state is NOT asserted, and that is a withdrawal rather than a
// simplification. `?1049h`/`?1049l` are not pure buffer switches: the 1049 pair
// SAVES and RESTORES the cursor, and the save slot is shared with DECSC —
// which is what pair's tab strip paints with (#199). Emitting `?1049l` on every
// switch consumed the strip's save/restore pairing, and typed characters landed
// mid-screen. The candidate is `?1047h`/`?1047l`, which switch buffers WITHOUT
// the cursor half, and it gets a probe before it ships rather than a second
// guess. When it returns, ORDER is the design: the buffer must be asserted
// BEFORE the clear, because a paint belongs to whichever buffer was active when
// it was written, so asserting afterwards discards everything just painted.
// Three earlier versions got that wrong.
//
// Mouse state is deliberately not asserted either. couch re-asserts its own
// mouse mode on every paint and ptychild's replay feeds the child's own bytes
// back through the scanner; a third writer would be two authorities for one
// terminal mode, which is the thing #172 spent its rounds separating.
// Cursor-save cannot be asserted at all: `\x1b7` saves the CURRENT cursor, and
// no sequence injects a previously-saved position.
func repaint(modes ChildModes, replay []byte) []byte {
	_ = modes // read by the ?1047 assertion when it lands; see above.
	out := make([]byte, 0, len(replay)+len(HomeAndClear))
	out = append(out, HomeAndClear...)
	return append(out, replay...)
}

// RepaintFor is repaint for a child: read the modes the composition needs, then
// compose. One entry point, because the read was copy-pasted at both consumers
// and NEITHER copy was pinned — ChildModes has no observable effect while the
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
func RepaintFor(child *ptychild.Child, replay []byte) []byte {
	var modes ChildModes
	if child != nil {
		modes.AltScreen, modes.AltScreenObserved = child.RepaintModes()
	}
	return repaint(modes, replay)
}
