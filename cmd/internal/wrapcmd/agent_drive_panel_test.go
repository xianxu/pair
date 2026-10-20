package wrapcmd

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/agentdrive"
)

// paintClaude2304 paints a captured Claude 2.1.304 viewport (pair#691) at the
// slot's 94x39 size. A ‹cursor› mark places a visible cursor, as on a live
// composer; without one the cursor is hidden, as Claude leaves it on a panel.
func paintClaude2304(t *testing.T, f *harnessSessionFake, lines []string) {
	t.Helper()
	f.resize(94, 39)
	cursor := "\x1b[1;31H\x1b[?25l"
	rows := make([]string, len(lines))
	for y, line := range lines {
		if before, after, ok := strings.Cut(line, "‹cursor›"); ok {
			cursor = fmt.Sprintf("\x1b[%d;%dH\x1b[?25h", y+1, len([]rune(before))+1)
			line = before + after
		}
		rows[y] = line
	}
	f.output("\x1b[?2026h\x1b[2J\x1b[H" + strings.Join(rows, "\r\n") + cursor + "\x1b[?2026l")
}

func claude2304Screen(t *testing.T, name string) []string {
	t.Helper()
	raw, err := os.ReadFile("testdata/drive/claude/2.1.304/" + name + ".screen")
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
}

func paintUsagePanel(t *testing.T, f *harnessSessionFake) {
	t.Helper()
	paintClaude2304(t, f, claude2304Screen(t, "usage-panel"))
}

// TestAgentDrivePanelAdmitsOnlyEsc covers the short and the taller-than-screen
// panel, each with and without the turn flag Claude's progress report leaves
// set while a panel shows.
func TestAgentDrivePanelAdmitsOnlyEsc(t *testing.T) {
	for _, panel := range []string{"usage-panel", "usage-panel-tall"} {
		for _, turn := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s turn=%v", panel, turn), func(t *testing.T) {
				f, d, _ := driveFixture(t)
				paintClaude2304(t, f, claude2304Screen(t, panel))
				f.proxy.turnActive.Store(turn)
				d.settled = false
				screen := f.proxy.agentDriveScreen()
				if !screen.Dismissable || screen.Recoverable || screen.Idle {
					t.Fatalf("panel screen %+v", screen)
				}
				if o := d.agentDriveObservation(); !o.Dismissable || o.Recoverable || o.Idle {
					t.Fatalf("panel observation %+v", o)
				}
				for _, keys := range []agentdrive.KeySequence{{{Kind: agentdrive.KeyClearLine}}, {{Kind: agentdrive.KeyEsc}, {Kind: agentdrive.KeyEsc}}, {{Kind: agentdrive.KeyText, Text: "x"}}, {{Kind: agentdrive.KeySubmit}}, {{Kind: agentdrive.KeyEsc}, {Kind: agentdrive.KeyText, Text: "x"}}} {
					if r, err := driveStart(t, d, agentdrive.Request{Keys: keys}); err != nil || r.Outcome != agentdrive.OutcomeRefused || r.Detail != agentdrive.PanelRefusal {
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
				paintClaude2304(t, f, claude2304Screen(t, panel))
				f.proxy.dispatchAgentDrive(&out)
				if r, _ = d.agentDriveResult("test-drive"); r.Terminal() {
					t.Fatalf("panel repaint ended the dismissal %+v", r)
				}
				f.output("\x1b[?2026h\x1b[2J" + peerEmptyComposer() + "\x1b[?2026l")
				f.proxy.dispatchAgentDrive(&out)
				if turn {
					// Composer back, progress not yet cleared: keep observing.
					if r, _ = d.agentDriveResult("test-drive"); r.Terminal() {
						t.Fatalf("composer return under the turn flag ended the drive %+v", r)
					}
					f.proxy.turnActive.Store(false)
					f.proxy.dispatchAgentDrive(&out)
				}
				if r, _ = d.agentDriveResult("test-drive"); r.Outcome != agentdrive.OutcomeConfirmed {
					t.Fatalf("composer return %+v", r)
				}
				if out.String() != "\x1b" {
					t.Fatalf("extra bytes %q", out.String())
				}
			})
		}
	}
}

