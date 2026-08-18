package wrapcmd

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

type harnessSessionFake struct {
	t       *testing.T
	proxy   *proxy
	rolling []byte
}

func newHarnessSessionFake(t *testing.T, harness string, remap bool) *harnessSessionFake {
	t.Helper()
	p := &proxy{agentBasename: harness}
	if err := p.configureHarnessTTY(remap, 80, 38); err != nil {
		t.Fatalf("configure harness TTY: %v", err)
	}
	return &harnessSessionFake{t: t, proxy: p}
}

func (f *harnessSessionFake) output(raw string) {
	f.t.Helper()
	f.proxy.handleChunk([]byte(raw), &f.rolling)
}

func (f *harnessSessionFake) resize(cols, rows int) {
	f.t.Helper()
	if err := f.proxy.resizeTerminal(cols, rows); err != nil {
		f.t.Fatalf("resize terminal: %v", err)
	}
}

func (f *harnessSessionFake) enter() []byte {
	f.t.Helper()
	out, leftover, paste := f.proxy.translateChunk([]byte{'\r'}, false)
	if len(leftover) != 0 || paste {
		f.t.Fatalf("plain Enter left state: leftover=%q paste=%t", leftover, paste)
	}
	return out
}

func (f *harnessSessionFake) altEnter() []byte {
	f.t.Helper()
	out, leftover, paste := f.proxy.translateChunk([]byte("\x1b\r"), false)
	if len(leftover) != 0 || paste {
		f.t.Fatalf("Alt+Enter left state: leftover=%q paste=%t", leftover, paste)
	}
	return out
}

func (f *harnessSessionFake) close() {
	f.t.Helper()
	if err := f.proxy.closeTerminal(); err != nil {
		f.t.Fatalf("close terminal: %v", err)
	}
}

func TestHarnessTTYIntegration_ProfileSelectionAndTerminalLifecycle(t *testing.T) {
	f := newHarnessSessionFake(t, "agy", true)
	if f.proxy.ttyProfile == nil || f.proxy.terminal == nil {
		t.Fatal("positive-gated Agy profile must own a terminal")
	}
	f.output("\x1b[10;1H──────────\x1b[11;1H> work\x1b[13;1H──────────\x1b[?25h\x1b[12;3H")
	if got := f.proxy.terminal.Snapshot(); got.Width != 80 || got.Height != 38 || !agyComposerActive(got) {
		t.Fatalf("startup snapshot = %dx%d active=%t, want 80x38 active", got.Width, got.Height, agyComposerActive(got))
	}
	f.resize(60, 30)
	if got := f.proxy.terminal.Snapshot(); got.Width != 60 || got.Height != 30 {
		t.Fatalf("resized snapshot = %dx%d, want 60x30", got.Width, got.Height)
	}
	f.close()
	if err := f.proxy.terminal.Feed([]byte("late")); err != io.ErrClosedPipe {
		t.Fatalf("feed after close = %v, want io.ErrClosedPipe", err)
	}
}

func TestHarnessTTYIntegration_StatefulReturnRouting(t *testing.T) {
	f := newHarnessSessionFake(t, "agy", true)
	t.Cleanup(f.close)

	if got := f.enter(); !bytes.Equal(got, []byte{'\r'}) {
		t.Fatalf("unknown startup Enter = %q, want bare CR", got)
	}
	f.output("\x1b[10;1H──────────\x1b[11;1H> work\x1b[13;1H──────────\x1b[?25h\x1b[12;3H")
	if got := f.enter(); !bytes.Equal(got, []byte{'\n'}) {
		t.Fatalf("composer Enter = %q, want multiline LF", got)
	}
	f.output("\x1b[?25lbusy")
	if got := f.enter(); !bytes.Equal(got, []byte{'\r'}) {
		t.Fatalf("hidden busy Enter = %q, want bare CR", got)
	}
	f.output("Do you want to proceed?")
	if got := f.enter(); !bytes.Equal(got, []byte{'\r'}) || f.proxy.pickerActive.Load() {
		t.Fatalf("overlay Enter = %q active=%t, want bare CR and clear", got, f.proxy.pickerActive.Load())
	}
	f.output("\x1bc")
	if got := f.enter(); !bytes.Equal(got, []byte{'\r'}) {
		t.Fatalf("reset Enter = %q, want bare CR", got)
	}
	if got := f.altEnter(); !bytes.Equal(got, []byte{'\r'}) {
		t.Fatalf("Alt+Enter = %q, want unconditional submit CR", got)
	}
}

func TestHarnessTTYIntegration_DisabledAndUnknownPassThrough(t *testing.T) {
	for _, tc := range []struct {
		name    string
		harness string
		remap   bool
	}{
		{name: "disabled", harness: "agy", remap: false},
		{name: "unknown", harness: "other", remap: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newHarnessSessionFake(t, tc.harness, tc.remap)
			if f.proxy.ttyProfile != nil || f.proxy.terminal != nil {
				t.Fatal("pass-through harness must not own profile or terminal")
			}
			var out bytes.Buffer
			f.proxy.translateStdinFrom(strings.NewReader("\r\x1b\r"), &out, pendingFlushAfter)
			if got := out.Bytes(); !bytes.Equal(got, []byte("\r\x1b\r")) {
				t.Fatalf("input = %q, want unchanged CR and Alt+Return", got)
			}
		})
	}
}

func TestHarnessTTYIntegration_CodexCaptureOverlayPrecedence(t *testing.T) {
	f := newHarnessSessionFake(t, "codex", true)
	t.Cleanup(f.close)
	f.proxy.captureOutPath = "capture"
	f.proxy.armCapture()
	if !f.proxy.pickerActive.Load() {
		t.Fatal("Codex capture must arm overlay state from its profile capability")
	}
	if got := f.enter(); !bytes.Equal(got, []byte{'\r'}) || f.proxy.pickerActive.Load() {
		t.Fatalf("capture overlay Enter = %q active=%t, want bare CR and clear", got, f.proxy.pickerActive.Load())
	}
}
