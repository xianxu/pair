// Package layoutcmd owns Pair workbench layout operations that need zellij
// pane inspection before choosing an action.
package layoutcmd

import (
	"fmt"
	"io"
	"os/exec"

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
	focused, ok := focusedRightTerminal(panes)
	if !ok {
		return 0
	}
	leftWidth, terminalWidth := workbenchWidths(panes, focused.ID)
	if leftWidth == 0 || terminalWidth == 0 {
		return 0
	}
	targetNumerator, targetDenominator := 2, 3
	if terminalWidth*100 >= (leftWidth+terminalWidth)*60 {
		targetNumerator, targetDenominator = 1, 2
	}
	if err := reconcileTerminalWidth(focused.ID, targetNumerator, targetDenominator, panes, rt); err != nil {
		fmt.Fprintf(stderr, "pair layout toggle-focused: resize terminal: %v\n", err)
		return 1
	}
	return 0
}

func focusedRightTerminal(panes []zellijpane.Pane) (zellijpane.Pane, bool) {
	for _, pane := range panes {
		if pane.IsPlugin || !pane.IsFocused {
			continue
		}
		if pane.IsFloating || !isRightTerminal(pane) {
			return zellijpane.Pane{}, false
		}
		return pane, true
	}
	return zellijpane.Pane{}, false
}

func reconcileTerminalWidth(paneID string, targetNumerator, targetDenominator int, panes []zellijpane.Pane, rt Runtime) error {
	const maxAttempts = 20
	previousDifference := int(^uint(0) >> 1)
	lastResize := ""
	for range maxAttempts {
		leftWidth, terminalWidth := workbenchWidths(panes, paneID)
		if leftWidth == 0 || terminalWidth == 0 {
			return nil
		}
		totalWidth := leftWidth + terminalWidth
		delta := terminalWidth*targetDenominator - totalWidth*targetNumerator
		difference := delta
		if difference < 0 {
			difference = -difference
		}
		if difference <= totalWidth*targetDenominator/100 {
			return nil
		}
		if difference >= previousDifference {
			if lastResize != "" {
				inverse := "increase"
				if lastResize == "increase" {
					inverse = "decrease"
				}
				return rt.RunZellijAction("resize", inverse, "left", "--pane-id", paneID)
			}
			return nil
		}
		previousDifference = difference
		resize := "increase"
		if delta > 0 {
			resize = "decrease"
		}
		if err := rt.RunZellijAction("resize", resize, "left", "--pane-id", paneID); err != nil {
			return err
		}
		lastResize = resize
		panesJSON, err := rt.ListPanesJSON()
		if err != nil {
			return err
		}
		panes = zellijpane.Parse(panesJSON)
	}
	return nil
}

func workbenchWidths(panes []zellijpane.Pane, terminalPaneID string) (int, int) {
	var leftWidth, terminalWidth int
	for _, pane := range panes {
		if pane.IsPlugin || pane.IsFloating {
			continue
		}
		switch {
		case pane.ID == terminalPaneID:
			terminalWidth = pane.Columns
		case workbenchshortcut.RoleForPane(pane) == workbenchshortcut.PaneRoleLeftAgent,
			workbenchshortcut.RoleForPane(pane) == workbenchshortcut.PaneRoleLeftDraft:
			if pane.Columns > leftWidth {
				leftWidth = pane.Columns
			}
		}
	}
	return leftWidth, terminalWidth
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
