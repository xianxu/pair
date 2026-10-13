package couchcore

import (
	"strings"
	"time"
)

type AgentSwitchOutcome string

const (
	AgentSwitchInProgress       AgentSwitchOutcome = "in-progress"
	AgentSwitchOldRunning       AgentSwitchOutcome = "old-running"
	AgentSwitchNeedsRecovery    AgentSwitchOutcome = "needs-recovery"
	AgentSwitchHandoffAttention AgentSwitchOutcome = "handoff-attention"
	AgentSwitchReady            AgentSwitchOutcome = "ready"
)

type AgentSwitchAction string

const (
	AgentSwitchWait      AgentSwitchAction = "wait"
	AgentSwitchRetry     AgentSwitchAction = "retry-switch"
	AgentSwitchRecover   AgentSwitchAction = "recover-slot"
	AgentSwitchResumeOld AgentSwitchAction = "resume-old-thread"
	AgentSwitchCopyBrief AgentSwitchAction = "copy-brief"
	AgentSwitchContinue  AgentSwitchAction = "continue"
)

// AgentSwitchEvidence is exact process/session evidence for saved identities.
// Unknown is the zero value. ActiveRequestID is supplied only by the live
// coordinator; reading a saved nonterminal phase after restart cannot revive it.
type AgentSwitchEvidence struct {
	Source, Target, Restoration Liveness
	ActiveRequestID             string
	Now                         time.Time
	Diagnostic                  string
}
type AgentSwitchView struct {
	Outcome   AgentSwitchOutcome `json:"outcome"`
	Action    AgentSwitchAction  `json:"action"`
	Detail    string             `json:"detail,omitempty"`
	RequestID string             `json:"request_id"`
	Brief     string             `json:"brief,omitempty"`
}

func (v AgentSwitchView) Label() string {
	switch v.Outcome {
	case AgentSwitchInProgress:
		return "Switch in progress"
	case AgentSwitchOldRunning:
		return "Old agent still running"
	case AgentSwitchNeedsRecovery:
		return "Switch needs recovery"
	case AgentSwitchHandoffAttention:
		return "New agent running; handoff needs attention"
	case AgentSwitchReady:
		return "New agent ready"
	default:
		return ""
	}
}
func (v AgentSwitchView) ActionLabel() string {
	switch v.Action {
	case AgentSwitchWait:
		return "Wait"
	case AgentSwitchRetry:
		return "Retry switch"
	case AgentSwitchRecover:
		return "Recover slot"
	case AgentSwitchResumeOld:
		return "Resume old thread"
	case AgentSwitchCopyBrief:
		return "Copy brief"
	case AgentSwitchContinue:
		return "Continue"
	default:
		return ""
	}
}

// ProjectAgentSwitch is the common operator-facing decision for inventory,
// notices and receipts. It grants no launch authority: even Resume old thread
// must invoke the exact-evidence recovery operation and recheck its proof.
func ProjectAgentSwitch(saved *SwitchRecovery, e AgentSwitchEvidence) AgentSwitchView {
	if saved == nil {
		return AgentSwitchView{}
	}
	view := AgentSwitchView{RequestID: saved.RequestID, Outcome: AgentSwitchNeedsRecovery, Action: AgentSwitchRecover}
	detail := func(reason string) string {
		parts := []string{reason}
		for _, p := range []string{saved.Error, e.Diagnostic} {
			if p != "" {
				parts = append(parts, p)
			}
		}
		return strings.Join(parts, "; ")
	}
	if e.ActiveRequestID == saved.RequestID && !e.Now.IsZero() && e.Now.Before(saved.OverallDeadline) {
		var reason string
		switch saved.Phase {
		case AgentSwitchWaitingBrief:
			reason = "waiting for handoff"
		case AgentSwitchWaitingSettled:
			reason = "waiting for turn"
		case AgentSwitchSwitching:
			reason = "starting agent"
		case AgentSwitchDeliveryPending:
			reason = "waiting for handoff confirmation"
		}
		if reason != "" {
			view.Outcome, view.Action, view.Detail = AgentSwitchInProgress, AgentSwitchWait, reason
			return view
		}
	}
	// An unproved identity dominates a seemingly live one: selecting that live
	// process without resolving the other could hide a second survivor.
	for _, presence := range []Liveness{e.Source, e.Target, e.Restoration} {
		if presence != Live && presence != Dead {
			view.Action = AgentSwitchResumeOld
			view.Detail = detail("could not prove which saved source or target session remains; recovery must recheck before resuming the old thread")
			return view
		}
	}
	live := 0
	for _, presence := range []Liveness{e.Source, e.Target, e.Restoration} {
		if presence == Live {
			live++
		}
	}
	if live > 1 {
		view.Detail = detail("multiple saved sessions remain; recovery must select the safe survivor")
		return view
	}
	if e.Source == Live || e.Restoration == Live {
		view.Outcome, view.Action, view.Detail = AgentSwitchOldRunning, AgentSwitchRetry, detail("the old agent is verified running")
		return view
	}
	if e.Target == Live {
		registered := saved.Target != nil && saved.Target.Helper != nil && saved.Target.Helper.State == "live"
		if !registered {
			view.Detail = detail("the target survives but registration is unconfirmed")
			return view
		}
		if saved.Content.Kind == SwitchContentFresh || saved.Phase == AgentSwitchSucceeded {
			view.Outcome, view.Action, view.Detail = AgentSwitchReady, AgentSwitchContinue, "the new agent is verified running"
			return view
		}
		if ValidateSwitchContent(saved.Content) == nil && saved.Content.Body != "" {
			view.Outcome, view.Action = AgentSwitchHandoffAttention, AgentSwitchCopyBrief
			view.Brief = saved.Content.Body
			view.Detail = detail("inspect any text already in the composer before using the saved brief; it will not be submitted automatically")
			return view
		}
		view.Detail = detail("the target survives without usable saved handoff content")
		return view
	}
	view.Detail = detail("no saved agent is verified running; recover the slot using its saved session evidence")
	return view
}
