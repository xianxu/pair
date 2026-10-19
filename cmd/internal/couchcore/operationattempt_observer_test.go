package couchcore

// activeOperationAttemptSlots observes coordinator retention only in tests.
func activeOperationAttemptSlots(r *LifecycleFailureRecorder) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.slots)
}
