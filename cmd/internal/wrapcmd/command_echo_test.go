package wrapcmd

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/xianxu/pair/cmd/internal/agentdrive"
)

func TestAgentDriveCommandTextDoesNotBroadenModelCleanup(t *testing.T) {
	for _, agent := range []string{"claude", "codex", "qoder"} {
		t.Run(agent, func(t *testing.T) {
			p := &proxy{agentBasename: agent}
			if err := p.configureHarnessTTY(true, 120, 38); err != nil {
				t.Fatal(err)
			}
			defer p.closeTerminal()
			var rolling []byte
			p.handleChunk([]byte("\x1b[?2026h"+slashComposerPaint(agent, "/usage")+"\x1b[?2026l"), &rolling)
			screen := p.agentDriveScreen()
			if !screen.Complete || !screen.Recoverable || !screen.CommandKnown || screen.CommandText != "/usage" {
				t.Fatalf("command proof missing: %+v", screen)
			}
			if screen.ComposerText != "" || agentdrive.ReadModelScreen(agent, screen).Phase == agentdrive.ScreenCommand {
				t.Fatal("general command gained model cleanup authority")
			}
		})
	}
}

func TestCommandReturnUsesCapturedProofFrame(t *testing.T) {
	p := &proxy{agentBasename: "claude"}
	if err := p.configureHarnessTTY(true, 120, 38); err != nil {
		t.Fatal(err)
	}
	defer p.closeTerminal()
	var rolling []byte
	p.handleChunk([]byte("\x1b[?2026h"+slashComposerPaint("claude", "/model")+"\x1b[?2026l"), &rolling)
	frame := p.terminal.Snapshot()
	// A second snapshot during emission would combine a command proof with a
	// different Return decision. Output may redraw after the captured gate.
	p.overlayConsumeHook = func() {
		if err := p.terminal.Feed([]byte("\x1b[?2026h\x1b[2J" + slashComposerPaint("claude", "prose") + "\x1b[?2026l")); err != nil {
			t.Fatal(err)
		}
	}
	got := p.emitPlainCRFromSnapshot(nil, &frame)
	if string(got) != "\r" {
		t.Fatalf("proved command became newline on a second snapshot: %q", got)
	}
	p.overlayConsumeHook = nil
	if got := p.emitPlainCR(nil); string(got) != "\\\r" {
		t.Fatalf("ordinary Return stopped taking fresh snapshots: %q", got)
	}
}

func TestAgentDriveCommandWaitsExactEcho(t *testing.T) {
	for _, structured := range []bool{false, true} {
		for _, change := range []string{"identical", "different", "operator", "redraw-at-emission"} {
			name := fmt.Sprintf("structured=%t/%s", structured, change)
			t.Run(name, func(t *testing.T) {
				f, d, _ := driveFixture(t)
				req := agentdrive.Request{Keys: agentdrive.KeySequence{{Kind: agentdrive.KeyClearLine}, {Kind: agentdrive.KeyText, Text: "/model"}, {Kind: agentdrive.KeyEnter}}}
				if structured {
					req = agentdrive.Request{Model: &agentdrive.ModelRequest{Model: "opus"}}
				}
				if result, err := driveStart(t, d, req); err != nil || result.Terminal() {
					t.Fatalf("start %+v %v", result, err)
				}
				var out bytes.Buffer
				f.proxy.dispatchAgentDrive(&out)
				f.proxy.dispatchAgentDrive(&out)
				prefix := out.String()
				if prefix != "\x15/model" {
					t.Fatalf("authored command: %q", prefix)
				}
				paint := func(text string) { f.output("\x1b[?2026h\x1b[2J" + slashComposerPaint("claude", text) + "\x1b[?2026l") }
				for _, text := range []string{"", "", "/mod", "/usage"} {
					paint(text)
					for range 3 {
						f.proxy.dispatchAgentDrive(&out)
					}
					if out.String() != prefix {
						t.Fatalf("Enter before exact echo %q: %q", text, out.String())
					}
				}
				paint("/model")
				f.proxy.dispatchAgentDrive(&out) // cache the exact command effect
				d.mu.Lock()
				bound := len(d.drive.active.effects) == 1 && d.drive.active.effects[0].CommandEnter()
				d.mu.Unlock()
				if !bound {
					t.Fatal("exact echo did not create a command-bound Enter")
				}
				switch change {
				case "identical":
					paint("/model")
				case "different":
					paint("/usage")
				case "operator":
					f.proxy.admitStdin([]byte("operator"), false)
				case "redraw-at-emission":
					f.proxy.overlayConsumeHook = func() {
						if err := f.proxy.terminal.Feed([]byte("\x1b[?2026h\x1b[2J" + slashComposerPaint("claude", "prose") + "\x1b[?2026l")); err != nil {
							t.Fatal(err)
						}
					}
				}
				f.proxy.dispatchAgentDrive(&out)
				f.proxy.overlayConsumeHook = nil
				want := prefix
				if change == "identical" || change == "redraw-at-emission" {
					want += "\r"
				}
				if out.String() != want {
					t.Fatalf("%s wrote %q want %q", change, out.String(), want)
				}
			})
		}
	}
}

func TestAgentDriveCommandTrailingSpacesReachNativeReturn(t *testing.T) {
	for _, command := range []string{"/model ", "/model opus  5.5   "} {
		t.Run(command, func(t *testing.T) {
			f, d, _ := driveFixture(t)
			req := agentdrive.Request{Keys: agentdrive.KeySequence{{Kind: agentdrive.KeyClearLine}, {Kind: agentdrive.KeyText, Text: command}, {Kind: agentdrive.KeyEnter}}}
			if result, err := driveStart(t, d, req); err != nil || result.Terminal() {
				t.Fatalf("start %+v %v", result, err)
			}
			var out bytes.Buffer
			f.proxy.dispatchAgentDrive(&out)
			f.proxy.dispatchAgentDrive(&out)
			prefix := "\x15" + command
			if out.String() != prefix {
				t.Fatalf("authored bytes changed: %q", out.String())
			}
			f.output("\x1b[?2026h\x1b[2J" + slashComposerPaint("claude", command) + "\x1b[?2026l")
			f.proxy.dispatchAgentDrive(&out)
			d.mu.Lock()
			bound := len(d.drive.active.effects) == 1 && d.drive.active.effects[0].CommandEnter()
			d.mu.Unlock()
			if !bound {
				t.Fatal("rendered trailing-space echo did not bind native Enter")
			}
			f.proxy.dispatchAgentDrive(&out)
			if out.String() != prefix+"\r" {
				t.Fatalf("native Return missing: %q", out.String())
			}
		})
	}
}
