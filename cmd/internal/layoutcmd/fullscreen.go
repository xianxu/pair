package layoutcmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
	"github.com/xianxu/pair/cmd/internal/zellijpane"
)

type FullscreenOperation uint8

const (
	FullscreenNoop FullscreenOperation = iota
	FullscreenExpand
	FullscreenCollapse
)

type FullscreenPlan struct {
	Operation            FullscreenOperation
	TerminalID, ReturnID string
}

// PlanFullscreen selects the tiled right terminal a press acts on and the pane
// to return to; PlanRightPane turns its Expand into Normal→Focus and its
// Collapse into Maximize→Normal (#417). It uses the caller's identity because
// zellij can report several panes as focused after leaving fullscreen. Only
// zellij owns fullscreen state.
func PlanFullscreen(panes []zellijpane.Pane, caller, last string, registered []string, recorded string) (FullscreenPlan, error) {
	var terminals []zellijpane.Pane
	var fullscreen string
	for _, p := range panes {
		if p.IsPlugin || p.IsFloating || !isRightTerminal(p, registered) {
			continue
		}
		if p.IsFullscreen == nil {
			return FullscreenPlan{}, fmt.Errorf("pane %s has unknown fullscreen state", p.ID)
		}
		terminals = append(terminals, p)
		if *p.IsFullscreen {
			if fullscreen != "" {
				return FullscreenPlan{}, fmt.Errorf("ambiguous fullscreen panes")
			}
			fullscreen = p.ID
		}
	}
	if len(terminals) == 0 {
		return FullscreenPlan{}, nil
	}
	find := func(id string) (zellijpane.Pane, bool) {
		id = strings.TrimPrefix(id, "terminal_")
		if _, err := strconv.ParseUint(id, 10, 32); err != nil {
			return zellijpane.Pane{}, false
		}
		for _, p := range panes {
			if !p.IsPlugin && strings.TrimPrefix(p.ID, "terminal_") == id {
				return p, true
			}
		}
		return zellijpane.Pane{}, false
	}
	if fullscreen != "" {
		ret, ok := find(recorded)
		if !ok {
			for _, p := range panes {
				if !p.IsPlugin && workbenchshortcut.RoleForPane(p) == workbenchshortcut.PaneRoleLeftDraft {
					ret = p
					break
				}
			}
		}
		return FullscreenPlan{FullscreenCollapse, fullscreen, ret.ID}, nil
	}
	if caller == "" {
		for _, p := range panes {
			if !p.IsPlugin && p.IsFocused {
				if caller != "" {
					return FullscreenPlan{}, fmt.Errorf("ambiguous invoking pane")
				}
				caller = p.ID
			}
		}
	}
	invoking, ok := find(caller)
	if !ok {
		return FullscreenPlan{}, fmt.Errorf("invoking pane %q is not present", caller)
	}
	terminal, _ := pickRightTerminal(terminals, last, registered)
	for _, p := range terminals {
		if p.ID == invoking.ID {
			terminal = p
			break
		}
	}
	return FullscreenPlan{FullscreenExpand, terminal.ID, invoking.ID}, nil
}
