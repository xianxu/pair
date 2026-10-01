package wrapcmd

import (
	uv "github.com/charmbracelet/ultraviolet"
	"os"
	"strings"
	"testing"
)

func peerSnapshot(t *testing.T, paint string) terminalSnapshot {
	t.Helper()
	m, err := newTerminalModel(120, 38)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { m.Close() })
	if err := m.Feed([]byte(paint)); err != nil {
		t.Fatal(err)
	}
	return m.Snapshot()
}
func TestPeerComposerConservativeText(t *testing.T) {
	for _, tc := range []struct {
		name, agent, paint, text string
		want                     PeerComposerState
	}{
		{"claude empty", "claude", claudeBox(5, "❯", "136;136;136", "") + "\x1b[?25h\x1b[7;3H", "", PeerComposerEmpty},
		{"claude occupied cursor before text", "claude", claudeBox(5, "❯", "136;136;136", "hello") + "\x1b[?25h\x1b[7;3H", "hello", PeerComposerOccupied},
		{"claude text below cursor", "claude", claudeBox(5, "❯", "136;136;136", "", "below") + "\x1b[?25h\x1b[7;3H", "\nbelow", PeerComposerOccupied},
		{"claude blank newline", "claude", claudeBox(5, "❯", "136;136;136", "", "") + "\x1b[?25h\x1b[8;3H", "\n", PeerComposerOccupied},
		{"claude shell", "claude", claudeBox(5, "!", "253;93;177", "") + "\x1b[?25h\x1b[7;3H", "", PeerComposerUnknown},
		{"claude slash", "claude", claudeBox(5, "❯", "136;136;136", "/help") + "\x1b[?25h\x1b[7;3H", "", PeerComposerUnknown},
		{"codex empty", "codex", "\x1b[20;1H\x1b[1m›\x1b[22m \x1b[?25h\x1b[20;3H", "", PeerComposerEmpty},
		{"codex text below cursor", "codex", "\x1b[20;1H\x1b[1m›\x1b[22m \x1b[21;3Hbelow\x1b[?25h\x1b[20;3H", "\nbelow", PeerComposerOccupied},
		{"codex gap before text", "codex", "\x1b[20;1H\x1b[1m›\x1b[22m \x1b[22;3Hbelow\x1b[?25h\x1b[20;3H", "\n\nbelow", PeerComposerOccupied},
		{"claude spaced text", "claude", claudeBox(5, "❯", "136;136;136", "hello world") + "\x1b[?25h\x1b[7;3H", "hello world", PeerComposerOccupied},
		{"codex menu below composer", "codex", "\x1b[20;1H\x1b[1m›\x1b[22m \x1b[21;1H› menu\x1b[?25h\x1b[20;3H", "", PeerComposerUnknown},
		{"codex image", "codex", "\x1b[20;1H\x1b[1m›\x1b[22m [Image #1]\x1b[?25h\x1b[20;3H", "[Image #1]", PeerComposerOccupied},
		{"unsupported", "muse", musePaintedComposer(">"), "", PeerComposerUnknown},
	} {
		t.Run(tc.name, func(t *testing.T) {
			snapshot := peerSnapshot(t, tc.paint)
			if got := peerComposerState(tc.agent, snapshot); got != tc.want {
				t.Fatalf("state %v want %v", got, tc.want)
			}
			text, ok := peerComposerText(tc.agent, snapshot)
			if ok != (tc.want != PeerComposerUnknown) || text != tc.text {
				t.Fatalf("text %q valid %v want %q", text, ok, tc.text)
			}
		})
	}
}

func TestPeerComposerCapturedFrames(t *testing.T) {
	for _, tc := range []struct{ agent, path string }{
		{"claude", "testdata/tty/claude/2.1.237/composer.raw"},
		{"codex", "testdata/orientation/codex/0.154.0/ready.raw"},
	} {
		raw, err := os.ReadFile(tc.path)
		if err != nil {
			t.Fatal(err)
		}
		s := peerSnapshot(t, string(raw))
		if got := peerComposerState(tc.agent, s); got != PeerComposerEmpty {
			t.Errorf("%s captured empty = %v", tc.agent, got)
		}
		// The same words in ordinary input styling must remain occupied.
		for i := range s.Cells {
			s.Cells[i].Style.Attrs &^= uv.AttrFaint
		}
		if got := peerComposerState(tc.agent, s); got == PeerComposerEmpty {
			t.Errorf("%s unstyled placeholder became empty", tc.agent)
		}
	}
}

func TestPeerComposerMatchesWrapAndWhitespace(t *testing.T) {
	body := "[Couch peer]\n" + strings.Repeat("x", 118) + "tail"
	s := peerSnapshot(t, claudeBox(5, "❯", "136;136;136", "[Couch peer]", strings.Repeat("x", 118), "tail")+"\x1b[?25h\x1b[9;7H")
	if !peerComposerMatches("claude", s, body) {
		t.Fatal("complete hardwrapped body did not match")
	}
	for _, wrong := range []string{body + " ", " " + body, strings.Replace(body, "peer", " peer", 1), body + "\n"} {
		if peerComposerMatches("claude", s, wrong) {
			t.Fatalf("changed body matched: %q", wrong)
		}
	}
}

func TestPeerComposerDeclinesCapturedStartupAndMenus(t *testing.T) {
	for _, path := range []string{"testdata/orientation/codex/0.154.0/loading.raw", "testdata/orientation/codex/0.154.0/trust.raw", "testdata/tty/codex/0.147.0/overlay.raw"} {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if got := peerComposerState("codex", peerSnapshot(t, string(raw))); got == PeerComposerEmpty {
			t.Fatalf("unsafe frame %s accepted", path)
		}
	}
}
