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
		{"claude", "testdata/peer/claude/2.1.286/startup-fix-lint.raw"},
		{"claude", "testdata/peer/claude/2.1.286/startup-write-test.raw"},
		{"claude", "testdata/peer/claude/2.1.286/startup-create-util.raw"},
		{"codex", "testdata/orientation/codex/0.154.0/ready.raw"},
		{"codex", "testdata/peer/codex/0.159.2/startup.raw"},
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

func TestPeerComposerModernCodexStartup(t *testing.T) {
	// v0.159.2 replaces boxed model:/directory: fields with title + bare path.
	paint := "\x1b[1;1H>_ OpenAI Codex (v0.159.2)\x1b[3;1H/private/tmp/project\x1b[10;1H\x1b[1m›\x1b[22m \x1b[2mAsk Codex to do anything\x1b[22m\x1b[12;3HGPT-6.1-Sol default · /private/tmp/project\x1b[?25h\x1b[10;3H"
	if got := peerComposerState("codex", peerSnapshot(t, paint)); got != PeerComposerEmpty {
		t.Fatalf("resolved modern startup = %v", got)
	}
	for _, unsafe := range []string{
		strings.Replace(paint, "/private/tmp/project", "loading", 1),
		strings.Replace(paint, "GPT-6.1-Sol default · /private/tmp/project", "loading", 1),
	} {
		if got := peerComposerState("codex", peerSnapshot(t, unsafe)); got == PeerComposerEmpty {
			t.Fatal("unresolved startup accepted")
		}
	}
}

func TestPeerComposerModernCodexCapturedPasteMatches(t *testing.T) {
	raw, err := os.ReadFile("testdata/peer/codex/0.159.2/paste-short.raw")
	if err != nil {
		t.Fatal(err)
	}
	s := peerSnapshot(t, string(raw))
	expected := "[Couch peer from peer:0; delivery peer-live-conformance]\nReply PEER_SMOKE_OK only. Do not use tools."
	if !peerComposerMatches("codex", s, expected) {
		text, known := peerComposerText("codex", s)
		t.Fatalf("captured paste mismatch known=%t text=%q", known, text)
	}
	if peerComposerMatches("codex", s, expected+" ") {
		t.Fatal("changed content matched")
	}
}

func TestPeerComposerModernCodexCapturedWordwrap(t *testing.T) {
	raw, err := os.ReadFile("testdata/peer/codex/0.159.2/paste-wrapped.raw")
	if err != nil {
		t.Fatal(err)
	}
	s := peerSnapshot(t, string(raw))
	expected := "[Couch peer from peer:0; delivery peer-live-conformance]\nDo not use tools. " + strings.TrimSuffix(strings.Repeat("harmless wrapped text ", 14), " ")
	if !peerComposerMatches("codex", s, expected) {
		actual, known := peerComposerText("codex", s)
		t.Fatalf("wordwrap match failed known=%t actual=%q expected=%q", known, actual, expected)
	}
	for _, changed := range []string{expected + " ", " " + expected, strings.Replace(expected, "harmless wrapped", "harmless  wrapped", 1), strings.Replace(expected, "harmless wrapped", "harmless\twrapped", 1)} {
		if peerComposerMatches("codex", s, changed) {
			t.Fatalf("ambiguous altered whitespace matched: %q", changed)
		}
	}
}

