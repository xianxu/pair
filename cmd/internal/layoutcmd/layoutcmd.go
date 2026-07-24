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
	layout, ok := toggleFocusedLayout(panes)
	if !ok {
		fmt.Fprintln(stderr, "pair layout toggle-focused: no focused Pair workbench pane")
		return 1
	}
	if err := rt.RunZellijAction("override-layout", "--apply-only-to-active-tab", "--layout-string", layout); err != nil {
		fmt.Fprintf(stderr, "pair layout toggle-focused: override layout: %v\n", err)
		return 1
	}
	return 0
}

func toggleFocusedLayout(panes []zellijpane.Pane) (string, bool) {
	var focused zellijpane.Pane
	leftWidth := 0
	rightWidth := 0
	draftRows := 12
	for _, pane := range panes {
		if pane.IsFloating || pane.IsPlugin {
			continue
		}
		role := workbenchshortcut.RoleForPane(pane)
		if pane.IsFocused {
			focused = pane
		}
		switch role {
		case workbenchshortcut.PaneRoleLeftAgent:
			if pane.Columns > leftWidth {
				leftWidth = pane.Columns
			}
		case workbenchshortcut.PaneRoleLeftDraft:
			if pane.Columns > leftWidth {
				leftWidth = pane.Columns
			}
			if pane.Rows > 0 {
				draftRows = pane.Rows
			}
		case workbenchshortcut.PaneRoleRightTerminal:
			if pane.Columns > rightWidth {
				rightWidth = pane.Columns
			}
		}
	}
	focusedRole := workbenchshortcut.RoleForPane(focused)
	if focusedRole != workbenchshortcut.PaneRoleLeftAgent &&
		focusedRole != workbenchshortcut.PaneRoleLeftDraft &&
		focusedRole != workbenchshortcut.PaneRoleRightTerminal {
		return "", false
	}
	total := leftWidth + rightWidth
	leftWide := total > 0 && leftWidth*100 >= total*60
	rightWide := total > 0 && rightWidth*100 >= total*60
	target := "balanced"
	if focusedRole == workbenchshortcut.PaneRoleRightTerminal {
		if !rightWide {
			target = "right"
		}
	} else if !leftWide {
		target = "left"
	}
	return workbenchLayout(target, draftRows), true
}

func workbenchLayout(widthMode string, draftRows int) string {
	leftSize := ""
	terminalSize := ""
	switch widthMode {
	case "left":
		leftSize = ` size="67%"`
	case "right":
		terminalSize = ` size="67%"`
	}
	return `layout {
    tab exact_panes=3 {
        pane split_direction="vertical" {
            pane` + leftSize + ` split_direction="horizontal" {
                pane name="agent"
                ` + draftPaneLine(draftRows) + `
            }
            pane` + terminalSize + ` name="terminal"
        }
    }
}
`
}

func draftPaneLine(rows int) string {
	if rows <= 2 {
		return `pane size=1 name="draft" borderless=true`
	}
	if rows <= 12 {
		return `pane size=12 name="draft" borderless=true`
	}
	return `pane size="33%" name="draft" borderless=true`
}

type OSRuntime struct{}

func (OSRuntime) ListPanesJSON() ([]byte, error) {
	return exec.Command("zellij", "action", "list-panes", "--json", "--command", "--state", "--geometry").Output()
}

func (OSRuntime) RunZellijAction(args ...string) error {
	cmdArgs := append([]string{"action"}, args...)
	return exec.Command("zellij", cmdArgs...).Run()
}
