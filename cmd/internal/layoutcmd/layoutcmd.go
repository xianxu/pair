// Package layoutcmd owns Pair workbench layout operations that need zellij
// pane inspection before choosing an action.
package layoutcmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
	"github.com/xianxu/pair/cmd/internal/zellijpane"
	"strconv"
)

type Runtime interface {
	ListPanesJSON() ([]byte, error)
	RunZellijAction(args ...string) error
	// LastTerminalPaneID returns the recorded pane id of the split half the
	// user last left (empty when none recorded). In the tiled tree no right
	// terminal reports is_focused while focus sits in the left stack, so the
	// record is the only last-used-half memory the return jump has.
	LastTerminalPaneID() (string, error)
	// TerminalPaneIDs returns the live registered `pair term` pane ids —
	// required to recognize Alt+Shift+d split halves, whose zellij pane
	// report carries no terminal_command and a #118 tab-strip title.
	TerminalPaneIDs() ([]string, error)
}

// FocusRightTerminal focuses the tiled right workbench terminal by pane id.
// The id-based jump (never a relative `move-focus right`) predates the tiled
// pivot and stays: it is immune to whatever pane happens to sit between the
// caller and the terminal, and after an Alt+Shift+d split it is the only way
// to target a specific half. When no right terminal exists (layout2), fall
// back to the relative move so two-pane layouts keep their old behavior.
func FocusRightTerminal(rt Runtime) error {
	terminal, ok, err := resolveRightTerminal(rt)
	if err != nil {
		return err
	}
	if !ok {
		return rt.RunZellijAction("move-focus", "right")
	}
	return rt.RunZellijAction("focus-pane-id", terminal.ID)
}

// resolveFromSidecars answers "which right terminal?" from the two sidecars
// alone, or declines. Pure — no Runtime, no IO — mirroring the shape of
// draftroute.ValidateCachedDraftPane, which takes bytes plus an aliveness
// predicate and returns an id (ARCH-PURE).
//
// Declining is not failure. It means the zellij-focus tie-break is genuinely
// needed, and the caller should go spend the 590ms that `list-panes --json`
// costs on zellij 0.44.3 (#220). The registry is a SUBSET of the right
// terminals — a pane that has not self-registered yet is absent — so every
// branch here is conditional on it actually being able to answer.
func resolveFromSidecars(liveIDs []string, lastTerminal string) (string, bool) {
	if len(liveIDs) == 0 {
		return "", false // absence proves nothing about what exists
	}
	if lastTerminal != "" {
		for _, id := range liveIDs {
			if id == lastTerminal {
				return id, true
			}
		}
		// Recorded half is dead, or alive but unregistered. Either way the
		// registry cannot confirm it; let the pane list decide.
		return "", false
	}
	if len(liveIDs) == 1 {
		return liveIDs[0], true // one live half: nothing to tie-break
	}
	return "", false // two or more halves and no record: only the pane list chooses
}

// resolveRightTerminal answers "which right terminal does a workbench action
// mean?" — the pane list, the two sidecar preference signals, and the picker.
//
// Extracted because sharing only the PICKER left its inputs re-derived at each
// caller (#216 BR-10): the guarantee that Alt+k and Alt+Shift+arrow cannot
// disagree about a split half holds only while both feed it the same signals,
// and a fourth preference signal or a change of degradation policy would
// otherwise land at one site and miss the other.
//
// Sidecar reads degrade gracefully by design: a missing or corrupt record or
// registry must never break the action — the picker just loses its preference
// signal and falls back to zellij focus, then pane order.
func resolveRightTerminal(rt Runtime) (zellijpane.Pane, bool, error) {
	id, ok, err := resolveRightTerminalID(rt)
	if err != nil || !ok {
		return zellijpane.Pane{}, false, err
	}
	return zellijpane.Pane{ID: id}, true, nil
}

