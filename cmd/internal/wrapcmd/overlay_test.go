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

// TestHandleChunk_OscScannedBeforeCarryIsBounded pins the shared-pump half of
// the BR-35 rule for the sibling harnesses: the pump used to bound `rolling`
// to rollingTailLen before checkOverlayOpen, so an OSC sitting more than a
// tail's worth of bytes before the end of one chunk was never scanned at all.
// Claude's picker OSC anywhere in the chunk must still arm the overlay.
func TestHandleChunk_OscScannedBeforeCarryIsBounded(t *testing.T) {
	profile, ok := profileForHarness("claude", true)
	if !ok {
		t.Fatal("claude profile missing")
	}
	p := &proxy{agentBasename: "claude", ttyProfile: &profile}
	rolling := make([]byte, 0, rollingTailLen*2)
	chunk := append([]byte("\x1b]777;"+pickerOpenOSCBody+"\x07"), bytes.Repeat([]byte("x"), rollingTailLen+128)...)
	p.handleChunk(chunk, &rolling)
	if !p.pickerActive.Load() {
		t.Fatal("picker OSC ahead of a long chunk's tail window was not detected")
	}
	if len(rolling) > rollingTailLen {
		t.Fatalf("carry = %d bytes, want bounded to %d", len(rolling), rollingTailLen)
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
		{
			name:     "muse startup text does not open overlay from generic enter hint",
			agent:    "muse",
			raw:      []byte("Enter to select"),
			wantOpen: false,
		},
		{
			// Qoder's permission picker, as captured live in overlay.raw: the
			// question row is painted word-by-word at absolute columns, so no
			// spaces survive the strip between its words.
			name:      "qoder permission picker question opens overlay",
			agent:     "qoder",
			raw:       []byte("\x1b[2GAllow\x1b[8Gthis\x1b[13Gcommand\x1b[21Gto\x1b[24Grun?\r\r\n"),
			wantOpen:  true,
			wantMatch: "Allowthiscommandtorun?",
		},
		{
			name:      "qoder permission picker header opens overlay",
			agent:     "qoder",
			raw:       []byte("\x1b[38;2;238;238;235mPermission Required\x1b[39m"),
			wantOpen:  true,
			wantMatch: "Permission Required",
		},
		{
			// The header marker is the one deliberate exception to "markers
			// must be chrome agent prose cannot forge": it is ordinary
			// English, but it titles every permission picker. BR-38 pins the
			// exemption's boundary — the topic in other words, other case, or
			// not contiguous must stay closed.
			name:     "qoder spaced prose about permissions does not open overlay",
			agent:    "qoder",
			raw:      []byte("Permission is required for this tool to run.\r\n"),
			wantOpen: false,
		},
		{
			name:     "qoder composer text does not open overlay",
			agent:    "qoder",
			raw:      []byte("\x1b[7;1H\x1b[38;2;149;149;146m────\x1b[8;1H\x1b[38;2;149;124;173m> \x1b[?25h\x1b[8;3HType your message or @path/to/file"),
			wantOpen: false,
		},
		{
			// Qoder's question picker header, as captured live in
			// selection.raw: two styled runs split by an absolute-column jump,
			// so the gap between "Asking" and "User" carries no space byte.
			name:      "qoder question picker header opens overlay",
			agent:     "qoder",
			raw:       []byte("\r\r\n\r\r\n\x1b[38;2;238;238;235m\x1b[1m Asking\x1b[22m\x1b[39m\x1b[9G\x1b[38;2;238;238;235m\x1b[1mUser\x1b[22m\x1b[39m\x1b[K\r\x1b[1B"),
			wantOpen:  true,
			wantMatch: "AskingUser",
		},
		{
			// The question picker's keybinding footer, verbatim from
			// selection.raw's strip. It is the family's own statement that
			// Enter selects the highlighted option.
			name:      "qoder question picker footer opens overlay",
			agent:     "qoder",
			raw:       []byte("\x1b[38;2;149;149;143m\u2191\u2193navigate\u00b7Enterselect\u00b7Esccancel\x1b[39m"),
			wantOpen:  true,
			wantMatch: "Enterselect",
		},
		{
			// A composer message that merely discusses selecting must not arm
			// the overlay: the marker is the glued footer paint, not the words.
			name:     "qoder prose about enter select does not open overlay",
			agent:    "qoder",
			raw:      []byte("press Enter to select an option"),
			wantOpen: false,
		},
		{
			// The same prose painted the way Qoder paints its picker bodies —
			// word-by-word at absolute columns, so the strip glues the words.
			// "for future sessions" was a marker in the first cut of
			// qoderPickerMarkers and would arm here; it is ordinary prose, so
			// any agent output gluing it would turn the user's next composer
			// Enter into a submit. Re-adding it fails this row.
			name:     "qoder glued prose about future sessions does not open overlay",
			agent:    "qoder",
			raw:      []byte("saved \x1b[8Gfor\x1b[12Gfuture\x1b[19Gsessions"),
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
	f.output(codexLiveComposerPaint())
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
