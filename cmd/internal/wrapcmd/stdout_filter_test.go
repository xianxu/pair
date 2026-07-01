package wrapcmd

import (
	"bytes"
	"testing"
)

func TestStdoutChunk_CodexStripsSynchronizedOutputMarkers(t *testing.T) {
	p := &proxy{agentBasename: "codex"}

	got := p.stdoutChunk([]byte("a\x1b[?2026hb\x1b[?2026lc"))
	if want := []byte("abc"); !bytes.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestStdoutChunk_CodexStripsFocusEventMode(t *testing.T) {
	p := &proxy{agentBasename: "codex"}

	got := p.stdoutChunk([]byte("a\x1b[?1004hb\x1b[?1004lc"))
	if want := []byte("abc"); !bytes.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestStdoutChunk_CodexStripsSplitSynchronizedOutputMarkers(t *testing.T) {
	p := &proxy{agentBasename: "codex"}

	got := p.stdoutChunk([]byte("a\x1b[?20"))
	if want := []byte("a"); !bytes.Equal(got, want) {
		t.Fatalf("first got %q, want %q", got, want)
	}
	got = p.stdoutChunk([]byte("26hb"))
	if want := []byte("b"); !bytes.Equal(got, want) {
		t.Fatalf("second got %q, want %q", got, want)
	}
}

func TestStdoutChunk_CodexKeepsKittyKeyboardProtocolByDefault(t *testing.T) {
	p := &proxy{agentBasename: "codex"}
	in := []byte("a\x1b[>4;0mb\x1b[>7uc\x1b[?ud")

	got := p.stdoutChunk(in)
	if !bytes.Equal(got, in) {
		t.Fatalf("got %q, want %q", got, in)
	}
}

func TestStdoutChunk_CodexFilterKKPStripsKittyKeyboardProtocol(t *testing.T) {
	p := &proxy{agentBasename: "codex", codexFilterKKP: true}

	got := p.stdoutChunk([]byte("a\x1b[>4;0mb\x1b[>7uc\x1b[?ud"))
	if want := []byte("abcd"); !bytes.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestStdoutChunk_CodexFilterKKPStripsSplitKittyKeyboardProtocol(t *testing.T) {
	p := &proxy{agentBasename: "codex", codexFilterKKP: true}

	got := p.stdoutChunk([]byte("a\x1b[>"))
	if want := []byte("a"); !bytes.Equal(got, want) {
		t.Fatalf("first got %q, want %q", got, want)
	}
	got = p.stdoutChunk([]byte("7ub"))
	if want := []byte("b"); !bytes.Equal(got, want) {
		t.Fatalf("second got %q, want %q", got, want)
	}
}

func TestStdoutChunk_NonCodexPassesSynchronizedOutputMarkers(t *testing.T) {
	p := &proxy{agentBasename: "claude"}
	in := []byte("a\x1b[?2026hb")

	got := p.stdoutChunk(in)
	if !bytes.Equal(got, in) {
		t.Fatalf("got %q, want %q", got, in)
	}
}

// PAIR_CODEX_SYNC_PASSTHROUGH (#68 A/B switch): when set, codex's DEC 2026/1004
// markers must reach zellij untouched so we can test whether the strip is what
// trips zellij's client-disconnect guard.
func TestStdoutChunk_CodexSyncPassthroughForwardsMarkers(t *testing.T) {
	p := &proxy{agentBasename: "codex", codexSyncPassthrough: true}
	in := []byte("a\x1b[?2026hb\x1b[?2026lc\x1b[?1004hd")

	got := p.stdoutChunk(in)
	if !bytes.Equal(got, in) {
		t.Fatalf("got %q, want %q (passthrough must forward markers untouched)", got, in)
	}
}

func TestStdoutChunk_CodexSyncPassthroughStillAllowsKKPDiagnosticFilter(t *testing.T) {
	p := &proxy{
		agentBasename:        "codex",
		codexSyncPassthrough: true,
		codexFilterKKP:       true,
	}
	in := []byte("a\x1b[?2026hb\x1b[>7uc")

	got := p.stdoutChunk(in)
	if want := []byte("a\x1b[?2026hbc"); !bytes.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}
