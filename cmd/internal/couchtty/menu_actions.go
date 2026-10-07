package couchtty

import (
	"slices"

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
	// Actor is what couchcore's resume/reboot admission table reads; the
	// resumable and unusable phases offer exactly couchcore.ActorActions.
	Actor          couchcore.ActorRowFacts
	AliasOffered   bool // menuAliasOffered
	AddSlotOffered bool // menuAddSlotPath != ""
}

func menuRowFactsOf(row couchcore.ActionableThreadSummary) menuRowFacts {
	f := menuRowFacts{
		Kind:         menuRowPrimary,
		Actor:        couchcore.ActorRowFactsOf(row),
		AliasOffered: menuAliasOffered(row),
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
	}
	return f
}

// menuRowActions is the per-row action authority, one table over kind x
// phase. Live rows get the lifecycle actions; rows that are not live get the
// two actor operations, resume and reboot, and a :0 among them also gets add
// slot unless its checkout is missing (pair#402). A row the table offers
// nothing on says why through menuRowNotice, which reads the same phase.
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
	case menuPhaseResumable, menuPhaseUnusable:
		// The actor operations are couchcore's one admission table, shared
		// with the recover-plan report (pair#367). A :1+ row whose directory
		// is missing offers nothing; menuRowAdviceOf says what brings it back.
		items := couchcore.ActorActions(f.Actor)
		// Adding a slot opens the start form for a new :1+ worktree and needs
		// nothing from :0's agent, so a parked :0 offers it too (pair#402). It
		// does need the primary checkout: a :0 whose directory is gone does not.
		if f.AddSlotOffered && !f.Actor.DirectoryMissing {
			items = append(items, "add-slot")
		}
		return items
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

// menuNextStep is one row-facing text that may tell the operator what to do
// next, and which row the action it names is taken on.
type menuNextStep struct {
	Text string
	// OnPrimary: the action Text names is taken on the repository's :0
	// row, not on this one. Only a :1+ row may say so.
	OnPrimary bool
}

// menuRowAdvice is every row-facing text that names a next step: the row's
// status explanation (its status column and Enter's reason), Enter's way
// forward when it will not act, and the reboot confirmation's clause where
// reboot can start nothing. It is chosen here, per kind and phase, from the
// same facts as menuRowActions, because a next step must name an action the
// row's kind can reach -- three findings in a row (lessons:
// refusal-names-unoffered-action) were hand-written advice naming an action
// the row did not offer. TestRowAdviceNamesOnlyReachableActions sweeps the
// derived row domain.
type menuRowAdvice struct {
	Notice     menuNextStep
	Enter      menuNextStep
	RebootCost menuNextStep
}

func menuRowAdviceOf(f menuRowFacts) menuRowAdvice {
	var a menuRowAdvice
	switch {
	case f.Phase == menuPhaseBusy:
		a.Notice.Text = "starting elsewhere"
	case f.Phase == menuPhaseUnknown:
		// "checking…" read like progress; it is the absence of a verdict, and
		// reboot stops a session, so nothing is offered until there is one.
		a.Notice.Text = "state could not be checked"
	case f.Actor.DirectoryMissing && f.Kind == menuRowSlot:
		// The reboot result's own words, so the row and the reboot agree. A
		// :1+ record lives inside its directory and offers nothing; add slot,
		// on the repository's :0, recreates the directory.
		a.Notice = menuNextStep{Text: couchcore.RebootDirectoryMissing, OnPrimary: true}
	case f.Actor.DirectoryMissing:
		// A :0 record outlives its checkout: reboot archives it alone, and
		// only the checkout coming back lets an agent start there again.
		a.Notice.Text = couchcore.RebootCheckoutMissing
		a.RebootCost.Text = " — checkout missing: archives the record only; restore the checkout to start here again"
	}
	if slices.Contains(menuRowActions(f), "reboot") {
		a.Enter.Text = "Tab → reboot"
	}
	return a
}

// menuRowNotice is what a row whose phase the table answers with a fixed
// explanation says, in its status column and on Enter. Empty means the row's
// state or reason speaks for itself.
func menuRowNotice(f menuRowFacts) string {
	return menuRowAdviceOf(f).Notice.Text
}
