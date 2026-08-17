package wrapcmd

import (
	"bytes"
	"testing"
)

func TestEmitPlainCR_MuseComposerActiveRewritesToNewline(t *testing.T) {
	p := museProxyWithComposer(true)

	got := p.emitPlainCR(nil)
	if want := []byte{'\n'}; !bytes.Equal(got, want) {
		t.Fatalf("got %q, want LF while Muse composer is active", got)
	}
}

func TestEmitPlainCR_MuseComposerActiveWithEmptyPromptRewritesToNewline(t *testing.T) {
	tr := newMuseComposerTracker()
	tr.resize(38, 120)
	// Empty prompt case that smoke hit: 30;1H "› " empty
	tr.feed([]byte("\x1b[30;1H\x1b[38;2;90;160;255;49m\xe2\x9f\xa9 \x1b[?25h\x1b[30;3H"))
	p := &proxy{
		agentBasename: "muse",
		sendKM:        sendKeymapByAgent["muse"],
		museComposer:  tr,
	}
	got := p.emitPlainCR(nil)
	if want := []byte{'\n'}; !bytes.Equal(got, want) {
		t.Fatalf("got %q, want LF for empty Muse prompt", got)
	}
}

func TestEmitPlainCR_MuseComposerInactiveSendsBareCR(t *testing.T) {
	p := museProxyWithComposer(false)

	got := p.emitPlainCR(nil)
	if want := []byte{'\r'}; !bytes.Equal(got, want) {
		t.Fatalf("got %q, want bare CR without active Muse composer", got)
	}
}

func TestEmitPlainCR_MuseOverlayBeatsComposer(t *testing.T) {
	p := museProxyWithComposer(true)
	p.pickerActive.Store(true)

	got := p.emitPlainCR(nil)
	if want := []byte{'\r'}; !bytes.Equal(got, want) {
		t.Fatalf("got %q, want bare CR while overlay active", got)
	}
	if p.pickerActive.Load() {
		t.Fatal("pickerActive should clear after confirming overlay")
	}
}

func TestHandleChunk_MuseFeedsComposerTracker(t *testing.T) {
	p := &proxy{agentBasename: "muse"}
	p.ensureMuseComposer().resize(38, 120)
	rolling := []byte{}

	p.handleChunk([]byte(
		"\x1b[30;1H\x1b[38;2;90;160;255;49m\xe2\x9f\xa9 \x1b[?25h\x1b[30;3H",
	), &rolling)

	if !p.museComposerActive() {
		t.Fatalf("muse composer should be active after handleChunk feed")
	}
}

func TestEmitPlainCR_MuseUnknownComposerSendsBareCR(t *testing.T) {
	p := &proxy{
		agentBasename: "muse",
		sendKM:        sendKeymapByAgent["muse"],
	}
	got := p.emitPlainCR(nil)
	if want := []byte{'\r'}; !bytes.Equal(got, want) {
		t.Fatalf("got %q, want bare CR for unknown muse composer state", got)
	}
}

func museProxyWithComposer(active bool) *proxy {
	tr := newMuseComposerTracker()
	tr.resize(38, 120)
	if active {
		tr.feed([]byte(
			"\x1b[30;1H\x1b[38;2;90;160;255;49m\xe2\x9f\xa9 \x1b[?25h\x1b[30;3H",
		))
	}
	return &proxy{
		agentBasename: "muse",
		sendKM:        sendKeymapByAgent["muse"],
		museComposer:  tr,
	}
}
