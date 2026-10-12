package recording

import (
	"bytes"
	"strings"
	"testing"
)

// page is a one-segment recording: <body><div id=a>hi</div><ul></ul></body>.
type page struct {
	b                        *Builder
	doc, html, body, div, ul Node
}

func newPage() *page {
	b := NewBuilder("t")
	div := b.El("div", []string{"id", "a"}, b.Txt("hi"))
	ul := b.El("ul", nil)
	doc := b.Page(SegLive, "http://x/", div, ul)
	html := doc.C[1]
	return &page{b: b, doc: doc, html: html, body: html.C[1], div: div, ul: ul}
}

func (p *page) reduce(t *testing.T) *State {
	t.Helper()
	s, err := Reduce(p.b.Recs, 1<<62)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// mustFail applies r to the page's state and wants an error that leaves the
// state byte-identical.
func mustFail(t *testing.T, s *State, r Record, want string) {
	t.Helper()
	before := s.Canonical()
	r.header().T = s.T
	err := s.Apply(r)
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("%s: got %v, want an error containing %q", r.Kind(), err, want)
	}
	if !bytes.Equal(before, s.Canonical()) {
		t.Fatalf("%s: a failed record changed the state", r.Kind())
	}
}

func kidIDs(n *DOMNode) []NodeID {
	var ids []NodeID
	for _, k := range n.Kids {
		ids = append(ids, k.ID)
	}
	return ids
}

func TestInsertPositions(t *testing.T) {
	p := newPage()
	a, b, c := p.b.El("li", nil), p.b.El("li", nil), p.b.El("li", nil)
	p.b.Add(&Ins{P: p.ul.ID, Node: b})              // [b]
	p.b.Add(&Ins{P: p.ul.ID, Node: a})              // first: [a b]
	p.b.Add(&Ins{P: p.ul.ID, After: b.ID, Node: c}) // last: [a b c]
	s := p.reduce(t)
	if got, want := kidIDs(s.Node(p.ul.ID)), []NodeID{a.ID, b.ID, c.ID}; !equalIDs(got, want) {
		t.Fatalf("children %v, want %v", got, want)
	}
	mustFail(t, s, &Ins{P: p.ul.ID, After: p.div.ID, Node: p.b.El("li", nil)}, "not a child")
	mustFail(t, s, &Ins{P: p.ul.ID, Node: Node{ID: a.ID, Y: YEl, N: "li"}}, "already live")
	mustFail(t, s, &Ins{P: 999, Node: p.b.El("li", nil)}, "unknown")
	mustFail(t, s, &Ins{P: p.div.C[0].ID, Node: p.b.El("i", nil)}, "has no children")
}

func equalIDs(a, b []NodeID) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestRemoveDropsSubtreeAndItsState(t *testing.T) {
	p := newPage()
	in := p.b.El("input", nil)
	p.b.Add(&Ins{P: p.div.ID, Node: in})
	p.b.Add(&Input{ID: in.ID, V: Str("typed")})
	p.b.Add(&Scroll{ID: p.div.ID, Y: 40})
	p.b.Add(&Focus{ID: in.ID})
	p.b.Add(&Rm{ID: p.div.ID})
	s := p.reduce(t)
	for _, id := range []NodeID{p.div.ID, p.div.C[0].ID, in.ID} {
		if s.Node(id) != nil {
			t.Errorf("node %d still live", id)
		}
	}
	if len(s.Inputs) != 0 || len(s.Scrolls) != 0 || s.Focus != 0 {
		t.Errorf("per-node state survived removal: inputs %v scrolls %v focus %d", s.Inputs, s.Scrolls, s.Focus)
	}
	mustFail(t, s, &Rm{ID: p.div.ID}, "unknown")
	mustFail(t, s, &Rm{ID: p.doc.ID}, "document")
}

