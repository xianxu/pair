package notes

import (
	"strings"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// contractPage is an article page with layout: a header with a nav, an
// article with a heading, two paragraphs and a script, and a footer far
// below.
type contractPage struct {
	b                     *recording.Builder
	nav, h1, p1, p2, foot recording.Node
	boxes                 map[recording.NodeID]recording.Box
}

func newContractPage() *contractPage {
	b := recording.NewBuilder("notes-contract")
	c := &contractPage{b: b}
	c.nav = b.El("nav", nil, b.El("a", []string{"href", "/"}, b.Txt("Home")))
	c.h1 = b.El("h1", nil, b.Txt("A post"))
	c.p1 = b.El("p", []string{"id", "first"}, b.Txt("The first paragraph."))
	c.p2 = b.El("p", nil, b.Txt(strings.Repeat("The second paragraph is long. ", 12)))
	c.foot = b.El("p", []string{"id", "foot"}, b.Txt("Footer"))
	header := b.El("header", nil, c.nav)
	script := b.El("script", nil, b.Txt("render('first paragraph')"))
	article := b.El("article", nil, c.h1, c.p1, c.p2, script)
	footer := b.El("footer", nil, c.foot)
	b.Page(recording.SegLive, "http://blog/post", header, b.El("main", nil, article), footer)
	c.boxes = map[recording.NodeID]recording.Box{
		header.ID: {0, 0, 1280, 64}, c.nav.ID: {0, 0, 1280, 64},
		article.ID: {240, 100, 800, 400}, c.h1.ID: {240, 100, 800, 60},
		c.p1.ID: {240, 180, 800, 120}, c.p2.ID: {240, 320, 800, 120},
		footer.ID: {0, 3000, 1280, 80}, c.foot.ID: {240, 3020, 800, 40},
	}
	b.Add(&recording.Layout{Boxes: c.boxes})
	return c
}

func (c *contractPage) state(t recording.TB) *recording.State {
	t.Helper()
	st, err := recording.Reduce(c.b.Recs, 1<<62)
	if err != nil {
		t.Fatalf("contract page: %v", err)
	}
	return st
}

// RunModelContract checks a Model. The real model (#461) and FakeModel both
// run it.
func RunModelContract(t recording.TB, newModel func() Model) {
	t.Helper()
	m := newModel()
	c := newContractPage()
	st := c.state(t)

	at, err := m.Anchor(st, c.p1.ID)
	if err != nil {
		t.Fatalf("anchor: %v", err)
	}
	if at.Node != c.p1.ID || at.Sel == "" || !strings.Contains(at.Text, "first paragraph") {
		t.Errorf("anchor %+v", at)
	}
	if at.Box == nil || *at.Box != c.boxes[c.p1.ID] {
		t.Errorf("anchor box %v, want %v", at.Box, c.boxes[c.p1.ID])
	}
	if long, _ := m.Anchor(st, c.p2.ID); len([]rune(long.Text)) > 200 || long.Text == "" {
		t.Errorf("anchor text is %d characters; the budget is 200", len([]rune(long.Text)))
	}
	if _, err := m.Anchor(st, c.p1.C[0].ID); err == nil {
		t.Errorf("anchoring a text node must fail: notes point at elements")
	}
	if id, ok := m.Resolve(st, at); !ok || id != c.p1.ID {
		t.Errorf("resolve of a fresh anchor: %d %v", id, ok)
	}

	ctx, err := m.Context(st, at)
	if err != nil {
		t.Fatalf("context: %v", err)
	}
	if ctx.Node != c.p1.ID || !isAncestorOrSelf(st, ctx.Landmark, c.p1.ID) || !strings.Contains(ctx.Text, "first paragraph") {
		t.Errorf("context %+v", ctx)
	}
	if strings.Contains(ctx.Text, "Footer") {
		t.Errorf("context must be focused, not the whole page: %q", ctx.Text)
	}
	if strings.Contains(ctx.Text, "render(") || strings.Contains(ctx.Text, "paragraph.The") {
		t.Errorf("context text must be what a person reads (no script, blocks apart): %q", ctx.Text)
	}

	foot, _ := m.Anchor(st, c.foot.ID)
	for _, set := range [][]recording.NoteAt{{at}, {at, foot}} {
		crops, err := m.Crops(st, set, nil)
		if err != nil || len(crops) == 0 {
			t.Fatalf("crops: %v %v", crops, err)
		}
		for _, a := range set {
			held := false
			for _, cr := range crops {
				held = held || cr.Contains(*a.Box)
			}
			if !held {
				t.Errorf("no crop holds the box %v of node %d (crops %v)", *a.Box, a.Node, crops)
			}
		}
	}
	hint := recording.Box{1, 2, 3, 4}
	if crops, _ := m.Crops(st, []recording.NoteAt{at}, fixedHint(hint)); !containsBox(crops, hint) {
		t.Errorf("a hint's crop must be used: %v", crops)
	}

	// A region note (#470): its crop holds the region and is not widened to
	// its element (or the element's landmark); element notes keep theirs.
	region := recording.Box{c.boxes[c.foot.ID][0] + 4, c.boxes[c.foot.ID][1] + 2, 30, 10}
	rn := &recording.Note{At: recording.NoteAt{Node: c.foot.ID}, Region: &region}
	crops, err := m.NoteCrops(st, []*recording.Note{{At: at}, rn}, nil)
	if err != nil || !holds(crops, region) || !holds(crops, *at.Box) {
		t.Errorf("note crops %v (%v) must hold the region %v and the element note's box %v", crops, err, region, *at.Box)
	}
	if crops, err := m.NoteCrops(st, []*recording.Note{rn}, nil); err != nil || len(crops) != 1 || crops[0][2] >= c.boxes[c.foot.ID][2] {
		t.Errorf("a region note crops its region, not its element: %v (%v)", crops, err)
	}

	// The element goes: its anchor no longer resolves.
	c.b.Add(&recording.Rm{ID: c.p1.ID})
	if _, ok := m.Resolve(c.state(t), at); ok {
		t.Errorf("an anchor whose element was removed must not resolve")
	}

	// Live notes: since the current document began; a resume keeps them,
	// a navigation clears them.
	c = newContractPage()
	c.b.Add(&recording.Note{ID: "a", Author: recording.Author{ID: "u"}, At: recording.NoteAt{Node: c.p1.ID}})
	c.b.Add(&recording.Stop{Why: recording.StopPause})
	c.b.Add(&recording.Seg{Seg: 2, Why: recording.SegResume, Tab: recording.Tab{ID: "tab1"}, VP: recording.Viewport{W: 1280, H: 800}})
	c.b.Snapshot("http://blog/post", c.b.El("html", nil))
	if got := ids(m.Live(c.state(t))); got != "a" {
		t.Errorf("live notes after a resume: %q, want a", got)
	}
	c.b.Add(&recording.Seg{Seg: 3, Why: recording.SegNavigate, Tab: recording.Tab{ID: "tab1"}, VP: recording.Viewport{W: 1280, H: 800}})
	p := c.b.El("p", nil)
	c.b.Snapshot("http://blog/other", c.b.El("html", nil, c.b.El("body", nil, p)))
	c.b.Add(&recording.Note{ID: "b", Author: recording.Author{ID: "u"}, At: recording.NoteAt{Node: p.ID}})
	if got := ids(m.Live(c.state(t))); got != "b" {
		t.Errorf("live notes after a navigation: %q, want b", got)
	}
}

type fixedHint recording.Box

func (h fixedHint) Crop(string, recording.Box) (recording.Box, bool) { return recording.Box(h), true }

func holds(bs []recording.Box, b recording.Box) bool {
	for _, x := range bs {
		if x.Contains(b) {
			return true
		}
	}
	return false
}

func containsBox(bs []recording.Box, b recording.Box) bool {
	for _, x := range bs {
		if x == b {
			return true
		}
	}
	return false
}

func isAncestorOrSelf(st *recording.State, anc, id recording.NodeID) bool {
	for n := st.Node(id); n != nil; n = n.ParentElement() {
		if n.ID == anc {
			return true
		}
	}
	return false
}

func ids(ns []*recording.NoteView) string {
	var out []string
	for _, n := range ns {
		out = append(out, n.ID)
	}
	return strings.Join(out, ",")
}
