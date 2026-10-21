package wrapcmd

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/agentdrive"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

// Unlike harnessSessionFake.output alone, the native output reader advances
// mailbox activity before handleChunk and forwards the completed output after it.
func codexNativeOutput(f *harnessSessionFake, d *peerDelivery, frame string) {
	d.observeOutput([]byte(frame))
	f.output(frame) // handleChunk defers outputForwarded.
}

// Derived ticking variant of the captured 94x39 Working-at-6s viewport. Only
// its native status row is repainted; the captured composer/footer stay intact.
func codexTickFrame(t *testing.T, seconds int) string {
	t.Helper()
	return capturedCodexOperatorWork(t, "operator-working-six") + fmt.Sprintf("\x1b[?2026h\x1b[34;1H\x1b[2K• Working (%ds • esc to interrupt) · 1 background terminal running · …\x1b[37;3H\x1b[?2026l", seconds)
}

func TestCodexInterruptSurvivesNativeWorkRedraws(t *testing.T) {
	for _, waiting := range []bool{false, true} {
		t.Run(fmt.Sprintf("waiting=%t", waiting), func(t *testing.T) {
			f, d := codexInterruptFixture(t)
			f.resize(94, 39)
			codexNativeOutput(f, d, codexTickFrame(t, 6))
			first := d.agentDriveObservation()
			activity := d.sequence
			codexNativeOutput(f, d, codexTickFrame(t, 7))
			fresh := d.agentDriveObservation()
			if !first.Interruptible || !fresh.Interruptible || first.Instance != fresh.Instance || first.Sequence != fresh.Sequence || d.sequence <= activity {
				t.Fatalf("working redraw invalidated stable admission: first=%+v fresh=%+v activity=%d->%d", first, fresh, activity, d.sequence)
			}
			start := couchmessage.DriveStart{Request: interruptRequest(), Instance: first.Instance, Sequence: first.Sequence, Deadline: d.now().Add(time.Second)}
			if r, err := d.startAgentDrive("test-drive", start); err != nil || r.Terminal() {
				t.Fatalf("tick between observation and start: %+v %v", r, err)
			}
			frame := codexTickFrame(t, 9)
			if waiting {
				frame = capturedCodexOperatorWork(t, "operator-waiting")
			}
			codexNativeOutput(f, d, frame)
			var out bytes.Buffer
			for i := 0; i < 3; i++ {
				f.proxy.dispatchAgentDrive(&out)
			}
			if out.String() != "\x03" {
				t.Fatalf("tick before write: got %q", out.String())
			}
			codexNativeOutput(f, d, frame)
			f.proxy.dispatchAgentDrive(&out)
			if r, _ := d.agentDriveResult("test-drive"); r.Terminal() {
				t.Fatalf("working redraw confirmed interruption: %+v", r)
			}
			codexNativeOutput(f, d, syntheticCodexWork("", false))
			f.proxy.dispatchAgentDrive(&out)
			if r, _ := d.agentDriveResult("test-drive"); r.Outcome != agentdrive.OutcomeConfirmed || out.String() != "\x03" {
				t.Fatalf("fresh idle result=%+v bytes=%q", r, out.String())
			}
			delete(d.drive.receipts, "test-drive")
			if r, err := driveStart(t, d, interruptRequest()); err != nil || r.Outcome != agentdrive.OutcomeRefused {
				t.Fatalf("second interrupt: %+v %v", r, err)
			}
		})
	}
}

