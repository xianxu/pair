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

func TestDecodeBareEscape(t *testing.T) {
	keys, _ := DecodePanelKeys([]byte("\x1b"))
	if len(keys) != 1 || keys[0].Kind != KeyEscape {
		t.Fatalf("a bare ESC decoded to %+v", keys)
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
