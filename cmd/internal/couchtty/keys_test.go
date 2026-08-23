package couchtty

import (
	"bytes"
	"strings"
	"testing"
)

// The split point IS the contract: in `x<ctrl-space>y`, x belongs to the child
// being left and y to the one landed on. A concatenated buffer cannot say that,
// and would send y to the wrong child.
func TestInterceptorSplitsAroundTheHotkey(t *testing.T) {
	var it Interceptor
	before, hit, rest := it.Feed([]byte("x\x00y"))

	if !hit {
		t.Fatal("hit = false for a bare NUL")
	}
	if string(before) != "x" {
		t.Fatalf("before = %q, want %q", before, "x")
	}
	if string(rest) != "y" {
		t.Fatalf("rest = %q, want %q", rest, "y")
	}
}

// The caller loops on rest, so two hotkeys in one read fire twice with the
// middle segment routed to the intermediate focus.
func TestInterceptorFiresTwiceInOneChunk(t *testing.T) {
	var it Interceptor
	var segments []string
	hits := 0

	buf := []byte("a\x00b\x00c")
	for {
		before, hit, rest := it.Feed(buf)
		segments = append(segments, string(before))
		if !hit {
			break
		}
		hits++
		buf = rest
	}

	if hits != 2 {
		t.Fatalf("hits = %d, want 2", hits)
	}
	want := []string{"a", "b", "c"}
	for i := range want {
		if i >= len(segments) || segments[i] != want[i] {
			t.Fatalf("segments = %q, want %q", segments, want)
		}
	}
}

// A paste can carry arbitrary bytes. A pasted NUL that silently switches actors
// AND eats a byte is a data-loss bug the operator would never diagnose.
func TestInterceptorIgnoresNULInsideABracketedPaste(t *testing.T) {
	var it Interceptor
	in := []byte("\x1b[200~before\x00after\x1b[201~tail")

	before, hit, rest := it.Feed(in)
	if hit {
		t.Fatalf("a NUL inside a bracketed paste fired the hotkey (rest=%q)", rest)
	}
	if !bytes.Contains(before, []byte("before\x00after")) {
		t.Fatalf("the pasted NUL did not reach the child: %q", before)
	}
	if !bytes.Contains(before, []byte("tail")) {
		t.Fatalf("bytes after the paste were lost: %q", before)
	}
}

// After the paste ends, the hotkey works again -- the suspension must not latch.
func TestInterceptorResumesAfterAPaste(t *testing.T) {
	var it Interceptor
	if _, hit, _ := it.Feed([]byte("\x1b[200~x\x1b[201~")); hit {
		t.Fatal("paste content fired the hotkey")
	}
	if _, hit, _ := it.Feed([]byte("\x00")); !hit {
		t.Fatal("the hotkey did not fire after the paste closed")
	}
}

// A pty read boundary falls wherever the kernel puts it, including inside a
// six-byte paste marker.
func TestInterceptorHandlesAPasteMarkerSplitAcrossReads(t *testing.T) {
	var it Interceptor
	if _, hit, _ := it.Feed([]byte("\x1b[20")); hit {
		t.Fatal("a partial marker fired the hotkey")
	}
	if _, hit, _ := it.Feed([]byte("0~data\x00still-pasting")); hit {
		t.Fatal("the paste was not recognised across the read boundary")
	}
}

// Buffer only a REAL prefix. `\x1b[2~` is the Insert key, not a paste marker:
// holding it would swallow a keystroke, and the repo has a lesson saying so.
func TestInterceptorDoesNotSwallowASequenceThatMerelyLooksLikeAMarker(t *testing.T) {
	var it Interceptor
	before, hit, _ := it.Feed([]byte("\x1b[2~"))
	if hit {
		t.Fatal("Insert fired the hotkey")
	}
	if string(before) != "\x1b[2~" {
		t.Fatalf("before = %q, want the Insert key forwarded intact", before)
	}
	// And the hotkey must still work afterwards -- nothing latched.
	if _, hit, _ := it.Feed([]byte("\x00")); !hit {
		t.Fatal("the hotkey stopped working after a marker-shaped non-marker")
	}
}

// With no hotkey, the caller must have exactly one place to look.
func TestInterceptorWithNoHotkeyReturnsEverythingInBefore(t *testing.T) {
	var it Interceptor
	before, hit, rest := it.Feed([]byte("ordinary typing"))
	if hit {
		t.Fatal("hit = true with no NUL present")
	}
	if string(before) != "ordinary typing" {
		t.Fatalf("before = %q", before)
	}
	if len(rest) != 0 {
		t.Fatalf("rest = %q, want empty", rest)
	}
}

func FuzzInterceptorFeed(f *testing.F) {
	for _, s := range []string{
		"", "\x00", "x\x00y", "\x1b[200~\x00\x1b[201~", "\x1b[2~", "\x1b[20",
		"\x1b", "\x1b[201~", "\x00\x00", "\x1b[200~",
	} {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, in []byte) {
		var it Interceptor
		before, _, rest := it.Feed(in) // must not panic
		if len(before)+len(rest) > len(in)+6 {
			t.Fatalf("Feed grew the input: %d + %d from %d", len(before), len(rest), len(in))
		}
	})
}

