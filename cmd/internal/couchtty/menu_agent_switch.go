package couchtty

import (
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/orientation"
)

// The view is the shared durable projection, not another menu lifecycle.
func switchPrimaryOperation(view *couchcore.AgentSwitchView) string {
	if view == nil {
		return ""
	}
	switch view.Action {
	case couchcore.AgentSwitchRetry:
		return "switch-agent"
	case couchcore.AgentSwitchRecover, couchcore.AgentSwitchResumeOld:
		return "recover"
	case couchcore.AgentSwitchCopyBrief:
		return "copy-orientation"
	case couchcore.AgentSwitchContinue:
		return "switch"
	}
	return ""
}

func switchRowText(view *couchcore.AgentSwitchView) string {
	if view == nil {
		return ""
	}
	return view.Label() + " · " + view.ActionLabel()
}

func switchActionLabel(row couchcore.ActionableThreadSummary) func(string) string {
	return func(item string) string {
		if row.AgentSwitch != nil && item == switchPrimaryOperation(row.AgentSwitch) {
			return row.AgentSwitch.ActionLabel()
		}
		return menuItemLabel(item)
	}
}

func switchRecoveryNotice(state MenuState) string {
	describe := func(row couchcore.ActionableThreadSummary) string {
		v := row.AgentSwitch
		if v == nil || v.Outcome == couchcore.AgentSwitchReady {
			return ""
		}
		text := row.Label() + ": " + switchRowText(v)
		if v.Detail != "" {
			text += ". " + v.Detail
		}
		if v.Action == couchcore.AgentSwitchCopyBrief {
			text += ". Inspect any text already in the composer before using the copied brief."
		}
		return text
	}
	if row, ok := selectedMenuThread(state); ok {
		if text := describe(row); text != "" {
			return text
		}
	}
	for _, row := range menuRows(state) {
		if text := describe(row); text != "" {
			return text
		}
	}
	return ""
}

func dispatchSwitchPrimary(state MenuState, row couchcore.ActionableThreadSummary) (MenuState, []MenuEffect, bool) {
	if row.AgentSwitch == nil {
		return state, nil, false
	}
	switch row.AgentSwitch.Action {
	case couchcore.AgentSwitchRetry:
		next, effects := openSwitchAgent(state, row.Address)
		return next, effects, true
	case couchcore.AgentSwitchCopyBrief:
		request := orientation.Request{Body: row.AgentSwitch.Brief}
		return state, []MenuEffect{{CopyOrientation: &request}}, true
	}
	return state, nil, false
}
