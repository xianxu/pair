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
	for _, name := range recording.Fixtures() {
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
				if cs, err := m.NoteCrops(st, []*recording.Note{{At: n.At}}); err != nil || !held(cs, *n.At.Box) {
					t.Errorf("%s: note %s crops %v %v", name, n.ID, cs, err)
				}
			}
		}
	}
}

func TestContextOfABlogHeading(t *testing.T) {
	n, st := fixtureNote(t, "rec-blog", "rec-blog-n1")
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
	// An older note with no box of its own crops the new element's.
	b.Add(&recording.Layout{Boxes: map[recording.NodeID]recording.Box{id: {100, 140, 600, 30}}})
	st = reduceAll(t, b)
	if cs, err := m.NoteCrops(st, []*recording.Note{{At: at}}); err != nil || len(cs) != 1 || !cs[0].Contains(st.Boxes[id]) {
		t.Errorf("an anchor with no box of its own takes the new element's: %v %v", cs, err)
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
	if id, ok := m.Resolve(st, gone); ok {
		t.Errorf("mismatched unique selector must hide: %d %v", id, ok)
	}
}

func TestResolveAmbiguity(t *testing.T) {
	m := Structural{}
	b := recording.NewBuilder("amb")
	a, c := b.El("span", []string{"id", "x"}, b.Txt("left")), b.El("span", []string{"id", "x"}, b.Txt("right"))
	b.Page(recording.SegLive, "http://x/", a, c)
	st := reduceAll(t, b)
	if id, ok := m.Resolve(st, recording.NoteAt{Node: 9999, Sel: "#x", Tag: "span", Text: "right"}); !ok || id != c.ID {
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

// pair#469: the app's Reports heading sits beside a table its script fills.
// The context is what a person reads: the cells apart, the script absent.
func TestContextOfAppReportsIsRenderedText(t *testing.T) {
	n, st := fixtureNote(t, "rec-app", "rec-app-n3")
	ctx, err := Structural{}.Context(st, n.At)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(ctx.Text, "Reports Week 1 0 tasks Week 2 137 tasks") {
		t.Errorf("blocks run together: %q", ctx.Text)
	}
	if strings.Contains(ctx.Text, "getElementById") || strings.Contains(ctx.Text, "insertAdjacentHTML") {
		t.Errorf("context carries the script body: %q", ctx.Text)
	}
}

func TestResolveRejectsConnectedConflictAndRouteChange(t *testing.T) {
	b := recording.NewBuilder("identity")
	n := b.El("button", []string{"id", "publish", "class", "primary"}, b.Txt("Publish report"))
	b.Page(recording.SegLive, "http://x/one", n)
	st := reduceAll(t, b)
	at, _ := Structural{}.Anchor(st, n.ID)
	if at.Tag != "button" || at.Stable["id"] != "publish" || len(at.Classes) != 1 || at.Route != "http://x/one" {
		t.Fatalf("missing frozen evidence: %+v", at)
	}
	st.Node(n.ID).A["id"] = "delete"
	if _, ok := (Structural{}).Resolve(st, at); ok {
		t.Fatal("reused connected identity resolved")
	}
	st.Node(n.ID).A["id"] = "publish"
	st.URL = "http://x/two"
	if _, ok := (Structural{}).Resolve(st, at); ok {
		t.Fatal("changed route resolved")
	}
}

func TestResolveReorderedSiblingAndAmbiguousCopy(t *testing.T) {
	b := recording.NewBuilder("reorder")
	old := b.El("li", nil, b.Txt("Quarterly report"))
	list := b.El("ul", []string{"id", "reports"}, old)
	b.Page(recording.SegLive, "http://x/", list)
	st := reduceAll(t, b)
	at, _ := Structural{}.Anchor(st, old.ID)
	replacement := b.El("li", nil, b.Txt("Quarterly report"))
	b.Add(&recording.Kids{P: list.ID, C: []recording.Node{b.El("li", nil, b.Txt("Other report")), replacement}})
	st = reduceAll(t, b)
	if id, ok := (Structural{}).Resolve(st, at); !ok || id != replacement.ID {
		t.Fatalf("reorder: %d %v", id, ok)
	}
	b.Add(&recording.Kids{P: list.ID, C: []recording.Node{replacement, b.El("li", nil, b.Txt("Quarterly report"))}})
	st = reduceAll(t, b)
	if _, ok := (Structural{}).Resolve(st, at); ok {
		t.Fatal("duplicate replacement resolved")
	}
}

func TestResolveEnumerationExhaustion(t *testing.T) {
	b := recording.NewBuilder("bounded")
	old := b.El("li", nil, b.Txt("Quarterly report"))
	list := b.El("ul", []string{"id", "reports"}, old)
	b.Page(recording.SegLive, "http://x/", list)
	st := reduceAll(t, b)
	at, _ := Structural{}.Anchor(st, old.ID)
	kids := []recording.Node{b.El("li", nil, b.Txt("Quarterly report"))}
	for i := 0; i < MaxAnchorCandidates; i++ {
		kids = append(kids, b.El("li", nil))
	}
	b.Add(&recording.Kids{P: list.ID, C: kids})
	st = reduceAll(t, b)
	if _, ok := (Structural{}).Resolve(st, at); ok {
		t.Fatal("truncated search resolved")
	}
}

func TestResolveStaysInShadowScope(t *testing.T) {
	b := recording.NewBuilder("scope")
	old := b.El("p", []string{"id", "report"}, b.Txt("Quarterly report"))
	shadow := recording.Node{ID: b.ID(), Y: recording.YShadow, Mode: "open", C: []recording.Node{old}}
	host := b.El("section", []string{"id", "host"})
	host.SR = &shadow
	outside := b.El("p", []string{"id", "report"}, b.Txt("Quarterly report"))
	b.Page(recording.SegLive, "http://x/", host, outside)
	st := reduceAll(t, b)
	at, _ := Structural{}.Anchor(st, old.ID)
	b.Add(&recording.Kids{P: shadow.ID, C: nil})
	st = reduceAll(t, b)
	if _, ok := (Structural{}).Resolve(st, at); ok {
		t.Fatal("escaped shadow tree")
	}
	replacement := b.El("p", []string{"id", "report"}, b.Txt("Quarterly report"))
	b.Add(&recording.Kids{P: shadow.ID, C: []recording.Node{replacement}})
	st = reduceAll(t, b)
	if id, ok := (Structural{}).Resolve(st, at); !ok || id != replacement.ID {
		t.Fatalf("same scope replacement: %d %v", id, ok)
	}
}

func TestLegacyEmptyPathIsSafe(t *testing.T) {
	b := recording.NewBuilder("empty-path")
	n := b.El("p", nil)
	b.Page(recording.SegLive, "http://x/", n)
	if id, ok := (Structural{}).Resolve(reduceAll(t, b), recording.NoteAt{Node: n.ID, Path: []string{""}}); !ok || id != n.ID {
		t.Fatal("legacy direct identity should work")
	}
}

// A matching selector/text in a navigated frame is not the original document.
// Until frame-route evidence is persisted, only connected node identity is safe.
func TestResolveHidesReplacementAfterFrameNavigation(t *testing.T) {
	for _, nestedShadow := range []bool{false, true} {
		t.Run(map[bool]string{false: "frame", true: "shadow-in-frame"}[nestedShadow], func(t *testing.T) {
			b := recording.NewBuilder("frame-route")
			frame := b.El("iframe", []string{"id", "preview"})
			b.Page(recording.SegLive, "http://x/", frame)
			content := func() (recording.Node, recording.Node) {
				item := b.El("h2", []string{"id", "report"}, b.Txt("Quarterly earnings"))
				container := b.El("section", []string{"id", "reports"}, item)
				if nestedShadow {
					container.C = nil
					container.SR = &recording.Node{ID: b.ID(), Y: recording.YShadow, Mode: "open", C: []recording.Node{item}}
				}
				return recording.Node{ID: b.ID(), Y: recording.YDoc, C: []recording.Node{b.El("html", nil, b.El("body", nil, container))}}, item
			}
			first, old := content()
			b.Add(&recording.Doc{Doc: first.ID, Host: frame.ID, URL: "http://y/original", Tree: first})
			st := reduceAll(t, b)
			at, _ := Structural{}.Anchor(st, old.ID)
			if id, ok := (Structural{}).Resolve(st, at); !ok || id != old.ID {
				t.Fatal("connected frame anchor must resolve")
			}
			next, _ := content()
			b.Add(&recording.Doc{Doc: next.ID, Host: frame.ID, URL: "http://y/replacement", Tree: next})
			st = reduceAll(t, b)
			if id, ok := (Structural{}).Resolve(st, at); ok {
				t.Fatalf("navigated frame reused anchor on %d", id)
			}
		})
	}
}
