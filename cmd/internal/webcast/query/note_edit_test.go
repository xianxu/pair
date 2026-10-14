package query

import (
	"bytes"
	"github.com/xianxu/pair/cmd/internal/webcast/notes"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
	"strings"
	"testing"
)

func TestEditedNoteQueryAfterClear(t *testing.T) {
	recs, _ := ContractRecording()
	edit := &recording.NoteEdit{Header: recording.Header{T: 10500}, ID: "n1", Text: "Updated explanation"}
	recs = append(recs[:len(recs)-1], edit, &recording.NoteState{Header: recording.Header{T: 11000}, ID: "n1", State: recording.NoteResolved, Who: &recording.Author{ID: "operator", Name: "Ada"}}, &recording.AnnotationClear{Header: recording.Header{T: 12000}, Who: recording.Author{ID: "operator"}})
	for _, s := range []Surface{New(recs, notes.FakeModel{}), FakeSurface{Records: recs, Model: notes.FakeModel{}}} {
		ns, err := s.Notes(9500)
		if err != nil {
			t.Fatal(err)
		}
		n := ns[0]
		if !n.Note.Edited || n.Note.Text != "Updated explanation" || n.Note.ResolvedBy == nil || n.Note.ResolvedBy.Name != "Ada" || n.Note.T != 10000 || !strings.Contains(n.Element, "Save") {
			t.Fatalf("lost edited state/context: %+v", n.Note)
		}
		if text := NoteText(n, 9500); !strings.Contains(text, "edited") || !strings.Contains(text, "resolved by Ada") {
			t.Fatal(text)
		}
		var out bytes.Buffer
		if err := notesJSON(ns, &out); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), `"edited": true`) || !strings.Contains(out.String(), `"resolved_by"`) {
			t.Fatal(out.String())
		}
	}
}

func TestNoteEditEventNarration(t *testing.T) {
	recs, _ := ContractRecording()
	recs = append(recs[:len(recs)-1], &recording.NoteEdit{Header: recording.Header{T: 10500}, ID: "n1", Text: "updated"}, &recording.NoteState{Header: recording.Header{T: 11000}, ID: "n1", State: recording.NoteResolved, Who: &recording.Author{ID: "operator", Name: "Ada"}})
	lines, err := EventLines(recs, 10500, 11000)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Join(lines, "\n")
	if !strings.Contains(text, `note n1 edited: "updated"`) || !strings.Contains(text, "note n1 resolved by Ada") {
		t.Fatal(text)
	}
	legacy, _ := ContractRecording()
	lines, err = EventLines(legacy, 11000, 11000)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(lines, "\n"), "by") {
		t.Fatal("legacy unknown actor invented")
	}
}
