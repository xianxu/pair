package query

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/xianxu/pair/cmd/internal/webcast/notes"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// SelectorGrammar is what Select understands, for its errors and --help.
const SelectorGrammar = "@ID (a recording node id), a note's selector as written, or CSS: tag, #id, .class, [attr], " +
	"[attr=v], [attr*=v], :nth-child(n), :nth-of-type(n), joined by a space (descendant) or > (child) within one " +
	"document, and >>> to go into a shadow root, frame or template"

// Select finds the elements a selector matches in the state at its moment,
// in document order. A note's selector is matched by the notes model's own
// rules (notes.Match), so it names what the note names; failing that, it is
// CSS scoped as CSS is: a selector starts in the top document, a space and
// > stay within one tree, and >>> steps from a host into its shadow root,
// frame document or template content. A selector it can't parse is an
// error naming the grammar, never a silent mismatch.
func Select(st *recording.State, sel string) ([]*recording.DOMNode, error) {
	sel = strings.TrimSpace(sel)
	if id, ok := strings.CutPrefix(sel, "@"); ok {
		n, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("query: %q is not a node id; selectors are %s", sel, SelectorGrammar)
		}
		if node := st.Node(recording.NodeID(n)); node != nil {
			return []*recording.DOMNode{node}, nil
		}
		return nil, nil
	}
	if st.Top == nil {
		return nil, nil
	}
	// An anchor's reading is narrower than CSS's (a leading tag step is its
	// tree's top level, not anywhere), so its matches are among CSS's; when
	// it names nothing, the text was meant as CSS.
	if found, ok := notes.Match(st, sel); ok && len(found) > 0 {
		return found, nil
	}
	steps, err := parseSelector(sel)
	if err != nil {
		return nil, err
	}
	var out []*recording.DOMNode
	walkAll(st.Top, func(n *recording.DOMNode) {
		if n.Y == recording.YEl && matches(st.Top, n, steps, len(steps)-1) {
			out = append(out, n)
		}
	})
	return out, nil
}

// walkAll visits n and everything under it in document order: children,
// then a shadow root, a template's content and an iframe's document.
func walkAll(n *recording.DOMNode, visit func(*recording.DOMNode)) {
	visit(n)
	for _, sub := range []*recording.DOMNode{n.Shadow, n.Content, n.Frame} {
		if sub != nil {
			walkAll(sub, visit)
		}
	}
	for _, k := range n.Kids {
		walkAll(k, visit)
	}
}

// step is one compound selector and the combinator joining it to the step
// before it: ' ', '>' or deep ('D', >>>); unused on the first.
type step struct {
	comb  byte
	tag   string
	conds []func(*recording.DOMNode) bool
}

// deep is the >>> combinator's mark in step.comb.
const deep = 'D'

// matches reports whether n matches steps[:i+1], read right to left with
// each combinator scoped to a tree; the first step's tree is top's.
func matches(top, n *recording.DOMNode, steps []step, i int) bool {
	if !steps[i].match(n) {
		return false
	}
	if i == 0 {
		return treeOf(n) == top
	}
	switch steps[i].comb {
	case '>':
		p := treeParent(n)
		return p != nil && matches(top, p, steps, i-1)
	case deep:
		h := treeOf(n).Host
		return h != nil && matches(top, h, steps, i-1)
	}
	for p := treeParent(n); p != nil; p = treeParent(p) {
		if matches(top, p, steps, i-1) {
			return true
		}
	}
	return false
}

// treeParent is n's parent element within its own tree, or nil.
func treeParent(n *recording.DOMNode) *recording.DOMNode {
	if n.Parent != nil && n.Parent.Y == recording.YEl {
		return n.Parent
	}
	return nil
}

// treeOf is the parentless node n hangs in: a document, a shadow root or a
// template's content. Its Host leads out of it.
func treeOf(n *recording.DOMNode) *recording.DOMNode {
	for n.Parent != nil {
		n = n.Parent
	}
	return n
}

func (s step) match(n *recording.DOMNode) bool {
	if s.tag != "" && s.tag != "*" && !strings.EqualFold(n.N, s.tag) {
		return false
	}
	for _, c := range s.conds {
		if !c(n) {
			return false
		}
	}
	return true
}

