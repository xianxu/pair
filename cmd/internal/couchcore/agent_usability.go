package couchcore

// agentUsabilityReason projects ephemeral terminal evidence into existing thread
// states. Missing evidence refuses action; observed unsafe terminals need an
// explicit relaunch, never an automatic retirement of their conversation.
func agentUsabilityReason(session SessionObservation) ThreadReason {
	if session.Terminal != nil && !session.Terminal.Usable {
		return ReasonNeedsRelaunch
	}
	if session.State == SessionAbsent {
		return ReasonNeedsRelaunch
	}
	if session.State != SessionPresent || session.Terminal == nil {
		return ReasonUnknown
	}
	return ""
}