func TestPeerComposerClaudeGhostShape(t *testing.T) {
	for _, tc := range []struct {
		name, text string
		faint      bool
		cursor     string
		empty      bool
	}{
		{"rotating hint", `Try "explain this code"`, true, "\x1b[7;3H", true},
		{"ordinary style", `Try "explain this code"`, false, "\x1b[7;3H", false},
		{"suggested next prompt", `check if parley.nvim:0 replied`, true, "\x1b[7;3H", true},
		{"suggestion with quotes", `explain "this" code`, true, "\x1b[7;3H", true},
		{"ordinary draft at origin", `check if parley.nvim:0 replied`, false, "\x1b[7;3H", false},
		{"wrong cursor", `Try "explain this code"`, true, "\x1b[7;4H", false},
		{"empty quote", `Try ""`, true, "\x1b[7;3H", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := tc.text
			if tc.faint {
				body = "\x1b[2m" + body + "\x1b[22m"
			}
			s := peerSnapshot(t, claudeBox(5, "❯", "136;136;136", body)+"\x1b[?25h"+tc.cursor)
			if empty := peerComposerState("claude", s) == PeerComposerEmpty; empty != tc.empty {
				t.Fatalf("empty=%t want=%t", empty, tc.empty)
			}
		})
	}
	s := peerSnapshot(t, claudeBox(5, "❯", "136;136;136", "\x1b[2mTry \"one\"\x1b[22m", "\x1b[2mextra\x1b[22m")+"\x1b[?25h\x1b[7;3H")
	if peerComposerState("claude", s) == PeerComposerEmpty {
		t.Fatal("multiline faint text accepted as ghost")
	}
}

func TestPeerComposerClaudeSuggestionDoesNotDependOnColor(t *testing.T) {
	for _, color := range []string{"39", "38;2;220;220;220", "38;2;40;40;40"} {
		for _, faint := range []bool{false, true} {
			body := "\x1b[" + color + "m"
			if faint {
				body += "\x1b[2m"
			}
			body += "suggested next prompt\x1b[0m"
			s := peerSnapshot(t, claudeBox(5, "❯", "136;136;136", body)+"\x1b[?25h\x1b[7;3H")
			if empty := peerComposerState("claude", s) == PeerComposerEmpty; empty != faint {
				t.Fatalf("color=%s faint=%v empty=%v", color, faint, empty)
			}
		}
	}
}

func TestPeerComposerClaudeCapturedPasteMatches(t *testing.T) {
	raw, err := os.ReadFile("testdata/peer/claude/2.1.286/paste-short.raw")
	if err != nil {
		t.Fatal(err)
	}
	s := peerSnapshot(t, string(raw))
	expected := "[Couch peer from peer:0; delivery peer-live-conformance]\nReply PEER_SMOKE_OK only. Do not use tools."
	if !peerComposerMatches("claude", s, expected) {
		text, known := peerComposerText("claude", s)
		t.Fatalf("captured paste mismatch known=%t text=%q", known, text)
	}
}

func TestPeerComposerClaudeCollapsedPasteRemainsUnsupported(t *testing.T) {
	raw, err := os.ReadFile("testdata/peer/claude/2.1.286/paste-multiline.raw")
	if err != nil {
		t.Fatal(err)
	}
	s := peerSnapshot(t, string(raw))
	expected := "[Couch peer from peer:0; delivery peer-live-conformance]\nReply PEER_SMOKE_OK only.\nDo not use tools.\nThis is harmless test text.\nFourth line.\nFifth line."
	text, known := peerComposerText("claude", s)
	if !known || !strings.Contains(text, "[Pasted") {
		t.Fatalf("expected captured collapsed marker, known=%t text=%q", known, text)
	}
	if peerComposerMatches("claude", s, expected) {
		t.Fatal("collapsed marker falsely proves complete body")
	}
}

func TestPeerComposerClaudeCapturedWordwrap(t *testing.T) {
	raw, err := os.ReadFile("testdata/peer/claude/2.1.286/paste-wrapped.raw")
	if err != nil {
		t.Fatal(err)
	}
	s := peerSnapshot(t, string(raw))
	expected := "[Couch peer from peer:0; delivery peer-live-conformance]\nDo not use tools. " + strings.TrimSuffix(strings.Repeat("harmless wrapped text ", 14), " ")
	if !peerComposerMatches("claude", s, expected) {
		actual, known := peerComposerText("claude", s)
		t.Fatalf("Claude wordwrap mismatch known=%t actual=%q", known, actual)
	}
	for _, changed := range []string{expected + " ", " " + expected, strings.Replace(expected, "harmless wrapped", "harmless  wrapped", 1)} {
		if peerComposerMatches("claude", s, changed) {
			t.Fatalf("changed whitespace accepted: %q", changed)
		}
	}
}