func parseSelector(sel string) ([]step, error) {
	bad := func(why string) ([]step, error) {
		return nil, fmt.Errorf("query: can't read selector %q (%s); selectors are %s", sel, why, SelectorGrammar)
	}
	if sel == "" {
		return bad("empty")
	}
	var steps []step
	cur := step{comb: ' '}
	empty := true
	pending := byte(0) // a combinator seen since the last compound
	finish := func() {
		if !empty {
			steps = append(steps, cur)
		}
		cur, empty = step{comb: ' '}, true
	}
	for i := 0; i < len(sel); {
		c := sel[i]
		switch {
		case strings.HasPrefix(sel[i:], ">>>"):
			// Into the host's shadow root, frame or template: what follows
			// is anywhere in that one tree.
			if !empty {
				finish()
			}
			if pending == '>' || pending == deep || len(steps) == 0 {
				return bad("a >>> with nothing before it")
			}
			pending = deep
			i += 3
			continue
		case c == ' ' || c == '>':
			if !empty {
				finish()
			}
			if c == '>' {
				if pending == '>' || pending == deep || len(steps) == 0 {
					return bad("a > with nothing before it")
				}
				pending = '>'
			} else if pending == 0 && len(steps) > 0 {
				pending = ' '
			}
			i++
		case c == '#' || c == '.':
			name, n := ident(sel[i+1:])
			if name == "" {
				return bad(fmt.Sprintf("a %c with no name", c))
			}
			if c == '#' {
				cur.conds = append(cur.conds, func(e *recording.DOMNode) bool { return e.A["id"] == name })
			} else {
				cur.conds = append(cur.conds, func(e *recording.DOMNode) bool { return hasClass(e.A["class"], name) })
			}
			i += 1 + n
		case c == '[':
			end := strings.IndexByte(sel[i:], ']')
			if end < 0 {
				return bad("an unclosed [")
			}
			cond, err := attrCond(sel[i+1 : i+end])
			if err != nil {
				return bad(err.Error())
			}
			cur.conds = append(cur.conds, cond)
			i += end + 1
		case c == ':':
			rest := sel[i:]
			pseudo := ""
			for _, p := range []string{":nth-child(", ":nth-of-type("} {
				if strings.HasPrefix(rest, p) {
					pseudo = p
				}
			}
			if pseudo == "" {
				return bad("only :nth-child(n) and :nth-of-type(n) of the pseudo-classes")
			}
			end := strings.IndexByte(rest, ')')
			k, err := strconv.Atoi(strings.TrimSpace(rest[len(pseudo):max(end, len(pseudo))]))
			if end < 0 || err != nil || k < 1 {
				return bad(pseudo + "n) takes a positive integer")
			}
			ofType := pseudo == ":nth-of-type("
			cur.conds = append(cur.conds, func(e *recording.DOMNode) bool { return childIndex(e, ofType) == k })
			i += end + 1
		default:
			name, n := ident(sel[i:])
			if n == 0 && c == '*' {
				name, n = "*", 1
			}
			if n == 0 || !empty {
				return bad(fmt.Sprintf("unexpected %q", c))
			}
			cur.tag = name
			i += n
		}
		if c != ' ' && c != '>' {
			if empty && pending != 0 {
				cur.comb = pending
			}
			empty, pending = false, 0
		}
	}
	finish()
	if pending == '>' || pending == deep {
		return bad("a combinator with nothing after it")
	}
	return steps, nil
}

func ident(s string) (string, int) {
	n := 0
	for n < len(s) {
		c := s[n]
		if c == '-' || c == '_' || c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= 0x80 {
			n++
			continue
		}
		break
	}
	return s[:n], n
}

func attrCond(body string) (func(*recording.DOMNode) bool, error) {
	op := "="
	i := strings.Index(body, "*=")
	if i >= 0 {
		op = "*="
	} else {
		i = strings.IndexByte(body, '=')
	}
	if i < 0 {
		name := strings.TrimSpace(body)
		if name == "" {
			return nil, fmt.Errorf("an empty []")
		}
		return func(e *recording.DOMNode) bool { _, ok := e.A[name]; return ok }, nil
	}
	name := strings.TrimSpace(body[:i])
	if name == "" || strings.ContainsAny(name, "~|^$") {
		return nil, fmt.Errorf("only [attr], [attr=v] and [attr*=v]")
	}
	v := strings.TrimSpace(body[i+len(op):])
	if len(v) >= 2 && (v[0] == '"' || v[0] == '\'') && v[len(v)-1] == v[0] {
		v = v[1 : len(v)-1]
	}
	if op == "*=" {
		return func(e *recording.DOMNode) bool { a, ok := e.A[name]; return ok && strings.Contains(a, v) }, nil
	}
	return func(e *recording.DOMNode) bool { a, ok := e.A[name]; return ok && a == v }, nil
}

func hasClass(list, name string) bool {
	for _, c := range strings.Fields(list) {
		if c == name {
			return true
		}
	}
	return false
}

// childIndex is an element's 1-based position among its parent's element
// children, or among those of its own tag when ofType (0 when it has no
// parent node).
func childIndex(e *recording.DOMNode, ofType bool) int {
	if e.Parent == nil {
		return 0
	}
	i := 0
	for _, k := range e.Parent.Kids {
		if k.Y == recording.YEl && (!ofType || k.N == e.N) {
			i++
		}
		if k == e {
			return i
		}
	}
	return 0
}
