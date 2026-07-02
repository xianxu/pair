package clipcmd

import (
	"io"
	"os"
	"path/filepath"
)

// RunCopyOnSelectCLI is the copy-on-select command body (zellij copy_command).
// The selection arrives on stdin.
func RunCopyOnSelectCLI(stdin io.Reader, getenv func(string) string, stderr io.Writer) int {
	home := getenv("PAIR_HOME")
	if home == "" {
		home = repoRootFromExe()
	}
	return RunCopyOnSelect(CopyOnSelectOptions{PairHome: home}, stdin, NewOSRuntime(), stderr)
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
