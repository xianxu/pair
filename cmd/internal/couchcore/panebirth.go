package couchcore

import "time"

// PaneMarks is one observation of a thread's agent pane sidecars
// (`pane-<tag>-<agent>.json`), path → mtime.
//
// A sidecar is Pair's birth evidence for a session (#287). The layout's pane
// command writes it as its first act, and a pane exists only once the first
// zellij client has initialized the session. Until then, zellij 0.45.1 panics
// when a connection it accepted closes, and `list-sessions` connects to every
// session socket. So a cold resume must not ask zellij anything until its
// pane has been born (probes/zellijbirthrace: 10 launches in 10 died under a
// 10 ms list-sessions loop, the cadence of the registration poll).
type PaneMarks map[string]time.Time

// BornIn reports whether now shows a pane born since m was observed: a sidecar
// m did not have, or one whose mtime moved.
//
// It compares equality only, never order, so no clock is set against another.
// Two things that are not births:
//   - A sidecar that disappeared. The launcher clears this agent's before it
//     starts zellij.
//   - A stale twin left by another agent. It sits unchanged.
func (m PaneMarks) BornIn(now PaneMarks) bool {
	for path, mtime := range now {
		if before, ok := m[path]; !ok || !before.Equal(mtime) {
			return true
		}
	}
	return false
}

// PaneBirthIO observes a thread's agent pane sidecars. A scope with no sidecars
// answers an empty PaneMarks, not an error.
type PaneBirthIO interface {
	PaneSidecars(ThreadAddress) (PaneMarks, error)
}
