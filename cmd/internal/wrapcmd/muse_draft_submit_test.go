package wrapcmd

import (
	"bytes"
	"testing"
)

// TestMuseDraftAltEnterSubmission verifies that Alt+Enter (the draft's
// "send" chord) always reaches Muse as a bare CR, regardless of composer
// state. This is the user-visible path for `Alt+Return` from the nvim draft:
// it must not sit idle in the composer.
func TestMuseDraftAltEnterSubmission(t *testing.T) {
	for _, name := range []string{"without composer", "with composer"} {
		t.Run(name, func(t *testing.T) {
			f := newHarnessSessionFake(t, "muse", true)
			defer f.close()
			if name == "with composer" {
				// Paint a live Muse composer so the proxy's terminal is in the
				// "active" state. The draft's Alt+Enter must still be a send.
				f.output("\x1b[7;1H\x1b[2m────\x1b[8;1H\x1b[22m⟩ hello\x1b[9;1H\x1b[2m────\x1b[?25h\x1b[8;8H")
				if !museComposerActive(f.proxy.terminal.Snapshot()) {
					t.Fatal("composer should be active")
				}
			}
			// Legacy Alt+Enter (\x1b\r) and KKP Alt+Enter (\x1b[13;3u) must
			// both become a single CR for Muse.
			for _, seq := range [][]byte{[]byte("\x1b\r"), []byte("\x1b[13;3u")} {
				out, leftover, inPaste := f.proxy.translateChunk(seq, false)
				if len(leftover) != 0 || inPaste {
					t.Fatalf("leftover=%q paste=%v for %q", leftover, inPaste, seq)
				}
				if !bytes.Equal(out, []byte{'\r'}) {
					t.Fatalf("Alt+Enter %q translated to %q, want CR", seq, out)
				}
			}
			// Muse's composer uses Shift+Return for a newline; Alt+Return
			// remains the unconditional submit chord.
			outPlain, _, _ := f.proxy.translateChunk([]byte{'\r'}, false)
			wantPlain := []byte{'\r'}
			if name == "with composer" {
				wantPlain = []byte("\x1b[13;2u")
			}
			if !bytes.Equal(outPlain, wantPlain) {
				t.Fatalf("plain Enter translated to %q, want %q", outPlain, wantPlain)
			}
		})
	}
}

func TestMuseDraftAltEnterSubmissionInsidePaste(t *testing.T) {
	for _, seq := range [][]byte{[]byte("\x1b\r"), []byte("\x1b[13;3u")} {
		t.Run(string(seq), func(t *testing.T) {
			f := newHarnessSessionFake(t, "muse", true)
			defer f.close()
			input := append([]byte("\x1b[200~draft body"), seq...)
			input = append(input, "\x1b[201~"...)
			out, leftover, inPaste := f.proxy.translateChunk(input, false)
			want := []byte("\x1b[200~draft body\r\x1b[201~")
			if len(leftover) != 0 || inPaste || !bytes.Equal(out, want) {
				t.Fatalf("translated=%q leftover=%q paste=%v, want %q/no leftover/not paste", out, leftover, inPaste, want)
			}
		})
	}
}

// TestMuseAgentPaneReturn verifies that Muse uses Shift+Return for plain input
// inside its composer and bare CR for Alt+Return or picker confirmation.
func TestMuseAgentPaneReturn(t *testing.T) {
	f := newHarnessSessionFake(t, "muse", true)
	defer f.close()

	// Unknown composer → plain is send.
	if got := f.enter(); !bytes.Equal(got, []byte{'\r'}) {
		t.Fatalf("plain enter without composer = %q, want CR", got)
	}

	// Active composer → plain inserts a native Shift+Return newline.
	f.output("\x1b[7;1H\x1b[2m────\x1b[8;1H\x1b[22m⟩ hello\x1b[9;1H\x1b[2m────\x1b[?25h\x1b[8;8H")
	if !museComposerActive(f.proxy.terminal.Snapshot()) {
		t.Fatal("composer should be active")
	}
	if got := f.enter(); !bytes.Equal(got, []byte("\x1b[13;2u")) {
		t.Fatalf("plain enter with composer = %q, want Shift+Return", got)
	}
	if got := f.altEnter(); !bytes.Equal(got, []byte{'\r'}) {
		t.Fatalf("alt enter with composer = %q, want CR", got)
	}

	// Overlay active → plain is send even though composer is still painted.
	f.output("Do you want to proceed?")
	if got := f.enter(); !bytes.Equal(got, []byte{'\r'}) {
		t.Fatalf("plain enter with overlay = %q, want CR", got)
	}
	// Overlay is one-shot: next plain returns to composer newline.
	if got := f.enter(); !bytes.Equal(got, []byte("\x1b[13;2u")) {
		t.Fatalf("plain enter after overlay = %q, want Shift+Return", got)
	}
}

// TestMuseComposerActive_RelaxedPrompt ensures a Muse UI refresh that changes
// the prompt glyph (e.g. "❯" or ">" instead of "⟩") does not silently break
// the Return remap. The box shape remains the discriminator.
func TestMuseComposerActive_RelaxedPrompt(t *testing.T) {
	for _, glyph := range []string{"⟩", "›", "❯", ">", "!", "●", "▶", "▸"} {
		t.Run(glyph, func(t *testing.T) {
			model := newTerminalModelForTest(t, 80, 38)
			// Paint a minimal Muse-style box with the given glyph.
			paint := "\x1b[7;1H\x1b[2m────\x1b[8;1H\x1b[22m" + glyph + " hello\x1b[9;1H\x1b[2m────\x1b[?25h\x1b[8;8H"
			if err := model.Feed([]byte(paint)); err != nil {
				t.Fatal(err)
			}
			if !museComposerActive(model.Snapshot()) {
				t.Fatalf("prompt %q not recognised as active muse composer", glyph)
			}
		})
	}
}

// TestMuseComposerActive_RelaxedRuleFaint ensures a rule style change
// (non-faint "─") does not break recognition when both rules share the same
// faint state. A mismatched faint pair must still be rejected.
func TestMuseComposerActive_RelaxedRuleFaint(t *testing.T) {
	// Both rules non-faint → should still be active (relaxed).
	paintBothNonFaint := "\x1b[7;1H────\x1b[8;1H\x1b[22m⟩ hello\x1b[9;1H────\x1b[?25h\x1b[8;8H"
	model := newTerminalModelForTest(t, 80, 38)
	if err := model.Feed([]byte(paintBothNonFaint)); err != nil {
		t.Fatal(err)
	}
	if !museComposerActive(model.Snapshot()) {
		t.Fatal("both non-faint rules should still be recognised")
	}
	// Mismatched faint → must be rejected (prevents random chrome pairing).
	paintMismatched := "\x1b[7;1H\x1b[2m────\x1b[8;1H\x1b[22m⟩ hello\x1b[9;1H────\x1b[?25h\x1b[8;8H"
	model2 := newTerminalModelForTest(t, 80, 38)
	if err := model2.Feed([]byte(paintMismatched)); err != nil {
		t.Fatal(err)
	}
	if museComposerActive(model2.Snapshot()) {
		t.Fatal("mismatched faint rules should not be recognised")
	}
}
