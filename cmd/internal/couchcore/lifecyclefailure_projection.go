package couchcore

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
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
		if f.SupersededAt != nil {
			continue
		}
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
	out := append([]ActionableThreadSummary(nil), rows...)
	for i := range out {
		row := &out[i]
		row.Failure = nil
		row.DiagnosticError = diagnosticError
		key := LifecycleFailureSlotKey(LifecycleFailureTarget{StartingPath: row.StartingPath})
		f, ok := latest[key]
		if !ok {
			f, ok = latest[lifecycleFailureTargetKey(LifecycleFailureTarget{Address: row.Address})]
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
