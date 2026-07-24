// Package layoutcmd owns Pair workbench layout operations that need zellij
// pane inspection before choosing an action.
package layoutcmd

import (
	"fmt"
	"io"
	"os/exec"
	"strconv"

	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
	"github.com/xianxu/pair/cmd/internal/zellijpane"
)

type Runtime interface {
	ListPanesJSON() ([]byte, error)
	RunZellijAction(args ...string) error
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
	actions, ok := toggleFocusedActions(panes)
	if !ok {
		return 0
	}
	for _, action := range actions {
		if err := rt.RunZellijAction(action...); err != nil {
			fmt.Fprintf(stderr, "pair layout toggle-focused: %s: %v\n", action[0], err)
			return 1
		}
	}
	return 0
}

func toggleFocusedActions(panes []zellijpane.Pane) ([][]string, bool) {
	var focused zellijpane.Pane
	for _, pane := range panes {
		if pane.IsPlugin || !pane.IsFocused {
			continue
		}
		focused = pane
		break
	}
	if !isRightTerminal(focused) {
		return nil, false
	}
	toggle := []string{"toggle-pane-embed-or-floating", "--pane-id", focused.ID}
	if focused.IsFloating {
		return [][]string{
			toggle,
			{"override-layout", "--apply-only-to-active-tab", "--layout-string", balancedLayout()},
		}, true
	}
	return [][]string{
		toggle,
		expandedTerminalCoordinates(focused.ID, panes),
	}, true
}

func expandedTerminalCoordinates(paneID string, panes []zellijpane.Pane) []string {
	x, y, width, height := "33%", "0%", "67%", "100%"
	if cols, rows := screenSize(panes); cols > 0 && rows > 0 {
		left := cols / 3
		x = strconv.Itoa(left)
		y = "0"
		width = strconv.Itoa(cols - left)
		height = strconv.Itoa(rows)
	}
	return []string{
		"change-floating-pane-coordinates",
		"--pane-id", paneID,
		"--x", x,
		"--y", y,
		"--width", width,
		"--height", height,
		"--borderless", "true",
		"--pinned", "true",
	}
}

func screenSize(panes []zellijpane.Pane) (int, int) {
	var cols, rows int
	for _, pane := range panes {
		if pane.IsPlugin || pane.IsFloating {
			continue
		}
		if right := pane.X + pane.Columns; right > cols {
			cols = right
		}
		if pane.Rows > rows {
			rows = pane.Rows
		}
	}
	return cols, rows
}

func balancedLayout() string {
	return `layout {
    tab exact_panes=3 {
        pane split_direction="vertical" {
            pane split_direction="horizontal" {
                pane name="agent"
                pane size=12 name="draft" borderless=true
            }
            pane name="terminal"
        }
    }
}
`
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

func (OSRuntime) RunZellijAction(args ...string) error {
	cmdArgs := append([]string{"action"}, args...)
	return exec.Command("zellij", cmdArgs...).Run()
}
