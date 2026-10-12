// Package notes is couch broadcast-web's notes model (#461): where a note
// points, the focused context around it, and the screenshot crops a DOM
// submission carries. It is pure, over the reducer's state at a moment.
//
// #457 defines the seam: Model, its stateful fake FakeModel, and
// RunModelContract, which the real model runs as well. A note's wire shape
// (recording.Note, recording.NoteAt) is in atlas/webcast-format.md; it is
// meant to be shared with the terminal notes of pair#455.
package notes

import "github.com/xianxu/pair/cmd/internal/webcast/recording"

// Context is the focused context of a note: the element, its nearest
// meaningful ancestor, and the text around it. Not the whole page.
type Context struct {
	Node recording.NodeID
	// Landmark is the nearest meaningful ancestor (an ARIA landmark, a
	// role, a card or form, a layout box); with none, the parent element.
	Landmark recording.NodeID
	// Text is the landmark's rendered text (what a person reads: no script
	// bodies, blocks kept apart), trimmed to a budget.
	Text string
	// Box is the landmark's box when layout is known.
	Box *recording.Box
}

// Hints override the default crop per app or URL. Crop gets the default
// region for a page and returns the one to use; false keeps the default.
type Hints interface {
	Crop(url string, region recording.Box) (recording.Box, bool)
}

// Model answers where notes point and what to show of the page with them.
// st is the reducer's state at the note's time; nothing here changes it.
type Model interface {
	// Anchor makes an anchor for a live element: its id, a stable
	// selector, its text (at most 200 characters) and its box when layout
	// is known.
	Anchor(st *recording.State, node recording.NodeID) (recording.NoteAt, error)
	// Resolve finds an anchor's element in st: the same node when it is
	// still live, else by its selector; false when it is gone.
	Resolve(st *recording.State, at recording.NoteAt) (recording.NodeID, bool)
	// Context is the focused context of an anchor.
	Context(st *recording.State, at recording.NoteAt) (Context, error)
	// Crops are the screenshot regions for a submission of these anchors:
	// each holds its anchors' boxes plus a margin, snapped to structure;
	// anchors far apart get separate crops. hints may be nil.
	Crops(st *recording.State, ats []recording.NoteAt, hints Hints) ([]recording.Box, error)
	// Live is the notes on the current document: those since it began.
	Live(st *recording.State) []*recording.NoteView
}
