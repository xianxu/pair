package recording

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// DOMNode is a node of the reduced DOM. Its fields are read-only to
// consumers: only Apply changes them.
type DOMNode struct {
	ID       NodeID
	Y, N, NS string
	A        map[string]string
	V        string // text and comment content
	Mode     string // shadow roots
	URL      string // documents

	Parent *DOMNode   // nil for a document, a shadow root and a template's content
	Kids   []*DOMNode // children in order
	// Host is what a parentless node hangs from: a shadow root's or a
	// template content's element, an iframe document's iframe; nil for the
	// top document.
	Host    *DOMNode
	Shadow  *DOMNode // an element's shadow root
	Content *DOMNode // a template's content
	Frame   *DOMNode // an iframe element's document
}

// SheetState is a stylesheet's current text and owner element (0:
// constructed).
type SheetState struct {
	Owner NodeID
	Text  string
}

// NoteView is a note with its current state and the segment it was made in.
type NoteView struct {
	*Note
	State      string
	Seg        int
	Revision   uint64
	ResolvedBy *Author
}

// NoteUnchanged is the writer's conditional-resolution guard. Reopening a
// note or editing its text after submission must not let a late ack resolve it.
func (s *State) NoteUnchanged(id string, revision uint64) bool {
	for _, n := range s.Notes {
		if n.ID == id {
			return n.State == NoteOpen && n.Revision == revision
		}
	}
	return false
}

// StrokeView is a stroke with the segment it was drawn in.
type StrokeView struct {
	*Stroke
	Seg int
}

// State is a recording reduced to one moment. Build it with Reduce, or with
// NewState and Apply one record at a time.
type State struct {
	T    int64
	Head *Head
	Seg  int
	// DocSeg is the segment the current document began in: the latest
	// that isn't a resume. Notes since then are on the current document.
	DocSeg int
	Why    string
	Live   bool
	// Ended: a stop end came. The cast is over; no segment follows.
	Ended bool
	Tab   Tab
	VP    Viewport
	URL   string
	Title string
	Top   *DOMNode

	Sheets    map[NodeID]*SheetState
	Adopted   map[NodeID][]NodeID
	Fonts     []*Font
	Assets    map[string]*Asset // by absolute URL
	Inputs    map[NodeID]*Input
	Scrolls   map[NodeID][2]float64
	Selection *Selection
	Focus     NodeID
	LastClick *Click
	// Hover is the element under the operator's pointer (0: none); Ptr is
	// the last sampled pointer position. Neither is in Canonical.
	Hover   NodeID
	Ptr     *[2]float64
	Boxes   map[NodeID]Box
	BoxesT  int64
	Notes   []*NoteView
	Strokes []*StrokeView
	Frames  []*Frame
	// Control is who drives (#466): per recording, ended by stop end.
	Control Control

	Sketches SketchState

	allowSketchSnapshot bool
	ctlSeen             bool // a ctl was applied: later ones must follow from it
	whole               bool // reading a whole recording from its head (Validate)
	nodes               map[NodeID]*DOMNode
	types               map[NodeID]string // id -> type and tag, for the whole recording
	awaitTop            bool              // a seg came; its top doc hasn't
	pageLoads           map[int]*PageLoad // immutable accepted segment provenance
}

// NewState is the state before any record.
func NewState() *State {
	s := &State{Assets: map[string]*Asset{}, types: map[NodeID]string{}}
	s.resetDOM()
	return s
}

func (s *State) resetDOM() {
	s.Top, s.Selection, s.Focus, s.LastClick, s.Fonts = nil, nil, 0, nil, nil
	s.Hover, s.Ptr = 0, nil
	s.nodes = map[NodeID]*DOMNode{}
	s.Sheets = map[NodeID]*SheetState{}
	s.Adopted = map[NodeID][]NodeID{}
	s.Inputs = map[NodeID]*Input{}
	s.Scrolls = map[NodeID][2]float64{}
	s.Boxes, s.BoxesT = map[NodeID]Box{}, 0
}

