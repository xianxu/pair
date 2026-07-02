package clipcmd

import (
	"fmt"
	"io"
	"time"

	"github.com/xianxu/pair/cmd/internal/zellijpane"
)

// Runtime is the IO/process boundary shared by the three clip helpers. The fs
// primitives (WriteFile / Executable) come from an embedded osfs.FS on the
// OSRuntime; the rest are the clipboard / zellij-IPC / spawn / exec seams.
//
// Two spawn modes are deliberately distinct (per the M4 plan-quality note):
// RunSubprocess call-and-returns (flash-pane must not block the focus change
// that follows), while ExecReplace replaces this process (the terminal
// clipboard-to-pane hand-off, the shell's `exec`).
type Runtime interface {
	WriteFile(path, data string) error // stage the quote file (osfs.FS)
	Executable(path string) bool       // `[ -x $PAIR_HOME/bin/flash-pane.sh ]` (osfs.FS)

	ClipboardCopy(text string) error        // pbcopy → wl-copy → xclip -i
	ClipboardPaste() (string, bool)         // pbpaste → wl-paste → xclip -o; ok=false when no tool
	ListPanes(command bool) (string, error) // `zellij action list-panes --json [--command]`

	SetPaneColor(id, bg string)                     // synchronous fg phase of the flash
	ResetPaneColorAfter(id string, d time.Duration) // detached (setsid) bg reset — survives caller exit
	FocusPane(id string) error                      // `zellij action focus-pane-id` (bare then terminal_ form)
	MoveFocus(dir string)                           // fallback when the nvim pane can't be resolved
	WriteKey(b byte)                                // `zellij action write <n>` (31 = Ctrl-_)

	RunSubprocess(path string, args ...string) error // flash-pane.sh (call-and-return)
	ExecReplace(path string, args ...string) error   // clipboard-to-pane.sh (process replace; only returns on error)

	Log(line string) // best-effort append to the clipboard-debug.log diagnostic
}

// CopyOnSelectOptions carries copy-on-select's resolved env.
type CopyOnSelectOptions struct {
	PairHome string
}

// RunCopyOnSelect is zellij's copy_command body: mirror the selection to the OS
// clipboard, and — unless the selection was made in the nvim draft pane — flash
// the source pane and hand off to clipboard-to-pane for the insert into nvim.
// Returns the process exit code; on the hand-off path it does not return in
// production (ExecReplace replaces the process).
func RunCopyOnSelect(opts CopyOnSelectOptions, stdin io.Reader, rt Runtime, stderr io.Writer) int {
	rt.Log("=== copy-on-select invoked ===")
	sel, _ := io.ReadAll(stdin)
	if len(sel) == 0 {
		rt.Log("empty sel, exiting")
		return 0
	}

	// 1. Mirror to the OS clipboard so other apps see it (best-effort).
	if err := rt.ClipboardCopy(string(sel)); err != nil {
		rt.Log("clipboard copy failed: " + err.Error())
	}

	// 2. Inspect the focused pane (where the selection happened).
	focusedID := ""
	inNvim := false
	if out, err := rt.ListPanes(true); err == nil {
		if p, ok := focusedPane(zellijpane.Parse([]byte(out))); ok {
			focusedID = p.ID
			inNvim = isNvimCommand(p.TerminalCommand)
		}
	}
	rt.Log(fmt.Sprintf("sel bytes: %d in_nvim: %v focused_id: %q", len(sel), inNvim, focusedID))

	// When the selection happened in nvim, skip flash + hand-off — otherwise it
	// would loop back and insert the selection beneath itself. (The "only paste
	// in insert mode" gate lives on the nvim side; see clipboard-to-pane.)
	if inNvim {
		return 0
	}

	// 3. Flash the source pane (delegated to flash-pane.sh so the flash idiom
	// lives in one place). Call-and-return: the flash's bg reset is detached, so
	// this doesn't block the hand-off's focus change.
	flashScript := opts.PairHome + "/bin/flash-pane.sh"
	if focusedID != "" && rt.Executable(flashScript) {
		if err := rt.RunSubprocess(flashScript, focusedID); err != nil {
			rt.Log("flash-pane failed: " + err.Error())
		}
	}

	// 4. Hand off to clipboard-to-pane.sh for the insert into nvim (it reads the
	// OS clipboard populated in step 1). This replaces the process (the shell's
	// `exec`); it only returns here on failure.
	clipScript := opts.PairHome + "/bin/clipboard-to-pane.sh"
	if err := rt.ExecReplace(clipScript); err != nil {
		fmt.Fprintf(stderr, "copy-on-select: exec %s: %v\n", clipScript, err)
		return 1
	}
	return 0
}

