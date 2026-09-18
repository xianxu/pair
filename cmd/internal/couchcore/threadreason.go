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
	// RETIRED in #256: `stale-incarnation` and `unrecorded-child`.
	//
	// Both named a DISAGREEMENT between the record's incarnation and what couch
	// could observe -- one for each direction. The classifier no longer consults
	// the incarnation, so there are no longer two sides to disagree: couch's own
	// observation is the live proof and the session is the recoverability proof.
	//
	// `stale-incarnation` was the more expensive of the two. It described every
	// couch crash as a lost thread, because the incarnation names the launcher,
	// which is couch's own child and dies with it -- measured on the operator's
	// store, all 11 records carrying it had a dead pid and three had an agent
	// still running.
	//
	// `unrecorded-child` will come back in #276, which gives it a producer: a
	// couch-tagged session with NO record at all. It is removed rather than kept
	// as a placeholder because a vocabulary entry nothing produces is exactly
	// what TestEveryReasonIsProducedBySomeShape exists to forbid, and an
	// exemption would silence the guard for every future orphan too.

	// ReasonSessionGone is a thread with no surviving session -- the honest
	// "finished" shape.
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
	// never read as debris and it still BLOCKS its path -- a record couch
	// cannot read is not evidence that the path is free. The OPERATOR can
	// still archive one (ArchivableState permits it), because that escape is
	// what keeps a corrupt record from locking its repository; doing so moves
	// the record's bytes and never stops its session.
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
