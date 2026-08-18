package wrapcmd

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestEmitPlainCR_ConcurrentOverlayRearmRetainsNewStateAndTail(t *testing.T) {
	p := proxyForHarness("codex")
	p.pickerActive.Store(true)
	p.overlayTextTail = "old overlay"

	consumeLocked := make(chan struct{})
	releaseConsume := make(chan struct{})
	p.overlayConsumeHook = func() {
		close(consumeLocked)
		<-releaseConsume
	}

	enterDone := make(chan []byte, 1)
	go func() { enterDone <- p.emitPlainCR(nil) }()
	<-consumeLocked

	detectStarted := make(chan struct{})
	detectDone := make(chan struct{})
	go func() {
		close(detectStarted)
		raw := []byte("Press enter to continue")
		p.checkOverlayOpen(raw, raw)
		close(detectDone)
	}()
	<-detectStarted
	close(releaseConsume)

	if got := <-enterDone; !bytes.Equal(got, []byte{'\r'}) {
		t.Fatalf("older overlay Enter = %q, want bare CR", got)
	}
	<-detectDone
	if !p.pickerActive.Load() {
		t.Fatal("new overlay was erased by older Enter")
	}
	p.overlayMu.Lock()
	tail := p.overlayTextTail
	p.overlayMu.Unlock()
	if !strings.Contains(tail, "Press enter to continue") {
		t.Fatalf("new overlay tail = %q, want new marker retained", tail)
	}
}

func TestHandleChunk_PanickingOverlayDetectorDoesNotStrandReturn(t *testing.T) {
	profile := harnessTTYProfile{
		keymap:       sendKeymap{plainCR: []byte{'\\', '\r'}},
		composerGate: composerGateLegacy,
		overlay: func(*proxy, []byte, []byte) (bool, string) {
			panic("injected detector panic")
		},
	}
	p := &proxy{ttyProfile: &profile}
	rolling := []byte{}

	p.handleChunk([]byte("detector input"), &rolling)

	done := make(chan []byte, 1)
	go func() { done <- p.emitPlainCR(nil) }()
	select {
	case got := <-done:
		if !bytes.Equal(got, []byte{'\\', '\r'}) {
			t.Fatalf("Return after recovered detector panic = %q, want remap", got)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Return blocked after recovered detector panic stranded overlay lock")
	}
}

func TestOverlayDetectorByAgent(t *testing.T) {
	cases := []struct {
		name      string
		agent     string
		raw       []byte
		wantOpen  bool
		wantMatch string
	}{
		{
			name:      "claude permission OSC opens overlay",
			agent:     "claude",
			raw:       []byte("\x1b]777;" + pickerOpenOSCBody + "\x07"),
			wantOpen:  true,
			wantMatch: pickerOpenOSCBody,
		},
		{
			name:     "claude waiting OSC is not overlay",
			agent:    "claude",
			raw:      []byte("\x1b]777;notify;Claude Code;Claude is waiting for your input\x07"),
			wantOpen: false,
		},
		{
			name:      "codex resume cwd picker opens overlay",
			agent:     "codex",
			raw:       []byte("\x1b[2m%Session = latest cwd\x1b[0m\r\n\x1b[7mUse session directory (/tmp/old)\x1b[0m"),
			wantOpen:  true,
			wantMatch: "Use session directory (",
		},
		{
			name:      "codex generic enter footer opens overlay",
			agent:     "codex",
			raw:       []byte("\x1b[?25lPress enter to continue\x1b[?25h"),
			wantOpen:  true,
			wantMatch: "Press enter to continue",
		},
		{
			name:      "codex quota model picker opens overlay",
			agent:     "codex",
			raw:       []byte("\x1b[2mPress enter to confirm or esc to go back\x1b[0m"),
			wantOpen:  true,
			wantMatch: "Press enter to confirm or esc to go back",
		},
		{
			name:      "codex permission picker cancel footer opens overlay",
			agent:     "codex",
			raw:       []byte("\x1b[38;2;137;180;250m1. Yes, proceed (y)  2. No, and tell Codex what to do differently (esc)\r\n\x1b[2mPress enter to confirm or esc to cancel\x1b[0m"),
			wantOpen:  true,
			wantMatch: "Press enter to confirm or esc to cancel",
		},
		{
			name:      "codex request user input OSC opens overlay",
			agent:     "codex",
			raw:       []byte("\x1b]9;Plan mode prompt: Probe\x07"),
			wantOpen:  true,
			wantMatch: "Plan mode prompt: Probe",
		},
		{
			name:     "codex normal textarea does not open overlay",
			agent:    "codex",
			raw:      []byte("+----------------------------------------+\r\n| > write a message                       |"),
			wantOpen: false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			profile, ok := profileForHarness(c.agent, true)
			if !ok {
				t.Fatalf("missing detector for %s", c.agent)
			}
			open, match := profile.overlay(&proxy{}, c.raw, c.raw)
			if open != c.wantOpen {
				t.Fatalf("open = %v, want %v (match %q)", open, c.wantOpen, match)
			}
			if c.wantMatch != "" && match != c.wantMatch {
				t.Fatalf("match = %q, want %q", match, c.wantMatch)
			}
		})
	}
}

func TestTranslateChunk_CodexPickerPlainEnterSelectsOnce(t *testing.T) {
	f := newHarnessSessionFake(t, "codex", true)
	t.Cleanup(f.close)
	f.output("\x1b[19;1H\x1b[48;2;57;57;57m\x1b[K" +
		"\x1b[20;1H\x1b[48;2;57;57;57m\x1b[K" +
		"\x1b[21;1H\x1b[48;2;57;57;57m\x1b[K" +
		"\x1b[?25h\x1b[20;3H")
	p := f.proxy
	p.pickerActive.Store(true)

	got, leftover, inPaste := p.translateChunk([]byte("\r\r"), false)
	if len(leftover) != 0 {
		t.Fatalf("leftover = %q, want none", leftover)
	}
	if inPaste {
		t.Fatal("inPaste = true, want false")
	}
	if want := []byte("\r\n"); !bytes.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
	if p.pickerActive.Load() {
		t.Fatal("pickerActive still set after first plain Enter")
	}
}

func TestArmCapture_CodexArmsImagePickerEnter(t *testing.T) {
	p := proxyForHarness("codex")
	p.captureOutPath = "capture"

	p.armCapture()
	if !p.pickerActive.Load() {
		t.Fatal("pickerActive should be true after Codex image capture starts")
	}
	got := p.emitPlainCR(nil)
	if want := []byte{'\r'}; !bytes.Equal(got, want) {
		t.Fatalf("got %q, want bare CR for image picker confirm", got)
	}
	if p.pickerActive.Load() {
		t.Fatal("pickerActive should clear after confirming Enter")
	}
}

func TestCheckOverlayOpen_CodexDoesNotRedetectStalePickerText(t *testing.T) {
	p := proxyForHarness("codex")
	rolling := []byte("Use session directory (/tmp/old)")

	p.checkOverlayOpen(rolling, rolling)
	if !p.pickerActive.Load() {
		t.Fatal("pickerActive should be true after codex picker text")
	}

	_ = p.emitPlainCR(nil)
	if p.pickerActive.Load() {
		t.Fatal("pickerActive should clear after confirming Enter")
	}

	// The OSC rolling buffer may still contain old picker text after the
	// confirming Enter. Codex detection must only scan new visible output
	// plus its own text carryover, not the stale raw rolling buffer.
	p.checkOverlayOpen([]byte("textarea ready"), rolling)
	if p.pickerActive.Load() {
		t.Fatal("pickerActive rearmed from stale rolling picker text")
	}
}
