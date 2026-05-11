package main

import (
	"bytes"
	"testing"
)

func TestTranslateChunk(t *testing.T) {
	p := &proxy{sendKM: sendKeymap{
		plainCR: []byte{'\\', '\r'},
		altCR:   []byte{'\r'},
	}}

	tests := []struct {
		name      string
		in        []byte
		startPase bool
		wantOut   []byte
		wantHold  []byte
		wantPaste bool
	}{
		{
			name:    "plain text passes through",
			in:      []byte("hello world"),
			wantOut: []byte("hello world"),
		},
		{
			name:    "Enter becomes backslash-Enter",
			in:      []byte("hi\r"),
			wantOut: []byte("hi\\\r"),
		},
		{
			name:    "Alt+Enter becomes plain Enter",
			in:      []byte("hi\x1b\r"),
			wantOut: []byte("hi\r"),
		},
		{
			name:    "mixed: Enter and Alt+Enter in same chunk",
			in:      []byte("a\rb\x1b\rc\r"),
			wantOut: []byte("a\\\rb\rc\\\r"),
		},
		{
			name:      "bracketed paste preserves embedded \\r",
			in:        []byte("\x1b[200~line1\rline2\r\x1b[201~"),
			wantOut:   []byte("\x1b[200~line1\rline2\r\x1b[201~"),
			wantPaste: false, // ends out of paste mode
		},
		{
			name:      "Enter after paste end gets rewritten",
			in:        []byte("\x1b[200~x\r\x1b[201~\r"),
			wantOut:   []byte("\x1b[200~x\r\x1b[201~\\\r"),
			wantPaste: false,
		},
		{
			name:      "paste start, mid-paste chunk",
			in:        []byte("\x1b[200~pasted text\r"),
			wantOut:   []byte("\x1b[200~pasted text\r"),
			wantPaste: true,
		},
		{
			name:      "paste continues into chunk, ends",
			startPase: true,
			in:        []byte("more\rstuff\x1b[201~Enter\r"),
			wantOut:   []byte("more\rstuff\x1b[201~Enter\\\r"),
			wantPaste: false,
		},
		{
			name:     "trailing ESC alone is held back",
			in:       []byte("hi\x1b"),
			wantOut:  []byte("hi"),
			wantHold: []byte("\x1b"),
		},
		{
			name:     "trailing partial bpStart held back",
			in:       []byte("hi\x1b[20"),
			wantOut:  []byte("hi"),
			wantHold: []byte("\x1b[20"),
		},
		{
			name:      "trailing partial bpEnd inside paste held back",
			startPase: true,
			in:        []byte("data\x1b[20"),
			wantOut:   []byte("data"),
			wantHold:  []byte("\x1b[20"),
			wantPaste: true,
		},
		{
			name:    "ESC followed by non-CR non-[200 is passed through ESC",
			in:      []byte("hi\x1b[A"), // arrow up
			wantOut: []byte("hi\x1b[A"),
		},
		{
			name:    "KKP plain Enter becomes backslash-Enter",
			in:      []byte("hi\x1b[13u"),
			wantOut: []byte("hi\\\r"),
		},
		{
			name:    "KKP plain Enter (explicit no-modifier) becomes backslash-Enter",
			in:      []byte("hi\x1b[13;1u"),
			wantOut: []byte("hi\\\r"),
		},
		{
			name:    "KKP Alt+Enter becomes plain Enter",
			in:      []byte("hi\x1b[13;3u"),
			wantOut: []byte("hi\r"),
		},
		{
			name:    "mixed KKP and legacy in one chunk",
			in:      []byte("a\rb\x1b[13;3uc\x1b[13u"),
			wantOut: []byte("a\\\rb\rc\\\r"),
		},
		{
			name:    "KKP arrow key still passes through (\\x1b[A)",
			in:      []byte("a\x1b[Ab"),
			wantOut: []byte("a\x1b[Ab"),
		},
		{
			name:     "partial KKP Alt+Enter held back at chunk end",
			in:       []byte("hi\x1b[13;3"),
			wantOut:  []byte("hi"),
			wantHold: []byte("\x1b[13;3"),
		},
		{
			name:     "partial KKP plain Enter held back at chunk end",
			in:       []byte("hi\x1b[13"),
			wantOut:  []byte("hi"),
			wantHold: []byte("\x1b[13"),
		},
	}

	t.Run("codex keymap", func(t *testing.T) {
		px := &proxy{sendKM: sendKeymap{
			plainCR: []byte{'\n'},
			altCR:   []byte{'\r'},
		}}
		cases := []struct{ in, want []byte }{
			{[]byte("hi\r"), []byte("hi\n")},     // Enter → newline
			{[]byte("hi\x1b\r"), []byte("hi\r")}, // Alt+Enter → send
			{[]byte("a\rb\x1b\r"), []byte("a\nb\r")},
			{[]byte("\x1b[200~text\rmore\x1b[201~"), []byte("\x1b[200~text\rmore\x1b[201~")}, // paste untouched
		}
		for _, c := range cases {
			got, _, _ := px.translateChunk(c.in, false)
			if !bytes.Equal(got, c.want) {
				t.Errorf("in=%q: got %q, want %q", c.in, got, c.want)
			}
		}
	})

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotOut, gotHold, gotPaste := p.translateChunk(tc.in, tc.startPase)
			if !bytes.Equal(gotOut, tc.wantOut) {
				t.Errorf("out: got %q, want %q", gotOut, tc.wantOut)
			}
			if !bytes.Equal(gotHold, tc.wantHold) {
				t.Errorf("hold: got %q, want %q", gotHold, tc.wantHold)
			}
			if gotPaste != tc.wantPaste {
				t.Errorf("paste: got %v, want %v", gotPaste, tc.wantPaste)
			}
		})
	}
}
