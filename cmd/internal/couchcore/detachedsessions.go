package couchcore

import "github.com/xianxu/pair/cmd/internal/launcher"

// SessionNameBinding is one {thread address -> zellij session name} row, read
// from a repo scope's session-name index.
//
// The index is Pair's stable socket binding and nothing else: Couch's mutable
// human thread name never renames it. It is the only bridge from a durable
// composite address to the name zellij knows.
type SessionNameBinding struct {
	Address     ThreadAddress
	SessionName string
}

// ProjectDetachedSessions is the pure detached rule: a thread is detached when
// its bound zellij session is still alive with no client attached.
//
// It reuses launcher's existing classification rather than teaching Couch a
// second way to ask whether a session is alive -- `SessionDetached` already
// means "live server session, zero clients", which is exactly the state an
// `alt+d` leaves behind and exactly what `pair resume` reattaches onto.
//
// Fail-closed in both ambiguous directions, because a wrong answer here puts a
// row in the switcher whose Enter cannot work:
//
//   - two addresses bound to one session name: Couch cannot tell which thread
//     that session belongs to, so neither gets a row.
//   - two zellij rows sharing one name: the snapshot itself is contradictory,
//     so that name proves nothing.
func ProjectDetachedSessions(bindings []SessionNameBinding, sessions []launcher.Session) []DetachedSessionObservation {
	if len(bindings) == 0 || len(sessions) == 0 {
		return nil
	}
	state := make(map[string]launcher.SessionState, len(sessions))
	ambiguousSession := make(map[string]bool, len(sessions))
	for _, session := range sessions {
		if session.Name == "" {
			continue
		}
		if _, seen := state[session.Name]; seen {
			ambiguousSession[session.Name] = true
			continue
		}
		state[session.Name] = session.State
	}

	claims := make(map[string]int, len(bindings))
	for _, binding := range bindings {
		if binding.SessionName != "" {
			claims[binding.SessionName]++
		}
	}

	var out []DetachedSessionObservation
	for _, binding := range bindings {
		if binding.SessionName == "" || claims[binding.SessionName] != 1 || ambiguousSession[binding.SessionName] {
			continue
		}
		if state[binding.SessionName] != launcher.SessionDetached {
			continue
		}
		out = append(out, DetachedSessionObservation{Address: binding.Address, SessionName: binding.SessionName})
	}
	return out
}