// ClipboardToPaneOptions carries clipboard-to-pane's resolved env.
type ClipboardToPaneOptions struct {
	DataDir     string
	XDGDataHome string
	Home        string
	Tag         string
	Agent       string
}

// RunClipboardToPane pulls the OS clipboard, stages it at $PAIR_DATA_DIR/quote-
// <tag>, focuses the nvim draft pane, and triggers PairPasteQuote via Ctrl-_.
// Formatting (par reflow, `> ` prefix vs inline) is decided in nvim, keyed on
// cursor position, so this only hands off the raw body.
func RunClipboardToPane(opts ClipboardToPaneOptions, rt Runtime, stderr io.Writer) int {
	rt.Log("=== clipboard-to-pane ===")
	clip, ok := rt.ClipboardPaste()
	if !ok {
		fmt.Fprintf(stderr, "clipboard-to-pane: no clipboard tool found\n")
		rt.Log("no clipboard tool found")
		return 1
	}
	if clip == "" {
		rt.Log("empty clipboard, exiting")
		return 0
	}

	// Find nvim's pane by terminal_command (its layout name is help text, not an
	// identifier), so we can target it explicitly — zellij's copy_command child
	// has no stable layout position for positional move-focus.
	nvimID := ""
	if out, err := rt.ListPanes(false); err == nil {
		if p, ok := nvimPane(zellijpane.Parse([]byte(out))); ok {
			nvimID = p.ID
		}
	}
	rt.Log(fmt.Sprintf("resolved nvim pane id: %q clip bytes: %d", nvimID, len(clip)))

	// Stage the raw selection for nvim to read.
	dataDir := pickDataDir(opts.DataDir, opts.XDGDataHome, opts.Home)
	tag := pickTag(opts.Tag, opts.Agent)
	qf := quoteFile(dataDir, tag)
	if err := rt.WriteFile(qf, clip); err != nil {
		fmt.Fprintf(stderr, "clipboard-to-pane: staging %s: %v\n", qf, err)
		return 1
	}

	// Target nvim, then trigger PairPasteQuote. `<C-_>` is mapped to
	// PairPasteQuote *only in insert mode* on the nvim side — that mapping IS the
	// gate, so we don't force-normal-mode here.
	if nvimID != "" {
		if err := rt.FocusPane(nvimID); err != nil {
			rt.Log("focus-pane-id failed for " + nvimID)
		}
	} else {
		rt.Log("could not resolve draft pane; falling back to move-focus down")
		rt.MoveFocus("down")
	}
	rt.WriteKey(31) // Ctrl-_
	rt.Log("triggered PairPasteQuote (Ctrl-_)")
	return 0
}

// FlashPaneOptions carries flash-pane's resolved env.
type FlashPaneOptions struct {
	PaneID  string // explicit pane id, or "" to flash the focused pane
	FlashBG string // PAIR_FLASH_BG override
	FlashMS string // PAIR_FLASH_MS override
}

// RunFlashPane flashes a zellij pane's background as a brief visual cue. The fg
// phase (set-pane-color) runs synchronously so a caller can chain a focus change
// immediately after; the bg reset is scheduled detached so it survives the
// caller exiting. Best-effort: many TUIs repaint their own bg on redraw, so the
// flash may be subtle.
func RunFlashPane(opts FlashPaneOptions, rt Runtime, stderr io.Writer) int {
	paneID := opts.PaneID
	if paneID == "" {
		if out, err := rt.ListPanes(true); err == nil {
			if p, ok := focusedPane(zellijpane.Parse([]byte(out))); ok {
				paneID = p.ID
			}
		}
	}
	if paneID == "" {
		rt.Log("flash-pane: no pane id")
		return 0
	}
	bg := pickFlashBG(opts.FlashBG)
	ms := pickFlashMS(opts.FlashMS)
	rt.SetPaneColor(paneID, bg)
	rt.ResetPaneColorAfter(paneID, time.Duration(ms)*time.Millisecond)
	rt.Log(fmt.Sprintf("flash-pane: bg=%s ms=%d id=%s", bg, ms, paneID))
	return 0
}
