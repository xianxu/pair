package couchcore

import (
	"context"
	"fmt"
	"path/filepath"
)

// launchFailuresForShow reads independent evidence without manufacturing a
// lifecycle record. An exact address or canonical starting path is authority;
// fuzzy thread labels never broaden the retained history lookup.
func (c *Couch) launchFailuresForShow(ctx context.Context, scope, ref string, matches []ThreadRecord, slot SlotIdentity) ([]LaunchFailure, string) {
	if c.LaunchFailures == nil {
		return nil, ""
	}
	history, err := c.LaunchFailures.Read()
	if err != nil {
		return nil, fmt.Sprintf("read launch diagnostics %s: %v", c.LaunchFailures.Path(), err)
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
	for _, m := range matches {
		wanted[m.Address] = true
	}
	canonical := map[string]bool{}
	for _, p := range paths {
		if physical, e := retainedPhysicalPath(p); e == nil {
			canonical[physical] = true
		}
	}
	var out []LaunchFailure
	for _, f := range history {
		matched := wanted[f.Address]
		if !matched && f.StartingPath != "" {
			if physical, e := retainedPhysicalPath(f.StartingPath); e == nil {
				matched = canonical[physical]
			}
		}
		if matched {
			out = append(out, cloneLaunchFailure(f))
		}
	}
	return out, ""
}
