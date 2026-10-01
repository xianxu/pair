package wrapcmd

import (
	"bytes"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"io"
	"strings"
	"testing"
	"time"
)

func peerIntegrationFixture(t *testing.T) (*harnessSessionFake, *peerDelivery) {
	t.Helper()
	f := newHarnessSessionFake(t, "claude", true)
	t.Cleanup(f.close)
	f.output("\x1b[?2004h" + strings.Replace(claudeLiveComposerPaint(), "alpha", "", 1) + "\x1b[21;3H")
	d := newPeerDelivery(peerTestMessage(time.Now()).To, time.Now)
	f.proxy.peer = d
	return f, d
}
func peerIntegrationEnqueue(t *testing.T, d *peerDelivery) {
	t.Helper()
	d.mu.Lock()
	sequence := d.sequence
	d.mu.Unlock()
	m := peerTestMessage(time.Now())
	if err := d.reserve(m.ID, sequence); err != nil {
		t.Fatal(err)
	}
	if err := d.enqueue(m); err != nil {
		t.Fatal(err)
	}
}

// The operator's input is already at the child but its repaint has not arrived.
// The earlier empty snapshot cannot authorize a paste into that new draft.
func TestPeerIntegrationUnrenderedOperatorInputBlocksPaste(t *testing.T) {
	f, d := peerIntegrationFixture(t)
	d.admitInput([]byte("human"))
	d.inputForwarded()
	peerIntegrationEnqueue(t, d)
	var out bytes.Buffer
	f.proxy.dispatchPeer(&out)
	if out.Len() != 0 {
		t.Fatalf("peer appended to unrendered operator text: %q", out.String())
	}
	if d.receipt().Status != couchmessage.Queued {
		t.Fatal("must wait for fresh safe input evidence")
	}
}

// Blank spaces remain operator-owned composer content even when Home places
// the cursor back at the prompt and the visible screen appears empty.
func TestPeerIntegrationSpacesThenHomeBlocksPaste(t *testing.T) {
	f, d := peerIntegrationFixture(t)
	d.admitInput([]byte("   \x01"))
	d.inputForwarded()
	repaint := []byte("\x1b[21;3H   \x1b[21;3H")
	d.observeOutput(repaint)
	f.output(string(repaint))
	peerIntegrationEnqueue(t, d)
	var out bytes.Buffer
	f.proxy.dispatchPeer(&out)
	if out.Len() != 0 {
		t.Fatalf("peer overwrote ownership of invisible human draft: %q", out.String())
	}
}

func TestPeerIntegrationBeforePasteObstaclesWaitAndAfterPasteCancel(t *testing.T) {
	for _, kind := range []string{"image", "menu"} {
		t.Run(kind, func(t *testing.T) {
			f, d := peerIntegrationFixture(t)
			peerIntegrationEnqueue(t, d)
			if kind == "image" {
				d.admitImage()
			} else {
				f.proxy.pickerActive.Store(true)
			}
			var out bytes.Buffer
			f.proxy.dispatchPeer(&out)
			if out.Len() != 0 || d.receipt().Status != couchmessage.Queued {
				t.Fatalf("obstacle failed to wait: %q %+v", out.String(), d.receipt())
			}
			if kind == "image" {
				d.humanSubmit()
				d.observeOutput([]byte("\x1b[21;3H"))
				f.output("\x1b[21;3H")
			} else {
				f.proxy.pickerActive.Store(false)
			}
			f.proxy.dispatchPeer(&out)
			if out.Len() == 0 {
				t.Fatal("did not paste after obstacle cleared")
			}
			before := out.String()
			if kind == "image" {
				d.admitImage()
			} else {
				f.proxy.pickerActive.Store(true)
			}
			f.proxy.dispatchPeer(&out)
			if out.String() != before || d.receipt().Status != couchmessage.Cancelled {
				t.Fatalf("afterpaste obstacle did not cancel: %q %+v", out.String(), d.receipt())
			}
		})
	}
}

func TestPeerIntegrationRawOutputMustRenderBeforePaste(t *testing.T) {
	f, d := peerIntegrationFixture(t)
	raw := []byte("\x1b[21;3Hoperator\x1b[21;11H")
	d.observeOutput(raw)
	peerIntegrationEnqueue(t, d)
	var out bytes.Buffer
	f.proxy.dispatchPeer(&out)
	if out.Len() != 0 {
		t.Fatalf("stale empty screen used while raw output queued: %q", out.String())
	}
	f.output(string(raw))
	f.proxy.dispatchPeer(&out)
	if out.Len() != 0 {
		t.Fatalf("occupied freshly rendered composer accepted: %q", out.String())
	}
}

