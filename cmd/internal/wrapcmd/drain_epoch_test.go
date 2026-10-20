package wrapcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"strings"
	"testing"
	"time"
)

func TestDrainEpochCountedLifetime(t *testing.T) {
	for _, cycles := range []int{1000, 2000} {
		t.Run(fmt.Sprint(cycles), func(t *testing.T) {
			f := newHarnessSessionFake(t, "claude", true)
			defer f.close()
			f.output("\x1b[?2004h" + strings.Replace(claudeLiveComposerPaint(), "alpha", "", 1) + "\x1b[21;3H")
			p := f.proxy
			d := newDrainTestDelivery(couchmessage.Binding{Slot: "pair:1"}, time.Now)
			p.peer = d
			d.drainProxy = p
			peak := 0
			var out bytes.Buffer
			for generation := 1; generation <= cycles; generation++ {
				interrupt := fmt.Sprintf("interrupt-%d", generation)
				notice := fmt.Sprintf("notice-%d", generation)
				p.inputAdmission.Lock()
				p.drainTurn++
				p.turnActive.Store(generation%2 == 0)
				p.inputAdmission.Unlock()
				observed := d.observeDrain()
				if _, err := d.startDrainInterrupt(interrupt, testDrainFenceAt(d, couchmessage.DrainInterruptControl, uint64(generation)), observed); err != nil {
					t.Fatalf("generation %d: %v", generation, err)
				}
				p.executeDrainInterrupt(&out)
				for _, id := range []string{notice, interrupt} {
					kind := couchmessage.DrainNoticeControl
					if id == interrupt {
						kind = couchmessage.DrainInterruptControl
					}
					r := d.revokeDrain(id, testDrainFenceAt(d, kind, uint64(generation)))
					want := "not-committed"
					if kind == couchmessage.DrainInterruptControl && generation%2 == 0 {
						want = "already-committed"
					}
					if r.Outcome != want {
						t.Fatalf("generation %d cannot cancel: %+v", generation, r)
					}
				}
				residue := len(d.drainInterrupts) + len(d.drainRevokes)
				if residue > peak {
					peak = residue
				}
			}
			if peak > 3 {
				t.Fatalf("lifetime residue grew with cycles: peak=%d cycles=%d", peak, cycles)
			}
			if out.Len() != cycles/2 || bytes.Count(out.Bytes(), []byte{0x1b}) != cycles/2 {
				t.Fatalf("input effects=%d want=%d", out.Len(), cycles/2)
			}
			if d.drainGeneration.View().Epoch != uint64(cycles) {
				t.Fatal("watermark did not track all generations")
			}
			t.Logf("cycles=%d peak current outcomes=%d final=%d effects=%d", cycles, peak, len(d.drainInterrupts)+len(d.drainRevokes), out.Len())
		})
	}
}

func newDrainTestDelivery(binding couchmessage.Binding, now func() time.Time) *peerDelivery {
	d := newPeerDelivery(binding, now)
	d.drainNamespace, _ = couchmessage.DrainNamespaceKey("/isolated-drain")
	return d
}
func testDrainFence(d *peerDelivery, kind couchmessage.DrainControlKind) couchmessage.DrainFence {
	epoch := d.drainGeneration.View().Epoch
	if epoch == 0 {
		epoch = 1
	}
	return testDrainFenceAt(d, kind, epoch)
}
func testDrainFenceAt(d *peerDelivery, kind couchmessage.DrainControlKind, epoch uint64) couchmessage.DrainFence {
	return couchmessage.DrainFence{Namespace: d.drainNamespace, Generation: fmt.Sprintf("%08x-0000-4000-8000-000000000001", epoch), Epoch: epoch, Instance: d.drive.instance, Kind: kind}
}