// Node is the live node with that id, or nil.
func (s *State) Node(id NodeID) *DOMNode { return s.nodes[id] }

// Paused reports whether a page was cast and the cast is paused on it (not
// ended): while it is, viewers do nothing (pair#499). The one rule the
// board, its writer and the viewer (model.js's paused) share.
func (s *State) Paused() bool { return s.Seg > 0 && !s.Live && !s.Ended }

// OnDoc reports whether the document that began at segment doc is still
// the current one and the cast is live on it: the check the stream's writer
// makes when it orders a record a viewer made on that document (#470), so a
// note checked before a navigation isn't filed on the new page, and one
// checked just before a pause isn't filed during it (pair#499: viewers do
// nothing while paused; it reverses #470's notes-while-paused).
func (s *State) OnDoc(doc int) bool { return s.Live && s.DocSeg == doc }

// NodeCount is how many nodes are live.
func (s *State) NodeCount() int { return len(s.nodes) }

// Reduce applies every record with t at or before t, in order. A record that
// fails stops it: the state is as of the record before, and the error says
// which record (by index).
func Reduce(recs []Record, t int64) (*State, error) {
	s := NewState()
	for i, r := range recs {
		if r.Time() > t {
			break
		}
		if err := s.Apply(r); err != nil {
			return s, fmt.Errorf("record %d (%s at t=%d): %w", i, r.Kind(), r.Time(), err)
		}
	}
	return s, nil
}

// Apply folds one record in. It is atomic: on error the state is unchanged.
// Every reference is checked before anything changes.
func (s *State) Apply(r Record) error {
	if s.Head == nil {
		h, ok := r.(*Head)
		if !ok {
			return errors.New("the first record must be head")
		}
		if h.V != Version {
			return fmt.Errorf("format version %d; this reader knows %d", h.V, Version)
		}
	}
	if sketchMetadata(r) != nil {
		duplicate, err := s.Sketches.Reconcile(r)
		if err != nil {
			return err
		}
		if duplicate {
			return nil
		}
	}
	if r.Time() < s.T {
		return fmt.Errorf("t=%d is before the previous record's t=%d", r.Time(), s.T)
	}
	if err := r.check(); err != nil {
		return err
	}
	if err := s.apply(r); err != nil {
		return err
	}
	s.T = r.Time()
	return nil
}

var errNotLive = errors.New("no live segment with a top document (a change outside seg … stop)")

func (s *State) inSegment() error {
	if !s.Live || s.Top == nil {
		return errNotLive
	}
	return nil
}