type peerIntegrationShortWriter struct{}

func (peerIntegrationShortWriter) Write(p []byte) (int, error) { return 0, io.ErrShortWrite }

func TestPeerIntegrationOnlyFullyWrittenHumanSendResetsAllowance(t *testing.T) {
	for _, tc := range []struct {
		name, input  string
		short, reset bool
	}{
		{"human alt send", "human\x1b\r", false, true},
		{"partial send", "human\x1b\r", true, false},
		{"pasted return", "\x1b[200~human\rtext\x1b[201~", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, d := peerIntegrationFixture(t)
			var out io.Writer = &bytes.Buffer{}
			if tc.short {
				out = peerIntegrationShortWriter{}
			}
			f.proxy.translateStdinFrom(strings.NewReader(tc.input), out, time.Millisecond)
			reset := false
			select {
			case <-d.submit:
				reset = true
			default:
			}
			if reset != tc.reset {
				t.Fatalf("budget reset=%t want %t", reset, tc.reset)
			}
			d.mu.Lock()
			owned := d.operatorDraft
			submissions := d.submissions
			d.mu.Unlock()
			if (tc.reset && submissions != 1) || (!tc.reset && submissions != 0) {
				t.Fatalf("human submission generation=%d reset=%t", submissions, tc.reset)
			}
			if owned == tc.reset {
				t.Fatalf("operator draft ownership=%t after full send=%t", owned, tc.reset)
			}
		})
	}
}

func TestPeerIntegrationMenuConfirmationDoesNotResetAllowance(t *testing.T) {
	f, d := peerIntegrationFixture(t)
	f.proxy.pickerActive.Store(true)
	var out bytes.Buffer
	f.proxy.translateStdinFrom(strings.NewReader("\x1b\r"), &out, time.Millisecond)
	select {
	case <-d.submit:
		t.Fatal("menu confirmation reset peer allowance")
	default:
	}
}

// A successful human write releases their draft, but the old empty screen is
// not evidence that the child has consumed the submission and is safe again.
func TestPeerIntegrationHumanSubmitNeedsFreshComposerEvidence(t *testing.T) {
	f, d := peerIntegrationFixture(t)
	d.admitInput([]byte("human\x1b\r"))
	// Exercise the same full-write accounting order as translateStdinFrom.
	d.humanSubmit()
	d.inputForwarded()
	peerIntegrationEnqueue(t, d)
	var out bytes.Buffer
	f.proxy.dispatchPeer(&out)
	if out.Len() != 0 {
		t.Fatalf("pre-submission snapshot authorized another paste: %q", out.String())
	}
	if d.receipt().Status != couchmessage.Queued {
		t.Fatal("submission must wait for new safe composer evidence")
	}
}

func TestPeerIntegrationFocusReportsDoNotOwnDraft(t *testing.T) {
	for _, tc := range []struct {
		name   string
		chunks []string
		owned  bool
	}{
		{"focus in", []string{"\x1b[I"}, false},
		{"focus out", []string{"\x1b[O"}, false},
		{"split focus in", []string{"\x1b", "[", "I"}, false},
		{"split focus out", []string{"\x1b[", "O"}, false},
		{"focus pair", []string{"\x1b[I\x1b[O"}, false},
		{"focus then text", []string{"\x1b[Ihuman"}, true},
		{"text then focus", []string{"human\x1b[O"}, true},
		{"split focus then text", []string{"\x1b[", "Ihuman"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := newPeerDelivery(peerTestMessage(time.Now()).To, time.Now)
			for _, chunk := range tc.chunks {
				d.admitInput([]byte(chunk))
				d.inputForwarded()
			}
			d.mu.Lock()
			owned := d.operatorDraft
			pending := len(d.replies.input)
			d.mu.Unlock()
			if owned != tc.owned {
				t.Fatalf("operator draft ownership=%t want=%t", owned, tc.owned)
			}
			if pending != 0 {
				t.Fatalf("complete input retained %d framing bytes", pending)
			}
		})
	}
}
