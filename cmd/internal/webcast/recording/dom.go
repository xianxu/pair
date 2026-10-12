package recording

import "strings"

// TextContent is the node's text, as the DOM's textContent: every text
// descendant in order (shadow roots and frames excluded).
func (n *DOMNode) TextContent() string {
	var b strings.Builder
	var walk func(*DOMNode)
	walk = func(n *DOMNode) {
		if n.Y == YText {
			b.WriteString(n.V)
		}
		for _, k := range n.Kids {
			walk(k)
		}
	}
	walk(n)
	return b.String()
}

// Element is the nearest element at or above n (through shadow hosts and
// frame owners), or nil.
func (n *DOMNode) Element() *DOMNode {
	for ; n != nil; n = up(n) {
		if n.Y == YEl {
			return n
		}
	}
	return nil
}

// ParentElement is the nearest element strictly above n, or nil.
func (n *DOMNode) ParentElement() *DOMNode {
	if p := up(n); p != nil {
		return p.Element()
	}
	return nil
}

// RenderedText is the text a person reads in n, whitespace-collapsed: like
// innerText, it skips what isn't rendered (script, style, … and elements
// the recording marks hidden) and separates what renders apart (blocks,
// table cells, line breaks, flex and grid items) with a space, where
// TextContent runs them together. A select lists all its options: its
// live choice isn't in the DOM. Shadow roots and frames are excluded, as
// in TextContent.
func (n *DOMNode) RenderedText() string {
	var b strings.Builder
	var walk func(n *DOMNode, apart, visible bool)
	walk = func(n *DOMNode, apart, visible bool) {
		switch {
		case n.Y == YText:
			if visible {
				b.WriteString(n.V)
			}
			return
		case n.Y == YEl && unrendered(n):
			return
		}
		if v, ok := visibility(n); ok { // inherited, and a descendant can set it back
			visible = v
		}
		apart = apart || n.Y == YEl && blockTags[n.N]
		if apart {
			b.WriteByte(' ')
		}
		items := n.Y == YEl && laidOutApart(n.A["class"])
		for _, k := range n.Kids {
			walk(k, items, visible)
		}
		if apart {
			b.WriteByte(' ')
		}
	}
	walk(n, false, true)
	return strings.Join(strings.Fields(b.String()), " ")
}

// visibility is an element's own inline visibility, if it sets one.
func visibility(e *DOMNode) (visible, set bool) {
	if e.Y != YEl {
		return false, false
	}
	switch style := inlineStyle(e); {
	case strings.Contains(style, "visibility:hidden"), strings.Contains(style, "visibility:collapse"):
		return false, true
	case strings.Contains(style, "visibility:visible"):
		return true, true
	}
	return false, false
}

func inlineStyle(e *DOMNode) string {
	return strings.ToLower(strings.Join(strings.Fields(e.A["style"]), ""))
}

var (
	// unrenderedTags never show their content.
	unrenderedTags = setOf("script", "style", "template", "noscript", "head", "title", "meta", "link", "base")
	// blockTags render on their own line or in their own cell (the HTML
	// spec's default display of block, list-item, table-*, plus br).
	blockTags = setOf("address", "article", "aside", "blockquote", "body", "br", "caption", "dd", "details",
		"dialog", "div", "dl", "dt", "fieldset", "figcaption", "figure", "footer", "form", "h1", "h2", "h3",
		"h4", "h5", "h6", "header", "hgroup", "hr", "html", "legend", "li", "main", "menu", "nav", "ol",
		"option", "p", "pre", "search", "section", "summary", "table", "tbody", "td", "tfoot", "th", "thead",
		"tr", "ul")
)

// unrendered: a non-rendered tag, the hidden attribute, an inline
// display:none, or Tailwind's hidden class with no variant (md:block, …) that could show it again.
func unrendered(e *DOMNode) bool {
	if unrenderedTags[e.N] {
		return true
	}
	if _, ok := e.A["hidden"]; ok {
		return true
	}
	if strings.Contains(inlineStyle(e), "display:none") {
		return true
	}
	hidden, shown := false, false
	for _, tok := range strings.Fields(e.A["class"]) {
		i := strings.LastIndexByte(tok, ':')
		hidden = hidden || tok == "hidden"
		shown = shown || i >= 0 && displayClasses[tok[i+1:]]
	}
	return hidden && !shown
}

var displayClasses = setOf("block", "inline-block", "inline", "flex", "inline-flex", "grid", "inline-grid",
	"table", "table-row", "table-cell", "contents", "list-item", "flow-root")

// laidOutApart: a flex or grid container (Tailwind's class), whose
// children are blockified, so each renders apart from its neighbours.
func laidOutApart(class string) bool {
	for _, tok := range strings.Fields(class) {
		switch tok[strings.LastIndexByte(tok, ':')+1:] {
		case "flex", "inline-flex", "grid", "inline-grid":
			return true
		}
	}
	return false
}

func setOf(xs ...string) map[string]bool {
	m := make(map[string]bool, len(xs))
	for _, x := range xs {
		m[x] = true
	}
	return m
}
