package wrapcmd

import (
	"bytes"
	"testing"
)

func TestEmitPlainCR_CodexComposerActiveRewritesToNewline(t *testing.T) {
	f := newHarnessSessionFake(t, "codex", true)
	t.Cleanup(f.close)
	f.output("\x1b[19;1H\x1b[48;2;57;57;57m\x1b[K" +
		"\x1b[20;1H\x1b[48;2;57;57;57m\x1b[K" +
		"\x1b[21;1H\x1b[48;2;57;57;57m\x1b[K" +
		"\x1b[?25h\x1b[20;3H")
	if got := f.proxy.emitPlainCR(nil); !bytes.Equal(got, []byte{'\n'}) {
		t.Fatalf("got %q, want LF while Codex composer is active", got)
	}
}

func TestEmitPlainCR_CodexComposerInactiveSendsBareCR(t *testing.T) {
	f := newHarnessSessionFake(t, "codex", true)
	t.Cleanup(f.close)
	if got := f.proxy.emitPlainCR(nil); !bytes.Equal(got, []byte{'\r'}) {
		t.Fatalf("got %q, want bare CR without active Codex composer", got)
	}
}

func TestEmitPlainCR_CodexOverlayBeatsComposer(t *testing.T) {
	f := newHarnessSessionFake(t, "codex", true)
	t.Cleanup(f.close)
	f.output("\x1b[19;1H\x1b[48;2;57;57;57m\x1b[K" +
		"\x1b[20;1H\x1b[48;2;57;57;57m\x1b[K" +
		"\x1b[21;1H\x1b[48;2;57;57;57m\x1b[K" +
		"\x1b[?25h\x1b[20;3H")
	f.proxy.pickerActive.Store(true)
	if got := f.proxy.emitPlainCR(nil); !bytes.Equal(got, []byte{'\r'}) {
		t.Fatalf("got %q, want bare CR while overlay active", got)
	}
	if f.proxy.pickerActive.Load() {
		t.Fatal("pickerActive should clear after confirming overlay")
	}
}

func TestEmitPlainCR_NonCodexKeepsExistingRemap(t *testing.T) {
	p := claudeProxy()
	if got := p.emitPlainCR(nil); !bytes.Equal(got, []byte{'\\', '\r'}) {
		t.Fatalf("got %q, want existing Claude newline remap", got)
	}
}
