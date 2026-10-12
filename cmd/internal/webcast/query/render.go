package query

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// Budgets keep an answer inside an agent's context: a page is cut at a line
// budget with a note saying how much is left, and long values are trimmed.
const (
	DefaultPageLines = 300
	ElementLines     = 20
	attrChars        = 80
	textChars        = 200
	describeChars    = 40
)

// keptAttrs are the attributes a page's text shows; the rest (inline
// styles, event handlers, framework data) are noise to an agent reading
// structure. --json keeps them all.
var keptAttrs = map[string]bool{
	"id": true, "class": true, "role": true, "name": true, "type": true, "href": true, "src": true, "alt": true,
	"title": true, "placeholder": true, "for": true, "value": true, "data-testid": true, "disabled": true,
	"checked": true, "selected": true, "hidden": true, "open": true, "lang": true,
}

func keepAttr(name string) bool { return keptAttrs[name] || strings.HasPrefix(name, "aria-") }

// Page is n and its subtree as compact text: one line per element,
// prefixed with its node id (@N, what every selector, event and note names),
// with the form state, focus and scroll of that moment overlaid. Script and
// style bodies, a document's head and SVG internals are folded to one line.
// At most budget lines; the last says how many more there were.
func Page(st *recording.State, n *recording.DOMNode, budget int) string {
	p := pageWriter{st: st, budget: budget, root: n}
	p.node(n, 0)
	if p.over > 0 {
		p.lines = append(p.lines, fmt.Sprintf("… %d more lines; narrow it with a selector (--sel)", p.over))
	}
	return strings.Join(p.lines, "\n") + "\n"
}

type pageWriter struct {
	st     *recording.State
	root   *recording.DOMNode
	budget int
	lines  []string
	over   int
}

func (p *pageWriter) emit(depth int, s string) {
	if len(p.lines) >= p.budget {
		p.over++
		return
	}
	p.lines = append(p.lines, strings.Repeat("  ", depth)+s)
}

func (p *pageWriter) node(n *recording.DOMNode, depth int) {
	switch n.Y {
	case recording.YDoc:
		p.emit(depth, fmt.Sprintf("@%d #document %s%s", n.ID, n.URL, p.stateText(n)))
		p.kids(n, depth+1)
	case recording.YShadow:
		p.emit(depth, fmt.Sprintf("@%d #shadow-root (%s)", n.ID, n.Mode))
		p.kids(n, depth+1)
	case recording.YFrag:
		p.emit(depth, fmt.Sprintf("@%d #template-content", n.ID))
		p.kids(n, depth+1)
	case recording.YText:
		if t := squash(n.V, textChars); t != "" {
			p.emit(depth, t)
		}
	case recording.YEl:
		p.element(n, depth)
	}
}

func (p *pageWriter) kids(n *recording.DOMNode, depth int) {
	for _, k := range n.Kids {
		p.node(k, depth)
	}
}

func (p *pageWriter) element(n *recording.DOMNode, depth int) {
	line := fmt.Sprintf("@%d <%s%s>%s", n.ID, n.N, attrText(n), p.stateText(n))
	switch {
	case n.N == "script" || n.N == "style":
		if l := len(n.TextContent()); l > 0 {
			line += fmt.Sprintf(" (%d chars)", l)
		}
		p.emit(depth, line)
		return
	case n.N == "head" && n != p.root:
		if sum := headSummary(n); sum != "" {
			line += " " + sum
		}
		p.emit(depth, line)
		return
	case n.N == "svg" && n != p.root:
		p.emit(depth, line+fmt.Sprintf(" (%d elements inside)", countElements(n)-1))
		return
	}
	if t, ok := onlyText(n); ok {
		if t != "" {
			line += " " + t
		}
		p.emit(depth, line)
		return
	}
	p.emit(depth, line)
	for _, sub := range []*recording.DOMNode{n.Shadow, n.Content, n.Frame} {
		if sub != nil {
			p.node(sub, depth+1)
		}
	}
	p.kids(n, depth+1)
}

// stateText is what the observer knows of n at this moment that its
// attributes don't say: form state, focus, scroll.
func (p *pageWriter) stateText(n *recording.DOMNode) string {
	var parts []string
	if in := p.st.Inputs[n.ID]; in != nil {
		if in.V != nil {
			parts = append(parts, "value "+strconv.Quote(trim(*in.V, textChars)))
		}
		if in.Checked != nil {
			parts = append(parts, map[bool]string{true: "checked", false: "unchecked"}[*in.Checked])
		}
		if len(in.Sel) > 0 {
			parts = append(parts, fmt.Sprintf("selected %v", in.Sel))
		}
	}
	if p.st.Focus == n.ID {
		parts = append(parts, "focused")
	}
	if sc, ok := p.st.Scrolls[n.ID]; ok {
		parts = append(parts, fmt.Sprintf("scrolled %s,%s", num(sc[0]), num(sc[1])))
	}
	if len(parts) == 0 {
		return ""
	}
	return " {" + strings.Join(parts, ", ") + "}"
}

