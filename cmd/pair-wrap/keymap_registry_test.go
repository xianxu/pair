package main

import (
	"bytes"
	"testing"
)

// TestSendKeymapByAgent_RegistrationTable pins the per-agent stdin
// rewrite table. Adding the row for a new agent or accidentally
// editing an existing one (typo in the byte literal, swapped fields)
// is the kind of change that's easy to miss in review — claude /
// codex / agy each have their own ergonomics expectations and
// the wrong bytes silently breaks Enter / Shift+Enter for that
// agent in production. Treat the table as a contract.
func TestSendKeymapByAgent_RegistrationTable(t *testing.T) {
	type row struct {
		plain, alt, altBS []byte
	}
	ctrlU := []byte{0x15} // Alt+Backspace → kill to line start (all agents)
	want := map[string]row{
		// claude reads `\<Enter>` as newline regardless of terminal
		// keyboard-protocol level — the documented portable path.
		"claude": {[]byte{'\\', '\r'}, []byte{'\r'}, ctrlU},
		// codex: plain Enter inserts newline (LF); Alt+Enter emits CR submit.
		"codex": {[]byte{'\n'}, []byte{'\r'}, ctrlU},
		// agy: plain Enter inserts newline; Alt+Enter emits CR submit.
		"agy": {[]byte{'\n'}, []byte{'\r'}, ctrlU},
	}
	if len(sendKeymapByAgent) != len(want) {
		t.Fatalf("sendKeymapByAgent has %d agents, want %d (%v)",
			len(sendKeymapByAgent), len(want), agentNames())
	}
	for agent, w := range want {
		got, ok := sendKeymapByAgent[agent]
		if !ok {
			t.Errorf("missing agent %q in sendKeymapByAgent", agent)
			continue
		}
		if !bytes.Equal(got.plainCR, w.plain) {
			t.Errorf("%s.plainCR: got %q, want %q", agent, got.plainCR, w.plain)
		}
		if !bytes.Equal(got.altCR, w.alt) {
			t.Errorf("%s.altCR: got %q, want %q", agent, got.altCR, w.alt)
		}
		if !bytes.Equal(got.altBS, w.altBS) {
			t.Errorf("%s.altBS: got %q, want %q", agent, got.altBS, w.altBS)
		}
	}
}

// TestTranslateChunk_AgyKeymap exercises the agy row through
// translateChunk so a typo in the registration table that happens to
// pass the registry test (e.g. swapped fields) also gets caught at
// the translation layer.
func TestTranslateChunk_AgyKeymap(t *testing.T) {
	p := &proxy{sendKM: sendKeymapByAgent["agy"]}
	cases := []struct{ in, want []byte }{
		{[]byte("hi\r"), []byte("hi\n")},                                                 // Enter → newline
		{[]byte("hi\x1b\r"), []byte("hi\r")},                                             // Alt+Enter → send
		{[]byte("a\rb\x1b\r"), []byte("a\nb\r")},                                         // both, same chunk
		{[]byte("hi\x1b\x7f"), []byte("hi\x15")},                                         // Alt+Backspace → Ctrl+U
		{[]byte("\x1b[200~text\rmore\x1b[201~"), []byte("\x1b[200~text\rmore\x1b[201~")}, // paste untouched
	}
	for _, c := range cases {
		got, _, _ := p.translateChunk(c.in, false)
		if !bytes.Equal(got, c.want) {
			t.Errorf("in=%q: got %q, want %q", c.in, got, c.want)
		}
	}
}

func agentNames() []string {
	var out []string
	for k := range sendKeymapByAgent {
		out = append(out, k)
	}
	return out
}
