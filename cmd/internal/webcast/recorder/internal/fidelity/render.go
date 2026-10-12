// Package fidelity checks a recording against the real tab (project
// Principle 6): it renders the reduced state at each screenshot's time in a
// headless Chrome, takes the same low-res screenshot, and diffs the pixels
// against frames/<n>.png.
//
// A development tool for #458's fidelity runs, compared with #453's
// prototype; the product fidelity check (wave 2) and the viewer (#460) are
// separate. The render here is static HTML from the reducer: no scripts, as a
// viewer has none.
package fidelity

import (
	"fmt"
	"html"
	"sort"
	"strings"

	rec "github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// RIDAttr marks each rendered element with its recording id, so scroll
// offsets can be set after load.
const RIDAttr = "data-fidelity-rid"

// Render is the state's top document as static HTML: scripts and event
// handlers dropped, iframes inlined as srcdoc, shadow roots as declarative
// templates, owned and adopted sheets as <style>, form state as attributes.
func Render(st *rec.State) string {
	if st.Top == nil {
		return "<!doctype html><html><body></body></html>"
	}
	r := renderer{st: st}
	r.doc(st.Top)
	return r.b.String()
}

type renderer struct {
	st *rec.State
	b  strings.Builder
}

func (r *renderer) doc(d *rec.DOMNode) {
	for _, k := range d.Kids {
		if k.Y == rec.YDoctype {
			fmt.Fprintf(&r.b, "<!doctype %s>", k.N)
			continue
		}
		r.node(k, d)
	}
}

// voids have no end tag.
var voids = map[string]bool{"area": true, "base": true, "br": true, "col": true, "embed": true, "hr": true, "img": true,
	"input": true, "link": true, "meta": true, "source": true, "track": true, "wbr": true}

func (r *renderer) node(n *rec.DOMNode, doc *rec.DOMNode) {
	switch n.Y {
	case rec.YText:
		if p := n.Parent; p != nil && (p.N == "style" || p.N == "script") {
			r.b.WriteString(strings.ReplaceAll(n.V, "</", "<\\/"))
			return
		}
		r.b.WriteString(html.EscapeString(n.V))
		return
	case rec.YComment:
		return
	case rec.YEl:
	default:
		return
	}
	switch n.N {
	case "script", "noscript":
		return // the real tab ran scripts; a viewer runs none
	}
	tag := n.N
	sheet, owned := r.ownedSheet(n.ID)
	if tag == "link" && owned {
		tag = "style"
	}
	fmt.Fprintf(&r.b, "<%s %s=\"%d\"", tag, RIDAttr, n.ID)
	in := r.st.Inputs[n.ID]
	attrs := make([]string, 0, len(n.A))
	for k := range n.A {
		attrs = append(attrs, k)
	}
	sort.Strings(attrs)
	for _, k := range attrs {
		v := n.A[k]
		switch {
		case strings.HasPrefix(k, "on"), k == "integrity", k == "nonce", k == RIDAttr:
			continue
		case n.N == "iframe" && (k == "src" || k == "srcdoc") && n.Frame != nil:
			continue
		case tag == "style" && n.N == "link":
			continue
		case in != nil && (k == "value" || k == "checked") && n.N == "input":
			continue // the control's state wins
		}
		fmt.Fprintf(&r.b, " %s=\"%s\"", k, html.EscapeString(v))
	}
	if in != nil && n.N == "input" {
		if in.V != nil {
			fmt.Fprintf(&r.b, " value=\"%s\"", html.EscapeString(*in.V))
		}
		if in.Secret {
			r.b.WriteString(" value=\"••••••••\"")
		}
		if in.Checked != nil && *in.Checked {
			r.b.WriteString(" checked")
		}
	}
	if n.N == "iframe" && n.Frame != nil {
		fr := renderer{st: r.st}
		if n.Frame.URL != "" {
			fmt.Fprintf(&fr.b, "<base href=\"%s\">", html.EscapeString(n.Frame.URL))
		}
		fr.doc(n.Frame)
		fmt.Fprintf(&r.b, " srcdoc=\"%s\"", html.EscapeString(fr.b.String()))
	}
	r.b.WriteString(">")
	if voids[n.N] && !(tag == "style") {
		return
	}
	if owned {
		r.b.WriteString(strings.ReplaceAll(sheet, "</", "<\\/"))
		fmt.Fprintf(&r.b, "</%s>", tag)
		return
	}
	if n.Shadow != nil {
		fmt.Fprintf(&r.b, "<template shadowrootmode=\"%s\">", n.Shadow.Mode)
		for _, k := range n.Shadow.Kids {
			r.node(k, doc)
		}
		r.adopted(n.Shadow.ID)
		r.b.WriteString("</template>")
	}
	if n.Content != nil {
		r.b.WriteString("<template>")
		for _, k := range n.Content.Kids {
			r.node(k, doc)
		}
		r.b.WriteString("</template>")
	}
	if n.N == "textarea" && in != nil && in.V != nil {
		r.b.WriteString(html.EscapeString(*in.V))
	} else {
		for i, k := range n.Kids {
			if n.N == "select" && in != nil && in.Sel != nil && k.Y == rec.YEl && k.N == "option" {
				r.option(k, doc, i, in.Sel)
				continue
			}
			r.node(k, doc)
		}
	}
	if n.N == "head" && n.Parent != nil && n.Parent.Parent == doc {
		r.adopted(doc.ID)
	}
	fmt.Fprintf(&r.b, "</%s>", tag)
}

// option renders a select's option with its selected state.
func (r *renderer) option(o, doc *rec.DOMNode, _ int, sel []int) {
	idx := 0
	for _, k := range o.Parent.Kids {
		if k == o {
			break
		}
		if k.Y == rec.YEl && k.N == "option" {
			idx++
		}
	}
	selected := false
	for _, s := range sel {
		selected = selected || s == idx
	}
	cp := *o
	cp.A = map[string]string{}
	for k, v := range o.A {
		if k != "selected" {
			cp.A[k] = v
		}
	}
	if selected {
		cp.A["selected"] = ""
	}
	r.node(&cp, doc)
}

func (r *renderer) ownedSheet(owner rec.NodeID) (string, bool) {
	var newest rec.NodeID
	text, ok := "", false
	for id, sh := range r.st.Sheets {
		if sh.Owner == owner && id > newest {
			newest, text, ok = id, sh.Text, true
		}
	}
	return text, ok
}

func (r *renderer) adopted(id rec.NodeID) {
	for _, sid := range r.st.Adopted[id] {
		if sh := r.st.Sheets[sid]; sh != nil {
			fmt.Fprintf(&r.b, "<style>%s</style>", strings.ReplaceAll(sh.Text, "</", "<\\/"))
		}
	}
}