func TestMoveKeepsID(t *testing.T) {
	p := newPage()
	p.b.Add(&Rm{ID: p.div.ID})
	p.b.Add(&Ins{P: p.ul.ID, Node: p.div})
	s := p.reduce(t)
	if n := s.Node(p.div.ID); n == nil || n.Parent.ID != p.ul.ID {
		t.Fatalf("moved node: %+v", n)
	}
}

func TestAnIDNeverChangesWhatItNames(t *testing.T) {
	p := newPage()
	p.b.Add(&Rm{ID: p.div.ID})
	s := p.reduce(t)
	mustFail(t, s, &Ins{P: p.ul.ID, Node: Node{ID: p.div.ID, Y: YEl, N: "span"}}, "names one thing")
	mustFail(t, s, &Ins{P: p.ul.ID, Node: Node{ID: p.div.C[0].ID, Y: YComment, V: Str("x")}}, "names one thing")
}

func TestKidsReplaces(t *testing.T) {
	p := newPage()
	old := p.b.El("li", nil)
	p.b.Add(&Ins{P: p.ul.ID, Node: old})
	n1, n2 := p.b.El("li", nil, p.b.Txt("1")), p.b.El("li", nil)
	p.b.Add(&Kids{P: p.ul.ID, C: []Node{n1, n2, old}}) // old reused: allowed under its parent
	s := p.reduce(t)
	if got := kidIDs(s.Node(p.ul.ID)); !equalIDs(got, []NodeID{n1.ID, n2.ID, old.ID}) {
		t.Fatalf("children %v", got)
	}
	mustFail(t, s, &Kids{P: p.ul.ID, C: []Node{{ID: p.ul.ID, Y: YEl, N: "ul"}}}, "already live")
	mustFail(t, s, &Kids{P: p.ul.ID, C: []Node{{ID: p.div.ID, Y: YEl, N: "div"}}}, "already live")
}

func TestAttrAndText(t *testing.T) {
	p := newPage()
	p.b.Add(&Attr{ID: p.div.ID, Name: "class", V: Str("x")})
	p.b.Add(&Attr{ID: p.div.ID, Name: "id"})
	p.b.Add(&Text{ID: p.div.C[0].ID, V: "bye"})
	s := p.reduce(t)
	d := s.Node(p.div.ID)
	if d.A["class"] != "x" || d.A["id"] != "" || len(d.A) != 1 || d.Kids[0].V != "bye" {
		t.Fatalf("got %+v / %q", d.A, d.Kids[0].V)
	}
	mustFail(t, s, &Text{ID: p.div.ID, V: "x"}, "not a text")
	mustFail(t, s, &Attr{ID: p.div.C[0].ID, Name: "a", V: Str("b")}, "not a live element")
}

func TestShadowRootsAndTemplates(t *testing.T) {
	p := newPage()
	root := Node{ID: p.b.ID(), Y: YShadow, Mode: "closed", C: []Node{p.b.Txt("inside")}}
	p.b.Add(&Shadow{Host: p.div.ID, Root: root})
	tpl := p.b.El("template", nil)
	tpl.TC = &Node{ID: p.b.ID(), Y: YFrag, C: []Node{p.b.El("li", nil)}}
	p.b.Add(&Ins{P: p.body.ID, After: p.ul.ID, Node: tpl})
	p.b.Add(&Ins{P: root.ID, After: root.C[0].ID, Node: p.b.El("b", nil)})
	s := p.reduce(t)
	sr := s.Node(p.div.ID).Shadow
	if sr == nil || sr.Mode != "closed" || len(sr.Kids) != 2 || sr.Host.ID != p.div.ID {
		t.Fatalf("shadow root: %+v", sr)
	}
	if c := s.Node(tpl.ID).Content; c == nil || len(c.Kids) != 1 {
		t.Fatalf("template content: %+v", c)
	}
	mustFail(t, s, &Shadow{Host: p.div.ID, Root: Node{ID: p.b.ID(), Y: YShadow, Mode: "open"}}, "already has")
	p.b.Add(&Rm{ID: p.div.ID})
	if s = p.reduce(t); s.Node(root.ID) != nil || s.Node(root.C[0].ID) != nil {
		t.Error("removing a host must drop its shadow tree")
	}
}

