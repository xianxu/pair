package notes

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

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
	evidence := candidateFor(n)
	at := recording.NoteAt{Node: node, Kind: recording.AnchorElement, Path: evidence.Path, Sel: selectorFor(n), Text: evidence.Text, Tag: evidence.Tag, Classes: evidence.Classes, Stable: evidence.Stable, Route: st.URL}
	if !slices.Contains(matchSelector(st, at.Sel), n) { // never store a selector that can't find its element
		at.Sel = ""
	}
	if b, ok := st.Boxes[node]; ok {
		at.Box = &b
	}
	return at, nil
}

// Resolve keeps a connected node only while its identifying evidence agrees.
// Replacements require a complete bounded search in the original tree scope.
func (Structural) Resolve(st *recording.State, at recording.NoteAt) (recording.NodeID, bool) {
	a := anchorEvidence(at)
	if at.Route != "" && at.Route != st.URL {
		return 0, false
	}
	if n := st.Node(at.Node); n != nil && n.Y == recording.YEl {
		if ScoreAnchor(a, candidateFor(n)) >= 0 {
			return n.ID, true
		}
		return 0, false
	}
	// Count every node, including nested trees, before doing selector/path work.
	// This also bounds the selector grammar's tree walks below.
	nodes := make([]*recording.DOMNode, 0)
	stack := []*recording.DOMNode{st.Top}
	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if n == nil {
			continue
		}
		if len(nodes) >= MaxAnchorCandidates {
			return 0, false
		}
		nodes = append(nodes, n)
		stack = append(stack, n.Kids...)
		stack = append(stack, n.Shadow, n.Frame, n.Content)
	}
	roots := map[*recording.DOMNode]bool{st.Top: true}
	if i := strings.LastIndex(at.Sel, deepSep); i >= 0 {
		roots = map[*recording.DOMNode]bool{}
		hosts := matchSelector(st, at.Sel[:i])
		// An ambiguous host is an ambiguous scope, even if only one subtree matches.
		if len(hosts) != 1 {
			return 0, false
		}
		for _, root := range []*recording.DOMNode{hosts[0].Shadow, hosts[0].Frame, hosts[0].Content} {
			if root != nil && !insideFrame(root) {
				roots[root] = true
			}
		}
	}
	candidates := make([]AnchorCandidate, 0)
	for _, n := range nodes {
		if n.Y == recording.YEl && roots[treeRoot(n)] {
			candidates = append(candidates, candidateFor(n))
		}
	}
	winner, ok := ChooseAnchor(a, candidates, false, st.URL)
	if !ok {
		return 0, false
	}
	id, err := strconv.ParseUint(winner.Key, 10, 64)
	return recording.NodeID(id), err == nil
}

// A frame's host selector survives navigation, but its document identity does
// not. Without frozen frame-route evidence replacements inside it must hide,
// including shadow/template trees nested within that frame.
func insideFrame(n *recording.DOMNode) bool {
	for n != nil {
		root := treeRoot(n)
		if root.Y == recording.YDoc && root.Host != nil {
			return true
		}
		n = root.Host
	}
	return false
}

func candidateFor(n *recording.DOMNode) AnchorCandidate {
	stable := map[string]string{}
	if id := n.A["id"]; len(id) <= 200 && identRe.MatchString(id) && !generatedRe.MatchString(id) {
		stable["id"] = id
	}
	for _, key := range testAttrs {
		if v := n.A[key]; v != "" && len(v) <= 200 {
			stable[key] = v
		}
	}
	classes := []string{}
	for _, c := range strings.Fields(n.A["class"]) {
		if len(c) <= 80 && len(classes) < 16 {
			classes = append(classes, c)
		}
	}
	return AnchorCandidate{Key: strconv.FormatUint(uint64(n.ID), 10), Tag: n.N, Stable: stable, Text: trim(n.TextContent()), Path: pathOf(n), Classes: classes}
}

// Older notes can supply tag/identity through their path or final selector step.
// A bare #id never proves the missing tag, so it cannot select a replacement.
func anchorEvidence(at recording.NoteAt) AnchorCandidate {
	a := AnchorCandidate{Tag: at.Tag, Stable: at.Stable, Text: at.Text, Path: at.Path, Classes: at.Classes, Route: at.Route}
	if a.Tag == "" && len(at.Path) > 0 {
		parts := strings.FieldsFunc(at.Path[len(at.Path)-1], func(r rune) bool { return r == '#' || r == '[' })
		if len(parts) > 0 {
			a.Tag = parts[0]
		}
	}
	parts := strings.Split(at.Sel, deepSep)
	steps := strings.Split(parts[len(parts)-1], childSep)
	if m := stepRe.FindStringSubmatch(steps[len(steps)-1]); m != nil {
		if a.Tag == "" {
			a.Tag = m[4]
		}
		if len(a.Stable) == 0 {
			if m[1] != "" {
				a.Stable = map[string]string{"id": m[1]}
			} else if m[2] != "" {
				a.Stable = map[string]string{m[2]: m[3]}
			}
		}
	}
	return a
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