func TestDrainEpochReadOnlyAndAliases(t *testing.T) {
	p := &proxy{}
	d := newDrainTestDelivery(couchmessage.Binding{Slot: "pair:1"}, time.Now)
	p.peer = d
	d.drainProxy = p
	before := d.observeDrain()
	future := testDrainFenceAt(d, couchmessage.DrainInterruptControl, 9)
	if _, err := d.drainInterruptStatus("future", future); err == nil {
		t.Fatal("future status acquired authority")
	}
	if after := d.observeDrain(); after.Epoch != before.Epoch || after.Generation != before.Generation {
		t.Fatal("status advanced watermark")
	}
	fence := testDrainFence(d, couchmessage.DrainInterruptControl)
	if _, err := d.startDrainInterrupt("aliased", fence, before); err != nil {
		t.Fatal(err)
	}
	notice := fence
	notice.Kind = couchmessage.DrainNoticeControl
	if err := d.reserveDrain("aliased", d.sequence, notice); err == nil {
		t.Fatal("control ID aliased across kinds")
	}
	if err := d.reserve("aliased", d.sequence); err == nil {
		t.Fatal("untyped reservation aliased drain control")
	}
	snapshot := d.drainGeneration.View()
	conflict := fence
	conflict.Generation = "00000002-0000-4000-8000-000000000001"
	if _, err := d.startDrainInterrupt("other", conflict, before); err == nil {
		t.Fatal("conflicting UUID reused epoch")
	}
	otherns := fence
	otherns.Namespace = strings.Repeat("a", 64)
	if _, err := d.startDrainInterrupt("other", otherns, before); err == nil {
		t.Fatal("foreign namespace acquired authority")
	}
	if d.drainGeneration.View() != snapshot {
		t.Fatal("rejected aliases mutated current generation")
	}
}

func TestDrainEpochDelayedCallbackAndQuarantine(t *testing.T) {
	f := newHarnessSessionFake(t, "claude", true)
	defer f.close()
	f.output("\x1b[?2004h" + strings.Replace(claudeLiveComposerPaint(), "alpha", "", 1) + "\x1b[21;3H")
	p := f.proxy
	d := newDrainTestDelivery(peerTestMessage(time.Now()).To, time.Now)
	p.peer = d
	d.drainProxy = p
	p.turnActive.Store(true)
	p.drainTurn = 1
	oldFence := testDrainFence(d, couchmessage.DrainInterruptControl)
	if _, err := d.startDrainInterrupt("old", oldFence, d.observeDrain()); err != nil {
		t.Fatal(err)
	}
	callback := d.drainInterrupt
	if r := d.revokeDrain("old", oldFence); r.Outcome != "not-committed" {
		t.Fatal(r)
	}
	p.automaticInput = automaticInputTransaction{phase: automaticInputNeedsOperator, residueCause: couchmessage.AutomaticInputStallUnsubmittedComposer}
	quarantine := p.automaticInput
	next := testDrainFenceAt(d, couchmessage.DrainInterruptControl, 2)
	if _, err := d.startDrainInterrupt("new", next, d.observeDrain()); err != nil {
		t.Fatal(err)
	}
	// A delayed callback object may outlive current-map retirement.
	callback.result.Outcome = "pending"
	var out bytes.Buffer
	p.inputAdmission.Lock()
	d.mu.Lock()
	p.executeDrainInterruptLocked(callback, &out)
	d.mu.Unlock()
	p.inputAdmission.Unlock()
	if out.Len() != 0 {
		t.Fatal("retired callback wrote Esc")
	}
	if p.automaticInput != quarantine {
		t.Fatal("epoch advance cleared input quarantine")
	}
	if _, err := d.drainInterruptStatus("old", oldFence); err == nil {
		t.Fatal("retired status revived old outcome")
	}
	d.releaseDrain("new", oldFence)
	if d.drainGeneration.View().Epoch != 2 {
		t.Fatal("stale release altered epoch")
	}
}