// TestAgentDriveWorkingTurnRefusesEsc pins the safety direction: during a real
// working turn, with no panel on screen, esc would interrupt the agent. Real
// 2.1.304 working-turn captures, and the same screens carrying the footer
// words anywhere but as a panel's last row, must refuse and write nothing.
func TestAgentDriveWorkingTurnRefusesEsc(t *testing.T) {
	for _, name := range []string{"working-thinking", "working-after-usage", "working-shell"} {
		lines := claude2304Screen(t, name)
		rule := len(lines) - 1
		for rule > 0 && !strings.HasPrefix(lines[rule], "─") {
			rule--
		}
		rule-- // the composer's opening rule
		for rule > 0 && !strings.HasPrefix(lines[rule], "─") {
			rule--
		}
		inHistory := append([]string{}, lines...)
		inHistory[3] = "  Esc to cancel"
		aboveComposer := append(append(append([]string{}, lines[1:rule]...), "  Esc to cancel"), lines[rule:]...)
		for variant, screen := range map[string][]string{"as captured": lines, "footer words in history": inHistory, "footer words above the composer": aboveComposer} {
			t.Run(name+", "+variant, func(t *testing.T) {
				f, d, _ := driveFixture(t)
				paintClaude2304(t, f, screen)
				f.proxy.turnActive.Store(true)
				d.settled = false
				if _, known := qualifiedComposerText("claude", f.proxy.terminal.Snapshot()); !known {
					t.Fatal("painted working turn shows no recognized composer; the capture did not paint faithfully")
				}
				if s := f.proxy.agentDriveScreen(); s.Dismissable || s.Idle || s.Recoverable {
					t.Fatalf("working turn gave input proof %+v", s)
				}
				r, err := driveStart(t, d, agentdrive.Request{Keys: agentdrive.KeySequence{{Kind: agentdrive.KeyEsc}}})
				if err != nil || r.Outcome != agentdrive.OutcomeRefused || r.Detail == agentdrive.PanelRefusal {
					t.Fatalf("esc during a working turn %+v %v", r, err)
				}
				var out bytes.Buffer
				f.proxy.dispatchAgentDrive(&out)
				if out.Len() != 0 {
					t.Fatalf("bytes written during a working turn %q", out.String())
				}
			})
		}
	}
}

// TestAgentDriveDismissalStopsIfThePanelGoes: a drive admitted on a panel
// writes nothing if a working turn replaces the panel before the writer
// starts it; the writer's start recheck reads the fresh screen.
func TestAgentDriveDismissalStopsIfThePanelGoes(t *testing.T) {
	f, d, _ := driveFixture(t)
	paintUsagePanel(t, f)
	f.proxy.turnActive.Store(true)
	d.settled = false
	if r, err := driveStart(t, d, agentdrive.Request{Keys: agentdrive.KeySequence{{Kind: agentdrive.KeyEsc}}}); err != nil || r.Terminal() {
		t.Fatalf("esc on a panel %+v %v", r, err)
	}
	paintClaude2304(t, f, claude2304Screen(t, "working-shell"))
	var out bytes.Buffer
	for i := 0; i < 3; i++ {
		f.proxy.dispatchAgentDrive(&out)
	}
	if out.Len() != 0 {
		t.Fatalf("esc written after the panel gave way to a working turn %q", out.String())
	}
	if r, _ := d.agentDriveResult("test-drive"); !r.Terminal() || r.Outcome == agentdrive.OutcomeConfirmed {
		t.Fatalf("stale dismissal %+v", r)
	}
}

func TestAgentDrivePanelProofNeedsInputOwnership(t *testing.T) {
	for _, fault := range []string{"picker", "buffer", "reply", "peer", "partial"} {
		t.Run(fault, func(t *testing.T) {
			f, d, _ := driveFixture(t)
			paintUsagePanel(t, f)
			d.settled = false
			switch fault {
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
