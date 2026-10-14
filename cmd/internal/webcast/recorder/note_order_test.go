package recorder

import (
	"bytes"
	rec "github.com/xianxu/pair/cmd/internal/webcast/recording"
	"reflect"
	"testing"
)

func TestMirrorEditSurvivesFileMergeAtFrozenClock(t *testing.T) {
	for _, boundary := range []string{"pause", "end", "navigate", "clear"} {
		t.Run(boundary, func(t *testing.T) {
			m := newMirror()
			b := rec.NewBuilder("ordering")
			p := b.El("p", nil)
			b.Page(rec.SegLive, "http://x/", p)
			for _, r := range b.Recs {
				if !m.Emit(2, r) {
					t.Fatal(m.Examples)
				}
			}
			for _, r := range []rec.Record{&rec.Note{Header: rec.Header{K: "note"}, ID: "n", Author: rec.Author{ID: "a"}, Text: "first", At: rec.NoteAt{Node: p.ID}}, &rec.NoteEdit{ID: "n", Text: "edited"}} {
				if !m.Emit(2, r) {
					t.Fatal(m.Examples)
				}
			}
			switch boundary {
			case "pause", "end":
				if !m.Emit(2, &rec.Stop{Why: boundary}) {
					t.Fatal(m.Examples)
				}
			case "navigate":
				if !m.Emit(2, &rec.Seg{Seg: 2, Why: rec.SegNavigate, VP: rec.Viewport{W: 10, H: 10}}) {
					t.Fatal(m.Examples)
				}
				if !m.Emit(2, &rec.Doc{Doc: 900, URL: "http://y/", Tree: rec.Node{ID: 900, Y: rec.YDoc}}) {
					t.Fatal(m.Examples)
				}
			case "clear":
				if !m.Emit(2, &rec.AnnotationClear{Who: rec.Author{ID: rec.OperatorID}}) {
					t.Fatal(m.Examples)
				}
			}
			files := map[string][]byte{}
			for file, rs := range rec.Split(m.take()) {
				for _, r := range rs {
					line, err := rec.Encode(r)
					if err != nil {
						t.Fatal(err)
					}
					files[file] = append(append(files[file], line...), '\n')
				}
			}
			bad, replayed := rec.Validate(files)
			if len(bad) > 0 {
				t.Fatal(bad)
			}
			if !bytes.Equal(m.st.Canonical(), replayed.Canonical()) || !reflect.DeepEqual(m.st.Notes, replayed.Notes) {
				t.Fatalf("live/replay differs: live=%+v replay=%+v", m.st.Notes, replayed.Notes)
			}
		})
	}
}

func TestMirrorRefusedEmissionDoesNotAdvanceOrdering(t *testing.T) {
	m := newMirror()
	b := rec.NewBuilder("ordering")
	p := b.El("p", nil)
	b.Page(rec.SegLive, "http://x/", p)
	for _, r := range b.Recs {
		if !m.Emit(2, r) {
			t.Fatal(m.Examples)
		}
	}
	if !m.Emit(2, &rec.Note{ID: "n", Author: rec.Author{ID: "a"}, At: rec.NoteAt{Node: p.ID}}) {
		t.Fatal(m.Examples)
	}
	for _, bad := range []rec.Record{&rec.Stroke{}, &rec.Focus{ID: 999}} {
		if m.Emit(2, bad) {
			t.Fatal("accepted invalid record")
		}
		if m.st.T != 2 || m.lastKind != "note" {
			t.Fatalf("refusal advanced ordering: %d %s", m.st.T, m.lastKind)
		}
	}
	if !m.Emit(0, &rec.NoteEdit{ID: "n", Text: "edited"}) || m.st.T != 2 {
		t.Fatal("same-file event advanced timestamp")
	}
	if !m.Emit(0, &rec.Stop{Why: rec.StopPause}) || m.st.T != 3 {
		t.Fatal("cross-file event did not advance timestamp")
	}
}
