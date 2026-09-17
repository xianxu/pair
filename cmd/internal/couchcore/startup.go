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
		// A row with no recorded activity carries the ZERO time, which is Before
		// everything, so After() is false and it can never displace a better
		// row -- it sorts last within its rank class, which is what we want. No
		// guard needed here, unlike the renderers (pair#187). Ties between two
		// such rows are deterministic: ProjectActionableThreads sorts by
		// (RepoScope, Tag).
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

// The occupancy questions, and why they are not one function.
//
// Three predicates read a thread's state and they are deliberately distinct,
// because they ask different things: `occupiedIncarnation` asks whether
// something is still ACTING on a thread (shared by archive and resume, and the
// one that was genuinely duplicated); `PathHoldsUsableThread` asks whether a
// path already holds work; `PathHoldsUnreadableThread` asks whether a scope
// holds something couch could not read. Collapsing them would force one answer
// onto three questions.
//
// What must not drift is their OVERLAP: anything `PathHoldsUsableThread`
// counts as holding a path must also be something the operator can reach, and
// TestOccupancyPredicatesAgreeWhereTheyOverlap pins that.

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

// PathHoldsUnreadableThread reports a record couch could not read at all.
//
// It is separate from PathHoldsUsableThread because it answers a different
// question. An unreadable record has no working path -- reading it is what would
// have supplied one -- so it cannot be matched by path, and treating "cannot
// read" as "not here" would let a second thread be created in a tree that may
// already hold live work. That is the ratchet M3 closed, reappearing silently
// where the old code at least failed loudly.
//
// Scope-wide rather than path-exact, deliberately: unknown is unknown, and the
// conservative reading of an unreadable record in this repo is that it might be
// the one at this path.
func PathHoldsUnreadableThread(rows []ActionableThreadSummary, repoScope string) (ThreadAddress, bool) {
	for _, row := range rows {
		if row.Address.RepoScope == repoScope && row.Reason == ReasonUnreadable {
			return row.Address, true
		}
	}
	return ThreadAddress{}, false
}

// startupAsks decides which resume-shaped candidates startup's blocking
// inventory resolves. It is the union of what the readers of startup's rows
// filter on, and it is the whole of pair#206 M1.
//
// THE READERS OF STARTUP'S ROWS -- this list is their one home; other comments
// and docs point here rather than restating it, because every restated copy
// has drifted (pair#206 M1 review):
//
//   - SelectResumableRoot, PathHoldsUsableThread (inside spawnResolved): read
//     only rows at the cwd (this repo scope AND this working path);
//   - PathHoldsUnreadableThread: scans the whole scope, but only for records
//     the store could not decode, which never reach the resume-shaped branch
//     this predicate gates -- so it is unaffected by the narrowing;
//   - ResolveLayoutConflicts: reads only rows whose layout differs from the one
//     couch was asked to start in, at any path.
//
// A candidate outside both sets keeps ProofUnresolved and classifies
// `unknown`. No reader of startup's rows can act on such a row: it is not at
// the cwd, and its layout agrees, so it is neither selectable nor a conflict.
// The rows never leave StartInteractive -- StartResult carries none -- so the
// unasked state cannot reach the switcher, which is what pair#228's close
// review closed off.
//
// TestNarrowedStartupAnswersAsAFullProofWould is the guard: it computes both
// inventories and asserts every reader in the list above answers identically. A
// new reader joins that list and that test, and widens this predicate if it
// filters differently.
func startupAsks(requested Layout, repoScope, workingPath string) func(ThreadRecord) bool {
	return func(record ThreadRecord) bool {
		if record.Address.RepoScope == repoScope && record.WorkingPath == workingPath {
			return true
		}
		return NormalizeLayout(string(record.Layout)) != requested
	}
}

// startupInventory is the blocking inventory StartInteractive reads, narrowed
// to what its readers consume.
//
// It takes the scope key its caller already resolved rather than resolving one
// of its own: the narrowing predicate has to agree EXACTLY with the selectors
// that read its rows, and two independent resolutions of the same path are two
// chances to disagree.
func (c *Couch) startupInventory(ctx context.Context, scopeKey, workingPath string) ([]ActionableThreadSummary, error) {
	snapshot, evidence, err := c.gatherThreadEvidence(ctx, nil, startupAsks(c.Layout, scopeKey, workingPath))
	if err != nil {
		return nil, err
	}
	return ProjectActionableThreads(FromSnapshot(snapshot, evidence)), nil
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
	rows, err := c.startupInventory(ctx, scope.Key, resolution.CanonicalPath)
	if err != nil {
		return StartResult{}, err
	}
	// The mixed-layout guard, deliberately placed HERE rather than in couchcmd:
	// these rows are the only session enumeration startup performs, and the
	// guard must add none of its own. It runs before any effect, so a refusal
	// starts no child.
	//
	// startupAsks proved exactly two sets: the cwd's candidates, and every
	// candidate whose layout differs -- which is this guard's own set. A row
	// left `unknown` is one no reader here can act on (pair#206 M1).
	//
	// It is a predictability feature, not a safety one -- #179 keeps a warm
	// reattach safe on its own by sending no layout flag at all -- so a startup
	// snapshot is the right strength and nothing here needs to be transactional
	// with the store.
	if conflicts := ResolveLayoutConflicts(c.Layout, rows); len(conflicts) > 0 {
		return StartResult{}, layoutConflictRefusal(c.Layout, conflicts)
	}
	if address, ok := SelectResumableRoot(rows, scope.Key, resolution.CanonicalPath); ok {
		opts := ResumeOptions{}
		for _, row := range rows {
			if row.Address == address {
				opts.WarmOnly = row.Detached()
				break
			}
		}
		record, handle, resumeErr := c.ResumeContextWith(ctx, address, opts)
		return StartResult{Record: record, Handle: handle}, startupResumeRefusal(address, resumeErr)
	}
	record, handle, err := c.spawnResolved(ctx, resolution, rows)
	return StartResult{Record: record, Handle: handle}, err
}

// startupResumeRefusal makes a startup failure actionable, and says something
// TRUE about the failure it is decorating.
//
// Startup deliberately has NO fallback: `couch` in a tree that already holds a
// resumable thread must not quietly start a second one, because two threads in
// one tree is the confusion couch exists to prevent. What was wrong was failing
// MUTELY -- the operator saw a code, or worse an internal store message, and had
// no next step.
//
// Two shapes, two messages. A structured refusal means couch DECIDED not to
// start, and names the thread it found. An internal failure means couch could
// not tell, and must not claim to have found a resumable thread -- that framing
// would be false for a store it could not read. Both end with the way forward,
// because that is what the operator needs and it does not depend on the shape.
//
// An earlier attempt made every producer carry a ResumeDiagnosticCode so this
// function could treat them alike. That changed what the code MEANS -- from "is
// a structured refusal" to "came out of resume" -- and broke every reader that
// used the distinction, so the branching lives here instead.
func startupResumeRefusal(address ThreadAddress, err error) error {
	if err == nil {
		return nil
	}
	const wayForward = "  inspect it:  couch --show %s\n" +
		"  work anyway: pair          (in this tree, without couch)"
	if ResumeDiagnosticOf(err) == "" {
		return fmt.Errorf(
			"%w\n\ncouch could not resume the thread in this tree (%s/%s) and will not start a second one.\n"+
				wayForward,
			err, address.RepoScope, address.Tag, address.Tag)
	}
	return fmt.Errorf(
		"%w\n\ncouch found one resumable thread here (%s/%s) and will not start a second in the same tree.\n"+
			wayForward,
		err, address.RepoScope, address.Tag, address.Tag)
}
