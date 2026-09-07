package couchcore

import "fmt"

// Layout is which pair layout couch launches its threads in. It is chosen once
// per couch process (Couch.Layout) and recorded per thread
// (ThreadRecord.Layout) as a witness of what that thread's session actually is.
type Layout string

const (
	Layout2 Layout = "layout2"
	Layout3 Layout = "layout3"
	// LayoutUnknown is a persisted value this binary does not recognise. It is
	// never chosen and never formatted: it exists so a row can carry "we cannot
	// prove this session's layout" instead of a fabricated layout2 the guard
	// would go on to trust. It conflicts with every requested layout.
	LayoutUnknown Layout = "unknown"
)

// ParseLayout turns an untrusted string -- a CLI flag or a persisted field --
// into a Layout. Empty means a record written before layout was recorded, and
// those are layout2 with certainty: couch pinned layout2 from 2026-08-22 until
// #198. An unrecognised value is refused rather than defaulted.
func ParseLayout(raw string) (Layout, error) {
	switch Layout(raw) {
	case "":
		return Layout2, nil
	case Layout2:
		return Layout2, nil
	case Layout3:
		return Layout3, nil
	}
	return "", fmt.Errorf("unknown layout %q", raw)
}

// NormalizeLayout is ParseLayout for the one caller that cannot return an
// error: ProjectActionableThreads. An unrecognised value becomes LayoutUnknown
// rather than a default, so the guard refuses visibly instead of proceeding on
// a value nothing verified.
func NormalizeLayout(raw string) Layout {
	layout, err := ParseLayout(raw)
	if err != nil {
		return LayoutUnknown
	}
	return layout
}

// Flag is the sole place a Layout becomes argv. LayoutUnknown never reaches it:
// Couch.Layout is only ever set from ParseLayout, which errors instead of
// returning it.
func (l Layout) Flag() string { return "--" + string(l) }

// LayoutConflict is one thread whose existing session disagrees with the layout
// couch was asked to start in.
type LayoutConflict struct {
	Address ThreadAddress
	Layout  Layout
	State   ActionableThreadState
}

// holdsSession reports the states whose zellij session is alive right now, and
// whose layout couch therefore cannot change: asking a live session for a
// different layout sends pair down the conflict path that offers to DELETE it
// (#179). Busy is included because a park in flight can still fail, leaving the
// session alive in its old layout.
//
// Parked is excluded deliberately -- park ends the session via the lifecycle
// quit protocol, so a parked thread's next cold resume takes couch's layout
// freely. That exclusion is what makes "park them first" the reachable remedy
// for a refusal rather than a dead end.
func (s ActionableThreadSummary) holdsSession() bool {
	return s.State == ThreadLive || s.State == ThreadDetached || s.State == ThreadBusy
}

// ResolveLayoutConflicts reports the threads that stop couch starting in
// `requested`. LayoutUnknown conflicts with everything: it means the record
// carried a value this binary cannot read, so agreement cannot be proved.
//
// This is deliberately NOT Resumable() (Parked||Detached), which would refuse
// startups that are safe.
func ResolveLayoutConflicts(requested Layout, rows []ActionableThreadSummary) []LayoutConflict {
	var conflicts []LayoutConflict
	for _, row := range rows {
		if !row.holdsSession() || row.Layout == requested {
			continue
		}
		conflicts = append(conflicts, LayoutConflict{
			Address: row.Address, Layout: row.Layout, State: row.State,
		})
	}
	return conflicts
}
