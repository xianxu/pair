package notes

import (
	"math"
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// cropsOf is Structural's crops for fixture notes made at about the same
// moment, at the first note's state.
func cropsOf(t *testing.T, fixture string, ids ...string) ([]recording.Box, []recording.NoteAt, *recording.State) {
	t.Helper()
	var ats []recording.NoteAt
	var st *recording.State
	for _, id := range ids {
		n, s := fixtureNote(t, fixture, id)
		if st == nil {
			st = s
		}
		ats = append(ats, n.At)
	}
	cs, err := Structural{}.Crops(st, ats, nil)
	if err != nil {
		t.Fatal(err)
	}
	return cs, ats, st
}

func TestCropSnapsAHeaderToFullWidth(t *testing.T) {
	cs, ats, st := cropsOf(t, "blog", "blog-n1")
	header := st.Boxes[st.Node(ats[0].Node).Parent.ID] // the h1's header, [0 157 1280 177]
	if len(cs) != 1 || cs[0][0] != 0 || cs[0][2] != 1280 || !cs[0].Contains(header) {
		t.Errorf("crop %v must be full width and hold the header %v", cs, header)
	}
}

func TestCropKeepsATallLandmarksWidthNotItsHeight(t *testing.T) {
	cs, ats, st := cropsOf(t, "blog", "blog-n2")
	img := *ats[0].Box
	land, _ := landmark(st.Node(ats[0].Node))
	li := st.Boxes[land.ID] // 1218 px tall: more than half the viewport
	c := cs[0]
	if c[0] > li[0] || c[0]+c[2] < li[0]+li[2] {
		t.Errorf("crop %v must span the li's width %v", c, li)
	}
	if want := img[3] + 2*defaultMargin; math.Abs(c[3]-want) > 0.01 || !c.Contains(img) {
		t.Errorf("crop %v: height %v, want the image's plus margins (%v)", c, c[3], want)
	}
}

func TestCropClusters(t *testing.T) {
	for _, c := range []struct {
		ids  []string
		want int
	}{
		{[]string{"blog-n1", "blog-n3"}, 2}, // 332 px apart once snapped
		{[]string{"blog-n1", "blog-n2"}, 2},
		{[]string{"blog-n1", "blog-n2", "blog-n3"}, 3},
	} {
		cs, ats, _ := cropsOf(t, "blog", c.ids...)
		if len(cs) != c.want {
			t.Errorf("%v: %d crops %v, want %d", c.ids, len(cs), cs, c.want)
		}
		for _, at := range ats {
			if !held(cs, *at.Box) {
				t.Errorf("%v: no crop holds %v", c.ids, *at.Box)
			}
		}
	}

	// Two neighbouring paragraphs share a crop.
	b := recording.NewBuilder("near")
	p1, p2 := b.El("p", nil), b.El("p", nil)
	b.Page(recording.SegLive, "http://x/", p1, p2)
	b.Add(&recording.Layout{Boxes: map[recording.NodeID]recording.Box{p1.ID: {100, 100, 400, 40}, p2.ID: {100, 180, 400, 40}}})
	st := reduceAll(t, b)
	a1, _ := Structural{}.Anchor(st, p1.ID)
	a2, _ := Structural{}.Anchor(st, p2.ID)
	if cs, _ := (Structural{}).Crops(st, []recording.NoteAt{a1, a2}, nil); len(cs) != 1 || cs[0] != (recording.Box{52, 52, 496, 216}) {
		t.Errorf("neighbours: %v, want one crop [52 52 496 216]", cs)
	}
}

func TestCropClipsToThePage(t *testing.T) {
	b := recording.NewBuilder("edge")
	corner, wide := b.El("p", nil), b.El("p", nil)
	doc := b.Page(recording.SegLive, "http://x/", corner, wide)
	b.Add(&recording.Layout{Boxes: map[recording.NodeID]recording.Box{
		doc.C[1].ID: {0, 0, 1280, 2000}, corner.ID: {10, 10, 100, 20}, wide.ID: {1200, 1900, 100, 50}}})
	st := reduceAll(t, b)
	ats := []recording.NoteAt{}
	for _, id := range []recording.NodeID{corner.ID, wide.ID} {
		at, _ := Structural{}.Anchor(st, id)
		ats = append(ats, at)
	}
	cs, _ := Structural{}.Crops(st, ats, nil)
	if len(cs) != 2 || cs[0] != (recording.Box{0, 0, 158, 78}) {
		t.Fatalf("corner crop %v, want [0 0 158 78]", cs)
	}
	// wide sticks out of the page on the right; its own box is never cut.
	if want := (recording.Box{1152, 1852, 148, 146}); cs[1] != want {
		t.Errorf("edge crop %v, want %v", cs[1], want)
	}
}

func TestCropUsesTheGivenBox(t *testing.T) {
	_, st := fixtureNote(t, "segments", "seg-n1")
	at := recording.NoteAt{Node: 1, Box: &recording.Box{500, 500, 10, 10}} // the viewer's, not the layout's [16 60 200 24]
	cs, err := Structural{}.Crops(st, []recording.NoteAt{at}, nil)
	if err != nil || len(cs) != 1 || !cs[0].Contains(*at.Box) || cs[0].Contains(st.Boxes[1]) {
		t.Errorf("crop %v %v must be around the given box", cs, err)
	}
	if _, err := (Structural{}).Crops(st, []recording.NoteAt{{Node: 8}}, nil); err == nil {
		t.Errorf("a note with no box anywhere must fail")
	}
}

func TestCropHintsReplaceEachCrop(t *testing.T) {
	n1, st := fixtureNote(t, "blog", "blog-n1")
	n2, _ := fixtureNote(t, "blog", "blog-n2")
	var seen []recording.Box
	h := hintFunc(func(url string, b recording.Box) (recording.Box, bool) {
		seen = append(seen, b)
		return recording.Box{1, 1, 1, 1}, url == st.URL
	})
	cs, _ := Structural{}.Crops(st, []recording.NoteAt{n1.At, n2.At}, h)
	if len(seen) != 2 || len(cs) != 2 || cs[0] != (recording.Box{1, 1, 1, 1}) || cs[1] != cs[0] {
		t.Errorf("hints saw %v, crops %v", seen, cs)
	}
}

type hintFunc func(string, recording.Box) (recording.Box, bool)

func (f hintFunc) Crop(url string, b recording.Box) (recording.Box, bool) { return f(url, b) }

func held(cs []recording.Box, b recording.Box) bool {
	for _, c := range cs {
		if c.Contains(b) {
			return true
		}
	}
	return false
}

// A region note crops its region plus the margin, never snapped (#470): the
// same region on the blog's header, where an element note goes full width.
func TestNoteCropsARegionByHand(t *testing.T) {
	n, st := fixtureNote(t, "blog", "blog-n1")
	region := recording.Box{400, 200, 120, 40}
	rn := &recording.Note{At: n.At, Region: &region}
	cs, err := Structural{}.NoteCrops(st, []*recording.Note{rn}, nil)
	if want := (recording.Box{400 - defaultMargin, 200 - defaultMargin, 120 + 2*defaultMargin, 40 + 2*defaultMargin}); err != nil || len(cs) != 1 || cs[0] != want {
		t.Errorf("region crop %v (%v), want %v", cs, err, want)
	}
	// With the element note it overlaps, the two share one crop.
	cs, err = Structural{}.NoteCrops(st, []*recording.Note{{At: n.At}, rn}, nil)
	if err != nil || len(cs) != 1 || !cs[0].Contains(region) || cs[0][2] != 1280 {
		t.Errorf("region and header crops %v (%v): want one full-width crop holding the region", cs, err)
	}
	// Crops is NoteCrops without regions.
	a, _ := Structural{}.Crops(st, []recording.NoteAt{n.At}, nil)
	b, _ := Structural{}.NoteCrops(st, []*recording.Note{{At: n.At}}, nil)
	if len(a) != 1 || len(b) != 1 || a[0] != b[0] {
		t.Errorf("Crops %v != NoteCrops %v", a, b)
	}
}
