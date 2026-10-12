package notes

import (
	"fmt"
	"slices"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// Structural is the notes model (#461). Anchors carry a stable selector
// (selector.go); context and crops follow the page's structure: the nearest
// meaningful container (landmark.go), crops padded, snapped to it and
// clustered (crop.go). The zero value is ready to use.
type Structural struct {
	// Margin around the notes in a crop, in CSS px; zero is 48.
	Margin float64
}

var _ Model = Structural{}

func (Structural) Anchor(st *recording.State, node recording.NodeID) (recording.NoteAt, error) {
	n := st.Node(node)
	if n == nil || n.Y != recording.YEl {
		return recording.NoteAt{}, fmt.Errorf("notes: node %d is not a live element", node)
	}
	at := recording.NoteAt{Node: node, Sel: selectorFor(n), Text: trim(n.TextContent())}
	if !slices.Contains(matchSelector(st, at.Sel), n) { // never store a selector that can't find its element
		at.Sel = ""
	}
	if b, ok := st.Boxes[node]; ok {
		at.Box = &b
	}
	return at, nil
}

// Resolve: the same node while it lives (ids are never reused), else the
// selector's match: the one whose text is the anchor's, or the only one.
// Several matches with no text to tell them apart don't resolve.
func (Structural) Resolve(st *recording.State, at recording.NoteAt) (recording.NodeID, bool) {
	if n := st.Node(at.Node); n != nil && n.Y == recording.YEl {
		return at.Node, true
	}
	m := matchSelector(st, at.Sel)
	for _, n := range m {
		if at.Text != "" && trim(n.TextContent()) == at.Text {
			return n.ID, true
		}
	}
	if len(m) == 1 {
		return m[0].ID, true
	}
	return 0, false
}

func (s Structural) Context(st *recording.State, at recording.NoteAt) (Context, error) {
	id, ok := s.Resolve(st, at)
	if !ok {
		return Context{}, fmt.Errorf("notes: the element of %q (node %d) is gone", at.Sel, at.Node)
	}
	n := st.Node(id)
	land, _ := landmark(n)
	c := Context{Node: id, Landmark: land.ID, Text: window(land.TextContent(), n.TextContent(), contextBudget)}
	if b, ok := st.Boxes[land.ID]; ok {
		c.Box = &b
	}
	return c, nil
}

func (s Structural) Crops(st *recording.State, ats []recording.NoteAt, hints Hints) ([]recording.Box, error) {
	return s.NoteCrops(st, anchorsOnly(ats), hints)
}

func (s Structural) NoteCrops(st *recording.State, ns []*recording.Note, hints Hints) ([]recording.Box, error) {
	m := s.Margin
	if m == 0 {
		m = defaultMargin
	}
	return crops(st, ns, m, hints, s.Resolve)
}

func (Structural) Live(st *recording.State) []*recording.NoteView { return liveNotes(st) }
