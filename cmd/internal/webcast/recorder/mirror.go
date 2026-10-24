package recorder

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	rec "github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// The mirror is the recorder's pure core: CDP's DOM, per session, mapped to
// records. It holds each session's node tree as CDP describes it and the
// recording's ids, and turns DOM events and observer reports into records.
// Every record is applied to its own reducer before it is emitted; one the
// reducer refuses is counted and not emitted, so the output validates by
// construction and what was lost is visible (Dropped).
//
// It makes no CDP calls. What it needs from Chrome (a snapshot, a full text
// CDP truncated) it reports through its fields, and the shell brings the
// answer back. It was ported from #457's fixturegen/cdpconv.go, retired
// with the prototype's fixtures (#474; git history has it).

// cnode is CDP's DOM.Node, the fields the mirror reads.
type cnode struct {
	NodeID          int      `json:"nodeId"`
	BackendNodeID   int      `json:"backendNodeId"`
	NodeType        int      `json:"nodeType"`
	NodeName        string   `json:"nodeName"`
	LocalName       string   `json:"localName"`
	NodeValue       string   `json:"nodeValue"`
	Attributes      []string `json:"attributes"`
	Children        []*cnode `json:"children"`
	ChildNodeCount  int      `json:"childNodeCount"`
	ShadowRoots     []*cnode `json:"shadowRoots"`
	ShadowRootType  string   `json:"shadowRootType"`
	TemplateContent *cnode   `json:"templateContent"`
	ContentDocument *cnode   `json:"contentDocument"`
	FrameID         string   `json:"frameId"`
	DocumentURL     string   `json:"documentURL"`
	PseudoType      string   `json:"pseudoType"`
	AdoptedSheets   []string `json:"adoptedStyleSheets"`

	parent *cnode
	host   *cnode // a shadow root's, template content's or content document's element
	// hidden: not page content. A user-agent shadow tree, or the cast's own
	// in-page control (data-couchcast-ui).
	hidden bool
}

// controlAttr marks the cast's in-page control (#454), never recorded.
const controlAttr = "data-couchcast-ui"

// maxCDPText is where CDP cuts a node's value (#453).
const maxCDPText = 10000

// graveSize bounds the removed nodes a session remembers, so a move (a
// remove, then an insert of the same node) keeps its id.
var graveSize = 50000 // a var so tests can reach it

// msession is one CDP session: a page's, or an out-of-process frame's.
type msession struct {
	id      string
	top     bool   // the cast page's own session
	frameID string // the frame its document is (a page's target id)
	root    *cnode
	// doc is the current document's key (the shell's: its loader id and
	// root backend id). Backend ids can repeat across a cross-process
	// navigation, so they alone don't say a document is the same one.
	doc   string
	nodes map[int]*cnode // CDP nodeId -> node, current tree
	// byBackend and frames index the current tree too, so per-event work
	// is O(change), never a scan of the document.
	byBackend map[int]*cnode
	frames    map[string]*cnode // frame id -> its iframe element
	ids       map[int]rec.NodeID
	grave     []int      // backendNodeIds removed, oldest first
	owner     rec.NodeID // an out-of-process frame: its iframe element
	sheets    map[string]rec.NodeID
}

func newSession(id, frameID string, top bool) *msession {
	return &msession{id: id, frameID: frameID, top: top, nodes: map[int]*cnode{}, byBackend: map[int]*cnode{},
		frames: map[string]*cnode{}, ids: map[int]rec.NodeID{}, sheets: map[string]rec.NodeID{}}
}

// TextFetch names a text node whose value CDP truncated: the shell reads the
// full text and hands it to FullText.
type TextFetch struct {
	Session string
	Backend int
}

