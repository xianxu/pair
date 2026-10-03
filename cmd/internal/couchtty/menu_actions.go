package couchtty

import (
	"github.com/xianxu/pair/cmd/internal/checkpoint"
	"github.com/xianxu/pair/cmd/internal/couchcore"
)

// menuRowKind is which slot a row stands for: the repository's primary (:0,
// an ordinary target) or one of its numbered slots (:1+).
type menuRowKind uint8

const (
	menuRowPrimary menuRowKind = iota + 1 // ordinary target: the repository's :0
	menuRowSlot                           // slot target: :1+
)

// menuRowPhase is the one reading of a row's state the action table, Enter and
// the row's own explanation share.
type menuRowPhase uint8

const (
	menuPhaseLive                    menuRowPhase = iota + 1
	menuPhaseLiveContinuationFailed               // #280: a failed request composes with live actions
	menuPhaseLiveContinuationRunning              // the request owns the thread: retry is the only exit
	menuPhaseLiveContinuationPending              // the request is about to run: nothing to do yet
	menuPhaseUnknown                              // unusable/unknown: no verdict this round, offer nothing
	menuPhaseResumable                            // parked or detached
	menuPhaseUnusable                             // unusable, or not live with an unfinished request
	menuPhaseBusy                                 // another couch is starting it
)

// menuRowFacts is everything the action table reads about a row.
type menuRowFacts struct {
	Kind  menuRowKind
	Phase menuRowPhase
	// ResumeOffered is read on unusable rows only: a slot (OpenSlot may adopt
	// a still-running agent), or a primary with Recovery.Recover or an
	// unfinished request (resume routes those to their own executors).
	ResumeOffered bool
	// DirectoryMissing is Reason == ReasonPathMissing.
	DirectoryMissing bool
	AliasOffered     bool // menuAliasOffered
	AddSlotOffered   bool // menuAddSlotPath != ""
}

func menuRowFactsOf(row couchcore.ActionableThreadSummary) menuRowFacts {
	f := menuRowFacts{
		Kind:             menuRowPrimary,
		DirectoryMissing: row.State == couchcore.ThreadUnusable && row.Reason == couchcore.ReasonPathMissing,
		AliasOffered:     menuAliasOffered(row),
	}
	if row.Target.Kind == couchcore.ThreadTargetSlot {
		f.Kind = menuRowSlot
	}
	f.AddSlotOffered = f.Kind == menuRowPrimary && menuAddSlotPath(row) != ""
	unfinished := checkpoint.Phase("")
	if request := row.Continuation; request != nil && request.Phase != checkpoint.Complete {
		unfinished = request.Phase
	}
	switch row.State {
	case couchcore.ThreadBusy:
		f.Phase = menuPhaseBusy
	case couchcore.ThreadLive:
		switch unfinished {
		case checkpoint.Failed:
			f.Phase = menuPhaseLiveContinuationFailed
		case checkpoint.Running:
			f.Phase = menuPhaseLiveContinuationRunning
		case checkpoint.Pending:
			f.Phase = menuPhaseLiveContinuationPending
		default:
			f.Phase = menuPhaseLive
		}
	case couchcore.ThreadParked, couchcore.ThreadDetached:
		f.Phase = menuPhaseResumable
	case couchcore.ThreadUnusable:
		f.Phase = menuPhaseUnusable
		if row.Reason == couchcore.ReasonUnknown {
			f.Phase = menuPhaseUnknown
		}
		f.ResumeOffered = f.Kind == menuRowSlot || unfinished != "" || (row.Recovery != nil && row.Recovery.Recover)
	}
	return f
}

// menuRowActions is the per-row action authority, one table over kind x
// phase. Live rows get the lifecycle actions; rows that are not live get the
// two actor operations, resume and reboot. A row the table offers nothing on
// says why through menuRowNotice, which reads the same phase.
func menuRowActions(f menuRowFacts) []string {
	switch f.Phase {
	case menuPhaseLive, menuPhaseLiveContinuationFailed:
		// Detach first: it is the safe, everyday gesture -- the agent keeps
		// running and only the client goes. Park is destructive and sits
		// behind it, in the position the operator has to travel to.
		items := []string{}
		for _, op := range []string{"detach", "relaunch", "park", "switch-agent"} {
			// A failed request COMPOSES with a live thread's actions (#280).
			// Only what continuationGuard refuses goes, and that list is
			// couchcore's, not restated here.
			if f.Phase == menuPhaseLiveContinuationFailed && couchcore.ContinuationRefuses(op) {
				continue
			}
			items = append(items, op)
			if op == "detach" && f.Phase == menuPhaseLiveContinuationFailed {
				items = append(items, "retry-continuation", "dismiss-continuation")
			}
		}
		if f.Kind == menuRowPrimary && f.AliasOffered {
			items = append(items, "alias")
		}
		if f.Kind == menuRowPrimary && f.AddSlotOffered {
			items = append(items, "add-slot")
		}
		return items
	case menuPhaseLiveContinuationRunning:
		// While a request is in flight the continuation owns the thread:
		// parking or detaching mid-replacement races its own reconciliation.
		// Retry is the exit, including for a request whose owner died (#280).
		return []string{"retry-continuation"}
	case menuPhaseResumable:
		return []string{"resume", "reboot"}
	case menuPhaseUnusable:
		if f.DirectoryMissing {
			// A :0 record outlives its directory, so reboot archives it alone.
			// A :1+ record lives inside its directory: there is nothing left
			// to retire, and add slot is what recreates it.
			if f.Kind == menuRowSlot {
				return nil
			}
			return []string{"reboot"}
		}
		if f.ResumeOffered {
			return []string{"resume", "reboot"}
		}
		return []string{"reboot"}
	}
	// Busy, unknown, live with a pending request: nothing to offer, and
	// menuRowNotice says why. "checking…" is not a verdict, and reboot stops a
	// session.
	return nil
}

// menuActionItems is what a row offers. It is NOT filtered through the
// declaration: a filter made the sweep's offered-implies-declared direction
// unfalsifiable -- offered became a subset of declared by construction -- and
// turned the mistake it was meant to catch into an item silently vanishing from
// the switcher. A guard must be able to fail, and production must not coerce its
// input into agreement. The test reads this function and compares.
func menuActionItems(row couchcore.ActionableThreadSummary) []string {
	return menuRowActions(menuRowFactsOf(row))
}

// menuRowNotice is what a row whose phase the table answers with a fixed
// explanation says, in its status column and on Enter -- read off the same
// facts as its actions, so the explanation cannot drift from the offer. Empty
// means the row's state or reason speaks for itself.
func menuRowNotice(f menuRowFacts) string {
	switch {
	case f.Phase == menuPhaseBusy:
		return "starting elsewhere"
	case f.Phase == menuPhaseUnknown:
		// "checking…" read like progress; it is the absence of a verdict, and
		// reboot stops a session, so nothing is offered until there is one.
		return "state could not be checked"
	case f.DirectoryMissing:
		// The reboot result's own words, so the row and the reboot agree.
		return couchcore.RebootDirectoryMissing
	}
	return ""
}
