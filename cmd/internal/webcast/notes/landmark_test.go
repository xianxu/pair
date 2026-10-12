package notes

import (
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func TestLandmarkOnFixtures(t *testing.T) {
	for _, c := range []struct {
		fixture, note, want string // want: tag[.class]
		meaningful          bool
	}{
		{"rec-blog", "rec-blog-n1", "header", true}, // a heading's landmark is its header
		{"rec-blog", "rec-blog-n2", "li", true},     // an image in a list item
		{"rec-blog", "rec-blog-n3", "ol", true},     // a list is its own landmark
		{"rec-stress", "rec-stress-n1", "section.side", true},
		{"rec-stress", "rec-stress-n2", "section.side", true}, // a closed shadow host
		{"rec-app", "rec-app-n3", "body", false},              // nothing meaningful: the parent, not snapped to
	} {
		n, st := fixtureNote(t, c.fixture, c.note)
		land, ok := landmark(st.Node(n.At.Node))
		got := land.N
		if cls := land.A["class"]; strings.Contains(c.want, ".") && cls != "" {
			got += "." + strings.Fields(cls)[0]
		}
		if got != c.want || ok != c.meaningful {
			t.Errorf("%s: landmark %s (meaningful %v), want %s (%v)", c.note, got, ok, c.want, c.meaningful)
		}
	}
}

func TestLandmarkRules(t *testing.T) {
	b := recording.NewBuilder("land")
	inRole := b.El("span", nil)
	inModal := b.El("span", nil)
	inCard := b.El("span", nil)
	field := b.El("input", []string{"class", "border rounded-lg"})
	inFrameDoc := b.El("span", nil)
	iframe := b.El("iframe", nil)
	b.Page(recording.SegLive, "http://x/",
		b.El("div", []string{"role", "toolbar"}, inRole),
		b.El("div", []string{"aria-modal", "true"}, inModal),
		b.El("div", []string{"class", "p-4 rounded-xl md:shadow-lg"}, inCard),
		b.El("div", []string{"class", "wrap"}, field),
		b.El("aside", nil, iframe))
	fd := recording.Node{ID: b.ID(), Y: recording.YDoc, C: []recording.Node{b.El("html", nil, b.El("body", nil, b.El("div", nil, inFrameDoc)))}}
	b.Add(&recording.Doc{Doc: fd.ID, Host: iframe.ID, URL: "http://y/", Tree: fd})
	st := reduceAll(t, b)
	for _, c := range []struct {
		node       recording.Node
		want       string
		meaningful bool
	}{
		{inRole, "toolbar", true},
		{inModal, "true", true},
		{inCard, "p-4 rounded-xl md:shadow-lg", true},
		{field, "wrap", false}, // a bordered input is not a card; its parent is no container
		{inFrameDoc, "", true}, // through the frame's body to the aside holding the iframe
	} {
		land, ok := landmark(st.Node(c.node.ID))
		got := land.A["role"] + land.A["aria-modal"] + land.A["class"]
		if c.node.ID == inFrameDoc.ID {
			got = strings.TrimPrefix(land.N, "aside")
		}
		if got != c.want || ok != c.meaningful {
			t.Errorf("node %d: landmark <%s> %q (meaningful %v), want %q (%v)", c.node.ID, land.N, got, ok, c.want, c.meaningful)
		}
	}
}

func TestWindowKeepsTheAnchor(t *testing.T) {
	long := strings.Repeat("before ", 200) + "THE ANCHOR TEXT" + strings.Repeat(" after", 200)
	w := window(long, "THE ANCHOR TEXT", contextBudget)
	if len([]rune(w)) != contextBudget || !strings.Contains(w, "THE ANCHOR TEXT") {
		t.Errorf("window %d runes, anchor kept: %v", len([]rune(w)), strings.Contains(w, "THE ANCHOR TEXT"))
	}
	if w := window("  short\n text ", "x", contextBudget); w != "short text" {
		t.Errorf("short text: %q", w)
	}
	if w := window(long, "", 10); w != "before bef" {
		t.Errorf("no focus starts at the beginning: %q", w)
	}
}

func TestAnchorPathNamesTheContainersDownToTheElement(t *testing.T) {
	b := recording.NewBuilder("path")
	p := b.El("p", []string{"id", "first"}, b.Txt("x"))
	span := b.El("span", nil, b.Txt("y"))
	div := b.El("div", []string{"class", "wrapper"}, p, span)
	nav := b.El("ul", []string{"role", "list"}, b.El("li", nil))
	b.Page(recording.SegLive, "http://x/", b.El("main", nil, b.El("article", nil, div)), nav)
	st := reduceAll(t, b)
	if got := pathOf(st.Node(p.ID)); strings.Join(got, " › ") != "main › article › p#first" {
		t.Errorf("path %q", got)
	}
	if got := pathOf(st.Node(nav.C[0].ID)); strings.Join(got, " › ") != "ul[list] › li" {
		t.Errorf("path %q", got)
	}
	deep := b.El("i", nil)
	e := deep
	for range 10 {
		e = b.El("section", nil, e)
	}
	b.Page(recording.SegLive, "http://x/deep", e)
	st = reduceAll(t, b)
	if got := pathOf(st.Node(deep.ID)); len(got) != maxPath || got[len(got)-1] != "i" {
		t.Errorf("a deep path keeps the innermost %d steps: %q", maxPath, got)
	}
}
