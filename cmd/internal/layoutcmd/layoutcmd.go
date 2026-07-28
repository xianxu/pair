// Package layoutcmd owns Pair workbench layout operations that need zellij
// pane inspection before choosing an action.
package layoutcmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
	"github.com/xianxu/pair/cmd/internal/zellijpane"
)

// resizeSettleDelay is how long the toggle loop waits after issuing a resize
// before re-reading geometry — zellij applies resizes asynchronously (IO
// pacing, so it lives here rather than in the pure resizeplan.go).
const resizeSettleDelay = 80 * time.Millisecond

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
	panesJSON, err := rt.ListPanesJSON()
	if err != nil {
		return err
	}
	// Sidecar reads degrade gracefully by design: a missing/corrupt record or
	// registry must never break the focus jump — the picker just loses its
	// preference signal and falls back to zellij focus / pane order.
	lastTerminal, err := rt.LastTerminalPaneID()
	if err != nil {
		lastTerminal = ""
	}
	terminalIDs, err := rt.TerminalPaneIDs()
	if err != nil {
		terminalIDs = nil
	}
	terminal, ok := pickRightTerminal(zellijpane.Parse(panesJSON), lastTerminal, terminalIDs)
	if !ok {
		return rt.RunZellijAction("move-focus", "right")
	}
	return rt.RunZellijAction("focus-pane-id", terminal.ID)
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
// screen and two thirds. Zellij's only tiled resize primitive is a step of
// runtime-defined size, so the executor loops read-geometry → plan → act
// (planner in resizeplan.go) until the width converges, makes no progress
// (zellij refusing), or hits the step cap.
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
	target, ok := terminalResizeTarget(focused.Columns, screenCols)
	if !ok {
		return 0
	}
	current := focused.Columns
	for i := 0; i < terminalResizeMaxSteps; i++ {
		action, done := terminalResizeStep(current, target)
		if done {
			break
		}
		if err := rt.RunZellijAction(action...); err != nil {
			fmt.Fprintf(stderr, "pair layout toggle-focused: resize: %v\n", err)
			return 1
		}
		// zellij applies resizes asynchronously; without a settle pause the
		// re-read races the application and the no-progress guard stops the
		// loop short of the target (seen live: collapse stuck at ~55%).
		time.Sleep(resizeSettleDelay)
		panesJSON, err := rt.ListPanesJSON()
		if err != nil {
			fmt.Fprintf(stderr, "pair layout toggle-focused: list panes: %v\n", err)
			return 1
		}
		next, ok := focusedRightTerminal(zellijpane.Parse(panesJSON), terminalIDs)
		if !ok || abs(target-next.Columns) >= abs(target-current) {
			break
		}
		current = next.Columns
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
	reg := workbenchshortcut.TerminalPaneRegistry{DataDir: workbenchshortcut.DataDirFromEnv(), Tag: os.Getenv("PAIR_TAG")}
	return reg.LiveIDs(func(pid int) bool { return procutil.Alive(fmt.Sprintf("%d", pid)) })
}

func (OSRuntime) RunZellijAction(args ...string) error {
	cmdArgs := append([]string{"action"}, args...)
	return exec.Command("zellij", cmdArgs...).Run()
}
