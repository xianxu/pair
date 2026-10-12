package main

import (
	"fmt"
	"strings"

	rec "github.com/xianxu/pair/cmd/internal/webcast/recording"
)

var (
	pm     = rec.Author{ID: "viewer-1", Name: "Pat (product)", Color: "#e8590c"}
	design = rec.Author{ID: "viewer-2", Name: "Ana (design)", Color: "#1c7ed6"}
	// driver is rec-driven's driver (#467): her seat's id, v1, so her note
	// is attributable to the driver.
	driver = rec.Author{ID: "v1", Name: "Ana", Color: "#2f9e44"}
)

// noteSpec places a note (and a stroke around its element) on a fixture:
// at t, on the first element matching sel. sel is "#id", "tag",
// "tag~text" (the first such element whose text contains text), or
// "tag[attr*=value]".
type noteSpec struct {
	t        int64
	sel      string
	author   rec.Author
	text     string
	resolveT int64 // when nonzero, a note-state resolves it then
}

var specs = map[string][]noteSpec{
	// The real recorder's runs (#472): #453's pages, at the moments their
	// scenarios reach those elements.
	"rec-stress": {
		{t: 2000, sel: "#cij", author: pm, text: "This box flickers through colours; is that on purpose?"},
		{t: 3000, sel: "#closed", author: design, text: "Closed shadow root: the badge should match the open one.", resolveT: 4500},
	},
	"rec-app": {
		{t: 6000, sel: "#search", author: pm, text: "Search should ignore case."},
		{t: 9800, sel: "#close-modal", author: design, text: "The modal's close button needs more contrast.", resolveT: 11000},
		{t: 19000, sel: "h1~Reports", author: pm, text: "Reports needs a date range."},
	},
	"rec-blog": {
		{t: 2500, sel: "h1", author: pm, text: "Title wraps awkwardly at this width."},
		{t: 6000, sel: "img[alt*=issue tracker]", author: design, text: "This screenshot is too small to read."},
		{t: 8000, sel: "ol", author: pm, text: "This list reads better as a table."},
	},
	// A recorded remote-driving session (#467): Ana drives from 1.4 s to
	// 7.9 s and from 9.9 s to 11 s.
	"rec-driven": {
		{t: 3000, sel: "#q", author: driver, text: "Searching by number works; can it take a range?"},
		{t: 6800, sel: "#approve", author: pm, text: "Approve should ask for a reason.", resolveT: 8200},
	},
	"rec-segments": {
		{t: 1500, sel: "h1", author: pm, text: "Reports needs a date range."},
		{t: 4200, sel: "td~Week 3", author: design, text: "Week 3 looks low; is that a gap in the data?"},
	},
}

// annotate makes the notes and strokes of a fixture from its spec, each
// anchored where the element was at that moment.
func annotate(name string, recs []rec.Record) ([]rec.Record, error) {
	var out []rec.Record
	for i, sp := range specs[name] {
		st, err := rec.Reduce(recs, sp.t)
		if err != nil {
			return nil, err
		}
		el := match(st, sp.sel)
		if el == nil {
			return nil, fmt.Errorf("%s note %d: nothing matches %q at t=%d", name, i, sp.sel, sp.t)
		}
		at := rec.NoteAt{Node: el.ID, Sel: sp.sel, Text: trim(el.TextContent())}
		if b, ok := st.Boxes[el.ID]; ok {
			at.Box = &b
		}
		id := fmt.Sprintf("%s-n%d", name, i+1)
		n := &rec.Note{ID: id, Author: sp.author, Text: sp.text, At: at}
		n.SetTime(sp.t)
		out = append(out, n)
		if at.Box != nil {
			b := *at.Box
			s := &rec.Stroke{ID: id + "-s", Author: sp.author, Color: sp.author.Color,
				Pts: [][3]float64{{b[0], b[1], 0}, {b[0] + b[2], b[1], 80}, {b[0] + b[2], b[1] + b[3], 160}, {b[0], b[1] + b[3], 240}}}
			s.SetTime(sp.t - 400)
			out = append(out, s)
		}
		if sp.resolveT != 0 {
			ns := &rec.NoteState{ID: id, State: rec.NoteResolved}
			ns.SetTime(sp.resolveT)
			out = append(out, ns)
		}
	}
	return out, nil
}

