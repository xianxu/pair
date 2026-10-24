package couchcore

import (
	"context"
	"fmt"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

// The occupancy questions, and why they are not one function.
//
// Three predicates read a thread's state and they are deliberately distinct,
// because they ask different things: `hasOccupiedIncarnation` asks whether couch
// itself is mid-operation on a thread, which is what relaunch and switch-agent
// need in order to know there is a source to park (archive stopped asking it in
// #256 M3 -- it is bookkeeping, and archive's question is about the world);
// `ScopeHoldsUsableThread` asks whether a repository already holds its primary;
// `PathHoldsUnreadableThread` asks whether a scope holds something couch could
// not read. Collapsing them would force one answer onto three questions.
//
// What must not drift is their OVERLAP: anything `ScopeHoldsUsableThread`
// counts as holding a repository must also be something the operator can
// reach, and TestOccupancyPredicatesAgreeWhereTheyOverlap pins that.

// primaryOfScope is whether a row is a primary (`:0`, an ordinary target) of
// the repository scope: the one row per repository the startup selector and
// the occupancy guard read.
func primaryOfScope(row ActionableThreadSummary, repoScope string) bool {
	return row.Address.RepoScope == repoScope && row.Target.Kind != ThreadTargetSlot
}

// ScopeHoldsUsableThread reports the primary thread a repository already has,
// if the operator can get back into it.
//
// One primary per repository is ENFORCED (pair#363, widened from one thread
// per path, #181): a start in a subdirectory of a repository whose `:0`
// already exists must return to that thread or refuse, never create a second
// primary. Debris deliberately does not count -- a repository whose only rows
// are unusable must still be startable, or a corrupted record would lock it out
// permanently (resolved ambiguity 9).
func ScopeHoldsUsableThread(rows []ActionableThreadSummary, repoScope string) (ActionableThreadSummary, bool) {
	for _, row := range rows {
		if !primaryOfScope(row, repoScope) {
			continue
		}
		switch row.State {
		case ThreadLive, ThreadDetached, ThreadParked:
			return row, true
		}
	}
	return ActionableThreadSummary{}, false
}

// PathHoldsUnreadableThread reports a record couch could not read at all.
//
// It is separate from ScopeHoldsUsableThread because it answers a different
// question. An unreadable record has no working path or target -- reading it is
// what would have supplied them -- and treating "cannot read" as "not here"
// would let a second thread be created in a repository that may already hold
// live work. That is the ratchet M3 closed, reappearing silently where the old
// code at least failed loudly.
//
// Scope-wide, deliberately: unknown is unknown, and the conservative reading of
// an unreadable record in this repo is that it might be its primary.
func PathHoldsUnreadableThread(rows []ActionableThreadSummary, repoScope string) (ThreadAddress, bool) {
	for _, row := range rows {
		if row.Address.RepoScope == repoScope && row.Reason == ReasonUnreadable {
			return row.Address, true
		}
	}
	return ThreadAddress{}, false
}

// StartInteractive only reattaches an existing selected hub when no action was
// requested. Cold resume and Fresh remain explicit operator actions.
func (c *Couch) StartInteractive(ctx context.Context, args StartArgs) (StartResult, error) {
	if args.Action != "" {
		return c.startExplicitInteractive(ctx, args)
	}
	if err := ctx.Err(); err != nil {
		return StartResult{}, err
	}
	path := args.WorkingDir()
	if path == "" {
		path = "."
	}
	tree, err := c.ResolveTree(path)
	if err != nil {
		if ctx.Err() != nil {
			return StartResult{}, ctx.Err()
		}
		return StartResult{}, nil
	}
	hub := string(tree)
	if physical, e := c.Path.Physical(hub); e == nil {
		hub = physical
	}
	snapshot, err := c.Threads.PreviewSnapshotContext(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return StartResult{}, ctx.Err()
		}
		return StartResult{}, nil
	}
	for _, candidate := range startupHubRecords(snapshot, hub) {
		session, present, err := c.listedTerminal(ctx, candidate)
		if err != nil {
			if ctx.Err() != nil {
				return StartResult{}, ctx.Err()
			}
			return StartResult{}, nil
		}
		if present {
			record, handle, err := c.attachListedTerminal(ctx, candidate, session, ResumeOptions{})
			if err != nil {
				err = fmt.Errorf("hub %s: %w", hub, err)
			}
			return StartResult{Record: record, Handle: handle}, err
		}
	}
	return StartResult{}, nil
}

