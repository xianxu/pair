package recording

import (
	"fmt"
	"testing"
)

func TestSketchReviewLabelAfterCastEndRefused(t *testing.T) {
	s := sketchFixture(t)
	applySketch(t, s, sketchBegin())
	applySketch(t, s, &Stop{Why: StopEnd})
	refuseSketch(t, s, &SketchLabel{SketchOperation: sketchOp(1), Sketch: "sk", Label: "too late"})
}

func TestSketchReviewIdentityAndTotalPointBounds(t *testing.T) {
	t.Run("all sketch identities remain counted after completion", func(t *testing.T) {
		s := sketchFixture(t)
		for i := 0; i < SketchMaxCount; i++ {
			b := sketchBegin()
			b.ID = fmt.Sprint("sk-", i)
			applySketch(t, s, b)
			applySketch(t, s, &SketchLabel{SketchOperation: sketchOp(1), Sketch: b.ID, Label: "done"})
		}
		b := sketchBegin()
		b.ID = "overflow"
		refuseSketch(t, s, b)
	})
	t.Run("empty and undone strokes remain counted", func(t *testing.T) {
		s := sketchFixture(t)
		applySketch(t, s, sketchBegin())
		seq := uint64(1)
		for i := 0; i < SketchMaxStrokes; i++ {
			id := fmt.Sprint("s-", i)
			applySketch(t, s, sketchStroke(seq, id))
			seq++
			applySketch(t, s, &SketchStrokeEnd{SketchOperation: sketchOp(seq), Sketch: "sk", ID: id})
			seq++
		}
		refuseSketch(t, s, sketchStroke(seq, "overflow"))
	})
	t.Run("undone points remain counted", func(t *testing.T) {
		s := sketchFixture(t)
		applySketch(t, s, sketchBegin())
		seq := uint64(1)
		for stroke := 0; stroke < 4; stroke++ {
			id := fmt.Sprint("s-", stroke)
			applySketch(t, s, sketchStroke(seq, id))
			seq++
			for chunk := 0; chunk < SketchMaxPointsPerStroke/SketchMaxPointsPerChunk; chunk++ {
				applySketch(t, s, &SketchPoints{SketchOperation: sketchOp(seq), Sketch: "sk", ID: id, Chunk: uint64(chunk), Pts: make([]SketchPoint, SketchMaxPointsPerChunk)})
				seq++
			}
			applySketch(t, s, &SketchStrokeEnd{SketchOperation: sketchOp(seq), Sketch: "sk", ID: id})
			seq++
			applySketch(t, s, &SketchUndo{SketchOperation: sketchOp(seq), Sketch: "sk", ID: id})
			seq++
		}
		applySketch(t, s, sketchStroke(seq, "overflow"))
		seq++
		refuseSketch(t, s, &SketchPoints{SketchOperation: sketchOp(seq), Sketch: "sk", ID: "overflow", Pts: []SketchPoint{{1, 1, 1}}})
	})
}

func TestSketchReviewIncomingOwnership(t *testing.T) {
	s := sketchFixture(t)
	b := sketchBegin()
	applySketch(t, s, b)
	b.ID = "mutated"
	b.Author.Color = "invalid"
	begin := sketchStroke(1, "s1")
	begin.At.Stable = map[string]string{"id": "original"}
	begin.At.Path = []string{"main", "button"}
	applySketch(t, s, begin)
	begin.At.Box[0] = 99
	begin.At.Stable["id"] = "mutated"
	begin.At.Path[0] = "mutated"
	pts := []SketchPoint{{1, 2, .5}}
	applySketch(t, s, &SketchPoints{SketchOperation: sketchOp(2), Sketch: "sk", ID: "s1", Pts: pts})
	pts[0][0] = 999
	v := s.Sketches.Snapshot().Sketches[0]
	if v.Begin.ID != "sk" || v.Begin.Author.Color != "#ff0000" || v.Strokes[0].Begin.At.Box[0] != 0 || v.Strokes[0].Begin.At.Stable["id"] != "original" || v.Strokes[0].Begin.At.Path[0] != "main" || v.Strokes[0].Pts[0][0] != 1 {
		t.Fatal("incoming mutable payload aliases core authority")
	}
}
