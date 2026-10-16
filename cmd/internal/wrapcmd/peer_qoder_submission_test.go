package wrapcmd

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

// A receiver owns its composer independently of the wrapper's receipt. It can
// drop a submit without repainting, like the observed native pending prompt.
type qoderReceiverFake struct {
	t                                   *testing.T
	f                                   *harnessSessionFake
	d                                   *peerDelivery
	now                                 time.Time
	wire                                bytes.Buffer
	body, paint                         string
	pastes, submits, accepted, acceptOn int
	reports                             []couchmessage.AutomaticInputStall
}

func newQoderReceiver(t *testing.T, acceptOn int) *qoderReceiverFake {
	t.Helper()
	r := &qoderReceiverFake{t: t, now: time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC), acceptOn: acceptOn}
	r.f = newHarnessSessionFake(t, "qoder", true)
	t.Cleanup(r.f.close)
	r.f.resize(94, 38)
	r.d = newPeerDelivery(peerTestMessage(r.now).To, func() time.Time { return r.now })
	r.f.proxy.peer = r.d
	r.f.proxy.automaticReport = func(report couchmessage.AutomaticInputStall, _ terminalSnapshot) {
		r.reports = append(r.reports, report)
	}
	peerRender(r.f, r.d, qoderRecorded(t, "1.1.71", "idle"))
	peerIntegrationEnqueue(t, r.d)
	return r
}
func (r *qoderReceiverFake) Write(p []byte) (int, error) {
	r.wire.Write(p)
	if bytes.HasPrefix(p, []byte("\x1b[200~")) {
		r.pastes++
		r.body = strings.TrimSuffix(strings.TrimPrefix(string(p), "\x1b[200~"), "\x1b[201~")
		r.paint = peerQoderPaint("*", strings.Split(r.body, "\n")...)
	} else if string(p) == "\r" {
		r.submits++
		if r.submits >= r.acceptOn && r.body != "" {
			r.accepted++
			r.body = ""
			r.paint = qoderRecorded(r.t, "1.1.71", "idle")
		}
	} else {
		r.t.Fatalf("unexpected writer bytes %q", p)
	}
	return len(p), nil
}
func (r *qoderReceiverFake) dispatch() {
	r.f.proxy.dispatchPeer(r)
	if r.paint != "" {
		paint := r.paint
		r.paint = ""
		peerRender(r.f, r.d, paint)
	}
}
func (r *qoderReceiverFake) firstSubmit() {
	r.dispatch()
	r.now = r.now.Add(PeerSubmitDelay)
	r.dispatch()
}

func TestPeerQoderDroppedSubmitRetriesOnlyKey(t *testing.T) {
	r := newQoderReceiver(t, 2)
	r.firstSubmit()
	if r.d.receipt().Status.Terminal() {
		t.Fatal("a dropped key was reported terminal")
	}
	r.now = r.now.Add(500 * time.Millisecond)
	r.dispatch()
	r.dispatch()
	if r.pastes != 1 || r.submits != 2 || r.accepted != 1 || r.d.receipt().Status != couchmessage.Submitted {
		t.Fatalf("paste=%d submit=%d accepted=%d receipt=%+v", r.pastes, r.submits, r.accepted, r.d.receipt())
	}
	for i := 0; i < 10; i++ {
		r.now = r.now.Add(time.Second)
		r.dispatch()
	}
	if r.pastes != 1 || r.submits != 2 || r.accepted != 1 {
		t.Fatal("confirmation repeated input")
	}
}
func TestPeerQoderTurnEdgeCannotConfirmOccupiedComposer(t *testing.T) {
	r := newQoderReceiver(t, 99)
	r.firstSubmit()
	r.f.proxy.turnActive.Store(true)
	peerRender(r.f, r.d, peerQoderPaint("*", strings.Split(r.body, "\n")...))
	r.dispatch()
	if r.d.receipt().Status != couchmessage.Delivering {
		t.Fatalf("unrelated turn confirmed pending input: %+v", r.d.receipt())
	}
}
func TestPeerQoderRetryRefusesChangedEnvelope(t *testing.T) {
	r := newQoderReceiver(t, 99)
	r.firstSubmit()
	peerRender(r.f, r.d, peerQoderPaint("*", strings.Split(r.body+" operator edit", "\n")...))
	r.now = r.now.Add(time.Second)
	r.dispatch()
	if r.submits != 1 || r.pastes != 1 {
		t.Fatal("retried altered composer")
	}
}
func TestPeerQoderResidueAlertsRefusesAndRecovers(t *testing.T) {
	r := newQoderReceiver(t, 99)
	r.firstSubmit()
	for i := 0; i < 6; i++ {
		r.now = r.now.Add(500 * time.Millisecond)
		r.dispatch()
	}
	r.now = r.now.Add(couchmessage.DeliveryTimeout)
	r.dispatch()
	if r.submits != 3 || r.d.receipt().Status != couchmessage.Indeterminate || len(r.reports) != 1 {
		t.Fatalf("submits=%d reports=%+v receipt=%+v", r.submits, r.reports, r.d.receipt())
	}
	next := peerTestMessage(r.now)
	next.ID = "blocked-after-residue"
	if err := r.d.reserve(next.ID, r.d.sequence); err != nil {
		t.Fatal(err)
	}
	if err := r.d.enqueue(next); err != nil {
		t.Fatal(err)
	}
	r.dispatch()
	r.d.reservation = "" // the production endpoint releases admission after enqueue
	if r.d.receipt().Status != couchmessage.Cancelled || !strings.Contains(r.d.receipt().Detail, "press Alt+Enter or clear it") || r.pastes != 1 || len(r.reports) != 1 {
		t.Fatalf("later delivery: %+v pastes=%d reports=%d", r.d.receipt(), r.pastes, len(r.reports))
	}
	// The existing render wake reconciles the terminal transaction, with no
	// fresh delivery required and no timer that could execute an expired body.
	peerRender(r.f, r.d, qoderRecorded(t, "1.1.71", "idle"))
	r.dispatch()
	if r.f.proxy.automaticInput.phase != automaticInputIdle {
		t.Fatal("empty render did not recover terminal lease")
	}
	next.ID = "healthy-after-empty"
	if err := r.d.reserve(next.ID, r.d.sequence); err != nil {
		t.Fatal(err)
	}
	if err := r.d.enqueue(next); err != nil {
		t.Fatal(err)
	}
	r.acceptOn = r.submits + 1
	r.firstSubmit()
	r.dispatch()
	if r.d.receipt().Status != couchmessage.Submitted || r.pastes != 2 || r.accepted != 1 {
		t.Fatalf("recovery: %+v", r.d.receipt())
	}
}

