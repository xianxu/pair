package ansi

import "testing"

func TestFrame(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want int
		st   Status
	}{
		{"CSI SGR", "\x1b[31mX", 5, Complete},
		{"CSI private mode", "\x1b[?1006h", 8, Complete},
		{"CSI no params", "\x1b[H", 3, Complete},
		{"OSC BEL", "\x1b]0;title\x07rest", 10, Complete},
		{"OSC ST", "\x1b]0;t\x1b\\rest", 7, Complete},
		{"charset designation", "\x1b(B", 3, Complete},
		{"two-byte escape", "\x1bM", 2, Complete},
		{"not an escape", "hello", 0, None},
		{"bare ESC at EOF", "\x1b", 0, Incomplete},
		{"incomplete CSI", "\x1b[31", 0, Incomplete},
		{"empty", "", 0, None},

		// Strict CSI: an out-of-range byte aborts rather than scanning on. This is
		// the anti-over-strip rule — over-stripping silently removes mouse mode,
		// Kitty encoding or the cursor shape, where a missed sequence is benign.
		{"CSI with out-of-range param byte", "\x1b[\x00A", 0, None},

		// NON-OBVIOUS, and the reason the oracle exists: `]` is 0x5D, inside the
		// two-byte class [0x5C-0x5F]. So an unterminated OSC is not "incomplete" —
		// the regex matched `\x1b]` as a two-byte escape and left the rest as text.
		{"unterminated OSC falls back to two-byte", "\x1b]0;title", 2, Complete},
		{"OSC with bare ESC falls back too", "\x1b]0;a\x1bZ\x07", 2, Complete},
	}
	for _, c := range cases {
		n, st := Frame([]byte(c.in))
		if n != c.want || st != c.st {
			t.Errorf("%s: Frame(%q) = (%d,%v), want (%d,%v)", c.name, c.in, n, st, c.want, c.st)
		}
	}
}

// TerminatorScan must NOT look at buf[1]: termcmd routes SS3 (`\x1bO…`) through it,
// and a dispatch on the introducer would frame "\x1bOX" as a two-byte escape and
// leak the X into a tab name (#128 PQ-5).
func TestTerminatorScanIsIntroducerIndependent(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"\x1b[31m", 5},
		{"\x1b[\x00A", 4}, // lenient: no range validation, unlike Frame
		{"\x1bOX", 3},     // SS3
		{"\x1bO@", 3},
		{"\x1b[31", -1}, // unterminated
	}
	for _, c := range cases {
		if got := TerminatorScan([]byte(c.in)); got != c.want {
			t.Errorf("TerminatorScan(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

// The one genuine strictness split between callers.
func TestOSCEndModes(t *testing.T) {
	withBareESC := []byte("\x1b]0;a\x1bZ\x07")
	if _, ok := OSCEnd(withBareESC, Strict); ok {
		t.Error("Strict must not scan past a bare ESC")
	}
	if n, ok := OSCEnd(withBareESC, Lenient); !ok || n != 8 {
		t.Errorf("Lenient OSCEnd = (%d,%v), want (8,true)", n, ok)
	}
	// Both modes agree on well-formed input.
	for _, m := range []Mode{Strict, Lenient} {
		if n, ok := OSCEnd([]byte("\x1b]0;t\x07"), m); !ok || n != 6 {
			t.Errorf("mode %v: OSCEnd = (%d,%v), want (6,true)", m, n, ok)
		}
	}
}

func TestStripRemovesSequencesKeepsText(t *testing.T) {
	if got := string(Strip([]byte("\x1b[31mred\x1b[0m done"))); got != "red done" {
		t.Errorf("Strip = %q", got)
	}
	// A trailing INCOMPLETE sequence is kept, not silently eaten — the caller's
	// tail-carry decides, and dropping it here would lose bytes at every chunk edge.
	if got := string(Strip([]byte("ok\x1b[3"))); got != "ok\x1b[3" {
		t.Errorf("Strip incomplete tail = %q, want it preserved", got)
	}
	// No ESC at all: returned unchanged, no allocation.
	in := []byte("plain text")
	if got := Strip(in); &got[0] != &in[0] {
		t.Error("Strip should not copy when there is nothing to strip")
	}
}
