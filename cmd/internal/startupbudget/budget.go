// Package startupbudget carries one absolute startup deadline across helpers.
package startupbudget

import (
	"fmt"
	"time"
)

// Default is the existing startup ceiling when a caller supplies no deadline.
const Default = 2 * time.Minute

// DeadlineEnv carries the selected absolute deadline into a helper process.
const DeadlineEnv = "PAIR_STARTUP_DEADLINE"

// Select preserves an explicit caller deadline, or selects a fallback once.
// A selected deadline must be future and representable by the exec codec.
func Select(now, callerDeadline time.Time, fallback time.Duration) (time.Time, error) {
	deadline := callerDeadline
	if deadline.IsZero() {
		if fallback <= 0 {
			return time.Time{}, fmt.Errorf("startup fallback must be positive")
		}
		deadline = now.Add(fallback)
	}
	if !deadline.After(now) {
		return time.Time{}, fmt.Errorf("startup deadline exhausted")
	}
	if year := deadline.UTC().Year(); year < 0 || year > 9999 {
		return time.Time{}, fmt.Errorf("startup deadline outside timestamp range")
	}
	return deadline, nil
}

// Encode returns the canonical UTC representation carried across exec.
func Encode(deadline time.Time) string { return deadline.UTC().Format(time.RFC3339Nano) }

// Decode accepts only a bounded canonical representation. It does not inspect
// the current time: the consuming caller checks expiry through Select.
func Decode(raw string) (time.Time, error) {
	if len(raw) == 0 || len(raw) > 64 {
		return time.Time{}, fmt.Errorf("invalid startup deadline length")
	}
	deadline, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid startup deadline: %w", err)
	}
	if Encode(deadline) != raw {
		return time.Time{}, fmt.Errorf("noncanonical startup deadline")
	}
	return deadline, nil
}
