package wrapcmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/xianxu/pair/cmd/internal/agentdrive"
)

// These native paint prefixes end at the first complete exact command echo,
// before submission or a command panel. They are independent of the versioned
// tty fixture catalog: this regression defends Return's input proof.
func TestSlashReturnNativeCapture(t *testing.T) {
	for _, tc := range []struct {
		agent, version, command, prose string
	}{
		{"claude", "2.1.309", "/status", "status!"},
		{"codex", "0.163.0", "/model", "hello!"},
	} {
		t.Run(tc.agent+"/"+tc.version, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join("testdata", "slash-return", tc.agent+".raw"))
			if err != nil {
				t.Fatal(err)
			}
			p := &proxy{agentBasename: tc.agent}
			if err := p.configureHarnessTTY(true, 120, 38); err != nil {
				t.Fatal(err)
			}
			defer p.closeTerminal()
			profile := *p.ttyProfile
			completeCommand := false
			for i := range raw {
				if err := p.terminal.Feed(raw[i : i+1]); err != nil {
					t.Fatal(err)
				}
				snapshot := p.terminal.Snapshot()
				text, known := qualifiedComposerText(tc.agent, snapshot)
				decision := decidePlainReturn(profile, false, &snapshot)
				// Inactive/menu Return is an existing, separate policy. It
				// must never acquire the slash-native decision from partial,
				// unknown, empty or prose composer evidence.
				if decision.reason == "plain Enter → native submit (slash command)" && (!snapshot.Complete || !known || !agentdrive.SlashCommand(text)) {
					t.Fatalf("slash authority without complete command at byte %d", i+1)
				}
				if profile.recognize(snapshot).Active && (!snapshot.Complete || !known || !agentdrive.SlashCommand(text)) && decision.submits {
					t.Fatalf("active unknown/prose composer submitted at byte %d", i+1)
				}
				if snapshot.Complete && known && text == tc.command {
					if i != len(raw)-1 {
						t.Fatalf("fixture extends beyond first complete command: byte %d of %d", i+1, len(raw))
					}
					completeCommand = true
				}
			}
			if !completeCommand {
				t.Fatal("capture lacks complete exact command echo")
			}
			frame := p.terminal.Snapshot()
			decision := decidePlainReturn(profile, false, &frame)
			if !decision.submits || decision.reason != "plain Enter → native submit (slash command)" || !bytes.Equal(decision.bytes, profile.keymap.altCR) {
				t.Fatalf("native command Return not proved: %+v", decision)
			}
			if got := p.emitPlainCRFromSnapshot(nil, &frame); !bytes.Equal(got, profile.keymap.altCR) {
				t.Fatalf("native command emitted %q, want %q", got, profile.keymap.altCR)
			}

			// Change only the captured composer's characters, retaining its
			// native cell styling and geometry. Equal-width prose must keep
			// the normal newline remap rather than inherit command authority.
			prose := cloneTerminalSnapshot(frame)
			region := profile.recognize(prose).region
			if !region.valid {
				t.Fatal("native composer region unavailable")
			}
			for i, r := range tc.prose {
				prose.Cells[region.top*prose.Width+region.left+i].Content = string(r)
			}
			if text, known := qualifiedComposerText(tc.agent, prose); !known || text != tc.prose {
				t.Fatal("native prose mutation did not preserve composer recognition")
			}
			incomplete := cloneTerminalSnapshot(frame)
			incomplete.Complete = false
			for name, snapshot := range map[string]terminalSnapshot{"prose": prose, "incomplete": incomplete} {
				d := decidePlainReturn(profile, false, &snapshot)
				if d.submits || !bytes.Equal(d.bytes, profile.keymap.plainCR) {
					t.Fatalf("%s gained native submit: %+v", name, d)
				}
			}
		})
	}
}
