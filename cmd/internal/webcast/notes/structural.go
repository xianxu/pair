package notes

import (
	"fmt"
	"slices"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// Structural is the notes model (#461). Anchors carry a stable selector
// (selector.go); context follows the page's structure, the nearest
// meaningful container (landmark.go); crops are the notes' boxes, padded
// (crop.go). The zero value is ready to use.
type Structural struct {
	// Margin around a note's box in its crop, in CSS px; zero is 16.
	Margin float64
}

var _ Model = Structural{}

func (Structural) Anchor(st *recording.State, node recording.NodeID) (recording.NoteAt, error) {
	n := st.Node(node)
	if n == nil || n.Y != recording.YEl {
		return recording.NoteAt{}, fmt.Errorf("notes: node %d is not a live element", node)
	}
	at := recording.NoteAt{Node: node, Kind: recording.AnchorElement, Path: pathOf(n), Sel: selectorFor(n), Text: trim(n.TextContent())}
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
	c := Context{Node: id, Landmark: land.ID, Text: window(land.RenderedText(), n.RenderedText(), contextBudget)}
	if b, ok := st.Boxes[land.ID]; ok {
		c.Box = &b
	}
	return c, nil
}

func (s Structural) NoteCrops(st *recording.State, ns []*recording.Note) ([]recording.Box, error) {
	return noteCrops(st, ns, marginOr(s.Margin), s.Resolve)
}

func marginOr(m float64) float64 {
	if m == 0 {
		return defaultMargin
	}
	return m
}

func (Structural) Live(st *recording.State) []*recording.NoteView { return liveNotes(st) }
