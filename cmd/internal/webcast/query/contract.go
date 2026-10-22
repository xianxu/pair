package query

import (
	"reflect"
	"strings"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// ContractRecording is a form page where the user types a name and clicks
// Save, and a viewer notes the button: the input of the surface contract.
func ContractRecording() (recs []recording.Record, save recording.NodeID) {
	b := recording.NewBuilder("query-contract")
	name := b.El("input", []string{"id", "name"})
	btn := b.El("button", []string{"id", "save"}, b.Txt("Save"))
	b.Page(recording.SegLive, "http://app/form", b.El("form", []string{"id", "f"}, name, btn), b.El("p", []string{"id", "other"}, b.Txt("Elsewhere")))
	b.At(1000).Add(&recording.Focus{ID: name.ID})
	b.At(1500).Add(&recording.Input{ID: name.ID, V: recording.Str("Ada")})
	b.At(2000).Add(&recording.Click{ID: btn.ID, X: 100, Y: 40})
	b.At(2100).Add(&recording.Attr{ID: btn.ID, Name: "disabled", V: recording.Str("")})
	b.At(9000).Add(&recording.Scroll{ID: b.Recs[2].(*recording.Doc).Doc, Y: 100})
	b.At(10000).Add(&recording.Note{ID: "n1", Author: recording.Author{ID: "pm"}, Text: "Save stays disabled",
		At: recording.NoteAt{Node: btn.ID, Sel: "#save", Text: "Save"}})
	b.At(11000).Add(&recording.NoteState{ID: "n1", State: recording.NoteResolved})
	return b.Recs, btn.ID
}

// RunSurfaceContract checks a Surface. newSurface builds one over a
// recording. The real surface (#462) and FakeSurface both run it.
func RunSurfaceContract(t recording.TB, newSurface func(recs []recording.Record) Surface) {
	t.Helper()
	RunSketchSurfaceContract(t, newSurface)
	recs, save := ContractRecording()
	s := newSurface(recs)

	page, err := s.DOMAt(10000, "")
	if err != nil || !strings.Contains(page, "Save") || !strings.Contains(page, "Elsewhere") {
		t.Errorf("DOMAt(t, \"\"): %v\n%s", err, page)
	}
	sub, err := s.DOMAt(10000, "#save")
	if err != nil || !strings.Contains(sub, "Save") || strings.Contains(sub, "Elsewhere") {
		t.Errorf("DOMAt under #save must hold the button only: %v\n%s", err, sub)
	}
	if early, _ := s.DOMAt(2050, "#save"); strings.Contains(early, "disabled") {
		t.Errorf("DOMAt must show the page as of t, before the button was disabled:\n%s", early)
	}
	if _, err := s.DOMAt(10000, "#missing"); err == nil {
		t.Errorf("a selector matching nothing must be an error")
	}

	evs, err := s.Events(1500, 2100)
	if err != nil {
		t.Fatalf("events: %v", err)
	}
	var kinds []string
	for i, e := range evs {
		if e.Time() < 1500 || e.Time() > 2100 || (i > 0 && e.Time() < evs[i-1].Time()) {
			t.Errorf("event %s@%d is out of the window or order", e.Kind(), e.Time())
		}
		kinds = append(kinds, e.Kind())
	}
	if got := strings.Join(kinds, ","); got != "input,click,attr" {
		t.Errorf("events in 1500..2100: %s", got)
	}

	ns, err := s.Notes(9500)
	if err != nil || len(ns) != 1 {
		t.Fatalf("notes: %v %d", err, len(ns))
	}
	n := ns[0]
	if n.Note.ID != "n1" || n.Note.T != 10000 || !strings.Contains(n.Element, "Save") || n.Context.Node != save {
		t.Errorf("note view %+v", n)
	}
	// An agent asks whether a note is settled now, not as it was made.
	if n.Note.State != recording.NoteResolved {
		t.Errorf("a note resolved after it was made reads %q, want its final state", n.Note.State)
	}
	// Edits retain creation context and latest resolution, including after clear.
	edited := append([]recording.Record(nil), recs[:len(recs)-1]...)
	edited = append(edited, &recording.NoteEdit{Header: recording.Header{T: 10500}, ID: "n1", Text: "Updated explanation"},
		&recording.NoteState{Header: recording.Header{T: 11000}, ID: "n1", State: recording.NoteResolved, Who: &recording.Author{ID: "operator"}},
		&recording.AnnotationClear{Header: recording.Header{T: 12000}, Who: recording.Author{ID: "operator"}})
	ev, err := newSurface(edited).Notes(9500)
	if err != nil || len(ev) != 1 {
		t.Fatalf("edited notes: %v %d", err, len(ev))
	}
	en := ev[0]
	if en.Note.Text != "Updated explanation" || !en.Note.Edited || en.Note.ResolvedBy == nil || en.Note.ResolvedBy.ID != "operator" || en.Note.T != 10000 || !strings.Contains(en.Element, "Save") || en.Context.Node != save {
		t.Errorf("edited projection: %+v", en)
	}
	var before []string
	for _, r := range n.Before {
		before = append(before, r.Kind())
	}
	if got := strings.Join(before, ","); got != "focus,input,click,scroll" {
		t.Errorf("actions before the note: %s, want focus,input,click,scroll", got)
	}
	// Every input the real surface takes, the fake answers the same way:
	// an empty recording has no page, no events and no notes.
	empty := newSurface(nil)
	if _, err := empty.DOMAt(0, ""); err == nil {
		t.Errorf("an empty recording has no page")
	}
	if evs, err := empty.Events(0, 1000); err != nil || len(evs) != 0 {
		t.Errorf("an empty recording's events: %v %d", err, len(evs))
	}
	if ns, err := empty.Notes(1000); err != nil || len(ns) != 0 {
		t.Errorf("an empty recording's notes: %v %d", err, len(ns))
	}
	if ns, _ := s.Notes(1500); len(ns[0].Before) != 1 {
		t.Errorf("a shorter window keeps only the scroll: %d actions", len(ns[0].Before))
	}
}

// RunSketchSurfaceContract checks source facts and exact version selection through
// the same public surface used by the CLI and submission evidence builder.
func RunSketchSurfaceContract(t recording.TB, newSurface func([]recording.Record) Surface) {
	t.Helper()
	recs, node := ContractRecording()
	begin := &recording.SketchBegin{Header: recording.Header{T: 12000}, SketchOperation: recording.SketchOperation{Operation: "begin", Digest: strings.Repeat("a", 64)}, ID: "shape", Author: recording.Author{ID: "operator", Color: "#ff0000"}, Owner: "operator", Doc: 1, Tab: "tab1", VP: recording.Viewport{W: 1280, H: 800, DPR: 1}}
	recs = append(recs, begin, &recording.SketchStrokeBegin{Header: recording.Header{T: 12001}, SketchOperation: recording.SketchOperation{Operation: "stroke", Sequence: 1, Digest: strings.Repeat("a", 64)}, Sketch: "shape", ID: "p", At: recording.NoteAt{Node: node, Box: &recording.Box{0, 0, 100, 100}}, Width: 3}, &recording.SketchPoints{Header: recording.Header{T: 12002}, SketchOperation: recording.SketchOperation{Operation: "points", Sequence: 2, Digest: strings.Repeat("a", 64)}, Sketch: "shape", ID: "p", Pts: []recording.SketchPoint{{10, 20, .5}}}, &recording.Stop{Header: recording.Header{T: 12003}, Why: recording.StopPause})
	surface := newSurface(recs)
	got, err := surface.SketchesAt(12003)
	st, reduceErr := recording.Reduce(recs, 12003)
	if err != nil || reduceErr != nil || len(got) != 1 {
		t.Fatalf("sketch surface: query=%v reducer=%v views=%d", err, reduceErr, len(got))
	}
	if !reflect.DeepEqual(got[0].History, st.Sketches.Snapshot().Sketches[0]) || got[0].Elements.Anchors[0].Node != node || got[0].Elements.Anchors[0].ID != "save" || got[0].Source.T != 12000 {
		t.Fatalf("sketch source/history mismatch: %+v", got[0])
	}
	selected, err := surface.SketchesAt(12003, SketchSelection{ID: "shape", NextSequence: 2, Digest: strings.Repeat("a", 64)})
	if err != nil || len(selected) != 1 || selected[0].History.NextSequence != 2 || selected[0].History.Strokes[0].Sealed {
		t.Fatalf("exact sketch prefix: %v %+v", err, selected)
	}
	if _, err = surface.SketchesAt(12003, SketchSelection{ID: "shape", NextSequence: 2, Digest: strings.Repeat("b", 64)}); err == nil {
		t.Fatalf("sketch surface accepted wrong selected digest")
	}
	if empty, err := newSurface(nil).SketchesAt(0); err != nil || len(empty) != 0 {
		t.Fatalf("empty sketch query: %v %+v", err, empty)
	}
}
