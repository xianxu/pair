package main

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

func TestStdoutChunk_NonCodexPassesSynchronizedOutputMarkers(t *testing.T) {
	p := &proxy{agentBasename: "claude"}
	in := []byte("a\x1b[?2026hb")

	got := p.stdoutChunk(in)
	if !bytes.Equal(got, in) {
		t.Fatalf("got %q, want %q", got, in)
	}
}