func TestCodexInterruptInputInvalidatesAcrossNativeRedraws(t *testing.T) {
	inputs := map[string]func(*peerDelivery){
		"typing": func(d *peerDelivery) { d.admitReleased([]byte("x")); d.inputForwarded(true, false) },
		"escape": func(d *peerDelivery) {
			d.admitReleased([]byte("\x1b"))
			d.flushPartialInput()
			d.inputForwarded(true, false)
		},
		"image":  (*peerDelivery).admitImage,
		"submit": (*peerDelivery).humanSubmit,
	}
	for name, input := range inputs {
		for _, beforeStart := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/beforeStart=%t", name, beforeStart), func(t *testing.T) {
				f, d := codexInterruptFixture(t)
				f.resize(94, 39)
				codexNativeOutput(f, d, codexTickFrame(t, 6))
				first := d.agentDriveObservation()
				start := couchmessage.DriveStart{Request: interruptRequest(), Instance: first.Instance, Sequence: first.Sequence, Deadline: d.now().Add(time.Second)}
				if !beforeStart {
					if r, err := d.startAgentDrive("test-drive", start); err != nil || r.Terminal() {
						t.Fatalf("start: %+v %v", r, err)
					}
				}
				input(d)
				// Clear transient flags as the native UI may do; input generation
				// must still revoke the old proof after another eligible redraw.
				d.image = false
				codexNativeOutput(f, d, codexTickFrame(t, 7))
				if d.agentDriveObservation().Sequence == first.Sequence {
					t.Fatal("operator input did not revoke old observation")
				}
				if beforeStart {
					if r, err := d.startAgentDrive("test-drive", start); err == nil && !r.Terminal() {
						t.Fatalf("old input generation admitted: %+v", r)
					}
				}
				if got := codexInterruptDispatch(f.proxy); got != "" {
					t.Fatalf("operator input did not revoke pending interrupt: %q", got)
				}
			})
		}
	}
}

func TestCodexInterruptRefusesUnconsumedNativeOutput(t *testing.T) {
	for _, gap := range []string{"observation", "start", "write"} {
		t.Run(gap, func(t *testing.T) {
			f, d := codexInterruptFixture(t)
			f.resize(94, 39)
			codexNativeOutput(f, d, codexTickFrame(t, 6))
			first := d.agentDriveObservation()
			start := couchmessage.DriveStart{Request: interruptRequest(), Instance: first.Instance, Sequence: first.Sequence, Deadline: d.now().Add(time.Second)}
			if gap == "write" {
				if r, err := d.startAgentDrive("test-drive", start); err != nil || r.Terminal() {
					t.Fatalf("start %+v %v", r, err)
				}
			}
			// The reader has seen bytes but handleChunk has not parsed them.
			// The old complete composer is not a current proof in this gap.
			d.observeOutput([]byte("\x1b[?2026h\x1b[2J"))
			if gap == "observation" && d.agentDriveObservation().Interruptible {
				t.Fatal("old rendered frame admitted while output awaits parsing")
			}
			if gap == "start" {
				if r, err := d.startAgentDrive("test-drive", start); err == nil && !r.Terminal() {
					t.Fatalf("start used old frame: %+v", r)
				}
			}
			if got := codexInterruptDispatch(f.proxy); got != "" {
				t.Fatalf("unconsumed output allowed interrupt: %q", got)
			}
		})
	}
}

func TestOrdinaryDriveObservationRetainsActivitySequence(t *testing.T) {
	f, d, _ := driveFixture(t)
	first := d.agentDriveObservation()
	d.observeOutput([]byte("\x1b[?2026h" + peerEmptyComposer() + "\x1b[?2026l"))
	f.output("\x1b[?2026h" + peerEmptyComposer() + "\x1b[?2026l")
	d.settled = true
	fresh := d.agentDriveObservation()
	if !first.Idle || !fresh.Idle || first.Sequence == fresh.Sequence || fresh.Sequence != d.sequence {
		t.Fatalf("ordinary drive lost activity fence: %+v %+v activity=%d", first, fresh, d.sequence)
	}
	r, err := d.startAgentDrive("test-drive", couchmessage.DriveStart{Request: agentdrive.Request{Keys: agentdrive.KeySequence{{Kind: agentdrive.KeyEnter}}}, Instance: first.Instance, Sequence: first.Sequence, Deadline: d.now().Add(time.Second)})
	if err == nil && !r.Terminal() {
		t.Fatalf("ordinary drive admitted stale output generation: %+v", r)
	}
}
