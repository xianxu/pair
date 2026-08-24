package couchtty

import "testing"

// The bug this decoder exists for: an SGR mouse report's bytes after the ESC
// are ALL printable, so a panel that took printable bytes as typeahead had
// `[<;0;M[<;;M...` typed into its filter by a mouse move -- which then matched
// nothing and showed "(nothing running)" with no way back.
func TestDecodeDropsMouseReports(t *testing.T) {
	keys, held := DecodePanelKeys([]byte("\x1b[<0;12;4M\x1b[<0;12;4m"))
	if len(keys) != 0 {
		t.Fatalf("mouse reports produced %d keystrokes: %+v", len(keys), keys)
	}
	if len(held) != 0 {
		t.Fatalf("held = %q, want nothing", held)
	}
}

func TestDecodeArrowsInBothModes(t *testing.T) {
	for _, seq := range []string{"\x1b[A", "\x1bOA"} {
		keys, _ := DecodePanelKeys([]byte(seq))
		if len(keys) != 1 || keys[0].Kind != KeyUp {
			t.Fatalf("%q decoded to %+v, want one KeyUp", seq, keys)
		}
	}
	for _, seq := range []string{"\x1b[B", "\x1bOB"} {
		keys, _ := DecodePanelKeys([]byte(seq))
		if len(keys) != 1 || keys[0].Kind != KeyDown {
			t.Fatalf("%q decoded to %+v, want one KeyDown", seq, keys)
		}
	}
}

func TestDecodeHoldsBareEscapeAsAPossibleSequencePrefix(t *testing.T) {
	keys, held := DecodePanelKeys([]byte("\x1b"))
	if len(keys) != 0 || string(held) != "\x1b" {
		t.Fatalf("a bare ESC decoded to keys=%+v held=%q, want a held prefix", keys, held)
	}

	keys, held = DecodePanelKeys(append(held, []byte("[B")...))
	if len(keys) != 1 || keys[0].Kind != KeyDown || len(held) != 0 {
		t.Fatalf("split down arrow decoded to keys=%+v held=%q", keys, held)
	}
}

func TestDecodeTypingAndEditing(t *testing.T) {
	keys, _ := DecodePanelKeys([]byte("ab\x7f\r"))
	want := []PanelKeyKind{KeyRune, KeyRune, KeyBackspace, KeyEnter}
	if len(keys) != len(want) {
		t.Fatalf("decoded %+v", keys)
	}
	for i := range want {
		if keys[i].Kind != want[i] {
			t.Fatalf("key %d = %v, want %v", i, keys[i].Kind, want[i])
		}
	}
}

// A sequence split across reads must be carried, not decayed into runes --
// otherwise half a mouse report is typed in.
func TestDecodeCarriesAPartialSequence(t *testing.T) {
	keys, held := DecodePanelKeys([]byte("x\x1b[<0;12"))
	if len(keys) != 1 || keys[0].Kind != KeyRune || keys[0].Rune != 'x' {
		t.Fatalf("keys = %+v, want just the x", keys)
	}
	if string(held) != "\x1b[<0;12" {
		t.Fatalf("held = %q, want the partial sequence", held)
	}

	keys2, held2 := DecodePanelKeys(append(held, []byte(";4My")...))
	if len(held2) != 0 {
		t.Fatalf("held2 = %q", held2)
	}
	if len(keys2) != 1 || keys2[0].Rune != 'y' {
		t.Fatalf("keys2 = %+v, want just the y — the mouse report should be dropped", keys2)
	}
}

func FuzzDecodePanelKeys(f *testing.F) {
	for _, s := range []string{"", "\x1b", "\x1b[A", "\x1b[<0;1;1M", "abc", "\x1b[", "\x7f\r\n"} {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, in []byte) {
		keys, held := DecodePanelKeys(in) // must not panic
		if len(keys)+len(held) > len(in)+8 {
			t.Fatalf("decode grew the input: %d keys + %d held from %d", len(keys), len(held), len(in))
		}
	})
}

// Under the Kitty keyboard protocol -- which zellij enables, so it is what a
// real session leaves the terminal in -- keys arrive as CSI-u rather than as
// their legacy bytes. Escape is `\x1b[27u`, not `\x1b`.
//
// This is the SECOND time this class has bitten #146: ctrl-space had the same
// problem in M2 and the fix was applied only to that one key. The operator
// reported Escape doing nothing in the panel.
func TestDecodeKittyProtocolKeys(t *testing.T) {
	cases := []struct {
		name string
		seq  string
		want PanelKeyKind
	}{
		{"escape", "\x1b[27u", KeyEscape},
		{"escape with modifier", "\x1b[27;1u", KeyEscape},
		{"enter", "\x1b[13u", KeyEnter},
		{"enter with modifier", "\x1b[13;1u", KeyEnter},
		{"backspace", "\x1b[127u", KeyBackspace},
		{"up with modifier", "\x1b[1;1A", KeyUp},
		{"down with modifier", "\x1b[1;1B", KeyDown},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			keys, held := DecodePanelKeys([]byte(c.seq))
			if len(held) != 0 {
				t.Fatalf("held %q", held)
			}
			if len(keys) != 1 || keys[0].Kind != c.want {
				t.Fatalf("%q decoded to %+v, want one %v", c.seq, keys, c.want)
			}
		})
	}
}

// A printable key reported as CSI-u must still type. With the "report all keys"
// flag set, `a` arrives as `\x1b[97u`.
func TestDecodeKittyPrintableKeys(t *testing.T) {
	keys, _ := DecodePanelKeys([]byte("\x1b[97u\x1b[98;1u"))
	if len(keys) != 2 {
		t.Fatalf("decoded %+v", keys)
	}
	for i, want := range []byte{'a', 'b'} {
		if keys[i].Kind != KeyRune || keys[i].Rune != want {
			t.Fatalf("key %d = %+v, want the rune %q", i, keys[i], want)
		}
	}
}

// ctrl-space is couch's, and it is intercepted BEFORE the panel -- but if one
// ever reaches the decoder it must not be typed in as a rune.
func TestDecodeDoesNotTypeControlCodepoints(t *testing.T) {
	keys, _ := DecodePanelKeys([]byte("\x1b[32;5u"))
	for _, k := range keys {
		if k.Kind == KeyRune {
			t.Fatalf("a modified key was typed as the rune %q", k.Rune)
		}
	}
}
