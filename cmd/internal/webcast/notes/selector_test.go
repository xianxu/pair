package notes

import (
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func TestSelectorSteps(t *testing.T) {
	b := recording.NewBuilder("sel")
	stable := b.El("p", []string{"id", "intro"})
	generated := b.El("p", []string{"id", ":r1:"})
	numbered := b.El("p", []string{"id", "row-12345"})
	dup1 := b.El("span", []string{"id", "twin"})
	dup2 := b.El("span", []string{"id", "twin"})
	tested := b.El("button", []string{"data-testid", "save"})
	arrow := b.El("i", []string{"data-testid", "a > b"}) // a combinator in a value can't round-trip
	inTemplate := b.El("b", nil)
	tmpl := recording.Node{ID: b.ID(), Y: recording.YEl, N: "template", A: map[string]string{"id": "row"},
		TC: &recording.Node{ID: b.ID(), Y: recording.YFrag, C: []recording.Node{inTemplate}}}
	inner := b.El("em", nil)
	under := b.El("div", []string{"id", "box"}, b.El("p", nil), b.El("p", nil, inner))
	b.Page(recording.SegLive, "http://x/", stable, generated, numbered, dup1, dup2, tested, under, arrow, tmpl)
	st := reduceAll(t, b)

	for _, c := range []struct {
		node recording.Node
		want string
	}{
		{stable, "#intro"},
		{generated, "html:nth-of-type(1) > body:nth-of-type(1) > p:nth-of-type(2)"},
		{numbered, "html:nth-of-type(1) > body:nth-of-type(1) > p:nth-of-type(3)"},
		{dup2, "html:nth-of-type(1) > body:nth-of-type(1) > span:nth-of-type(2)"},
		{tested, `[data-testid="save"]`},
		{inner, "#box > p:nth-of-type(2) > em:nth-of-type(1)"},
		{arrow, "html:nth-of-type(1) > body:nth-of-type(1) > i:nth-of-type(1)"},
		{inTemplate, "#row >>> b:nth-of-type(1)"},
	} {
		got := selectorFor(st.Node(c.node.ID))
		if got != c.want {
			t.Errorf("selector of %d: %q, want %q", c.node.ID, got, c.want)
		}
		if m := matchSelector(st, got); len(m) != 1 || m[0].ID != c.node.ID {
			t.Errorf("%q matches %v, want only %d", got, m, c.node.ID)
		}
	}
	for _, foreign := range []string{"h1~Reports", "p.intro", "img[alt*=x]", "#twin"} {
		if m := matchSelector(st, foreign); foreign != "#twin" && len(m) != 0 {
			t.Errorf("%q is outside the subset and must match nothing: %v", foreign, m)
		}
	}
	if m := matchSelector(st, "#twin"); len(m) != 2 {
		t.Errorf("a duplicated id matches both: %v", m)
	}
}

// Every element of the real fixtures, through shadow roots and frames,
// matches its own selector and nothing else.
func TestSelectorRoundTripOnFixtures(t *testing.T) {
	for _, name := range []string{"rec-stress", "rec-app", "rec-blog"} {
		st := fixtureEnd(t, name)
		deep := 0
		walkElements(st.Top, func(n *recording.DOMNode) {
			sel := selectorFor(n)
			if treeRoot(n).Host != nil {
				deep++
			}
			if m := matchSelector(st, sel); len(m) != 1 || m[0] != n {
				t.Errorf("%s: %q of node %d matches %d elements", name, sel, n.ID, len(m))
			}
		})
		if name == "rec-stress" && deep == 0 {
			t.Errorf("rec-stress has shadow roots and a frame; no element inside them was checked")
		}
	}
}

// Match answers for the anchor grammar only: anything else is "not mine",
// so a caller with a wider grammar can fall back to it (#462).
func TestMatchKnowsItsGrammar(t *testing.T) {
	r, err := recording.ReadDir(recording.Fixture("rec-stress"))
	if err != nil {
		t.Fatal(err)
	}
	st, err := recording.Reduce(r.Records, 3000)
	if err != nil {
		t.Fatal(err)
	}
	for _, sel := range []string{"#closed", `[data-testid="x"]`, "html:nth-of-type(1) > body:nth-of-type(1)", "#closed >>> span:nth-of-type(1)"} {
		if _, ok := Match(st, sel); !ok {
			t.Errorf("%q is an anchor", sel)
		}
	}
	for _, sel := range []string{"", "span", ".pill", "a b", "html > body", "#closed span"} {
		if _, ok := Match(st, sel); ok {
			t.Errorf("%q is not an anchor", sel)
		}
	}
	if got, _ := Match(st, "html:nth-of-type(1)"); len(got) != 1 || got[0].Parent != st.Top {
		t.Errorf("a path starts at the top document's root element: %d matches", len(got))
	}
}
