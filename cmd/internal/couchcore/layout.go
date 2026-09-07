package couchcore

import (
	"fmt"
	"strings"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

// Layout is which pair layout couch launches its threads in. It is chosen once
// per couch process (Couch.Layout) and recorded per thread
// (ThreadRecord.Layout) as a witness of what that thread's session actually is.
//
// It is an ALIAS for launcher.LayoutMode, not a second type: launcher already
// owns this vocabulary, its parse, and the argv spellings that `pair` accepts.
// couch's whole feature rests on pair parsing the flag couch emits, so the two
// must not be able to drift -- see TestCouchLayoutFlagsAreWhatPairParses.
type Layout = launcher.LayoutMode

const (
	Layout2 = launcher.Layout2
	Layout3 = launcher.Layout3
	// LayoutUnknown is a persisted value this binary does not recognise. It is
	// never chosen and never formatted: it exists so a row can carry "we cannot
	// prove this session's layout" instead of a fabricated layout2 the guard
	// would go on to trust. It conflicts with every requested layout, and
	// KnownLayout reports false for it.
	LayoutUnknown Layout = "unknown"
)

// ParseLayout turns an untrusted string -- a CLI flag or a persisted field --
// into a Layout. Empty means a record written before layout was recorded, and
// those are layout2 with certainty: couch pinned layout2 from 2026-08-22 until
// #198. An unrecognised value is refused rather than defaulted.
func ParseLayout(raw string) (Layout, error) {
	if strings.TrimSpace(raw) == "" {
		return Layout2, nil
	}
	layout, ok := launcher.ParseLayoutMode(raw)
	if !ok {
		return "", fmt.Errorf("unknown layout %q", raw)
	}
	return layout, nil
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

// KnownLayout reports whether a Layout is one couch can actually launch, and so
// whether it can appear in a command suggested to the operator. LayoutUnknown
// and any unrecognised value are not.
func KnownLayout(layout Layout) bool {
	_, ok := launcher.ParseLayoutMode(string(layout))
	return ok
}

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

// layoutRemedy is the way forward a refusal ends with. A refusal that does not
// terminate in an action the operator can actually take is not actionable, so
// the three cases are answered separately rather than collapsed into one
// sentence that is wrong for two of them.
//
//   - every conflict in ONE known layout: park them from a couch that can host
//     them, then start the one that was asked for.
//   - conflicts in DIFFERENT known layouts: still reachable, but not in one
//     pass -- each thread parks from the couch matching its own layout.
//     Reachable when a process dies between a launch and its witness CAS.
//   - any conflict whose layout is UNREADABLE: no couch can host it, so `park`
//     (a TUI row action, ops.go) cannot be reached for it at all. The remedy
//     leaves the tool: inspect the thread and end its session directly.
func layoutRemedy(requested Layout, conflicts []LayoutConflict) string {
	host, single := Layout(""), true
	unreadable := ""
	for i, conflict := range conflicts {
		if !KnownLayout(conflict.Layout) {
			if unreadable == "" {
				unreadable = string(conflict.Address.Tag)
			}
			continue
		}
		if i == 0 || host == "" {
			host = conflict.Layout
			continue
		}
		if conflict.Layout != host {
			single = false
		}
	}
	if unreadable != "" {
		// No `couch --layoutN` will start while this thread holds its session,
		// so every in-tool route is closed. Say what is left.
		return "this thread's layout cannot be read, so no couch can host it and\n" +
			"  `park` is out of reach. Inspect it and end its session directly:\n" +
			"    couch --show " + unreadable + "\n" +
			"    zellij kill-session <the session it names>"
	}
	if !single {
		return "these threads are in DIFFERENT layouts, so no single couch can park\n" +
			"  them all. Park each from the couch matching its own layout, then:\n" +
			"    couch " + requested.Flag()
	}
	return "park them first:  couch " + host.Flag() +
		"   then park each thread from the switcher and quit\n" +
		"  then:             couch " + requested.Flag()
}

// layoutConflictRefusal makes a mixed-layout refusal actionable, in the shape
// startupResumeRefusal established: what happened, which threads, and the way
// forward.
//
// The way forward normally exists BECAUSE the blocking set excludes parked
// threads: parking every conflicting thread empties the set, so the operator
// reaches the layout they asked for through the tool rather than by hand-editing
// records. layoutRemedy handles the cases where that route is not available.
func layoutConflictRefusal(requested Layout, conflicts []LayoutConflict) error {
	if len(conflicts) == 0 {
		return nil
	}
	width := 0
	for _, conflict := range conflicts {
		if n := len(conflict.Address.Tag); n > width {
			width = n
		}
	}
	var rows strings.Builder
	for _, conflict := range conflicts {
		layout := string(conflict.Layout)
		if !KnownLayout(conflict.Layout) {
			layout = "unreadable layout"
		}
		fmt.Fprintf(&rows, "\n  %-*s  (%s, %s)", width, conflict.Address.Tag, layout, conflict.State)
	}
	noun := "thread"
	if len(conflicts) > 1 {
		noun = "threads"
	}
	return fmt.Errorf(
		"cannot start in %s: %d %s already hold a session in another layout:%s\n\n"+
			"couch keeps one layout across every thread, so it will not mix them.\n"+
			"  %s",
		requested, len(conflicts), noun, rows.String(), layoutRemedy(requested, conflicts))
}
