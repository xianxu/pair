// Package layoutcmd owns Pair workbench layout operations that need zellij
// pane inspection before choosing an action.
package layoutcmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"

	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
	"github.com/xianxu/pair/cmd/internal/zellijpane"
)

type Runtime interface {
	ListPanesJSON() ([]byte, error)
	RunZellijAction(args ...string) error
	// LastTerminalPaneID returns the recorded pane id of the split half the
	// user last left (empty when none recorded). In the tiled tree no right
	// terminal reports is_focused while focus sits in the left stack, so the
	// record is the only last-used-half memory the return jump has.
	LastTerminalPaneID() (string, error)
}

func AlignFloatingTerminal(rt Runtime) error {
	panesJSON, err := rt.ListPanesJSON()
	if err != nil {
		return err
	}
	panes := zellijpane.Parse(panesJSON)
	var terminal zellijpane.Pane
	for _, pane := range panes {
		if pane.IsFloating && isRightTerminal(pane) {
			terminal = pane
			break
		}
	}
	fillerX := terminalFillerX(panes)
	screenColumns, screenRows := tiledScreenSize(panes)
	if terminal.ID == "" || fillerX == 0 || screenColumns <= fillerX || screenRows == 0 {
		return nil
	}
	if terminal.X == fillerX && terminal.Columns == screenColumns-fillerX {
		return nil
	}
	return rt.RunZellijAction(
		"change-floating-pane-coordinates",
		"--pane-id", terminal.ID,
		"--x", strconv.Itoa(fillerX),
		"--y", "0",
		"--width", strconv.Itoa(screenColumns-fillerX),
		"--height", strconv.Itoa(screenRows),
		"--borderless", "false",
		"--pinned", "true",
	)
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
	lastTerminal, err := rt.LastTerminalPaneID()
	if err != nil {
		lastTerminal = ""
	}
	terminal, ok := pickRightTerminal(zellijpane.Parse(panesJSON), lastTerminal)
	if !ok {
		return rt.RunZellijAction("move-focus", "right")
	}
	return rt.RunZellijAction("focus-pane-id", terminal.ID)
}

// pickRightTerminal chooses among the tiled right terminals — after an
// Alt+Shift+d split there are two. A half actively holding focus wins;
// otherwise the recorded last-used half (written when Alt+k leaves the
// terminal side); otherwise the first in pane order.
func pickRightTerminal(panes []zellijpane.Pane, lastTerminalID string) (zellijpane.Pane, bool) {
	var recorded, first zellijpane.Pane
	var haveRecorded, found bool
	for _, pane := range panes {
		if pane.IsPlugin || !isRightTerminal(pane) {
			continue
		}
		if pane.IsFocused {
			return pane, true
		}
		if pane.ID == lastTerminalID {
			recorded, haveRecorded = pane, true
		}
		if !found {
			first, found = pane, true
		}
	}
	if haveRecorded {
		return recorded, true
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
	focused, ok := focusedRightTerminal(panes)
	if !ok {
		return 0
	}
	action := floatingTerminalCoordinates(focused, panes)
	if err := rt.RunZellijAction(action...); err != nil {
		fmt.Fprintf(stderr, "pair layout toggle-focused: change terminal coordinates: %v\n", err)
		return 1
	}
	return 0
}

func focusedRightTerminal(panes []zellijpane.Pane) (zellijpane.Pane, bool) {
	for _, pane := range panes {
		if pane.IsPlugin || !pane.IsFocused {
			continue
		}
		if !pane.IsFloating || !isRightTerminal(pane) {
			continue
		}
		return pane, true
	}
	return zellijpane.Pane{}, false
}

func floatingTerminalCoordinates(terminal zellijpane.Pane, panes []zellijpane.Pane) []string {
	x, y, width, height := "25%", "0%", "75%", "100%"
	screenColumns, screenRows := tiledScreenSize(panes)
	if screenColumns > 0 && screenRows > 0 {
		targetX, targetWidth := screenColumns/4, screenColumns-screenColumns/4
		if terminal.Columns*100 >= screenColumns*60 {
			if fillerX := terminalFillerX(panes); fillerX > 0 {
				targetX, targetWidth = fillerX, screenColumns-fillerX
			} else {
				targetX, targetWidth = screenColumns/2, screenColumns-screenColumns/2
			}
		}
		x, y = strconv.Itoa(targetX), "0"
		width, height = strconv.Itoa(targetWidth), strconv.Itoa(screenRows)
	}
	return []string{
		"change-floating-pane-coordinates",
		"--pane-id", terminal.ID,
		"--x", x,
		"--y", y,
		"--width", width,
		"--height", height,
		"--borderless", "false",
		"--pinned", "true",
	}
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

func terminalFillerX(panes []zellijpane.Pane) int {
	for _, pane := range panes {
		if pane.IsPlugin || pane.IsFloating {
			continue
		}
		if pane.Title == "terminal-filler" {
			return pane.X
		}
	}
	return 0
}

func isRightTerminal(pane zellijpane.Pane) bool {
	if pane.ID == "" {
		return false
	}
	pane.IsFloating = false
	return workbenchshortcut.RoleForPane(pane) == workbenchshortcut.PaneRoleRightTerminal
}

type OSRuntime struct{}

func (OSRuntime) ListPanesJSON() ([]byte, error) {
	return exec.Command("zellij", "action", "list-panes", "--json", "--command", "--state", "--geometry").Output()
}

func (OSRuntime) LastTerminalPaneID() (string, error) {
	store := workbenchshortcut.LastTerminalPaneStore{DataDir: workbenchshortcut.DataDirFromEnv(), Tag: os.Getenv("PAIR_TAG")}
	return store.Read()
}

func (OSRuntime) RunZellijAction(args ...string) error {
	cmdArgs := append([]string{"action"}, args...)
	return exec.Command("zellij", cmdArgs...).Run()
}
