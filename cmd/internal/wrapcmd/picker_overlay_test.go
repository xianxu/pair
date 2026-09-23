package wrapcmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestCheckOverlayOpen_PickerVariantFlipsFlag confirms the OSC
// body that claude emits for AskUserQuestion / tool-permission
// overlays trips pickerActive. This is the open half of the
// suspend-Enter-remap-during-overlay contract.
func TestCheckOverlayOpen_PickerVariantFlipsFlag(t *testing.T) {
	p := proxyForHarness("claude")
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
	p := proxyForHarness("claude")
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
	p := proxyForHarness("agy")
	checkOverlayBytes(p, []byte("Do you want to proceed?\r\n> 1. Yes\r\n2. No"))
	if !p.pickerActive.Load() {
		t.Fatalf("pickerActive should be true after seeing agy picker marker")
	}
}

// TestCheckOverlayOpen_QoderPermissionPicker confirms that Qoder's permission
// picker trips pickerActive through the profile registry, so the next plain
// Return confirms the highlighted choice instead of remapping to a newline the
// picker would never accept. The question row arrives word-by-word, matching
// the frozen overlay.raw paint.
func TestCheckOverlayOpen_QoderPermissionPicker(t *testing.T) {
	p := proxyForHarness("qoder")
	checkOverlayBytes(p, []byte("\x1b[2GAllow\x1b[8Gthis\x1b[13Gcommand\x1b[21Gto\x1b[24Grun?\r\r\n"))
	if !p.pickerActive.Load() {
		t.Fatalf("pickerActive should be true after seeing qoder permission picker")
	}
}

// TestCheckOverlayOpen_QoderDoesNotRedetectStalePickerText is the consumption
// counterpart of the codex stale-text test, over both frozen qoder captures:
// the raw haystack that makes marker detection split-proof must be cleared
// when the confirming Enter consumes pickerActive, or its own consumed bytes
// re-arm the flag on the next chunk and the following composer Enter passes a
// bare CR, submitting a draft the user meant to continue on a new line.
func TestCheckOverlayOpen_QoderDoesNotRedetectStalePickerText(t *testing.T) {
	for _, file := range []string{"overlay.raw", "selection.raw"} {
		t.Run(file, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join("testdata", "tty", "qoder", "1.1.60", file))
			if err != nil {
				t.Fatalf("read %s: %v", file, err)
			}
			p := proxyForHarness("qoder")
			checkOverlayBytes(p, raw)
			if !p.pickerActive.Load() {
				t.Fatalf("%s paint must arm pickerActive", file)
			}
			if got := p.emitPlainCR(nil); !bytes.Equal(got, []byte{'\r'}) {
				t.Fatalf("confirming Enter = %q, want bare CR", got)
			}
			if p.pickerActive.Load() {
				t.Fatal("pickerActive should clear after the confirming Enter")
			}
			// A chunk with no marker of its own must not re-arm the flag from
			// the consumed picker bytes still inside the raw window.
			checkOverlayBytes(p, []byte("\x1b[1G\x1b[2K"))
			if p.pickerActive.Load() {
				t.Fatalf("%s: pickerActive re-armed from consumed picker bytes", file)
			}
		})
	}
}

// TestCheckOverlayOpen_QoderSplitFooterSurvivesLongSecondChunk is BR-35's pin:
// the raw window must be scanned at its full carry+chunk length BEFORE it is
// re-bounded to one tail. The split below cuts the footer's \x1b[23m — the same
// escape the selection.raw 6426/6616 replay cut — so the marker exists only in
// the concatenation. A window bounded before the scan drops the marker's head
// once the second chunk carries more than a tail's worth of bytes: measured
// during the M3 review as armed with 0 and 100 bytes of filler, disarmed from
// 400. The 0-byte row is the short-chunk control; 600 and 2000 discriminate.
func TestCheckOverlayOpen_QoderSplitFooterSurvivesLongSecondChunk(t *testing.T) {
	head := []byte("\x1b[38;2;149;149;143m\u2191\u2193navigate\u00b7Enter\x1b[2")
	tail := []byte("3mselect\u00b7Esccancel\x1b[39m")
	for _, filler := range []int{0, 600, 2000} {
		t.Run(fmt.Sprintf("filler=%d", filler), func(t *testing.T) {
			p := proxyForHarness("qoder")
			checkOverlayBytes(p, head)
			if p.pickerActive.Load() {
				t.Fatal("the split head must not arm the overlay on its own")
			}
			second := append(append([]byte(nil), tail...), bytes.Repeat([]byte("x"), filler)...)
			checkOverlayBytes(p, second)
			if !p.pickerActive.Load() {
				t.Fatalf("footer split inside its escape with %d bytes of filler did not arm pickerActive", filler)
			}
		})
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
		p := proxyForHarness("claude")
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
