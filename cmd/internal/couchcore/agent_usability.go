package couchcore

import "github.com/xianxu/pair/cmd/internal/launcher"

// agentUsabilityReason projects ephemeral terminal evidence into existing thread
// states. Missing evidence refuses action; observed unsafe terminals need an
// explicit relaunch, never an automatic retirement of their conversation.
func agentUsabilityReason(session SessionObservation) ThreadReason {
	if session.Terminal != nil && session.Terminal.State == launcher.TerminalUnsafe {
		return ReasonNeedsRelaunch
	}
	if session.State == SessionAbsent {
		return ReasonNeedsRelaunch
	}
	if session.State != SessionPresent || session.Terminal == nil {
		return ReasonUnknown
	}
	switch session.Terminal.State {
	case launcher.TerminalUsable:
		return ""
	default:
		return ReasonUnknown
	}
}
