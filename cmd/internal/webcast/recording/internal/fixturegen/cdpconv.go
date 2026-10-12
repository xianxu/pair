package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	rec "github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// The raw input: one line of the #453 prototype recorder's cdp.jsonl (mode
// B, extended for #457 with layout, click, focus, frameNavigated and asset
// bodies). Most lines are a CDP DOM event's params with a type and a
// session; the rest are the prototype's own (snapshot, sheet, rules,
// styleText, asset, assetBody, font, value, scroll, click, focus, layout).
type raw struct {
	T     int64           `json:"t"`
	S     string          `json:"s"`
	Type  string          `json:"type"`
	Line  int             `json:"-"`
	Extra json.RawMessage `json:"-"`
}

// cdpNode is CDP's DOM.Node, the fields the converter reads.
type cdpNode struct {
	NodeID          int        `json:"nodeId"`
	BackendNodeID   int        `json:"backendNodeId"`
	NodeType        int        `json:"nodeType"`
	NodeName        string     `json:"nodeName"`
	LocalName       string     `json:"localName"`
	NodeValue       string     `json:"nodeValue"`
	Attributes      []string   `json:"attributes"`
	Children        []*cdpNode `json:"children"`
	ShadowRoots     []*cdpNode `json:"shadowRoots"`
	ShadowRootType  string     `json:"shadowRootType"`
	TemplateContent *cdpNode   `json:"templateContent"`
	ContentDocument *cdpNode   `json:"contentDocument"`
	FrameID         string     `json:"frameId"`
	DocumentURL     string     `json:"documentURL"`
	PseudoType      string     `json:"pseudoType"`
	AdoptedSheets   []string   `json:"adoptedStyleSheets"`

	parent *cdpNode
	host   *cdpNode // shadow root / template content / content document: its element
	ua     bool     // inside a user-agent shadow tree: not page content
}

// session is one CDP target: the top page ("") or an out-of-process iframe.
type session struct {
	id      string
	frameID string
	root    *cdpNode
	nodes   map[int]*cdpNode      // CDP nodeId -> node (current snapshot)
	ids     map[int]rec.NodeID    // backendNodeId -> recording id (whole run)
	owner   rec.NodeID            // the iframe element this document hangs from
	sheets  map[string]rec.NodeID // styleSheetId -> sheet id
}

// converter turns one raw run into records. Every record it emits is
// applied to a reducer first; one the reducer refuses is dropped and
// counted, so the output always validates and the counts say what was lost.
type converter struct {
	t0       int64
	tab      string
	vp       rec.Viewport
	st       *rec.State
	out      []rec.Record
	next     rec.NodeID
	sessions map[string]*session
	bodies   map[string][]byte // asset bodies by hash
	dropped  map[string]int
	examples []string
	segs     int
	exclude  []string // asset URLs left out (see excluded)
}

func newConverter(name string, t0 int64) *converter {
	c := &converter{t0: t0, st: rec.NewState(), sessions: map[string]*session{},
		bodies: map[string][]byte{}, dropped: map[string]int{}, vp: rec.Viewport{W: 1280, H: 800, DPR: 1}}
	c.emit(t0, &rec.Head{Format: rec.Format, V: rec.Version, ID: name, Started: time.UnixMilli(t0).UTC().Format(time.RFC3339Nano)})
	return c
}

func (c *converter) sess(id string) *session {
	s := c.sessions[id]
	if s == nil {
		s = &session{id: id, nodes: map[int]*cdpNode{}, ids: map[int]rec.NodeID{}, sheets: map[string]rec.NodeID{}}
		c.sessions[id] = s
	}
	return s
}

func (c *converter) newID() rec.NodeID { c.next++; return c.next }

// idOf is a node's recording id: one per backendNodeId per session, so a
// node keeps its id through CDP's re-snapshots and moves.
func (c *converter) idOf(s *session, n *cdpNode) rec.NodeID {
	if id, ok := s.ids[n.BackendNodeID]; ok && n.BackendNodeID != 0 {
		return id
	}
	id := c.newID()
	if n.BackendNodeID != 0 {
		s.ids[n.BackendNodeID] = id
	}
	return id
}

