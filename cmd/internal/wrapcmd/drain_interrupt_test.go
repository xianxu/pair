package wrapcmd

import (
	"bytes"
	"context"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"io"

	"strings"
	"testing"
	"time"
)

func TestDrainInterruptCannotHitCheckpointTurn(t *testing.T) {
	for _, change := range []string{"same", "turn", "operator", "picker", "automatic", "unsupported", "buffered", "image", "ended", "instance"} {
		t.Run(change, func(t *testing.T) {
			f := newHarnessSessionFake(t, "claude", true)
			defer f.close()
			f.output("\x1b[?2004h" + strings.Replace(claudeLiveComposerPaint(), "alpha", "", 1) + "\x1b[21;3H")
			p := f.proxy
			d := newPeerDelivery(peerTestMessage(time.Now()).To, time.Now)
			p.peer = d
			d.drainProxy = p
			p.turnActive.Store(true)
			p.drainTurn = 1
			observed := d.observeDrain()
			if _, err := d.startDrainInterrupt("drain-one", observed); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "turn":
				p.drainTurn = 2
			case "operator":
				human, _ := d.admitInput([]byte("x"))
				d.inputForwarded(human, false)
			case "picker":
				p.pickerActive.Store(true)
			case "automatic":
				p.automaticInput.phase = automaticInputPeer
			case "unsupported":
				p.ttyProfile = nil
			case "buffered":
				d.inputBuffered = true
			case "image":
				d.image = true
			case "ended":
				p.turnActive.Store(false)
			case "instance":
				d.drive.instance = "replacement"
			}
			var out bytes.Buffer
			p.executeDrainInterrupt(&out)
			want := ""
			if change == "same" {
				want = "\x1b"
			}
			if out.String() != want {
				t.Fatalf("Esc affected replacement/unsafe turn: %q", out.String())
			}
			first, _ := d.drainInterruptStatus("drain-one")
			_, err := d.startDrainInterrupt("drain-one", observed)
			if err != nil {
				t.Fatal(err)
			}
			p.executeDrainInterrupt(&out)
			again, _ := d.drainInterruptStatus("drain-one")
			if out.String() != want || first != again {
				t.Fatalf("duplicate rewrote input: %q %+v %+v", out.String(), first, again)
			}
		})
	}
}
func TestDrainCancelRevokesNotice(t *testing.T) {
	for _, phase := range []string{"before-reserve", "reserved", "queued", "pasted"} {
		t.Run(phase, func(t *testing.T) {
			f := newHarnessSessionFake(t, "claude", true)
			defer f.close()
			f.output("\x1b[?2004h" + strings.Replace(claudeLiveComposerPaint(), "alpha", "", 1) + "\x1b[21;3H")
			p := f.proxy
			d := newPeerDelivery(peerTestMessage(time.Now()).To, time.Now)
			p.peer = d
			d.drainProxy = p
			m := peerTestMessage(time.Now())
			var out bytes.Buffer
			if phase != "before-reserve" {
				if err := d.reserve(m.ID, 0); err != nil {
					t.Fatal(err)
				}
			}
			if phase == "queued" || phase == "pasted" {
				if err := d.enqueue(m); err != nil {
					t.Fatal(err)
				}
			}
			if phase == "pasted" {
				p.dispatchPeer(&out)
			}
			before := out.String()
			r := d.revokeDrain(m.ID)
			want := "not-committed"
			if phase == "pasted" {
				want = "already-committed"
			}
			if r.Outcome != want {
				t.Fatalf("%+v", r)
			}
			if again := d.revokeDrain(m.ID); again.Outcome != r.Outcome {
				t.Fatalf("not retained: %+v", again)
			}
			p.dispatchPeer(&out)
			if out.String() != before {
				t.Fatalf("revoked notice wrote input: %q -> %q", before, out.String())
			}
			if err := d.reserve(m.ID, d.sequence); err == nil {
				t.Fatal("revoked identity reusable")
			}
			if r.Outcome == "already-committed" && r.Receipt == nil {
				t.Fatal("lost retained outcome")
			}
		})
	}
}

