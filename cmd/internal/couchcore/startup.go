package couchcore

import (
	"context"
	"fmt"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

// SelectUniqueResumableRoot selects only an exact, unambiguous actionable
// resumable row for one repository scope and physical path.
//
// Both resumable states qualify. Parked is cold -- the zellij session was torn
// down -- and detached is warm, its session still running with no client; but
// `couch` in a directory means the same thing either way, reattach what is
// already there, and both converge on one `pair resume`. Naming it Parked while
// it selects detached rows would be a lie the next reader pays for.
//
// A live row is deliberately never selected: couch is a singleton holding its
// supervisor lease for the whole run, so a live row at startup is one THIS couch
// already hosts.
//
// Exactness, not ranking: a parked row and a detached row at one path are TWO
// matches and create a new thread, exactly as two parked rows do. Preferring
// warm over cold would be a policy, and this selector deliberately has none.
func SelectUniqueResumableRoot(rows []ActionableThreadSummary, repoScope, workingPath string) (ThreadAddress, bool) {
	var selected ThreadAddress
	found := false
	for _, row := range rows {
		if !row.Resumable() || row.Address.RepoScope != repoScope || row.WorkingPath != workingPath {
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
	if address, ok := SelectUniqueResumableRoot(rows, scope.Key, resolution.CanonicalPath); ok {
		record, handle, resumeErr := c.ResumeContext(ctx, address)
		return StartResult{Record: record, Handle: handle}, startupResumeRefusal(address, resumeErr)
	}
	record, handle, err := c.spawnResolved(ctx, resolution)
	return StartResult{Record: record, Handle: handle}, err
}

// startupResumeRefusal makes a startup refusal actionable.
//
// Startup deliberately has NO fallback: `couch` in a tree that already holds a
// resumable thread must not quietly start a second one, because two threads in
// one tree is the confusion couch exists to prevent. What was wrong was
// refusing MUTELY -- the operator saw a diagnostic code and had no next step.
// So the refusal stands, and it says which thread, what happened, and the two
// ways forward.
func startupResumeRefusal(address ThreadAddress, err error) error {
	if err == nil {
		return nil
	}
	code := ResumeDiagnosticOf(err)
	if code == "" {
		return err
	}
	return fmt.Errorf(
		"%w\n\ncouch found one resumable thread here (%s/%s) and will not start a second in the same tree.\n"+
			"  inspect it:  couch --show %s\n"+
			"  work anyway: pair          (in this tree, without couch)",
		err, address.RepoScope, address.Tag, address.Tag)
}