func TestQoderSubmissionEvidenceCapturedPendingEnvelope(t *testing.T) {
	raw, err := os.ReadFile("testdata/peer/qoder/1.1.71/pending-restart.raw")
	if err != nil {
		t.Fatal(err)
	}
	f := newHarnessSessionFake(t, "qoder", true)
	t.Cleanup(f.close)
	f.resize(94, 38)
	f.output("\x1b[2J\x1b[6;1H\x1b[?2026h" + string(raw) + "\x1b[?2026l\x1b[?25l\x1b[8;4H")
	expected := "[Couch peer from ops:0; delivery 44c73545-5fed-4759-9c68-fd4865c10085]\nCouch restarted and the hub is available. If your last report to the hub is still unanswered and relevant, resend it now. Otherwise do nothing."
	s := f.proxy.terminal.Snapshot()
	state, owned := qoderSubmissionEvidence(s, expected)
	if state != PeerComposerOccupied || !owned {
		text, known := peerComposerText("qoder", s)
		t.Fatalf("state=%v owned=%v known=%v text=%q", state, owned, known, text)
	}
	for _, changed := range []string{expected + "!", strings.Replace(expected, "Otherwise", "otherwise", 1), strings.Replace(expected, "available. If", "available.\nIf", 1)} {
		if _, owned := qoderSubmissionEvidence(s, changed); owned {
			t.Fatalf("different envelope matched: %q", changed)
		}
	}
	s.Complete = false
	if _, owned := qoderSubmissionEvidence(s, expected); owned {
		t.Fatal("partial frame matched")
	}
}

