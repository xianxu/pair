package wrapcmd

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/agentdrive"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

func recoveryPaint(f *harnessSessionFake, text string) {
	lines := strings.Split(text, "\n")
	cursor := ""
	if len(lines) > 1 {
		cursor = fmt.Sprintf("\x1b[%d;3H", 7+len(lines)-1)
	}
	f.output("\x1b[?2026h\x1b[2J" + peerFilledComposer(lines...) + cursor + "\x1b[?2026l")
}
func TestAgentDriveRecoveryProjectionAndAdmission(t *testing.T) {
	for _, text := range []string{"draft", "/model", "!shell"} {
		t.Run(text, func(t *testing.T) {
			f, d, _ := driveFixture(t)
			recoveryPaint(f, text)
			d.settled = false
			screen := f.proxy.agentDriveScreen()
			if !screen.Recoverable || screen.Idle {
				t.Fatalf("screen %+v", screen)
			}
			if text == "/model" && screen.ComposerText != "/model" {
				t.Fatalf("command proof %q", screen.ComposerText)
			}
			if _, ok := peerComposerText("claude", f.proxy.terminal.Snapshot()); (strings.HasPrefix(text, "/") || strings.HasPrefix(text, "!")) && ok {
				t.Fatal("ordinary peer accepts command")
			}
			observation := d.agentDriveObservation()
			if !observation.Recoverable || observation.Idle {
				t.Fatalf("observation %+v", observation)
			}
			result, err := driveStart(t, d, agentdrive.Request{Keys: agentdrive.KeySequence{{Kind: agentdrive.KeyClearLine}}})
			if err != nil || result.Terminal() {
				t.Fatalf("recovery admission %+v %v", result, err)
			}
		})
	}
}
func TestAgentDriveRecoveryExclusionAndLease(t *testing.T) {
	for _, fault := range []string{"turn", "picker", "image", "buffer", "reply", "reservation", "peer", "drive", "unknown", "partial"} {
		t.Run(fault, func(t *testing.T) {
			f, d, _ := driveFixture(t)
			recoveryPaint(f, "/model")
			d.settled = false
			switch fault {
			case "turn":
				f.proxy.turnActive.Store(true)
			case "picker":
				f.proxy.pickerActive.Store(true)
			case "image":
				d.image = true
			case "buffer":
				d.inputBuffered = true
			case "reply":
				d.replies.inFlight = 1
			case "reservation":
				d.reservation = "other"
			case "peer":
				f.proxy.automaticInput.phase = automaticInputPeer
			case "drive":
				f.proxy.automaticInput.phase = automaticInputDrive
			case "unknown":
				f.output("\x1b[?2026h\x1b[2J\x1b[Hunknown\x1b[?2026l")
			case "partial":
				f.output("\x1b[?2026h")
			}
			_, _ = driveStart(t, d, agentdrive.Request{Keys: agentdrive.KeySequence{{Kind: agentdrive.KeyClearLine}}})
			var out bytes.Buffer
			f.proxy.dispatchAgentDrive(&out)
			if out.Len() != 0 {
				t.Fatalf("recovery bytes after %s: %q", fault, out.String())
			}
		})
	}
}
func TestAgentDriveRecoveryLeavesNormalAdmissionStrict(t *testing.T) {
	for _, request := range []agentdrive.Request{{Keys: agentdrive.KeySequence{{Kind: agentdrive.KeyEnter}}}, {Model: &agentdrive.ModelRequest{Model: "opus"}}} {
		f, d, _ := driveFixture(t)
		recoveryPaint(f, "/model")
		d.settled = false
		result, err := driveStart(t, d, request)
		if err != nil || result.Outcome != agentdrive.OutcomeRefused {
			t.Fatalf("ordinary admission %+v %v", result, err)
		}
	}
}
func TestAgentDriveRecoveryHandsLeaseToNormalDelivery(t *testing.T) {
	f, d, now := driveFixture(t)
	recoveryPaint(f, "/model")
	d.settled = false
	f.proxy.automaticInput = automaticInputTransaction{phase: automaticInputAwaitingEmpty, renderFence: f.proxy.automaticRenderEpoch.Load()}
	result, err := driveStart(t, d, agentdrive.Request{Keys: agentdrive.KeySequence{{Kind: agentdrive.KeyClearLine}}})
	if err != nil || result.Terminal() {
		t.Fatalf("admission %+v %v", result, err)
	}
	var out bytes.Buffer
	f.proxy.dispatchPeer(&out)
	if out.String() != "\x15" {
		t.Fatalf("recovery bytes %q", out.String())
	}
	f.output("\x1b[?2026h\x1b[2J" + peerEmptyComposer() + "\x1b[?2026l")
	f.proxy.dispatchPeer(&out)
	result, _ = d.agentDriveResult("test-drive")
	if result.Outcome != agentdrive.OutcomeConfirmed || f.proxy.automaticInput.phase != automaticInputIdle {
		t.Fatalf("recovery %+v lease %v", result, f.proxy.automaticInput.phase)
	}
	message := peerTestMessage(*now)
	message.To = d.binding
	message.ID = "after-recovery"
	if err = d.reserve(message.ID, d.agentDriveObservation().Sequence); err != nil {
		t.Fatal(err)
	}
	if err = d.enqueue(message); err != nil {
		t.Fatal(err)
	}
	f.proxy.dispatchPeer(&out)
	if !strings.Contains(out.String(), peerEnvelope(message)) {
		t.Fatalf("delivery wedged %+v", d.receipt())
	}
	peerRender(f, d, "\x1b[?2026h"+peerFilledComposer("[Pasted text #1 +3 lines]")+"\x1b[?2026l")
	*now = now.Add(PeerSubmitDelay + time.Millisecond)
	f.proxy.dispatchPeer(&out)
	peerRender(f, d, "\x1b[?2026h"+peerEmptyComposer()+"\x1b[?2026l")
	f.proxy.dispatchPeer(&out)
	if d.receipt().Status != couchmessage.Submitted {
		t.Fatalf("delivery %+v", d.receipt())
	}
}

