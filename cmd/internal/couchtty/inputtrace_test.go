package couchtty

import (
	"slices"
	"testing"

	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
)

// The probe's whole job is to let an operator compare what the terminal sent
// against what the chord table watches for, so it must render a chord the same
// way the table spells it.
func TestRenderInputBytesSpellsChordsLikeTheTable(t *testing.T) {
	for _, tc := range []struct {
		name  string
		chord workbenchshortcut.Chord
		want  string
	}{
		{"alt+n", workbenchshortcut.ChordAltN, `\x1b[110;3u`},
		{"ctrl+alt+n", workbenchshortcut.ChordCtrlAltN, `\x1b[110;7u`},
		{"alt+shift+n", workbenchshortcut.ChordAltShiftN, `\x1b[78;4u`},
		{"alt+x kitty", workbenchshortcut.ChordAltX, `\x1b[120;3u`},
		// alt+x also has a LEGACY encoding, and alt+n does not. That asymmetry
		// is the whole reason this probe exists: with the Kitty protocol off,
		// alt+x still reaches couch and alt+n cannot.
		{"alt+x legacy", workbenchshortcut.ChordAltX, `\x1bx`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var rendered []string
			for _, encoding := range workbenchshortcut.ChordEncodings(tc.chord) {
				rendered = append(rendered, renderInputBytes(encoding))
			}
			if !slices.Contains(rendered, tc.want) {
				t.Errorf("renderings %q do not include %q", rendered, tc.want)
			}
		})
	}
}

func TestRenderInputBytesKeepsTextReadableAndEscapesTheRest(t *testing.T) {
	if got, want := renderInputBytes([]byte("hi there")), "hi there"; got != want {
		t.Errorf("printable text: got %q, want %q", got, want)
	}
	// ctrl-space's legacy encoding is NUL, the one byte an operator is most
	// likely to be hunting for and the one a naive renderer drops silently.
	if got, want := renderInputBytes([]byte{0x00}), `\x00`; got != want {
		t.Errorf("NUL: got %q, want %q", got, want)
	}
	if got, want := renderInputBytes([]byte{0x08, 0x7f}), `\x08\x7f`; got != want {
		t.Errorf("control bytes: got %q, want %q", got, want)
	}
}

// A nil tracer is the OFF state, and it is the state every operator runs in.
func TestNilInputTracerRecordsWithoutPanicking(t *testing.T) {
	var tracer *inputTracer
	tracer.record([]byte("\x1b[110;3u"))
}