func TestPeerQoderRetryAdmissionGuards(t *testing.T) {
	for _, kind := range []string{"human", "picker", "partial frame", "unknown", "changed body", "too soon", "deadline"} {
		t.Run(kind, func(t *testing.T) {
			r := newQoderReceiver(t, 99)
			r.firstSubmit()
			r.now = r.now.Add(time.Second)
			switch kind {
			case "human":
				human, _ := r.d.admitInput([]byte("human"))
				r.d.inputForwarded(human, false)
			case "picker":
				r.f.proxy.pickerActive.Store(true)
			case "partial frame":
				peerRender(r.f, r.d, "\x1b[?2026h")
			case "unknown":
				peerRender(r.f, r.d, "\x1b[2J")
			case "changed body":
				peerRender(r.f, r.d, peerQoderPaint("*", strings.Split(r.body+"!", "\n")...))
			case "too soon":
				r.now = r.now.Add(-time.Second + 499*time.Millisecond)
			case "deadline":
				r.now = r.now.Add(couchmessage.DeliveryTimeout)
			}
			r.dispatch()
			if r.submits != 1 || r.pastes != 1 {
				t.Fatalf("%s authorized input: submits=%d pastes=%d", kind, r.submits, r.pastes)
			}
			if r.d.receipt().Status == couchmessage.Submitted {
				t.Fatal("unconsumed body was reported submitted")
			}
		})
	}
}
func TestPeerQoderMidTurnConsumptionConfirms(t *testing.T) {
	r := newQoderReceiver(t, 1)
	r.f.proxy.turnActive.Store(true)
	r.firstSubmit()
	r.dispatch()
	if r.accepted != 1 || r.d.receipt().Status != couchmessage.Submitted || r.pastes != 1 || r.submits != 1 {
		t.Fatalf("mid-turn receipt=%+v accepted=%d", r.d.receipt(), r.accepted)
	}
}
func TestPeerQoderRawOutputCannotConfirmStaleEmpty(t *testing.T) {
	r := newQoderReceiver(t, 99)
	r.dispatch()
	r.dispatch() // observe the owned paste before the pre-submit clear
	peerRender(r.f, r.d, qoderRecorded(t, "1.1.71", "idle"))
	r.now = r.now.Add(PeerSubmitDelay)
	r.dispatch()
	// Arrival sequence advances, but no post-submit rendering has happened.
	r.d.observeOutput([]byte("not rendered yet"))
	r.f.proxy.turnActive.Store(true)
	r.dispatch()
	if r.d.receipt().Status.Terminal() {
		t.Fatalf("raw arrival confirmed stale frame: %+v", r.d.receipt())
	}
	r.d.outputForwarded()
	peerRender(r.f, r.d, qoderRecorded(t, "1.1.71", "idle"))
	r.dispatch()
	if r.d.receipt().Status != couchmessage.Submitted {
		t.Fatalf("fresh empty did not confirm: %+v", r.d.receipt())
	}
}
func TestQoderSubmissionEvidenceRejectsUnsupportedText(t *testing.T) {
	for _, text := range []string{"hello\tworld", "hello\rworld", "hello\x1b[0m", "hello ", string([]byte{0xff})} {
		s := peerSnapshot(t, peerQoderPaint("*", text))
		if _, owned := qoderSubmissionEvidence(s, text); owned {
			t.Fatalf("unsupported input matched %q", text)
		}
	}
}
func FuzzQoderSubmissionEvidence(f *testing.F) {
	f.Add("hello", uint16(80), uint16(38))
	f.Add("\x1b[2J", uint16(0), uint16(65535))
	f.Fuzz(func(t *testing.T, envelope string, w, h uint16) {
		// Arbitrary malformed geometry is not authority, even when the model has
		// too few cells for it. The parser must decline rather than panic.
		s := terminalSnapshot{Width: int(w), Height: int(h), Complete: true}
		_, owned := qoderSubmissionEvidence(s, envelope)
		if owned {
			t.Fatal("malformed snapshot authorized retry")
		}
	})
}

func TestPeerQoderQuarantineRefusalPrecedesPickerHold(t *testing.T) {
	r := newQoderReceiver(t, 99)
	r.firstSubmit()
	r.now = r.now.Add(couchmessage.DeliveryTimeout)
	r.dispatch()
	next := peerTestMessage(r.now)
	next.ID = "quarantined-with-picker"
	if err := r.d.reserve(next.ID, r.d.sequence); err != nil {
		t.Fatal(err)
	}
	if err := r.d.enqueue(next); err != nil {
		t.Fatal(err)
	}
	r.f.proxy.pickerActive.Store(true)
	r.dispatch()
	if r.d.receipt().Status != couchmessage.Cancelled || !strings.Contains(r.d.receipt().Detail, "press Alt+Enter or clear it") || r.pastes != 1 {
		t.Fatalf("quarantine became a picker hold: %+v", r.d.receipt())
	}
}

func TestPeerQoderRecoveryAdviceRoutesToNativeSubmit(t *testing.T) {
	r := newQoderReceiver(t, 99)
	r.firstSubmit()
	r.now = r.now.Add(couchmessage.DeliveryTimeout)
	r.dispatch()
	next := peerTestMessage(r.now)
	next.ID = "operator-recovery-advice"
	if err := r.d.reserve(next.ID, r.d.sequence); err != nil {
		t.Fatal(err)
	}
	if err := r.d.enqueue(next); err != nil {
		t.Fatal(err)
	}
	r.dispatch()
	advice := r.d.receipt().Detail
	// Follow the actual instruction through the production input translator,
	// instead of proving merely that two strings agree about a key name.
	var key []byte
	switch {
	case strings.Contains(advice, "Alt+Enter"):
		key = []byte("\x1b\r")
	case strings.Contains(advice, "Enter"):
		key = []byte("\r")
	default:
		t.Fatalf("no supported recovery action in %q", advice)
	}
	translated, left, paste := r.f.proxy.translateChunk(key, false)
	if string(translated) != "\r" || len(left) != 0 || paste {
		t.Fatalf("advice %q emits %q, not a native submit", advice, translated)
	}
	r.acceptOn = r.submits + 1
	if _, err := r.Write(translated); err != nil {
		t.Fatal(err)
	}
	r.dispatch()
	r.dispatch()
	if r.accepted != 1 || r.f.proxy.automaticInput.phase != automaticInputIdle {
		t.Fatalf("manual recovery left input pending: accepted=%d phase=%v", r.accepted, r.f.proxy.automaticInput.phase)
	}
}