func match(st *rec.State, sel string) *rec.DOMNode {
	if st.Top == nil {
		return nil
	}
	tag, text, _ := strings.Cut(sel, "~")
	id, byID := strings.CutPrefix(sel, "#")
	var attrName, attrSub string
	if t, rest, ok := strings.Cut(tag, "["); ok {
		tag = t
		attrName, attrSub, _ = strings.Cut(strings.TrimSuffix(rest, "]"), "*=")
	}
	var found *rec.DOMNode
	var walk func(n *rec.DOMNode)
	walk = func(n *rec.DOMNode) {
		if found != nil {
			return
		}
		if n.Y == rec.YEl {
			switch {
			case byID && n.A["id"] == id:
				found = n
			case !byID && n.N == tag && strings.Contains(n.TextContent(), text) &&
				(attrName == "" || strings.Contains(n.A[attrName], attrSub)):
				found = n
			}
		}
		if n.Shadow != nil {
			walk(n.Shadow)
		}
		for _, k := range n.Kids {
			walk(k)
		}
	}
	walk(st.Top)
	return found
}

func trim(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > 200 {
		return string(r[:200])
	}
	return s
}

// segmentsFixture is hand-made: go live, type, pause, act while paused,
// resume on the same document (ids kept), switch tabs, come back by
// navigating, end. Notes span the resume and clear on the switch.
func segmentsFixture() []rec.Record {
	b := rec.NewBuilder("segments")
	input := b.El("input", []string{"id", "q"})
	list := b.El("ul", []string{"id", "list"}, b.El("li", nil, b.Txt("one")))
	doc := b.Page(rec.SegLive, "http://app.local/", b.El("h1", nil, b.Txt("Inbox")), input, list)
	b.At(500).Add(&rec.Layout{Boxes: map[rec.NodeID]rec.Box{input.ID: {16, 60, 200, 24}, list.ID: {16, 100, 600, 40}}})
	b.At(800).Add(&rec.Focus{ID: input.ID})
	b.At(1200).Add(&rec.Input{ID: input.ID, V: rec.Str("inv")})
	n1 := &rec.Note{ID: "seg-n1", Author: pm, Text: "Typing here lags.", At: rec.NoteAt{Node: input.ID, Sel: "#q", Box: &rec.Box{16, 60, 200, 24}}}
	b.At(1500).Add(n1)
	b.At(2000).Add(&rec.Stop{Why: rec.StopPause})
	// Paused: the operator adds an item. Resume snapshots the same
	// document again, keeping every id that persisted.
	two := b.El("li", nil, b.Txt("two"))
	b.At(5000).Add(&rec.Seg{Seg: b.NextSeg(), Why: rec.SegResume, Tab: rec.Tab{ID: "tab1", URL: "http://app.local/"}, VP: rec.Viewport{W: 1280, H: 800, DPR: 1}})
	html := clone(doc.C[1]) // the first snapshot's tree stays as recorded
	body := &html.C[1]
	body.C[2].C = append(body.C[2].C, two)
	b.Snapshot("http://app.local/", html)
	b.At(5200).Add(&rec.Input{ID: input.ID, V: rec.Str("invoice")})
	b.At(5500).Add(&rec.Note{ID: "seg-n2", Author: design, Text: "Second item: wrong icon.", At: rec.NoteAt{Node: two.ID}})
	// The operator switches to another tab: a new document, notes clear.
	b.At(7000)
	other := b.El("p", nil, b.Txt("Settings"))
	b.Add(&rec.Seg{Seg: b.NextSeg(), Why: rec.SegTab, Tab: rec.Tab{ID: "tab2", URL: "http://app.local/settings", Title: "Settings"}, VP: rec.Viewport{W: 1280, H: 800, DPR: 1}})
	b.Snapshot("http://app.local/settings", b.El("html", nil, b.El("head", nil), b.El("body", nil, other)))
	b.At(7500).Add(&rec.Title{V: "Settings"})
	b.At(8000).Add(&rec.Nav{URL: "http://app.local/settings#privacy"})
	b.At(9000).Add(&rec.Stop{Why: rec.StopEnd})
	return b.Recs
}

// clone deep-copies a wire tree.
func clone(n rec.Node) rec.Node {
	c := n
	if n.A != nil {
		c.A = map[string]string{}
		for k, v := range n.A {
			c.A[k] = v
		}
	}
	c.C = nil
	for _, k := range n.C {
		c.C = append(c.C, clone(k))
	}
	if n.SR != nil {
		sr := clone(*n.SR)
		c.SR = &sr
	}
	if n.TC != nil {
		tc := clone(*n.TC)
		c.TC = &tc
	}
	return c
}