func TestDrainInterruptLifecycleFenceAndUnknownWrite(t *testing.T) {
	f := newHarnessSessionFake(t, "claude", true)
	defer f.close()
	f.output("\x1b[?2004h" + strings.Replace(claudeLiveComposerPaint(), "alpha", "", 1) + "\x1b[21;3H")
	p := f.proxy
	d := newPeerDelivery(peerTestMessage(time.Now()).To, time.Now)
	p.peer = d
	d.drainProxy = p
	p.processLifecycleObservation(TurnObservation{Kind: ObservationUserSubmission})
	observed := d.observeDrain()
	if !observed.Active || !observed.ComposerKnown || !observed.Complete {
		t.Fatalf("bad fixture: %+v", observed)
	}
	if _, err := d.startDrainInterrupt("old-turn", observed); err != nil {
		t.Fatal(err)
	}
	p.processLifecycleObservation(TurnObservation{Kind: ObservationNativeCompletion})
	p.processLifecycleObservation(TurnObservation{Kind: ObservationUserSubmission})
	var out bytes.Buffer
	p.executeDrainInterrupt(&out)
	if out.Len() != 0 {
		t.Fatalf("Esc to checkpoint turn: %q", out.String())
	}
	r, _ := d.drainInterruptStatus("old-turn")
	if r.Outcome != "skipped" {
		t.Fatalf("%+v", r)
	}
	next := d.observeDrain()
	if next.Turn <= observed.Turn {
		t.Fatal("logical turn fence did not advance")
	}
	if _, err := d.startDrainInterrupt("uncertain-write", next); err != nil {
		t.Fatal(err)
	}
	w := &drainFailWriter{}
	p.executeDrainInterrupt(w)
	p.executeDrainInterrupt(w)
	r, _ = d.drainInterruptStatus("uncertain-write")
	if r.Outcome != "unknown" || w.calls != 1 {
		t.Fatalf("unknown write retried: %+v calls=%d", r, w.calls)
	}
}

type drainFailWriter struct{ calls int }

func (w *drainFailWriter) Write(p []byte) (int, error) { w.calls++; return 0, io.ErrClosedPipe }

func TestDrainEndpointRealTransportAndRevoke(t *testing.T) {
	f := newHarnessSessionFake(t, "claude", true)
	defer f.close()
	f.output("\x1b[?2004h" + strings.Replace(claudeLiveComposerPaint(), "alpha", "", 1) + "\x1b[21;3H")
	p := f.proxy
	d := newPeerDelivery(peerTestMessage(time.Now()).To, time.Now)
	p.peer = d
	d.drainProxy = p
	// Fixture's toy bindings deliberately aren't endpoint-valid; use a complete one.
	d.binding = couchmessage.Binding{Slot: "pair:1", Repository: "/repo", Scope: "scope", Tag: "tag", Session: "session", Nonce: "nonce", Agent: "claude", Version: "test", PID: 123, Start: "start"}
	namespace := t.TempDir()
	socket, err := couchmessage.EndpointSocket(namespace, d.binding)
	if err != nil {
		t.Fatal(err)
	}
	server, err := couchmessage.StartServer(context.Background(), socket, d.handleEndpoint)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	e := couchmessage.RemoteEndpoint{Namespace: namespace, Binding: d.binding}
	o, err := e.ObserveDrain(context.Background())
	if err != nil || !o.Idle {
		t.Fatalf("%+v %v", o, err)
	}
	r, err := e.InterruptDrain(context.Background(), "idle-drain", o)
	if err != nil || r.Outcome != "pending" {
		t.Fatalf("%+v %v", r, err)
	}
	var out bytes.Buffer
	p.executeDrainInterrupt(&out)
	r, err = e.DrainInterruptStatus(context.Background(), "idle-drain")
	if err != nil || r.Outcome != "skipped" || out.Len() != 0 {
		t.Fatalf("%+v %v %q", r, err, out.String())
	}
	revoked, err := e.RevokeDrain(context.Background(), "notice-one")
	if err != nil || revoked.Outcome != "not-committed" {
		t.Fatalf("%+v %v", revoked, err)
	}
	if err = e.Reserve(context.Background(), "notice-one", o.Input); err == nil {
		t.Fatal("remote revocation reused")
	}
}

func TestDrainInterruptRequiresPositiveOriginalEvidence(t *testing.T) {
	safe := couchmessage.DrainObservation{Instance: "wrapper", Turn: 1, Active: true, Complete: true, ComposerKnown: true}
	for _, change := range []string{"inactive", "picker", "automatic", "incomplete", "unknown-composer"} {
		t.Run(change, func(t *testing.T) {
			original := safe
			switch change {
			case "inactive":
				original.Active = false
			case "picker":
				original.Picker = true
			case "automatic":
				original.AutomaticInput = true
			case "incomplete":
				original.Complete = false
			case "unknown-composer":
				original.ComposerKnown = false
			}
			outcome, _ := ReduceDrainInterrupt(original, safe)
			if outcome == "interrupted" {
				t.Fatal("Esc granted without positive original evidence")
			}
		})
	}
}

