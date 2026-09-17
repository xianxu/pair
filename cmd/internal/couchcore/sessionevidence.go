package couchcore

import (
	"context"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

// SessionState is what couch knows about one thread's zellij session.
//
// Three values, not a boolean, because the two ways of not having a session are
// not the same fact. After #256 recoverability is keyed to the session and
// `session-gone` is archive-eligible, so collapsing "could not ask" into "no
// session" would offer to retire a thread whose agent is still running.
//
// It is NOT four values. An earlier design added `SessionHeldElsewhere` for a
// session with a client attached, but the refresh path never asks about clients
// -- `list-clients` costs ~250 ms per session (#228) and the reattach path
// re-observes with attach state before committing (`resume.go`). A value no
// producer can emit is a state that exists only to be handled.
type SessionState uint8

const (
	// SessionUnresolved is the ZERO VALUE on purpose: an observation nobody
	// populated must fail closed. If absence were the zero value, a gather
	// branch that silently stopped running would assert "no session" for every
	// thread it skipped -- the shape of the anonymous refusals #181 removed.
	SessionUnresolved SessionState = iota
	// SessionAbsent means the question was asked and no live session is bound.
	SessionAbsent
	// SessionPresent means a live, non-exited session is bound to this address.
	SessionPresent
)

func (s SessionState) String() string {
	switch s {
	case SessionAbsent:
		return "absent"
	case SessionPresent:
		return "present"
	}
	return "unresolved"
}

// SessionObservation is one thread's session, and whether couch could look.
type SessionObservation struct {
	State SessionState
	// Name is the zellij session this address is bound to. Carried so a
	// consumer that acts on the observation does not re-derive the name from a
	// second index read, which is how the name a thread is judged by and the
	// name it is acted on could drift apart.
	Name string
}

func (o SessionObservation) Present() bool { return o.State == SessionPresent }

// SessionPresenceResolver answers existence for many threads with one host-wide
// snapshot.
//
// It exists beside DetachedSessionResolver rather than replacing it because they
// ask different questions at different prices. This one asks "is there a live
// session", from one `list-sessions`, for EVERY record. That one asks "is there
// a live session with no client attached", which costs a `list-clients` per
// candidate, and it stays the authority for the ACTION path -- where
// `RequireAttachState` refuses a snapshot that never asked.
type SessionPresenceResolver interface {
	SessionPresence(ctx context.Context, addresses []ThreadAddress) (map[ThreadAddress]SessionObservation, error)
}

// ProjectSessionPresence is the pure existence rule: one observation per binding
// it was given.
//
// Fail-closed in both ambiguous directions, matching ProjectDetachedSessions,
// because a wrong answer here decides whether a thread is recoverable or debris:
//
//   - two addresses bound to one session name: couch cannot tell whose session
//     that is, so neither gets an answer.
//   - two snapshot rows sharing one name: the snapshot contradicts itself, so
//     that name proves nothing.
//
// An address with no binding is simply not in the result, and reading a missing
// key yields the zero value -- unresolved. That is deliberate: the caller owns
// the distinction between "no row in the index" (asked, absent) and "index
// unreadable" (not asked), because only the caller knows which happened.
// `claims` counts each name over every binding the caller READ, not just the
// ones passed in -- the same widening ProjectDetachedSessions requires. A caller
// that asks about a subset would otherwise see a contested name as unique and
// call a thread recoverable whose session belongs to something else (#206).
func ProjectSessionPresence(bindings []SessionNameBinding, sessions []launcher.Session, claims map[string]int) map[ThreadAddress]SessionObservation {
	out := make(map[ThreadAddress]SessionObservation, len(bindings))
	if len(bindings) == 0 {
		return out
	}

	live := make(map[string]bool, len(sessions))
	ambiguous := make(map[string]bool, len(sessions))
	for _, session := range sessions {
		if session.Name == "" {
			continue
		}
		if _, seen := live[session.Name]; seen {
			ambiguous[session.Name] = true
			continue
		}
		live[session.Name] = session.State != launcher.SessionExited
	}

	for _, binding := range bindings {
		observation := SessionObservation{Name: binding.SessionName}
		switch {
		case binding.SessionName == "":
			// A binding with no name answers nothing; the caller decides
			// whether that means "no row" or "could not read".
			observation.State = SessionUnresolved
		case claims[binding.SessionName] != 1 || ambiguous[binding.SessionName]:
			observation.State = SessionUnresolved
		case live[binding.SessionName]:
			observation.State = SessionPresent
		default:
			// Listed and exited, or not listed at all. Both are the honest
			// "asked, and there is no live session" -- an EXITED row is a
			// resurrect record, not a running server (#67).
			observation.State = SessionAbsent
		}
		out[binding.Address] = observation
	}
	return out
}
