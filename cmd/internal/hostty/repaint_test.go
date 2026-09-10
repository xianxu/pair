package hostty

import (
	"bytes"
	"testing"
)

// The buffer assertion is WITHDRAWN (#209), and this pins the withdrawal so it
// cannot be quietly re-added without the probe that would justify it.
//
// `?1049h`/`?1049l` save and restore the cursor as part of switching buffers.
// Emitting `?1049l` on every switch therefore restored the cursor from a slot
// that may alias DECSC (`\x1b7`) — which pair's tab strip paints with. Observed
// live: typed characters landed mid-screen after a repaint. Mode 4 stays unfixed
// until `?1047h`/`?1047l` (buffer switch WITHOUT cursor side effects) is
// measured the way probes/zellijrepaint measured the repaint assumption.
func TestRepaintEmitsNoCursorMovingBufferAssertion(t *testing.T) {
	for _, alt := range []bool{false, true} {
		for _, observed := range []bool{false, true} {
			got := Repaint(ChildModes{AltScreen: alt, AltScreenObserved: observed}, []byte("PAINT"), RepaintReplace)
			for _, forbidden := range []string{EnterAltScreen, LeaveAltScreen} {
				if bytes.Contains(got, []byte(forbidden)) {
					t.Fatalf("AltScreen=%v observed=%v: repaint = %q, which moves the cursor via the 1049 save slot",
						alt, observed, got)
				}
			}
		}
	}
}

// The two intents differ in exactly one case, and it is not inferable from the
// slice: `pair term` deliberately repaints with nil to blank a new tab before
// releasing its startup output, while a replace with nothing retained means the
// ring could not answer — where blanking is strictly worse than a stale frame.
func TestRepaintCarriesIntentRatherThanInferringItFromAnEmptyReplay(t *testing.T) {
	replace := Repaint(ChildModes{}, nil, RepaintReplace)
	if bytes.Contains(replace, []byte(HomeAndClear)) {
		t.Errorf("replace with nothing retained = %q, want no clear — a blank screen is worse than a stale one", replace)
	}
	if len(replace) != 0 {
		t.Errorf("replace with nothing retained = %q, want no output at all", replace)
	}

	deliberate := Repaint(ChildModes{}, nil, RepaintClear)
	if !bytes.Contains(deliberate, []byte(HomeAndClear)) {
		t.Errorf("deliberate clear = %q, want HomeAndClear", deliberate)
	}
}

func TestRepaintCarriesTheRetainedTailVerbatim(t *testing.T) {
	replay := []byte("\x1b[32mgreen\x1b[0m tail")
	got := Repaint(ChildModes{AltScreenObserved: true}, replay, RepaintReplace)
	if !bytes.HasSuffix(got, replay) {
		t.Fatalf("repaint = %q, want it to end with the tail verbatim", got)
	}
}
