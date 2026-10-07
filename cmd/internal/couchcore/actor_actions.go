package couchcore

import "github.com/xianxu/pair/cmd/internal/checkpoint"

// ActorRowFacts is everything the actor-operation admission table reads about
// a row (pair#367, extracted from the switcher's menuRowActions). The
// switcher, the recover-plan report's steps and (M2) the socket's admission
// all read ActorActions, so a step the report suggests is one the switcher
// would offer on the same row (ARCH-DRY).
type ActorRowFacts struct {
	// Slot is a :1+ slot row; otherwise the row is a repository's :0.
	Slot   bool
	State  ActionableThreadState
	Reason ThreadReason
	// ResumeOffered is read on unusable rows only: a slot (OpenSlot may adopt
	// a still-running agent) unless its parked conversation is binding-lost,
	// or a primary with Recovery.Recover; an unfinished request either way
	// (resume routes those to their own executors).
	ResumeOffered bool
	// DirectoryMissing is an unusable row whose reason is path-missing.
	DirectoryMissing bool
	// Orphaned is a row carrying an orphaned server (row.Orphan), live rows
	// included (#399).
	Orphaned bool
}

// ActorRowFactsOf reads the admission facts off one actionable row.
func ActorRowFactsOf(row ActionableThreadSummary) ActorRowFacts {
	f := ActorRowFacts{
		Slot:             row.Target.Kind == ThreadTargetSlot,
		State:            row.State,
		Reason:           row.Reason,
		DirectoryMissing: row.State == ThreadUnusable && row.Reason == ReasonPathMissing,
		Orphaned:         row.Orphan != nil,
	}
	if row.State == ThreadUnusable {
		unfinished := row.Continuation != nil && row.Continuation.Phase != checkpoint.Complete
		if f.Slot {
			// OpenSlot may adopt a still-running agent, except where a park
			// quiesced the session and its conversation cannot be resolved
			// (binding-lost: e.g. the agent never took a turn). An unfinished
			// continuation still has its own executor.
			f.ResumeOffered = unfinished || !IsBindingFailure(row.Reason)
		} else {
			f.ResumeOffered = unfinished || (row.Recovery != nil && row.Recovery.Recover)
		}
	}
	return f
}

// ActorActions is the per-row admission table for the actor operations --
// resume, reboot, and reap for an orphaned server (#399). The switcher's
// recover is offered wherever this offers anything. Live rows get lifecycle
// actions from the switcher instead, except a live orphan, which gets reap
// here (its lifecycle actions would all refuse); busy, archived and
// unusable/unknown rows get nothing ("checking…" is not a verdict, and reboot
// stops a session).
func ActorActions(f ActorRowFacts) []string {
	if f.Orphaned {
		// Its agent may still be writing: resume would add a second one and
		// reboot would archive a running conversation. Only a confirmed reap
		// of the orphaned server tree is safe (#399), live or not.
		return []string{"reap"}
	}
	switch f.State {
	case ThreadParked, ThreadDetached:
		return []string{"resume", "reboot"}
	case ThreadUnusable:
		if f.Reason == ReasonUnknown {
			return nil
		}
		if f.Reason == ReasonOrphanedServer {
			// Its agent may still be writing: resume would add a second one and
			// reboot would archive a running conversation. Only a confirmed reap
			// of the orphaned server tree is safe (#399).
			return []string{"reap"}
		}
		if f.DirectoryMissing {
			// A :0 record outlives its checkout, so reboot archives it alone.
			// A :1+ record lives inside its directory: there is nothing left
			// to retire. (couch --reboot / --reconcile still reconcile a
			// deleted slot, pair#387.)
			if f.Slot {
				return nil
			}
			return []string{"reboot"}
		}
		if f.ResumeOffered {
			return []string{"resume", "reboot"}
		}
		return []string{"reboot"}
	}
	return nil
}
