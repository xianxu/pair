package wrapcmd

import (
	"bytes"
	"io"
	"testing"
	"time"
)

// TestCheckOverlayOpen_PickerVariantFlipsFlag confirms the OSC
// body that claude emits for AskUserQuestion / tool-permission
// overlays trips pickerActive. This is the open half of the
// suspend-Enter-remap-during-overlay contract.
func TestCheckOverlayOpen_PickerVariantFlipsFlag(t *testing.T) {
	p := &proxy{agentBasename: "claude"}
	checkOverlayBytes(p, []byte("\x1b]777;"+pickerOpenOSCBody+"\x07"))
	if !p.pickerActive.Load() {
		t.Fatalf("pickerActive should be true after picker-open OSC")
	}
}

// TestCheckOverlayOpen_WaitingForInputDoesNotFlip pins the
// negative case: the end-of-turn OSC 777 body ("Claude is waiting for
// your input") fires while the textarea has focus. The remap MUST
// stay engaged or the user's next Enter loses its newline.
func TestCheckOverlayOpen_WaitingForInputDoesNotFlip(t *testing.T) {
	p := &proxy{agentBasename: "claude"}
	checkOverlayBytes(p, []byte("\x1b]777;notify;Claude Code;Claude is waiting for your input\x07"))
	if p.pickerActive.Load() {
		t.Fatalf("pickerActive should stay false for end-of-turn OSC")
	}
}

// TestCheckOverlayOpen_AgentsWithoutDetectorSkipped pins the agent-gate:
// agents without an overlay detector must ignore Claude's OSC body.
func TestCheckOverlayOpen_AgentsWithoutDetectorSkipped(t *testing.T) {
	for _, name := range []string{"unsupported", ""} {
		p := &proxy{agentBasename: name}
		checkOverlayBytes(p, []byte("\x1b]777;"+pickerOpenOSCBody+"\x07"))
		if p.pickerActive.Load() {
			t.Fatalf("agent %q: pickerActive should stay false (no detector)", name)
		}
	}
}

// TestCheckOverlayOpen_AgyPickerMarkers confirms that when a visible chunk
// contains any agy picker markers, pickerActive is set to true.
func TestCheckOverlayOpen_AgyPickerMarkers(t *testing.T) {
	p := &proxy{agentBasename: "agy"}
	checkOverlayBytes(p, []byte("Do you want to proceed?\r\n> 1. Yes\r\n2. No"))
	if !p.pickerActive.Load() {
		t.Fatalf("pickerActive should be true after seeing agy picker marker")
	}
}

// TestCheckOverlayOpen_UnrelatedOSCSkipped covers the broader
// negative case: any OSC that isn't the picker-open body must be a
// no-op, even on claude. Guards against accidentally tripping on
// OSC 9 / OSC 1337 / arbitrary 777 bodies.
func TestCheckOverlayOpen_UnrelatedOSCSkipped(t *testing.T) {
	cases := []struct {
		ps, body string
	}{
		{"9", "Claude needs your permission"},
		{"777", "notify;Other App;some message"},
		{"777", "title;ignored"},
		{"1337", "anything"},
	}
	for _, c := range cases {
		p := &proxy{agentBasename: "claude"}
		checkOverlayBytes(p, []byte("\x1b]"+c.ps+";"+c.body+"\x07"))
		if p.pickerActive.Load() {
			t.Fatalf("ps=%q body=%q: should not flip pickerActive", c.ps, c.body)
		}
	}
}

func checkOverlayBytes(p *proxy, raw []byte) {
	p.checkOverlayOpen(raw, raw)
}

// TestEmitPlainCR_OverlayActiveSendsBareCRAndClears is the close half
// of the contract: while pickerActive is set, the user's plain Enter
// passes through as \r so the overlay confirms — and the flag clears
// so the next Enter (back in the textarea) gets the normal \<CR>
// remap.
func TestEmitPlainCR_OverlayActiveSendsBareCRAndClears(t *testing.T) {
	p := claudeProxy()
	p.pickerActive.Store(true)

	out := p.emitPlainCR(nil)
	if !bytes.Equal(out, []byte{'\r'}) {
		t.Fatalf("got %q, want bare \\r while overlay active", out)
	}
	if p.pickerActive.Load() {
		t.Fatalf("pickerActive should clear after consuming one Enter")
	}

	// Next Enter goes through the normal remap again.
	out = p.emitPlainCR(nil)
	if !bytes.Equal(out, []byte{'\\', '\r'}) {
		t.Fatalf("got %q, want \\\\r after overlay cleared", out)
	}
}

// TestEmitPlainCR_OverlayInactivePreservesRemap pins the default
// path: with no overlay, the user's plain Enter must still translate
// to the textarea-aware \<CR>. Guards against a flag-Load() typo
// silently disabling the remap everywhere.
func TestEmitPlainCR_OverlayInactivePreservesRemap(t *testing.T) {
	p := claudeProxy()
	out := p.emitPlainCR(nil)
	if !bytes.Equal(out, []byte{'\\', '\r'}) {
		t.Fatalf("got %q, want \\\\r when overlay inactive", out)
	}
}

// TestTranslateStdin_OverlayActiveBypassesRemap drives the close
// path through the same goroutine + select-loop machinery the
// production path uses. With pickerActive set before the user's
// Enter arrives, the byte stream out should be a bare \r (overlay
// confirm), and subsequent Enters revert to the normal \\r remap.
func TestTranslateStdin_OverlayActiveBypassesRemap(t *testing.T) {
	r, w := io.Pipe()
	out := newDrainBuffer()
	p := claudeProxy()
	p.pickerActive.Store(true)

	done := make(chan struct{})
	go func() {
		p.translateStdinFrom(r, out, 10*time.Millisecond)
		close(done)
	}()

	_, _ = w.Write([]byte{'\r'}) // user confirms picker
	if !waitFor(200*time.Millisecond, func() bool { return bytes.Equal(out.Bytes(), []byte{'\r'}) }) {
		t.Fatalf("got %q, want bare \\r (overlay confirm)", out.Bytes())
	}
	if p.pickerActive.Load() {
		t.Fatalf("pickerActive should clear after the confirming Enter")
	}

	// Next Enter is back in the textarea — must use the remap.
	_, _ = w.Write([]byte{'\r'})
	want := []byte{'\r', '\\', '\r'}
	if !waitFor(200*time.Millisecond, func() bool { return bytes.Equal(out.Bytes(), want) }) {
		t.Fatalf("got %q, want %q (overlay then textarea Enter)", out.Bytes(), want)
	}

	_ = w.Close()
	<-done
}