// resolveRightTerminalID is the IO shell: sidecars first, pane list only when
// they decline. Callers of this need the pane's ID and nothing else — the ones
// that need geometry (RunToggleFocused) still list panes unconditionally,
// because no sidecar carries geometry.
func resolveRightTerminalID(rt Runtime) (string, bool, error) {
	// Sidecar reads degrade gracefully: a missing record or registry costs the
	// resolution its fast path, never its correctness.
	lastTerminal, err := rt.LastTerminalPaneID()
	if err != nil {
		lastTerminal = ""
	}
	terminalIDs, err := rt.TerminalPaneIDs()
	if err != nil {
		terminalIDs = nil
	}
	if id, ok := resolveFromSidecars(terminalIDs, lastTerminal); ok {
		return id, true, nil
	}
	panesJSON, err := rt.ListPanesJSON()
	if err != nil {
		return "", false, err
	}
	terminal, ok := pickRightTerminal(zellijpane.Parse(panesJSON), lastTerminal, terminalIDs)
	if !ok {
		return "", false, nil
	}
	return terminal.ID, true, nil
}

// pickRightTerminal chooses among the tiled right terminals — after an
// Alt+Shift+d split there are two. The recorded last-used half (written by
// pair at the moment Alt+k left the terminal side) wins: it is pair-authored
// ground truth, whereas zellij's is_focused flag on unfocused-side panes is
// stale memory (live smoke: it pointed at the top half right after the user
// left the bottom one). zellij focus is the fallback signal, then pane order.
func pickRightTerminal(panes []zellijpane.Pane, lastTerminalID string, terminalPaneIDs []string) (zellijpane.Pane, bool) {
	var focused, first zellijpane.Pane
	var haveFocused, found bool
	for _, pane := range panes {
		if pane.IsPlugin || !isRightTerminal(pane, terminalPaneIDs) {
			continue
		}
		if pane.ID == lastTerminalID && lastTerminalID != "" {
			return pane, true
		}
		if pane.IsFocused && !haveFocused {
			focused, haveFocused = pane, true
		}
		if !found {
			first, found = pane, true
		}
	}
	if haveFocused {
		return focused, true
	}
	return first, found
}

// SwitchRightTerminalTab delivers a tab-switch chord to the right terminal
// pane WITHOUT moving focus, so the operator can check another tab from the
// draft without losing the cursor they are typing at (#216).
//
// It reuses pickRightTerminal rather than resolving the pane itself: after an
// Alt+Shift+d split there are two right terminals with independent tab sets,
// and if this picked differently from FocusRightTerminal then Alt+k would land
// in one half while Alt+Shift+arrow switched the other's tabs (ARCH-DRY).
//
// Delivery is fire-and-forget by construction: the pane is resolved and then
// written in a separate zellij action, so a terminal that exits in between
// yields a stale id and the write fails. That is the right trade — the
// alternative is an error for a pane the operator just closed.
func SwitchRightTerminalTab(rt Runtime, chord workbenchshortcut.Chord) error {
	terminal, ok, err := resolveRightTerminal(rt)
	if err != nil {
		return err
	}
	if !ok {
		// layout2, or a layout3 whose right pane exited: nothing to switch, and
		// nothing to report — the chord is simply inert.
		return nil
	}
	args, ok := workbenchshortcut.DeliverChordArgs(terminal.ID, chord)
	if !ok {
		return nil
	}
	return rt.RunZellijAction(args...)
}

// RunSwitchTerminalTab is the CLI entry the draft pane's Lua function calls, so
// pane resolution and chord encoding stay in Go rather than being restated in
// Lua (ARCH-DRY).
func RunSwitchTerminalTab(args []string, rt Runtime, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "usage: pair layout switch-terminal-tab prev|next")
		return 2
	}
	var action workbenchshortcut.ShortcutAction
	switch args[0] {
	case "prev":
		action = workbenchshortcut.ActionTerminalPrevTab
	case "next":
		action = workbenchshortcut.ActionTerminalNextTab
	default:
		fmt.Fprintf(stderr, "pair layout switch-terminal-tab: unknown direction %q (want prev|next)\n", args[0])
		return 2
	}
	// Through the same mapping the pane executors use, so the CLI cannot drift
	// from them about what "prev" delivers.
	chord, ok := workbenchshortcut.TabChordFor(action)
	if !ok {
		fmt.Fprintf(stderr, "pair layout switch-terminal-tab: no chord for %v\n", action)
		return 1
	}
	if err := SwitchRightTerminalTab(rt, chord); err != nil {
		fmt.Fprintf(stderr, "pair layout switch-terminal-tab: %v\n", err)
		return 1
	}
	return 0
}

