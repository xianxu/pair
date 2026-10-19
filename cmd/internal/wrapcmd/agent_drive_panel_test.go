package wrapcmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/agentdrive"
)

// paintUsagePanel paints the PROVISIONAL /usage fixture (pair#691) as a real
// frame: styled-tail markup becomes the SGR it was rendered from.
func paintUsagePanel(t *testing.T, f *harnessSessionFake) {
	t.Helper()
	raw, err := os.ReadFile("testdata/drive/claude/provisional/usage-panel.screen")
	if err != nil {
		t.Fatal(err)
	}
	frame := strings.NewReplacer("‹rev›", "\x1b[7m", "‹/rev›", "\x1b[27m", "\n", "\r\n").Replace(strings.TrimRight(string(raw), "\n"))
	f.output("\x1b[?2026h\x1b[2J\x1b[H" + frame + "\x1b[?25l\x1b[?2026l")
}

func TestAgentDrivePanelAdmitsOnlyEsc(t *testing.T) {
	f, d, _ := driveFixture(t)
	paintUsagePanel(t, f)
	d.settled = false
	screen := f.proxy.agentDriveScreen()
	if !screen.Dismissable || screen.Recoverable || screen.Idle {
		t.Fatalf("panel screen %+v", screen)
	}
	if o := d.agentDriveObservation(); !o.Dismissable || o.Recoverable || o.Idle {
		t.Fatalf("panel observation %+v", o)
	}
	for _, keys := range []agentdrive.KeySequence{{{Kind: agentdrive.KeyClearLine}}, {{Kind: agentdrive.KeyEsc}, {Kind: agentdrive.KeyEsc}}, {{Kind: agentdrive.KeyText, Text: "x"}}, {{Kind: agentdrive.KeySubmit}}, {{Kind: agentdrive.KeyEsc}, {Kind: agentdrive.KeyText, Text: "x"}}} {
		if r, err := driveStart(t, d, agentdrive.Request{Keys: keys}); err != nil || r.Outcome != agentdrive.OutcomeRefused {
			t.Fatalf("%v on a panel: %+v %v", keys, r, err)
		}
		delete(d.drive.receipts, "test-drive")
	}
	r, err := driveStart(t, d, agentdrive.Request{Keys: agentdrive.KeySequence{{Kind: agentdrive.KeyEsc}}})
	if err != nil || r.Terminal() {
		t.Fatalf("esc on a panel %+v %v", r, err)
	}
	var out bytes.Buffer
	for i := 0; i < 3; i++ {
		f.proxy.dispatchAgentDrive(&out)
	}
	if out.String() != "\x1b" {
		t.Fatalf("dismissal bytes %q", out.String())
	}
	paintUsagePanel(t, f)
	f.proxy.dispatchAgentDrive(&out)
	if r, _ = d.agentDriveResult("test-drive"); r.Terminal() {
		t.Fatalf("panel repaint ended the dismissal %+v", r)
	}
	f.output("\x1b[?2026h\x1b[2J" + peerEmptyComposer() + "\x1b[?2026l")
	f.proxy.dispatchAgentDrive(&out)
	if r, _ = d.agentDriveResult("test-drive"); r.Outcome != agentdrive.OutcomeConfirmed {
		t.Fatalf("composer return %+v", r)
	}
	if out.String() != "\x1b" {
		t.Fatalf("extra bytes %q", out.String())
	}
}

func TestAgentDrivePanelProofNeedsInputOwnership(t *testing.T) {
	for _, fault := range []string{"turn", "picker", "buffer", "reply", "peer", "partial"} {
		t.Run(fault, func(t *testing.T) {
			f, d, _ := driveFixture(t)
			paintUsagePanel(t, f)
			d.settled = false
			switch fault {
			case "turn":
				f.proxy.turnActive.Store(true)
			case "picker":
				f.proxy.pickerActive.Store(true)
			case "buffer":
				d.inputBuffered = true
			case "reply":
				d.replies.inFlight = 1
			case "peer":
				f.proxy.automaticInput.phase = automaticInputPeer
			case "partial":
				f.output("\x1b[?2026h")
			}
			_, _ = driveStart(t, d, agentdrive.Request{Keys: agentdrive.KeySequence{{Kind: agentdrive.KeyEsc}}})
			var out bytes.Buffer
			f.proxy.dispatchAgentDrive(&out)
			if out.Len() != 0 {
				t.Fatalf("esc written with %s: %q", fault, out.String())
			}
		})
	}
}
