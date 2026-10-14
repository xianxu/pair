package recording

import (
	"strings"
	"testing"
)

// twoDocs: doc A (seg 1) with a note, paused, resumed (seg 2, same doc) with
// a note, then a navigation to doc B (seg 3) with a note.
func twoDocs() (*Builder, Node, Node) {
	b := NewBuilder("c")
	h := strings.Repeat("1", 64)
	b.Add(&Asset{URL: "http://x/a.png", Hash: h, MIME: "image/png"})
	divA := b.El("div", nil)
	docA := b.Page(SegLive, "http://x/a", divA)
	b.Add(&Note{ID: "a1", Author: Author{ID: "u"}, At: NoteAt{Node: divA.ID}})
	b.Add(&Note{ID: "old", Author: Author{ID: "u"}, At: NoteAt{Node: divA.ID}})
	b.Add(&Stop{Why: StopPause})
	b.Add(&Seg{Seg: b.NextSeg(), Why: SegResume, Tab: Tab{ID: "tab1", URL: "http://x/a"}, VP: Viewport{W: 1280, H: 800}})
	b.Snapshot("http://x/a", docA.C[1])
	b.Add(&Note{ID: "a2", Author: Author{ID: "u"}, At: NoteAt{Node: divA.ID}})
	b.Add(&Asset{URL: "http://x/b.png", Hash: h, MIME: "image/png"})
	divB := b.El("p", nil)
	b.Page(SegNavigate, "http://x/b", divB)
	b.Add(&Note{ID: "b1", Author: Author{ID: "u"}, At: NoteAt{Node: divB.ID}})
	b.Add(&NoteState{ID: "old", State: NoteResolved}) // a note of the earlier document
	return b, divA, divB
}

func noteIDs(s *State) string {
	var ids []string
	for _, n := range s.Notes {
		ids = append(ids, n.ID)
	}
	return strings.Join(ids, ",")
}

// The catch-up's DOM, assets and all retained notes are the whole
// recording's, at every cut point.
func TestCatchUpEquivalence(t *testing.T) {
	b, _, _ := twoDocs()
	for cut := 1; cut <= len(b.Recs); cut++ {
		if err := CheckCatchUp(b.Recs[:cut], 1<<62); err != nil {
			t.Fatalf("cut %d: %v", cut, err)
		}
	}
}

func TestCatchUpCarriesNotesAcrossResume(t *testing.T) {
	b, _, _ := twoDocs()
	// Cut just before the navigation: seg 2 (resume) is current; the
	// document began in seg 1.
	var cut int
	for i, r := range b.Recs {
		if s, ok := r.(*Seg); ok && s.Why == SegNavigate {
			cut = i
		}
	}
	cu, _ := Reduce(CatchUp(b.Recs[:cut]), 1<<62)
	if got := noteIDs(cu); got != "a1,old,a2" {
		t.Fatalf("notes %q", got)
	}
	cu, _ = Reduce(CatchUp(b.Recs), 1<<62)
	if got := noteIDs(cu); got != "a1,old,a2,b1" {
		t.Fatalf("after navigation, notes %q", got)
	}
}

func TestCatchUpWhilePaused(t *testing.T) {
	p := newPage()
	p.b.Add(&Stop{Why: StopPause})
	cu, err := Reduce(CatchUp(p.b.Recs), 1<<62)
	if err != nil || cu.Live || cu.Node(p.div.ID) == nil {
		t.Fatalf("paused catch-up: %v live=%v", err, cu.Live)
	}
}

func TestCatchUpCarriesControl(t *testing.T) {
	recs := DrivenRecording()
	for cut := 1; cut <= len(recs); cut++ {
		if err := CheckCatchUp(recs[:cut], recs[cut-1].Time()); err != nil {
			t.Fatalf("cut %d (%s at %d): %v", cut, recs[cut-1].Kind(), recs[cut-1].Time(), err)
		}
	}
	// The catch-up keeps one ctl from before the latest segment, not all.
	var before int
	for _, r := range CatchUp(recs[:len(recs)-1]) {
		if _, ok := r.(*Ctl); ok && r.Time() < 5600 {
			before++
		}
	}
	if before != 1 {
		t.Errorf("the catch-up carries %d ctl records from before the latest segment, want 1", before)
	}
}

// The catch-up keeps only the latest hover and ptr of the current segment
// (state, like the latest ctl), in place: a viewer joining gets the
// operator's pointer now, not its history. Equivalent at every cut,
// including one where the hovered element was removed after.
func TestCatchUpKeepsLatestHoverAndPtr(t *testing.T) {
	p := newPage()
	li := p.b.El("li", nil)
	p.b.Add(&Ins{P: p.ul.ID, Node: li})
	for i := 0; i < 5; i++ {
		p.b.At(int64(10 + i)).Add(&Hover{ID: p.div.ID}).Add(&Ptr{X: float64(i), Y: 1})
	}
	p.b.At(20).Add(&Hover{ID: li.ID}).Add(&Rm{ID: li.ID}).Add(&Ptr{X: 9, Y: 9})
	recs := p.b.Recs
	for cut := 1; cut <= len(recs); cut++ {
		if err := CheckCatchUp(recs[:cut], 1<<62); err != nil {
			t.Fatalf("cut %d: %v", cut, err)
		}
	}
	hovers, ptrs := 0, 0
	for _, r := range CatchUp(recs) {
		switch r.(type) {
		case *Hover:
			hovers++
		case *Ptr:
			ptrs++
		}
	}
	if hovers != 1 || ptrs != 1 {
		t.Fatalf("the catch-up carries %d hovers and %d ptrs, want 1 each", hovers, ptrs)
	}
	// Only the current segment's: a new segment clears them.
	p.b.At(30).Add(&Stop{Why: StopPause})
	p.b.Add(&Seg{Seg: p.b.NextSeg(), Why: SegNavigate, Tab: Tab{ID: "t"}, VP: Viewport{W: 1, H: 1, DPR: 1}})
	p.b.Snapshot("http://x/2", p.b.El("body", nil))
	for _, r := range CatchUp(p.b.Recs) {
		if r.Kind() == "hover" || r.Kind() == "ptr" {
			t.Fatalf("a %s from an earlier segment was carried", r.Kind())
		}
	}
	if err := CheckCatchUp(p.b.Recs, 1<<62); err != nil {
		t.Fatal(err)
	}
}

func TestCatchUpCarriesHistoricalEditsAndClear(t *testing.T) {
	b, _, _ := twoDocs()
	b.Add(&NoteEdit{ID: "a1", Text: "updated off-view"})
	b.Add(&AnnotationClear{Who: Author{ID: OperatorID}})
	for cut := 1; cut <= len(b.Recs); cut++ {
		if err := CheckCatchUp(b.Recs[:cut], 1<<62); err != nil {
			t.Fatalf("cut %d: %v", cut, err)
		}
	}
}