func TestAgentDriveRecoveryPreservesStallSequence(t *testing.T) {
	for _, phase := range []automaticInputPhase{automaticInputAwaitingEmpty, automaticInputAlerted} {
		t.Run(fmt.Sprint(phase), func(t *testing.T) {
			f, d, _ := driveFixture(t)
			recoveryPaint(f, "/model")
			d.settled = false
			f.proxy.automaticInput = automaticInputTransaction{phase: phase, sequence: 17, renderFence: f.proxy.automaticRenderEpoch.Load()}
			result, err := driveStart(t, d, agentdrive.Request{Keys: agentdrive.KeySequence{{Kind: agentdrive.KeyClearLine}}})
			if err != nil || result.Terminal() {
				t.Fatalf("admission %+v %v", result, err)
			}
			var out bytes.Buffer
			f.proxy.dispatchAgentDrive(&out)
			if out.String() != "\x15" {
				t.Fatalf("residual lease did not admit explicit recovery: %q", out.String())
			}
			f.output("\x1b[?2026h\x1b[2J" + peerEmptyComposer() + "\x1b[?2026l")
			f.proxy.dispatchAgentDrive(&out)
			result, _ = d.agentDriveResult("test-drive")
			if result.Outcome != agentdrive.OutcomeConfirmed || f.proxy.automaticInput.phase != automaticInputIdle || f.proxy.automaticInput.sequence != 17 {
				t.Fatalf("recovery lost tripwire generation: %+v tx=%+v", result, f.proxy.automaticInput)
			}
		})
	}
}
func TestAgentDriveRecoveryTickUsesExactOwnedCommand(t *testing.T) {
	for _, text := range []string{"/model", "unrelated", "/model\n"} {
		t.Run(text, func(t *testing.T) {
			f, d, now := driveFixture(t)
			o := d.agentDriveObservation()
			_, err := d.startAgentDrive("cleanup", couchmessage.DriveStart{Request: agentdrive.Request{Model: &agentdrive.ModelRequest{Model: "opus"}}, Instance: o.Instance, Sequence: o.Sequence, Deadline: now.Add(10 * time.Second)})
			if err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			writeDriveModelCommand(t, f, &out)
			recoveryPaint(f, text)
			f.proxy.dispatchAgentDrive(&out)
			before := out.Len()
			*now = now.Add(9500 * time.Millisecond)
			f.proxy.dispatchAgentDrive(&out)
			if text == "/model" {
				if out.Len() != before+1 || out.Bytes()[out.Len()-1] != 0x15 {
					t.Fatalf("missing owned cleanup %q", out.Bytes()[before:])
				}
			} else if out.Len() != before {
				t.Fatalf("cleanup over %q: %q", text, out.Bytes()[before:])
			}
		})
	}
}

func TestAgentDriveRecoveryOperatorCancellationAddsNoCleanup(t *testing.T) {
	f, d, _ := driveFixture(t)
	recoveryPaint(f, "/model")
	d.settled = false
	_, err := driveStart(t, d, agentdrive.Request{Keys: agentdrive.KeySequence{{Kind: agentdrive.KeyClearLine}, {Kind: agentdrive.KeyEsc}}})
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	f.proxy.dispatchAgentDrive(&out)
	before := out.String()
	f.proxy.admitStdin([]byte("operator draft"), false)
	f.proxy.dispatchAgentDrive(&out)
	if out.String() != before {
		t.Fatalf("cleanup after operator cancellation %q", out.String())
	}
	result, _ := d.agentDriveResult("test-drive")
	if !result.Terminal() || result.Outcome == agentdrive.OutcomeConfirmed {
		t.Fatalf("cancel result %+v", result)
	}
}

