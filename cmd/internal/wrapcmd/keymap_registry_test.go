package wrapcmd

import (
	"bytes"
	"github.com/xianxu/pair/cmd/internal/layoutcmd"
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
	"testing"
)

// TestTranslateChunk_AgyKeymap exercises the agy row through
// translateChunk so a typo in the registration table that happens to
// pass the registry test (e.g. swapped fields) also gets caught at
// the translation layer.
func TestTranslateChunk_AgyKeymap(t *testing.T) {
	f := newHarnessSessionFake(t, "agy", true)
	t.Cleanup(f.close)
	f.output(agyLiveComposerPaint())
	cases := []struct{ in, want []byte }{
		{[]byte("hi\r"), []byte("hi\n")},                                                 // Enter → newline
		{[]byte("hi\x1b\r"), []byte("hi\r")},                                             // Alt+Enter → send
		{[]byte("a\rb\x1b\r"), []byte("a\nb\r")},                                         // both, same chunk
		{[]byte("hi\x1b\x7f"), []byte("hi\x15")},                                         // Alt+Backspace → Ctrl+U
		{[]byte("\x1b[200~text\rmore\x1b[201~"), []byte("\x1b[200~text\rmore\x1b[201~")}, // paste untouched
	}
	for _, c := range cases {
		got, _, _ := f.proxy.translateChunk(c.in, false)
		if !bytes.Equal(got, c.want) {
			t.Errorf("in=%q: got %q, want %q", c.in, got, c.want)
		}
	}
}

// The agent pane's half of #216. Both the picker and the byte encoding live
// elsewhere, so without this a missing case in executeWorkbenchDecision would
// leave every other test green while the chord did nothing from this pane.
func TestAgentPaneDeliversTabChordsToTheRightTerminal(t *testing.T) {
	for _, test := range []struct {
		name  string
		chord workbenchshortcut.Chord
		want  workbenchshortcut.Chord
	}{
		{"previous", workbenchshortcut.ChordAltShiftLeft, workbenchshortcut.ChordAltLeft},
		{"next", workbenchshortcut.ChordAltShiftRight, workbenchshortcut.ChordAltRight},
	} {
		t.Run(test.name, func(t *testing.T) {
			original := switchTerminalTab
			t.Cleanup(func() { switchTerminalTab = original })
			var delivered []workbenchshortcut.Chord
			switchTerminalTab = func(_ layoutcmd.Runtime, chord workbenchshortcut.Chord) error {
				delivered = append(delivered, chord)
				return nil
			}

			decision, ok := workbenchshortcut.DecideGlobal(test.chord)
			if !ok {
				t.Fatalf("%v is not a global chord", test.chord)
			}
			p := &proxy{}
			if !p.executeWorkbenchDecision(decision) {
				t.Fatal("executeWorkbenchDecision did not handle the chord")
			}
			if len(delivered) != 1 || delivered[0] != test.want {
				t.Fatalf("delivered %v, want [%v]", delivered, test.want)
			}
		})
	}
}
