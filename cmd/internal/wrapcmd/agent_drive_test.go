package wrapcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/xianxu/pair/cmd/internal/agentdrive"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/ttyio"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

func driveFixture(t *testing.T) (*harnessSessionFake, *peerDelivery, *time.Time) {
	t.Helper()
	f := newHarnessSessionFake(t, "claude", true)
	t.Cleanup(f.close)
	f.output("\x1b[?2026h" + peerEmptyComposer() + "\x1b[?2026l")
	now := time.Now()
	b := couchmessage.Binding{Slot: "pair:1", Repository: "/fixture/.git", Scope: "fixture", Tag: "receiver", Session: "fixture-session", Nonce: "launch", Agent: "claude", Version: "fixture", PID: 1, Start: "fixture-start"}
	d := newPeerDelivery(b, func() time.Time { return now })
	f.proxy.peer = d
	d.driveProbe = f.proxy.agentDriveScreenLocked
	d.settled = true
	return f, d, &now
}

// writeDriveModelCommand paints the command acknowledgement before native Enter.
func writeDriveModelCommand(t *testing.T, f *harnessSessionFake, out *bytes.Buffer) {
	t.Helper()
	for i := 0; i < 2; i++ {
		f.proxy.dispatchAgentDrive(out)
	}
	f.output("\x1b[?2026h\x1b[2J" + slashComposerPaint(f.proxy.agentBasename, "/model") + "\x1b[?2026l")
	f.proxy.dispatchAgentDrive(out) // observe exact echo
	f.proxy.dispatchAgentDrive(out) // write the proved command Enter
	if !strings.HasSuffix(out.String(), "/model\r") {
		t.Fatalf("command entry lacked native Return: %q", out.String())
	}
}

