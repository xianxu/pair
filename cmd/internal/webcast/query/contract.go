package query

import (
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
	return b.Recs, btn.ID
}

// RunSurfaceContract checks a Surface. newSurface builds one over a
// recording. The real surface (#462) and FakeSurface both run it.
func RunSurfaceContract(t recording.TB, newSurface func(recs []recording.Record) Surface) {
	t.Helper()
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
	var before []string
	for _, r := range n.Before {
		before = append(before, r.Kind())
	}
	if got := strings.Join(before, ","); got != "focus,input,click,scroll" {
		t.Errorf("actions before the note: %s, want focus,input,click,scroll", got)
	}
	if ns, _ := s.Notes(1500); len(ns[0].Before) != 1 {
		t.Errorf("a shorter window keeps only the scroll: %d actions", len(ns[0].Before))
	}
}
