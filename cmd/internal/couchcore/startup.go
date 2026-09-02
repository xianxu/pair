package couchcore

import (
	"context"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

// SelectUniqueParkedRoot selects only an exact, unambiguous actionable parked
// thread for an already-normalized repository scope and physical working path.
func SelectUniqueParkedRoot(rows []ActionableThreadSummary, repoScope, workingPath string) (ThreadAddress, bool) {
	var selected ThreadAddress
	found := false
	for _, row := range rows {
		if row.State != ThreadParked || row.Address.RepoScope != repoScope || row.WorkingPath != workingPath {
			continue
		}
		if found {
			return ThreadAddress{}, false
		}
		selected = row.Address
		found = true
	}
	return selected, found
}

// StartInteractive chooses the root/home actor for one interactive Couch
// startup before performing either resume or new-thread effects.
func (c *Couch) StartInteractive(ctx context.Context, args StartArgs) (StartResult, error) {
	resolution, err := c.resolveStartResolution(ctx, args)
	if err != nil {
		return StartResult{}, err
	}
	scope, err := launcher.ResolveRepoScope(string(resolution.Worktree))
	if err != nil {
		return StartResult{}, err
	}
	rows, err := c.ActionableThreadInventoryContext(ctx, nil)
	if err != nil {
		return StartResult{}, err
	}
	if address, ok := SelectUniqueParkedRoot(rows, scope.Key, resolution.CanonicalPath); ok {
		record, handle, resumeErr := c.ResumeContext(ctx, address)
		return StartResult{Record: record, Handle: handle}, resumeErr
	}
	record, handle, err := c.spawnResolved(ctx, resolution)
	return StartResult{Record: record, Handle: handle}, err
}
