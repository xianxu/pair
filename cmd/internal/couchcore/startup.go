package couchcore

import (
	"context"
	"fmt"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

// SelectResumableRoot picks the thread `couch <path>` should return to:
// detached first, then parked, most recently active within each class.
//
// This REVERSES the selector's previous refusal to have a policy. It used to
// require exactly one resumable row and create a new thread otherwise --
// "Preferring warm over cold would be a policy, and this selector deliberately
// has none." Exactness turned out to be a ratchet: two resumable rows at one
// path made a third, which guaranteed the next startup made a fourth. The
// operator's store reached six threads in one repo that way, and the reversal
// is theirs: couch in a tree should return to the work there, and wanting a
// fresh agent instead costs one chord inside Pair (Alt+Shift+N restarts the
// conversation without touching the workbench).
//
// Detached before parked because warm costs nothing: the agent is already
// running and reattaching preserves whatever it was doing, where a parked
// resume relaunches it. Recency within a class because that is the thread the
// operator was last in, and a wrong guess costs one ctrl-space.
//
// A live row is still never selected: couch is a singleton holding its
// supervisor lease for the whole run, so a live row is one THIS couch hosts.
// Unusable rows are never selected either -- they are debris, and a path whose
// only rows are debris correctly starts something new.
func SelectResumableRoot(rows []ActionableThreadSummary, repoScope, workingPath string) (ThreadAddress, bool) {
	best := ActionableThreadSummary{}
	found := false
	rank := func(row ActionableThreadSummary) int {
		switch row.State {
		case ThreadDetached:
			return 2
		case ThreadParked:
			return 1
		}
		return 0
	}
	for _, row := range rows {
		if row.Address.RepoScope != repoScope || row.WorkingPath != workingPath || rank(row) == 0 {
			continue
		}
		if !found || rank(row) > rank(best) ||
			(rank(row) == rank(best) && row.LastActiveAt.After(best.LastActiveAt)) {
			best, found = row, true
		}
	}
	if !found {
		return ThreadAddress{}, false
	}
	return best.Address, true
}

// PathHoldsUsableThread reports whether a path already has a thread the
// operator can get back into.
//
// One thread per repo path is ENFORCED for now: several threads at one path
// without separate worktrees is confusing, and per-repo policy is a design
// space of its own. Debris deliberately does not count -- a path whose only
// rows are unusable must still be startable, or a corrupted record would lock
// its repo out permanently.
func PathHoldsUsableThread(rows []ActionableThreadSummary, repoScope, workingPath string) (ThreadAddress, bool) {
	for _, row := range rows {
		if row.Address.RepoScope != repoScope || row.WorkingPath != workingPath {
			continue
		}
		switch row.State {
		case ThreadLive, ThreadDetached, ThreadParked:
			return row.Address, true
		}
	}
	return ThreadAddress{}, false
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
	if address, ok := SelectResumableRoot(rows, scope.Key, resolution.CanonicalPath); ok {
		record, handle, resumeErr := c.ResumeContext(ctx, address)
		return StartResult{Record: record, Handle: handle}, startupResumeRefusal(address, resumeErr)
	}
	record, handle, err := c.spawnResolved(ctx, resolution, rows)
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
