// Package draftroute sends Pair workbench actions directly to the draft
// Neovim pane without relying on focus choreography.
package draftroute

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/layoutcmd"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
	"github.com/xianxu/pair/cmd/internal/zellijpane"
	"github.com/xianxu/pair/cmd/internal/zellijreply"
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
	CurrentPaneID() string
	ListPanesJSONContext(context.Context) ([]byte, error)
	RunZellijAction(args ...string) error
}

func RouteLua(rt Runtime, function string, focusDraft bool) error {
	ctx, cancel := zellijreply.AdmissionContext(context.Background())
	defer cancel()
	return RouteLuaContext(ctx, rt, function, focusDraft)
}

// RouteLuaContext retains an enclosing shortcut's admission budget. A validated
// local cache needs no query; native fallback confirms the invoking tab first.
func RouteLuaContext(ctx context.Context, rt Runtime, function string, focusDraft bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	draftID, ok := rt.CachedDraftPaneID()
	if !ok {
		_, _, _, err := layoutcmd.ConfirmWorkbenchPanesContext(ctx, rt, rt.CurrentPaneID(), nil, func(panes []zellijpane.Pane) error {
			var err error
			draftID, err = UniqueDraftPaneID(panes)
			if err == nil && draftID == "" {
				return errors.New("draft pane not found")
			}
			return err
		})
		if err != nil {
			return fmt.Errorf("list panes: %w", err)
		}
	}
	return RouteLuaToPaneContext(ctx, rt, draftID, function, focusDraft)
}

// UniqueDraftPaneID selects at most one draft from a scoped and confirmed tab.
func UniqueDraftPaneID(panes []zellijpane.Pane) (string, error) {
	draftID := ""
	for _, pane := range panes {
		if workbenchshortcut.RoleForPane(pane) == workbenchshortcut.PaneRoleLeftDraft {
			if draftID != "" {
				return "", errors.New("ambiguous draft panes")
			}
			draftID = pane.ID
		}
	}
	return draftID, nil
}

// RouteLuaToPaneContext consumes the caller's admitted draft directly, avoiding
// another native lookup between admission and pane-addressed input.
func RouteLuaToPaneContext(ctx context.Context, rt interface{ RunZellijAction(...string) error }, draftID, function string, focusDraft bool) error {
	if draftID == "" {
		return errors.New("draft pane not found")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if focusDraft {
		if err := zellijreply.Effect(ctx, func() error { return rt.RunZellijAction("focus-pane-id", draftID) }); err != nil {
			return fmt.Errorf("focus draft pane %s: %w", draftID, err)
		}
	}
	for _, action := range [][]string{
		{"write", "--pane-id", draftID, "28"},
		{"write", "--pane-id", draftID, "14"},
		{"write-chars", "--pane-id", draftID, ":lua " + function + "()"},
		{"write", "--pane-id", draftID, "13"},
	} {
		if err := zellijreply.Effect(ctx, func() error { return rt.RunZellijAction(action...) }); err != nil {
			return fmt.Errorf("route %s to draft pane %s: %w", function, draftID, err)
		}
	}
	return nil
}