func RunFocusTerminal(args []string, rt Runtime, stderr io.Writer) int {
	if len(args) > 0 {
		fmt.Fprintln(stderr, "usage: pair layout focus-terminal")
		return 2
	}
	if err := FocusRightTerminal(rt); err != nil {
		fmt.Fprintf(stderr, "pair layout focus-terminal: %v\n", err)
		return 1
	}
	return 0
}

// RunToggleFocused re-tiles the right terminal column between half the
// screen and ~two thirds by firing the planner's fixed three-step burst
// (resizeplan.go) back-to-back — no geometry re-reads, no pacing (live #124:
// consecutive resize actions all apply).
func RunToggleFocused(args []string, rt Runtime, stderr io.Writer) int {
	if len(args) > 0 {
		fmt.Fprintln(stderr, "usage: pair layout toggle-focused")
		return 2
	}
	panesJSON, err := rt.ListPanesJSON()
	if err != nil {
		fmt.Fprintf(stderr, "pair layout toggle-focused: list panes: %v\n", err)
		return 1
	}
	panes := zellijpane.Parse(panesJSON)
	// Graceful degradation as in FocusRightTerminal: a registry read error
	// only narrows classification to report-derived signals.
	terminalIDs, err := rt.TerminalPaneIDs()
	if err != nil {
		terminalIDs = nil
	}
	focused, ok := focusedRightTerminal(panes, terminalIDs)
	if !ok {
		return 0
	}
	screenCols, _ := tiledScreenSize(panes)
	burst, ok := terminalToggleBurst(focused.Columns, screenCols)
	if !ok {
		return 0
	}
	for _, action := range burst {
		if err := rt.RunZellijAction(action...); err != nil {
			fmt.Fprintf(stderr, "pair layout toggle-focused: resize: %v\n", err)
			return 1
		}
	}
	return 0
}

func focusedRightTerminal(panes []zellijpane.Pane, terminalPaneIDs []string) (zellijpane.Pane, bool) {
	for _, pane := range panes {
		if pane.IsPlugin || !pane.IsFocused || !isRightTerminal(pane, terminalPaneIDs) {
			continue
		}
		return pane, true
	}
	return zellijpane.Pane{}, false
}

func tiledScreenSize(panes []zellijpane.Pane) (int, int) {
	var columns, rows int
	for _, pane := range panes {
		if pane.IsPlugin || pane.IsFloating {
			continue
		}
		if right := pane.X + pane.Columns; right > columns {
			columns = right
		}
		if pane.Rows > rows {
			rows = pane.Rows
		}
	}
	return columns, rows
}

func isRightTerminal(pane zellijpane.Pane, terminalPaneIDs []string) bool {
	if pane.ID == "" {
		return false
	}
	return workbenchshortcut.RoleForPaneWith(pane, terminalPaneIDs) == workbenchshortcut.PaneRoleRightTerminal
}

type OSRuntime struct{}

func (OSRuntime) ListPanesJSON() ([]byte, error) {
	return exec.Command("zellij", "action", "list-panes", "--json", "--command", "--state", "--geometry").Output()
}

func (OSRuntime) LastTerminalPaneID() (string, error) {
	store := workbenchshortcut.LastTerminalPaneStore{DataDir: workbenchshortcut.DataDirFromEnv(), Tag: os.Getenv("PAIR_TAG")}
	return store.Read()
}

func (OSRuntime) TerminalPaneIDs() ([]string, error) {
	return workbenchshortcut.LiveTerminalPaneIDsFromEnv(func(pid int) bool {
		return procutil.Alive(strconv.Itoa(pid))
	})
}

func (OSRuntime) RunZellijAction(args ...string) error {
	cmdArgs := append([]string{"action"}, args...)
	return exec.Command("zellij", cmdArgs...).Run()
}
