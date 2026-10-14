package board

import (
	"bytes"
	"context"
	rec "github.com/xianxu/pair/cmd/internal/webcast/recording"
	"reflect"
	"testing"
)

func TestWriterEditSurvivesFileMergeAtFrozenClock(t *testing.T) {
	for _, boundary := range []string{"pause", "end", "navigate", "clear"} {
		t.Run(boundary, func(t *testing.T) {
			sink := &rec.MemSink{}
			w := NewWriter(sink, func() int64 { return 2 })
			b := rec.NewBuilder("ordering")
			p := b.El("p", nil)
			b.At(2).Page(rec.SegLive, "http://x/", p)
			for _, r := range b.Recs {
				if err := w.Put(r); err != nil {
					t.Fatal(err)
				}
			}
			for _, r := range []rec.Record{&rec.Note{Header: rec.Header{K: "note"}, ID: "n", Author: rec.Author{ID: "a"}, Text: "first", At: rec.NoteAt{Node: p.ID}}, &rec.NoteEdit{ID: "n", Text: "edited"}} {
				if err := w.Emit(r, 1); err != nil {
					t.Fatal(err)
				}
			}
			var err error
			switch boundary {
			case "pause", "end":
				err = w.Put(&rec.Stop{Header: rec.Header{T: 2}, Why: boundary})
			case "navigate":
				err = w.Put(&rec.Seg{Header: rec.Header{T: 2}, Seg: 2, Why: rec.SegNavigate, VP: rec.Viewport{W: 10, H: 10}})
				if err == nil {
					err = w.Put(&rec.Doc{Header: rec.Header{T: 2}, Doc: 900, URL: "http://y/", Tree: rec.Node{ID: 900, Y: rec.YDoc}})
				}
			case "clear":
				err = w.ClearAnnotations(context.Background(), 1)
			}
			if err != nil {
				t.Fatal(err)
			}
			files := map[string][]byte{}
			for file, rs := range rec.Split(sink.Records()) {
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
			if !bytes.Equal(w.st.Canonical(), replayed.Canonical()) || !reflect.DeepEqual(w.st.Notes, replayed.Notes) {
				t.Fatalf("live/replay differs: live=%+v replay=%+v", w.st.Notes, replayed.Notes)
			}
		})
	}
}

func TestWriterRefusedEmissionDoesNotAdvanceOrdering(t *testing.T) {
	w := NewWriter(&rec.MemSink{}, func() int64 { return 2 })
	b := rec.NewBuilder("ordering")
	p := b.El("p", nil)
	b.At(2).Page(rec.SegLive, "http://x/", p)
	for _, r := range b.Recs {
		if err := w.Put(r); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Emit(&rec.Note{ID: "n", Author: rec.Author{ID: "a"}, At: rec.NoteAt{Node: p.ID}}, 1); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []rec.Record{&rec.Stroke{}, &rec.Focus{ID: 999}} {
		if w.Emit(bad, 1) == nil {
			t.Fatal("accepted invalid record")
		}
		if w.st.T != 2 || w.lastKind != "note" {
			t.Fatalf("refusal advanced ordering: %d %s", w.st.T, w.lastKind)
		}
	}
	if err := w.Put(&rec.NoteEdit{ID: "n", Text: "edited"}); err != nil || w.st.T != 2 {
		t.Fatalf("same-file stamp: %v %d", err, w.st.T)
	}
	if err := w.Put(&rec.Stop{Why: rec.StopPause}); err != nil || w.st.T != 3 {
		t.Fatalf("cross-file stamp: %v %d", err, w.st.T)
	}
}
