package diagnosticlog

import (
	"errors"
	"sort"
	"time"
)

// KeepDiagnosticsUntilBackup fences all diagnostic payload eviction until the
// history-backup owner provides archive-before-evict. Rotation still preserves
// generations without a count limit; persisted count limits remain dormant.
const KeepDiagnosticsUntilBackup = true
const BackupFenceReason = "diagnostic eviction awaits history backup"

var ErrBackupRequired = errors.New(BackupFenceReason)

const MaxClosedGenerations = 3
const maxPolicyCandidates = 64

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

// SelectExcessGenerations requires a complete, bounded, certified inventory.
// It never treats a current path, malformed clock, or uncertainty as deletion
// authority. Quarantined immutable generations count but remain unverified.
func SelectExcessGenerations(now time.Time, candidates []generation, limits DiagnosticLimits, complete bool) ([]generation, error) {
	if limits.MaxBytes <= 0 || limits.MaxBytes > MaxGenerationBytes || limits.MaxGenerations < 0 || limits.MaxGenerations > MaxClosedGenerations || !complete || len(candidates) > maxPolicyCandidates {
		return nil, errors.New("diagnostic policy inventory incomplete")
	}
	seen := map[string]bool{}
	for _, g := range candidates {
		if !validSegment(g.Name) || seen[g.Name] || g.Identity == (identity{}) || g.Size < 0 || g.Start.IsZero() || g.LastWrite.IsZero() || g.ModTime.IsZero() || g.Start.After(g.LastWrite) || g.LastWrite.After(now) || !validUnverifiedTail(g) {
			return nil, errors.New("invalid diagnostic policy generation")
		}
		seen[g.Name] = true
	}
	ordered := append([]generation(nil), candidates...)
	sort.Slice(ordered, func(i, j int) bool {
		if !ordered[i].LastWrite.Equal(ordered[j].LastWrite) {
			return ordered[i].LastWrite.Before(ordered[j].LastWrite)
		}
		return ordered[i].Name < ordered[j].Name
	})
	excess := max(0, len(ordered)-limits.MaxGenerations)
	return ordered[:excess], nil
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
func policyGenerations(path string, o Options) ([]generation, error) {
	names, _, complete, err := generationPage(path, "", maxPolicyCandidates+1, false, o)
	if err != nil {
		return nil, err
	}
	if !complete || len(names) > maxPolicyCandidates {
		return nil, ErrMore
	}
	out := make([]generation, 0, len(names))
	for _, name := range names {
		if err := o.checkContext(); err != nil {
			return nil, err
		}
		p := segmentPath(path, name)
		var g generation
		if err := readJSON(p+".json", &g); err != nil {
			return nil, err
		}
		if g.Name != name {
			return nil, errors.New("diagnostic policy name changed")
		}
		st, err := regular(p)
		if err != nil {
			return nil, err
		}
		if err := matches(st, g); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, nil
}

// Caller has proven the complete writer inventory under the matching log lock.
// Existing durable deletion intents own every destructive effect and its replay.
func trimDiagnosticGenerations(path string, s *diskState, o Options, keep int) ([]Segment, error) {
	if KeepDiagnosticsUntilBackup {
		return nil, nil
	}
	candidates, err := policyGenerations(path, o)
	if err != nil {
		return nil, err
	}
	limits := s.Limits
	if s.Version == 1 {
		limits, err = ResolveDiagnosticLimits(DiagnosticLimits{}, DiagnosticLimits{o.MaxBytes, o.MaxGenerations})
		if err != nil {
			return nil, err
		}
	}
	limits.MaxGenerations = keep
	excess, err := SelectExcessGenerations(o.Now(), candidates, limits, true)
	if err != nil {
		return nil, err
	}
	var removed []Segment
	for _, g := range excess {
		if err := o.checkContext(); err != nil {
			return removed, err
		}
		s.Deleting = &g
		if err := save(path, *s, true, o); err != nil {
			return removed, err
		}
		if err := fault(o, "delete-intent"); err != nil {
			return removed, err
		}
		if err := recoverDeletion(path, s, o); err != nil {
			return removed, err
		}
		removed = append(removed, Segment{Path: segmentPath(path, g.Name), Bytes: g.Size, LastWrite: g.LastWrite, Eligible: true, Reason: "diagnostic generation count limit", Unverified: g.UnverifiedTail != nil})
	}
	return removed, nil
}

// diagnosticCountDebt inspects names only on the admission path. It never
// reads generation metadata or proves writers. Full certification and pruning
// belong to maintenance. Pre-existing/tightened excess blocks further growth.
func diagnosticCountDebt(path string, limits DiagnosticLimits, o Options) (bool, error) {
	if KeepDiagnosticsUntilBackup {
		return false, nil
	}
	names, _, complete, err := generationPage(path, "", limits.MaxGenerations+1, false, o)
	if err != nil {
		return false, err
	}
	if len(names) > limits.MaxGenerations {
		return true, nil
	}
	if !complete {
		return false, ErrMore
	}
	return false, nil
}
