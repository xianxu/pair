package wrapcmd

import (
	"bytes"
	"os"
	"testing"
)

func TestEmitPlainCR_MuseComposerActiveRewritesToNewline(t *testing.T) {
	f := newHarnessSessionFake(t, "muse", true)
	t.Cleanup(f.close)
	raw, err := os.ReadFile("testdata/tty/muse/0.1.0-R708.1/composer.raw")
	if err != nil {
		t.Fatal(err)
	}
	f.output(string(raw))
	if got := f.proxy.emitPlainCR(nil); !bytes.Equal(got, []byte{'\n'}) {
		t.Fatalf("got %q, want LF while Muse composer is active", got)
	}
}

func TestEmitPlainCR_MuseComposerInactiveSendsBareCR(t *testing.T) {
	f := newHarnessSessionFake(t, "muse", true)
	t.Cleanup(f.close)
	if got := f.proxy.emitPlainCR(nil); !bytes.Equal(got, []byte{'\r'}) {
		t.Fatalf("got %q, want bare CR without active Muse composer", got)
	}
}

func TestEmitPlainCR_MuseOverlayBeatsComposer(t *testing.T) {
	f := newHarnessSessionFake(t, "muse", true)
	t.Cleanup(f.close)
	f.proxy.pickerActive.Store(true)
	if got := f.proxy.emitPlainCR(nil); !bytes.Equal(got, []byte{'\r'}) {
		t.Fatalf("got %q, want bare CR while overlay active", got)
	}
	if f.proxy.pickerActive.Load() {
		t.Fatal("pickerActive should clear after confirming overlay")
	}
}

func TestEmitPlainCR_MuseUnknownComposerSendsBareCR(t *testing.T) {
	f := newHarnessSessionFake(t, "muse", true)
	t.Cleanup(f.close)
	if got := f.proxy.emitPlainCR(nil); !bytes.Equal(got, []byte{'\r'}) {
		t.Fatalf("got %q, want bare CR for unknown Muse composer state", got)
	}
}
