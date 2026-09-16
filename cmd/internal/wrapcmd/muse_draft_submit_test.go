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
			// Plain Enter's behaviour depends on composer state, but Alt must
			// never be mistaken for a plain newline. Plain is \n when active,
			// \r otherwise — Alt is always \r.
			outPlain, _, _ := f.proxy.translateChunk([]byte{'\r'}, false)
			wantPlain := []byte{'\r'}
			if name == "with composer" {
				wantPlain = []byte{'\n'}
			}
			if !bytes.Equal(outPlain, wantPlain) {
				t.Fatalf("plain Enter translated to %q, want %q", outPlain, wantPlain)
			}
		})
	}
}

// TestMuseAgentPaneReturn follows the pair convention for coding harnesses:
// when the Muse composer is positively recognised, plain Return inserts a
// newline (\n) so the user can compose multi-line prompts; otherwise it
// falls through as a bare CR (send). Alt+Return always sends. When a picker
// overlay is active, plain Return must also send (confirm), regardless of
// composer state.
func TestMuseAgentPaneReturn(t *testing.T) {
	f := newHarnessSessionFake(t, "muse", true)
	defer f.close()

	// Unknown composer → plain is send.
	if got := f.enter(); !bytes.Equal(got, []byte{'\r'}) {
		t.Fatalf("plain enter without composer = %q, want CR", got)
	}

	// Active composer → plain is newline.
	f.output("\x1b[7;1H\x1b[2m────\x1b[8;1H\x1b[22m⟩ hello\x1b[9;1H\x1b[2m────\x1b[?25h\x1b[8;8H")
	if !museComposerActive(f.proxy.terminal.Snapshot()) {
		t.Fatal("composer should be active")
	}
	if got := f.enter(); !bytes.Equal(got, []byte{'\n'}) {
		t.Fatalf("plain enter with composer = %q, want LF", got)
	}
	if got := f.altEnter(); !bytes.Equal(got, []byte{'\r'}) {
		t.Fatalf("alt enter with composer = %q, want CR", got)
	}

	// Overlay active → plain is send even though composer is still painted.
	f.output("Do you want to proceed?")
	if got := f.enter(); !bytes.Equal(got, []byte{'\r'}) {
		t.Fatalf("plain enter with overlay = %q, want CR", got)
	}
	// Overlay is one-shot: next plain returns to newline.
	if got := f.enter(); !bytes.Equal(got, []byte{'\n'}) {
		t.Fatalf("plain enter after overlay = %q, want LF", got)
	}
}

// TestMuseDraftAltEnterSubmission_InsidePaste verifies the draft's Alt+Enter
// still submits when Zellij coalesces the write-chars paste and the
// send-keys into one stdin chunk. Without the paste-aware Alt handling,
// the proxy would forward Alt as literal paste content and the draft would
// sit idle in the composer.
func TestMuseDraftAltEnterSubmission_InsidePaste(t *testing.T) {
	for _, seq := range [][]byte{[]byte("\x1b\r"), []byte("\x1b[13;3u")} {
		t.Run(string(seq), func(t *testing.T) {
			f := newHarnessSessionFake(t, "muse", true)
			defer f.close()
			f.output("\x1b[7;1H\x1b[2m────\x1b[8;1H\x1b[22m⟩ hello\x1b[9;1H\x1b[2m────\x1b[?25h\x1b[8;8H")
			// Simulate Zellij's write-chars body wrapped as bracketed paste
			// followed immediately by Alt+Enter in the same chunk.
			body := "\x1b[200~draft body\x1b[201~"
			in := append([]byte(body), seq...)
			out, leftover, inPaste := f.proxy.translateChunk(in, false)
			if len(leftover) != 0 || inPaste {
				t.Fatalf("leftover=%q paste=%v", leftover, inPaste)
			}
			want := append([]byte(body), '\r')
			if !bytes.Equal(out, want) {
				t.Fatalf("inside-paste Alt+Enter %q translated to %q, want %q", seq, out, want)
			}
			// Also cover the coalesced-before-end case where Alt arrives
			// before the paste close marker (chunked body streaming).
			in2 := []byte("\x1b[200~draft body")
			in2 = append(in2, seq...)
			in2 = append(in2, "\x1b[201~"...)
			out2, leftover2, paste2 := f.proxy.translateChunk(in2, false)
			if len(leftover2) != 0 || paste2 {
				t.Fatalf("inside-paste-before-end leftover=%q paste=%v", leftover2, paste2)
			}
			// Alt inside paste must still become a CR, not literal ESC CR.
			if !bytes.Contains(out2, []byte{'\r'}) {
				t.Fatalf("Alt inside paste before end not translated: %q", out2)
			}
			if bytes.Contains(out2, []byte("\x1b\r")) || bytes.Contains(out2, []byte("\x1b[13;3u")) {
				t.Fatalf("Alt inside paste leaked as literal: %q", out2)
			}
		})
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