func paintDrivePicker(t *testing.T, f *harnessSessionFake) {
	t.Helper()
	raw, err := os.ReadFile("testdata/drive/claude/2.1.299/session-before.screen")
	if err != nil {
		t.Fatal(err)
	}
	f.resize(120, 38)
	f.output("\x1b[?2026h\x1b[2J\x1b[H" + strings.ReplaceAll(string(raw), "\n", "\r\n") + "\x1b[?25l\x1b[?2026l")
}
func TestAgentDriveRejectsCachedApplyAfterRepaint(t *testing.T) {
	for _, changed := range []string{"\x1b[?2026h\x1b[2J\x1b[Hunknown screen\x1b[?2026l", "\x1b[?2026h\x1b[2J\x1b[Hpartial paint"} {
		t.Run(changed, func(t *testing.T) {
			f, d, _ := driveFixture(t)
			_, err := driveStart(t, d, agentdrive.Request{Model: &agentdrive.ModelRequest{Model: "opus 5.5", Effort: "medium"}})
			if err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			writeDriveModelCommand(t, f, &out)
			paintDrivePicker(t, f)
			f.proxy.dispatchAgentDrive(&out) // observe target, cache session apply
			d.mu.Lock()
			if len(d.drive.active.effects) != 1 || d.drive.active.effects[0].Key.Text != "s" {
				t.Fatalf("did not cache apply: %#v", d.drive.active.effects)
			}
			d.mu.Unlock()
			before := out.String()
			f.output(changed)
			f.proxy.dispatchAgentDrive(&out)
			if out.String() != before {
				t.Fatalf("stale effect wrote %q", out.String()[len(before):])
			}
			r, _ := d.agentDriveResult("test-drive")
			if !r.Terminal() || r.Outcome == agentdrive.OutcomeConfirmed {
				t.Fatalf("stale effect result %+v", r)
			}
		})
	}
}
func TestAgentDriveRevalidatesIdenticalPickerRepaint(t *testing.T) {
	f, d, _ := driveFixture(t)
	_, err := driveStart(t, d, agentdrive.Request{Model: &agentdrive.ModelRequest{Model: "opus 5.5", Effort: "medium"}})
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	writeDriveModelCommand(t, f, &out)
	paintDrivePicker(t, f)
	f.proxy.dispatchAgentDrive(&out)
	before := out.String()
	paintDrivePicker(t, f) // identical evidence, new terminal generation
	f.proxy.dispatchAgentDrive(&out)
	if out.String() != before+"s" {
		t.Fatalf("identical picker was not revalidated: %q", out.String())
	}
}
func driveStart(t *testing.T, d *peerDelivery, r agentdrive.Request) (agentdrive.Result, error) {
	t.Helper()
	o := d.agentDriveObservation()
	return d.startAgentDrive("test-drive", couchmessage.DriveStart{Request: r, Instance: o.Instance, Sequence: o.Sequence, Deadline: d.now().Add(time.Second)})
}
func TestAgentDriveWriterHasNoChromeAndSeparateEscape(t *testing.T) {
	f, d, _ := driveFixture(t)
	r, err := driveStart(t, d, agentdrive.Request{Keys: agentdrive.KeySequence{{Kind: agentdrive.KeyEsc}, {Kind: agentdrive.KeyEnter}}})
	if err != nil || r.Terminal() {
		t.Fatalf("admit %+v %v", r, err)
	}
	var out bytes.Buffer
	for i := 0; i < 4; i++ {
		f.proxy.dispatchAgentDrive(&out)
	}
	if string(out.Bytes()) != "\x1b\\\r" {
		t.Fatalf("named keys bypassed translation/ESC boundary: %q", out.Bytes())
	}
	if strings.Contains(out.String(), "Couch peer") {
		t.Fatal("message chrome inserted")
	}
	f.output("\x1b[?2026h" + peerEmptyComposer() + "\x1b[?2026l")
	f.proxy.dispatchAgentDrive(&out)
	r, err = d.agentDriveResult("test-drive")
	if err != nil || r.Outcome != agentdrive.OutcomeConfirmed {
		t.Fatalf("result %+v %v", r, err)
	}
	if f.proxy.automaticInput.phase != automaticInputIdle {
		t.Fatal("empty final composer did not release owner")
	}
}
func TestAgentDriveOperatorInputCancelsBeforeAnotherKey(t *testing.T) {
	f, d, now := driveFixture(t)
	_, err := driveStart(t, d, agentdrive.Request{Keys: agentdrive.KeySequence{{Kind: agentdrive.KeyText, Text: "first"}, {Kind: agentdrive.KeyWait, WaitMS: 500}, {Kind: agentdrive.KeyText, Text: "forbidden"}}})
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	f.proxy.dispatchAgentDrive(&out)
	f.proxy.admitStdin([]byte("operator"), false)
	*now = now.Add(time.Second)
	f.proxy.dispatchAgentDrive(&out)
	if strings.Contains(out.String(), "forbidden") {
		t.Fatal("drive wrote after operator input")
	}
	r, _ := d.agentDriveResult("test-drive")
	if r.Outcome == agentdrive.OutcomeConfirmed {
		t.Fatal("cancel became success")
	}
}
func TestAgentDriveRealInputLoopForwardsOperatorAndStopsDrive(t *testing.T) {
	f, d, _ := driveFixture(t)
	in, operator := io.Pipe()
	out := newDrainBuffer()
	done := make(chan struct{})
	go func() { f.proxy.translateStdinFrom(in, out, time.Millisecond); close(done) }()
	t.Cleanup(func() {
		operator.Close()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Error("input writer did not stop")
		}
	})
	_, err := driveStart(t, d, agentdrive.Request{Keys: agentdrive.KeySequence{{Kind: agentdrive.KeyText, Text: "first"}, {Kind: agentdrive.KeyWait, WaitMS: 1000}, {Kind: agentdrive.KeyText, Text: "forbidden"}}})
	if err != nil {
		t.Fatal(err)
	}
	if !waitFor(t, func() bool { return strings.Contains(string(out.Bytes()), "first") }) {
		t.Fatal("input loop did not service drive wake")
	}
	if _, err = operator.Write([]byte("operator")); err != nil {
		t.Fatal(err)
	}
	if !waitFor(t, func() bool { return strings.Contains(string(out.Bytes()), "operator") }) {
		t.Fatal("operator input was blocked")
	}
	if !waitFor(t, func() bool { r, _ := d.agentDriveResult("test-drive"); return r.Terminal() }) {
		t.Fatal("operator input did not cancel drive")
	}
	if strings.Contains(string(out.Bytes()), "forbidden") {
		t.Fatal("automatic input continued after operator")
	}
}
func TestAgentDriveRealInputLoopHandsOffToNormalDelivery(t *testing.T) {
	f, d, _ := driveFixture(t)
	d.now = time.Now
	in, operator := io.Pipe()
	out := newDrainBuffer()
	done := make(chan struct{})
	go func() { f.proxy.translateStdinFrom(in, out, time.Millisecond); close(done) }()
	t.Cleanup(func() {
		operator.Close()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Error("input writer did not stop")
		}
	})
	o := d.agentDriveObservation()
	r, err := d.startAgentDrive("handoff-drive", couchmessage.DriveStart{
		Request:  agentdrive.Request{Keys: agentdrive.KeySequence{{Kind: agentdrive.KeyText, Text: "/status"}, {Kind: agentdrive.KeyClearLine}}},
		Instance: o.Instance, Sequence: o.Sequence, Deadline: time.Now().Add(couchmessage.DriveBudget),
	})
	if err != nil || r.Terminal() {
		t.Fatalf("admit %+v: %v", r, err)
	}
	if !waitFor(t, func() bool { return strings.Contains(string(out.Bytes()), "/status\x15") }) {
		t.Fatal("key drive did not use input writer")
	}
	f.output("\x1b[?2026h" + peerEmptyComposer() + "\x1b[?2026l")
	if !waitFor(t, func() bool { r, _ := d.agentDriveResult("handoff-drive"); return r.Terminal() }) {
		t.Fatal("key drive did not finish on fresh empty composer")
	}
	r, _ = d.agentDriveResult("handoff-drive")
	if r.Outcome != agentdrive.OutcomeConfirmed {
		t.Fatalf("drive %+v", r)
	}
	m := peerTestMessage(time.Now())
	m.To = d.binding
	m.ID = "after-key-drive"
	m.Deadline = time.Now().Add(couchmessage.DeliveryTimeout)
	if err := d.reserve(m.ID, d.agentDriveObservation().Sequence); err != nil {
		t.Fatal(err)
	}
	if err := d.enqueue(m); err != nil {
		t.Fatal(err)
	}
	pasted := func() bool { return strings.Contains(string(out.Bytes()), peerEnvelope(m)) }
	if !waitFor(t, func() bool {
		r := d.receipt()
		return pasted() || r.Status.Terminal() || strings.Contains(r.Detail, "waiting for previous automatic input to clear")
	}) || !pasted() {
		t.Fatalf("normal delivery wedged after key drive: %+v", d.receipt())
	}
	peerRender(f, d, "\x1b[?2026h"+peerFilledComposer("[Pasted text #1 +3 lines]")+"\x1b[?2026l")
	if !waitFor(t, func() bool { return bytes.HasSuffix(out.Bytes(), []byte("\r")) }) {
		t.Fatalf("normal delivery did not submit: %+v", d.receipt())
	}
	peerRender(f, d, "\x1b[?2026h"+peerEmptyComposer()+"\x1b[?2026l")
	if !waitFor(t, func() bool { return d.receipt().Status.Terminal() }) {
		t.Fatal("normal delivery did not confirm")
	}
	if r := d.receipt(); r.Status != couchmessage.Submitted {
		t.Fatalf("normal delivery failed after drive: %+v", r)
	}
}
func TestAgentDriveAdmissionProofAndBusyRefusal(t *testing.T) {
	_, d, _ := driveFixture(t)
	o := d.agentDriveObservation()
	s := couchmessage.DriveStart{Request: agentdrive.Request{Keys: agentdrive.KeySequence{{Kind: agentdrive.KeyEnter}}}, Deadline: d.now().Add(time.Second), Instance: o.Instance, Sequence: o.Sequence}
	s.Instance = "replacement"
	r, err := d.startAgentDrive("test-drive", s)
	if err != nil || r.Outcome != agentdrive.OutcomeRefused {
		t.Fatalf("stale instance %+v %v", r, err)
	}
	s.Instance = o.Instance
	d.settled = false
	r, err = d.startAgentDrive("busy-drive", s)
	if err != nil || r.Outcome != agentdrive.OutcomeRefused {
		t.Fatalf("busy %+v %v", r, err)
	}
}

