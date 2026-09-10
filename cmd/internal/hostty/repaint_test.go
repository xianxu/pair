package hostty

import (
	"bytes"
	"strings"
	"testing"
)

// The ORDER is the design (#209). Asserting the buffer after the paint discards
// it, because ?1049h/l switch buffers and the paint belongs to whichever buffer
// was active when it was written.
func TestRepaintAssertsTheBufferBeforeItPaints(t *testing.T) {
	for _, test := range []struct {
		name string
		alt  bool
		want string
	}{
		{"child is on the alternate buffer", true, EnterAltScreen},
		{"child is on the primary buffer", false, LeaveAltScreen},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := string(Repaint(ChildModes{AltScreen: test.alt, AltScreenObserved: true}, []byte("PAINT"), RepaintReplace))
			if !strings.HasPrefix(got, test.want) {
				t.Fatalf("repaint = %q, want it to open with %q", got, test.want)
			}
			buffer := strings.Index(got, test.want)
			clear := strings.Index(got, HomeAndClear)
			paint := strings.Index(got, "PAINT")
			if !(buffer < clear && clear < paint) {
				t.Fatalf("order buffer=%d clear=%d paint=%d, want buffer < clear < paint", buffer, clear, paint)
			}
		})
	}
}

// Absence of evidence is not evidence of absence. A Screen that never witnessed
// the child say anything about the buffer must not be used to assert one —
// #196's shape, one field over.
func TestRepaintAssertsNoBufferItNeverObserved(t *testing.T) {
	for _, alt := range []bool{false, true} {
		got := Repaint(ChildModes{AltScreen: alt, AltScreenObserved: false}, []byte("PAINT"), RepaintReplace)
		if bytes.Contains(got, []byte(EnterAltScreen)) || bytes.Contains(got, []byte(LeaveAltScreen)) {
			t.Fatalf("AltScreen=%v unobserved: repaint = %q, want no buffer assertion at all", alt, got)
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
