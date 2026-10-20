package wrapcmd

import (
	"bytes"
	"testing"

	"github.com/xianxu/pair/cmd/internal/agentdrive"
)

func interruptRequest() agentdrive.Request { return agentdrive.Request{Interrupt: true} }

// workingTurn paints a real Claude 2.1.304 working-turn capture with the turn
// flag set, as during a hub-sent turn: the hub's message was submitted and the
// automatic-input lease waits for the composer to empty again.
func workingTurn(t *testing.T) (*harnessSessionFake, *peerDelivery) {
	t.Helper()
	f, d, _ := driveFixture(t)
	paintClaude2304(t, f, claude2304Screen(t, "working-shell"))
	f.proxy.turnActive.Store(true)
	f.proxy.automaticInput = automaticInputTransaction{phase: automaticInputAwaitingEmpty, renderFence: f.proxy.automaticRenderEpoch.Load()}
	d.settled = false
	return f, d
}

func TestAgentDriveInterruptWritesOneCtrlCAndConfirmsOnTurnEnd(t *testing.T) {
	f, d := workingTurn(t)
	if s := f.proxy.agentDriveScreen(); !s.Interruptible || s.Idle || s.Dismissable {
		t.Fatalf("working turn screen %+v", s)
	}
	if o := d.agentDriveObservation(); !o.Interruptible || o.Idle {
		t.Fatalf("working turn observation %+v", o)
	}
	if r, err := driveStart(t, d, interruptRequest()); err != nil || r.Terminal() {
		t.Fatalf("interrupt admission %+v %v", r, err)
	}
	var out bytes.Buffer
	for i := 0; i < 3; i++ {
		f.proxy.dispatchAgentDrive(&out)
	}
	if out.String() != "\x03" {
		t.Fatalf("interrupt bytes %q", out.String())
	}
	paintClaude2304(t, f, claude2304Screen(t, "working-shell"))
	f.proxy.dispatchAgentDrive(&out)
	if r, _ := d.agentDriveResult("test-drive"); r.Terminal() {
		t.Fatalf("a working frame ended the interrupt %+v", r)
	}
	f.proxy.turnActive.Store(false)
	f.output("\x1b[?2026h\x1b[2J" + peerEmptyComposer() + "\x1b[?2026l")
	f.proxy.dispatchAgentDrive(&out)
	if r, _ := d.agentDriveResult("test-drive"); r.Outcome != agentdrive.OutcomeConfirmed {
		t.Fatalf("turn ended %+v", r)
	}
	if out.String() != "\x03" {
		t.Fatalf("more than one Ctrl+C: %q", out.String())
	}
	// A second interrupt finds no working turn: two presses never exit.
	delete(d.drive.receipts, "test-drive")
	if r, err := driveStart(t, d, interruptRequest()); err != nil || r.Outcome != agentdrive.OutcomeRefused || r.Detail != agentdrive.InterruptRefusal {
		t.Fatalf("second interrupt %+v %v", r, err)
	}
}

func TestAgentDriveInterruptRefusesWithoutAWorkingTurn(t *testing.T) {
	for name, setup := range map[string]func(*harnessSessionFake, *peerDelivery){
		"idle composer": func(f *harnessSessionFake, d *peerDelivery) {
			f.proxy.turnActive.Store(false)
			f.proxy.automaticInput = automaticInputTransaction{}
			f.output("\x1b[?2026h\x1b[2J" + peerEmptyComposer() + "\x1b[?2026l")
			d.settled = true
		},
		"picker open":          func(f *harnessSessionFake, d *peerDelivery) { f.proxy.pickerActive.Store(true) },
		"peer input owns slot": func(f *harnessSessionFake, d *peerDelivery) { f.proxy.automaticInput.phase = automaticInputPeer },
		"operator typing":      func(f *harnessSessionFake, d *peerDelivery) { d.inputBuffered = true },
		"needs operator": func(f *harnessSessionFake, d *peerDelivery) {
			f.proxy.automaticInput.phase = automaticInputNeedsOperator
		},
	} {
		t.Run(name, func(t *testing.T) {
			f, d := workingTurn(t)
			setup(f, d)
			_, _ = driveStart(t, d, interruptRequest())
			var out bytes.Buffer
			for i := 0; i < 3; i++ {
				f.proxy.dispatchAgentDrive(&out)
			}
			if out.Len() != 0 {
				t.Fatalf("Ctrl+C written with %s: %q", name, out.String())
			}
		})
	}
}

// TestAgentDriveInterruptRechecksBeforeTheWrite: the turn ends between
// admission and the writer; nothing is written.
func TestAgentDriveInterruptRechecksBeforeTheWrite(t *testing.T) {
	f, d := workingTurn(t)
	if r, err := driveStart(t, d, interruptRequest()); err != nil || r.Terminal() {
		t.Fatalf("interrupt admission %+v %v", r, err)
	}
	f.proxy.turnActive.Store(false)
	f.output("\x1b[?2026h\x1b[2J" + peerEmptyComposer() + "\x1b[?2026l")
	var out bytes.Buffer
	for i := 0; i < 3; i++ {
		f.proxy.dispatchAgentDrive(&out)
	}
	if out.Len() != 0 {
		t.Fatalf("Ctrl+C written after the turn ended: %q", out.String())
	}
	if r, _ := d.agentDriveResult("test-drive"); !r.Terminal() || r.Outcome == agentdrive.OutcomeConfirmed {
		t.Fatalf("stale interrupt %+v", r)
	}
}
