package clipcmd

import (
	"io"
	"os"
	"path/filepath"
)

// RunCopyOnSelectCLI is the copy-on-select command body. Invoked two ways: by
// zellij's copy_command with the selection on stdin (the fast HOOK), and by the
// hook itself as `copy-on-select --orchestrate` (the detached paste orchestrator,
// #100). The hook mirrors the selection and spawns the orchestrator; the
// orchestrator does the slow flash + hand-off out from under zellij's reap.
func RunCopyOnSelectCLI(args []string, stdin io.Reader, getenv func(string) string, stderr io.Writer) int {
	home := getenv("PAIR_HOME")
	if home == "" {
		home = repoRootFromExe()
	}
	opts := CopyOnSelectOptions{PairHome: home, SelfExe: selfPairExe(home)}
	if len(args) > 0 && args[0] == "--orchestrate" {
		return RunCopyOnSelectOrchestrate(opts, NewOSRuntime(), stderr)
	}
	return RunCopyOnSelect(opts, stdin, NewOSRuntime(), stderr)
}

// selfPairExe resolves the `pair` executable for the self-exec hand-offs
// (`pair clip …`). It targets the `pair` SIBLING of the running binary
// (dir(os.Executable())/pair): in production the hook runs as `pair` so the
// sibling is pair itself, and it works in the copied/Homebrew layout (pair is
// never in its own $PAIR_HOME/bin bundle). Resolving the sibling — rather than
// os.Executable() — also keeps the flow correct if invoked under a helper's name.
// Falls back to <home>/bin/pair only if os.Executable() is unavailable.
func selfPairExe(home string) string {
	if exe, err := os.Executable(); err == nil && exe != "" {
		return filepath.Join(filepath.Dir(exe), "pair")
	}
	return filepath.Join(home, "bin", "pair")
}

// RunClipboardToPaneCLI is the clipboard-to-pane command body.
func RunClipboardToPaneCLI(getenv func(string) string, stderr io.Writer) int {
	return RunClipboardToPane(ClipboardToPaneOptions{
		DataDir:     getenv("PAIR_DATA_DIR"),
		XDGDataHome: getenv("XDG_DATA_HOME"),
		Home:        getenv("HOME"),
		Tag:         getenv("PAIR_TAG"),
		Agent:       getenv("PAIR_AGENT"),
	}, NewOSRuntime(), stderr)
}

// RunFlashPaneCLI is the flash-pane command body: `flash-pane [<pane-id>]`.
func RunFlashPaneCLI(args []string, getenv func(string) string, stderr io.Writer) int {
	paneID := ""
	if len(args) > 0 {
		paneID = args[0]
	}
	return RunFlashPane(FlashPaneOptions{
		PaneID:  paneID,
		FlashBG: getenv("PAIR_FLASH_BG"),
		FlashMS: getenv("PAIR_FLASH_MS"),
	}, NewOSRuntime(), stderr)
}

// repoRootFromExe mirrors the shims' PAIR_HOME resolution for the case the Go
// binary is run directly (dev): it lives at <root>/bin/copy-on-select.
func repoRootFromExe() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(filepath.Dir(exe))
}