func (s *State) apply(r Record) error {
	if sketchMetadata(r) != nil {
		delta, err := NextSketch(&s.Sketches, r, SketchContext{Live: s.Live, Ended: s.Ended, Doc: s.DocSeg, Tab: s.Tab.ID, Driver: s.Control.Driving()})
		if err != nil {
			return err
		}
		delta.commit()
		return nil
	}
	// After stop end the cast is over: nothing may change its state again.
	switch r.(type) {
	case *Seg, *Ctl, *AnnotationClear:
		if s.Ended {
			return fmt.Errorf("a %s after stop end: the cast is over", r.Kind())
		}
	}
	switch r := r.(type) {
	case *SketchSnapshot:
		return s.installSketchSnapshot(r)
	case *Head:
		if s.Head != nil {
			return errors.New("a second head")
		}
		s.Head = r
	case *Seg:
		if r.Seg <= s.Seg {
			return fmt.Errorf("segment %d after segment %d", r.Seg, s.Seg)
		}
		if r.PageLoad != nil {
			if s.pageLoads == nil {
				s.pageLoads = map[int]*PageLoad{}
			}
			s.pageLoads[r.Seg] = clonePageLoad(r.PageLoad)
		}
		if r.Why != SegResume || r.Tab.ID != s.Tab.ID {
			s.Sketches.interrupt(SketchInterruptNavigation)
		}
		s.resetDOM()
		s.Seg, s.Why, s.Live, s.Tab, s.VP = r.Seg, r.Why, true, r.Tab, r.VP
		if r.Why != SegResume || s.DocSeg == 0 {
			s.DocSeg = r.Seg
		}
		s.URL, s.Title, s.awaitTop = r.Tab.URL, r.Tab.Title, true
	case *Stop:
		if !s.Live {
			return errors.New("stop without a live segment")
		}
		if r.Why == StopPause {
			s.Sketches.pause()
		} else {
			s.Sketches.interrupt(SketchInterruptEnd)
		}
		s.Live, s.awaitTop = false, false
		if r.Why == StopEnd {
			s.Control, s.Ended = Control{}, true
		}
	case *Ctl:
		return s.applyCtl(r)
	case *AnnotationClear:
		s.Sketches.clear()
		s.Notes, s.Strokes = nil, nil
	case *Doc:
		return s.applyDoc(r)
	case *Asset:
		s.Assets[r.URL] = r
	case *Note:
		s.Notes = append(s.Notes, &NoteView{Note: r, State: NoteOpen, Seg: s.Seg})
	case *NoteEdit:
		// Catch-up carries historical notes under a bare live seg before the
		// latest resume snapshot, so live+DocSeg (not Top) is the replay guard.
		if !s.Live || s.Ended {
			return errors.New("note edit requires a live segment")
		}
		for _, n := range s.Notes {
			if n.ID == r.ID {
				if n.State != NoteOpen {
					return errors.New("note edit requires an open note")
				}
				edited := *n.Note
				edited.Text, edited.Edited = r.Text, true
				n.Note = &edited
				n.Revision++
				return nil
			}
		}
		return fmt.Errorf("note %q is unknown", r.ID)
	case *NoteState:
		for _, n := range s.Notes {
			if n.ID == r.ID {
				n.State = r.State
				n.ResolvedBy = nil
				if r.State == NoteResolved && r.Who != nil {
					who := *r.Who
					n.ResolvedBy = &who
				}
				n.Revision++
				return nil
			}
		}
		return fmt.Errorf("note %q is unknown", r.ID)
	case *Stroke:
		s.Strokes = append(s.Strokes, &StrokeView{Stroke: r, Seg: s.Seg})
	case *Frame:
		s.Frames = append(s.Frames, r)
	case *Unknown:
	default:
		if err := s.inSegment(); err != nil {
			return err
		}
		if err := s.checkSource(r); err != nil {
			return err
		}
		return s.applyInSegment(r)
	}
	return nil
}

func (s *State) applyDoc(r *Doc) error {
	if r.Host == 0 {
		if !s.awaitTop {
			return errors.New("a top doc comes only right after seg")
		}
		if err := s.checkNew(&r.Tree); err != nil {
			return err
		}
		s.Top = s.build(&r.Tree, nil)
		s.Top.URL, s.URL, s.awaitTop = r.URL, r.URL, false
		return nil
	}
	if err := s.inSegment(); err != nil {
		return err
	}
	host, err := s.element(r.Host)
	if err != nil {
		return err
	}
	if host.N != "iframe" && host.N != "frame" {
		return fmt.Errorf("host %d is a %s, not an iframe", r.Host, host.N)
	}
	if err := s.checkNew(&r.Tree, host.Frame); err != nil { // nil Frame: none replaced
		return err
	}
	if host.Frame != nil {
		s.forget(host.Frame)
	}
	host.Frame = s.build(&r.Tree, nil)
	host.Frame.Host, host.Frame.URL = host, r.URL
	return nil
}

