package layoutcmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
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

// PlanFullscreen uses the caller's identity because zellij can report several
// panes as focused after leaving fullscreen. Only zellij owns fullscreen state.
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

type FullscreenEvent uint8

const (
	FullscreenBegin FullscreenEvent = iota
	FullscreenConfirmed
	FullscreenFailed
	FullscreenUnconfirmed
)

type FullscreenEffect uint8

const (
	FullscreenNothing FullscreenEffect = iota
	FullscreenSave
	FullscreenToggle
	FullscreenFocus
	FullscreenClear
)

type fullscreenPhase uint8

const (
	fullscreenInitial fullscreenPhase = iota
	fullscreenSaving
	fullscreenToggling
	fullscreenFocusing
	fullscreenClearing
	fullscreenDone
	fullscreenStopped
)

type FullscreenState struct {
	plan  FullscreenPlan
	phase fullscreenPhase
}

func NewFullscreenState(plan FullscreenPlan) FullscreenState { return FullscreenState{plan: plan} }

// FullscreenTransition is the sole owner of operation ordering. Failed or
// unconfirmed effects stop the invocation; they never authorize an inverse toggle.
func FullscreenTransition(s FullscreenState, event FullscreenEvent) (FullscreenState, FullscreenEffect) {
	if s.phase >= fullscreenDone {
		return s, FullscreenNothing
	}
	if s.phase != fullscreenInitial && (event == FullscreenFailed || event == FullscreenUnconfirmed) {
		s.phase = fullscreenStopped
		return s, FullscreenNothing
	}
	var effect FullscreenEffect
	switch {
	case s.phase == fullscreenInitial && event == FullscreenBegin:
		switch s.plan.Operation {
		case FullscreenExpand:
			s.phase, effect = fullscreenSaving, FullscreenSave
		case FullscreenCollapse:
			s.phase, effect = fullscreenToggling, FullscreenToggle
		default:
			s.phase = fullscreenDone
		}
	case s.phase == fullscreenSaving && event == FullscreenConfirmed:
		s.phase, effect = fullscreenToggling, FullscreenToggle
	case s.phase == fullscreenToggling && event == FullscreenConfirmed:
		if s.plan.Operation == FullscreenExpand {
			s.phase = fullscreenDone
		} else if s.plan.ReturnID != "" && s.plan.ReturnID != s.plan.TerminalID {
			s.phase, effect = fullscreenFocusing, FullscreenFocus
		} else {
			s.phase, effect = fullscreenClearing, FullscreenClear
		}
	case s.phase == fullscreenFocusing && event == FullscreenConfirmed:
		s.phase, effect = fullscreenClearing, FullscreenClear
	case s.phase == fullscreenClearing && event == FullscreenConfirmed:
		s.phase = fullscreenDone
	}
	return s, effect
}

type FullscreenRuntime interface {
	Runtime
	CurrentPaneID() string
	FullscreenStore() workbenchshortcut.FullscreenStore
}

// RunToggleFocused uses native fullscreen: UI bars cost rows, while this action
// is for gaining columns, so toggle-no-ui-fullscreen is deliberately not used.
func RunToggleFocused(args []string, rt FullscreenRuntime, stderr io.Writer) int {
	if len(args) != 0 {
		fmt.Fprintln(stderr, "usage: pair layout toggle-focused")
		return 2
	}
	store := rt.FullscreenStore()
	fail := func(stage string, err error) int {
		diagnostic := fmt.Errorf("pair layout toggle-focused: %s: %w", stage, err)
		store.LogFailure(diagnostic)
		// Shortcut callers discard this CLI channel; diagnostics are retained above.
		fmt.Fprintln(stderr, diagnostic)
		return 1
	}
	unlock, acquired, err := store.TryLock()
	if err != nil {
		return fail("lock", err)
	}
	if !acquired {
		return 0
	}
	defer unlock()
	raw, err := rt.ListPanesJSON()
	if err != nil {
		return fail("list panes", err)
	}
	if !json.Valid(raw) {
		return fail("list panes", fmt.Errorf("invalid JSON"))
	}
	panes := zellijpane.Parse(raw)
	if len(panes) == 0 {
		return fail("list panes", fmt.Errorf("no pane observations"))
	}
	last, _ := rt.LastTerminalPaneID()
	ids, _ := rt.TerminalPaneIDs()
	recorded, err := store.Read()
	if err != nil {
		return fail("read return pane", err)
	}
	plan, err := PlanFullscreen(panes, rt.CurrentPaneID(), last, ids, recorded)
	if err != nil {
		return fail("plan", err)
	}
	state, effect := FullscreenTransition(NewFullscreenState(plan), FullscreenBegin)
	for effect != FullscreenNothing {
		var stage string
		switch effect {
		case FullscreenSave:
			stage = "save return"
			err = store.Write(plan.ReturnID)
		case FullscreenToggle:
			stage = "toggle-fullscreen"
			err = rt.RunZellijAction("toggle-fullscreen", "--pane-id", plan.TerminalID)
		case FullscreenFocus:
			stage = "focus-pane-id"
			err = rt.RunZellijAction("focus-pane-id", plan.ReturnID)
		case FullscreenClear:
			stage = "clear return"
			err = store.Clear()
		}
		outcome := FullscreenConfirmed
		if err != nil {
			outcome = FullscreenFailed
			if effect == FullscreenToggle || effect == FullscreenFocus {
				outcome = FullscreenUnconfirmed
			}
		}
		state, effect = FullscreenTransition(state, outcome)
		if err != nil {
			return fail(fmt.Sprintf("%s target=%s return=%s", stage, plan.TerminalID, plan.ReturnID), err)
		}
	}
	return 0
}

func (OSRuntime) CurrentPaneID() string { return os.Getenv("ZELLIJ_PANE_ID") }
func (OSRuntime) FullscreenStore() workbenchshortcut.FullscreenStore {
	return workbenchshortcut.FullscreenReturnStore{DataDir: workbenchshortcut.DataDirFromEnv(), Tag: os.Getenv("PAIR_TAG")}
}
