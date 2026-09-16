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
	// The composer state and the plain-Return expectation it implies are table
	// fields, not decisions keyed off the subtest name: renaming a case must not
	// be able to silently flip which bytes the assertion demands.
	for _, tc := range []struct {
		name      string
		composer  bool
		wantPlain []byte
	}{
		{name: "without composer", composer: false, wantPlain: []byte{'\r'}},
		{name: "with composer", composer: true, wantPlain: []byte("\x1b[13;2u")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newHarnessSessionFake(t, "muse", true)
			defer f.close()
			if tc.composer {
				// Paint a live Muse composer so the proxy's terminal is in the
				// "active" state. The draft's Alt+Enter must still be a send.
				f.output(musePaintedComposer("⟩"))
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
			if !bytes.Equal(outPlain, tc.wantPlain) {
				t.Fatalf("plain Enter translated to %q, want %q", outPlain, tc.wantPlain)
			}
		})
	}
}

// TestMuseDraftBodyPasteStaysLiteral pins the non-behavior for the profile the
// regression was found on. Zellij wraps the draft's write-chars body in a
// bracketed paste, and Muse enables ?2004h — so anything the translator emits
// between the markers arrives as pasted text, including a CR. The translator
// must therefore forward an Alt+Enter chord inside that window verbatim rather
// than "helpfully" turning it into a submit that cannot be one (#266).
func TestMuseDraftBodyPasteStaysLiteral(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   []byte
	}{
		{name: "legacy chord", in: []byte("\x1b[200~ok\x1b\r\x1b[201~")},
		{name: "KKP chord", in: []byte("\x1b[200~ok\x1b[13;3u\x1b[201~")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newHarnessSessionFake(t, "muse", true)
			defer f.close()
			f.output(musePaintedComposer("⟩"))
			out, leftover, inPaste := f.proxy.translateChunk(tc.in, false)
			if !bytes.Equal(out, tc.in) {
				t.Fatalf("paste window rewritten: got %q, want %q", out, tc.in)
			}
			if len(leftover) != 0 || inPaste {
				t.Fatalf("leftover=%q paste=%v after a complete paste", leftover, inPaste)
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
	f.output(musePaintedComposer("⟩"))
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

// musePaintedComposer paints a minimal Muse-style composer box — two rule rows
// enclosing a prompt row, cursor inside — with the given prompt glyph.
func musePaintedComposer(glyph string) string {
	return "\x1b[7;1H\x1b[2m────\x1b[8;1H\x1b[22m" + glyph + " hello\x1b[9;1H\x1b[2m────\x1b[?25h\x1b[8;8H"
}

// TestMuseComposerActive_RelaxedPrompt ensures a Muse UI refresh that swaps the
// prompt glyph for another chevron does not silently break the Return remap.
// The case list is musePromptGlyphs itself, so a glyph admitted to the shared
// authority cannot arrive without coverage.
func TestMuseComposerActive_RelaxedPrompt(t *testing.T) {
	if len(musePromptGlyphs) == 0 {
		t.Fatal("musePromptGlyphs is empty")
	}
	for glyph := range musePromptGlyphs {
		t.Run(glyph, func(t *testing.T) {
			model := newTerminalModelForTest(t, 80, 38)
			if err := model.Feed([]byte(musePaintedComposer(glyph))); err != nil {
				t.Fatal(err)
			}
			if !museComposerActive(model.Snapshot()) {
				t.Fatalf("prompt %q not recognised as active muse composer", glyph)
			}
		})
	}
}

// TestMuseComposerActive_RejectsSelectionMarkers pins the glyphs deliberately
// kept OUT of musePromptGlyphs (#266 close BR-3). These are how TUIs mark a
// highlighted menu row, so admitting them would let the positive gate call a
// picker a composer — and for Muse that means plain Return inserts a newline
// where the picker wanted a confirmation.
func TestMuseComposerActive_RejectsSelectionMarkers(t *testing.T) {
	for _, glyph := range []string{"!", "●", "▶", "▸", "◆", "*"} {
		t.Run(glyph, func(t *testing.T) {
			if musePromptGlyphs[glyph] {
				t.Fatalf("%q is a selection marker and must not be an admitted Muse prompt glyph", glyph)
			}
			model := newTerminalModelForTest(t, 80, 38)
			if err := model.Feed([]byte(musePaintedComposer(glyph))); err != nil {
				t.Fatal(err)
			}
			if museComposerActive(model.Snapshot()) {
				t.Fatalf("selection marker %q recognised as a muse composer", glyph)
			}
		})
	}
}

// TestMusePromptAuthorityIsShared pins the invariant BR-4 names: the Return
// remap's gate and the orientation auto-submit gate must admit exactly the same
// prompt glyphs. A glyph accepted by one and refused by the other is a state
// where Return inserts a newline into a composer orientation will not submit
// into — the drift that put the same literal list in two files twice.
func TestMusePromptAuthorityIsShared(t *testing.T) {
	for _, glyph := range []string{"⟩", "›", "❯", ">", "!", "●", "▶", "▸", "◆", "x", "─"} {
		if got, want := orientationPromptOK("muse", glyph), musePromptGlyphs[glyph]; got != want {
			t.Errorf("orientationPromptOK(muse, %q) = %t, musePromptGlyphs = %t", glyph, got, want)
		}
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