func (c *converter) emit(t int64, r rec.Record) bool {
	rt := t - c.t0
	if rt < c.st.T {
		rt = c.st.T
	}
	r.SetTime(rt)
	if err := c.st.Apply(r); err != nil {
		c.dropped[r.Kind()]++
		if len(c.examples) < 12 {
			c.examples = append(c.examples, fmt.Sprintf("t=%d %s: %v", rt, r.Kind(), err))
		}
		return false
	}
	c.out = append(c.out, r)
	return true
}

// visible: a node that is page content (not a user-agent shadow tree, not a
// pseudo element) in a session whose document is attached.
func (c *converter) visible(s *session, n *cdpNode) bool {
	if n == nil || n.ua || n.PseudoType != "" {
		return false
	}
	return s.id == "" || s.owner != 0
}

// index walks a fresh subtree: parent links, UA marks, the nodeId index.
func (s *session) index(n, parent, host *cdpNode, ua bool) {
	n.parent, n.host, n.ua = parent, host, ua
	s.nodes[n.NodeID] = n
	for _, k := range n.Children {
		s.index(k, n, nil, ua)
	}
	for _, r := range n.ShadowRoots {
		s.index(r, nil, n, ua || r.ShadowRootType == "user-agent")
	}
	if n.TemplateContent != nil {
		s.index(n.TemplateContent, nil, n, ua)
	}
	if n.ContentDocument != nil {
		s.index(n.ContentDocument, nil, n, ua)
	}
}

func (s *session) forget(n *cdpNode) {
	delete(s.nodes, n.NodeID)
	for _, k := range n.Children {
		s.forget(k)
	}
	for _, r := range n.ShadowRoots {
		s.forget(r)
	}
	if n.TemplateContent != nil {
		s.forget(n.TemplateContent)
	}
	if n.ContentDocument != nil {
		s.forget(n.ContentDocument)
	}
}

// ns is the namespace an element is in, from its ancestors.
func ns(n *cdpNode) string {
	name := n.LocalName
	switch name {
	case "svg":
		return "svg"
	case "math":
		return "math"
	}
	p := n.parent
	if p == nil && n.host != nil {
		p = n.host
	}
	for ; p != nil; p = p.parent {
		if p.NodeType != 1 {
			continue
		}
		if p.LocalName == "foreignobject" || p.LocalName == "foreignObject" {
			return ""
		}
		if pn := ns(p); pn != "" {
			return pn
		}
		return ""
	}
	return ""
}

// wire serializes a CDP subtree into a recording node. Frames found on the
// way (an iframe with an inline content document, or one whose
// out-of-process session is known) are returned to be emitted after it.
func (c *converter) wire(s *session, n *cdpNode, frames *[]frameRef) *rec.Node {
	if n.ua || n.PseudoType != "" {
		return nil
	}
	w := &rec.Node{ID: c.idOf(s, n)}
	switch n.NodeType {
	case 9:
		w.Y = rec.YDoc
	case 10:
		w.Y, w.N = rec.YDoctype, n.NodeName
	case 1:
		w.Y, w.N, w.NS = rec.YEl, n.LocalName, ns(n)
		if len(n.Attributes) > 0 {
			w.A = map[string]string{}
			for i := 0; i+1 < len(n.Attributes); i += 2 {
				w.A[n.Attributes[i]] = n.Attributes[i+1]
			}
		}
		for _, r := range n.ShadowRoots {
			if r.ShadowRootType != "user-agent" {
				w.SR = c.wire(s, r, frames)
			}
		}
		if n.TemplateContent != nil {
			w.TC = c.wire(s, n.TemplateContent, frames)
			w.TC.Y = rec.YFrag
		}
		if n.LocalName == "iframe" || n.LocalName == "frame" {
			*frames = append(*frames, frameRef{s: s, el: n})
		}
	case 3, 4:
		w.Y, w.V = rec.YText, rec.Str(n.NodeValue)
	case 8:
		w.Y, w.V = rec.YComment, rec.Str(n.NodeValue)
	case 11:
		if n.ShadowRootType != "" {
			w.Y, w.Mode = rec.YShadow, n.ShadowRootType
		} else {
			w.Y = rec.YFrag
		}
	default:
		return nil
	}
	for _, k := range n.Children {
		if kw := c.wire(s, k, frames); kw != nil {
			w.C = append(w.C, *kw)
		}
	}
	return w
}

