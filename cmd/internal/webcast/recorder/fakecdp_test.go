package recorder

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"slices"
	"strings"
	"testing"

	rec "github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// fakePage is the stateful double of one CDP session's DOM (ARCH-MOCK): a
// tree that answers DOM.getDocument as Chrome does (whitespace included,
// pierce) and records the DOM events each mutation makes Chrome send. Node
// ids are per snapshot, backend ids per node for its life, as in Chrome.
type fakePage struct {
	frameID     string
	url         string
	doc         *fnode
	nextNode    int
	nextBackend int
	byNode      map[int]*fnode
	events      []fevent
}

type fnode struct {
	nodeID, backend, typ int
	local, value         string
	attrs                []string
	kids                 []*fnode
	shadow               *fnode
	shadowType           string
	parent, host         *fnode
	frame                string // an iframe element's frame id
}

type fevent struct {
	method string
	params json.RawMessage
}

func newFakePage(url string) *fakePage {
	p := &fakePage{frameID: "F1", url: url, byNode: map[int]*fnode{}}
	p.doc = p.node(9, "", "")
	html := p.el("html")
	head, body := p.el("head"), p.el("body")
	html.kids = []*fnode{head, p.txt("\n"), body}
	head.parent, body.parent = html, html
	html.kids[1].parent = html
	dt := p.node(10, "html", "")
	p.doc.kids = []*fnode{dt, html}
	dt.parent, html.parent = p.doc, p.doc
	return p
}

func (p *fakePage) node(typ int, local, value string) *fnode {
	p.nextNode++
	p.nextBackend++
	n := &fnode{nodeID: p.nextNode, backend: p.nextBackend, typ: typ, local: local, value: value}
	p.byNode[n.nodeID] = n
	return n
}

func (p *fakePage) el(local string, attrs ...string) *fnode {
	n := p.node(1, local, "")
	n.attrs = attrs
	return n
}

func (p *fakePage) txt(v string) *fnode { return p.node(3, "", v) }

func (p *fakePage) body() *fnode { return p.doc.kids[1].kids[2] }

// cdp is a node as CDP sends it, with its whole subtree.
func (p *fakePage) cdp(n *fnode) map[string]any {
	m := map[string]any{"nodeId": n.nodeID, "backendNodeId": n.backend, "nodeType": n.typ,
		"localName": n.local, "nodeName": strings.ToUpper(n.local), "nodeValue": n.value}
	if n.typ == 9 {
		m["documentURL"] = p.url
		m["frameId"] = p.frameID
		m["nodeName"] = "#document"
	}
	if n.typ == 10 {
		m["nodeName"] = n.local
	}
	if n.typ == 3 {
		m["nodeName"] = "#text"
	}
	if n.typ == 11 {
		m["shadowRootType"] = n.shadowType
		m["nodeName"] = "#document-fragment"
	}
	if n.attrs != nil {
		m["attributes"] = n.attrs
	}
	if n.frame != "" {
		m["frameId"] = n.frame
	}
	kids := []any{}
	for _, k := range n.kids {
		kids = append(kids, p.cdp(k))
	}
	m["children"] = kids
	m["childNodeCount"] = len(n.kids)
	if n.shadow != nil {
		m["shadowRoots"] = []any{p.cdp(n.shadow)}
	}
	return m
}