func attrText(n *recording.DOMNode) string {
	names := make([]string, 0, len(n.A))
	for k := range n.A {
		if keepAttr(k) {
			names = append(names, k)
		}
	}
	sort.Strings(names)
	var b strings.Builder
	for _, k := range names {
		v := trim(strings.Join(strings.Fields(n.A[k]), " "), attrChars)
		switch {
		case v == "":
			b.WriteString(" " + k)
		case strings.ContainsAny(v, " \"'=<>`"):
			fmt.Fprintf(&b, " %s=%q", k, v)
		default:
			fmt.Fprintf(&b, " %s=%s", k, v)
		}
	}
	return b.String()
}

// onlyText is an element's text when text is all it holds, for one line.
// Adjacent text nodes (a framework's split interpolations) read as the one
// string the DOM's textContent gives.
func onlyText(n *recording.DOMNode) (string, bool) {
	if n.Shadow != nil || n.Content != nil || n.Frame != nil {
		return "", false
	}
	var b strings.Builder
	for _, k := range n.Kids {
		switch k.Y {
		case recording.YComment:
		case recording.YText:
			b.WriteString(k.V)
		default:
			return "", false
		}
	}
	return squash(b.String(), textChars), true
}

func headSummary(n *recording.DOMNode) string {
	counts := map[string]int{}
	title := ""
	for _, k := range n.Kids {
		if k.Y != recording.YEl {
			continue
		}
		if k.N == "title" {
			title = squash(k.TextContent(), describeChars)
			continue
		}
		counts[k.N]++
	}
	tags := make([]string, 0, len(counts))
	for t := range counts {
		tags = append(tags, t)
	}
	sort.Strings(tags)
	parts := []string{}
	if title != "" {
		parts = append(parts, "title "+strconv.Quote(title))
	}
	for _, t := range tags {
		parts = append(parts, fmt.Sprintf("%d %s", counts[t], t))
	}
	if len(parts) == 0 {
		return ""
	}
	return "(" + strings.Join(parts, "; ") + ")"
}

func countElements(n *recording.DOMNode) int {
	c := 0
	walkAll(n, func(d *recording.DOMNode) {
		if d.Y == recording.YEl {
			c++
		}
	})
	return c
}

// Describe names an element in one line, as of its state: its id, tag,
// #id, up to two classes, and the start of its text. Events and notes use
// it to say what was acted on.
func Describe(n *recording.DOMNode) string {
	if n == nil {
		return "(gone)"
	}
	switch n.Y {
	case recording.YDoc:
		return fmt.Sprintf("@%d #document", n.ID)
	case recording.YShadow:
		return fmt.Sprintf("@%d #shadow-root", n.ID)
	case recording.YText, recording.YComment:
		if e := n.ParentElement(); e != nil {
			return Describe(e)
		}
		return fmt.Sprintf("@%d #%s", n.ID, n.Y)
	case recording.YEl:
	default:
		return fmt.Sprintf("@%d", n.ID)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "@%d <%s", n.ID, n.N)
	if id := n.A["id"]; id != "" {
		b.WriteString("#" + id)
	}
	for i, c := range strings.Fields(n.A["class"]) {
		if i == 2 {
			b.WriteString(".…")
			break
		}
		b.WriteString("." + c)
	}
	b.WriteString(">")
	if t := squash(spacedText(n), describeChars); t != "" && n.N != "html" && n.N != "body" {
		b.WriteString(" " + strconv.Quote(t))
	}
	return b.String()
}

// spacedText is n's text nodes joined by spaces: what a reader sees of a
// row of cells, where textContent would run "0row-0" together.
func spacedText(n *recording.DOMNode) string {
	var parts []string
	var walk func(*recording.DOMNode)
	walk = func(n *recording.DOMNode) {
		if n.Y == recording.YText {
			parts = append(parts, n.V)
		}
		if n.Y == recording.YEl && (n.N == "script" || n.N == "style") {
			return
		}
		for _, k := range n.Kids {
			walk(k)
		}
	}
	walk(n)
	return strings.Join(parts, " ")
}

// squash collapses whitespace and trims to at most max runes.
func squash(s string, max int) string { return trim(strings.Join(strings.Fields(s), " "), max) }

func trim(s string, max int) string {
	if r := []rune(s); len(r) > max {
		return string(r[:max-1]) + "…"
	}
	return s
}

func num(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }
