package couchtty

import (
	"strconv"
	"testing"
)

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
	for _, seq := range []string{"\x1b[D", "\x1bOD"} {
		keys, _ := DecodePanelKeys([]byte(seq))
		if len(keys) != 1 || keys[0].Kind != KeyLeft {
			t.Fatalf("%q decoded to %+v, want one KeyLeft", seq, keys)
		}
	}
	for _, seq := range []string{"\x1b[C", "\x1bOC"} {
		keys, _ := DecodePanelKeys([]byte(seq))
		if len(keys) != 1 || keys[0].Kind != KeyRight {
			t.Fatalf("%q decoded to %+v, want one KeyRight", seq, keys)
		}
	}
}

func TestDecodeRecognisedKeysAtEverySplit(t *testing.T) {
	cases := []struct {
		seq  string
		want PanelKeyKind
	}{
		{"\x1b[A", KeyUp},
		{"\x1b[B", KeyDown},
		{"\x1bOA", KeyUp},
		{"\x1bOB", KeyDown},
		{"\x1b[D", KeyLeft},
		{"\x1b[C", KeyRight},
		{"\x1bOD", KeyLeft},
		{"\x1bOC", KeyRight},
		{"\x1b[27u", KeyEscape},
		{"\x1b[13u", KeyEnter},
		{"\x1b[127u", KeyBackspace},
	}
	for _, c := range cases {
		for split := 1; split < len(c.seq); split++ {
			t.Run(strconv.Quote(c.seq)+"/split-"+strconv.Itoa(split), func(t *testing.T) {
				keys, held := DecodePanelKeys([]byte(c.seq[:split]))
				if len(keys) != 0 || string(held) != c.seq[:split] {
					t.Fatalf("first decode = keys=%+v held=%q, want held prefix %q", keys, held, c.seq[:split])
				}
				keys, held = DecodePanelKeys(append(held, c.seq[split:]...))
				if len(held) != 0 || len(keys) != 1 || keys[0].Kind != c.want {
					t.Fatalf("completed decode = keys=%+v held=%q, want one %v", keys, held, c.want)
				}
			})
		}
	}
}

func TestDecodedHorizontalArrowsDriveStartAgentSelection(t *testing.T) {
	for _, tc := range []struct {
		sequence string
		want     string
	}{
		{sequence: "\x1b[D", want: "codex"},
		{sequence: "\x1bOD", want: "codex"},
		{sequence: "\x1b[C", want: "muse"},
		{sequence: "\x1bOC", want: "muse"},
	} {
		t.Run(strconv.Quote(tc.sequence), func(t *testing.T) {
			state := NewMenuState(menuThreads(), menuAddress("couch-one"))
			state.Agents = []string{"codex", "claude", "muse"}
			state.RootAgent = "claude"
			state, _ = reduceKey(state, PanelKey{Kind: KeyCtrlSpace})
			state, _ = reduceKey(state, PanelKey{Kind: KeyDown})
			keys, held := DecodePanelKeys([]byte(tc.sequence))
			if len(held) != 0 || len(keys) != 1 {
				t.Fatalf("decoded keys=%+v held=%q", keys, held)
			}
			state, _ = reduceKey(state, keys[0])
			if got := state.CurrentFrame().Agent; got != tc.want {
				t.Fatalf("agent = %q, want %q", got, tc.want)
			}
		})
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
	for _, s := range []string{"", "\x1b", "\x1b[A", "\x1b[<0;1;1M", "abc", "\x1b[", "\x7f\r\n", "\t", "\x1b[9u", "\x1b[9;5u"} {
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
	for i, want := range []rune{'a', 'b'} {
		if keys[i].Kind != KeyRune || keys[i].Rune != want {
			t.Fatalf("key %d = %+v, want the rune %q", i, keys[i], want)
		}
	}
}

func TestRemoveLastRune(t *testing.T) {
	if got := removeLastRune("路径"); got != "路" {
		t.Fatalf("removeLastRune(路径) = %q, want 路", got)
	}
}

func TestDecodePanelKeysHoldsSplitUTF8Rune(t *testing.T) {
	encoded := []byte("路")
	for split := 1; split < len(encoded); split++ {
		keys, held := DecodePanelKeys(encoded[:split])
		if len(keys) != 0 || string(held) != string(encoded[:split]) {
			t.Fatalf("split %d first decode = keys=%+v held=%q, want held prefix", split, keys, held)
		}
		keys, held = DecodePanelKeys(append(held, encoded[split:]...))
		if len(held) != 0 || len(keys) != 1 || keys[0].Kind != KeyRune || keys[0].Rune != '路' {
			t.Fatalf("split %d completed decode = keys=%+v held=%q, want rune 路", split, keys, held)
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

func TestDecodePanelKeysTabAcrossEverySplit(t *testing.T) {
	for _, sequence := range []string{"\t", "\x1b[9u"} {
		for split := 0; split <= len(sequence); split++ {
			keys, held := DecodePanelKeys([]byte(sequence[:split]))
			keys2, held2 := DecodePanelKeys(append(held, []byte(sequence[split:])...))
			keys = append(keys, keys2...)
			if len(held2) != 0 || len(keys) != 1 || keys[0].Kind != KeyTab {
				t.Fatalf("sequence %q split %d = keys %+v held %q", sequence, split, keys, held2)
			}
		}
	}
}

func TestDecodePanelKeysTabRejectsModifiedCSIu(t *testing.T) {
	keys, held := DecodePanelKeys([]byte("\x1b[9;5u"))
	if len(keys) != 0 || len(held) != 0 {
		t.Fatalf("modified Tab became input: keys=%+v held=%q", keys, held)
	}
}

func TestDecodePanelKeysTabHasFailSafeZeroKind(t *testing.T) {
	if PanelKeyKind(0) != KeyUnknown || KeyRune == KeyUnknown {
		t.Fatalf("zero kind authorizes input: unknown=%v rune=%v", KeyUnknown, KeyRune)
	}
}

// The panel decoder computed `modified` and then ignored it for backspace, so
// ctrl+backspace decoded as a plain backspace -- a latent bug independent of
// interception, and the reason the home key would have worked everywhere except
// inside the switcher, which is where it is used most.
func TestDecodePanelKeysDistinguishesCtrlBackspaceFromBackspace(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   string
		want []PanelKey
	}{
		{"plain backspace stays backspace", "\x7f", []PanelKey{{Kind: KeyBackspace}}},
		{"legacy 0x08 stays backspace for the panel", "\x08", []PanelKey{{Kind: KeyBackspace}}},
		{"unmodified CSI-u backspace", "\x1b[127u", []PanelKey{{Kind: KeyBackspace}}},
		{"explicitly unmodified CSI-u backspace", "\x1b[127;1u", []PanelKey{{Kind: KeyBackspace}}},
		{"ctrl+backspace is NOT backspace", "\x1b[127;5u", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, rest := DecodePanelKeys([]byte(tc.in))
			if len(rest) != 0 {
				t.Fatalf("rest = %q, want none", rest)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("keys = %+v, want %+v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("keys = %+v, want %+v", got, tc.want)
				}
			}
		})
	}
}
