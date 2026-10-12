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