func TestAgentDriveRecoveryRechecksComposerBeforeEachKey(t *testing.T) {
	for _, fault := range []string{"unknown", "picker", "image", "buffer"} {
		t.Run(fault, func(t *testing.T) {
			f, d, _ := driveFixture(t)
			recoveryPaint(f, "/model")
			d.settled = false
			_, err := driveStart(t, d, agentdrive.Request{Keys: agentdrive.KeySequence{{Kind: agentdrive.KeyClearLine}, {Kind: agentdrive.KeyEsc}}})
			if err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			f.proxy.dispatchAgentDrive(&out)
			before := out.String()
			switch fault {
			case "unknown":
				f.output("\x1b[?2026h\x1b[2J\x1b[Hunknown\x1b[?2026l")
			case "picker":
				f.proxy.pickerActive.Store(true)
			case "image":
				d.image = true
			case "buffer":
				d.inputBuffered = true
			}
			f.proxy.dispatchAgentDrive(&out)
			if out.String() != before {
				t.Fatalf("recovery continued after %s: %q", fault, out.String())
			}
		})
	}
}

func TestAgentDriveRecoveryCodexCommandProof(t *testing.T) {
	f := newHarnessSessionFake(t, "codex", true)
	defer f.close()
	now := time.Now()
	d := newPeerDelivery(couchmessage.Binding{Agent: "codex"}, func() time.Time { return now })
	f.proxy.peer = d
	d.driveProbe = f.proxy.agentDriveScreenLocked
	paint := "\x1b[?2026h\x1b[2J\x1b[20;1H\x1b[1m›\x1b[22m /model\x1b[22;3HGPT-6.1-Sol default · /fixture\x1b[?25h\x1b[20;9H\x1b[?2026l"
	f.output(paint)
	screen := f.proxy.agentDriveScreen()
	if !screen.Recoverable || screen.Idle || screen.ComposerText != "/model" {
		t.Fatalf("Codex command proof %+v", screen)
	}
	if _, ok := peerComposerText("codex", f.proxy.terminal.Snapshot()); ok {
		t.Fatal("ordinary peer delivery accepted Codex slash command")
	}
	result, err := driveStart(t, d, agentdrive.Request{Keys: agentdrive.KeySequence{{Kind: agentdrive.KeyClearLine}}})
	if err != nil || result.Terminal() {
		t.Fatalf("Codex recovery %+v %v", result, err)
	}
	var out bytes.Buffer
	f.proxy.dispatchAgentDrive(&out)
	if out.String() != "\x15" {
		t.Fatalf("Codex recovery bytes %q", out.String())
	}
}

func TestAgentDriveScreenReadersDoNotRaceInputOwnership(t *testing.T) {
	f, d, _ := driveFixture(t)
	recoveryPaint(f, "/model")
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 1000; i++ {
			d.mu.Lock()
			f.proxy.automaticInput.phase = automaticInputPeer
			d.image = true
			d.mu.Unlock()
			d.mu.Lock()
			f.proxy.automaticInput.phase = automaticInputIdle
			d.image = false
			d.mu.Unlock()
		}
	}()
	for i := 0; i < 1000; i++ {
		screen := f.proxy.agentDriveScreen()
		if screen.ComposerText != "/model" {
			t.Fatalf("display evidence changed: %+v", screen)
		}
		_ = d.agentDriveObservation()
	}
	<-done
}

func TestAgentDriveRecoveryQualificationChangesBeforeInput(t *testing.T) {
	for _, stage := range []string{"admission", "dispatch"} {
		for _, change := range []string{"sequence", "screen"} {
			t.Run(stage+"/"+change, func(t *testing.T) {
				f, d, now := driveFixture(t)
				recoveryPaint(f, "/model")
				d.settled = false
				observation := d.agentDriveObservation()
				if !observation.Recoverable {
					t.Fatal("missing qualified baseline")
				}
				start := couchmessage.DriveStart{Request: agentdrive.Request{Keys: agentdrive.KeySequence{{Kind: agentdrive.KeyClearLine}}}, Instance: observation.Instance, Sequence: observation.Sequence, Deadline: now.Add(time.Second)}
				alter := func() {
					if change == "sequence" {
						d.sequence++
					} else {
						f.output("\x1b[?2026h\x1b[2J\x1b[Hunknown\x1b[?2026l")
					}
				}
				if stage == "admission" {
					alter()
				}
				result, err := d.startAgentDrive("qualification", start)
				if err != nil {
					t.Fatal(err)
				}
				if stage == "dispatch" {
					if result.Terminal() {
						t.Fatalf("unexpected admission refusal: %+v", result)
					}
					alter()
				}
				var out bytes.Buffer
				f.proxy.dispatchAgentDrive(&out)
				if stage == "dispatch" {
					result, err = d.agentDriveResult("qualification")
					if err != nil {
						t.Fatal(err)
					}
				}
				if result.Outcome != agentdrive.OutcomeRefused || result.Step != 0 || out.Len() != 0 {
					t.Fatalf("unsafe changed qualification: %+v bytes=%q", result, out.String())
				}
				recoveryPaint(f, "/model")
				f.proxy.dispatchAgentDrive(&out)
				if out.Len() != 0 {
					t.Fatalf("refused operation replayed: %q", out.String())
				}
			})
		}
	}
}
