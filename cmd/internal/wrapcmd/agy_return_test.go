package wrapcmd

import (
	"bytes"
	"testing"
)

func TestEmitPlainCR_AgyComposerActiveRewritesToNewline(t *testing.T) {
	f := newHarnessSessionFake(t, "agy", true)
	t.Cleanup(f.close)
	f.output("\x1b[10;1H──────────\x1b[11;1H> work\x1b[13;1H──────────\x1b[?25h\x1b[12;3H")
	if got := f.proxy.emitPlainCR(nil); !bytes.Equal(got, []byte{'\n'}) {
		t.Fatalf("got %q, want LF while Agy composer is active", got)
	}
}

func TestEmitPlainCR_AgyUnknownComposerSendsBareCR(t *testing.T) {
	f := newHarnessSessionFake(t, "agy", true)
	t.Cleanup(f.close)
	if got := f.proxy.emitPlainCR(nil); !bytes.Equal(got, []byte{'\r'}) {
		t.Fatalf("got %q, want bare CR for unknown Agy composer", got)
	}
}
