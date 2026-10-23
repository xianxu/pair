package board

import (
	"fmt"
	"math"
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast/query"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func TestFeedbackNoteChangesInvalidateSelection(t *testing.T) {
	r := newRig(t).page(recording.SegLive)
	before, _ := r.bd.Selection()
	id, err := r.note(r.p1.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	after, _ := r.bd.Selection()
	if after.SketchGeneration != before.SketchGeneration+1 {
		t.Fatalf("note creation generation=%d, want %d", after.SketchGeneration, before.SketchGeneration+1)
	}
	for _, change := range []recording.Record{
		&recording.NoteEdit{ID: id, Text: "revised"},
		&recording.NoteState{ID: id, State: recording.NoteResolved},
		&recording.NoteState{ID: id, State: recording.NoteOpen},
	} {
		before = after
		if err := r.w.Emit(change, r.bd.st.DocSeg); err != nil {
			t.Fatal(err)
		}
		after, _ = r.bd.Selection()
		if after.SketchGeneration != before.SketchGeneration+1 {
			t.Fatalf("%T did not invalidate note revision", change)
		}
	}
}

func TestFeedbackContextsMatchExactQueryPrefixesAndIgnoreDuplicates(t *testing.T) {
	b := New(nil, nil)
	original := submissionSketchRecords(1)
	var records []recording.Record
	var top recording.NodeID
	for _, r := range original {
		if d, ok := r.(*recording.Doc); ok {
			top = d.Doc
		}
		if _, ok := r.(*recording.SketchBegin); ok {
			scroll := &recording.Scroll{ID: top, X: 4, Y: 30}
			scroll.SetTime(r.Time())
			records = append(records, scroll)
		}
		if _, ok := r.(*recording.SketchLabel); ok {
			scroll := &recording.Scroll{ID: top, X: 8, Y: 60}
			scroll.SetTime(r.Time())
			records = append(records, scroll)
		}
		records = append(records, r)
	}
	for _, r := range records {
		b.Put(r)
	}
	selected, err := b.Selection()
	if err != nil {
		t.Fatal(err)
	}
	v := selected.Sketches[0]
	views, err := query.New(records, nil).SketchesAt(math.MaxInt64, query.SketchSelection{ID: v.ID, NextSequence: v.NextSequence, Digest: v.Digest})
	if err != nil {
		t.Fatal(err)
	}
	if v.Source != views[0].Source {
		t.Fatalf("source %+v != query %+v", v.Source, views[0].Source)
	}
	if v.Finish.RecordIndex != len(records)-1 || v.Finish.Scroll != [2]float64{8, 60} || v.Finish.Doc != 1 || v.Finish.Tab != "tab1" {
		t.Fatalf("finish %+v", v.Finish)
	}
	oracle := recording.NewState()
	for _, r := range records[:v.Finish.RecordIndex+1] {
		if err := oracle.Apply(r); err != nil {
			t.Fatal(err)
		}
	}
	wantFinish := query.SketchSourceFacts{T: oracle.T, RecordIndex: len(records) - 1, Doc: oracle.DocSeg, Tab: oracle.Tab.ID, URL: oracle.URL, Title: oracle.Title, VP: oracle.VP, Scroll: oracle.Scrolls[oracle.Top.ID]}
	if v.Finish != wantFinish {
		t.Fatalf("finish %+v != independent prefix %+v", v.Finish, wantFinish)
	}
	// Later same-time records and duplicate accepted receipts are not a new finish.
	later := &recording.Scroll{ID: top, Y: 99}
	later.SetTime(v.Finish.T)
	b.Put(later)
	b.Put(records[len(records)-1])
	b.Put(original[3]) // duplicate BEGIN
	again, _ := b.Selection()
	if again.Sketches[0] != v || again.SketchGeneration != selected.SketchGeneration {
		t.Fatalf("duplicate changed witnesses/generation: %+v", again)
	}
	bad := append([]SelectedSketch(nil), selected.Sketches...)
	bad[0].Finish.RecordIndex++
	if _, err = b.FreezeSubmissionSelection(again.SketchGeneration, bad, again.Notes); err == nil {
		t.Fatal("forged finish froze")
	}
	clearEvent := &recording.AnnotationClear{Who: recording.Author{ID: recording.OperatorID}}
	clearEvent.SetTime(b.st.T)
	b.Put(clearEvent)
	if b.broken != nil {
		t.Fatal(b.broken)
	}
	if len(b.feedback.sketches) != 0 {
		t.Fatal("clear retained scalar witnesses")
	}
}

func TestFeedbackContextWitnessesBoundedBySketchLimit(t *testing.T) {
	b := New(nil, nil)
	for _, r := range submissionSketchRecords(1)[:3] {
		b.Put(r)
	}
	for i := 0; i < recording.SketchMaxCount; i++ {
		id := fmt.Sprintf("s%d", i)
		for _, r := range submissionSketchRecords(1)[3:] {
			switch v := r.(type) {
			case *recording.SketchBegin:
				v.ID = id
			case *recording.SketchStrokeBegin:
				v.Sketch = id
			case *recording.SketchPoints:
				v.Sketch = id
			case *recording.SketchStrokeEnd:
				v.Sketch = id
			case *recording.SketchLabel:
				v.Sketch = id
			}
			b.Put(r)
		}
	}
	if b.broken != nil {
		t.Fatal(b.broken)
	}
	s, err := b.Selection()
	if err != nil {
		t.Fatal(err)
	}
	if len(b.feedback.sketches) != 256 || len(s.Sketches) != 256 {
		t.Fatalf("bounded witnesses=%d candidates=%d", len(b.feedback.sketches), len(s.Sketches))
	}
	clearEvent := &recording.AnnotationClear{Who: recording.Author{ID: recording.OperatorID}}
	clearEvent.SetTime(b.st.T)
	b.Put(clearEvent)
	if len(b.feedback.sketches) != 0 {
		t.Fatal("clear retained witness map entries")
	}
}

func TestFeedbackFreezeRequiresExactDisplayedNoteOrderAndRevision(t *testing.T) {
	r := newRig(t).page(recording.SegLive)
	r.note(r.p1.ID, 0)
	r.note(r.p2.ID, 1)
	s, _ := r.bd.Selection()
	for _, notes := range [][]SelectedNote{nil, s.Notes[:1], {s.Notes[1], s.Notes[0]}, {{ID: s.Notes[0].ID, Revision: s.Notes[0].Revision + 1}, s.Notes[1]}} {
		if _, err := r.bd.FreezeSubmissionSelection(s.SketchGeneration, nil, notes); err == nil {
			t.Fatalf("accepted changed displayed notes %+v", notes)
		}
	}
	frozen, err := r.bd.FreezeSubmissionSelection(s.SketchGeneration, nil, s.Notes)
	if err != nil {
		t.Fatal(err)
	}
	s.Notes[0].ID = "mutated"
	if frozen.Notes[0].ID == "mutated" {
		t.Fatal("frozen notes alias caller")
	}
}

func TestFeedbackContextRecordWorkIsLinearAndPressureDoesNotWake(t *testing.T) {
	for _, n := range []int{32, 64} {
		b := New(nil, nil)
		records := submissionSketchRecords(1)
		for _, r := range records {
			b.Put(r)
		}
		before, _ := b.Selection()
		for i := 0; i < n; i++ {
			update := &recording.Scroll{ID: b.st.Top.ID, Y: float64(i)}
			update.SetTime(b.st.T)
			b.Put(update)
		}
		if b.broken != nil {
			t.Fatal(b.broken)
		}
		after, _ := b.Selection()
		if b.feedback.records != len(records)+n || len(b.feedback.sketches) != 1 {
			t.Fatalf("metadata growth %+v", b.feedback)
		}
		if after.SketchGeneration != before.SketchGeneration {
			t.Fatal("DOM/view updates invalidated fixed feedback")
		}
	}
}