// Reported from the M2 smoke: ctrl-space reached draft nvim instead of couch.
//
// zellij explicitly enables the Kitty keyboard protocol, so the terminal stops
// sending the legacy NUL for ctrl-space and sends CSI-u instead: `\x1b[32;5u`
// (space is 32, ctrl is modifier bitmask 4, encoded as 4+1). An interceptor
// that knows only 0x00 forwards it to the child, which is exactly what the
// operator saw. pair's own chord table carries both encodings for every chord
// (workbenchshortcut/shortcut.go:294-312) -- couch has to as well.
func TestInterceptorFiresOnTheKittyProtocolEncoding(t *testing.T) {
	var it Interceptor
	before, hit, rest := it.Feed([]byte("x\x1b[32;5uy"))

	if !hit {
		t.Fatal("the Kitty-protocol ctrl-space did not fire the hotkey")
	}
	if string(before) != "x" || string(rest) != "y" {
		t.Fatalf("split = (%q, %q), want (x, y)", before, rest)
	}
}

// Both encodings, since which one arrives depends on whether the child has
// enabled the protocol -- and that can change mid-session.
func TestInterceptorFiresOnBothEncodings(t *testing.T) {
	for _, seq := range []string{"\x00", "\x1b[32;5u"} {
		var it Interceptor
		if _, hit, _ := it.Feed([]byte(seq)); !hit {
			t.Fatalf("%q did not fire the hotkey", seq)
		}
	}
}

// couch must not eat the WORKBENCH's chords. Alt+j and friends are pair's, and
// they arrive in the same CSI-u shape.
func TestInterceptorForwardsOtherKittyChordsUntouched(t *testing.T) {
	for _, seq := range []string{
		"\x1b[106;3u", // Alt+j
		"\x1b[119;3u", // Alt+w
		"\x1b[32;3u",  // Alt+space, not ctrl
		"\x1b[32;2u",  // Shift+space
		"\x1b[33;5u",  // ctrl+!, adjacent codepoint
	} {
		var it Interceptor
		before, hit, _ := it.Feed([]byte(seq))
		if hit {
			t.Fatalf("%q fired couch's hotkey; it belongs to the child", seq)
		}
		if string(before) != seq {
			t.Fatalf("%q was mangled to %q", seq, before)
		}
	}
}

func TestInterceptorHandlesTheKittyHotkeySplitAcrossReads(t *testing.T) {
	var it Interceptor
	if _, hit, _ := it.Feed([]byte("\x1b[32;")); hit {
		t.Fatal("a partial CSI-u fired the hotkey")
	}
	before, hit, rest := it.Feed([]byte("5utail"))
	if !hit {
		t.Fatal("the hotkey did not fire once the sequence completed")
	}
	if len(before) != 0 || string(rest) != "tail" {
		t.Fatalf("split = (%q, %q), want ('', tail)", before, rest)
	}
}

// The paste suspension covers both encodings: a CSI-u ctrl-space inside pasted
// content is content.
func TestInterceptorIgnoresTheKittyHotkeyInsideAPaste(t *testing.T) {
	var it Interceptor
	before, hit, _ := it.Feed([]byte("\x1b[200~a\x1b[32;5ub\x1b[201~"))
	if hit {
		t.Fatal("a Kitty-protocol ctrl-space inside a paste fired the hotkey")
	}
	if !strings.Contains(string(before), "\x1b[32;5u") {
		t.Fatalf("the pasted sequence did not reach the child: %q", before)
	}
}

// A lone ESC keystroke must reach the child IMMEDIATELY.
//
// ESC is a prefix of both paste markers and of the CSI-u hotkey, so a naive
// "hold every real prefix" rule buffers it until the operator's NEXT keystroke
// -- and then delivers the two glued together, which a terminal reads as
// Alt+<key>. In practice that means ESC does nothing in nvim or claude until
// you press something else, and then does the wrong thing (M2 BR-22).
//
// The discriminator is that a keystroke arrives as its own read. A split escape
// sequence has bytes BEFORE the ESC in the same chunk; a pressed ESC does not.
func TestInterceptorDoesNotHoldALoneEscKeystroke(t *testing.T) {
	var it Interceptor
	before, hit, _ := it.Feed([]byte("\x1b"))
	if hit {
		t.Fatal("a lone ESC fired the hotkey")
	}
	if string(before) != "\x1b" {
		t.Fatalf("a lone ESC was held instead of forwarded: before=%q", before)
	}

	// And the following keystroke must arrive on its own, not glued to the ESC.
	before, _, _ = it.Feed([]byte("i"))
	if string(before) != "i" {
		t.Fatalf("the next keystroke was glued to the held ESC: %q", before)
	}
}

// ESC pressed twice in a row -- interrupt in claude, and a normal-mode escape
// hatch in nvim -- must deliver two ESCs.
func TestInterceptorForwardsRepeatedEscKeystrokes(t *testing.T) {
	var it Interceptor
	for i := 0; i < 3; i++ {
		before, _, _ := it.Feed([]byte("\x1b"))
		if string(before) != "\x1b" {
			t.Fatalf("ESC %d was not forwarded: %q", i+1, before)
		}
	}
}

// The other half: a genuine sequence split across reads is still recognised,
// because its ESC arrives with earlier bytes in the same chunk.
func TestInterceptorStillHoldsASplitSequenceAfterOtherBytes(t *testing.T) {
	var it Interceptor
	before, hit, _ := it.Feed([]byte("abc\x1b[32;"))
	if hit {
		t.Fatal("a partial hotkey fired early")
	}
	if string(before) != "abc" {
		t.Fatalf("before = %q, want the bytes ahead of the partial", before)
	}
	if _, hit, _ := it.Feed([]byte("5u")); !hit {
		t.Fatal("the split hotkey was not recognised once completed")
	}
}
