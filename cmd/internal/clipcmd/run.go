package clipcmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/xianxu/pair/cmd/internal/zellijreply"
)

// Runtime is the IO/process boundary shared by the three clip helpers. The fs
// primitives (WriteFile) come from an embedded osfs.FS on the
// OSRuntime; the rest are clipboard, native commands and detached spawning.
//
// The fast copy hook detaches once. Its orchestrator admits the complete
// source/destination plan and executes it without another process handoff.
type Runtime interface {
	WriteFile(path, data string) error // stage the quote file (osfs.FS)

	ClipboardCopy(text string) error // pbcopy → wl-copy → xclip -i
	ClipboardPaste() (string, bool)  // pbpaste → wl-paste → xclip -o; ok=false when no tool
	ListPanesContext(context.Context) ([]byte, error)
	CurrentPaneID() string
	ListClientsContext(context.Context) ([]byte, error) // strict command/state/tab observations
	// TerminalPaneIDs returns the live registered `pair term` pane ids (#123's
	// registry). Needed because an Alt+Shift+d split half reports no
	// terminal_command and a "[terminal N]" title, so neither identifies it —
	// without this the right-pane gate below would miss split halves.
	TerminalPaneIDs() ([]string, error)

	SetPaneColor(id, bg string) error                        // synchronous fg phase of the flash
	ResetPaneColorAfter(id string, d time.Duration) error    // detached (setsid) bg reset — survives caller exit
	FocusPaneContext(context.Context, string) error          // one canonical explicit native target
	NotifyRefusal(string) error                              // independent existing wrapper broker
	WritePaneKeyContext(context.Context, string, byte) error // addressed write; 31 = Ctrl-_

	// SpawnDetached starts path in a new session (setsid) and does NOT wait — the
	// copy-on-select hook uses it to hand the flash+paste to a process that
	// outlives zellij's reap of the copy_command child (#100).
	SpawnDetached(path string, args ...string)

	// LogFresh truncates the clipboard-debug.log then writes line — called once
	// at the pipeline head (copy-on-select) so the diagnostic holds exactly one
	// selection's chain and doesn't grow unbounded. Log appends thereafter.
	LogFresh(line string)
	Log(line string) // best-effort append to the clipboard-debug.log diagnostic
}

// CopyOnSelectOptions carries copy-on-select's resolved env.
type CopyOnSelectOptions struct {
	PairHome string
	// SelfExe is the running pair executable, resolved at the CLI boundary
	// for the hook's single detached orchestrator spawn.
	SelfExe   string
	Clipboard ClipboardToPaneOptions
	Flash     FlashPaneOptions
}

// RunCopyOnSelect is zellij's copy_command body — the HOOK: mirror the selection
// to the OS clipboard, then hand the flash + paste to a DETACHED orchestrator and
// return immediately. Keeping the slow zellij round-trips out of the hook is what
// stops zellij reaping the copy_command child mid-paste (#100): the hook must
// return fast on any machine, cold or warm. The orchestrator (setsid) reads the
// selection back off the clipboard, so no data crosses the process boundary.
func RunCopyOnSelect(opts CopyOnSelectOptions, stdin io.Reader, rt Runtime, stderr io.Writer) int {
	rt.LogFresh("=== copy-on-select invoked ===")
	sel, _ := io.ReadAll(stdin)
	if len(sel) == 0 {
		rt.Log("empty sel, exiting")
		return 0
	}

	// Mirror to the OS clipboard so other apps see it AND so the detached
	// orchestrator can read it back (via clipboard-to-pane). Best-effort. This
	// completes before the spawn below, so the orchestrator's paste is populated.
	if err := rt.ClipboardCopy(string(sel)); err != nil {
		rt.Log("clipboard copy failed: " + err.Error())
	}

	// Detach the rest: the orchestrator (setsid) survives zellij's reap of this
	// copy_command child, so the paste completes even when the chain is slow.
	// The orchestrator decides whether to hand off at all — since #125 a
	// selection in the RIGHT TERMINAL does not auto-paste.
	rt.SpawnDetached(opts.SelfExe, "clip", "copy-on-select", "--orchestrate")
	rt.Log(fmt.Sprintf("sel bytes: %d — detached orchestrator spawned; hook returning", len(sel)))
	return 0
}

