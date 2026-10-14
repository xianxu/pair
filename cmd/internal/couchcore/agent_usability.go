package couchcore

// A successful live session listing authorizes Continue. Helper processes and
// metadata cannot substitute for a listing, or invalidate a positive listing.
func agentUsabilityReason(session SessionObservation) ThreadReason {
	switch session.State {
	case SessionPresent:
		return ""
	case SessionAbsent, SessionOrphaned:
		return ReasonNeedsRelaunch
	default:
		return ReasonUnknown
	}
}
