package couchcore

// DrainFlags is transient operator intent, independent of thread identity or
// observed activity. The service owns synchronization and the current value.
type DrainFlags map[string]bool

// AddDrainFlags returns a fresh set, excluding the issuing steward even when a
// previous request flagged it. Repeated requests are idempotent.
func AddDrainFlags(old DrainFlags, selected []string, steward string) DrainFlags {
	next := make(DrainFlags, len(old)+len(selected))
	for workspace, flagged := range old {
		if flagged && workspace != "" && workspace != steward {
			next[workspace] = true
		}
	}
	for _, workspace := range selected {
		if workspace != "" && workspace != steward {
			next[workspace] = true
		}
	}
	return next
}

// ClearDrainFlags returns a fresh set. A nil selection clears every flag.
func ClearDrainFlags(old DrainFlags, selected []string) DrainFlags {
	next := make(DrainFlags)
	if selected == nil {
		return next
	}
	for workspace, flagged := range old {
		if flagged {
			next[workspace] = true
		}
	}
	for _, workspace := range selected {
		delete(next, workspace)
	}
	return next
}