type frameRef struct {
	s  *session
	el *cdpNode
}

// attachFrames emits the documents of iframes that just appeared: an inline
// content document, or the out-of-process session with that frame id.
func (c *converter) attachFrames(t int64, frames []frameRef) {
	for _, f := range frames {
		host := c.idOf(f.s, f.el)
		if f.el.ContentDocument != nil {
			var more []frameRef
			d := c.wire(f.s, f.el.ContentDocument, &more)
			if c.emit(t, &rec.Doc{Doc: d.ID, Host: host, URL: f.el.ContentDocument.DocumentURL, Tree: *d}) {
				c.attachFrames(t, more)
			}
			continue
		}
		for _, o := range c.sessions {
			if o.id != "" && o.frameID == f.el.FrameID && f.el.FrameID != "" && o.root != nil {
				o.owner = host
				c.emitSessionDoc(t, o)
			}
		}
	}
}

func (c *converter) emitSessionDoc(t int64, s *session) {
	var frames []frameRef
	d := c.wire(s, s.root, &frames)
	if !c.emit(t, &rec.Doc{Doc: d.ID, Host: s.owner, URL: s.root.DocumentURL, Tree: *d}) {
		s.owner = 0
		return
	}
	c.emitAdopted(t, s, s.root)
	c.attachFrames(t, frames)
}

// emitAdopted emits the adoptedStyleSheets carried on a snapshot's
// documents and shadow roots.
func (c *converter) emitAdopted(t int64, s *session, n *cdpNode) {
	if n.ua {
		return
	}
	if len(n.AdoptedSheets) > 0 {
		c.adopt(t, s, n, n.AdoptedSheets)
	}
	for _, k := range n.Children {
		c.emitAdopted(t, s, k)
	}
	for _, r := range n.ShadowRoots {
		c.emitAdopted(t, s, r)
	}
}

func (c *converter) adopt(t int64, s *session, n *cdpNode, ids []string) {
	var sheets []rec.NodeID
	for _, sid := range ids {
		id, ok := s.sheets[sid]
		if !ok {
			id = c.newID()
			s.sheets[sid] = id
			c.emit(t, &rec.Sheet{ID: id})
		}
		sheets = append(sheets, id)
	}
	c.emit(t, &rec.Adopt{ID: c.idOf(s, n), Sheets: sheets})
}

// snapshot handles a DOM.getDocument result for a session.
func (c *converter) snapshot(r raw, frameID string, root *cdpNode) {
	s := c.sess(r.S)
	s.frameID = frameID
	sameDoc := s.root != nil && s.root.BackendNodeID == root.BackendNodeID
	if s.root != nil {
		s.forget(s.root)
	}
	s.root = root
	s.index(root, nil, nil, false)
	if r.S != "" {
		if s.owner != 0 && c.st.Node(s.owner) != nil {
			c.emitSessionDoc(r.T, s)
		}
		return
	}
	if sameDoc && c.st.Top != nil {
		// CDP re-snapshotted the same document: same ids, new children.
		var frames []frameRef
		d := c.wire(s, root, &frames)
		if c.emit(r.T, &rec.Kids{P: d.ID, C: d.C}) {
			c.emitAdopted(r.T, s, root)
			c.attachFrames(r.T, frames)
		}
		return
	}
	c.segs++
	why := rec.SegLive
	if c.segs > 1 {
		why = rec.SegNavigate
	}
	c.emit(r.T, &rec.Seg{Seg: c.segs, Why: why, Tab: rec.Tab{ID: frameID, URL: root.DocumentURL, Title: title(root)}, VP: c.vp})
	var frames []frameRef
	d := c.wire(s, root, &frames)
	c.emit(r.T, &rec.Doc{Doc: d.ID, URL: root.DocumentURL, Tree: *d})
	c.emitAdopted(r.T, s, root)
	c.attachFrames(r.T, frames)
}

func title(doc *cdpNode) string {
	var found string
	var walk func(n *cdpNode)
	walk = func(n *cdpNode) {
		if found != "" {
			return
		}
		if n.NodeType == 1 && n.LocalName == "title" {
			for _, k := range n.Children {
				found += k.NodeValue
			}
			return
		}
		for _, k := range n.Children {
			walk(k)
		}
	}
	walk(doc)
	return strings.TrimSpace(found)
}

