package wrapcmd

import (
	"bytes"
	"errors"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/creack/pty"
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
	if f.proxy.terminal == nil {
		return
	}
	if err := f.proxy.terminal.Resize(cols, rows); err != nil {
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
	f.output(agyLiveComposerPaint())
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
	f.output(agyLiveComposerPaint())
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

func TestHarnessTTYIntegration_SetWinsizeRejectsInvalidBeforeSideEffects(t *testing.T) {
	f := newHarnessSessionFake(t, "agy", true)
	t.Cleanup(f.close)
	f.output(agyLiveComposerPaint())
	before := f.proxy.terminal.Snapshot()
	f.proxy.stdinFile = new(os.File)
	f.proxy.ptmx = new(os.File)
	f.proxy.getWinsize = func(*os.File) (*pty.Winsize, error) {
		return &pty.Winsize{Cols: 0, Rows: 30}, nil
	}
	setCalls := 0
	f.proxy.setPTYWinsize = func(*os.File, *pty.Winsize) error {
		setCalls++
		return nil
	}

	f.proxy.setWinsize()

	if setCalls != 0 {
		t.Fatalf("PTY resize calls = %d, want zero for invalid dimensions", setCalls)
	}
	if after := f.proxy.terminal.Snapshot(); !reflect.DeepEqual(after, before) {
		t.Fatal("invalid production resize mutated terminal model")
	}
}

func TestHarnessTTYIntegration_SetWinsizeFailureLatchesAuthorization(t *testing.T) {
	f := newHarnessSessionFake(t, "agy", true)
	t.Cleanup(f.close)
	f.output(agyLiveComposerPaint())
	if got := f.enter(); !bytes.Equal(got, []byte{'\n'}) {
		t.Fatalf("precondition Enter = %q, want active composer LF", got)
	}

	f.proxy.stdinFile = new(os.File)
	f.proxy.ptmx = new(os.File)
	f.proxy.getWinsize = func(*os.File) (*pty.Winsize, error) {
		return &pty.Winsize{Cols: 60, Rows: 30}, nil
	}
	setErr := errors.New("injected PTY resize failure")
	f.proxy.setPTYWinsize = func(*os.File, *pty.Winsize) error { return setErr }
	f.proxy.setWinsize()

	f.output(agyLiveComposerPaint())
	if got := f.enter(); !bytes.Equal(got, []byte{'\r'}) {
		t.Fatalf("Enter after failed PTY resize = %q, want latched bare CR", got)
	}

	f.proxy.setPTYWinsize = func(*os.File, *pty.Winsize) error { return nil }
	f.proxy.setWinsize()
	if got := f.enter(); !bytes.Equal(got, []byte{'\r'}) {
		t.Fatalf("Enter after resize commit without fresh show = %q, want bare CR", got)
	}
	f.output("\x1b[?25h")
	if got := f.enter(); !bytes.Equal(got, []byte{'\n'}) {
		t.Fatalf("Enter after fresh post-commit show = %q, want LF", got)
	}
}

// codexLiveComposerPaint is the byte sequence that paints a live Codex
// composer: a bold U+203A prompt at column 0 with the cursor resting in the
// text that follows. Tests share one definition so a Codex repaint is updated
// in a single place.
func codexLiveComposerPaint() string {
	return "\x1b[20;1H\x1b[1m\u203a\x1b[22m alpha\x1b[?25h\x1b[20;9H"
}

// agyLiveComposerPaint is the byte sequence that paints a live Agy composer:
// dim rules (SGR 90) enclosing a bright-blue prompt (SGR 94) with the cursor
// inside the box. Agy's permission picker paints an unstyled ">" in the same
// place, so the styling is load-bearing, not decoration.
func agyLiveComposerPaint() string {
	return "\x1b[10;1H\x1b[90m──────────\x1b[11;1H\x1b[94m>\x1b[39m work" +
		"\x1b[13;1H\x1b[90m──────────\x1b[39m\x1b[?25h\x1b[12;3H"
}

// claudeLiveComposerPaint is the byte sequence that paints a live Claude
// composer: a prompt glyph at column 0 between two rules sharing one
// foreground, with the cursor in the text that follows. Claude repaints both
// glyph and rule colour per input mode, so the recognizer keys on the shape
// rather than on these particular values.
func claudeLiveComposerPaint() string {
	rule := "\x1b[38;2;136;136;136m" + strings.Repeat("─", 40) + "\x1b[39m"
	return "\x1b[20;1H" + rule +
		"\x1b[21;1H❯ alpha" +
		"\x1b[22;1H" + rule +
		"\x1b[?25h\x1b[21;9H"
}
