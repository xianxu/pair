// Package query is couch broadcast-web's agent queries (#462): what an agent
// asks of a recording. The page at time t, what happened in a window, and
// every note with its element, its focused context and what the user did
// just before.
//
// #457 defines the seam: Surface, its stateful fake FakeSurface, and
// RunSurfaceContract, which the real surface (a Couch CLI) runs as well.
package query

import (
	"github.com/xianxu/pair/cmd/internal/webcast/notes"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// NoteView is a note as an agent reads it.
type NoteView struct {
	// Elements owns literal declarations at this historical context moment.
	Elements DeclaredElementFacts

	Note *recording.NoteView
	// Element is the note's element at the note's time, as compact text;
	// empty when it was already gone.
	Element string
	// Context is the notes model's focused context.
	Context notes.Context
	// Before is what the user did in the window before the note: clicks,
	// form input, focus, navigation, scrolling, selection, and segment
	// starts and stops (UserAction).
	Before []recording.Record
	// Landmark describes Context.Landmark in one line, as of the note's
	// time (Describe); URL and Header are the page's at that time. The
	// real surface fills them; the fake leaves them empty.
	Landmark, URL, Header string
	// BeforeLines are Before in words, one per record with its time,
	// targets named as they were (the real surface only).
	BeforeLines []string
}

// Surface answers an agent's questions about one recording. Times are ms
// since the recording started.
type Surface interface {
	// DOMAt is the page at t as compact readable text, under selector when
	// it is not empty.
	DOMAt(t int64, selector string) (string, error)
	// Events are the records with t1 <= t <= t2, in order.
	Events(t1, t2 int64) ([]recording.Record, error)
	// Notes are every note, in order, with the user's actions in the
	// before ms preceding each.
	Notes(before int64) ([]NoteView, error)
	// SketchesAt freezes original source facts and an optional exact accepted prefix.
	SketchesAt(at int64, selected ...SketchSelection) ([]SketchView, error)
}

// UserAction reports whether a record is something the user (or the page
// acting on its own) did, as opposed to a DOM change. A segment's start and
// stop count too (a navigation, a tab switch, a pause and resume), so the
// actions before a note never run across a page change unmarked (#462).
func UserAction(r recording.Record) bool {
	switch r.(type) {
	case *recording.Click, *recording.Input, *recording.Focus, *recording.Nav, *recording.Scroll, *recording.Selection,
		*recording.Seg, *recording.Stop:
		return true
	}
	return false
}