// prevVisible is the closest earlier sibling that is page content, or nil.
func (c *converter) prevVisible(s *session, parent *cdpNode, before int) *cdpNode {
	for i := before; i >= 0; i-- {
		if k := parent.Children[i]; c.visible(s, k) {
			return k
		}
	}
	return nil
}

// event handles one raw line.
func (c *converter) event(r raw) error {
	s := c.sess(r.S)
	var p map[string]json.RawMessage
	if err := json.Unmarshal(r.Extra, &p); err != nil {
		return err
	}
	get := func(k string, v any) bool { return p[k] != nil && json.Unmarshal(p[k], v) == nil }
	node := func(k string) *cdpNode {
		var id int
		if !get(k, &id) {
			return nil
		}
		return s.nodes[id]
	}
	switch r.Type {
	case "viewport":
		var w, h int
		get("w", &w)
		get("h", &h)
		c.vp = rec.Viewport{W: w, H: h, DPR: 1}
	case "snapshot":
		var root cdpNode
		var frameID string
		get("root", &root)
		get("frameId", &frameID)
		c.snapshot(r, frameID, &root)
	case "childNodeInserted":
		parent := node("parentNodeId")
		var n cdpNode
		var prevID int
		if parent == nil || !get("node", &n) {
			return errSkip
		}
		get("previousNodeId", &prevID)
		at := 0
		for i, k := range parent.Children {
			if k.NodeID == prevID {
				at = i + 1
			}
		}
		parent.Children = append(parent.Children[:at], append([]*cdpNode{&n}, parent.Children[at:]...)...)
		s.index(&n, parent, nil, parent.ua)
		if !c.visible(s, parent) || !c.visible(s, &n) {
			return nil
		}
		var after rec.NodeID
		if pv := c.prevVisible(s, parent, at-1); pv != nil {
			after = c.idOf(s, pv)
		}
		var frames []frameRef
		w := c.wire(s, &n, &frames)
		if c.emit(r.T, &rec.Ins{P: c.idOf(s, parent), After: after, Node: *w}) {
			c.emitAdopted(r.T, s, &n)
			c.attachFrames(r.T, frames)
		}
	case "childNodeRemoved":
		n := node("nodeId")
		if n == nil || n.parent == nil {
			return errSkip
		}
		par := n.parent
		for i, k := range par.Children {
			if k == n {
				par.Children = append(par.Children[:i], par.Children[i+1:]...)
				break
			}
		}
		s.forget(n)
		if c.visible(s, n) && c.visible(s, par) {
			c.emit(r.T, &rec.Rm{ID: c.idOf(s, n)})
		}
	case "setChildNodes":
		parent := node("parentId")
		var kids []*cdpNode
		if parent == nil || !get("nodes", &kids) {
			return errSkip
		}
		for _, k := range parent.Children {
			s.forget(k)
		}
		parent.Children = kids
		for _, k := range kids {
			s.index(k, parent, nil, parent.ua)
		}
		if !c.visible(s, parent) {
			return nil
		}
		var frames []frameRef
		var ws []rec.Node
		for _, k := range kids {
			if w := c.wire(s, k, &frames); w != nil {
				ws = append(ws, *w)
			}
		}
		if c.emit(r.T, &rec.Kids{P: c.idOf(s, parent), C: ws}) {
			for _, k := range kids {
				c.emitAdopted(r.T, s, k)
			}
			c.attachFrames(r.T, frames)
		}
	case "attributeModified", "attributeRemoved":
		n := node("nodeId")
		var name, value string
		if n == nil || !get("name", &name) {
			return errSkip
		}
		var v *string
		if r.Type == "attributeModified" && get("value", &value) {
			v = &value
		}
		setAttr(n, name, v)
		if c.visible(s, n) {
			c.emit(r.T, &rec.Attr{ID: c.idOf(s, n), Name: name, V: v})
		}
	case "characterDataModified":
		n := node("nodeId")
		var v string
		if n == nil || !get("characterData", &v) {
			return errSkip
		}
		n.NodeValue = v
		if c.visible(s, n) {
			c.emit(r.T, &rec.Text{ID: c.idOf(s, n), V: v})
		}
	case "shadowRootPushed":
		host := node("hostId")
		var root cdpNode
		if host == nil || !get("root", &root) {
			return errSkip
		}
		host.ShadowRoots = append(host.ShadowRoots, &root)
		s.index(&root, nil, host, host.ua || root.ShadowRootType == "user-agent")
		if c.visible(s, host) && root.ShadowRootType != "user-agent" {
			var frames []frameRef
			w := c.wire(s, &root, &frames)
			if c.emit(r.T, &rec.Shadow{Host: c.idOf(s, host), Root: *w}) {
				c.attachFrames(r.T, frames)
			}
		}
	case "sheet":
		// A constructed sheet's text (adopted sheets).
		var id, text string
		get("styleSheetId", &id)
		get("text", &text)
		sid, ok := s.sheets[id]
		if !ok {
			sid = c.newID()
			s.sheets[id] = sid
		}
		c.emit(r.T, &rec.Sheet{ID: sid, Text: text})
	case "adoptedStyleSheetsModified":
		n := node("nodeId")
		var ids []string
		if n == nil || !get("adoptedStyleSheets", &ids) {
			return errSkip
		}
		if c.visible(s, n) {
			c.adopt(r.T, s, n, ids)
		}
	case "rules":
		// A <style>'s CSSOM after insertRule: its whole rule list.
		var owner int
		var rules []string
		get("ownerNode", &owner)
		get("rules", &rules)
		id, ok := s.ids[owner]
		if !ok {
			return errSkip
		}
		key := "rules:" + strconv.Itoa(owner)
		sid, ok := s.sheets[key]
		if !ok {
			sid = c.newID()
			s.sheets[key] = sid
		}
		c.emit(r.T, &rec.Sheet{ID: sid, Owner: id, Text: strings.Join(rules, "\n")})
	case "styleText":
		// A <style> over CDP's 10k text cut: its full text, into its text
		// node, so the DOM itself is right.
		var owner int
		var text string
		get("ownerNode", &owner)
		get("text", &text)
		var el *cdpNode
		for _, n := range s.nodes {
			if n.BackendNodeID == owner {
				el = n
			}
		}
		if el == nil || len(el.Children) == 0 {
			return errSkip
		}
		tn := el.Children[0]
		tn.NodeValue = text
		if c.visible(s, tn) {
			c.emit(r.T, &rec.Text{ID: c.idOf(s, tn), V: text})
		}
	case "asset":
		// A stylesheet body with absolute url()s.
		var url, text string
		get("url", &url)
		get("text", &text)
		c.asset(r.T, url, "text/css", []byte(text))
	case "assetBody":
		var url, mime, b64 string
		get("url", &url)
		get("mime", &mime)
		get("b64", &b64)
		if strings.HasPrefix(mime, "text/css") {
			return nil // its absolutized text came as "asset"
		}
		body, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			return err
		}
		c.asset(r.T, url, mime, body)
	case "font":
		var f struct {
			Font struct {
				Family  string `json:"fontFamily"`
				Style   string `json:"fontStyle"`
				Weight  string `json:"fontWeight"`
				Stretch string `json:"fontStretch"`
				Range   string `json:"unicodeRange"`
				Src     string `json:"src"`
			} `json:"font"`
		}
		json.Unmarshal(r.Extra, &f)
		if f.Font.Src == "" || f.Font.Family == "" {
			return nil // system fonts carry no src
		}
		c.emit(r.T, &rec.Font{Family: f.Font.Family, Src: f.Font.Src, Style: f.Font.Style, Weight: f.Font.Weight, Stretch: f.Font.Stretch, Range: f.Font.Range})
	case "value":
		n := node("nodeId")
		var v string
		if n == nil || !get("v", &v) || !c.visible(s, n) {
			return errSkip
		}
		in := &rec.Input{ID: c.idOf(s, n)}
		switch {
		case n.LocalName == "select":
			i, _ := strconv.Atoi(v)
			in.Sel = []int{i}
		case attr(n, "type") == "checkbox" || attr(n, "type") == "radio":
			b := v == "1"
			in.Checked = &b
		default:
			in.V = &v
		}
		c.emit(r.T, in)
	case "scroll":
		n := node("nodeId")
		var v string
		if n == nil || !get("v", &v) || !c.visible(s, n) {
			return errSkip
		}
		x, y := pair(v)
		id := c.idOf(s, n)
		if n.LocalName == "html" && n.parent != nil && n.parent.NodeType == 9 {
			id = c.idOf(s, n.parent) // the scrolling element: the viewport
		}
		c.emit(r.T, &rec.Scroll{ID: id, X: x, Y: y})
	case "click":
		n := node("nodeId")
		var v string
		if n == nil || !get("v", &v) || !c.visible(s, n) {
			return errSkip
		}
		x, y := pair(v)
		sx, sy := c.pageScroll()
		c.emit(r.T, &rec.Click{ID: c.idOf(s, n), X: x + sx, Y: y + sy})
	case "focus":
		n := node("nodeId")
		if n == nil || !c.visible(s, n) {
			return errSkip
		}
		c.emit(r.T, &rec.Focus{ID: c.idOf(s, n)})
	case "layout":
		return c.layout(r)
	}
	return nil
}