func TestFrameDocuments(t *testing.T) {
	p := newPage()
	frame := p.b.El("iframe", []string{"src", "http://y/"})
	p.b.Add(&Ins{P: p.body.ID, Node: frame})
	inner := p.b.El("p", nil, p.b.Txt("framed"))
	d1 := Node{ID: p.b.ID(), Y: YDoc, C: []Node{p.b.El("html", nil, p.b.El("body", nil, inner))}}
	p.b.Add(&Doc{Doc: d1.ID, Host: frame.ID, URL: "http://y/", Tree: d1})
	p.b.Add(&Text{ID: inner.C[0].ID, V: "changed"})
	s := p.reduce(t)
	f := s.Node(frame.ID).Frame
	if f == nil || f.URL != "http://y/" || f.Host.ID != frame.ID || s.Node(inner.C[0].ID).V != "changed" {
		t.Fatalf("frame doc: %+v", f)
	}
	// The frame navigates: the new document replaces the old one.
	d2 := Node{ID: p.b.ID(), Y: YDoc}
	p.b.Add(&Doc{Doc: d2.ID, Host: frame.ID, URL: "http://y/2", Tree: d2})
	s = p.reduce(t)
	if s.Node(d1.ID) != nil || s.Node(inner.ID) != nil || s.Node(frame.ID).Frame.ID != d2.ID {
		t.Fatal("the old frame document must be gone")
	}
	mustFail(t, s, &Doc{Doc: p.b.ID(), Host: p.div.ID, Tree: Node{ID: p.b.next, Y: YDoc}}, "not an iframe")
}

func TestSegmentsStopAndResume(t *testing.T) {
	p := newPage()
	p.b.Add(&Asset{URL: "http://x/a.css", Hash: strings.Repeat("0", 64), MIME: "text/css"})
	p.b.Add(&Stop{Why: StopPause})
	s := p.reduce(t)
	if s.Live || s.Node(p.div.ID) == nil {
		t.Fatal("a stop freezes the state as it was, not live")
	}
	mustFail(t, s, &Attr{ID: p.div.ID, Name: "x", V: Str("y")}, "no live segment")
	mustFail(t, s, &Doc{Doc: p.b.ID(), Tree: Node{ID: p.b.next, Y: YDoc}}, "only right after seg")

	// Resume: same document, ids kept for persisting nodes.
	p.b.Add(&Seg{Seg: 2, Why: SegResume, Tab: Tab{ID: "tab1", URL: "http://x/"}, VP: Viewport{W: 1280, H: 800}})
	s = p.reduce(t)
	if s.Node(p.div.ID) != nil || s.Top != nil {
		t.Fatal("a seg drops the DOM")
	}
	if s.Assets["http://x/a.css"] == nil {
		t.Fatal("a seg keeps the asset table")
	}
	mustFail(t, s, &Attr{ID: p.div.ID, Name: "x", V: Str("y")}, "no live segment")
	p.b.Snapshot("http://x/", p.html)
	s = p.reduce(t)
	if s.Seg != 2 || !s.Live || s.Node(p.div.ID) == nil {
		t.Fatal("resume with the same ids")
	}
	mustFail(t, s, &Seg{Seg: 2, Why: SegLive, VP: Viewport{W: 1, H: 1}}, "segment 2 after segment 2")
}

func TestHeadAndTime(t *testing.T) {
	s := NewState()
	if err := s.Apply(&Seg{Seg: 1, Why: SegLive, VP: Viewport{W: 1, H: 1}}); err == nil {
		t.Error("the first record must be head")
	}
	if err := s.Apply(&Head{Format: Format, V: 2, ID: "x", Started: "s"}); err == nil {
		t.Error("an unknown major version must be refused")
	}
	p := newPage()
	s = p.reduce(t)
	mustFail(t, s, &Head{Format: Format, V: 1, ID: "x", Started: "s"}, "second head")
	r := &Title{V: "x"}
	r.T = s.T - 1
	if err := s.Apply(r); err == nil {
		t.Error("time going backwards must be refused")
	}
}

