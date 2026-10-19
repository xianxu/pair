package couchcore

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"time"
)

// launchFailuresForShow reads independent evidence without manufacturing a
// lifecycle record. Exact dispatched references, addresses and canonical paths
// select evidence; fuzzy labels never broaden this display-only lookup.
func (c *Couch) launchFailuresForShow(ctx context.Context, scope, ref string, matches []ThreadRecord, slot SlotIdentity) ([]LaunchFailure, string) {
	if c.LaunchFailures == nil {
		return nil, ""
	}
	history, err := c.lifecycleFailureRecorder().Snapshot()
	if err != nil {
		return nil, fmt.Sprintf("read lifecycle diagnostics %s: %v", c.LaunchFailures.Path(), err)
	}
	paths := []string{}
	if slot.WorktreeRoot != "" {
		paths = append(paths, slot.WorktreeRoot)
	}
	if filepath.IsAbs(ref) {
		paths = append(paths, ref)
	} else if p, recognized, e := c.WorkspaceReferencePath(ctx, ref); recognized && e == nil {
		paths = append(paths, p)
	}
	wanted := map[ThreadAddress]bool{{RepoScope: scope, Tag: ThreadTag(ref)}: true}
	successes := map[string]time.Time{}
	displayKeys := map[string]string{}
	// The same resolved-slot target selection serves rows and show, including
	// an empty slot with no current thread record. It is display evidence only.
	slotRow := ActionableThreadSummary{StartingPath: slot.WorktreeRoot, Target: ThreadTarget{Kind: ThreadTargetSlot, Slot: slot}}
	for _, target := range lifecycleFailureRowTargets(slotRow) {
		if key := lifecycleFailureTargetKey(target); key != "" {
			displayKeys[key] = lifecycleFailureTargetKey(LifecycleFailureTarget{StartingPath: slot.WorktreeRoot})
		}
	}
	for _, m := range matches {
		wanted[m.Address] = true
		success := LastLifecycleSuccessAt(m)
		displayKey := lifecycleFailureTargetKey(LifecycleFailureTarget{StartingPath: m.StartingPath, Address: m.Address})
		for _, target := range lifecycleFailureRowTargets(ActionableThreadSummary{Address: m.Address, StartingPath: m.StartingPath, Target: ThreadTarget{Kind: ThreadTargetSlot, Slot: slot}}) {
			key := lifecycleFailureTargetKey(target)
			if key != "" {
				displayKeys[key] = displayKey
			}
			if key != "" && success.After(successes[key]) {
				successes[key] = success
			}
		}
	}
	canonical := map[string]bool{}
	for _, p := range paths {
		if physical, e := retainedPhysicalPath(p); e == nil {
			canonical[physical] = true
		}
	}
	selected := map[string]LaunchFailure{}
	var out []LaunchFailure
	for _, f := range latestLifecycleFailures(history) {
		_, selectedByResolvedTarget := displayKeys[lifecycleFailureProjectionKey(f)]
		matched := selectedByResolvedTarget || wanted[f.Address] || (f.Target != "" && f.Target == ref)
		if !matched && f.StartingPath != "" {
			if physical, e := retainedPhysicalPath(f.StartingPath); e == nil {
				matched = canonical[physical]
			}
		}
		success := successes[lifecycleFailureProjectionKey(f)]
		if matched && lifecycleFailureVisible(f, success) {
			key := lifecycleFailureProjectionKey(f)
			if displayKey := displayKeys[key]; displayKey != "" {
				key = displayKey
			}
			if old, exists := selected[key]; !exists || compareLifecycleFailureRecency(f, old) > 0 {
				selected[key] = f
			}
		}
	}
	for _, f := range selected {
		out = append(out, cloneLaunchFailure(f))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].FailedAt.Equal(out[j].FailedAt) {
			return out[i].Nonce > out[j].Nonce
		}
		return out[i].FailedAt.After(out[j].FailedAt)
	})
	return out, ""
}