var errSkip = fmt.Errorf("skipped: refers to a node the mirror doesn't have")

func (c *converter) pageScroll() (float64, float64) {
	if c.st.Top == nil {
		return 0, 0
	}
	sc := c.st.Scrolls[c.st.Top.ID]
	return sc[0], sc[1]
}

func (c *converter) asset(t int64, url, mime string, body []byte) {
	for _, x := range c.exclude {
		if strings.Contains(url, x) {
			return
		}
	}
	hash := rec.HashOf(body)
	c.bodies[hash] = body
	c.emit(t, &rec.Asset{URL: url, Hash: hash, MIME: mime, Size: int64(len(body))})
}

// layout maps a DOMSnapshot capture of the top document to boxes of live
// elements.
func (c *converter) layout(r raw) error {
	var l struct {
		Docs []struct {
			URL        string      `json:"url"`
			Backend    []int       `json:"backend"`
			LayoutNode []int       `json:"layoutNode"`
			Bounds     [][]float64 `json:"bounds"`
		} `json:"docs"`
	}
	if err := json.Unmarshal(r.Extra, &l); err != nil {
		return err
	}
	if len(l.Docs) == 0 || c.st.Top == nil {
		return errSkip
	}
	s := c.sess("")
	d := l.Docs[0]
	boxes := map[rec.NodeID]rec.Box{}
	for i, ni := range d.LayoutNode {
		if ni >= len(d.Backend) || i >= len(d.Bounds) || len(d.Bounds[i]) < 4 {
			continue
		}
		id, ok := s.ids[d.Backend[ni]]
		if !ok {
			continue
		}
		if n := c.st.Node(id); n == nil || n.Y != rec.YEl {
			continue
		}
		b := d.Bounds[i]
		boxes[id] = rec.Box{round(b[0]), round(b[1]), round(b[2]), round(b[3])}
	}
	c.emit(r.T, &rec.Layout{Boxes: boxes})
	return nil
}

func round(f float64) float64 { return math.Round(f*100) / 100 }

func pair(v string) (float64, float64) {
	a, b, _ := strings.Cut(v, ",")
	x, _ := strconv.ParseFloat(a, 64)
	y, _ := strconv.ParseFloat(b, 64)
	return x, y
}

func attr(n *cdpNode, name string) string {
	for i := 0; i+1 < len(n.Attributes); i += 2 {
		if n.Attributes[i] == name {
			return n.Attributes[i+1]
		}
	}
	return ""
}

func setAttr(n *cdpNode, name string, v *string) {
	for i := 0; i+1 < len(n.Attributes); i += 2 {
		if n.Attributes[i] == name {
			if v == nil {
				n.Attributes = append(n.Attributes[:i], n.Attributes[i+2:]...)
			} else {
				n.Attributes[i+1] = *v
			}
			return
		}
	}
	if v != nil {
		n.Attributes = append(n.Attributes, name, *v)
	}
}

// summary lists what was dropped, for the generator's report.
func (c *converter) summary() string {
	var kinds []string
	for k, n := range c.dropped {
		kinds = append(kinds, fmt.Sprintf("%s=%d", k, n))
	}
	sort.Strings(kinds)
	return strings.Join(kinds, " ")
}