// mirror maps CDP to records. Not safe for concurrent use: the recorder's
// goroutine owns it.
type mirror struct {
	st       *rec.State
	lastKind string
	budget   rec.SketchBudget
	next     rec.NodeID
	sessions map[string]*msession
	out      []rec.Record
	// Dropped counts records the reducer refused, by kind; Desync counts
	// events naming a node the mirror doesn't have. Examples keeps a few.
	Dropped  map[string]int
	Desync   int
	Examples []string
	// Truncated lists text nodes to fetch in full, in order; the shell
	// drains it.
	Truncated []TextFetch
	// Expand lists nodes that arrived without their children (CDP sends a
	// count only, as for a shadow root pushed before its content): the
	// shell asks for them (DOM.requestChildNodes), and they come back as
	// setChildNodes.
	Expand []Expand
	// Refs lists assets the recorded elements reference; the shell fetches
	// those the recording lacks.
	Refs []Ref
}

// Ref is an asset an element references (img src and srcset, a stylesheet
// link, a poster), as written, with its document's URL to resolve it by.
type Ref struct {
	Session, DocURL, Raw string
	Srcset               bool // Raw is a srcset: comma-separated candidates
	Lazy                 bool // a loading=lazy image: maybe never loaded
	// Chosen: an image whose source the browser picks (it has a srcset, or
	// sits in a <picture>), so its src may not be what it loaded.
	Chosen bool
}

// Expand names a node whose children CDP hasn't sent.
type Expand struct {
	Session string
	NodeID  int
}

func newMirror() *mirror {
	return &mirror{st: rec.NewState(), sessions: map[string]*msession{}, Dropped: map[string]int{}}
}

// take returns and clears the records emitted so far.
func (m *mirror) take() []rec.Record {
	out := m.out
	m.out = nil
	return out
}

// emit applies r at time t to the mirror's reducer and queues it. t never
// goes back.
func (m *mirror) emit(t int64, r rec.Record) bool {
	refuse := func(err error) bool {
		m.Dropped[r.Kind()]++
		m.note(fmt.Sprintf("t=%d %s: %v", t, r.Kind(), err))
		return false
	}
	if r.Kind() == "sketch-snapshot" {
		return refuse(errors.New("catch-up snapshot cannot be persisted"))
	}
	duplicate, err := m.st.Sketches.Reconcile(r)
	if err != nil {
		return refuse(err)
	}
	if duplicate {
		return true
	}
	t, err = rec.OrderedTime(t, m.st.T, m.lastKind, r.Kind())
	if err != nil {
		return refuse(err)
	}
	r.SetTime(t)
	size, err := rec.SketchEncodedBytes(r)
	if err != nil {
		return refuse(err)
	}
	delta, err := m.budget.Next(r, size, false)
	if err != nil {
		return refuse(err)
	}
	if err := m.st.Apply(r); err != nil {
		return refuse(err)
	}
	m.budget.Commit(delta)
	m.lastKind = r.Kind()
	m.out = append(m.out, r)
	return true
}

func (m *mirror) note(s string) {
	if len(m.Examples) < 20 {
		m.Examples = append(m.Examples, s)
	}
}

func (m *mirror) newID() rec.NodeID { m.next++; return m.next }

// idOf is a node's recording id: one per backendNodeId per session for the
// document's life, so a node keeps its id across re-snapshots, a pause and
// resume, and moves.
func (m *mirror) idOf(s *msession, n *cnode) rec.NodeID {
	if id, ok := s.ids[n.BackendNodeID]; ok && n.BackendNodeID != 0 {
		return id
	}
	id := m.newID()
	if n.BackendNodeID != 0 {
		s.ids[n.BackendNodeID] = id
	}
	return id
}

// IDOf is the recording id of a session's node by backendNodeId (#467's
// injector maps both ways).
func (m *mirror) IDOf(session string, backend int) (rec.NodeID, bool) {
	s := m.sessions[session]
	if s == nil {
		return 0, false
	}
	id, ok := s.ids[backend]
	return id, ok && m.st.Node(id) != nil
}

// NodeOf is the session and backendNodeId of a live recording id.
func (m *mirror) NodeOf(id rec.NodeID) (string, int, bool) {
	if m.st.Node(id) == nil {
		return "", 0, false
	}
	for _, s := range m.sessions {
		for b, i := range s.ids {
			if i == id {
				return s.id, b, true
			}
		}
	}
	return "", 0, false
}

