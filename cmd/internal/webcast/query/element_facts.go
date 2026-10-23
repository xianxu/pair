package query

import (
	"cmp"
	"math"
	"slices"
	"strings"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

const (
	ElementFactsNodeLimit     = 20000
	ElementFactsCountLimit    = 64
	ElementFactsByteLimit     = 64 << 10
	ElementFactsAncestorLimit = 32
)

// ElementFact is a literal historical declaration, never a live selector.
type ElementFact struct {
	Node        recording.NodeID  `json:"node"`
	Parent      recording.NodeID  `json:"parent,omitempty"`
	Tag         string            `json:"tag,omitempty"`
	ID          string            `json:"id,omitempty"`
	Data        map[string]string `json:"data,omitempty"`
	Unavailable string            `json:"unavailable,omitempty"`
	Truncated   bool              `json:"truncated,omitempty"`
}
type DeclaredElementFacts struct {
	Anchors        []ElementFact `json:"anchors"`
	Ancestors      []ElementFact `json:"ancestors"`
	Intersections  []ElementFact `json:"intersections"`
	Truncated      bool          `json:"truncated"`
	MissingLayout  bool          `json:"missing_layout"`
	InspectedNodes int           `json:"inspected_nodes"`
	Bytes          int           `json:"bytes"`
}
type factNode struct {
	fact ElementFact
	box  *recording.Box
}
type elementFactSource struct {
	nodes     map[recording.NodeID]factNode
	inspected int
	truncated bool
	// Attribute inspection is linear in input attributes. Retention is at most
	// 20,000 node headers and 16 data-* string pairs plus one ID per node; string
	// values are immutable and shared, while every retained map is privately owned.
	attributes, declarations int
}

// DeclaredElementFactsAt owns its bounded values. Call at the evidence's source
// moment; it never resolves a historical handle against a different document.
func DeclaredElementFactsAt(st *recording.State, anchors []recording.NodeID, ink *recording.Box) DeclaredElementFacts {
	return captureElementFacts(st).facts(anchors, ink)
}
func captureElementFacts(st *recording.State) elementFactSource {
	out := elementFactSource{nodes: map[recording.NodeID]factNode{}}
	if st == nil || st.Top == nil {
		return out
	}
	stack := []*recording.DOMNode{st.Top}
	for len(stack) > 0 {
		if out.inspected >= ElementFactsNodeLimit {
			out.truncated = true
			break
		}
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		out.inspected++
		// Reverse push preserves the recorded child order without sorting IDs.
		for i := len(n.Kids) - 1; i >= 0; i-- {
			stack = append(stack, n.Kids[i])
		}
		for _, child := range []*recording.DOMNode{n.Shadow, n.Content, n.Frame} {
			if child != nil {
				stack = append(stack, child)
			}
		}
		if n.Y != recording.YEl {
			continue
		}
		f := ElementFact{Node: n.ID, Tag: n.N}
		if len(f.Tag) > 256 {
			f.Tag = ""
			f.Truncated = true
		}
		if p := n.ParentElement(); p != nil {
			f.Parent = p.ID
		}
		if id, ok := n.A["id"]; ok {
			if len(id) <= 1024 {
				f.ID = id
				out.declarations++
			} else {
				f.Truncated = true
			}
		}
		// Keep only the lexicographically first 16 valid data attributes.
		// A bounded insertion array avoids an input-sized key allocation/sort,
		// and is deterministic despite Go map iteration order.
		type attribute struct{ name, value string }
		attrs := make([]attribute, 0, min(16, len(n.A)))
		for name, value := range n.A {
			out.attributes++
			if !strings.HasPrefix(name, "data-") {
				continue
			}
			if len(name) > 256 || len(value) > 1024 {
				f.Truncated = true
				continue
			}
			pos := 0
			for pos < len(attrs) && attrs[pos].name < name {
				pos++
			}
			if len(attrs) == 16 {
				f.Truncated = true
				if pos == 16 {
					continue
				}
				attrs = attrs[:15]
			}
			attrs = append(attrs, attribute{})
			copy(attrs[pos+1:], attrs[pos:])
			attrs[pos] = attribute{name, value}
		}
		if len(attrs) > 0 {
			f.Data = make(map[string]string, len(attrs))
			for _, a := range attrs {
				f.Data[a.name] = a.value
			}
			out.declarations += len(attrs)
		}

		v := factNode{fact: f}
		if box, ok := st.Boxes[n.ID]; ok {
			b := box
			v.box = &b
		}
		out.nodes[n.ID] = v
	}
	return out
}
func factBytes(f ElementFact) int {
	n := len(f.Tag) + len(f.ID) + len(f.Unavailable) + 32
	for k, v := range f.Data {
		n += len(k) + len(v)
	}
	return n
}
func (s elementFactSource) facts(anchors []recording.NodeID, ink *recording.Box) DeclaredElementFacts {
	out := DeclaredElementFacts{Anchors: []ElementFact{}, Ancestors: []ElementFact{}, Intersections: []ElementFact{}, Truncated: s.truncated, InspectedNodes: s.inspected}
	// Node zero is the absence of an anchor (region-only notes), never
	// historical identity. Legal sketches contain at most 64 distinct anchors.
	unique := make([]recording.NodeID, 0, min(len(anchors), ElementFactsCountLimit))
	anchorSeen := map[recording.NodeID]bool{}
	for _, id := range anchors {
		if id <= 0 || anchorSeen[id] {
			continue
		}
		anchorSeen[id] = true
		if len(unique) == ElementFactsCountLimit {
			out.Truncated = true
			continue
		}
		unique = append(unique, id)
	}
	anchors = unique
	missing := func(id recording.NodeID) ElementFact {
		return ElementFact{Node: id, Unavailable: "element absent or outside source inspection bound"}
	}
	fallback := func(id recording.NodeID) ElementFact {
		return ElementFact{Node: id, Unavailable: "declared facts byte limit", Truncated: true}
	}
	minimum := func(id recording.NodeID) int {
		if _, ok := s.nodes[id]; !ok {
			return factBytes(missing(id))
		}
		return factBytes(fallback(id))
	}
	reserved := 0
	for _, id := range anchors {
		reserved += minimum(id)
	}
	seen := map[recording.NodeID]bool{}
	retained := 0
	take := func(id recording.NodeID, mandatory bool) (ElementFact, bool) {
		if seen[id] {
			return ElementFact{}, false
		}
		seen[id] = true
		v, ok := s.nodes[id]
		f := v.fact
		if !ok {
			f = missing(id)
		}
		if !mandatory && retained >= ElementFactsCountLimit {
			out.Truncated = true
			return ElementFact{}, false
		}
		size := factBytes(f)
		if out.Bytes+size+reserved > ElementFactsByteLimit {
			out.Truncated = true
			if !mandatory {
				return ElementFact{}, false
			}
			f = fallback(id)
			size = factBytes(f)
		}
		out.Truncated = out.Truncated || f.Truncated
		out.Bytes += size
		retained++
		return f, true
	}
	for _, id := range anchors {
		reserved -= minimum(id)
		if f, ok := take(id, true); ok {
			out.Anchors = append(out.Anchors, f)
		}
	}

	for _, id := range anchors {
		parent := s.nodes[id].fact.Parent
		for depth := 0; parent != 0; depth++ {
			if depth >= ElementFactsAncestorLimit {
				out.Truncated = true
				break
			}
			v, exists := s.nodes[parent]
			if !exists {
				out.Truncated = true
				break
			}
			if f, ok := take(parent, false); ok {
				out.Ancestors = append(out.Ancestors, f)
			}
			if v.fact.ID != "" {
				break
			}
			parent = v.fact.Parent
		}
	}
	if ink == nil {
		return out
	}
	type candidate struct {
		id       recording.NodeID
		distance float64
	}
	candidates := []candidate{}
	cx, cy := ink[0]+ink[2]/2, ink[1]+ink[3]/2
	for id, v := range s.nodes {
		if v.box == nil {
			out.MissingLayout = true
			continue
		}
		b := *v.box
		if b[2] <= 0 || b[3] <= 0 || b[0] > ink[0]+ink[2] || b[1] > ink[1]+ink[3] || b[0]+b[2] < ink[0] || b[1]+b[3] < ink[1] {
			continue
		}
		distance := math.Hypot(b[0]+b[2]/2-cx, b[1]+b[3]/2-cy)
		if math.IsNaN(distance) || math.IsInf(distance, 0) {
			out.Truncated = true
			continue
		}
		candidates = append(candidates, candidate{id, distance})
	}
	slices.SortFunc(candidates, func(a, b candidate) int {
		if n := cmp.Compare(a.distance, b.distance); n != 0 {
			return n
		}
		return cmp.Compare(a.id, b.id)
	})
	for _, c := range candidates {
		if f, ok := take(c.id, false); ok {
			out.Intersections = append(out.Intersections, f)
		}
	}
	return out
}
