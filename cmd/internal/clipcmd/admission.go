package clipcmd

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
	"github.com/xianxu/pair/cmd/internal/zellijpane"
	"github.com/xianxu/pair/cmd/internal/zellijreply"
)

type pastePlan struct {
	Source, Draft zellijpane.Pane
	Skip          bool
	Quote         string
}
type sourceObservation struct{ Clients, Panes []byte }

func uniqueDraft(panes []zellijpane.Pane, tab *int) (zellijpane.Pane, error) {
	var found zellijpane.Pane
	for _, p := range panes {
		if p.IsPlugin || (tab != nil && (p.TabID == nil || *p.TabID != *tab)) || workbenchshortcut.RoleForPane(p) != workbenchshortcut.PaneRoleLeftDraft {
			continue
		}
		if found.ID != "" {
			return zellijpane.Pane{}, fmt.Errorf("draft pane is ambiguous")
		}
		found = p
	}
	if found.ID == "" {
		return found, fmt.Errorf("draft pane is missing")
	}
	return found, nil
}

// decisionPane excludes cosmetic title and dynamic actual commands. The selected
// layout command, role and tab identity are authority for this clipboard action.
func decisionPane(p zellijpane.Pane) string {
	b, _ := json.Marshal(struct {
		ID, Command   string
		Tab, Position *int
	}{p.ID, p.TerminalCommand, p.TabID, p.TabPosition})
	return string(b)
}
func admitClipboardPaste(ctx context.Context, rt Runtime) (pastePlan, error) {
	var result pastePlan
	callerID := strings.TrimPrefix(rt.CurrentPaneID(), "terminal_")
	if callerID == "" {
		return result, fmt.Errorf("invoking pane identity is missing")
	}
	_, err := zellijreply.Confirm(ctx, rt.ListPanesContext, func(raw []byte) (string, error) {
		panes, err := zellijreply.ParsePanes(raw, zellijreply.PaneRequirements{Command: true, Tab: true})
		if err != nil {
			return "", err
		}
		var caller zellijpane.Pane
		for _, p := range panes {
			if !p.IsPlugin && p.ID == callerID {
				caller = p
				break
			}
		}
		if caller.ID == "" {
			return "", fmt.Errorf("invoking pane is missing")
		}
		draft, err := uniqueDraft(panes, caller.TabID)
		if err != nil {
			return "", err
		}
		if *draft.TabPosition != *caller.TabPosition {
			return "", fmt.Errorf("draft tab position contradicts invoking pane")
		}
		result = pastePlan{Draft: draft}
		return decisionPane(caller) + "\n" + decisionPane(draft), nil
	})
	return result, err
}

// No-argument flash belongs to the independently known invoking pane. Client
// provenance queries are exclusive to the server-spawned auto-quote hook.
func admitSource(ctx context.Context, rt Runtime) (pastePlan, error) {
	var result pastePlan
	callerID := strings.TrimPrefix(rt.CurrentPaneID(), "terminal_")
	if callerID == "" {
		return result, fmt.Errorf("invoking pane identity is missing")
	}
	_, err := zellijreply.Confirm(ctx, rt.ListPanesContext, func(raw []byte) (string, error) {
		panes, err := zellijreply.ParsePanes(raw, zellijreply.PaneRequirements{Command: true, Tab: true})
		if err != nil {
			return "", err
		}
		for _, p := range panes {
			if !p.IsPlugin && p.ID == callerID {
				result = pastePlan{Source: p}
				return decisionPane(p), nil
			}
		}
		return "", fmt.Errorf("invoking pane is missing")
	})
	return result, err
}
func admitAutoQuote(ctx context.Context, rt Runtime) (pastePlan, error) {
	ids, err := rt.TerminalPaneIDs()
	if err != nil {
		return pastePlan{}, fmt.Errorf("terminal registry: %w", err)
	}
	var result pastePlan
	read := func(ctx context.Context) ([]byte, error) {
		clients, err := rt.ListClientsContext(ctx)
		if err != nil {
			return nil, err
		}
		parsed, err := zellijreply.ParseClients(clients)
		if err != nil {
			return nil, err
		}
		if _, err = clientSource(parsed); err != nil {
			return nil, err
		}
		if err = ctx.Err(); err != nil {
			return nil, err
		}
		panes, err := rt.ListPanesContext(ctx)
		if err != nil {
			return nil, err
		}
		return json.Marshal(sourceObservation{clients, panes})
	}
	_, err = zellijreply.Confirm(ctx, read, func(raw []byte) (string, error) {
		var observed sourceObservation
		if err := json.Unmarshal(raw, &observed); err != nil {
			return "", err
		}
		clients, err := zellijreply.ParseClients(observed.Clients)
		if err != nil {
			return "", err
		}
		sourceID, err := clientSource(clients)
		if err != nil {
			return "", err
		}
		panes, err := zellijreply.ParsePanes(observed.Panes, zellijreply.PaneRequirements{Command: true, Tab: true})
		if err != nil {
			return "", err
		}
		var source zellijpane.Pane
		for _, p := range panes {
			if !p.IsPlugin && "terminal_"+p.ID == sourceID {
				source = p
				break
			}
		}
		if source.ID == "" {
			return "", fmt.Errorf("copy source pane is missing")
		}
		role := workbenchshortcut.RoleForPaneWith(source, ids)
		skip := isNvimCommand(source.TerminalCommand) || role == workbenchshortcut.PaneRoleRightTerminal
		if !skip && role != workbenchshortcut.PaneRoleLeftAgent {
			return "", fmt.Errorf("copy source role is unconfirmed")
		}
		plan := pastePlan{Source: source, Skip: skip}
		if !skip {
			plan.Draft, err = uniqueDraft(panes, source.TabID)
			if err != nil {
				return "", err
			}
			if *plan.Draft.TabPosition != *source.TabPosition {
				return "", fmt.Errorf("draft tab position contradicts source")
			}
		}
		// Stable client membership is part of the observed source context; ordering is not.
		members := make([]int, len(clients))
		for i, c := range clients {
			members[i] = c.ID
		}
		sort.Ints(members)
		projection, _ := json.Marshal(struct {
			Clients       []int
			Source, Draft string
			Role          workbenchshortcut.PaneRole
			Skip          bool
		}{members, decisionPane(source), decisionPane(plan.Draft), role, skip})
		result = plan
		return string(projection), nil
	})
	return result, err
}

func clientSource(clients []zellijreply.Client) (string, error) {
	if len(clients) == 0 {
		return "", fmt.Errorf("copy source client is missing")
	}
	id := clients[0].PaneID
	if !strings.HasPrefix(id, "terminal_") {
		return "", fmt.Errorf("copy source is not a terminal")
	}
	for _, c := range clients {
		if c.PaneID != id {
			return "", fmt.Errorf("attached clients disagree on source")
		}
	}
	return id, nil
}
