package hostty

import (
	"bytes"
	"testing"

	"github.com/xianxu/pair/cmd/internal/ptychild"
)

// The buffer assertion is WITHDRAWN (#209), and this pins the withdrawal so it
// cannot be quietly re-added without the probe that would justify it.
//
// `?1049h`/`?1049l` save and restore the cursor as part of switching buffers.
// Emitting `?1049l` on every switch therefore restored the cursor from a slot
// that may alias DECSC (`\x1b7`) — which pair's tab strip paints with. Observed
// live: typed characters landed mid-screen after a repaint. Mode 4 stays unfixed
// until `?1047h`/`?1047l` (buffer switch WITHOUT cursor side effects) is
// measured the way cmd/probes/zellijrepaint measured the repaint assumption.
func TestRepaintEmitsNoCursorMovingBufferAssertion(t *testing.T) {
	for _, alt := range []bool{false, true} {
		for _, observed := range []bool{false, true} {
			got := repaint(ChildModes{AltScreen: alt, AltScreenObserved: observed}, []byte("PAINT"))
			for _, forbidden := range []string{enterAltScreen, LeaveAltScreen} {
				if bytes.Contains(got, []byte(forbidden)) {
					t.Fatalf("AltScreen=%v observed=%v: repaint = %q, which moves the cursor via the 1049 save slot",
						alt, observed, got)
				}
			}
		}
	}
}

// A TAKEOVER ALWAYS BLANKS, whatever it has to draw (#209 C-1).
//
// The version this replaces emitted nothing for an empty replay, on the theory
// that a stale frame beats a blank one. The premise was that the stale frame
// belongs to the child being repainted, and not one of the five takeover sites
// is that case — so what it actually preserved was the PREVIOUS thread's screen,
// or the panel's own body, under the new thread's label. A blank frame for one
// settle while the child redraws is honest; a foreign frame is not.
//
// This is the pin, because the argument for the other behaviour was persuasive
// enough to ship twice.
func TestATakeoverAlwaysBlanksEvenWithNothingToDraw(t *testing.T) {
	for _, tt := range []struct {
		name   string
		replay []byte
	}{
		{"nothing retained", nil},
		{"empty rather than nil", []byte{}},
		{"a retained tail", []byte("frame")},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := repaint(ChildModes{}, tt.replay)
			if !bytes.HasPrefix(got, []byte(HomeAndClear)) {
				t.Fatalf("repaint = %q, want it to begin by blanking — the frame on screen "+
					"belongs to whatever was there before, not to this child", got)
			}
			if !bytes.HasSuffix(got, tt.replay) {
				t.Fatalf("repaint = %q, want it to end with the tail verbatim", got)
			}
		})
	}
}

// The wiring the boundary review found unpinned (#209 BR-2): both consumers
// used to build a ChildModes literal from the child themselves, and neither
// literal was defended — replacing either with the zero value left both suites
// green, because ChildModes has no observable effect while the buffer assertion
// is withdrawn. RepaintFor is now the one read, so it is pinned once, HERE,
// against the values rather than against the emitted bytes.
func TestRepaintForReadsTheChildsObservedModesRatherThanAssuming(t *testing.T) {
	// A child that entered the alt screen: observed AND set.
	entered := ptychild.NewFakeChild([]byte("\x1b[?1049hfull-screen"))
	if alt, observed := entered.RepaintModes(); !alt || !observed {
		t.Fatalf("RepaintModes() = (%v, %v), want (true, true) — RepaintFor composes from these", alt, observed)
	}

	// A child that has said nothing about the buffer: NOT observed, so a
	// composition must not assert either way (#196's shape).
	quiet := ptychild.NewFakeChild([]byte("just text"))
	if alt, observed := quiet.RepaintModes(); alt || observed {
		t.Fatalf("RepaintModes() = (%v, %v), want (false, false) for a child that never mentioned the buffer", alt, observed)
	}

	// And a nil child contributes no modes rather than panicking or asserting
	// the zero value as fact — couch's panel takes the screen over with its own
	// surface, and there is no child behind it.
	if got := RepaintFor(nil, []byte("panel")); !bytes.HasSuffix(got, []byte("panel")) {
		t.Fatalf("RepaintFor(nil, ...) = %q, want the body composed", got)
	}
}

// The GOLDEN both consumers are held to (#209 BR-9). The plan promised a
// differential row — "couchtty and termcmd produce byte-identical repaint
// output for the same child state" — and asserting that directly needs a test
// that can drive both consoles, which nothing in this tree is placed to do.
//
// So the differential closes TRANSITIVELY and each leg is a real assertion
// rather than a restatement: this table fixes the exact bytes RepaintFor emits,
// and each console's own test asserts that what it WROTE to its terminal
// equals RepaintFor's output for that child and replay —
// couchtty's TestSwitchWritesExactlyTheComposedRepaint and termcmd's
// TestTakeoverWritesExactlyTheComposedRepaint. If either console adds, drops or
// reorders a byte of its own, that console's test fails; if the composition
// changes, this one does.
func TestRepaintForComposesExactlyHomeAndClearPlusTheTail(t *testing.T) {
	replay := []byte("\x1b[32mretained tail\x1b[0m")
	for _, tt := range []struct {
		name   string
		seed   string
		replay []byte
		want   []byte
	}{
		{"alt-screen child, tail retained", "\x1b[?1049hfull-screen", replay,
			append([]byte(HomeAndClear), replay...)},
		{"primary-screen child, tail retained", "on the primary screen", replay,
			append([]byte(HomeAndClear), replay...)},
		{"child that said nothing, tail retained", "", replay,
			append([]byte(HomeAndClear), replay...)},
		{"nothing retained still blanks", "\x1b[?1049hfull-screen", nil,
			[]byte(HomeAndClear)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := RepaintFor(ptychild.NewFakeChild([]byte(tt.seed)), tt.replay)
			if !bytes.Equal(got, tt.want) {
				t.Fatalf("RepaintFor = %q, want %q", got, tt.want)
			}
		})
	}
}
