package couchcore

// ThreadReason is why a thread is not actionable.
//
// The switcher used to answer that question with `continue`: nine of the
// operator's thirteen threads reached no row, with no notice, no log line and
// no way to ask why (#181). A closed vocabulary replaces those anonymous
// refusals, and it is the same vocabulary the archive rule is written over, so
// what the operator is shown and what may be retired cannot drift apart.
type ThreadReason string

const (
	// ReasonBindingLost is a thread whose durable state is fine but whose
	// resume proof is gone -- pair#168's shape, where a launch ledger row with
	// no binding after it shadows the last established binding. RECOVERABLE:
	// never retire one.
	ReasonBindingLost ThreadReason = "binding-lost"
	// ReasonStaleIncarnation is a record claiming a live incarnation that
	// nothing hosts: the shape a couch that died without leaving cleanly
	// leaves behind (pair#171). Reconcilable, so never retired either.
	ReasonStaleIncarnation ThreadReason = "stale-incarnation"
	// ReasonUnrecordedChild is the opposite disagreement: a hosted child for a
	// record carrying no incarnation. It should be unreachable; failing closed
	// beats guessing which side is right.
	ReasonUnrecordedChild ThreadReason = "unrecorded-child"
	// ReasonSessionGone is a thread with no incarnation and no surviving
	// session -- the honest "finished" shape.
	ReasonSessionGone ThreadReason = "session-gone"
	// ReasonNeverStarted is a reservation that never became a running thread.
	ReasonNeverStarted ThreadReason = "never-started"
	// ReasonInvalid is a record that fails ValidateThreadRecord.
	ReasonInvalid ThreadReason = "invalid"
	// ReasonUnreadable is a manifest-listed record couch could not read or
	// decode at all -- distinct from `invalid`, which is a verdict about a
	// record that WAS read.
	//
	// The distinction is the same one ProofStatus draws one layer up. An older
	// couch reading a store written by a newer one cannot decode any record:
	// calling that "invalid" would classify every thread as debris and offer to
	// archive the operator's live work. Unreadable means unknown, so it is
	// never archive-eligible and it still BLOCKS its path -- a record couch
	// cannot read is not evidence that the path is free.
	ReasonUnreadable ThreadReason = "unreadable"
	// ReasonPathMissing is a working path that could not be physicalized. It
	// must stay a refusal: SelectResumableRoot compares paths by exact
	// string, so an unphysicalized row could be auto-selected at startup.
	ReasonPathMissing ThreadReason = "path-missing"
	// ReasonProfileMissing is a thread with no saved launch profile to resume from.
	ReasonProfileMissing ThreadReason = "profile-missing"
	// ReasonAgentUnsupported is a saved profile naming an agent this build
	// cannot launch.
	ReasonAgentUnsupported ThreadReason = "unsupported-agent"
	// ReasonUnknown is the evidence itself failing to resolve this round.
	//
	// It exists because absence of proof is not proof of absence, and a total
	// classifier that cannot say so turns every unresolved question into a
	// positive claim: one failed zellij query would assert session-gone on
	// every detached row, and session-gone is a reason retirement acts on. This
	// is the only transient reason: it says the evidence did not resolve this
	// round, not that anything about the thread is settled.
	ReasonUnknown ThreadReason = "unknown"
)

// AllThreadReasons is the vocabulary itself, so display tables and the
// retirement rule iterate it rather than restating it. Go cannot check a switch
// for exhaustiveness; this enumeration is what does.
func AllThreadReasons() []ThreadReason {
	return []ThreadReason{
		ReasonBindingLost,
		ReasonStaleIncarnation,
		ReasonUnrecordedChild,
		ReasonSessionGone,
		ReasonNeverStarted,
		ReasonInvalid,
		ReasonUnreadable,
		ReasonPathMissing,
		ReasonProfileMissing,
		ReasonAgentUnsupported,
		ReasonUnknown,
	}
}

// Label is the operator's wording for a reason, and the single source of it.
//
// Three surfaces render these -- the switcher column, the switcher's Enter
// notice and `couch --list` -- and only two had a guard, so the CLI printed raw
// slugs while the switcher printed English. One switch, one guard, and a reason
// added later cannot ship unlabelled anywhere.
func (r ThreadReason) Label() string {
	switch r {
	case ReasonBindingLost:
		return "binding lost — repairable"
	case ReasonStaleIncarnation:
		return "stale — couch exited unexpectedly"
	case ReasonUnrecordedChild:
		return "running but unrecorded"
	case ReasonSessionGone:
		return "session gone"
	case ReasonNeverStarted:
		return "never started"
	case ReasonInvalid:
		return "record failed validation"
	case ReasonUnreadable:
		return "could not be read — may need a newer couch"
	case ReasonPathMissing:
		return "path unavailable"
	case ReasonProfileMissing:
		return "no saved launch"
	case ReasonAgentUnsupported:
		return "unsupported agent"
	case ReasonUnknown:
		return "checking…"
	}
	// Legible beats silent: an unlabelled reason shows its slug rather than an
	// empty column, and the vocabulary guard fails so it does not stay that way.
	return string(r)
}