func TestStylesheets(t *testing.T) {
	p := newPage()
	style := p.b.El("style", nil, p.b.Txt(""))
	p.b.Add(&Ins{P: p.html.C[0].ID, Node: style})
	p.b.Add(&Sheet{ID: p.b.ID(), Owner: style.ID, Text: ".a{color:red}"})
	cs := p.b.ID()
	p.b.Add(&Sheet{ID: cs, Text: "#a{border:1px solid}"})
	p.b.Add(&Adopt{ID: p.doc.ID, Sheets: []NodeID{cs}})
	p.b.Add(&Sheet{ID: cs, Text: "#a{border:2px solid}"})
	s := p.reduce(t)
	c := string(s.Canonical())
	for _, want := range []string{`"$sheet": ".a{color:red}"`, `"#a{border:2px solid}"`} {
		if !strings.Contains(c, want) {
			t.Errorf("canonical lacks %s", want)
		}
	}
	mustFail(t, s, &Sheet{ID: p.b.ID(), Owner: p.div.ID, Text: ""}, "not a style or link")
	mustFail(t, s, &Adopt{ID: p.doc.ID, Sheets: []NodeID{4242}}, "unknown")
	mustFail(t, s, &Adopt{ID: p.div.ID}, "not a document or shadow root")
	mustFail(t, s, &Sheet{ID: p.div.ID, Text: ""}, "live node")
}

func TestObserverAndContext(t *testing.T) {
	p := newPage()
	chk := p.b.El("input", []string{"type", "checkbox"})
	p.b.Add(&Ins{P: p.body.ID, Node: chk})
	yes := true
	p.b.Add(&Input{ID: chk.ID, Checked: &yes})
	p.b.Add(&Scroll{ID: p.doc.ID, Y: 300})
	p.b.Add(&Selection{Anchor: &Pos{int64(p.div.C[0].ID), 0}, Focus: &Pos{int64(p.div.C[0].ID), 2}})
	p.b.Add(&Click{ID: p.div.ID, X: 10, Y: 20})
	p.b.Add(&Nav{URL: "http://x/#two"})
	p.b.Add(&Title{V: "Two"})
	p.b.Add(&VP{Viewport: Viewport{W: 1000, H: 700, DPR: 2}})
	p.b.Add(&Layout{Boxes: map[NodeID]Box{p.div.ID: {0, 0, 100, 20}}})
	s := p.reduce(t)
	if !*s.Inputs[chk.ID].Checked || s.Scrolls[p.doc.ID][1] != 300 || s.Selection == nil ||
		s.LastClick.ID != p.div.ID || s.URL != "http://x/#two" || s.Title != "Two" || s.VP.W != 1000 ||
		s.Boxes[p.div.ID][2] != 100 {
		t.Fatalf("state: %+v", s)
	}
	p.b.Add(&Selection{})
	if s = p.reduce(t); s.Selection != nil {
		t.Error("an empty selection clears it")
	}
	p.b.Add(&Layout{Boxes: map[NodeID]Box{p.ul.ID: {0, 20, 100, 0}}})
	if s = p.reduce(t); len(s.Boxes) != 1 || s.Boxes[p.ul.ID] != (Box{0, 20, 100, 0}) {
		t.Errorf("layout replaces the previous boxes: %v", s.Boxes)
	}
	mustFail(t, s, &Input{ID: 999, V: Str("x")}, "not a live element")
	mustFail(t, s, &Scroll{ID: p.div.C[0].ID}, "not an element or document")
	mustFail(t, s, &Selection{Anchor: &Pos{999, 0}}, "unknown")
	mustFail(t, s, &Focus{ID: 999}, "unknown")
	mustFail(t, s, &Layout{Boxes: map[NodeID]Box{999: {}}}, "unknown")
}

