package main

import (
	"bytes"
	"testing"
)

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
			detect, ok := overlayDetectorByAgent[c.agent]
			if !ok {
				t.Fatalf("missing detector for %s", c.agent)
			}
			open, match := detect(&proxy{}, c.raw, c.raw)
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
	p := &proxy{agentBasename: "codex", sendKM: sendKeymapByAgent["codex"]}
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
	p := &proxy{
		agentBasename:  "codex",
		sendKM:         sendKeymapByAgent["codex"],
		captureOutPath: "capture",
	}

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
	p := &proxy{agentBasename: "codex"}
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
