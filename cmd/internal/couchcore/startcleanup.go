package couchcore

// The cleanup rule for a start that failed AFTER its helper was acknowledged.
//
// By then the helper has execed the target, so two things may need undoing: the
// zellij session, and this start's own durable write. They are INDEPENDENT
// questions, and collapsing them is what pair#230 found: quiescing -- which
// deletes the session and kills its server -- was applied to every post-ack
// failure, including a WARM reattach. A warm reattach attached to a session it
// did not create, and preserving the agent inside that session is its entire
// purpose, so a cancelled reattach destroyed the thing it was reattaching to.
//
// The rule is a pure function so every combination can be enumerated in a
// table, the way ReconcileStart (starttransaction.go) is.

// StartShape is which kind of start failed. It is THREE-valued, not a warm
// boolean, because the two owning shapes already had different tails and this
// rule must not merge them:
//
//   - a spawn reconciles its record in BOTH phases, never rolling back on the
//     session's absence;
//   - a cold resume's claim-phase failure rolls back or marks unknown on the
//     session's absence;
//   - a warm reattach is the one that must not quiesce.
type StartShape string

const (
	// StartSpawn created both the thread and its session.
	StartSpawn StartShape = "spawn"
	// StartColdResume relaunched a parked thread, creating a new session.
	StartColdResume StartShape = "cold-resume"
	// StartWarmReattach attached to a session that predates it.
	StartWarmReattach StartShape = "warm-reattach"
)

// OwnsSession is THE authority on whether cleanup may end this start's zellij
// session, and the only question `Quiesce` ever was: a start ends the session
// it created and never one it borrowed.
//
// It is written as a positive test of the two owning shapes so that an
// unrecognised value -- an empty string from a record some other version wrote
// -- answers NO. Guessing wrong in that direction leaves a session behind;
// guessing wrong in the other kills somebody's agent.
func (s StartShape) OwnsSession() bool { return s == StartSpawn || s == StartColdResume }

// SessionPresence is what could be observed about the thread's zellij session.
// Unobserved is not absent: the question could not be asked, which is the case
// that must never be answered destructively.
type SessionPresence uint8

const (
	PresenceUnobserved SessionPresence = iota
	PresenceAbsent
	PresencePresent
)

// DurableAction is what happens to the thread record.
type DurableAction string

const (
	// DurableRollback removes this start's own claim, leaving no incarnation.
	DurableRollback DurableAction = "rollback"
	// DurableRetire retires the live incarnation, returning the thread to
	// detached -- the same disposition Detach applies.
	DurableRetire DurableAction = "retire"
	// DurableMarkUnknown leaves the record occupied but flagged. It is the
	// fail-closed answer: recoverable, where a deleted session is not.
	DurableMarkUnknown DurableAction = "mark-unknown"
	// DurableReconcile is the tail that predates pair#230: reconcile
	// interrupted starts against their registration evidence, then mark this
	// exact live incarnation unknown if one is still recorded.
	DurableReconcile DurableAction = "reconcile"
)

// StartCleanupInput is everything the rule reads.
type StartCleanupInput struct {
	Shape StartShape
	// HelperDead is true when the exact helper process is proven gone. Nothing
	// durable is undone while a process that may still be writing is
	// unaccounted for.
	HelperDead bool
	// Presence is the thread's session, observed after the helper was ended.
	Presence SessionPresence
	// LiveRecord distinguishes a start that reached a live incarnation (the
	// registry-persist and console-attach failures) from one still holding a
	// claim (everything before registration was promoted).
	LiveRecord bool
}

// DecideStartCleanup answers what happens to the RECORD. The session question
// is StartShape.OwnsSession, answered earlier and separately, because the two
// are decided at different moments: the session is ended first, and its absence
// afterwards is an input here.
//
// Every branch for an owning shape reproduces the behaviour that predates
// pair#230 exactly, so the existing spawn and cold-resume tests are what prove
// they did not change.
func DecideStartCleanup(in StartCleanupInput) DurableAction {
	out := DurableMarkUnknown
	switch {
	case in.Shape == StartSpawn:
		// Unchanged, both phases: reconcile every interrupted start against its
		// registration evidence, then mark a live incarnation unknown.
		out = DurableReconcile
	case in.LiveRecord:
		// Routes 5-6 for either resume shape. An owning one keeps the reconcile
		// tail; a warm one gives the thread back to detached, under the same two
		// proofs Detach requires.
		if in.Shape.OwnsSession() {
			out = DurableReconcile
		} else if in.HelperDead && in.Presence == PresencePresent {
			out = DurableRetire
		}
	case in.Shape == StartColdResume:
		// Unchanged: roll back only once the session it created is gone AND
		// its helper is dead; otherwise leave the record recoverable.
		if in.HelperDead && in.Presence == PresenceAbsent {
			out = DurableRollback
		}
	default:
		// Warm, still a claim. Presence is deliberately NOT consulted:
		// rollback removes this start's claim and nothing else, so it is right
		// whether the session survived (the thread reads detached again) or
		// died on its own (it reads session-gone). Demanding presence here
		// would strand a thread whose session died for unrelated reasons.
		if in.HelperDead {
			out = DurableRollback
		}
	}
	return out
}
