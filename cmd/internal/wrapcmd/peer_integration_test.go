package wrapcmd

import (
	"bytes"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

func peerIntegrationFixture(t *testing.T) (*harnessSessionFake, *peerDelivery) {
	t.Helper()
	f := newHarnessSessionFake(t, "claude", true)
	t.Cleanup(f.close)
	f.output("\x1b[?2004h" + strings.Replace(claudeLiveComposerPaint(), "alpha", "", 1) + "\x1b[21;3H")
	d := newPeerDelivery(peerTestMessage(time.Now()).To, time.Now)
	d.session = &recordingPeerSink{}
	f.proxy.peer = d
	return f, d
}

// recordingPeerSink stands in for the wrapper's broker session.
type recordingPeerSink struct {
	mu               sync.Mutex
	updates, submits int
	last             couchmessage.Observation
	settles          []bool
}

func (s *recordingPeerSink) Settle(v bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.settles = append(s.settles, v)
}

func (s *recordingPeerSink) Update(o couchmessage.Observation) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updates++
	s.last = o
}
func (s *recordingPeerSink) Submit(o couchmessage.Observation) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.submits++
	s.last = o
}
func (s *recordingPeerSink) counts() (updates, submits int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.updates, s.submits
}
func peerSubmits(d *peerDelivery) int {
	_, n := d.session.(*recordingPeerSink).counts()
	return n
}
func peerIntegrationEnqueue(t *testing.T, d *peerDelivery) {
	t.Helper()
	d.mu.Lock()
	sequence := d.sequence
	d.mu.Unlock()
	m := peerTestMessage(d.now())
	m.Deadline = d.now().Add(couchmessage.DeliveryTimeout)
	if err := d.reserve(m.ID, sequence); err != nil {
		t.Fatal(err)
	}
	if err := d.enqueue(m); err != nil {
		t.Fatal(err)
	}
}

func TestPeerIntegrationErasedDraftAllowsDelivery(t *testing.T) {
	f, d := peerIntegrationFixture(t)
	now := time.Now()
	d.now = func() time.Time { return now }
	d.admitInput([]byte("human"))
	d.inputForwarded(true, false)
	f.output("\x1b[21;3Hhuman\x1b[21;8H")
	peerIntegrationEnqueue(t, d)
	now = now.Add(time.Second)
	var out bytes.Buffer
	f.proxy.dispatchPeer(&out)
	if out.Len() != 0 {
		t.Fatal("delivered into human text")
	}
	d.admitInput([]byte("\x7f\x7f\x7f\x7f\x7f"))
	d.inputForwarded(true, false)
	f.output("\x1b[21;3H\x1b[K\x1b[21;3H")
	now = now.Add(time.Second)
	f.proxy.dispatchPeer(&out)
	if !strings.Contains(out.String(), "[Couch peer from ") {
		t.Fatal("erased draft still blocks delivery")
	}
}

func TestPeerIntegrationClaudeSuggestionWaitsForRecentHumanInput(t *testing.T) {
	for _, human := range []bool{false, true} {
		t.Run(fmt.Sprint(human), func(t *testing.T) {
			f, d := peerIntegrationFixture(t)
			if human {
				d.admitInput([]byte("human\x01"))
				d.inputForwarded(true, false)
			}
			// Same faint composer and origin cursor as the operator's 2.1.286
			// capture. Even this paint cannot bypass the input settling interval.
			f.output("\x1b[21;3H\x1b[2mcheck if parley.nvim:0 replied\x1b[22m\x1b[21;3H")
			peerIntegrationEnqueue(t, d)
			var out bytes.Buffer
			f.proxy.dispatchPeer(&out)
			if human {
				if out.Len() != 0 {
					t.Fatalf("overwrote human draft: %q", out.String())
				}
			} else if !strings.Contains(out.String(), "[Couch peer from ") {
				t.Fatalf("suggestion blocked peer paste: %q", out.String())
			}
		})
	}
}

// The operator's input is already at the child but its repaint has not arrived.
// The earlier empty snapshot cannot authorize a paste into that new draft.
func TestPeerIntegrationUnrenderedOperatorInputBlocksPaste(t *testing.T) {
	f, d := peerIntegrationFixture(t)
	d.admitInput([]byte("human"))
	d.inputForwarded(true, false)
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

// An apparently blank composer cannot bypass just-forwarded input settling.
func TestPeerIntegrationSpacesThenHomeWaitsForSettling(t *testing.T) {
	f, d := peerIntegrationFixture(t)
	d.admitInput([]byte("   \x01"))
	d.inputForwarded(true, false)
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
			reset := peerSubmits(d) == 1
			if n := peerSubmits(d); n > 1 || reset != tc.reset {
				t.Fatalf("budget reset=%t want %t", reset, tc.reset)
			}
			d.mu.Lock()
			submissions := d.submissions
			d.mu.Unlock()
			if (tc.reset && submissions != 1) || (!tc.reset && submissions != 0) {
				t.Fatalf("human submission generation=%d reset=%t", submissions, tc.reset)
			}
		})
	}
}

func TestPeerIntegrationMenuConfirmationDoesNotResetAllowance(t *testing.T) {
	f, d := peerIntegrationFixture(t)
	f.proxy.pickerActive.Store(true)
	var out bytes.Buffer
	f.proxy.translateStdinFrom(strings.NewReader("\x1b\r"), &out, time.Millisecond)
	if peerSubmits(d) != 0 {
		t.Fatal("menu confirmation reset peer allowance")
	}
}

// A successful human write releases their draft, but the old empty screen is
// not evidence that the child has consumed the submission and is safe again.
func TestPeerIntegrationHumanSubmitNeedsFreshComposerEvidence(t *testing.T) {
	f, d := peerIntegrationFixture(t)
	d.admitInput([]byte("human\x1b\r"))
	// Exercise the same full-write accounting order as translateStdinFrom.
	d.humanSubmit()
	d.inputForwarded(true, false)
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
				human := d.admitInput([]byte(chunk))
				d.inputForwarded(human, false)
			}
			d.mu.Lock()
			owned := !d.lastInput.IsZero()
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

// The session sees every activity change and each genuine submission once;
// coalescing to ≤1 frame/s is the session client's job (couchmessage), so the
// PTY output path only records (#365).
func TestPeerOutputAndSubmitReachTheSession(t *testing.T) {
	f, d := peerIntegrationFixture(t)
	sink := d.session.(*recordingPeerSink)
	for i := 0; i < 1000; i++ {
		d.observeOutput([]byte("x"))
	}
	updates, submits := sink.counts()
	if updates != 1000 || submits != 0 {
		t.Fatalf("output: updates=%d submits=%d", updates, submits)
	}
	f.proxy.translateStdinFrom(strings.NewReader("human\x1b\r"), &bytes.Buffer{}, time.Millisecond)
	if _, submits = sink.counts(); submits != 1 {
		t.Fatalf("human send: submits=%d", submits)
	}
	sink.mu.Lock()
	last := sink.last
	sink.mu.Unlock()
	if last.Submission != 1 {
		t.Fatalf("submit carried %+v", last)
	}
}