type driveShortWriter struct{}

type driveContextWriter struct{ *ttyio.Fake }

func (w driveContextWriter) Write(p []byte) (int, error) {
	return w.WriteContext(context.Background(), p)
}

func TestAgentDriveBlockedWriteCanceledByOperator(t *testing.T) {
	f, d, _ := driveFixture(t)
	_, err := driveStart(t, d, agentdrive.Request{Keys: agentdrive.KeySequence{{Kind: agentdrive.KeyText, Text: "automatic"}}})
	if err != nil {
		t.Fatal(err)
	}
	w := driveContextWriter{ttyio.NewFake()}
	w.Enqueue(ttyio.WriteStep{Block: make(chan struct{})})
	done := make(chan struct{})
	go func() { f.proxy.dispatchAgentDrive(w); close(done) }()
	select {
	case <-w.Started():
	case <-time.After(time.Second):
		t.Fatal("write not attempted")
	}
	f.proxy.admitStdin([]byte("operator"), false)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("operator cancellation did not release writer")
	}
	if len(w.Bytes()) != 0 {
		t.Fatalf("canceled write accepted bytes %q", w.Bytes())
	}
	r, _ := d.agentDriveResult("test-drive")
	if !r.Terminal() || r.Outcome == agentdrive.OutcomeConfirmed {
		t.Fatalf("canceled write %+v", r)
	}
	f.proxy.dispatchAgentDrive(w)
	if w.Calls() != 1 {
		t.Fatal("canceled write replayed")
	}
}

