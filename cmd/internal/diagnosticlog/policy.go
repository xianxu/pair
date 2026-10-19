package diagnosticlog

import (
	"errors"
	"fmt"
	"strconv"
	"time"
)

// Legacy crash-report sweeps retain their separate backup fence; managed
// diagnostic generations no longer consult these compatibility constants.
const KeepDiagnosticsUntilBackup = true
const BackupFenceReason = "diagnostic eviction awaits history backup"

const MaxClosedGenerations = 3

// DiagnosticLimits is persisted under the stable per-log lock. Handle-local
// options can tighten this authority, never enlarge it.
type DiagnosticLimits struct {
	MaxBytes       int64 `json:"max_bytes"`
	MaxGenerations int   `json:"max_generations"`
}

func validDiagnosticLimits(l DiagnosticLimits) bool {
	return l.MaxBytes > 0 && l.MaxBytes <= MaxGenerationBytes && l.MaxGenerations > 0 && l.MaxGenerations <= MaxClosedGenerations
}

// ResolveDiagnosticLimits accepts zero requested fields as unspecified. Only
// an entirely zero persisted value denotes a legacy log without an authority.
func ResolveDiagnosticLimits(persisted, requested DiagnosticLimits) (DiagnosticLimits, error) {
	if requested.MaxBytes < 0 || requested.MaxGenerations < 0 {
		return DiagnosticLimits{}, errors.New("invalid requested diagnostic limits")
	}
	if persisted == (DiagnosticLimits{}) {
		persisted = DiagnosticLimits{MaxGenerationBytes, MaxClosedGenerations}
	}
	if !validDiagnosticLimits(persisted) {
		return DiagnosticLimits{}, errors.New("invalid persisted diagnostic limits")
	}
	if requested.MaxBytes > 0 {
		persisted.MaxBytes = min(persisted.MaxBytes, requested.MaxBytes)
	}
	if requested.MaxGenerations > 0 {
		persisted.MaxGenerations = min(persisted.MaxGenerations, requested.MaxGenerations)
	}
	return persisted, nil
}

func ensureDiagnosticLimits(path string, s *diskState, o Options) error {
	limits, err := ResolveDiagnosticLimits(s.Limits, DiagnosticLimits{o.MaxBytes, o.MaxGenerations})
	if err != nil {
		return err
	}
	if s.Version == 2 && s.Limits == limits {
		return nil
	}
	if s.Pending != nil || s.Deleting != nil || s.Creating != nil || s.Appending != nil {
		return errors.New("diagnostic migration recovery pending")
	}
	next := *s
	next.Version = 2
	next.Limits = limits
	if err := save(path, next, true, o); err != nil {
		return err
	}
	*s = next
	return nil
}

// ParseRetentionDays validates the one diagnostic retention setting before GC.
func ParseRetentionDays(value string) (time.Duration, error) {
	if value == "" {
		return RetentionPeriod, nil
	}
	days, err := strconv.Atoi(value)
	if err != nil || days < 1 || days > 365 {
		return 0, errors.New("PAIR_DIAGNOSTIC_DAYS must be an integer from 1 to 365")
	}
	return time.Duration(days) * 24 * time.Hour, nil
}
func RetentionReason(period time.Duration) string {
	if period == 0 {
		period = RetentionPeriod
	}
	return fmt.Sprintf("within %d-day retention", int(period/(24*time.Hour)))
}

// DecideGeneration selects whole generations by certified age or byte excess.
// Unknown/future clocks and invalid limits never grant deletion authority.
func DecideGeneration(now, lastWrite time.Time, size, maxBytes int64, period time.Duration) bool {
	if period == 0 {
		period = RetentionPeriod
	}
	if period < 24*time.Hour || period > 365*24*time.Hour || lastWrite.IsZero() || lastWrite.After(now) || size < 0 || maxBytes <= 0 || maxBytes > MaxGenerationBytes {
		return false
	}
	return size > maxBytes || !now.Before(lastWrite.Add(period))
}
