package wrapcmd

import (
	"bytes"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

// pair#464: a picker latch outliving its picker. The screens are re-rendered
// from the recorded pair:3 pane (testdata README).

const claudePermissionOSC = "\x1b]777;" + pickerOpenOSCBody + "\x07"

func pickerFake(t *testing.T) *harnessSessionFake {
	t.Helper()
	f := newHarnessSessionFake(t, "claude", true)
	t.Cleanup(f.close)
	f.resize(94, 39)
	return f
}

func TestRecordedIdleScreensReadDeliverable(t *testing.T) {
	for name, want := range map[string]PeerComposerState{
		"permission-autodeny":   PeerComposerUnknown,
		"permission-autodenied": PeerComposerEmpty,
		"shell-footer":          PeerComposerEmpty,
		"idle-suggestion":       PeerComposerEmpty,
	} {
		f := pickerFake(t)
		f.output(peerBusyScreen(t, name))
		if got := peerComposerState("claude", f.proxy.terminal.Snapshot()); got != want {
			t.Errorf("%s: composer %v want %v", name, got, want)
		}
		if f.proxy.pickerActive.Load() {
			t.Errorf("%s: no picker OSC was sent, yet the latch armed", name)
		}
	}
}

// Claude Code auto-denies an unattended permission prompt: the picker closes
// with no key pressed, so only the screen can say it closed.
func TestPickerLatchClearsWhenComposerReturns(t *testing.T) {
	f := pickerFake(t)
	f.output(peerBusyScreen(t, "permission-autodeny") + claudePermissionOSC)
	if !f.proxy.pickerActive.Load() {
		t.Fatal("permission OSC did not arm the latch")
	}
	f.output(peerBusyScreen(t, "permission-autodeny")) // the countdown repaints
	if !f.proxy.pickerActive.Load() {
		t.Fatal("latch cleared while the picker is still on screen")
	}
	f.output(peerBusyScreen(t, "permission-autodenied"))
	if f.proxy.pickerActive.Load() {
		t.Fatal("latch still armed after the auto-deny brought the composer back")
	}
	f.proxy.overlayMu.Lock()
	hidden, tail := f.proxy.overlayComposerHidden, f.proxy.overlayRawTail
	f.proxy.overlayMu.Unlock()
	if hidden || tail != nil {
		t.Fatalf("close left overlay state: hidden=%v tail=%q", hidden, tail)
	}
	// The next Enter is the composer's again, not the picker's.
	if got := f.enter(); !bytes.Equal(got, f.proxy.ttyProfile.keymap.plainCR) {
		t.Fatalf("Enter after the close = %q, want composer remap %q", got, f.proxy.ttyProfile.keymap.plainCR)
	}
}

// The open OSC can arrive while the composer is still drawn, ahead of the
// picker's paint; that composer is not the picker closing.
func TestPickerLatchHoldsUntilThePickerHidTheComposer(t *testing.T) {
	f := pickerFake(t)
	f.output(peerBusyScreen(t, "idle-suggestion") + claudePermissionOSC)
	f.output(peerBusyScreen(t, "idle-suggestion"))
	if !f.proxy.pickerActive.Load() {
		t.Fatal("latch cleared by a composer drawn before the picker painted")
	}
	f.output(peerBusyScreen(t, "permission-autodeny"))
	f.output(peerBusyScreen(t, "idle-suggestion"))
	if f.proxy.pickerActive.Load() {
		t.Fatal("latch still armed after picker → composer")
	}
}

// Only a harness measured to hide its composer behind pickers closes the latch
// from the screen; the rest keep the Enter-only rule.
func TestPickerLatchScreenCloseIsPerProfile(t *testing.T) {
	for name, profile := range harnessTTYProfiles {
		if profile.overlayHidesComposer != (name == "claude") {
			t.Errorf("%s: overlayHidesComposer=%v; only claude is measured", name, profile.overlayHidesComposer)
		}
	}
	f := pickerFake(t)
	profile := *f.proxy.ttyProfile
	profile.overlayHidesComposer = false
	f.proxy.ttyProfile = &profile
	f.output(peerBusyScreen(t, "permission-autodeny") + claudePermissionOSC)
	f.output(peerBusyScreen(t, "permission-autodenied"))
	if !f.proxy.pickerActive.Load() {
		t.Fatal("a profile without overlayHidesComposer closed the latch from the screen")
	}
}

// The pair:3 sequence end to end: a message to the idle slot after an
// auto-denied prompt is submitted, not held "waiting for picker to close".
func TestPeerDeliveryAfterAutoDeniedPrompt(t *testing.T) {
	f := pickerFake(t)
	now := time.Now()
	d := newPeerDelivery(peerTestMessage(now).To, func() time.Time { return now })
	f.proxy.peer = d
	peerRender(f, d, peerBusyScreen(t, "permission-autodeny")+claudePermissionOSC)
	peerRender(f, d, peerBusyScreen(t, "permission-autodenied"))
	peerRender(f, d, peerBusyScreen(t, "idle-suggestion"))
	peerIntegrationEnqueue(t, d)
	var out bytes.Buffer
	f.proxy.dispatchPeer(&out)
	if !strings.Contains(out.String(), "hello") {
		t.Fatalf("no paste into the idle composer: %+v", d.receipt())
	}
	// The paste replaces the suggestion; Claude Code draws it, and the
	// submit follows the fixed delay.
	pasted := strings.Replace(peerBusyScreen(t, "idle-suggestion"), "The operator reviewed the format; proceed to close and merge", "hello", 1)
	peerRender(f, d, strings.Replace(pasted, "\x1b[2m", "", -1))
	now = now.Add(PeerSubmitDelay)
	f.proxy.dispatchPeer(&out)
	if !bytes.HasSuffix(out.Bytes(), f.proxy.ttyProfile.keymap.altCR) {
		t.Fatalf("no submit: %+v", d.receipt())
	}
	now = now.Add(time.Second)
	peerRender(f, d, peerBusyScreen(t, "permission-autodenied"))
	f.proxy.dispatchPeer(&out)
	if r := d.receipt(); r.Status != couchmessage.Submitted {
		t.Fatalf("not submitted: %+v", r)
	}
}

// The screen close leans on the composer recognizer, so a picker drawn in the
// composer's shape would close the latch while it is open. The nearest
// recorded one: a question picker's free-text row, `❯ 4. Type something.`,
// at column 0 between two rules, with the cursor on it.
func TestPickerLatchHoldsOverQuestionPickerTextRow(t *testing.T) {
	raw, err := os.ReadFile("testdata/peer/claude/2026-08-31/question-picker.raw")
	if err != nil {
		t.Fatal(err)
	}
	f := pickerFake(t)
	f.resize(80, 38)
	f.output(string(raw) + claudePermissionOSC)
	for i := 0; i < 3; i++ {
		f.output(string(raw))
	}
	if !f.proxy.pickerActive.Load() {
		t.Fatal("a question picker's free-text row read as the composer coming back")
	}
	if got := peerComposerState("claude", f.proxy.terminal.Snapshot()); got != PeerComposerUnknown {
		t.Fatalf("question picker reads as composer %v", got)
	}
}

// pickerReports records what the wrapper tells Couch about its picker.
type pickerReports struct {
	mu  sync.Mutex
	got []bool
}

func (r *pickerReports) Picker(open bool) {
	r.mu.Lock()
	r.got = append(r.got, open)
	r.mu.Unlock()
}

func (r *pickerReports) all() []bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]bool(nil), r.got...)
}

