package wrapcmd

import (
	"bytes"
	"testing"
)

// TestEmitPlainCR_ClaudeComposerActiveRewritesToNewline pins Claude's remap
// while its composer is recognized. claudeProxy paints one.
func TestEmitPlainCR_ClaudeComposerActiveRewritesToNewline(t *testing.T) {
	p := claudeProxy()
	if got := p.emitPlainCR(nil); !bytes.Equal(got, []byte{'\\', '\r'}) {
		t.Fatalf("got %q, want Claude's backslash-CR remap while the composer is active", got)
	}
}

// TestEmitPlainCR_ClaudeComposerInactiveSendsBareCR pins the branch pair#138
// made reachable: before the flip Claude always remapped, so a screen with no
// recognizable composer now has to fall through to a bare CR.
func TestEmitPlainCR_ClaudeComposerInactiveSendsBareCR(t *testing.T) {
	f := newHarnessSessionFake(t, "claude", true)
	t.Cleanup(f.close)

	if got := f.proxy.emitPlainCR(nil); !bytes.Equal(got, []byte{'\r'}) {
		t.Fatalf("got %q, want bare CR without an active Claude composer", got)
	}
}

// TestEmitPlainCR_ClaudeComposerUnknownSendsBareCR covers the other decline
// path: a positively gated profile with no terminal snapshot at all.
func TestEmitPlainCR_ClaudeComposerUnknownSendsBareCR(t *testing.T) {
	p := proxyForHarness("claude")
	if p.terminal != nil {
		t.Fatal("expected a proxy with no terminal state")
	}
	if got := p.emitPlainCR(nil); !bytes.Equal(got, []byte{'\r'}) {
		t.Fatalf("got %q, want bare CR when the composer state is unknown", got)
	}
}
