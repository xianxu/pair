package couchtty

import (
	"bytes"
	"strconv"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
)

func TestInterceptorAltXFraming(t *testing.T) {
	for _, sequence := range workbenchshortcut.ChordEncodings(workbenchshortcut.ChordAltX) {
		for split := 0; split <= len(sequence); split++ {
			var interceptor Interceptor
			first := append([]byte("before"), sequence[:split]...)
			if split == len(sequence) {
				first = append(first, []byte("after")...)
			}
			before, hit, rest := interceptor.FeedHit(first)
			if split < len(sequence) {
				if hit != HitNone {
					t.Fatalf("%q split %d fired early", sequence, split)
				}
				before2, hit2, rest2 := interceptor.FeedHit(append(sequence[split:], []byte("after")...))
				before = append(before, before2...)
				hit, rest = hit2, rest2
			}
			if hit != HitPark || string(before) != "before" || string(rest) != "after" {
				t.Fatalf("%q split %d = before %q hit %v rest %q", sequence, split, before, hit, rest)
			}
		}

		paste := append([]byte("\x1b[200~before"), sequence...)
		paste = append(paste, []byte("after\x1b[201~")...)
		var interceptor Interceptor
		before, hit, rest := interceptor.FeedHit(paste)
		if hit != HitNone || len(rest) != 0 || !bytes.Equal(before, paste) {
			t.Fatalf("pasted %q = before %q hit %v rest %q", sequence, before, hit, rest)
		}
	}
}

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

