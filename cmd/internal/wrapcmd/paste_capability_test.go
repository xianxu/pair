package wrapcmd

import (
	"bytes"
	"testing"
)

// pair#211 frames every draft send and review poke as one bracketed paste. That
// must not narrow which agents pair can drive: a child that never enabled
// DECSET 2004 gets the body as typed input, exactly as before the framing.

func TestTerminalModelTracksBracketedPaste(t *testing.T) {
	m, err := newTerminalModel(80, 24)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	if m.BracketedPaste() {
		t.Fatal("a fresh child has not enabled bracketed paste")
	}
	if err := m.Feed([]byte(bracketedPasteOn)); err != nil {
		t.Fatal(err)
	}
	if !m.BracketedPaste() {
		t.Fatal("DECSET 2004 not tracked")
	}
	if err := m.Feed([]byte("\x1b[?2004l")); err != nil {
		t.Fatal(err)
	}
	if m.BracketedPaste() {
		t.Fatal("DECRST 2004 not tracked")
	}
}

func TestProfiledTranslatorHonorsChildPasteMode(t *testing.T) {
	framed := []byte("\x1b[200~a\rb\x1b[201~")
	for _, tc := range []struct {
		name      string
		startup   string
		want      string
		wantPaste bool
	}{
		// Without 2004 the CR is typed input again: Claude's composer newline.
		{"child without paste gets typed input", "", "a\\\rb", false},
		{"child with paste gets the paste verbatim", bracketedPasteOn, string(framed), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newHarnessSessionFake(t, "claude", true)
			defer f.close()
			f.output(tc.startup + claudeLiveComposerPaint())
			out, leftover, inPaste := f.proxy.translateChunk(framed, false)
			if string(out) != tc.want || len(leftover) != 0 || inPaste != tc.wantPaste {
				t.Fatalf("out=%q leftover=%q paste=%v; want %q", out, leftover, inPaste, tc.want)
			}
		})
	}
}

func TestPassThroughHonorsChildPasteMode(t *testing.T) {
	// An unprofiled agent: the pass-through path, which is where an agent
	// pair knows nothing about lives.
	m, err := newTerminalModel(80, 24)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	p := &proxy{agentBasename: "someagent", terminal: m}

	// A marker split across two reads is held back, then dropped whole.
	out, pending, _ := p.passThroughChunk([]byte("x\x1b[20"), false)
	if string(out) != "x" || string(pending) != "\x1b[20" {
		t.Fatalf("split marker: out=%q pending=%q", out, pending)
	}
	out, pending, inPaste := p.passThroughChunk(append(pending, []byte("0~hi\x1b[201~")...), false)
	if string(out) != "hi" || len(pending) != 0 || inPaste {
		t.Fatalf("child without paste: out=%q pending=%q paste=%v, want typed %q", out, pending, inPaste, "hi")
	}

	if err := m.Feed([]byte(bracketedPasteOn)); err != nil {
		t.Fatal(err)
	}
	framed := []byte("\x1b[200~hi\x1b[201~")
	if out, _, _ := p.passThroughChunk(framed, false); !bytes.Equal(out, framed) {
		t.Fatalf("child with paste: out=%q, want the paste verbatim", out)
	}
}

func TestStripPasteMarkers(t *testing.T) {
	for in, want := range map[string]string{
		"plain":                        "plain",
		"\x1b[200~a\x1b[201~":          "a",
		"\x1b[200~a\x1b[Ab\x1b[201~c":  "a\x1b[Abc", // other escapes are the text's
		"\x1b[201~\x1b[200~\x1b[200~x": "x",
	} {
		if got := string(stripPasteMarkers([]byte(in))); got != want {
			t.Errorf("stripPasteMarkers(%q) = %q, want %q", in, got, want)
		}
	}
}