func (s *State) applyInSegment(r Record) error {
	switch r := r.(type) {
	case *Ins:
		p, err := s.container(r.P)
		if err != nil {
			return err
		}
		at := 0
		if r.After != 0 {
			i := slices.IndexFunc(p.Kids, func(k *DOMNode) bool { return k.ID == r.After })
			if i < 0 {
				return fmt.Errorf("after %d is not a child of %d", r.After, r.P)
			}
			at = i + 1
		}
		if err := s.checkNew(&r.Node); err != nil {
			return err
		}
		n := s.build(&r.Node, p)
		p.Kids = slices.Insert(p.Kids, at, n)
	case *Rm:
		n := s.nodes[r.ID]
		if n == nil {
			return fmt.Errorf("node %d is unknown", r.ID)
		}
		if n.Y == YDoc {
			return fmt.Errorf("node %d is a document; documents go with seg or a new doc for their frame", r.ID)
		}
		s.detach(n)
		s.forget(n)
	case *Kids:
		p, err := s.container(r.P)
		if err != nil {
			return err
		}
		for i := range r.C {
			// Only p's children are replaced: its shadow root and
			// template content stay, so their ids stay taken.
			if err := s.checkNew(&r.C[i], p.Kids...); err != nil {
				return err
			}
		}
		if err := dupIDs(r.C, r.P); err != nil {
			return err
		}
		for _, k := range p.Kids {
			s.forget(k)
		}
		p.Kids = p.Kids[:0]
		for i := range r.C {
			p.Kids = append(p.Kids, s.build(&r.C[i], p))
		}
	case *Attr:
		n, err := s.element(r.ID)
		if err != nil {
			return err
		}
		if n.N == "input" && passwordWithValue(withAttr(n.A, r.Name, r.V)) {
			return fmt.Errorf("element %d would be a password input with a value attribute; a password is never recorded", r.ID)
		}
		if r.V == nil {
			delete(n.A, r.Name)
		} else {
			if n.A == nil {
				n.A = map[string]string{}
			}
			n.A[r.Name] = *r.V
		}
	case *Text:
		n := s.nodes[r.ID]
		if n == nil || (n.Y != YText && n.Y != YComment) {
			return fmt.Errorf("node %d is not a text or comment node", r.ID)
		}
		n.V = r.V
	case *Shadow:
		host, err := s.element(r.Host)
		if err != nil {
			return err
		}
		if host.Shadow != nil {
			return fmt.Errorf("element %d already has a shadow root", r.Host)
		}
		if err := s.checkNew(&r.Root); err != nil {
			return err
		}
		host.Shadow = s.build(&r.Root, nil)
		host.Shadow.Host = host
	case *Sheet:
		if r.Owner != 0 {
			o, err := s.element(r.Owner)
			if err != nil {
				return err
			}
			if o.N != "style" && o.N != "link" {
				return fmt.Errorf("sheet owner %d is a %s, not a style or link", r.Owner, o.N)
			}
		}
		if s.nodes[r.ID] != nil {
			return fmt.Errorf("id %d names a live node, not a sheet", r.ID)
		}
		if err := s.claim(r.ID, "sheet"); err != nil {
			return err
		}
		s.types[r.ID] = "sheet"
		s.Sheets[r.ID] = &SheetState{Owner: r.Owner, Text: r.Text}
	case *Adopt:
		n := s.nodes[r.ID]
		if n == nil || (n.Y != YDoc && n.Y != YShadow) {
			return fmt.Errorf("node %d is not a document or shadow root", r.ID)
		}
		for _, id := range r.Sheets {
			if s.Sheets[id] == nil {
				return fmt.Errorf("sheet %d is unknown", id)
			}
		}
		s.Adopted[r.ID] = slices.Clone(r.Sheets)
	case *Font:
		s.Fonts = append(s.Fonts, r)
	case *Input:
		el, err := s.element(r.ID)
		if err != nil {
			return err
		}
		if isPassword(el.N, el.A) && r.V != nil && *r.V != "" {
			return fmt.Errorf("element %d is a password input: its value is never recorded (secret instead)", r.ID)
		}
		s.Inputs[r.ID] = r
	case *Scroll:
		n := s.nodes[r.ID]
		if n == nil || (n.Y != YEl && n.Y != YDoc) {
			return fmt.Errorf("node %d is not an element or document", r.ID)
		}
		s.Scrolls[r.ID] = [2]float64{r.X, r.Y}
	case *Selection:
		for _, p := range []*Pos{r.Anchor, r.Focus} {
			if p != nil && s.nodes[NodeID(p[0])] == nil {
				return fmt.Errorf("selection node %d is unknown", p[0])
			}
		}
		s.Selection = r
		if r.Anchor == nil && r.Focus == nil {
			s.Selection = nil
		}
	case *Nav:
		s.URL = r.URL
	case *Title:
		s.Title = r.V
	case *VP:
		s.VP = r.Viewport
	case *Click:
		if s.nodes[r.ID] == nil {
			return fmt.Errorf("node %d is unknown", r.ID)
		}
		s.LastClick = r
	case *Focus:
		if r.ID != 0 && s.nodes[r.ID] == nil {
			return fmt.Errorf("node %d is unknown", r.ID)
		}
		s.Focus = r.ID
	case *Hover:
		if r.ID != 0 && s.nodes[r.ID] == nil {
			return fmt.Errorf("node %d is unknown", r.ID)
		}
		s.Hover = r.ID
	case *Ptr:
		s.Ptr = &[2]float64{r.X, r.Y}
	case *Layout:
		for id := range r.Boxes {
			if s.nodes[id] == nil {
				return fmt.Errorf("layout of unknown node %d", id)
			}
		}
		s.Boxes, s.BoxesT = make(map[NodeID]Box, len(r.Boxes)), r.T
		for id, b := range r.Boxes {
			s.Boxes[id] = b
		}
	default:
		return fmt.Errorf("kind %s has no meaning in a segment", r.Kind())
	}
	return nil
}

