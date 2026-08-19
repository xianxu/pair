package wrapcmd

import (
	"bytes"
	"testing"
)

func TestEmitPlainCR_AgyComposerActiveRewritesToNewline(t *testing.T) {
	f := newHarnessSessionFake(t, "agy", true)
	t.Cleanup(f.close)
	f.output(agyLiveComposerPaint())
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