// A pty read boundary is not a key boundary. Every legal split of every
// sequence the interceptor recognises must make the same decision as a
// one-shot read; the split immediately after ESC is the one BR-42 exposed.
func TestInterceptorRecognisesEverySequenceAtEverySplit(t *testing.T) {
	for _, seq := range knownSequences {
		for split := 1; split < len(seq.bytes); split++ {
			t.Run(string(seq.bytes)+"/split-"+strconv.Itoa(split), func(t *testing.T) {
				var it Interceptor
				before, hit, rest := it.Feed(seq.bytes[:split])
				if len(before) != 0 || hit || len(rest) != 0 {
					t.Fatalf("first feed = (%q, %v, %q), want held", before, hit, rest)
				}
				before, hit, rest = it.Feed(seq.bytes[split:])
				if seq.kind.intercepts() {
					if len(before) != 0 || !hit || len(rest) != 0 {
						t.Fatalf("completed chord %q = (%q, %v, %q), want it consumed", seq.bytes, before, hit, rest)
					}
					return
				}
				if hit || string(before) != string(seq.bytes) || len(rest) != 0 {
					t.Fatalf("completed marker = (%q, %v, %q), want %q", before, hit, rest, seq.bytes)
				}
			})
		}
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

// A lone ESC keystroke must reach the child when the short sequence-ambiguity
// window expires. It cannot be forwarded immediately: that is also the first
// byte of every Kitty key and bracketed-paste marker.
//
// ESC is a prefix of both paste markers and of the CSI-u hotkey, so a naive
// "hold every real prefix" rule buffers it until the operator's NEXT keystroke
// -- and then delivers the two glued together, which a terminal reads as
// Alt+<key>. In practice that means ESC does nothing in nvim or claude until
// you press something else, and then does the wrong thing (M2 BR-22).
//
// The discriminator is that a keystroke arrives as its own read. A split escape
// sequence has bytes BEFORE the ESC in the same chunk; a pressed ESC does not.
func TestInterceptorFlushesALoneEscKeystroke(t *testing.T) {
	var it Interceptor
	before, hit, _ := it.Feed([]byte("\x1b"))
	if hit {
		t.Fatal("a lone ESC fired the hotkey")
	}
	if len(before) != 0 {
		t.Fatalf("a lone ESC was forwarded before ambiguity resolution: before=%q", before)
	}
	if got := it.Flush(); string(got) != "\x1b" {
		t.Fatalf("Flush = %q, want ESC", got)
	}
	if got := it.Flush(); len(got) != 0 {
		t.Fatalf("second Flush = %q, want empty", got)
	}
}

// ESC pressed twice in a row -- interrupt in claude, and a normal-mode escape
// hatch in nvim -- must deliver two ESCs.
func TestInterceptorForwardsRepeatedEscKeystrokes(t *testing.T) {
	var it Interceptor
	for i := 0; i < 3; i++ {
		before, _, _ := it.Feed([]byte("\x1b"))
		before = append(before, it.Flush()...)
		if string(before) != "\x1b" {
			t.Fatalf("ESC %d was not flushed: %q", i+1, before)
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

// The cases the first ESC fix still glued (M2 BR-22 round 2). The length of the
// CHUNK is not the discriminator; the length of the PARTIAL is.
func TestInterceptorFlushesABareTrailingEscWithoutGluingTheNextKey(t *testing.T) {
	cases := []struct {
		name  string
		first string
		then  string
		want  string // what the child must receive across both feeds
	}{
		{"ESC after other bytes", "abc\x1b", "i", "abc\x1bi"},
		{"two ESCs in one read", "\x1b\x1b", "", "\x1b\x1b"},
		{"ESC ending a long chunk", "hello world\x1b", "x", "hello world\x1bx"},
		{"lone ESC", "\x1b", "", "\x1b"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var it Interceptor
			var got []byte
			before, _, _ := it.Feed([]byte(c.first))
			got = append(got, before...)
			got = append(got, it.Flush()...)
			if c.then != "" {
				before, _, _ = it.Feed([]byte(c.then))
				got = append(got, before...)
			}
			if string(got) != c.want {
				t.Fatalf("child received %q, want %q — an ESC was held and glued", got, c.want)
			}
		})
	}
}

// Nothing may be left stranded in the hold buffer for a completed input.
func TestInterceptorHoldsNothingAfterACompleteChunk(t *testing.T) {
	var it Interceptor
	for _, in := range []string{"plain", "\x1b[32;5u"} {
		it.Feed([]byte(in))
		if len(it.held) != 0 {
			t.Fatalf("after %q the interceptor still holds %q", in, it.held)
		}
	}
}

// ctrl+backspace arrives in two encodings and BOTH have to be recognised. The
// legacy form is the bare byte 0x08, not an escape sequence, so it needs the
// same shape of branch as ctrl-space's NUL rather than a knownSequences row;
// the Kitty form is an ordinary exact string. Missing either gives a home key
// that works in one terminal mode and silently types a backspace in the other.
func TestInterceptorRecognisesCtrlBackspaceInBothEncodings(t *testing.T) {
	for _, tc := range []struct {
		name  string
		chord []byte
	}{
		{"legacy 0x08", []byte{0x08}},
		{"kitty CSI-u", []byte("\x1b[127;5u")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var it Interceptor
			in := append(append([]byte("before"), tc.chord...), []byte("after")...)
			before, hit, rest := it.FeedHit(in)
			if hit != HitPrevious {
				t.Fatalf("hit = %v, want HitPrevious", hit)
			}
			if string(before) != "before" || string(rest) != "after" {
				t.Fatalf("split = (%q, %q), want (\"before\", \"after\")", before, rest)
			}
		})
	}
}

// Plain backspace must stay plain: it is how the operator edits the switcher's
// filter, and stealing it would be far worse than never adding the home key.
func TestInterceptorLeavesPlainBackspaceAlone(t *testing.T) {
	var it Interceptor
	before, hit, rest := it.FeedHit([]byte{0x7f})
	if hit != HitNone || len(rest) != 0 || string(before) != "\x7f" {
		t.Fatalf("FeedHit(0x7f) = (%q, %v, %q), want the byte forwarded untouched", before, hit, rest)
	}
}

// A read boundary is not a keystroke boundary. The Kitty form can arrive split
// anywhere, and a naive implementation forwards its prefix to the child.
func TestInterceptorHoldsASplitCtrlBackspace(t *testing.T) {
	full := []byte("\x1b[127;5u")
	for cut := 1; cut < len(full); cut++ {
		var it Interceptor
		before, hit, _ := it.FeedHit(full[:cut])
		if hit != HitNone || len(before) != 0 {
			t.Fatalf("cut %d: prefix leaked (%q, %v)", cut, before, hit)
		}
		before, hit, rest := it.FeedHit(full[cut:])
		if hit != HitPrevious || len(before) != 0 || len(rest) != 0 {
			t.Fatalf("cut %d: resumed = (%q, %v, %q), want a clean HitPrevious", cut, before, hit, rest)
		}
	}
}

// Inside a bracketed paste every byte is content. A pasted 0x08 that silently
// switched actors would be data loss the operator could never trace back --
// the same reason ctrl-space suspends inside a paste.
func TestInterceptorIgnoresCtrlBackspaceInsideAPaste(t *testing.T) {
	for _, chord := range [][]byte{{0x08}, []byte("\x1b[127;5u")} {
		var it Interceptor
		in := append(append([]byte("\x1b[200~x"), chord...), []byte("y\x1b[201~")...)
		before, hit, rest := it.FeedHit(in)
		if hit != HitNone || len(rest) != 0 {
			t.Fatalf("chord %q inside a paste fired: hit=%v rest=%q", chord, hit, rest)
		}
		if !bytes.Contains(before, chord) {
			t.Fatalf("chord %q was eaten from paste content: %q", chord, before)
		}
	}
}