// element is the live element with that id.
func (s *State) element(id NodeID) (*DOMNode, error) {
	n := s.nodes[id]
	if n == nil || n.Y != YEl {
		return nil, fmt.Errorf("node %d is not a live element", id)
	}
	return n, nil
}

// container is a live node that can have children.
func (s *State) container(id NodeID) (*DOMNode, error) {
	n := s.nodes[id]
	if n == nil {
		return nil, fmt.Errorf("node %d is unknown", id)
	}
	switch n.Y {
	case YEl, YDoc, YShadow, YFrag:
		return n, nil
	}
	return nil, fmt.Errorf("node %d is a %s and has no children", id, n.Y)
}

// signature is what an id may never change: its type and tag.
func signature(n *Node) string { return n.Y + ":" + n.NS + ":" + n.N }

// claim records an id's signature for the recording, refusing a change.
func (s *State) claim(id NodeID, sig string) error {
	if was, ok := s.types[id]; ok && was != sig {
		return fmt.Errorf("id %d was %s and is now %s; an id names one thing", id, was, sig)
	}
	return nil
}

// checkNew checks a subtree about to be added: every id is new to the live
// DOM (or lives inside one of the subtrees it replaces), unique within the
// subtree, and keeps its recording-wide signature.
func (s *State) checkNew(n *Node, replacing ...*DOMNode) error {
	seen := map[NodeID]bool{}
	var walk func(n *Node) error
	walk = func(n *Node) error {
		if seen[n.ID] {
			return fmt.Errorf("id %d appears twice in one subtree", n.ID)
		}
		seen[n.ID] = true
		if live := s.nodes[n.ID]; live != nil && !underAny(live, replacing) {
			return fmt.Errorf("id %d is already live", n.ID)
		}
		if err := s.claim(n.ID, signature(n)); err != nil {
			return err
		}
		if n.Y == YEl && n.N == "input" && passwordWithValue(n.A) {
			return fmt.Errorf("node %d is a password input with a value attribute; a password is never recorded", n.ID)
		}
		for _, sub := range []*Node{n.SR, n.TC} {
			if sub != nil {
				if err := walk(sub); err != nil {
					return err
				}
			}
		}
		for i := range n.C {
			if err := walk(&n.C[i]); err != nil {
				return err
			}
		}
		return nil
	}
	return walk(n)
}

