package wrapcmd

import (
	"bytes"
	"testing"
)

func TestEmitPlainCR_CodexComposerActiveRewritesToNewline(t *testing.T) {
	p := codexProxyWithComposer(true)

	got := p.emitPlainCR(nil)
	if want := []byte{'\n'}; !bytes.Equal(got, want) {
		t.Fatalf("got %q, want LF while Codex composer is active", got)
	}
}

func TestEmitPlainCR_CodexComposerInactiveSendsBareCR(t *testing.T) {
	p := codexProxyWithComposer(false)

	got := p.emitPlainCR(nil)
	if want := []byte{'\r'}; !bytes.Equal(got, want) {
		t.Fatalf("got %q, want bare CR without active Codex composer", got)
	}
}

func TestEmitPlainCR_CodexOverlayBeatsComposer(t *testing.T) {
	p := codexProxyWithComposer(true)
	p.pickerActive.Store(true)

	got := p.emitPlainCR(nil)
	if want := []byte{'\r'}; !bytes.Equal(got, want) {
		t.Fatalf("got %q, want bare CR while overlay active", got)
	}
	if p.pickerActive.Load() {
		t.Fatal("pickerActive should clear after confirming overlay")
	}
}

func TestEmitPlainCR_NonCodexKeepsExistingRemap(t *testing.T) {
	p := claudeProxy()

	got := p.emitPlainCR(nil)
	if want := []byte{'\\', '\r'}; !bytes.Equal(got, want) {
		t.Fatalf("got %q, want existing claude newline remap", got)
	}
}

func TestHandleChunk_CodexFeedsComposerTracker(t *testing.T) {
	p := &proxy{agentBasename: "codex"}
	p.ensureCodexComposer().resize(38, 120)
	rolling := []byte{}

	p.handleChunk([]byte(
		"\x1b[35;1H\x1b[48;2;57;57;57m\x1b[K"+
			"\x1b[36;1H\x1b[48;2;57;57;57m\x1b[K"+
			"\x1b[37;1H\x1b[48;2;57;57;57m\x1b[K"+
			"\x1b[?25h\x1b[36;3H",
	), &rolling)

	if !p.codexComposerActive() {
		t.Fatalf("codex composer should be active after handleChunk feed")
	}
}

func codexProxyWithComposer(active bool) *proxy {
	tr := newCodexComposerTracker()
	tr.resize(38, 120)
	if active {
		tr.feed([]byte(
			"\x1b[35;1H\x1b[48;2;57;57;57m\x1b[K" +
				"\x1b[36;1H\x1b[48;2;57;57;57m\x1b[K" +
				"\x1b[37;1H\x1b[48;2;57;57;57m\x1b[K" +
				"\x1b[?25h\x1b[36;3H",
		))
	}
	return &proxy{
		agentBasename: "codex",
		sendKM:        sendKeymapByAgent["codex"],
		codexComposer: tr,
	}
}
