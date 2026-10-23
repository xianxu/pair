package board

import (
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func submissionSketchRecords(n int) []recording.Record {
	b := recording.NewBuilder("submission-sketch")
	node := b.El("button", []string{"id", "save"})
	b.Page(recording.SegLive, "https://example.test", node)
	for i := 0; i < n; i++ {
		id := string(rune('a' + i))
		op := func(seq uint64) recording.SketchOperation {
			return recording.SketchOperation{Operation: id + string(rune('0'+seq)), Sequence: seq, Digest: strings.Repeat("a", 64)}
		}
		b.At(int64(i + 1)).Add(&recording.SketchBegin{SketchOperation: op(0), ID: id, Author: recording.Author{ID: "operator", Color: "#ff0000"}, Owner: "operator", Doc: 1, Tab: "tab1", VP: recording.Viewport{W: 1280, H: 800, DPR: 1}})
		b.Add(&recording.SketchStrokeBegin{SketchOperation: op(1), Sketch: id, ID: "p", At: recording.NoteAt{Node: node.ID, Box: &recording.Box{10, 20, 100, 40}}, Width: 3})
		b.Add(&recording.SketchPoints{SketchOperation: op(2), Sketch: id, ID: "p", Pts: []recording.SketchPoint{{20, 30, 0}, {40, 35, 1}}})
		b.Add(&recording.SketchStrokeEnd{SketchOperation: op(3), Sketch: id, ID: "p"})
		b.Add(&recording.SketchLabel{SketchOperation: op(4), Sketch: id, Label: "Move control"})
	}
	return b.Recs
}

func TestSubmissionSketchFreezeRejectsChangedCandidates(t *testing.T) {
	b := New(nil, nil)
	records := submissionSketchRecords(2)
	var first Selection
	for _, r := range records {
		if err := b.Put(r); err != nil {
			t.Fatal(err)
		}
		if v, ok := r.(*recording.SketchLabel); ok && v.Sketch == "a" {
			var err error
			first, err = b.Selection()
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if len(first.Sketches) != 1 || first.Sketches[0].ID != "a" || first.Sketches[0].NextSequence != 5 {
		t.Fatalf("selection %+v", first)
	}
	if _, err := b.FreezeSubmissionSelection(first.SketchGeneration, first.Sketches, first.Notes); err == nil {
		t.Fatal("stale preview froze changed candidate set")
	}
	current, err := b.Selection()
	if err != nil {
		t.Fatal(err)
	}
	frozen, err := b.FreezeSubmissionSelection(current.SketchGeneration, current.Sketches[:1], current.Notes)
	if err != nil {
		t.Fatal(err)
	}
	if len(frozen.Sketches) != 1 || frozen.Sketches[0] != first.Sketches[0] {
		t.Fatalf("freeze %+v", frozen)
	}
	bad := append([]SelectedSketch(nil), current.Sketches[:1]...)
	bad[0].Digest = strings.Repeat("b", 64)
	if _, err = b.FreezeSubmissionSelection(current.SketchGeneration, bad, current.Notes); err == nil {
		t.Fatal("forged version frozen")
	}
	b.Put(&recording.AnnotationClear{Who: recording.Author{ID: "operator"}})
	if _, err = b.FreezeSubmissionSelection(current.SketchGeneration, current.Sketches[:1], current.Notes); err == nil {
		t.Fatal("cleared preview frozen")
	}
}

func TestSubmissionSketchWakeExcludesPressureAndUnchangedUpdates(t *testing.T) {
	b := New(nil, nil)
	wake, stop := b.SubscribeSubmissionCandidates()
	defer stop()
	records := submissionSketchRecords(1)
	for _, r := range records[:len(records)-1] {
		b.Put(r)
	}
	select {
	case <-wake:
		t.Fatal("unfinished ink woke eligible preview")
	default:
	}
	before, _ := b.Selection()
	b.Put(records[len(records)-1])
	select {
	case <-wake:
	default:
		t.Fatal("finished sketch did not wake preview")
	}
	after, _ := b.Selection()
	if after.SketchGeneration == before.SketchGeneration {
		t.Fatal("eligible generation unchanged")
	}
	b.Put(&recording.Scroll{X: 0, Y: 10})
	select {
	case <-wake:
		t.Fatal("ordinary page update woke preview")
	default:
	}
	b.Close()
	select {
	case _, open := <-wake:
		if open {
			t.Fatal("subscription retained after close")
		}
	default:
		t.Fatal("subscription not closed")
	}
}
