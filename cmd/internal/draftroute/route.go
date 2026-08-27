// Package draftroute sends Pair workbench actions directly to the draft
// Neovim pane without relying on focus choreography.
package draftroute

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
	"github.com/xianxu/pair/cmd/internal/zellijpane"
)

type CachedPaneRecord struct {
	Session string    `json:"session"`
	PaneID  string    `json:"pane_id"`
	PID     ProcessID `json:"pid"`
}

// ProcessID accepts the numeric JSON emitted by Neovim's getpid() and the
// string form used by older tests/cache records.
type ProcessID string

func (p *ProcessID) UnmarshalJSON(data []byte) error {
	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		*p = ProcessID(text)
		return nil
	}
	var number json.Number
	if err := json.Unmarshal(data, &number); err != nil {
		return err
	}
	if _, err := number.Int64(); err != nil {
		return err
	}
	*p = ProcessID(number.String())
	return nil
}

func ValidateCachedDraftPane(data []byte, session string, alive func(string) bool) (string, bool) {
	var record CachedPaneRecord
	if json.Unmarshal(data, &record) != nil ||
		record.Session == "" || record.Session != session ||
		record.PaneID == "" || !alive(string(record.PID)) {
		return "", false
	}
	return record.PaneID, true
}

func CachedDraftPaneIDFromEnv() (string, bool) {
	dataDir := os.Getenv("PAIR_DATA_DIR")
	tag := os.Getenv("PAIR_TAG")
	session := os.Getenv("ZELLIJ_SESSION_NAME")
	if dataDir == "" || tag == "" || session == "" {
		return "", false
	}
	paths, err := artifactpath.ResolveScoped(dataDir, tag)
	if err != nil {
		return "", false
	}
	data, err := os.ReadFile(paths.DraftPane())
	if err != nil {
		return "", false
	}
	return ValidateCachedDraftPane(data, session, procutil.Alive)
}

type Runtime interface {
	CachedDraftPaneID() (string, bool)
	ListPanesJSON() ([]byte, error)
	RunZellijAction(args ...string) error
}

func RouteLua(rt Runtime, function string, focusDraft bool) error {
	draftID, ok := rt.CachedDraftPaneID()
	if !ok {
		data, err := rt.ListPanesJSON()
		if err != nil {
			return fmt.Errorf("list panes: %w", err)
		}
		for _, pane := range zellijpane.Parse(data) {
			if workbenchshortcut.RoleForPane(pane) == workbenchshortcut.PaneRoleLeftDraft {
				draftID = pane.ID
				break
			}
		}
	}
	if draftID == "" {
		return errors.New("draft pane not found")
	}
	if focusDraft {
		if err := rt.RunZellijAction("focus-pane-id", draftID); err != nil {
			return fmt.Errorf("focus draft pane %s: %w", draftID, err)
		}
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