// pair#465: Couch hears the picker the moment the latch changes, not after a
// quiet interval: the auto-deny countdown repaints every second, so a
// quiet-gated report would never come while the prompt is up.
func TestPickerReportedToCouchAtOnce(t *testing.T) {
	f := pickerFake(t)
	reports := &pickerReports{}
	f.proxy.pickerSink = reports
	f.output(peerBusyScreen(t, "permission-autodeny") + claudePermissionOSC)
	if got := reports.all(); !reflect.DeepEqual(got, []bool{true}) {
		t.Fatalf("after the picker opened, reports %v", got)
	}
	f.output(peerBusyScreen(t, "permission-autodeny") + claudePermissionOSC) // the countdown repaints
	if got := reports.all(); !reflect.DeepEqual(got, []bool{true}) {
		t.Fatalf("a repaint re-reported: %v", got)
	}
	f.output(peerBusyScreen(t, "permission-autodenied"))
	if got := reports.all(); !reflect.DeepEqual(got, []bool{true, false}) {
		t.Fatalf("after the auto-deny closed it, reports %v", got)
	}
}

// pair#465: a message to a slot behind a picker is held, says which picker,
// outlives its own deadline while the picker is up, and is pasted once the
// picker closes.
func TestPeerDeliveryHeldByAPickerNamesIt(t *testing.T) {
	f := pickerFake(t)
	now := time.Now()
	d := newPeerDelivery(peerTestMessage(now).To, func() time.Time { return now })
	f.proxy.peer = d
	peerRender(f, d, peerBusyScreen(t, "permission-autodeny")+claudePermissionOSC)
	peerIntegrationEnqueue(t, d)
	var out bytes.Buffer
	f.proxy.dispatchPeer(&out)
	want := "blocked by a picker: Do you want to proceed?"
	if r := d.receipt(); r.Status != couchmessage.Held || r.Detail != want || out.Len() != 0 {
		t.Fatalf("receipt %+v, out %q; want held %q", r, out.String(), want)
	}
	// The countdown repaints past the message's own deadline: still held.
	now = now.Add(couchmessage.DeliveryTimeout + time.Minute)
	peerRender(f, d, peerBusyScreen(t, "permission-autodeny"))
	f.proxy.dispatchPeer(&out)
	if r := d.receipt(); r.Status != couchmessage.Held || r.Detail != want {
		t.Fatalf("past its deadline behind the picker: %+v", r)
	}
	// The auto-deny closes the picker: the message goes in.
	peerRender(f, d, peerBusyScreen(t, "permission-autodenied"))
	now = now.Add(2 * time.Second)
	f.proxy.dispatchPeer(&out)
	if !strings.Contains(out.String(), "hello") {
		t.Fatalf("no paste after the picker closed: %+v", d.receipt())
	}
}

// A picker that outlasts the hold limit expires the message with its reason.
func TestPeerDeliveryExpiresBehindAPickerWithItsReason(t *testing.T) {
	f := pickerFake(t)
	now := time.Now()
	d := newPeerDelivery(peerTestMessage(now).To, func() time.Time { return now })
	f.proxy.peer = d
	peerRender(f, d, peerBusyScreen(t, "permission-autodeny")+claudePermissionOSC)
	peerIntegrationEnqueue(t, d)
	var out bytes.Buffer
	f.proxy.dispatchPeer(&out)
	now = now.Add(couchmessage.DeliveryTimeout + couchmessage.MaxHold + time.Second)
	f.proxy.dispatchPeer(&out)
	r := d.receipt()
	if r.Status != couchmessage.Expired || !strings.Contains(r.Detail, "held past 30m0s, blocked by a picker: Do you want to proceed?") {
		t.Fatalf("receipt %+v", r)
	}
}
