package notes

import (
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func TestStructuralMeetsTheModelContract(t *testing.T) {
	RunModelContract(t, func() Model { return Structural{} })
}

// Every fixture note resolves at its moment, and gets a focused context
// and crops that hold its box.
func TestStructuralOnFixtures(t *testing.T) {
	m := Structural{}
	for _, name := range recording.Fixtures {
		r, err := recording.ReadDir(recording.Fixture(name))
		if err != nil {
			t.Fatal(err)
		}
		for _, x := range r.Records {
			n, ok := x.(*recording.Note)
			if !ok {
				continue
			}
			st, err := recording.Reduce(r.Records, n.T)
			if err != nil {
				t.Fatal(err)
			}
			if id, ok := m.Resolve(st, n.At); !ok || id != n.At.Node {
				t.Errorf("%s: note %s doesn't resolve", name, n.ID)
			}
			ctx, err := m.Context(st, n.At)
			if err != nil || !isAncestorOrSelf(st, ctx.Landmark, n.At.Node) || len([]rune(ctx.Text)) > contextBudget {
				t.Errorf("%s: note %s context %+v %v", name, n.ID, ctx, err)
			}
			if n.At.Text != "" && !strings.Contains(ctx.Text, n.At.Text[:min(len(n.At.Text), 40)]) {
				t.Errorf("%s: note %s context %q lacks the anchor's text", name, n.ID, ctx.Text)
			}
			if n.At.Box != nil {
				if cs, err := m.Crops(st, []recording.NoteAt{n.At}, nil); err != nil || !held(cs, *n.At.Box) {
					t.Errorf("%s: note %s crops %v %v", name, n.ID, cs, err)
				}
			}
		}
	}
}

func TestContextOfABlogHeading(t *testing.T) {
	n, st := fixtureNote(t, "blog", "blog-n1")
	ctx, err := Structural{}.Context(st, n.At)
	if err != nil {
		t.Fatal(err)
	}
	header := st.Node(ctx.Landmark)
	if header.N != "header" || ctx.Box == nil || *ctx.Box != st.Boxes[header.ID] {
		t.Errorf("landmark <%s> box %v", header.N, ctx.Box)
	}
	// The header holds the date and reading time as well as the title.
	if !strings.Contains(ctx.Text, "Building Blocks of AI Coding") || !strings.Contains(ctx.Text, "min read") {
		t.Errorf("context text %q", ctx.Text)
	}
}

// A re-render replaces an element with an equal one under a new id; the
// anchor follows by its selector. Two equal candidates resolve by text, or
// not at all.
func TestResolveAfterARerender(t *testing.T) {
	m := Structural{}
	b := recording.NewBuilder("rerender")
	list := b.El("ul", []string{"id", "todo"}, b.El("li", nil, b.Txt("one")), b.El("li", nil, b.Txt("two")))
	b.Page(recording.SegLive, "http://x/", list)
	st := reduceAll(t, b)
	two := st.Node(list.ID).Kids[1]
	at, _ := m.Anchor(st, two.ID)
	if at.Sel != "#todo > li:nth-of-type(2)" {
		t.Fatalf("selector %q", at.Sel)
	}

	b.Add(&recording.Kids{P: list.ID, C: []recording.Node{b.El("li", nil, b.Txt("one")), b.El("li", nil, b.Txt("two"))}})
	st = reduceAll(t, b)
	id, ok := m.Resolve(st, at)
	if !ok || id == two.ID || st.Node(id).TextContent() != "two" {
		t.Errorf("after a re-render: %d %v", id, ok)
	}
	if ctx, err := m.Context(st, at); err != nil || ctx.Node != id {
		t.Errorf("context after a re-render: %+v %v", ctx, err)
	}
	// Its crop still snaps to the new element's landmark (the li itself).
	b.Add(&recording.Layout{Boxes: map[recording.NodeID]recording.Box{id: {100, 140, 600, 30}}})
	st = reduceAll(t, b)
	if cs, err := m.Crops(st, []recording.NoteAt{at}, nil); err != nil || len(cs) != 1 || !cs[0].Contains(st.Boxes[id]) {
		t.Errorf("an anchor with no box of its own takes the new element's: %v %v", cs, err)
	}
	at.Box = &recording.Box{110, 140, 300, 30}
	if cs, _ := m.Crops(st, []recording.NoteAt{at}, nil); len(cs) != 1 || cs[0][0]+cs[0][2] != 700 {
		t.Errorf("crop after a re-render %v must reach the li's right edge, 700", cs)
	}

	// A selector that names nothing.
	amb := recording.NoteAt{Node: two.ID, Sel: "#todo > li:nth-of-type(2) >>> x:nth-of-type(1)"}
	if _, ok := m.Resolve(st, amb); ok {
		t.Errorf("a selector naming nothing must not resolve")
	}
	twin := b.El("p", []string{"class", "a"}, b.Txt("same"))
	twin2 := b.El("p", []string{"class", "a"}, b.Txt("other"))
	b.Add(&recording.Ins{P: st.Top.Kids[1].Kids[1].ID, After: list.ID, Node: b.El("div", []string{"id", "pair"}, twin, twin2)})
	st = reduceAll(t, b)
	gone := recording.NoteAt{Node: 9999, Sel: "#pair > p:nth-of-type(1)", Text: "other"}
	if id, ok := m.Resolve(st, gone); !ok || id != twin.ID {
		t.Errorf("one match resolves even when its text changed: %d %v", id, ok)
	}
}

func TestResolveAmbiguity(t *testing.T) {
	m := Structural{}
	b := recording.NewBuilder("amb")
	a, c := b.El("span", []string{"id", "x"}, b.Txt("left")), b.El("span", []string{"id", "x"}, b.Txt("right"))
	b.Page(recording.SegLive, "http://x/", a, c)
	st := reduceAll(t, b)
	if id, ok := m.Resolve(st, recording.NoteAt{Node: 9999, Sel: "#x", Text: "right"}); !ok || id != c.ID {
		t.Errorf("two matches, one with the anchor's text: %d %v", id, ok)
	}
	if _, ok := m.Resolve(st, recording.NoteAt{Node: 9999, Sel: "#x", Text: "neither"}); ok {
		t.Errorf("two matches and no text to tell them apart must not resolve")
	}
}

// Anchor never stores a selector that can't find its element: a tag outside
// the subset's grammar leaves the selector empty, and the anchor resolves by
// its node.
func TestAnchorDropsASelectorThatCantRoundTrip(t *testing.T) {
	b := recording.NewBuilder("odd")
	odd := b.El("x.y", nil)
	b.Page(recording.SegLive, "http://x/", odd)
	st := reduceAll(t, b)
	at, err := Structural{}.Anchor(st, odd.ID)
	if err != nil || at.Sel != "" {
		t.Errorf("anchor %+v %v: want no selector", at, err)
	}
}