func TestDrainCancelRevokesInterrupt(t *testing.T) {
	for _, phase := range []string{"not-started", "pending", "interrupted", "unknown"} {
		t.Run(phase, func(t *testing.T) {
			f := newHarnessSessionFake(t, "claude", true)
			defer f.close()
			f.output("\x1b[?2004h" + strings.Replace(claudeLiveComposerPaint(), "alpha", "", 1) + "\x1b[21;3H")
			p := f.proxy
			d := newPeerDelivery(peerTestMessage(time.Now()).To, time.Now)
			p.peer = d
			d.drainProxy = p
			p.turnActive.Store(true)
			p.drainTurn = 1
			observed := d.observeDrain()
			var out bytes.Buffer
			if phase != "not-started" {
				if _, err := d.startDrainInterrupt("stop-one", observed); err != nil {
					t.Fatal(err)
				}
			}
			if phase == "interrupted" {
				p.executeDrainInterrupt(&out)
			}
			if phase == "unknown" {
				p.executeDrainInterrupt(&drainFailWriter{})
			}
			before := out.String()
			revoked := d.revokeDrain("stop-one")
			want := "not-committed"
			if phase == "interrupted" {
				want = "already-committed"
			}
			if phase == "unknown" {
				want = "unknown"
			}
			if revoked.Outcome != want {
				t.Fatalf("%s: %+v", phase, revoked)
			}
			if _, err := d.startDrainInterrupt("stop-one", observed); err != nil {
				t.Fatal(err)
			}
			p.executeDrainInterrupt(&out)
			if out.String() != before {
				t.Fatalf("late canceled interrupt wrote Esc: %q -> %q", before, out.String())
			}
			if again := d.revokeDrain("stop-one"); again.Outcome != want {
				t.Fatalf("unstable revoke: %+v", again)
			}
		})
	}
}

func TestDrainNoticeWrapperInstanceFence(t *testing.T) {
	for _, phase := range []string{"enqueue", "paste", "batch", "effect-paste", "effect-submit"} {
		t.Run(phase, func(t *testing.T) {
			f := newHarnessSessionFake(t, "claude", true)
			defer f.close()
			f.output("\x1b[?2004h" + strings.Replace(claudeLiveComposerPaint(), "alpha", "", 1) + "\x1b[21;3H")
			p := f.proxy
			d := newPeerDelivery(peerTestMessage(time.Now()).To, time.Now)
			p.peer = d
			d.drainProxy = p
			d.binding = couchmessage.Binding{Slot: "pair:1", Repository: "/repo", Scope: "scope", Tag: "tag", Session: "session", Nonce: "nonce", Agent: "claude", Version: "test", PID: 123, Start: "start"}
			m := peerTestMessage(time.Now())
			m.To = d.binding
			m.From = d.binding
			m.From.Slot = "ops:0"
			m.Origin = couchmessage.OriginSystem
			m.DrainInstance = d.drive.instance
			if phase == "enqueue" {
				d.drive = newDriveMailbox()
			}
			if err := d.reserve(m.ID, 0); err != nil {
				t.Fatal(err)
			}
			var err error
			if phase == "batch" {
				err = d.prepareBatch(m)
			} else {
				err = d.enqueue(m)
			}
			if phase == "enqueue" {
				if err == nil {
					t.Fatal("same-binding replacement accepted old drain notice")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if phase == "batch" {
				d.batch.phase = couchmessage.BatchReady
				d.drive = newDriveMailbox()
				if err := d.finalizeBatch(m.ID, []couchmessage.Message{m}); err == nil {
					t.Fatal("replacement finalized old-instance batch")
				}
			}
			d.drive = newDriveMailbox()
			var out bytes.Buffer
			if phase == "effect-paste" || phase == "effect-submit" {
				event := couchmessage.PeerDeliveryEvent{Kind: couchmessage.PeerComposerObserved, Ready: true}
				if phase == "effect-submit" {
					d.state = couchmessage.PeerDeliveryState{Phase: couchmessage.PeerDeliveryPasted, BodyWritten: true}
					d.current.Status = couchmessage.Delivering
					d.deliverBy = d.now().Add(time.Minute)
					event.Kind = couchmessage.PeerRenderObserved
				}
				p.inputAdmission.Lock()
				d.mu.Lock()
				p.advancePeer(event, &out)
				d.mu.Unlock()
				p.inputAdmission.Unlock()
			} else {
				p.dispatchPeer(&out)
			}
			if out.Len() != 0 {
				t.Fatalf("old drain notice reached replacement PTY: %q", out.String())
			}
			if !d.receipt().Status.Terminal() {
				t.Fatalf("instance mismatch left notice pending: %+v", d.receipt())
			}
		})
	}
}

func TestDrainRevokeReportsLeaseInstanceAcrossReplacement(t *testing.T) {
	f := newHarnessSessionFake(t, "claude", true)
	defer f.close()
	p := f.proxy
	binding := peerTestMessage(time.Now()).To
	original := newPeerDelivery(binding, time.Now)
	p.peer = original
	original.drainProxy = p
	observed := original.observeDrain()
	replacement := newPeerDelivery(binding, time.Now)
	p.peer = replacement
	replacement.drainProxy = p
	r := replacement.revokeDrain("old-notice")
	if r.Instance != replacement.drive.instance || r.Instance == observed.Instance {
		t.Fatalf("replacement absence masqueraded as original-wrapper proof: observed=%s result=%+v", observed.Instance, r)
	}
	if again := replacement.revokeDrain("old-notice"); again.Instance != r.Instance || again.Outcome != r.Outcome {
		t.Fatalf("retained revoke lost instance: %+v -> %+v", r, again)
	}
}
