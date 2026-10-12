package notes

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// A stable selector is a small CSS subset, made and matched in this file
// only, so the two can't drift:
//
//	#id                  a stable id, unique in its tree
//	[data-testid="v"]    a test attribute (also data-test, data-cy), unique
//	tag:nth-of-type(k)   otherwise, one step of a path
//
// Steps join with " > " from the nearest ancestor that has a stable id or
// test attribute, else from the tree's root element. A node inside a shadow
// root, an iframe's document or a template's content is its host's selector, " >>> ", then the path
// inside (the deep combinator is ours, not CSS). Anything else matches
// nothing.

const (
	childSep = " > "
	deepSep  = " >>> "
)

var (
	testAttrs   = []string{"data-testid", "data-test", "data-cy"}
	identRe     = regexp.MustCompile(`^[A-Za-z][\w-]*$`)
	generatedRe = regexp.MustCompile(`\d{4,}`)
	stepRe      = regexp.MustCompile(`^(?:#([A-Za-z][\w-]*)|\[([\w-]+)="([^"]*)"\]|([\w-]+):nth-of-type\((\d+)\))$`)
)

// selectorFor is n's stable selector. n is a live element.
func selectorFor(n *recording.DOMNode) string {
	var steps []string
	e := n
	for {
		if s := uniqueStep(treeRoot(e), e); s != "" {
			steps = append([]string{s}, steps...)
			break
		}
		steps = append([]string{fmt.Sprintf("%s:nth-of-type(%d)", e.N, typeIndex(e))}, steps...)
		if e.Parent == nil || e.Parent.Y != recording.YEl {
			break
		}
		e = e.Parent
	}
	sel := strings.Join(steps, childSep)
	if root := treeRoot(n); root.Host != nil { // a shadow root, template content or frame document
		return selectorFor(root.Host) + deepSep + sel
	}
	return sel
}

// uniqueStep is e's id or test-attribute step when that names e alone in
// its tree, else "".
func uniqueStep(root, e *recording.DOMNode) string {
	if id := e.A["id"]; identRe.MatchString(id) && !generatedRe.MatchString(id) {
		if s := "#" + id; len(matchStep(root, s)) == 1 {
			return s
		}
	}
	for _, a := range testAttrs {
		if v, ok := e.A[a]; ok && !strings.ContainsAny(v, `">`) { // a quote or a combinator can't round-trip
			if s := fmt.Sprintf(`[%s="%s"]`, a, v); len(matchStep(root, s)) == 1 {
				return s
			}
		}
	}
	return ""
}

// Match is what an anchor selector names in st, by this file's rules, and
// whether sel is in the anchor grammar at all. Other readers of a note's
// selector (the agent queries, #462) match through it, so the grammar keeps
// one definition.
func Match(st *recording.State, sel string) ([]*recording.DOMNode, bool) {
	if !isAnchor(sel) {
		return nil, false
	}
	return matchSelector(st, sel), true
}

// isAnchor reports whether sel is made of this file's steps and separators.
func isAnchor(sel string) bool {
	if sel == "" {
		return false
	}
	for _, part := range strings.Split(sel, deepSep) {
		for _, step := range strings.Split(part, childSep) {
			if !stepRe.MatchString(step) {
				return false
			}
		}
	}
	return true
}

// matchSelector is every element of st's top document that sel names, in
// document order. A selector outside the subset matches nothing.
func matchSelector(st *recording.State, sel string) []*recording.DOMNode {
	if st.Top == nil || sel == "" {
		return nil
	}
	scopes := []*recording.DOMNode{st.Top}
	for i, part := range strings.Split(sel, deepSep) {
		if i > 0 { // step into each match's shadow root, frame document or template content
			var next []*recording.DOMNode
			for _, h := range scopes {
				for _, sub := range []*recording.DOMNode{h.Shadow, h.Frame, h.Content} {
					if sub != nil {
						next = append(next, sub)
					}
				}
			}
			scopes = next
		}
		var found []*recording.DOMNode
		for _, root := range scopes {
			found = append(found, matchPath(root, strings.Split(part, childSep))...)
		}
		scopes = found
	}
	return scopes
}

// matchPath matches a " > " path within one tree. Its first step is an id or
// test attribute anywhere in the tree, or a tag step for the tree's root
// element; each later step is among the previous matches' children.
func matchPath(root *recording.DOMNode, steps []string) []*recording.DOMNode {
	m := stepRe.FindStringSubmatch(steps[0])
	if m == nil {
		return nil
	}
	cur := []*recording.DOMNode{root}
	if m[4] == "" {
		cur = matchStep(root, steps[0])
		steps = steps[1:]
	}
	for _, s := range steps {
		var next []*recording.DOMNode
		for _, p := range cur {
			for _, k := range p.Kids {
				if stepMatches(k, s) {
					next = append(next, k)
				}
			}
		}
		cur = next
	}
	return cur
}

// matchStep is every element of root's tree (not crossing into shadow roots,
// frames or template contents) that one step names.
func matchStep(root *recording.DOMNode, step string) []*recording.DOMNode {
	var out []*recording.DOMNode
	var walk func(*recording.DOMNode)
	walk = func(n *recording.DOMNode) {
		if stepMatches(n, step) {
			out = append(out, n)
		}
		for _, k := range n.Kids {
			walk(k)
		}
	}
	walk(root)
	return out
}

func stepMatches(n *recording.DOMNode, step string) bool {
	if n.Y != recording.YEl {
		return false
	}
	m := stepRe.FindStringSubmatch(step)
	switch {
	case m == nil:
		return false
	case m[1] != "":
		return n.A["id"] == m[1]
	case m[2] != "":
		v, ok := n.A[m[2]]
		return ok && v == m[3]
	default:
		k, _ := strconv.Atoi(m[5])
		return n.N == m[4] && typeIndex(n) == k
	}
}

// typeIndex is n's 1-based position among its siblings of the same tag.
func typeIndex(n *recording.DOMNode) int {
	if n.Parent == nil {
		return 1
	}
	i := 0
	for _, k := range n.Parent.Kids {
		if k.Y == recording.YEl && k.N == n.N {
			i++
		}
		if k == n {
			break
		}
	}
	return i
}

// treeRoot is the parentless node n hangs in: a document, a shadow root or a
// template's content.
func treeRoot(n *recording.DOMNode) *recording.DOMNode {
	for n.Parent != nil {
		n = n.Parent
	}
	return n
}