// dupIDs: no id twice among new children, and none is their parent's.
func dupIDs(c []Node, parent NodeID) error {
	seen := map[NodeID]bool{parent: true}
	var walk func(n *Node) error
	walk = func(n *Node) error {
		if seen[n.ID] {
			return fmt.Errorf("id %d appears twice", n.ID)
		}
		seen[n.ID] = true
		for _, sub := range []*Node{n.SR, n.TC} {
			if sub != nil {
				if err := walk(sub); err != nil {
					return err
				}
			}
		}
		for i := range n.C {
			if err := walk(&n.C[i]); err != nil {
				return err
			}
		}
		return nil
	}
	for i := range c {
		if err := walk(&c[i]); err != nil {
			return err
		}
	}
	return nil
}

func underAny(n *DOMNode, roots []*DOMNode) bool {
	for _, r := range roots {
		if under(n, r) {
			return true
		}
	}
	return false
}

// under reports whether n is inside root's subtree (through children, shadow
// roots, template contents and frame documents), root included.
func under(n, root *DOMNode) bool {
	if root == nil {
		return false
	}
	for ; n != nil; n = up(n) {
		if n == root {
			return true
		}
	}
	return false
}

func up(n *DOMNode) *DOMNode {
	if n.Parent != nil {
		return n.Parent
	}
	return n.Host
}

// build adds a checked subtree to the live DOM under parent.
func (s *State) build(n *Node, parent *DOMNode) *DOMNode {
	d := &DOMNode{ID: n.ID, Y: n.Y, N: n.N, NS: n.NS, Mode: n.Mode, Parent: parent}
	if n.V != nil {
		d.V = *n.V
	}
	if len(n.A) > 0 {
		d.A = make(map[string]string, len(n.A))
		for k, v := range n.A {
			d.A[k] = v
		}
	}
	s.nodes[n.ID] = d
	s.types[n.ID] = signature(n)
	if n.SR != nil {
		d.Shadow = s.build(n.SR, nil)
		d.Shadow.Host = d
	}
	if n.TC != nil {
		d.Content = s.build(n.TC, nil)
		d.Content.Host = d
	}
	for i := range n.C {
		d.Kids = append(d.Kids, s.build(&n.C[i], d))
	}
	return d
}

// detach takes n out of whatever holds it.
func (s *State) detach(n *DOMNode) {
	switch {
	case n.Parent != nil:
		p := n.Parent
		p.Kids = slices.DeleteFunc(p.Kids, func(k *DOMNode) bool { return k == n })
	case n.Host != nil && n.Host.Shadow == n:
		n.Host.Shadow = nil
	case n.Host != nil && n.Host.Content == n:
		n.Host.Content = nil
	}
}

// forget drops n's subtree from the id index and every per-node state.
func (s *State) forget(n *DOMNode) {
	if s.nodes[n.ID] == n {
		delete(s.nodes, n.ID)
	}
	delete(s.Inputs, n.ID)
	delete(s.Scrolls, n.ID)
	delete(s.Boxes, n.ID)
	delete(s.Adopted, n.ID)
	if n.N == "style" || n.N == "link" { // the only owners: no scan per node
		for id, sh := range s.Sheets {
			if sh.Owner == n.ID {
				delete(s.Sheets, id)
			}
		}
	}
	if s.Focus == n.ID {
		s.Focus = 0
	}
	if s.Hover == n.ID {
		s.Hover = 0
	}
	for _, sub := range []*DOMNode{n.Shadow, n.Content, n.Frame} {
		if sub != nil {
			s.forget(sub)
		}
	}
	for _, k := range n.Kids {
		s.forget(k)
	}
}

// isPassword: an input element whose type is password (any case).
func isPassword(tag string, a map[string]string) bool {
	return tag == "input" && strings.EqualFold(strings.TrimSpace(a["type"]), "password")
}

// passwordWithValue: an input's attributes make it a password input with a
// non-empty value attribute.
func passwordWithValue(a map[string]string) bool {
	return isPassword("input", a) && a["value"] != ""
}

// withAttr is a's attributes with one set (v nil: removed), without
// changing a.
func withAttr(a map[string]string, name string, v *string) map[string]string {
	out := make(map[string]string, len(a)+1)
	for k, val := range a {
		out[k] = val
	}
	if v == nil {
		delete(out, name)
	} else {
		out[name] = *v
	}
	return out
}
