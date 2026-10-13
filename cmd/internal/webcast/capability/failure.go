package capability

import "errors"

// Snapshot projects an authority's current phase with its last failure evidence.
// Consumers derive presentation and available actions from the embedded state.
type Snapshot struct {
	CapabilityState
	Problem string
}

// Failure distinguishes a definitely recoverable cleanup from a terminal or
// uncertain effect. Only the adapter that observed the effect may mark retryable.
type Failure struct {
	Outcome CleanupOutcome
	Cause   error
}

func (f *Failure) Error() string { return f.Cause.Error() }
func (f *Failure) Unwrap() error { return f.Cause }

// FailureOutcome defaults closed: an unclassified failure cannot promise that
// repeating a command is safe. No classification depends on error message text.
func FailureOutcome(err error) CleanupOutcome {
	var failure *Failure
	if errors.As(err, &failure) && failure != nil && failure.Outcome == Retryable {
		return Retryable
	}
	return Broken
}