func raw(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

// snapshot is DOM.getDocument's root, decoded as the mirror reads it.
func (p *fakePage) snapshot() *cnode {
	var root cnode
	if err := json.Unmarshal(raw(p.cdp(p.doc)), &root); err != nil {
		panic(err)
	}
	return &root
}

func (p *fakePage) emit(method string, params any) {
	p.events = append(p.events, fevent{method, raw(params)})
}

func (p *fakePage) take() []fevent {
	ev := p.events
	p.events = nil
	return ev
}

// insert puts n (a fresh subtree, or a removed node: a move) under parent
// after prev (nil: first).
func (p *fakePage) insert(parent, prev, n *fnode) {
	at := 0
	prevID := 0
	if prev != nil {
		at = slices.Index(parent.kids, prev) + 1
		prevID = prev.nodeID
	}
	p.renumber(n) // CDP gives a re-inserted node a new nodeId
	n.parent = parent
	parent.kids = slices.Insert(parent.kids, at, n)
	p.emit("DOM.childNodeInserted", map[string]any{"parentNodeId": parent.nodeID, "previousNodeId": prevID, "node": p.cdp(n)})
}

func (p *fakePage) renumber(n *fnode) {
	delete(p.byNode, n.nodeID)
	p.nextNode++
	n.nodeID = p.nextNode
	p.byNode[n.nodeID] = n
	for _, k := range n.kids {
		p.renumber(k)
	}
	if n.shadow != nil {
		p.renumber(n.shadow)
	}
}

func (p *fakePage) remove(n *fnode) {
	par := n.parent
	par.kids = slices.DeleteFunc(par.kids, func(k *fnode) bool { return k == n })
	n.parent = nil
	p.emit("DOM.childNodeRemoved", map[string]any{"parentNodeId": par.nodeID, "nodeId": n.nodeID})
}

func (p *fakePage) setAttr(n *fnode, name, v string) {
	for i := 0; i+1 < len(n.attrs); i += 2 {
		if n.attrs[i] == name {
			n.attrs[i+1] = v
			p.emit("DOM.attributeModified", map[string]any{"nodeId": n.nodeID, "name": name, "value": v})
			return
		}
	}
	n.attrs = append(n.attrs, name, v)
	p.emit("DOM.attributeModified", map[string]any{"nodeId": n.nodeID, "name": name, "value": v})
}

func (p *fakePage) removeAttr(n *fnode, name string) {
	for i := 0; i+1 < len(n.attrs); i += 2 {
		if n.attrs[i] == name {
			n.attrs = append(n.attrs[:i], n.attrs[i+2:]...)
			p.emit("DOM.attributeRemoved", map[string]any{"nodeId": n.nodeID, "name": name})
			return
		}
	}
}

func (p *fakePage) setText(n *fnode, v string) {
	n.value = v
	p.emit("DOM.characterDataModified", map[string]any{"nodeId": n.nodeID, "characterData": v})
}

func (p *fakePage) attachShadow(host *fnode, mode string) *fnode {
	r := p.node(11, "", "")
	r.shadowType, r.host = mode, host
	host.shadow = r
	p.emit("DOM.shadowRootPushed", map[string]any{"hostId": host.nodeID, "root": p.cdp(r)})
	return r
}

// elements lists every element and shadow root under n (document order).
func (p *fakePage) containers(n *fnode, out *[]*fnode) {
	if n.typ == 1 || n.typ == 11 {
		*out = append(*out, n)
	}
	for _, k := range n.kids {
		p.containers(k, out)
	}
	if n.shadow != nil {
		p.containers(n.shadow, out)
	}
}

func (p *fakePage) all(n *fnode, out *[]*fnode) {
	*out = append(*out, n)
	for _, k := range n.kids {
		p.all(k, out)
	}
	if n.shadow != nil {
		p.all(n.shadow, out)
	}
}

func isUnder(n, root *fnode) bool {
	for ; n != nil; n = up(n) {
		if n == root {
			return true
		}
	}
	return false
}

func up(n *fnode) *fnode {
	if n.parent != nil {
		return n.parent
	}
	return n.host
}

// mutate makes one random change, as a page's scripts would.
func (p *fakePage) mutate(r *rand.Rand) {
	var cs, all []*fnode
	p.containers(p.body(), &cs)
	p.all(p.body(), &all)
	pick := func(l []*fnode) *fnode { return l[r.Intn(len(l))] }
	switch op := r.Intn(9); {
	case op <= 2: // insert a fresh subtree
		parent := pick(cs)
		var prev *fnode
		if len(parent.kids) > 0 && r.Intn(2) == 0 {
			prev = pick(parent.kids)
		}
		n := p.el([]string{"div", "span", "li", "p"}[r.Intn(4)], "class", fmt.Sprintf("c%d", r.Intn(5)))
		if r.Intn(2) == 0 {
			t := p.txt(fmt.Sprintf("t%d", r.Intn(100)))
			t.parent = n
			n.kids = append(n.kids, t)
		}
		if r.Intn(6) == 0 {
			pw := p.el("input", "type", "password", "value", "s3cret-planted")
			pw.parent = n
			n.kids = append(n.kids, pw)
		}
		p.insert(parent, prev, n)
	case op == 3: // remove
		if n := pick(all); n != p.body() && n.parent != nil {
			p.remove(n)
		}
	case op == 4: // move
		n := pick(all)
		parent := pick(cs)
		if n == p.body() || n.parent == nil || isUnder(parent, n) {
			return
		}
		p.remove(n)
		var prev *fnode
		if len(parent.kids) > 0 && r.Intn(2) == 0 {
			prev = pick(parent.kids)
		}
		p.insert(parent, prev, n)
	case op == 5: // attribute
		if n := pick(all); n.typ == 1 {
			if r.Intn(3) == 0 {
				p.removeAttr(n, "class")
			} else {
				p.setAttr(n, "class", fmt.Sprintf("k%d", r.Intn(9)))
			}
		}
	case op == 6: // text
		if n := pick(all); n.typ == 3 {
			p.setText(n, fmt.Sprintf("x%d", r.Intn(100)))
		}
	case op == 7: // password games: a value, a type flip
		if n := pick(all); n.typ == 1 && n.local == "input" {
			// The planted value only ever sits on a password input, and a
			// password never flips back to text (that shows it on screen,
			// where recording it is accepted).
			v := attrOf(n, "type")
			if r.Intn(2) == 0 {
				if strings.EqualFold(v, "password") {
					p.setAttr(n, "value", "s3cret-planted")
				} else {
					p.setAttr(n, "value", "plain")
				}
			} else {
				p.setAttr(n, "type", []string{"password", "PassWord"}[r.Intn(2)])
			}
		}
	case op == 8: // shadow root
		if n := pick(all); n.typ == 1 && n.shadow == nil && n.local == "div" {
			p.attachShadow(n, []string{"open", "closed"}[r.Intn(2)])
		}
	}
}

// sameTree compares the fake's tree with the reduced DOM: types, names, the
// recorded attributes (a password's value never), text, shadow roots,
// children in order.
func sameTree(f *fnode, d *rec.DOMNode, path string) error {
	if d == nil {
		return fmt.Errorf("%s: missing in the recording", path)
	}
	want := map[int]string{9: rec.YDoc, 10: rec.YDoctype, 1: rec.YEl, 3: rec.YText, 11: rec.YShadow}[f.typ]
	if d.Y != want {
		return fmt.Errorf("%s: type %s, want %s", path, d.Y, want)
	}
	if f.typ == 1 {
		if d.N != f.local {
			return fmt.Errorf("%s: tag %s, want %s", path, d.N, f.local)
		}
		wa := map[string]string{}
		for i := 0; i+1 < len(f.attrs); i += 2 {
			wa[f.attrs[i]] = f.attrs[i+1]
		}
		if f.local == "input" && strings.EqualFold(wa["type"], "password") {
			delete(wa, "value")
		}
		if fmt.Sprint(wa) != fmt.Sprint(nonNil(d.A)) {
			return fmt.Errorf("%s: attrs %v, want %v", path, d.A, wa)
		}
	}
	if f.typ == 3 && d.V != f.value {
		return fmt.Errorf("%s: text %q, want %q", path, d.V, f.value)
	}
	if (f.shadow == nil) != (d.Shadow == nil) {
		return fmt.Errorf("%s: shadow root presence differs", path)
	}
	if f.shadow != nil {
		if err := sameTree(f.shadow, d.Shadow, path+"/#shadow"); err != nil {
			return err
		}
	}
	if len(f.kids) != len(d.Kids) {
		return fmt.Errorf("%s: %d children, want %d", path, len(d.Kids), len(f.kids))
	}
	for i := range f.kids {
		if err := sameTree(f.kids[i], d.Kids[i], fmt.Sprintf("%s/%d", path, i)); err != nil {
			return err
		}
	}
	return nil
}

func nonNil(a map[string]string) map[string]string {
	if a == nil {
		return map[string]string{}
	}
	return a
}

// files encodes records into a recording's files, for recording.Validate.
func files(t *testing.T, recs []rec.Record) map[string][]byte {
	t.Helper()
	out := map[string][]byte{}
	for name, rs := range rec.Split(recs) {
		var b bytes.Buffer
		for _, r := range rs {
			line, err := rec.Encode(r)
			if err != nil {
				t.Fatal(err)
			}
			b.Write(line)
			b.WriteByte('\n')
		}
		out[name] = b.Bytes()
	}
	return out
}

func attrOf(n *fnode, name string) string {
	for i := 0; i+1 < len(n.attrs); i += 2 {
		if n.attrs[i] == name {
			return n.attrs[i+1]
		}
	}
	return ""
}

// byBackend finds a node by backend id.
func (p *fakePage) byBackend(b int) *fnode {
	var all []*fnode
	p.all(p.doc, &all)
	for _, n := range all {
		if n.backend == b {
			return n
		}
	}
	return nil
}
