package couchcore

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// lifecycleFailureSnapshotCache belongs to one recorder and never authorizes an effect.
type lifecycleFailureSnapshotCache struct {
	mu      sync.Mutex
	info    os.FileInfo
	valid   bool
	history []LaunchFailure
}

func cloneFailureSnapshot(h []LaunchFailure) []LaunchFailure {
	out := make([]LaunchFailure, len(h))
	for i, f := range h {
		out[i] = cloneLaunchFailure(f)
	}
	return out
}
func sameFailureSnapshot(a, b os.FileInfo) bool {
	return a != nil && b != nil && os.SameFile(a, b) && a.Size() == b.Size() && a.ModTime() == b.ModTime()
}

// Snapshot validates metadata before reusing a bounded payload. Mutating store
// operations always reread independently under their publication lock.
func (r *LifecycleFailureRecorder) Snapshot() ([]LaunchFailure, error) {
	if r == nil || r.store == nil {
		return nil, nil
	}
	cache := &r.snapshot
	cache.mu.Lock()
	defer cache.mu.Unlock()
	path := r.store.Path()
	if !filepath.IsAbs(path) {
		return r.store.Read()
	}
	if err := provisionSafePath(path); err != nil {
		cache.valid = false
		return nil, err
	}
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		cache.valid = false
		cache.info = nil
		cache.history = nil
		return nil, nil
	}
	if err != nil {
		cache.valid = false
		return nil, err
	}
	if !info.Mode().IsRegular() {
		cache.valid = false
		return nil, fmt.Errorf("lifecycle diagnostics is not a regular file: %s", path)
	}
	if cache.valid && sameFailureSnapshot(cache.info, info) {
		return cloneFailureSnapshot(cache.history), nil
	}
	cache.valid = false
	h, err := r.store.Read()
	if err != nil {
		return nil, err
	}
	// A concurrent replacement is returned as an observed snapshot, but is not
	// cached: the next refresh must inspect its payload independently.
	after, err := os.Lstat(path)
	if err == nil && sameFailureSnapshot(info, after) {
		cache.history = cloneFailureSnapshot(h)
		cache.info = after
		cache.valid = true
	}
	return cloneFailureSnapshot(h), nil
}
func lifecycleFailureProjectionKey(f LifecycleFailure) string {
	return lifecycleFailureTargetKey(failureTarget(f))
}
func latestLifecycleFailures(history []LaunchFailure) map[string]LifecycleFailure {
	latest := make(map[string]LifecycleFailure, len(history))
	for _, f := range history {
		key := lifecycleFailureProjectionKey(f)
		old, ok := latest[key]
		recency := compareLifecycleFailureRecency(f, old)
		if !ok || recency > 0 || (recency == 0 && f.Nonce > old.Nonce) {
			latest[key] = f
		}
	}
	return latest
}

// ProjectLifecycleFailures annotates rows in O(history+rows), without changing
// lifecycle state or action authority. The returned evidence owns its bytes.
func ProjectLifecycleFailures(rows []ActionableThreadSummary, history []LaunchFailure, diagnosticError string) []ActionableThreadSummary {
	latest := latestLifecycleFailures(history)
	out := make([]ActionableThreadSummary, len(rows))
	copy(out, rows)
	for i := range out {
		row := &out[i]
		row.Failure = nil
		row.DiagnosticError = diagnosticError
		f, ok := latestLifecycleFailureForTargets(latest, lifecycleFailureRowTargets(*row))
		if ok && !lifecycleFailureVisible(f, row.LastLifecycleSuccessAt) {
			ok = false
		}

		if ok {
			v := cloneLaunchFailure(f)
			row.Failure = &v
		}
	}
	return out
}
func (c *Couch) applyLifecycleFailures(rows []ActionableThreadSummary) []ActionableThreadSummary {
	h, err := c.lifecycleFailureRecorder().Snapshot()
	detail := ""
	if err != nil {
		detail = fmt.Sprintf("read lifecycle diagnostics: %v", err)
	}
	return ProjectLifecycleFailures(rows, h, detail)
}

// LastLifecycleSuccessAt is the one display-only definition of a recorded
// lifecycle success: a verified park, a committed live start, and a proven
// teardown (pair#695). Activity/retirement alone is deliberately not success.
func LastLifecycleSuccessAt(record ThreadRecord) time.Time {
	var latest time.Time
	if record.VerifiedPark != nil {
		latest = record.VerifiedPark.ParkedAt
	}
	if record.ProvenDownAt != nil && record.ProvenDownAt.After(latest) {
		latest = *record.ProvenDownAt
	}
	for _, incarnation := range record.Incarnations {
		if incarnation.State == IncarnationLive && incarnation.Start == nil && incarnation.StartedAt.After(latest) {
			latest = incarnation.StartedAt
		}
	}
	return latest
}

func lifecycleFailureRowTargets(row ActionableThreadSummary) []LifecycleFailureTarget {
	targets := []LifecycleFailureTarget{{StartingPath: row.StartingPath}, {Address: row.Address}}
	if row.Target.Kind == ThreadTargetSlot {
		slot := row.Target.Slot
		if slot.Repo != "" {
			targets = append(targets, LifecycleFailureTarget{Target: fmt.Sprintf("%s:%d", slot.Repo, slot.Number)})
		}
		if slot.PrimaryRoot != "" {
			targets = append(targets, LifecycleFailureTarget{Target: fmt.Sprintf("%s:%d", slot.PrimaryRoot, slot.Number)})
		}
	}
	return targets
}
func latestLifecycleFailureForTargets(latest map[string]LifecycleFailure, targets []LifecycleFailureTarget) (LifecycleFailure, bool) {
	var result LifecycleFailure
	found := false
	for _, target := range targets {
		key := lifecycleFailureTargetKey(target)
		if key == "" {
			continue
		}
		f, ok := latest[key]
		if ok && (!found || compareLifecycleFailureRecency(f, result) > 0) {
			result = f
			found = true
		}
	}
	return result, found
}

// lifecycleFailureVisible is shared by row and show projections. Only a
// positively later recorded success hides evidence; equality keeps it visible.
func lifecycleFailureVisible(f LifecycleFailure, success time.Time) bool {
	return !success.After(f.FailedAt)
}