func TestNotesOutliveTheirElementsAndSegments(t *testing.T) {
	p := newPage()
	p.b.Add(&Note{ID: "n1", Author: Author{ID: "a"}, Text: "this", At: NoteAt{Node: p.div.ID}})
	p.b.Add(&Rm{ID: p.div.ID})
	p.b.Add(&NoteState{ID: "n1", State: NoteResolved})
	p.b.Add(&Stroke{ID: "s1", Author: Author{ID: "a"}, Pts: [][3]float64{{1, 1, 0}}})
	p.b.Add(&Stop{Why: StopEnd})
	p.b.Add(&Note{ID: "n2", Author: Author{ID: "a"}, At: NoteAt{Node: 4242}})
	s := p.reduce(t)
	if len(s.Notes) != 2 || s.Notes[0].State != NoteResolved || s.Notes[0].Seg != 1 || len(s.Strokes) != 1 {
		t.Fatalf("notes %+v", s.Notes)
	}
	mustFail(t, s, &NoteState{ID: "nope", State: NoteOpen}, "unknown")
}

func TestReduceStopsAtT(t *testing.T) {
	p := newPage()
	p.b.At(100).Add(&Title{V: "later"})
	s, err := Reduce(p.b.Recs, 99)
	if err != nil || s.Title != "" {
		t.Fatalf("t=99: %q %v", s.Title, err)
	}
	if s, _ = Reduce(p.b.Recs, 100); s.Title != "later" {
		t.Fatalf("t=100: %q", s.Title)
	}
}

func TestUnknownKindsAreIgnored(t *testing.T) {
	p := newPage()
	r, _ := Decode([]byte(`{"t":0,"k":"hover","id":3}`))
	p.b.Recs = append(p.b.Recs, r)
	p.reduce(t)
}

func TestCanonicalIsDeterministicAndComplete(t *testing.T) {
	p := newPage()
	frame := p.b.El("iframe", nil)
	p.b.Add(&Ins{P: p.body.ID, Node: frame})
	fd := Node{ID: p.b.ID(), Y: YDoc, C: []Node{p.b.El("html", []string{"b", "2", "a", "1"})}}
	p.b.Add(&Doc{Doc: fd.ID, Host: frame.ID, URL: "http://y/", Tree: fd})
	p.b.Add(&Input{ID: p.div.ID, V: Str("v")})
	s := p.reduce(t)
	a, b := s.Canonical(), s.Canonical()
	if !bytes.Equal(a, b) {
		t.Fatal("canonical must be deterministic")
	}
	for _, want := range []string{`"frame": {`, `"url": "http://y/"`, `"$input": {`} {
		if !bytes.Contains(a, []byte(want)) {
			t.Errorf("canonical lacks %q:\n%s", want, a)
		}
	}
	if i, j := bytes.Index(a, []byte(`"a": "1"`)), bytes.Index(a, []byte(`"b": "2"`)); i < 0 || j < i {
		t.Error("attributes must be in name order")
	}
}

// One namespace: a sheet id is never a node's, and a node's never a sheet's.
func TestSheetAndNodeIDsShareOneNamespace(t *testing.T) {
	p := newPage()
	sid := p.b.ID()
	p.b.Add(&Sheet{ID: sid, Text: "a{}"})
	s := p.reduce(t)
	mustFail(t, s, &Ins{P: p.ul.ID, Node: Node{ID: sid, Y: YEl, N: "li"}}, "names one thing")
	mustFail(t, s, &Sheet{ID: p.div.ID, Text: ""}, "live node")
}

// kids replaces p's children only: ids in p's shadow root or template
// content stay taken.
func TestKidsCannotReuseIDsOutsideTheReplacedChildren(t *testing.T) {
	p := newPage()
	inner := p.b.Txt("in shadow")
	p.b.Add(&Shadow{Host: p.ul.ID, Root: Node{ID: p.b.ID(), Y: YShadow, Mode: "open", C: []Node{inner}}})
	s := p.reduce(t)
	mustFail(t, s, &Kids{P: p.ul.ID, C: []Node{inner}}, "already live")
}
