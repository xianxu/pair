package wrapcmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/agentdrive"
)

// Drain through the production lifecycle reducer, as the master proxy loop does.
func drainCommandLifecycle(p *proxy) []TurnObservation {
	var events []TurnObservation
	for {
		select {
		case event := <-p.lifecycleEvents:
			events = append(events, event)
			p.processLifecycleObservation(event)
		default:
			return events
		}
	}
}

func TestAgentDriveRawCommandEnterLifecycle(t *testing.T) {
	f, d, _ := driveFixture(t)
	f.proxy.lifecycleEvents = make(chan TurnObservation, 32)
	req := agentdrive.Request{Keys: agentdrive.KeySequence{{Kind: agentdrive.KeyText, Text: "/usage"}, {Kind: agentdrive.KeyEnter}}}
	if _, err := driveStart(t, d, req); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	f.proxy.dispatchAgentDrive(&out)
	f.output("\x1b[?2026h\x1b[2J" + slashComposerPaint("claude", "/usage") + "\x1b[?2026l")
	f.proxy.dispatchAgentDrive(&out)
	f.proxy.dispatchAgentDrive(&out)
	if out.String() != "/usage\r" {
		t.Fatalf("command bytes %q", out.String())
	}
	events := drainCommandLifecycle(f.proxy)
	for _, event := range events {
		if event.Kind == ObservationUserSubmission || event.Kind == ObservationBareReturn {
			t.Fatalf("command published conversational submission: %+v", event)
		}
	}
	if f.proxy.turnActive.Load() || f.proxy.notificationLifecycle.Generation != 0 {
		t.Fatal("command Enter opened a synthetic turn")
	}
	f.output("\x1b[?2026h\x1b[2J" + peerEmptyComposer() + "\x1b[?2026l")
	f.proxy.dispatchAgentDrive(&out)
	result, _ := d.agentDriveResult("test-drive")
	if result.Outcome != agentdrive.OutcomeConfirmed {
		t.Fatalf("raw slash self-canceled: %+v", result)
	}
}

func TestAgentDriveRawPublicSubmitStillOpensLifecycleTurn(t *testing.T) {
	f, d, _ := driveFixture(t)
	f.proxy.lifecycleEvents = make(chan TurnObservation, 32)
	req := agentdrive.Request{Keys: agentdrive.KeySequence{{Kind: agentdrive.KeyText, Text: "ordinary prose"}, {Kind: agentdrive.KeySubmit}, {Kind: agentdrive.KeyText, Text: "forbidden"}}}
	if _, err := driveStart(t, d, req); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	f.proxy.dispatchAgentDrive(&out)
	f.proxy.dispatchAgentDrive(&out)
	events := drainCommandLifecycle(f.proxy)
	submitted := false
	for _, event := range events {
		submitted = submitted || event.Kind == ObservationUserSubmission
	}
	if !submitted || !f.proxy.turnActive.Load() || f.proxy.notificationLifecycle.Generation == 0 {
		t.Fatalf("public submit lost ordinary lifecycle: %+v", events)
	}
	f.proxy.dispatchAgentDrive(&out)
	if strings.Contains(out.String(), "forbidden") {
		t.Fatal("ordinary drive continued into active turn")
	}
}

func TestAgentDriveCommandNativeWorkingStillVetoes(t *testing.T) {
	for _, beforeWrite := range []bool{true, false} {
		t.Run(map[bool]string{true: "before-command-write", false: "after-command-write"}[beforeWrite], func(t *testing.T) {
			f, d, _ := driveFixture(t)
			f.proxy.lifecycleEvents = make(chan TurnObservation, 32)
			req := agentdrive.Request{Keys: agentdrive.KeySequence{{Kind: agentdrive.KeyText, Text: "/usage"}, {Kind: agentdrive.KeyEnter}, {Kind: agentdrive.KeyText, Text: "forbidden"}}}
			if _, err := driveStart(t, d, req); err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			f.proxy.dispatchAgentDrive(&out)
			f.output("\x1b[?2026h\x1b[2J" + slashComposerPaint("claude", "/usage") + "\x1b[?2026l")
			f.proxy.dispatchAgentDrive(&out)
			if !beforeWrite {
				f.proxy.dispatchAgentDrive(&out)
			}
			drainCommandLifecycle(f.proxy)
			f.proxy.publishLifecycleObservation(TurnObservation{Kind: ObservationWorking})
			drainCommandLifecycle(f.proxy)
			if !f.proxy.turnActive.Load() || !f.proxy.turnWorking.Load() {
				t.Fatal("native working observation was suppressed")
			}
			prefix := out.String()
			f.proxy.dispatchAgentDrive(&out)
			result, _ := d.agentDriveResult("test-drive")
			if out.String() != prefix || !result.Terminal() || !strings.Contains(result.Detail, "turn") {
				t.Fatalf("native work did not stop drive: bytes=%q result=%+v", out.String(), result)
			}
		})
	}
}
