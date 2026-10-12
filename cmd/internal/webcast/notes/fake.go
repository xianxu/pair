package notes

import (
	"fmt"
	"strings"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// FakeModel is the stateful fake Model, deliberately simple: the selector is
// an id or a child-index path, and the landmark is the parent element.
// Crops are the real model's (#488). Enough for the queries (#462) to build on
// before the real model (#461) lands.
type FakeModel struct {
	// Margin around a note's box in its crop, in CSS px; zero is 16.
	Margin float64
}

const textBudget = 200

func (FakeModel) Anchor(st *recording.State, node recording.NodeID) (recording.NoteAt, error) {
	n := st.Node(node)
	if n == nil || n.Y != recording.YEl {
		return recording.NoteAt{}, fmt.Errorf("notes: node %d is not a live element", node)
	}
	at := recording.NoteAt{Node: node, Kind: recording.AnchorElement, Path: []string{n.N}, Sel: selector(n), Text: trim(n.TextContent())}
	if b, ok := st.Boxes[node]; ok {
		at.Box = &b
	}
	return at, nil
}

func (FakeModel) Resolve(st *recording.State, at recording.NoteAt) (recording.NodeID, bool) {
	if n := st.Node(at.Node); n != nil && n.Y == recording.YEl {
		return at.Node, true
	}
	if st.Top == nil {
		return 0, false
	}
	if n := find(st.Top, func(n *recording.DOMNode) bool { return n.Y == recording.YEl && selector(n) == at.Sel }); n != nil {
		return n.ID, true
	}
	return 0, false
}

func (FakeModel) Context(st *recording.State, at recording.NoteAt) (Context, error) {
	n := st.Node(at.Node)
	if n == nil {
		return Context{}, fmt.Errorf("notes: node %d is gone", at.Node)
	}
	land := n
	if p := n.ParentElement(); p != nil {
		land = p
	}
	c := Context{Node: n.ID, Landmark: land.ID, Text: trim(land.RenderedText())}
	if b, ok := st.Boxes[land.ID]; ok {
		c.Box = &b
	}
	return c, nil
}

// NoteCrops are the real model's: the rule is simple enough to share.
func (m FakeModel) NoteCrops(st *recording.State, ns []*recording.Note) ([]recording.Box, error) {
	return noteCrops(st, ns, marginOr(m.Margin), m.Resolve)
}

func (FakeModel) Live(st *recording.State) []*recording.NoteView { return liveNotes(st) }

// liveNotes are the notes on the current document: those made since it
// began (State.DocSeg), so a resume keeps them and a navigation clears them.
func liveNotes(st *recording.State) []*recording.NoteView {
	var out []*recording.NoteView
	for _, n := range st.Notes {
		if n.Seg >= st.DocSeg {
			out = append(out, n)
		}
	}
	return out
}

// selector: #id when the element has one, else a child-index path from the
// document.
func selector(n *recording.DOMNode) string {
	if id := n.A["id"]; id != "" {
		return "#" + id
	}
	var parts []string
	for e := n; e != nil; e = e.ParentElement() {
		i := 1
		if e.Parent != nil {
			for _, k := range e.Parent.Kids {
				if k == e {
					break
				}
				if k.Y == recording.YEl {
					i++
				}
			}
		}
		parts = append([]string{fmt.Sprintf("%s:nth-child(%d)", e.N, i)}, parts...)
	}
	return strings.Join(parts, " > ")
}

func find(n *recording.DOMNode, pred func(*recording.DOMNode) bool) *recording.DOMNode {
	if pred(n) {
		return n
	}
	for _, k := range n.Kids {
		if f := find(k, pred); f != nil {
			return f
		}
	}
	return nil
}

func trim(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > textBudget {
		return string(r[:textBudget])
	}
	return s
}
