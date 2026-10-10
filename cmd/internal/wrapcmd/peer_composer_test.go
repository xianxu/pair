package wrapcmd

import (
	"fmt"
	"os"
	"strings"
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
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

func TestPeerComposerSharedFaintSuggestion(t *testing.T) {
	for _, agent := range []string{"claude", "codex"} {
		for _, style := range []string{"2", "0", "38;2;136;136;136"} {
			body := "\x1b[" + style + "many suggestion wording\x1b[0m"
			paint := claudeBox(5, "❯", "136;136;136", body) + "\x1b[?25h\x1b[7;3H"
			if agent == "codex" {
				paint = "\x1b[20;1H\x1b[1m›\x1b[22m " + body + "\x1b[?25h\x1b[20;3H"
			}
			s := peerSnapshot(t, paint)
			if empty := peerComposerState(agent, s) == PeerComposerEmpty; empty != (style == "2") {
				t.Errorf("%s style=%s empty=%v", agent, style, empty)
			}
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

// pair#418 (operator decision): Claude's collapsed paste marker proves our paste
// rendered only in its strict form. The marker is the composer's whole
// content, the cursor sits right after it, and its "+M lines" equals the
// envelope's newline count.
func TestPeerComposerClaudeCollapsedPasteStrict(t *testing.T) {
	raw, err := os.ReadFile("testdata/peer/claude/2.1.286/paste-multiline.raw")
	if err != nil {
		t.Fatal(err)
	}
	s := peerSnapshot(t, string(raw))
	expected := "[Couch peer from peer:0; delivery peer-live-conformance]\nReply PEER_SMOKE_OK only.\nDo not use tools.\nThis is harmless test text.\nFourth line.\nFifth line."
	if text, known := peerComposerText("claude", s); !known || text != "[Pasted text #1 +5 lines]" {
		t.Fatalf("captured collapsed marker changed: known=%t text=%q", known, text)
	}
	if !peerComposerMatches("claude", s, expected) {
		t.Fatal("captured collapsed marker with the envelope's line count did not match")
	}
	for _, wrong := range []string{
		strings.Replace(expected, "\nFifth line.", "", 1), // +4 lines
		expected + "\nSixth line.",                        // +6 lines
	} {
		if peerComposerMatches("claude", s, wrong) {
			t.Fatalf("marker accepted for a different line count: %q", wrong)
		}
	}
	twoLine := "[Couch peer from ariadne:1; delivery x]\n" + strings.Repeat("long body ", 110)
	for _, tc := range []struct {
		name  string
		paint string
		want  bool
	}{
		{"pair:1 capture shape", claudeBox(5, "❯", "136;136;136", "[Pasted text #3 +1 lines]") + "\x1b[?25h\x1b[7;28H", true},
		{"marker plus typed text", claudeBox(5, "❯", "136;136;136", "[Pasted text #3 +1 lines] and more") + "\x1b[?25h\x1b[7;38H", false},
		{"cursor not after marker", claudeBox(5, "❯", "136;136;136", "[Pasted text #3 +1 lines]") + "\x1b[?25h\x1b[7;10H", false},
		{"marker on a second line", claudeBox(5, "❯", "136;136;136", "hi", "[Pasted text #3 +1 lines]") + "\x1b[?25h\x1b[8;28H", false},
		{"image marker", claudeBox(5, "❯", "136;136;136", "[Image #1]") + "\x1b[?25h\x1b[7;13H", false},
	} {
		if got := peerComposerMatches("claude", peerSnapshot(t, tc.paint), twoLine); got != tc.want {
			t.Errorf("%s: matches=%v want %v", tc.name, got, tc.want)
		}
	}
	if peerClaudeCollapsedMarker.MatchString("[Pasted text #1 +1 line]") {
		t.Fatal("singular form was never observed and must not match")
	}
}

// pair#418: ariadne:2's composer on Claude Code 2.1.295 at 94 columns, captured
// from its scrollback. Claude wraps at spaces only, moving the hyphenated path
// whole to the next line where ansi.Wordwrap would break it after "ariadne-".
func TestPeerComposerClaudeCapturedHyphenWrap(t *testing.T) {
	lines := []string{
		"[Couch peer from ariadne:1; delivery 80a1b3c3-1d5d-478d-b2f6-a440e98adf8f]",
		"TL (ariadne:1) dispatch for project ariadne-robustness-1",
		"(workshop/projects/ariadne-robustness-1.md): please work on ariadne#300. Its branch was",
		"handed off from ariadne:1 and is pushed; claiming resumes there. Scope per the 2026-10-09",
		"Log entry and the project detail block: absorbs #271 (backgrounded reviewer, no verdict)",
		"plus evidence D2 (sandbox blocks the judge API; ledger records the failed round as passed)",
		"and D3 (30-minute timeout). Evidence:",
		"workshop/pensive/2026-10-09-01-pensive-sdlc-robustness-evidence.md, Part 2 section D. When",
		"done, #189 is next in this slot. Send blocking questions to ariadne:1.",
	}
	m, err := newTerminalModel(94, 39)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { m.Close() })
	last := lines[len(lines)-1]
	if err := m.Feed([]byte(claudeBox(5, "❯", "136;136;136", lines...) + fmt.Sprintf("\x1b[?25h\x1b[%d;%dH", 7+len(lines)-1, 3+len(last)))); err != nil {
		t.Fatal(err)
	}
	s := m.Snapshot()
	expected := lines[0] + "\n" + strings.Join(lines[1:], " ")
	if !peerComposerMatches("claude", s, expected) {
		text, known := peerComposerText("claude", s)
		t.Fatalf("captured hyphen wrap did not match: known=%t text=%q", known, text)
	}
	if ansi.Wordwrap(expected, s.Width-4, "") == strings.Join(lines, "\n") {
		t.Fatal("fixture no longer distinguishes hyphen breaking; it proves nothing")
	}
	if peerComposerMatches("claude", s, strings.Replace(expected, "Evidence: workshop", "Evidence:  workshop", 1)) {
		t.Fatal("changed whitespace accepted")
	}
}

func TestPeerSpaceWordwrap(t *testing.T) {
	for _, tc := range []struct {
		text  string
		width int
		want  string
		ok    bool
	}{
		{"aaa bbb ccc", 7, "aaa bbb\nccc", true},
		{"a-b-c d-e", 5, "a-b-c\nd-e", true},
		{"x\ny z", 3, "x\ny z", true},
		{"toolongword x", 5, "", false},
	} {
		got, ok := peerSpaceWordwrap(tc.text, tc.width)
		if got != tc.want || ok != tc.ok {
			t.Errorf("peerSpaceWordwrap(%q,%d)=%q,%v want %q,%v", tc.text, tc.width, got, ok, tc.want, tc.ok)
		}
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
