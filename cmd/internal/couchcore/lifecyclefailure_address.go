package couchcore

// completeFailureAddress copies a complete diagnostic identity atomically.
// Partial evidence cannot erase an exact address or borrow its other component.
// This composes diagnostic evidence only; it grants no lifecycle authority.
func completeFailureAddress(preferred, fallback ThreadAddress) ThreadAddress {
	if preferred.RepoScope != "" && preferred.Tag != "" {
		return preferred
	}
	if fallback.RepoScope != "" && fallback.Tag != "" {
		return fallback
	}
	return ThreadAddress{}
}