// bury sends the backend ids that left the tree to the graveyard and
// prunes it to graveSize, dropping the oldest ids. Call it after the update
// is applied: an id is pruned only when its node is absent from the tree as
// it now is, so a node a re-snapshot or set of children still has keeps its
// id, and a moved node (removed, then inserted) keeps its id while its
// removal is among the last graveSize.
func (s *msession) bury(gone []int) {
	for _, b := range gone {
		if !s.liveBackend(b) {
			s.grave = append(s.grave, b)
		}
	}
	if over := len(s.grave) - graveSize; over > 0 {
		for _, b := range s.grave[:over] {
			if !s.liveBackend(b) {
				delete(s.ids, b)
			}
		}
		s.grave = append(s.grave[:0], s.grave[over:]...)
	}
}

// backends lists a subtree's backend ids.
func backends(n *cnode) []int {
	var out []int
	walk(n, func(c *cnode) {
		if c.BackendNodeID != 0 {
			out = append(out, c.BackendNodeID)
		}
	})
	return out
}

func (s *msession) liveBackend(b int) bool { return s.byBackend[b] != nil }

func walk(n *cnode, f func(*cnode)) {
	if n == nil {
		return
	}
	f(n)
	for _, k := range n.Children {
		walk(k, f)
	}
	for _, r := range n.ShadowRoots {
		walk(r, f)
	}
	walk(n.TemplateContent, f)
	walk(n.ContentDocument, f)
}

// index links a fresh subtree into the session: parents, hidden marks, the
// nodeId index.
func (m *mirror) index(s *msession, n, parent, host *cnode, hidden bool) {
	n.parent, n.host = parent, host
	n.hidden = hidden || (n.NodeType == 1 && attr(n, controlAttr) != nil)
	s.nodes[n.NodeID] = n
	if n.BackendNodeID != 0 {
		s.byBackend[n.BackendNodeID] = n
	}
	if n.NodeType == 1 && n.FrameID != "" {
		s.frames[n.FrameID] = n
	}
	if n.ChildNodeCount > 0 && len(n.Children) == 0 && !n.hidden {
		m.Expand = append(m.Expand, Expand{Session: s.id, NodeID: n.NodeID})
	}
	for _, k := range n.Children {
		m.index(s, k, n, nil, n.hidden)
	}
	for _, r := range n.ShadowRoots {
		m.index(s, r, nil, n, n.hidden || r.ShadowRootType == "user-agent")
	}
	if n.TemplateContent != nil {
		m.index(s, n.TemplateContent, nil, n, n.hidden)
	}
	if n.ContentDocument != nil {
		m.index(s, n.ContentDocument, nil, n, n.hidden)
	}
}

func (s *msession) forget(n *cnode) {
	walk(n, func(c *cnode) {
		if s.nodes[c.NodeID] == c {
			delete(s.nodes, c.NodeID)
		}
		if s.byBackend[c.BackendNodeID] == c {
			delete(s.byBackend, c.BackendNodeID)
		}
		if c.FrameID != "" && s.frames[c.FrameID] == c {
			delete(s.frames, c.FrameID)
		}
	})
}

// visible: page content, in a session whose document is in the recording.
func (m *mirror) visible(s *msession, n *cnode) bool {
	if n == nil || n.hidden || n.PseudoType != "" {
		return false
	}
	return s.top || s.owner != 0
}

// ns is an element's namespace, from its ancestors.
func ns(n *cnode) string {
	switch n.LocalName {
	case "svg":
		return "svg"
	case "math":
		return "math"
	}
	p := n.parent
	if p == nil {
		p = n.host
	}
	for ; p != nil; p = p.parent {
		if p.NodeType != 1 {
			continue
		}
		if strings.EqualFold(p.LocalName, "foreignObject") {
			return ""
		}
		return ns(p)
	}
	return ""
}

// wiredAttrs is what the recording shows of an element's attributes: all of
// them, except a password input's value, which is never recorded.
func wiredAttrs(n *cnode) map[string]string {
	if len(n.Attributes) == 0 {
		return nil
	}
	a := map[string]string{}
	for i := 0; i+1 < len(n.Attributes); i += 2 {
		a[n.Attributes[i]] = n.Attributes[i+1]
	}
	if isPassword(n) {
		delete(a, "value")
	}
	return a
}