// RunCopyOnSelectOrchestrate is the detached second half (#100): inspect the
// focused pane, and — unless the selection was made in the nvim draft pane — flash
// the admitted source pane and paste into the admitted draft in process. Runs
// setsid'd (spawned by the hook), so zellij's reap of the copy_command child can't
// truncate it. The selection is already on the OS clipboard (the hook mirrored it),
// so this reads no stdin.
func RunCopyOnSelectOrchestrate(opts CopyOnSelectOptions, rt Runtime, stderr io.Writer) int {
	rt.Log("=== copy-on-select --orchestrate ===")
	ctx, cancel := zellijreply.AdmissionContext(context.Background())
	defer cancel()
	plan, err := admitAutoQuote(ctx, rt)
	if err != nil {
		return refuseClip(rt, stderr, "Copied; auto-quote refused: uncertain Zellij reply", err)
	}
	if plan.Skip {
		return 0
	}
	text, ok := rt.ClipboardPaste()
	if !ok {
		return refuseClip(rt, stderr, "Copied; auto-quote refused", fmt.Errorf("no clipboard tool found"))
	}
	if text == "" {
		return 0
	}
	if err := ctx.Err(); err != nil {
		return refuseClip(rt, stderr, "Copied; auto-quote refused: uncertain Zellij reply", err)
	}
	plan.Quote, err = quotePath(opts.Clipboard)
	if err != nil {
		return refuseClip(rt, stderr, "Copied; auto-quote refused", err)
	}
	if err := flashAdmitted(ctx, opts.Flash, plan.Source.ID, rt); err != nil {
		return refuseClip(rt, stderr, "Copied; auto-quote incomplete", err)
	}
	if err := pasteAdmitted(ctx, opts.Clipboard, plan, text, rt); err != nil {
		return refuseClip(rt, stderr, "Copied; auto-quote incomplete", err)
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
	ctx, cancel := zellijreply.AdmissionContext(context.Background())
	defer cancel()
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

	plan, err := admitClipboardPaste(ctx, rt)
	if err != nil {
		return refuseClip(rt, stderr, "Paste refused: uncertain Zellij reply", err)
	}
	if err := pasteAdmitted(ctx, opts, plan, clip, rt); err != nil {
		return refuseClip(rt, stderr, "Paste incomplete", err)
	}
	return 0
}

func pasteAdmitted(ctx context.Context, opts ClipboardToPaneOptions, plan pastePlan, text string, rt Runtime) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	qf := plan.Quote
	var err error
	if qf == "" {
		qf, err = quotePath(opts)
	}
	if err != nil {
		return fmt.Errorf("resolve quote artifact: %w", err)
	}
	if err := zellijreply.Effect(ctx, func() error { return rt.WriteFile(qf, text) }); err != nil {
		return fmt.Errorf("stage quote: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := zellijreply.Effect(ctx, func() error { return rt.FocusPaneContext(ctx, plan.Draft.ID) }); err != nil {
		return fmt.Errorf("focus draft: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := zellijreply.Effect(ctx, func() error { return rt.WritePaneKeyContext(ctx, plan.Draft.ID, 31) }); err != nil {
		return fmt.Errorf("trigger paste: %w", err)
	}
	return nil
}
func quotePath(opts ClipboardToPaneOptions) (string, error) {
	return quoteFile(pickDataDir(opts.DataDir, opts.XDGDataHome, opts.Home), pickTag(opts.Tag, opts.Agent))
}
func refuseClip(rt Runtime, stderr io.Writer, message string, reason error) int {
	rt.Log(message + ": " + reason.Error())
	fmt.Fprintf(stderr, "%s: %v\n", message, reason)
	if err := rt.NotifyRefusal(message); err != nil {
		rt.Log("refusal notification failed: " + err.Error())
		fmt.Fprintf(stderr, "refusal notification failed: %v\n", err)
	}
	return 1
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
	ctx := context.Background()
	if paneID == "" {
		var cancel context.CancelFunc
		ctx, cancel = zellijreply.AdmissionContext(ctx)
		defer cancel()
		plan, err := admitSource(ctx, rt)
		if err != nil {
			return refuseClip(rt, stderr, "Flash refused: uncertain Zellij reply", err)
		}
		if err := ctx.Err(); err != nil {
			return refuseClip(rt, stderr, "Flash refused: uncertain Zellij reply", err)
		}
		paneID = plan.Source.ID
	}
	if err := flashAdmitted(ctx, opts, paneID, rt); err != nil {
		return refuseClip(rt, stderr, "Flash incomplete", err)
	}
	return 0
}
func flashAdmitted(ctx context.Context, opts FlashPaneOptions, paneID string, rt Runtime) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	bg := pickFlashBG(opts.FlashBG)
	ms := pickFlashMS(opts.FlashMS)
	attempted := false
	applyErr := zellijreply.Effect(ctx, func() error {
		attempted = true
		return rt.SetPaneColor(paneID, bg)
	})
	if !attempted {
		return applyErr
	}
	// Issuing a temporary color creates a cleanup obligation even if the native
	// result is an error or admission expires during the command. Cleanup retains
	// its existing detached lifetime; cancellation forbids only subsequent effects.
	resetErr := rt.ResetPaneColorAfter(paneID, time.Duration(ms)*time.Millisecond)
	if applyErr != nil {
		applyErr = fmt.Errorf("apply flash: %w", applyErr)
	}
	if resetErr != nil {
		resetErr = fmt.Errorf("schedule flash reset: %w", resetErr)
	}
	rt.Log(fmt.Sprintf("flash-pane: bg=%s ms=%d id=%s", bg, ms, paneID))
	return errors.Join(applyErr, resetErr, ctx.Err())
}
