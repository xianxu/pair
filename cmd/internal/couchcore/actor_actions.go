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
}

// ActorRowFactsOf reads the admission facts off one actionable row.
func ActorRowFactsOf(row ActionableThreadSummary) ActorRowFacts {
	f := ActorRowFacts{
		Slot:             row.Target.Kind == ThreadTargetSlot,
		State:            row.State,
		Reason:           row.Reason,
		DirectoryMissing: row.State == ThreadUnusable && row.Reason == ReasonPathMissing,
	}
	if row.State == ThreadUnusable {
		unfinished := row.Continuation != nil && row.Continuation.Phase != checkpoint.Complete
		if f.Slot {
			// OpenSlot may adopt a still-running agent, except where a park
			// quiesced the session and its conversation cannot be resolved
			// (binding-lost: e.g. the agent never took a turn). An unfinished
			// continuation still has its own executor.
			f.ResumeOffered = unfinished || row.Reason != ReasonBindingLost
		} else {
			f.ResumeOffered = unfinished || (row.Recovery != nil && row.Recovery.Recover)
		}
	}
	return f
}

// ActorActions is the per-row admission table for the two actor operations,
// resume and reboot, over rows that are not live. Live rows get lifecycle
// actions from the switcher instead; busy, archived and unusable/unknown rows
// get nothing ("checking…" is not a verdict, and reboot stops a session).
func ActorActions(f ActorRowFacts) []string {
	switch f.State {
	case ThreadParked, ThreadDetached:
		return []string{"resume", "reboot"}
	case ThreadUnusable:
		if f.Reason == ReasonUnknown || f.Reason == ReasonOrphanedServer {
			return nil
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
