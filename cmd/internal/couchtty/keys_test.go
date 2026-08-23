package couchtty

import (
	"bytes"
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