func (c *Couch) startExplicitInteractive(ctx context.Context, args StartArgs) (StartResult, error) {
	if err := ctx.Err(); err != nil {
		return StartResult{}, err
	}
	resolution, err := c.resolveStartResolution(ctx, args)
	if err != nil {
		return StartResult{}, err
	}
	return c.startInteractiveResolution(ctx, resolution)
}

// StartFreshHub is called only with the initial zero-history startup permission.
// A primary opens normally; a never-used numbered slot needs its existing fresh
// path, because OpenSlot deliberately requires a resumable current conversation.
func (c *Couch) StartFreshHub(ctx context.Context, path string) (StartResult, error) {
	if err := ctx.Err(); err != nil {
		return StartResult{}, err
	}
	resolution, err := c.resolveStartResolution(ctx, StartArgs{Cwd: path, Action: StartOpen})
	if err != nil {
		return StartResult{}, err
	}
	if resolution.Target.Kind == ThreadTargetSlot {
		return c.startFreshSlotWithRecovery(ctx, resolution.Target.Slot.WorktreeRoot, resolution.RequestedAgent, true, &resolution, nil, nil, freshSlotEmptyStartup)
	}
	record, handle, err := c.spawnResolvedWithAdmission(ctx, resolution, nil, true)
	return StartResult{Record: record, Handle: handle}, err
}

func (c *Couch) startInteractiveResolution(ctx context.Context, resolution StartResolution) (StartResult, error) {
	if resolution.Target.Kind == ThreadTargetSlot {
		return c.spawnManagedResolution(ctx, resolution)
	}
	if err := c.enrollPrimaryResolution(ctx, resolution); err != nil {
		return StartResult{}, err
	}
	scope, err := launcher.ResolveRepoScope(string(resolution.Worktree))
	if err != nil {
		return StartResult{}, err
	}
	snapshot, err := c.Threads.PreviewSnapshotContext(ctx)
	if err != nil {
		return StartResult{}, err
	}
	primaries := make(map[ThreadAddress]bool)
	for _, row := range ProjectActionableThreads(FromSnapshot(snapshot, nil)) {
		if primaryOfScope(row, scope.Key) {
			primaries[row.Address] = true
		}
	}
	var warm *ThreadRecord
	var session string
	for _, record := range snapshot.Records {
		if !primaries[record.Address] {
			continue
		}
		name, listed, err := c.listedTerminal(ctx, record)
		if err != nil {
			return StartResult{}, err
		}
		if listed && (warm == nil || record.LastActiveAt.After(warm.LastActiveAt)) {
			candidate := record
			warm, session = &candidate, name
		}
	}
	if warm != nil {
		record, handle, err := c.attachListedTerminal(ctx, *warm, session, ResumeOptions{})
		return StartResult{Record: record, Handle: handle}, err
	}
	snapshot, evidence, err := c.gatherThreadEvidence(ctx, nil, func(ThreadRecord) bool { return false })
	if err != nil {
		return StartResult{}, err
	}
	rows := ProjectActionableThreads(FromSnapshot(snapshot, evidence))
	var cold *ThreadRecord
	for _, record := range snapshot.Records {
		if !primaries[record.Address] {
			continue
		}
		binding, err := c.resumeEvidence(ctx, record)
		if err != nil && !isBindingDiagnostic(ResumeDiagnosticOf(err)) {
			return StartResult{}, err
		}
		if err == nil && coldResumeAuthorized(binding) && CheckResumePreconditions(record, binding, c.workingPathExists(record)) == nil && (cold == nil || record.LastActiveAt.After(cold.LastActiveAt)) {
			candidate := record
			cold = &candidate
		}
	}
	if cold != nil {
		record, handle, err := c.ResumeContextWith(ctx, cold.Address, ResumeOptions{})
		return StartResult{Record: record, Handle: handle}, err
	}
	record, handle, err := c.spawnResolved(ctx, resolution, rows)
	return StartResult{Record: record, Handle: handle}, err
}