func TestDrainEpochRevokeReconcilesOnlyHandedSubmit(t *testing.T) {
	for _, phase := range []string{"paste-only", "submit-handed"} {
		t.Run(phase, func(t *testing.T) {
			f := newHarnessSessionFake(t, "claude", true)
			defer f.close()
			f.output("\x1b[?2004h" + strings.Replace(claudeLiveComposerPaint(), "alpha", "", 1) + "\x1b[21;3H")
			p := f.proxy
			d := newDrainTestDelivery(peerTestMessage(time.Now()).To, time.Now)
			p.peer = d
			d.drainProxy = p
			m := peerTestMessage(time.Now())
			m.Origin = couchmessage.OriginSystem
			m.DrainFence = testDrainFence(d, couchmessage.DrainNoticeControl)
			if err := d.reserveDrain(m.ID, 0, m.DrainFence); err != nil {
				t.Fatal(err)
			}
			if err := d.enqueue(m); err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			p.dispatchPeer(&out)
			event := func(e couchmessage.PeerDeliveryEvent) {
				p.inputAdmission.Lock()
				d.mu.Lock()
				p.advancePeer(e, &out)
				d.mu.Unlock()
				p.inputAdmission.Unlock()
			}
			if phase == "submit-handed" {
				event(couchmessage.PeerDeliveryEvent{Kind: couchmessage.PeerRenderObserved, Ready: true})
				if d.state.Phase != couchmessage.PeerDeliveryConfirming {
					t.Fatalf("submit fixture not handed: %+v", d.state)
				}
			}
			before := out.String()
			first := d.revokeDrain(m.ID, m.DrainFence)
			if first.Outcome != "unknown" || first.Receipt == nil || first.Receipt.Status != couchmessage.Delivering {
				t.Fatalf("premature revoke proof: %+v", first)
			}
			next := testDrainFenceAt(d, couchmessage.DrainInterruptControl, 2)
			if _, err := d.startDrainInterrupt("next", next, d.observeDrain()); err == nil {
				t.Fatal("unknown notice permitted next epoch")
			}
			if phase == "paste-only" {
				event(couchmessage.PeerDeliveryEvent{Kind: couchmessage.PeerRenderObserved, Ready: true})
				if r := d.revokeDrain(m.ID, m.DrainFence); r.Outcome != "unknown" {
					t.Fatalf("unhanded submit became known: %+v", r)
				}
				if p.automaticInput.phase == automaticInputIdle {
					t.Fatal("unsubmitted text lost quarantine")
				}
			} else {
				// Confirmation of a submit already handed is an observation, not input.
				event(couchmessage.PeerDeliveryEvent{Kind: couchmessage.PeerConfirmObserved, Ready: true, EmptyConfirmed: true})
				if r := d.revokeDrain(m.ID, m.DrainFence); r.Outcome != "already-committed" || r.Receipt == nil || r.Receipt.Status != couchmessage.Submitted {
					t.Fatalf("stale retained revoke did not refresh: %+v", r)
				}
				if _, err := d.startDrainInterrupt("next", next, d.observeDrain()); err != nil {
					t.Fatalf("known cancellation prevented next epoch: %v", err)
				}
			}
			if out.String() != before {
				t.Fatalf("reconciliation wrote input: %q -> %q", before, out.String())
			}
		})
	}
}

