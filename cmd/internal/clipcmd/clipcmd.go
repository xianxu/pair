// Package clipcmd is the copy-on-select clipboard pipeline (#93 M4, ported from
// bin/{copy-on-select,clipboard-to-pane,flash-pane}.sh). The pure decisions —
// in-nvim classification, pane selection, quote-file / flash-default resolution
// — live here; everything touching the OS clipboard, zellij IPC, process
// spawn/exec, or fs sits behind the Runtime seam (runtime.go) so the three
// orchestrations are unit-testable with a fake.
//
// The three helpers stay a chain: copy-on-select mirrors the selection to the
// OS clipboard, and (unless the selection was made in the nvim draft pane)
// flashes the source pane and hands off to clipboard-to-pane, which stages the
// text and triggers nvim's PairPasteQuote. The hand-off keeps exec'ing the
// bin/*.sh names so the by-path stubs in tests/copy-on-select-test.sh still
// drive the real chain.
package clipcmd

import (
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/xianxu/pair/cmd/internal/zellijpane"
)

// Flash defaults mirror flash-pane.sh: dracula "green" (the active-frame color
// in the user's zellij theme) for 100ms, both overridable via PAIR_FLASH_*.
const (
	defaultFlashBG = "#50fa7b"
	defaultFlashMS = 100
)

// nvimCmdRe is the copy-on-select in_nvim gate: the FOCUSED pane's
// terminal_command (NOT its title) is matched case-insensitively against
// nvim|draft. Keying on terminal_command is the #copy-on-select-test fix — the
// agent overwrites its pane title with "claude [<cwd>]", so a repo whose path
// contains "nvim" (e.g. parley.nvim) would misclassify the agent pane as the
// draft and skip the paste. terminal_command never embeds the cwd.
var nvimCmdRe = regexp.MustCompile(`(?i)nvim|draft`)

func isNvimCommand(cmd string) bool { return nvimCmdRe.MatchString(cmd) }

// focusedPane picks the pane the selection happened in: focused, non-plugin,
// non-floating. The plugin/floating filter matters because zellij reports BOTH
// a focused floating plugin (e.g. "About Zellij") AND the underlying terminal
// as is_focused=true; without it we'd pick the plugin and misclassify in_nvim /
// flash the wrong terminal id.
func focusedPane(panes []zellijpane.Pane) (zellijpane.Pane, bool) {
	for _, p := range panes {
		if p.IsFocused && !p.IsPlugin && !p.IsFloating {
			return p, true
		}
	}
	return zellijpane.Pane{}, false
}

// nvimPane picks the draft pane for the clipboard-to-pane hand-off: the first
// pane whose terminal_command contains "nvim" (the shell's `test("nvim")`,
// case-sensitive — matches `exec nvim …`). Not filtered on is_plugin/is_floating
// to stay faithful to clipboard-to-pane.sh.
func nvimPane(panes []zellijpane.Pane) (zellijpane.Pane, bool) {
	for _, p := range panes {
		if nvimNameRe.MatchString(p.TerminalCommand) {
			return p, true
		}
	}
	return zellijpane.Pane{}, false
}

var nvimNameRe = regexp.MustCompile(`nvim`)

// quoteFile is where clipboard-to-pane stages the raw selection for nvim's
// PairPasteQuote to read ($PAIR_DATA_DIR/quote-<tag>).
func quoteFile(dataDir, tag string) string {
	return filepath.Join(dataDir, "quote-"+tag)
}

// pickTag resolves the pane tag the quote file is keyed by: PAIR_TAG, else
// PAIR_AGENT, else "claude".
func pickTag(pairTag, pairAgent string) string {
	if pairTag != "" {
		return pairTag
	}
	if pairAgent != "" {
		return pairAgent
	}
	return "claude"
}

// pickDataDir mirrors clipboard-to-pane.sh's data-dir resolution: PAIR_DATA_DIR,
// else $XDG_DATA_HOME/pair, else $HOME/.local/share/pair.
func pickDataDir(pairDataDir, xdgDataHome, home string) string {
	if pairDataDir != "" {
		return pairDataDir
	}
	if xdgDataHome != "" {
		return filepath.Join(xdgDataHome, "pair")
	}
	return filepath.Join(home, ".local", "share", "pair")
}

func pickFlashBG(env string) string {
	if env != "" {
		return env
	}
	return defaultFlashBG
}

// pickFlashMS parses PAIR_FLASH_MS; a missing or non-numeric value falls back to
// the 100ms default.
func pickFlashMS(env string) int {
	if n, err := strconv.Atoi(env); err == nil && n >= 0 {
		return n
	}
	return defaultFlashMS
}
