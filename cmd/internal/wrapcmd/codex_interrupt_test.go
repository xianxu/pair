package wrapcmd

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/agentdrive"
)

func codexInterruptFixture(t *testing.T) (*harnessSessionFake, *peerDelivery) {
	t.Helper()
	f, d, _ := driveFixture(t)
	f.proxy.agentBasename = "codex"
	if err := f.proxy.configureHarnessTTY(true, 120, 38); err != nil {
		t.Fatal(err)
	}
	d.binding.Agent = "codex"
	d.settled = false
	return f, d
}

func capturedCodexWorking(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile("testdata/drive/codex/0.163.0/working-prefix.raw")
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func capturedCodexBackgroundWorking(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile("testdata/drive/codex/0.163.0/background-prefix.raw")
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// This synthetic frame preserves the captured status/composer separation.
// Waiting wording comes from the TL's morning live check, not our capture.
func syntheticCodexWork(status string, tall bool) string {
	var b strings.Builder
	b.WriteString("\x1b[?2026h\x1b[2J\x1b[11;1H" + status + "\x1b[14;1H\x1b[1m›\x1b[22m ")
	cursorRow, footerRow := 14, 16
	if !tall {
		b.WriteString("\x1b[2mAsk Codex to do anything\x1b[22m")
	}
	if tall {
		for row := 15; row <= 35; row++ {
			fmt.Fprintf(&b, "\x1b[%d;3Hcontinuation", row)
		}
		cursorRow, footerRow = 35, 38
	}
	fmt.Fprintf(&b, "\x1b[%d;3HGPT-6.1-Sol default · /scratch/repo\x1b[%d;3H\x1b[?25h\x1b[?2026l", footerRow, cursorRow)
	return b.String()
}

func codexInterruptDispatch(p *proxy) string {
	var out bytes.Buffer
	for i := 0; i < 3; i++ {
		p.dispatchAgentDrive(&out)
	}
	return out.String()
}

func TestCodexInterruptUsesRenderedWorkThroughProductionOutput(t *testing.T) {
	for name, paint := range map[string]string{
		"captured top anchored":                            capturedCodexWorking(t),
		"captured running background terminal":             capturedCodexBackgroundWorking(t),
		"waiting with native background summary synthetic": syntheticCodexWork("• Waiting for background terminal (15s • esc to interrupt) · 1 background terminal running · /ps to view · /stop to close", false),
		"alternate captured spinner":                       syntheticCodexWork("◦ Working (3s • esc to interrupt)", false),
		"tall composer":                                    syntheticCodexWork("• Working (3s • esc to interrupt)", true),
		"waiting background synthetic":                     syntheticCodexWork("• Waiting for background terminal (15s • esc to interrupt)", false),
		"waiting alternate synthetic":                      syntheticCodexWork("◦ Waiting for background terminal (15s • esc to interrupt)", false),
	} {
		t.Run(name, func(t *testing.T) {
			f, d := codexInterruptFixture(t)
			f.output(paint)
			if !f.proxy.turnWorking.Load() {
				t.Fatal("production output did not recognize native work")
			}
			if s := f.proxy.agentDriveScreen(); !s.TurnActive || !s.Interruptible {
				t.Fatalf("native work proof: %+v", s)
			}
			if r, err := driveStart(t, d, interruptRequest()); err != nil || r.Terminal() {
				t.Fatalf("admission %+v %v", r, err)
			}
			if got := codexInterruptDispatch(f.proxy); got != "\x03" {
				t.Fatalf("interrupt writes=%q", got)
			}
			f.output(paint)
			f.proxy.dispatchAgentDrive(&bytes.Buffer{})
			if r, _ := d.agentDriveResult("test-drive"); r.Terminal() {
				t.Fatalf("working redraw confirmed interruption: %+v", r)
			}
			f.output(syntheticCodexWork("", false))
			f.proxy.dispatchAgentDrive(&bytes.Buffer{})
			if r, _ := d.agentDriveResult("test-drive"); r.Outcome != agentdrive.OutcomeConfirmed {
				t.Fatalf("fresh idle confirmation %+v", r)
			}
			delete(d.drive.receipts, "test-drive")
			if r, err := driveStart(t, d, interruptRequest()); err != nil || r.Outcome != agentdrive.OutcomeRefused {
				t.Fatalf("second interrupt %+v %v", r, err)
			}
		})
	}
}

func TestCodexInterruptCurrentWorkOutranksStaleLifecycleCompletion(t *testing.T) {
	for name, observation := range map[string]ObservationKind{"completion": ObservationNativeCompletion, "stop grace": ObservationStopped} {
		t.Run(name, func(t *testing.T) {
			f, d := codexInterruptFixture(t)
			f.output(capturedCodexWorking(t))
			f.proxy.processLifecycleObservation(TurnObservation{Kind: observation})
			if observation == ObservationStopped && !f.proxy.notificationLifecycle.GracePending {
				t.Fatal("fixture did not enter cached stop grace")
			}
			if s := f.proxy.agentDriveScreen(); !s.TurnActive || !s.Interruptible {
				t.Fatalf("current work lost to stale completion: %+v", s)
			}
			if r, err := driveStart(t, d, interruptRequest()); err != nil || r.Terminal() {
				t.Fatalf("admission %+v %v", r, err)
			}
			if got := codexInterruptDispatch(f.proxy); got != "\x03" {
				t.Fatalf("writes=%q", got)
			}
			f.output(syntheticCodexWork("• Working (4s • esc to interrupt)", false))
			f.proxy.processLifecycleObservation(TurnObservation{Kind: observation})
			f.proxy.dispatchAgentDrive(&bytes.Buffer{})
			if r, _ := d.agentDriveResult("test-drive"); r.Terminal() {
				t.Fatalf("stale lifecycle confirmed working screen %+v", r)
			}
		})
	}
}

func TestCodexCurrentWorkDoesNotAuthorizeRecoveryKeys(t *testing.T) {
	f, d := codexInterruptFixture(t)
	f.output(capturedCodexWorking(t))
	f.proxy.processLifecycleObservation(TurnObservation{Kind: ObservationNativeCompletion})
	r, err := driveStart(t, d, agentdrive.Request{Keys: agentdrive.KeySequence{{Kind: agentdrive.KeyClearLine}}})
	if err == nil && !r.Terminal() {
		t.Fatalf("current working screen authorized recovery input: %+v", r)
	}
	if got := codexInterruptDispatch(f.proxy); got != "" {
		t.Fatalf("recovery bytes reached a working turn: %q", got)
	}
}

func TestCodexWorkBackgroundSummaryGrammar(t *testing.T) {
	for _, tc := range []struct {
		text string
		want bool
	}{
		{"", true},
		{" · 1 background terminal running · /ps to view · /stop to close", true},
		{" · 2 background terminals running · /ps to view · /stop to close", true},
		{" · 0 background terminals running · /ps to view · /stop to close", false},
		{" · 2 background terminal running · /ps to view · /stop to close", false},
		{" · +1 background terminal running · /ps to view · /stop to close", false},
		{" · 1 background terminal running", false},
		{" unknown suffix", false},
	} {
		if got := codexBackgroundSummary(tc.text); got != tc.want {
			t.Errorf("summary %q: got %v want %v", tc.text, got, tc.want)
		}
	}
}

func TestCodexInterruptRejectsUnqualifiedFramesAndOwners(t *testing.T) {
	for name, setup := range map[string]func(*harnessSessionFake, *peerDelivery){
		"idle": func(f *harnessSessionFake, d *peerDelivery) { f.output(syntheticCodexWork("", false)) },
		"MCP status": func(f *harnessSessionFake, d *peerDelivery) {
			f.output(syntheticCodexWork("• Starting MCP servers (1/2): codex_apps (0s • esc to interrupt)", false))
		},
		"quoted status": func(f *harnessSessionFake, d *peerDelivery) {
			f.output(syntheticCodexWork("> • Working (3s • esc to interrupt)", false))
		},
		"prose": func(f *harnessSessionFake, d *peerDelivery) {
			f.output(syntheticCodexWork("Codex says • Working (3s • esc to interrupt)", false))
		},
		"status in composer": func(f *harnessSessionFake, d *peerDelivery) {
			f.output(syntheticCodexWork("", false))
			f.output("\x1b[?2026h\x1b[14;3H• Working (3s • esc to interrupt)\x1b[14;3H\x1b[?2026l")
		},
		"distant output": func(f *harnessSessionFake, d *peerDelivery) {
			f.output(syntheticCodexWork("", false))
			f.output("\x1b[?2026h\x1b[3;1H• Working (3s • esc to interrupt)\x1b[14;3H\x1b[?2026l")
		},
		"partial redraw": func(f *harnessSessionFake, d *peerDelivery) { f.output("\x1b[?2026h\x1b[11;1H\x1b[2K") },
		"picker":         func(f *harnessSessionFake, d *peerDelivery) { f.proxy.pickerActive.Store(true) },
		"operator input": func(f *harnessSessionFake, d *peerDelivery) { d.inputBuffered = true },
		"peer owner":     func(f *harnessSessionFake, d *peerDelivery) { f.proxy.automaticInput.phase = automaticInputPeer },
	} {
		t.Run(name, func(t *testing.T) {
			f, d := codexInterruptFixture(t)
			f.output(capturedCodexWorking(t))
			setup(f, d)
			r, err := driveStart(t, d, interruptRequest())
			if err == nil && !r.Terminal() {
				t.Fatalf("admitted %+v", r)
			}
			if got := codexInterruptDispatch(f.proxy); got != "" {
				t.Fatalf("unexpected bytes %q", got)
			}
		})
	}
}

func TestCodexInterruptRechecksCurrentFrameBeforeWriting(t *testing.T) {
	for _, change := range []string{syntheticCodexWork("", false), "\x1b[?2026h\x1b[11;1H\x1b[2K"} {
		f, d := codexInterruptFixture(t)
		f.output(capturedCodexWorking(t))
		if r, err := driveStart(t, d, interruptRequest()); err != nil || r.Terminal() {
			t.Fatalf("admission %+v %v", r, err)
		}
		f.output(change)
		if got := codexInterruptDispatch(f.proxy); got != "" {
			t.Fatalf("stale proof wrote %q", got)
		}
	}
}

func TestCodexInterruptProofDoesNotInventWorkOnAModelPicker(t *testing.T) {
	f, _ := codexInterruptFixture(t)
	raw, err := os.ReadFile("testdata/drive/codex/0.162.1/model-picker.raw")
	if err != nil {
		t.Fatal(err)
	}
	f.output(string(raw))
	if s := f.proxy.agentDriveScreen(); s.TurnActive || s.Idle || s.Interruptible {
		t.Fatalf("picker fabricated work or idle evidence: %+v", s)
	}
}

func TestCodexWorkingIgnoresPartialRedrawUntilCoherentIdle(t *testing.T) {
	f, _ := codexInterruptFixture(t)
	f.output(capturedCodexWorking(t))
	if !f.proxy.turnWorking.Load() {
		t.Fatal("working capture not observed")
	}
	f.output("\x1b[?2026h\x1b[11;1H\x1b[2K")
	if !f.proxy.turnWorking.Load() || f.proxy.notificationLifecycle.GracePending {
		t.Fatal("partial redraw stopped current work")
	}
	if f.proxy.agentDriveScreen().Interruptible {
		t.Fatal("partial redraw authorizes interrupt")
	}
	f.output("\x1b[14;3H\x1b[?2026l")
	if f.proxy.turnWorking.Load() || !f.proxy.notificationLifecycle.GracePending {
		t.Fatal("coherent idle did not stop lifecycle work")
	}
}

func TestCodexInterruptDoesNotConfirmOnUnknownOrPartialFrame(t *testing.T) {
	for name, paint := range map[string]string{
		"unknown coherent": "\x1b[?2026h\x1b[2J\x1b[1;1Hunknown panel\x1b[?25l\x1b[?2026l",
		"partial":          "\x1b[?2026h\x1b[2J",
	} {
		t.Run(name, func(t *testing.T) {
			f, d := codexInterruptFixture(t)
			f.output(capturedCodexWorking(t))
			if r, err := driveStart(t, d, interruptRequest()); err != nil || r.Terminal() {
				t.Fatalf("admission %+v %v", r, err)
			}
			if got := codexInterruptDispatch(f.proxy); got != "\x03" {
				t.Fatalf("writes=%q", got)
			}
			f.output(paint)
			f.proxy.dispatchAgentDrive(&bytes.Buffer{})
			if r, _ := d.agentDriveResult("test-drive"); r.Terminal() {
				t.Fatalf("unknown frame confirmed interruption: %+v", r)
			}
		})
	}
}