func TestDrainEpochRejectsRetiredNoticeAtEveryBoundary(t *testing.T) {
	for _, path := range []string{"reserve", "enqueue", "prepare", "finalize", "paste", "submit", "release", "untyped-reserve", "untyped-release"} {
		t.Run(path, func(t *testing.T) {
			f := newHarnessSessionFake(t, "claude", true)
			defer f.close()
			f.output("\x1b[?2004h" + strings.Replace(claudeLiveComposerPaint(), "alpha", "", 1) + "\x1b[21;3H")
			p := f.proxy
			b := couchmessage.Binding{Slot: "pair:1", Repository: "/repo", Scope: "scope", Tag: "tag", Session: "session", Nonce: "nonce", Agent: "claude", Version: "test", PID: 123, Start: "start"}
			d := newDrainTestDelivery(b, time.Now)
			p.peer = d
			d.drainProxy = p
			old := peerTestMessage(time.Now())
			old.To = b
			old.From = b
			old.Origin = couchmessage.OriginSystem
			old.DrainFence = testDrainFence(d, couchmessage.DrainNoticeControl)
			if err := d.reserveDrain(old.ID, 0, old.DrainFence); err != nil {
				t.Fatal(err)
			}
			if err := d.enqueue(old); err != nil {
				t.Fatal(err)
			}
			if r := d.revokeDrain(old.ID, old.DrainFence); r.Outcome != "not-committed" {
				t.Fatal(r)
			}
			next := testDrainFenceAt(d, couchmessage.DrainNoticeControl, 2)
			if err := d.reserveDrain("new-notice", 0, next); err != nil {
				t.Fatal(err)
			}
			before := d.drainGeneration.View()
			var out bytes.Buffer
			switch path {
			case "reserve":
				if err := d.reserveDrain(old.ID, 0, old.DrainFence); err == nil {
					t.Fatal("retired reserve admitted")
				}
			case "enqueue":
				if err := d.enqueue(old); err == nil {
					t.Fatal("retired enqueue admitted")
				}
			case "prepare":
				if err := d.prepareBatch(old); err == nil {
					t.Fatal("retired batch preparation admitted")
				}
			case "finalize":
				if err := d.finalizeBatch(old.ID, []couchmessage.Message{old}); err == nil {
					t.Fatal("retired batch finalization admitted")
				}
			case "paste", "submit":
				d.current = couchmessage.Receipt{Message: old, Status: couchmessage.Queued}
				d.state = couchmessage.PeerDeliveryState{}
				event := couchmessage.PeerDeliveryEvent{Kind: couchmessage.PeerComposerObserved, Ready: true}
				if path == "submit" {
					d.current.Status = couchmessage.Delivering
					d.state = couchmessage.PeerDeliveryState{Phase: couchmessage.PeerDeliveryPasted, BodyWritten: true}
					d.deliverBy = time.Now().Add(time.Minute)
					event.Kind = couchmessage.PeerRenderObserved
				}
				p.inputAdmission.Lock()
				d.mu.Lock()
				p.advancePeer(event, &out)
				d.mu.Unlock()
				p.inputAdmission.Unlock()
				if out.Len() != 0 {
					t.Fatal("retired notice wrote input")
				}
			case "release":
				d.releaseDrain("new-notice", old.DrainFence)
			case "untyped-reserve":
				if err := d.reserve("new-notice", 0); err == nil {
					t.Fatal("untyped reserve aliased typed control")
				}
			case "untyped-release":
				raw, err := json.Marshal(couchmessage.EndpointRequest{Op: "release", Binding: b, ID: "new-notice"})
				if err != nil {
					t.Fatal(err)
				}
				if _, err := d.handleEndpoint(context.Background(), raw); err != nil {
					t.Fatal(err)
				}
			}
			if d.drainGeneration.View() != before {
				t.Fatal("retired/untyped callback changed current authority")
			}
			if d.reservation != "new-notice" || d.reservationFence != next {
				t.Fatal("retired/untyped callback released new reservation")
			}
		})
	}
}
func TestDrainEpochUnsupportedObservationCannotInterrupt(t *testing.T) {
	p := &proxy{}
	d := newDrainTestDelivery(couchmessage.Binding{Slot: "pair:1"}, time.Now)
	p.peer = d
	d.drainProxy = p
	o := d.observeDrain()
	o.Protocol = 1
	o.Active = true
	o.Complete = true
	o.ComposerKnown = true
	r, err := d.startDrainInterrupt("unsupported", testDrainFence(d, couchmessage.DrainInterruptControl), o)
	if err != nil || r.Outcome != "refused" {
		t.Fatalf("%+v %v", r, err)
	}
	var out bytes.Buffer
	p.executeDrainInterrupt(&out)
	if out.Len() != 0 {
		t.Fatal("unsupported observer granted input")
	}
}