func isPassword(n *cnode) bool {
	t := attr(n, "type")
	return n.NodeType == 1 && n.LocalName == "input" && t != nil && strings.EqualFold(strings.TrimSpace(*t), "password")
}

func attr(n *cnode, name string) *string {
	for i := 0; i+1 < len(n.Attributes); i += 2 {
		if n.Attributes[i] == name {
			return &n.Attributes[i+1]
		}
	}
	return nil
}

func setAttr(n *cnode, name string, v *string) {
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

type frameRef struct {
	s  *msession
	el *cnode
}

// wire serializes a CDP subtree. Iframes found on the way are collected, to
// be emitted after it.
func (m *mirror) wire(s *msession, n *cnode, frames *[]frameRef) *rec.Node {
	if n.hidden || n.PseudoType != "" {
		return nil
	}
	w := &rec.Node{ID: m.idOf(s, n)}
	switch n.NodeType {
	case 9:
		w.Y = rec.YDoc
	case 10:
		w.Y, w.N = rec.YDoctype, n.NodeName
	case 1:
		w.Y, w.N, w.NS = rec.YEl, n.LocalName, ns(n)
		w.A = wiredAttrs(n)
		m.refs(s, n, w.A)
		for _, r := range n.ShadowRoots {
			if r.ShadowRootType != "user-agent" && !r.hidden {
				w.SR = m.wire(s, r, frames)
			}
		}
		if n.TemplateContent != nil {
			if tc := m.wire(s, n.TemplateContent, frames); tc != nil {
				tc.Y = rec.YFrag
				w.TC = tc
			}
		}
		if n.LocalName == "iframe" || n.LocalName == "frame" {
			*frames = append(*frames, frameRef{s: s, el: n})
		}
	case 3, 4:
		w.Y, w.V = rec.YText, rec.Str(n.NodeValue)
		m.checkTruncated(s, n)
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
		if kw := m.wire(s, k, frames); kw != nil {
			w.C = append(w.C, *kw)
		}
	}
	return w
}

// refAttrs: the attributes that reference an asset, by element.
var refAttrs = map[string][]string{"img": {"src", "srcset"}, "source": {"src", "srcset"}, "link": {"href"},
	"video": {"poster"}, "input": {"src"}, "image": {"href", "xlink:href"}}

// refs lists an element's asset references.
func (m *mirror) refs(s *msession, n *cnode, a map[string]string) {
	names := refAttrs[n.LocalName]
	if len(names) == 0 || len(a) == 0 {
		return
	}
	if n.LocalName == "link" && !strings.Contains(strings.ToLower(a["rel"]), "stylesheet") && !strings.Contains(strings.ToLower(a["rel"]), "icon") {
		return
	}
	doc := ""
	for p := n; p != nil; p = up(p) {
		if p.NodeType == 9 {
			doc = p.DocumentURL
			break
		}
	}
	for _, k := range names {
		if v := a[k]; v != "" {
			m.Refs = append(m.Refs, Ref{Session: s.id, DocURL: doc, Raw: v, Srcset: k == "srcset",
				Lazy:   strings.EqualFold(a["loading"], "lazy"),
				Chosen: a["srcset"] != "" || n.LocalName == "source" || (n.parent != nil && n.parent.LocalName == "picture")})
		}
	}
}

func up(n *cnode) *cnode {
	if n.parent != nil {
		return n.parent
	}
	return n.host
}

// checkTruncated asks for a text node's full value when CDP may have cut it.
func (m *mirror) checkTruncated(s *msession, n *cnode) {
	if n.BackendNodeID != 0 && utf8.RuneCountInString(n.NodeValue) >= maxCDPText {
		m.Truncated = append(m.Truncated, TextFetch{Session: s.id, Backend: n.BackendNodeID})
	}
}

// FullText sets a text node's full value, once the shell has read it.
func (m *mirror) FullText(t int64, session string, backend int, text string) {
	s := m.sessions[session]
	if s == nil {
		return
	}
	if n := s.byBackend[backend]; n != nil && (n.NodeType == 3 || n.NodeType == 4) && n.NodeValue != text {
		n.NodeValue = text
		if m.visible(s, n) {
			m.emit(t, &rec.Text{ID: m.idOf(s, n), V: text})
		}
	}
}

// attachFrames emits the documents of iframes that just appeared: an
// in-process frame's content document, or the out-of-process session whose
// frame it is.
func (m *mirror) attachFrames(t int64, frames []frameRef) {
	for _, f := range frames {
		host := m.idOf(f.s, f.el)
		if d := f.el.ContentDocument; d != nil {
			var more []frameRef
			w := m.wire(f.s, d, &more)
			if w != nil && m.emit(t, &rec.Doc{Doc: w.ID, Host: host, URL: d.DocumentURL, Tree: *w}) {
				m.emitAdopted(t, f.s, d)
				m.attachFrames(t, more)
			}
			continue
		}
		if f.el.FrameID == "" {
			continue
		}
		for _, o := range m.sessions {
			if !o.top && o.frameID == f.el.FrameID && o.root != nil {
				o.owner = host
				m.emitSessionDoc(t, o)
			}
		}
	}
}

func (m *mirror) emitSessionDoc(t int64, s *msession) {
	var frames []frameRef
	w := m.wire(s, s.root, &frames)
	if w == nil || !m.emit(t, &rec.Doc{Doc: w.ID, Host: s.owner, URL: s.root.DocumentURL, Tree: *w}) {
		s.owner = 0
		return
	}
	m.emitAdopted(t, s, s.root)
	m.attachFrames(t, frames)
}

// emitAdopted emits the adoptedStyleSheets a snapshot carries.
func (m *mirror) emitAdopted(t int64, s *msession, n *cnode) {
	walk(n, func(c *cnode) {
		if !c.hidden && len(c.AdoptedSheets) > 0 {
			m.adopt(t, s, c, c.AdoptedSheets)
		}
	})
}

func (m *mirror) adopt(t int64, s *msession, n *cnode, ids []string) {
	var sheets []rec.NodeID
	for _, sid := range ids {
		sheets = append(sheets, m.sheetID(t, s, sid))
	}
	m.emit(t, &rec.Adopt{ID: m.idOf(s, n), Sheets: sheets})
}

// sheetID is a constructed sheet's id, emitting an empty sheet the first
// time so an adopt can name it before its text arrives.
func (m *mirror) sheetID(t int64, s *msession, styleSheetID string) rec.NodeID {
	id, ok := s.sheets[styleSheetID]
	if !ok {
		id = m.newID()
		s.sheets[styleSheetID] = id
		m.emit(t, &rec.Sheet{ID: id})
	}
	return id
}

// Session registers a session before its first snapshot: the cast page's
// (top) or an out-of-process frame's.
func (m *mirror) Session(id, frameID string, top bool) {
	if s := m.sessions[id]; s != nil {
		s.top, s.frameID = top, frameID
		return
	}
	m.sessions[id] = newSession(id, frameID, top)
}

// Drop forgets a session (it detached): its tree and ids. Its frame's last
// document stays in the recording until its iframe goes or gets another.
func (m *mirror) Drop(id string) { delete(m.sessions, id) }

// SameDocument reports whether key names the document the session already
// mirrors (a resume, or a re-snapshot of the same document).
func (m *mirror) SameDocument(session, key string) bool {
	s := m.sessions[session]
	return s != nil && s.root != nil && key != "" && s.doc == key
}

// install makes root, the document key names, the session's current tree. A
// new document drops the old one's ids: they named its nodes only.
func (m *mirror) install(s *msession, root *cnode, key string) {
	same := key != "" && key == s.doc
	var old []int
	if same && s.root == nil {
		// Inactive top documents lose their tree but retain bounded backend IDs.
		// Reconcile removals on return through the same graveyard as live updates.
		old = make([]int, 0, len(s.ids))
		for backend := range s.ids {
			old = append(old, backend)
		}
	}
	if s.root != nil {
		if same {
			old = backends(s.root)
		}
		s.forget(s.root)
	}
	if !same {
		s.ids, s.grave, s.sheets = map[int]rec.NodeID{}, nil, map[string]rec.NodeID{}
	}
	s.root, s.doc = root, key
	m.index(s, root, nil, nil, false)
	// The same document again: nodes it no longer has go to the graveyard,
	// so the id map stays bounded.
	s.bury(old)
}

// Seg emits a segment start and the top document's snapshot, then its
// in-process frames. Out-of-process frames' trees are dropped (the reducer
// drops all DOM state at a segment): they come again through FrameDoc, once
// each, after this.
func (m *mirror) Seg(t int64, seg *rec.Seg, session string, root *cnode, key string) error {
	s := m.sessions[session]
	if s == nil || !s.top {
		return fmt.Errorf("recorder: session %s is not the cast page's", session)
	}
	for _, o := range m.sessions {
		if o != s {
			// One cast page at a time; frames come again through FrameDoc.
			o.top, o.owner = false, 0
			if o.root != nil {
				o.forget(o.root)
				o.root = nil
			}
		}
	}
	m.install(s, root, key)
	if !m.emit(t, seg) {
		return errors.New("recorder: segment refused")
	}
	var frames []frameRef
	w := m.wire(s, root, &frames)
	if w == nil || !m.emit(t, &rec.Doc{Doc: w.ID, URL: root.DocumentURL, Tree: *w}) {
		return errors.New("recorder: top document refused")
	}
	m.emitAdopted(t, s, root)
	m.attachFrames(t, frames)
	return nil
}

// Resnapshot replaces a document's children with a fresh snapshot of the
// same document (a resync, or CDP's documentUpdated on the same document).
func (m *mirror) Resnapshot(t int64, session string, root *cnode) {
	s := m.sessions[session]
	if s == nil {
		return
	}
	m.install(s, root, s.doc)
	if !m.visible(s, root) {
		return
	}
	var frames []frameRef
	w := m.wire(s, root, &frames)
	if w != nil && m.emit(t, &rec.Kids{P: w.ID, C: w.C}) {
		m.emitAdopted(t, s, root)
		m.attachFrames(t, frames)
	}
}

// FrameDoc takes an out-of-process frame's snapshot. Its document is emitted
// when its iframe element is in the recording (now, or when it appears).
func (m *mirror) FrameDoc(t int64, session string, root *cnode, key string) {
	s := m.sessions[session]
	if s == nil || s.top {
		return
	}
	m.install(s, root, key)
	s.owner = 0
	if host := m.frameOwner(s.frameID); host != 0 {
		s.owner = host
		m.emitSessionDoc(t, s)
	}
}

// frameOwner is the recorded iframe element showing frame id, or 0.
func (m *mirror) frameOwner(frameID string) rec.NodeID {
	for _, o := range m.sessions {
		if !m.visible(o, o.root) {
			continue
		}
		if n := o.frames[frameID]; n != nil && m.visible(o, n) {
			if id, ok := o.ids[n.BackendNodeID]; ok && m.st.Node(id) != nil {
				return id
			}
		}
	}
	return 0
}

// prevVisible is the closest earlier sibling that is page content.
func (m *mirror) prevVisible(s *msession, parent *cnode, before int) *cnode {
	for i := before; i >= 0; i-- {
		if k := parent.Children[i]; m.visible(s, k) {
			return k
		}
	}
	return nil
}

// errDesync: an event names a node the mirror doesn't have. The shell
// resyncs; the event is never applied blindly.
var errDesync = errors.New("recorder: event names a node the mirror doesn't have")

// Event applies one CDP DOM event of a session.
func (m *mirror) Event(t int64, session, method string, params json.RawMessage) error {
	s := m.sessions[session]
	if s == nil || s.root == nil {
		return nil
	}
	var p map[string]json.RawMessage
	if err := json.Unmarshal(params, &p); err != nil {
		return err
	}
	get := func(k string, v any) bool { return p[k] != nil && json.Unmarshal(p[k], v) == nil }
	node := func(k string) *cnode {
		var id int
		if !get(k, &id) {
			return nil
		}
		return s.nodes[id]
	}
	desync := func() error { m.Desync++; m.note(fmt.Sprintf("t=%d %s: unknown node", t, method)); return errDesync }
	switch method {
	case "DOM.childNodeInserted":
		parent := node("parentNodeId")
		var n cnode
		var prevID int
		if parent == nil || !get("node", &n) {
			return desync()
		}
		get("previousNodeId", &prevID)
		at := 0
		if prevID != 0 {
			at = -1
			for i, k := range parent.Children {
				if k.NodeID == prevID {
					at = i + 1
				}
			}
			if at < 0 {
				return desync() // never guess a position
			}
		}
		parent.Children = append(parent.Children[:at], append([]*cnode{&n}, parent.Children[at:]...)...)
		m.index(s, &n, parent, nil, parent.hidden)
		if !m.visible(s, parent) || !m.visible(s, &n) {
			return nil
		}
		var after rec.NodeID
		if pv := m.prevVisible(s, parent, at-1); pv != nil {
			after = m.idOf(s, pv)
		}
		var frames []frameRef
		w := m.wire(s, &n, &frames)
		if m.emit(t, &rec.Ins{P: m.idOf(s, parent), After: after, Node: *w}) {
			m.emitAdopted(t, s, &n)
			m.attachFrames(t, frames)
		}
	case "DOM.childNodeRemoved":
		n := node("nodeId")
		if n == nil || n.parent == nil {
			return desync()
		}
		par := n.parent
		for i, k := range par.Children {
			if k == n {
				par.Children = append(par.Children[:i], par.Children[i+1:]...)
				break
			}
		}
		s.forget(n)
		s.bury(backends(n))
		if m.visible(s, n) && m.visible(s, par) {
			m.emit(t, &rec.Rm{ID: m.idOf(s, n)})
		}
	case "DOM.setChildNodes":
		parent := node("parentId")
		var kids []*cnode
		if parent == nil || !get("nodes", &kids) {
			return desync()
		}
		var old []int
		for _, k := range parent.Children {
			old = append(old, backends(k)...)
			s.forget(k)
		}
		parent.Children = kids
		for _, k := range kids {
			m.index(s, k, parent, nil, parent.hidden)
		}
		s.bury(old)
		if !m.visible(s, parent) {
			return nil
		}
		var frames []frameRef
		var ws []rec.Node
		for _, k := range kids {
			if w := m.wire(s, k, &frames); w != nil {
				ws = append(ws, *w)
			}
		}
		if m.emit(t, &rec.Kids{P: m.idOf(s, parent), C: ws}) {
			for _, k := range kids {
				m.emitAdopted(t, s, k)
			}
			m.attachFrames(t, frames)
		}
	case "DOM.attributeModified", "DOM.attributeRemoved":
		n := node("nodeId")
		var name, value string
		if n == nil || !get("name", &name) {
			return desync()
		}
		var v *string
		if method == "DOM.attributeModified" && get("value", &value) {
			v = &value
		}
		before := wiredAttrs(n)
		setAttr(n, name, v)
		if m.visible(s, n) {
			m.attrDiff(t, m.idOf(s, n), name, before, wiredAttrs(n))
		}
	case "DOM.characterDataModified":
		n := node("nodeId")
		var v string
		if n == nil || !get("characterData", &v) {
			return desync()
		}
		n.NodeValue = v
		if m.visible(s, n) {
			m.emit(t, &rec.Text{ID: m.idOf(s, n), V: v})
			m.checkTruncated(s, n)
		}
	case "DOM.shadowRootPushed":
		host := node("hostId")
		var root cnode
		if host == nil || !get("root", &root) {
			return desync()
		}
		host.ShadowRoots = append(host.ShadowRoots, &root)
		m.index(s, &root, nil, host, host.hidden || root.ShadowRootType == "user-agent")
		if m.visible(s, host) && !root.hidden {
			var frames []frameRef
			if w := m.wire(s, &root, &frames); w != nil && m.emit(t, &rec.Shadow{Host: m.idOf(s, host), Root: *w}) {
				m.attachFrames(t, frames)
			}
		}
	case "DOM.adoptedStyleSheetsModified":
		n := node("nodeId")
		var ids []string
		if n == nil || !get("adoptedStyleSheets", &ids) {
			return desync()
		}
		if m.visible(s, n) {
			m.adopt(t, s, n, ids)
		}
	case "DOM.childNodeCountUpdated":
		// CDP sends only this for a node whose children it never sent (one
		// inserted empty, then filled by script: pair#513's player). Ask
		// for them as for a node sent with a count only (index); they come
		// back as setChildNodes, which replaces whatever the mirror held.
		n := node("nodeId")
		var count int
		if n != nil && get("childNodeCount", &count) && count != len(n.Children) && !n.hidden {
			n.ChildNodeCount = count
			// Once per node while queued: a script filling it sends a
			// count per insert, and one request brings them all.
			if e := (Expand{Session: s.id, NodeID: n.NodeID}); !slices.Contains(m.Expand, e) {
				m.Expand = append(m.Expand, e)
			}
		}
	case "DOM.inlineStyleInvalidated", "DOM.documentUpdated",
		"DOM.pseudoElementAdded", "DOM.pseudoElementRemoved", "DOM.shadowRootPopped", "DOM.distributedNodesUpdated":
		// The shell handles documentUpdated and inline style; the rest
		// carry nothing the format holds.
	}
	return nil
}

// attrDiff emits the attribute changes between two wired views, the named
// attribute first.
func (m *mirror) attrDiff(t int64, id rec.NodeID, name string, before, after map[string]string) {
	keys := []string{name}
	for k := range before {
		if k != name {
			keys = append(keys, k)
		}
	}
	for k := range after {
		if _, ok := before[k]; !ok && k != name {
			keys = append(keys, k)
		}
	}
	// Removals before additions, so a type flip to password drops the value
	// first.
	for pass := 0; pass < 2; pass++ {
		for _, k := range keys {
			b, hadB := before[k]
			a, hasA := after[k]
			switch {
			case pass == 0 && hadB && !hasA:
				m.emit(t, &rec.Attr{ID: id, Name: k})
			case pass == 1 && hasA && (!hadB || a != b):
				m.emit(t, &rec.Attr{ID: id, Name: k, V: rec.Str(a)})
			}
		}
	}
}

// InlineStyle sets an element's style attribute as CDP read it after an
// inlineStyleInvalidated (v nil: removed).
func (m *mirror) InlineStyle(t int64, session string, nodeID int, v *string) error {
	b, _ := json.Marshal(map[string]any{"nodeId": nodeID, "name": "style", "value": v})
	method := "DOM.attributeModified"
	if v == nil {
		method = "DOM.attributeRemoved"
	}
	return m.Event(t, session, method, b)
}

// ConstructedSheet sets a constructed sheet's text.
func (m *mirror) ConstructedSheet(t int64, session, styleSheetID, text string) {
	s := m.sessions[session]
	if s == nil {
		return
	}
	id := m.sheetID(t, s, styleSheetID)
	m.emit(t, &rec.Sheet{ID: id, Text: text})
}

// OwnedSheet sets a style or link element's CSSOM text (rules added by
// insertRule, or a CSSOM-only <style>).
func (m *mirror) OwnedSheet(t int64, session string, ownerBackend int, text string) {
	s := m.sessions[session]
	if s == nil {
		return
	}
	owner, ok := s.ids[ownerBackend]
	if !ok || m.st.Node(owner) == nil {
		return
	}
	key := fmt.Sprintf("owned:%d", ownerBackend)
	id, ok := s.sheets[key]
	if !ok {
		id = m.newID()
		s.sheets[key] = id
	}
	m.emit(t, &rec.Sheet{ID: id, Owner: owner, Text: text})
}

// Emit passes a record the mirror doesn't derive (head, stop, asset, font,
// title, frame) through its reducer. Success means acceptance into the ordered
// queue, never proof of persistence. The recorder's write result/r.err owns
// sink uncertainty; receipt consumers must check it before claiming durability.
func (m *mirror) Emit(t int64, r rec.Record) bool { return m.emit(t, r) }

// State is the mirror's reducer state (read-only to callers).
func (m *mirror) State() *rec.State { return m.st }