func (driveShortWriter) Write(p []byte) (int, error) { return len(p) / 2, io.ErrShortWrite }
func TestAgentDrivePartialWriteNeverReplays(t *testing.T) {
	f, d, _ := driveFixture(t)
	_, err := driveStart(t, d, agentdrive.Request{Keys: agentdrive.KeySequence{{Kind: agentdrive.KeyText, Text: "never replay"}}})
	if err != nil {
		t.Fatal(err)
	}
	f.proxy.dispatchAgentDrive(driveShortWriter{})
	var late bytes.Buffer
	f.proxy.dispatchAgentDrive(&late)
	r, _ := d.agentDriveResult("test-drive")
	if r.Outcome != agentdrive.OutcomeUnconfirmed || late.Len() != 0 {
		t.Fatalf("%+v late=%q", r, late.String())
	}
}
func TestAgentDriveRawSubmitWithoutRemapOpensTurn(t *testing.T) {
	f, d, _ := driveFixture(t)
	f.proxy.ttyProfile = nil
	f.proxy.lifecycleEvents = make(chan TurnObservation, 8)
	_, err := driveStart(t, d, agentdrive.Request{Keys: agentdrive.KeySequence{{Kind: agentdrive.KeySubmit}, {Kind: agentdrive.KeyText, Text: "forbidden"}}})
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	f.proxy.dispatchAgentDrive(&out)
	select {
	case event := <-f.proxy.lifecycleEvents:
		f.proxy.processLifecycleObservation(event)
	default:
		t.Fatal("raw submit bypassed ordinary turn accounting")
	}
	if !f.proxy.turnActive.Load() {
		t.Fatal("submission did not open a turn")
	}
	f.proxy.dispatchAgentDrive(&out)
	if strings.Contains(out.String(), "forbidden") {
		t.Fatal("drive continued into active turn")
	}
}
func TestAgentDriveEndpointBindingAndRetainedIdentity(t *testing.T) {
	_, d, _ := driveFixture(t)
	o := d.agentDriveObservation()
	start := couchmessage.DriveStart{Request: agentdrive.Request{Keys: agentdrive.KeySequence{{Kind: agentdrive.KeyEnter}}}, Deadline: d.now().Add(time.Second), Instance: o.Instance, Sequence: o.Sequence}
	request := couchmessage.EndpointRequest{Op: "drive-start", ID: "test-drive", Binding: d.binding, Drive: &start}
	raw, _ := json.Marshal(request)
	answer, err := d.handleEndpoint(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	var response couchmessage.EndpointResponse
	if err = json.Unmarshal(answer, &response); err != nil {
		t.Fatal(err)
	}
	if response.DriveResult == nil || response.DriveResult.ID != request.ID {
		t.Fatalf("wrong result %+v", response)
	}
	changed := start
	changed.Request = agentdrive.Request{Keys: agentdrive.KeySequence{{Kind: agentdrive.KeySubmit}}}
	if _, err = d.startAgentDrive(request.ID, changed); err == nil {
		t.Fatal("reused ID changed input")
	}
}
