package recording

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestNoteEditWireAndProjection(t *testing.T) {
	b := NewBuilder("edit")
	p := b.El("p", nil)
	b.Page(SegLive, "http://x/", p)
	original := &Note{ID: "n", Author: ana, Text: "original", At: NoteAt{Node: p.ID}}
	b.At(1).Add(original)
	st, err := Reduce(b.Recs, 1)
	if err != nil {
		t.Fatal(err)
	}
	view := st.Notes[0]
	wire := []byte(`{"t":2,"k":"note-edit","id":"n","text":"updated"}`)
	edit, err := Decode(wire)
	if err != nil {
		t.Fatal(err)
	}
	if FileOf(edit.Kind()) != FileNotes {
		t.Fatal("edit must live in notes.jsonl")
	}
	if err := st.Apply(edit); err != nil {
		t.Fatal(err)
	}
	if st.Notes[0] != view || view.Text != "updated" || original.Text != "original" || view.T != 1 || st.NoteUnchanged("n", 0) {
		t.Fatalf("edit lost identity, creation, or revision: %+v", view)
	}
	if len(st.Notes) != 1 {
		t.Fatal("edit created a note")
	}
}

func TestNoteEditInvalidWire(t *testing.T) {
	for _, text := range []string{"", " \n\t", strings.Repeat("x", 1025), "bad\u0000", "bad\u202e"} {
		// Decode and the reducer share the typed event, including text validation.
		value, _ := json.Marshal(text)
		raw := `{"t":2,"k":"note-edit","id":"n","text":` + string(value) + `}`
		if r, err := Decode([]byte(raw)); err == nil && r.check() == nil {
			t.Errorf("accepted invalid text %q", text)
		}
	}
}

func TestNoteEditRefusalIsAtomic(t *testing.T) {
	for _, mode := range []string{"unknown", "resolved", "paused", "ended", "old-document", "cleared"} {
		t.Run(mode, func(t *testing.T) {
			b := NewBuilder("edit")
			p := b.El("p", nil)
			b.Page(SegLive, "http://x/", p)
			b.At(1).Add(&Note{ID: "n", Author: ana, Text: "original", At: NoteAt{Node: p.ID}})
			switch mode {
			case "resolved":
				b.At(2).Add(&NoteState{ID: "n", State: NoteResolved})
			case "paused":
				b.At(2).Add(&Stop{Why: StopPause})
			case "ended":
				b.At(2).Add(&Stop{Why: StopEnd})
			case "old-document":
				b.At(2).Page(SegNavigate, "http://y/")
			case "cleared":
				b.At(2).Add(&AnnotationClear{Who: Author{ID: OperatorID}})
			}
			st, err := Reduce(b.Recs, 2)
			if err != nil {
				t.Fatal(err)
			}
			before, _ := Reduce(b.Recs, 2)
			id := "n"
			if mode == "unknown" {
				id = "missing"
			}
			edit, err := Decode([]byte(`{"t":3,"k":"note-edit","id":"` + id + `","text":"updated"}`))
			if err != nil {
				t.Fatal(err)
			}
			if st.Apply(edit) == nil {
				t.Fatal("accepted edit")
			}
			if !reflect.DeepEqual(st, before) {
				t.Fatal("failed edit mutated state")
			}
		})
	}
}

func TestNoteEditGeneratedPrefixes(t *testing.T) {
	for count := 1; count <= 8; count++ {
		b := NewBuilder("edits")
		p := b.El("p", nil)
		b.Page(SegLive, "http://x/", p)
		original := &Note{ID: "n", Owner: "public-digest", Author: ana, Text: "original", At: NoteAt{Node: p.ID}}
		b.At(1).Add(original)
		for i := 0; i < count; i++ {
			b.At(int64(2 + i*4)).Add(&NoteEdit{ID: "n", Text: strings.Repeat("x", i+1)})
			b.At(int64(3 + i*4)).Add(&NoteState{ID: "n", State: NoteResolved, Who: &bo})
			b.At(int64(4 + i*4)).Add(&NoteState{ID: "n", State: NoteOpen})
		}
		b.At(100).Add(&Stop{Why: StopPause})
		b.At(101).Page(SegResume, "http://x/", b.El("p", nil))
		b.At(102).Add(&NoteEdit{ID: "n", Text: "after resume"})
		b.At(103).Add(&AnnotationClear{Who: Author{ID: OperatorID}})
		b.At(104).Page(SegNavigate, "http://y/", b.El("p", nil))
		for cut := 1; cut <= len(b.Recs); cut++ {
			if err := CheckCatchUp(b.Recs[:cut], 1<<62); err != nil {
				t.Fatalf("count=%d cut=%d: %v", count, cut, err)
			}
		}
		st, err := Reduce(b.Recs, 102)
		if err != nil {
			t.Fatal(err)
		}
		n := st.Notes[0]
		if n.Revision != uint64(count*3+1) || n.ResolvedBy != nil || !n.Edited || n.Text != "after resume" || n.Owner != "public-digest" || original.Text != "original" {
			t.Fatalf("projection drift: %+v", n)
		}
	}
}

func TestNoteEditMaximumEncodedSize(t *testing.T) {
	// Each edit is bounded; recordings inherit the existing no-total-cap and
	// operator-delete retention policy rather than silently pruning history.
	line, err := Encode(&NoteEdit{ID: "n1", Text: strings.Repeat("<", 1024)})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("maximum escaped text edit, two-byte ID: %d encoded bytes", len(line))
	if len(line) > 8<<10 {
		t.Fatal("edit exceeds existing note body budget")
	}
}

func TestNoteResolutionAttribution(t *testing.T) {
	b := NewBuilder("resolution")
	p := b.El("p", nil)
	b.Page(SegLive, "http://x/", p)
	b.Add(&Note{ID: "n", Author: ana, At: NoteAt{Node: p.ID}})
	st, err := Reduce(b.Recs, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, who := range []*Author{&bo, nil, &ana} {
		if err := st.Apply(&NoteState{ID: "n", State: NoteResolved, Who: who}); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(st.Notes[0].ResolvedBy, who) {
			t.Fatal("actor was fabricated or lost")
		}
		if who != nil && st.Notes[0].ResolvedBy == who {
			t.Fatal("actor record was aliased")
		}
		if err := st.Apply(&NoteState{ID: "n", State: NoteOpen}); err != nil {
			t.Fatal(err)
		}
		if st.Notes[0].ResolvedBy != nil {
			t.Fatal("reopening kept prior resolver")
		}
	}
}
