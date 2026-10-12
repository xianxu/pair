package query

import (
	"slices"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast/notes"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// selectPage: a nav with two links, a form, a shadow host and a template.
func selectPage(t *testing.T) *recording.State {
	t.Helper()
	b := recording.NewBuilder("sel")
	inner := b.El("span", []string{"class", "badge"}, b.Txt("in shadow"))
	host := b.El("x-card", []string{"id", "card"})
	host.SR = &recording.Node{ID: b.ID(), Y: recording.YShadow, Mode: "closed", C: []recording.Node{inner}}
	tpl := b.El("template", []string{"id", "tpl"})
	tpl.TC = &recording.Node{ID: b.ID(), Y: recording.YFrag, C: []recording.Node{b.El("i", []string{"class", "badge"})}}
	b.Page(recording.SegLive, "http://app/",
		b.El("nav", []string{"class", "top main"},
			b.El("a", []string{"href", "/a", "class", "link"}, b.Txt("A")),
			b.El("a", []string{"href", "/b", "class", "link active", "aria-label", "go to B"}, b.Txt("B"))),
		b.El("form", nil, b.El("input", []string{"name", "q", "type", "text"})),
		host, tpl)
	st, err := recording.Reduce(b.Recs, 0)
	if err != nil {
		t.Fatal(err)
	}
	return st
}

func TestSelect(t *testing.T) {
	st := selectPage(t)
	for _, c := range []struct {
		sel  string
		want string // the matches' Describe, joined by |
	}{
		{"a", `<a.link> "A"|<a.link.active> "B"`},
		{"nav a.active", `<a.link.active> "B"`},
		{"nav > a:nth-child(2)", `<a.link.active> "B"`},
		{"body > a", ``},
		{"nav  >  a.link[href=/a]", `<a.link> "A"`},
		{`a[aria-label*="to B"]`, `<a.link.active> "B"`},
		{"[name]", `<input>`},
		{"input[type='text']", `<input>`},
		{".top.main", `<nav.top.main> "A B"`},
		{"#card span", ``},                             // a space doesn't cross into a shadow root
		{"#card > span", ``},                           // nor does >
		{"#card >>> span", `<span.badge> "in shadow"`}, // >>> does, closed or open
		{"x-card>>>.badge", `<span.badge> "in shadow"`},
		{"#tpl >>> i", `<i.badge>`}, // into a template's content
		{".badge", ``},              // a selector starts in the top document
		{"body >>> span", ``},       // >>> enters only the host's own tree
		{"*:nth-child(3)", `<x-card#card>`},
		{"a:nth-of-type(2)", `<a.link.active> "B"`},
		{"form > input:nth-of-type(1)", `<input>`},
		{"#card >>> span:nth-of-type(1)", `<span.badge> "in shadow"`}, // a note's selector, matched by notes.Match
	} {
		got, err := Select(st, c.sel)
		if err != nil {
			t.Errorf("%q: %v", c.sel, err)
			continue
		}
		var ds []string
		for _, n := range got {
			d := Describe(n)
			ds = append(ds, d[strings.IndexByte(d, ' ')+1:])
		}
		if s := strings.Join(ds, "|"); s != c.want {
			t.Errorf("%q matched %s, want %s", c.sel, s, c.want)
		}
	}
}

func TestSelectByNodeID(t *testing.T) {
	st := selectPage(t)
	got, err := Select(st, "@1")
	if err != nil || len(got) != 1 || got[0].ID != 1 {
		t.Errorf("@1: %v %v", got, err)
	}
	if got, err := Select(st, "@9999"); err != nil || len(got) != 0 {
		t.Errorf("an id that isn't live matches nothing: %v %v", got, err)
	}
}

// Anything outside the grammar is refused with the grammar, never matched
// loosely: an agent's :hover or ~ must not silently mean something else.
func TestSelectRefusesWhatItCantRead(t *testing.T) {
	st := selectPage(t)
	for _, sel := range []string{"", "a:hover", "h1~Reports", "a + b", "a >", "> a", "a >> b", ">>> a", "a >>>", "a > >>> b", "a >>> > b", "a:nth-of-type(x)", "[x", "[a~=b]", ":nth-child(0)", "#", "@x", "a[=b]"} {
		if _, err := Select(st, sel); err == nil || !strings.Contains(err.Error(), "selectors are") {
			t.Errorf("%q: %v, want a refusal naming the grammar", sel, err)
		}
	}
}

// A note's selector pastes into --sel and names the note's element. For
// the real model the oracle is its own Resolve: an anchor it resolves to
// exactly one element (no node id to fall back on) must select exactly that
// element, so an ambiguous match can't pass. The fake's child-index paths
// are unique by construction within the top document.
func TestModelSelectorsSelectTheirElement(t *testing.T) {
	for _, name := range []string{"app", "stress", "blog"} {
		r, err := recording.ReadDir(recording.Fixture(name))
		if err != nil {
			t.Fatal(err)
		}
		st, err := recording.Reduce(r.Records, 3000)
		if err != nil {
			t.Fatal(err)
		}
		unique, checked := 0, 0
		walkAll(st.Top, func(n *recording.DOMNode) {
			if n.Y != recording.YEl {
				return
			}
			at, err := notes.Structural{}.Anchor(st, n.ID)
			if err != nil {
				t.Fatal(err)
			}
			if at.Sel != "" {
				got, err := Select(st, at.Sel)
				if err != nil {
					t.Fatalf("%s %s: %v", name, at.Sel, err)
				}
				resolved, ok := notes.Structural{}.Resolve(st, recording.NoteAt{Node: -1, Sel: at.Sel})
				switch {
				case ok && resolved == n.ID:
					if len(got) != 1 || got[0] != n {
						t.Errorf("%s %s: %d matches, want exactly @%d", name, at.Sel, len(got), n.ID)
					}
					unique++
				case !slices.Contains(got, n):
					t.Errorf("%s %s: %d matches, none @%d", name, at.Sel, len(got), n.ID)
				}
			}
			if fake, _ := (notes.FakeModel{}).Anchor(st, n.ID); treeOf(n) == st.Top && fake.Sel[0] != '#' {
				if got, err := Select(st, fake.Sel); err != nil || len(got) != 1 || got[0] != n {
					t.Errorf("%s fake %s: %v %d matches, want exactly @%d", name, fake.Sel, err, len(got), n.ID)
				}
			}
			checked++
		})
		if checked < 30 || unique < checked/2 {
			t.Errorf("%s: %d elements, %d with a unique anchor", name, checked, unique)
		}
	}
}

// The review's probes: anchors that a boundary-crossing reading made
// ambiguous name one element.
func TestAnchorsInsideOtherTreesAreExact(t *testing.T) {
	for _, c := range []struct {
		fixture, sel string
		at           int64
	}{
		{"stress", "html:nth-of-type(1)", 3000},
		{"blog", "astro-dev-toolbar >>> style:nth-of-type(1)", 2500},
	} {
		r, err := recording.ReadDir(recording.Fixture(c.fixture))
		if err != nil {
			t.Fatal(err)
		}
		st, err := recording.Reduce(r.Records, c.at)
		if err != nil {
			t.Fatal(err)
		}
		if got, err := Select(st, c.sel); err != nil || len(got) != 1 {
			t.Errorf("%s %q: %v, %d matches, want 1", c.fixture, c.sel, err, len(got))
		}
	}
}
