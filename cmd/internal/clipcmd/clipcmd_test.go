package clipcmd

import (
	"testing"

	"github.com/xianxu/pair/cmd/internal/zellijpane"
)

func TestIsNvimCommand(t *testing.T) {
	cases := []struct {
		name string
		cmd  string
		want bool
	}{
		{"nvim launch", `sh -c exec nvim -u /h/nvim/init.lua /d/draft-t.md`, true},
		{"draft in args", `sh -c exec vim /d/draft-t.md`, true},
		{"case-insensitive", `EXEC NVIM x`, true},
		// The #copy-on-select-test regression: the agent overwrites its pane
		// title with "claude [~/workspace/parley.nvim]", but the in_nvim gate
		// keys on terminal_command (the pair-wrap launch), which never embeds
		// the cwd — so this must NOT classify as nvim.
		{"parley.nvim agent cmd (no cwd)", `sh -c exec pair-wrap --scrollback-log /d/s.raw claude`, false},
		{"plain agent", `sh -c exec pair-wrap claude`, false},
		{"empty", ``, false},
	}
	for _, c := range cases {
		if got := isNvimCommand(c.cmd); got != c.want {
			t.Errorf("%s: isNvimCommand(%q) = %v, want %v", c.name, c.cmd, got, c.want)
		}
	}
}

func TestQuoteFile(t *testing.T) {
	if got := quoteFile("/data/dir", "t"); got != "/data/dir/quote-t" {
		t.Errorf("quoteFile = %q", got)
	}
}

func TestPickTag(t *testing.T) {
	if got := pickTag("tag", "agent"); got != "tag" {
		t.Errorf("PAIR_TAG wins: %q", got)
	}
	if got := pickTag("", "agent"); got != "agent" {
		t.Errorf("PAIR_AGENT fallback: %q", got)
	}
	if got := pickTag("", ""); got != "claude" {
		t.Errorf("default claude: %q", got)
	}
}

func TestPickDataDir(t *testing.T) {
	if got := pickDataDir("/dd", "/xdg", "/home"); got != "/dd" {
		t.Errorf("PAIR_DATA_DIR wins: %q", got)
	}
	if got := pickDataDir("", "/xdg", "/home"); got != "/xdg/pair" {
		t.Errorf("XDG fallback: %q", got)
	}
	if got := pickDataDir("", "", "/home"); got != "/home/.local/share/pair" {
		t.Errorf("HOME fallback: %q", got)
	}
}

func TestPickFlashDefaults(t *testing.T) {
	if got := pickFlashBG(""); got != "#50fa7b" {
		t.Errorf("default bg: %q", got)
	}
	if got := pickFlashBG("#fff"); got != "#fff" {
		t.Errorf("override bg: %q", got)
	}
	if got := pickFlashMS(""); got != 100 {
		t.Errorf("default ms: %d", got)
	}
	if got := pickFlashMS("250"); got != 250 {
		t.Errorf("override ms: %d", got)
	}
	if got := pickFlashMS("garbage"); got != 100 {
		t.Errorf("non-numeric ms → default: %d", got)
	}
}

// panes mirrors the copy-on-select-test fixture: a focused agent pane whose
// title contains "nvim" (parley.nvim cwd) but whose terminal_command is the
// pair-wrap launch, plus the real draft (nvim) pane, plus a focused floating
// plugin that must be ignored by focusedPane.
func fixturePanes() []zellijpane.Pane {
	return []zellijpane.Pane{
		{ID: "9", IsFocused: true, IsPlugin: true, IsFloating: true, Title: "About Zellij"},
		{ID: "0", IsFocused: true, Title: "claude [~/workspace/parley.nvim]",
			TerminalCommand: `sh -c exec pair-wrap --scrollback-log /d/s.raw claude`},
		{ID: "1", IsFocused: false, Title: "draft",
			TerminalCommand: `sh -c exec nvim -u /h/nvim/init.lua /d/draft-t.md`},
	}
}

func TestFocusedPaneSkipsFloatingPlugin(t *testing.T) {
	p, ok := focusedPane(fixturePanes())
	if !ok {
		t.Fatal("no focused pane found")
	}
	if p.ID != "0" {
		t.Errorf("focusedPane picked %q, want the agent pane 0 (not the focused floating plugin)", p.ID)
	}
}

func TestFocusedPaneNone(t *testing.T) {
	if _, ok := focusedPane([]zellijpane.Pane{{ID: "1", IsFocused: false}}); ok {
		t.Error("no focused pane should yield ok=false")
	}
}

func TestNvimPaneByTerminalCommand(t *testing.T) {
	p, ok := nvimPane(fixturePanes())
	if !ok {
		t.Fatal("no nvim pane found")
	}
	// Must pick the draft (terminal_command runs nvim), NOT the agent pane whose
	// TITLE merely contains "nvim".
	if p.ID != "1" {
		t.Errorf("nvimPane picked %q, want the draft pane 1", p.ID)
	}
}
