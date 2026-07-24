// Package draftroute sends Pair workbench actions directly to the draft
// Neovim pane without relying on focus choreography.
package draftroute

import (
	"errors"
	"fmt"

	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
	"github.com/xianxu/pair/cmd/internal/zellijpane"
)

type Runtime interface {
	ListPanesJSON() ([]byte, error)
	RunZellijAction(args ...string) error
}

func RouteLua(rt Runtime, function string) error {
	data, err := rt.ListPanesJSON()
	if err != nil {
		return fmt.Errorf("list panes: %w", err)
	}
	var draftID string
	for _, pane := range zellijpane.Parse(data) {
		if workbenchshortcut.RoleForPane(pane) == workbenchshortcut.PaneRoleLeftDraft {
			draftID = pane.ID
			break
		}
	}
	if draftID == "" {
		return errors.New("draft pane not found")
	}
	for _, action := range [][]string{
		{"write", "--pane-id", draftID, "28"},
		{"write", "--pane-id", draftID, "14"},
		{"write-chars", "--pane-id", draftID, ":lua " + function + "()"},
		{"write", "--pane-id", draftID, "13"},
	} {
		if err := rt.RunZellijAction(action...); err != nil {
			return fmt.Errorf("route %s to draft pane %s: %w", function, draftID, err)
		}
	}
	return nil
}
