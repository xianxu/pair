package notes

import (
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// boxNote is a note on box b, anchored on the page's first element.
func boxNote(b recording.Box) *recording.Note {
	return &recording.Note{At: recording.NoteAt{Node: 1}, Region: &b}
}

func padded(b recording.Box) recording.Box {
	return recording.Box{b[0] - defaultMargin, b[1] - defaultMargin, b[2] + 2*defaultMargin, b[3] + 2*defaultMargin}
}

// A note's crop is its box plus the margin (#488), never snapped: on the
// blog's header, where #461 widened an element's crop to full width.
func TestNoteCropIsTheBoxPlusAMargin(t *testing.T) {
	n, st := fixtureNote(t, "rec-blog", "rec-blog-n1")
	region := recording.Box{400, 200, 120, 40}
	cs, err := Structural{}.NoteCrops(st, []*recording.Note{{At: n.At, Region: &region}})
	if err != nil || len(cs) != 1 || cs[0] != padded(region) {
		t.Errorf("crop %v (%v), want %v", cs, err, padded(region))
	}
	if defaultMargin != 16 {
		t.Errorf("the margin is the operator's 16 px, got %v", defaultMargin)
	}
}

func TestNoteCropsMergeOnlyWhenTheyOverlap(t *testing.T) {
	b := recording.NewBuilder("merge")
	b.Page(recording.SegLive, "http://x/", b.El("p", nil))
	st := reduceAll(t, b)
	a, near, touching, far := recording.Box{100, 100, 100, 40}, recording.Box{180, 130, 100, 40}, recording.Box{100, 170, 100, 40}, recording.Box{100, 400, 100, 40}
	for _, c := range []struct {
		name string
		in   []recording.Box
		want int
	}{
		{"overlapping boxes share a crop", []recording.Box{a, near}, 1},
		{"crops whose margins overlap share one", []recording.Box{a, touching}, 1}, // 30 px apart: the margins of 16 overlap
		{"boxes far apart get a crop each", []recording.Box{a, far}, 2},
		{"merging chains", []recording.Box{far, a, touching, near}, 2},
	} {
		var ns []*recording.Note
		for _, x := range c.in {
			ns = append(ns, boxNote(x))
		}
		cs, err := Structural{}.NoteCrops(st, ns)
		if err != nil || len(cs) != c.want {
			t.Errorf("%s: %d crops %v (%v), want %d", c.name, len(cs), cs, err, c.want)
		}
		for _, x := range c.in {
			if !held(cs, x) {
				t.Errorf("%s: no crop holds %v", c.name, x)
			}
		}
	}
	// Margins that only meet (twice the margin apart) don't overlap.
	if cs, _ := (Structural{}).NoteCrops(st, []*recording.Note{boxNote(a), boxNote(recording.Box{100, 172, 100, 40})}); len(cs) != 2 {
		t.Errorf("32 px apart: %v, want two crops", cs)
	}
}

func TestNoteCropClipsToThePage(t *testing.T) {
	b := recording.NewBuilder("edge")
	doc := b.Page(recording.SegLive, "http://x/", b.El("p", nil))
	b.Add(&recording.Layout{Boxes: map[recording.NodeID]recording.Box{doc.C[1].ID: {0, 0, 1280, 2000}}})
	st := reduceAll(t, b)
	cs, _ := Structural{}.NoteCrops(st, []*recording.Note{boxNote(recording.Box{4, 4, 100, 20}), boxNote(recording.Box{1200, 1900, 100, 50})})
	if len(cs) != 2 || cs[0] != (recording.Box{0, 0, 120, 40}) {
		t.Fatalf("corner crop %v, want [0 0 120 40]", cs)
	}
	// The second sticks out of the page on the right: its own box is never
	// cut, its margin is.
	if want := (recording.Box{1184, 1884, 116, 82}); cs[1] != want {
		t.Errorf("edge crop %v, want %v", cs[1], want)
	}
}

// An older note, made before #488, has no region: it crops the box it was
// made on, else its element's layout box now.
func TestOlderNoteCropsItsAnchorBox(t *testing.T) {
	_, st := fixtureNote(t, "segments", "seg-n1")
	given := recording.NoteAt{Node: 1, Box: &recording.Box{500, 500, 10, 10}} // the viewer's, not the layout's [16 60 200 24]
	cs, err := Structural{}.NoteCrops(st, []*recording.Note{{At: given}})
	if err != nil || len(cs) != 1 || cs[0] != padded(*given.Box) {
		t.Errorf("crop %v %v must be around the given box", cs, err)
	}
	cs, err = Structural{}.NoteCrops(st, []*recording.Note{{At: recording.NoteAt{Node: 1}}})
	if err != nil || len(cs) != 1 || !cs[0].Contains(st.Boxes[1]) {
		t.Errorf("crop %v %v must be around the element's layout box", cs, err)
	}
	if _, err := (Structural{}).NoteCrops(st, []*recording.Note{{At: recording.NoteAt{Node: 8}}}); err == nil {
		t.Errorf("a note with no box anywhere must fail")
	}
}

func TestClampToPage(t *testing.T) {
	b := recording.NewBuilder("clamp")
	doc := b.Page(recording.SegLive, "http://x/", b.El("p", nil))
	b.Add(&recording.Layout{Boxes: map[recording.NodeID]recording.Box{doc.C[1].ID: {0, 0, 1280, 300}}})
	st := reduceAll(t, b)
	st.VP = recording.Viewport{W: 1280, H: 800}
	for _, c := range []struct {
		in, want recording.Box
		ok       bool
	}{
		{recording.Box{10, 10, 50, 50}, recording.Box{10, 10, 50, 50}, true},
		{recording.Box{-40, -40, 100, 100}, recording.Box{0, 0, 60, 60}, true},
		{recording.Box{1250, 500, 100, 100}, recording.Box{1250, 500, 30, 100}, true}, // below a short page, in the viewport
		{recording.Box{1250, 1e5, 100, 100}, recording.Box{}, false},
		{recording.Box{-500, 10, 100, 100}, recording.Box{}, false},
	} {
		got, ok := ClampToPage(st, c.in)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("ClampToPage(%v) = %v %v, want %v %v", c.in, got, ok, c.want, c.ok)
		}
	}
}

func held(cs []recording.Box, b recording.Box) bool {
	for _, c := range cs {
		if c.Contains(b) {
			return true
		}
	}
	return false
}
