package recording

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"
)

func sketchOp(n uint64) SketchOperation {
	return SketchOperation{Operation: fmt.Sprintf("op-%d", n), Sequence: n, Digest: strings.Repeat("a", 64)}
}
func sketchBegin() *SketchBegin {
	return &SketchBegin{SketchOperation: sketchOp(0), ID: "sk", Author: Author{ID: OperatorID, Color: "#ff0000"}, Owner: OperatorID, Doc: 1, Tab: "tab1", VP: Viewport{W: 100, H: 100, DPR: 1}}
}
func sketchStroke(n uint64, id string) *SketchStrokeBegin {
	return &SketchStrokeBegin{SketchOperation: sketchOp(n), Sketch: "sk", ID: id, At: NoteAt{Node: 99, Box: &Box{0, 0, 100, 100}}, Width: 4}
}
func sketchFixture(t *testing.T) *State {
	t.Helper()
	b := NewBuilder("sketch")
	b.Page(SegLive, "http://x/")
	s, err := Reduce(b.Recs, 1<<62)
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func applySketch(t *testing.T, s *State, r Record) {
	t.Helper()
	r.SetTime(s.T + 1)
	if err := s.Apply(r); err != nil {
		t.Fatalf("%s: %v", r.Kind(), err)
	}
}
func sketchJSON(s *State) string { b, _ := json.Marshal(s.Sketches.Snapshot()); return string(b) }
func refuseSketch(t *testing.T, s *State, r Record) {
	t.Helper()
	before := sketchJSON(s)
	oldT := s.T
	r.SetTime(oldT + 1)
	if err := s.Apply(r); err == nil {
		t.Fatalf("accepted %s", r.Kind())
	}
	if before != sketchJSON(s) || oldT != s.T {
		t.Fatal("refusal mutated state")
	}
}

func TestSketchHistoryReceiptsAndBranches(t *testing.T) {
	s := sketchFixture(t)
	applySketch(t, s, sketchBegin())
	applySketch(t, s, sketchStroke(1, "s1"))
	p := &SketchPoints{SketchOperation: sketchOp(2), Sketch: "sk", ID: "s1", Chunk: 0, Pts: []SketchPoint{{1, 2, 0}, {3, 4, .5}}}
	applySketch(t, s, p)
	before := sketchJSON(s)
	oldT := s.T
	p.SetTime(oldT + 9)
	if err := s.Apply(p); err != nil {
		t.Fatal(err)
	}
	if before != sketchJSON(s) || s.T != oldT {
		t.Fatal("retry changed state")
	}
	forged := *p
	forged.Pts = []SketchPoint{{7, 8, 1}}
	refuseSketch(t, s, &forged)
	gap := *p
	gap.SketchOperation = sketchOp(9)
	gap.Chunk = 1
	refuseSketch(t, s, &gap)
	applySketch(t, s, &SketchStrokeEnd{SketchOperation: sketchOp(3), Sketch: "sk", ID: "s1"})
	applySketch(t, s, &SketchUndo{SketchOperation: sketchOp(4), Sketch: "sk", ID: "s1"})
	v := s.Sketches.Snapshot().Sketches[0]
	if len(v.Visible) != 0 || !reflect.DeepEqual(v.Redo, []string{"s1"}) {
		t.Fatalf("undo %+v", v)
	}
	applySketch(t, s, &SketchRedo{SketchOperation: sketchOp(5), Sketch: "sk", ID: "s1"})
	applySketch(t, s, &SketchUndo{SketchOperation: sketchOp(6), Sketch: "sk", ID: "s1"})
	applySketch(t, s, sketchStroke(7, "s2"))
	applySketch(t, s, &SketchStrokeEnd{SketchOperation: sketchOp(8), Sketch: "sk", ID: "s2"})
	v = s.Sketches.Snapshot().Sketches[0]
	if len(v.Redo) != 0 || len(v.Visible) != 0 || len(v.Strokes) != 2 {
		t.Fatalf("empty branch %+v", v)
	}
	applySketch(t, s, &SketchLabel{SketchOperation: sketchOp(9), Sketch: "sk", Label: "design"})
	refuseSketch(t, s, &SketchLabel{SketchOperation: sketchOp(10), Sketch: "sk", Label: "replace"})
	refuseSketch(t, s, sketchStroke(10, "s3"))
	// Projections own copies, including nested anchors and original pressure paths.
	v.Strokes[0].Pts[0][0] = 999
	v.Begin.Author.Color = "bad"
	v.Strokes[0].Begin.At.Box[0] = 999
	got := s.Sketches.Snapshot().Sketches[0]
	if got.Strokes[0].Pts[0][0] != 1 || got.Begin.Author.Color != "#ff0000" || got.Strokes[0].Begin.At.Box[0] != 0 {
		t.Fatal("projection aliases authority")
	}
}

func TestSketchContextAtomicityAndLateLabel(t *testing.T) {
	s := sketchFixture(t)
	applySketch(t, s, sketchBegin())
	applySketch(t, s, sketchStroke(1, "s1"))
	applySketch(t, s, &SketchPoints{SketchOperation: sketchOp(2), Sketch: "sk", ID: "s1", Pts: []SketchPoint{{1, 2, 1}}})
	who := Author{ID: "driver", Color: "#00ff00"}
	request := &Ctl{Op: CtlRequest, Who: who, Control: Control{Pending: []Author{who}}}
	applySketch(t, s, request)
	if s.Sketches.Snapshot().Active != "sk" {
		t.Fatal("request interrupted")
	}
	bad := &Ctl{Op: CtlGrant, Who: who, Control: Control{}}
	before := sketchJSON(s)
	oldT := s.T
	bad.SetTime(oldT + 1)
	if s.Apply(bad) == nil {
		t.Fatal("invalid ctl accepted")
	}
	if before != sketchJSON(s) || s.T != oldT || s.Control.Driver != nil {
		t.Fatal("ctl refusal not atomic")
	}
	applySketch(t, s, &Ctl{Op: CtlGrant, Who: who, Control: Control{Driver: &who}})
	v := s.Sketches.Snapshot().Sketches[0]
	if v.Phase != "interrupted" || v.Reason != "handoff" || !v.Strokes[0].Truncated || len(v.Visible) != 1 {
		t.Fatalf("handoff %+v", v)
	}
	// Implicitly sealed end is acknowledged once, without another visible push.
	applySketch(t, s, &SketchStrokeEnd{SketchOperation: sketchOp(3), Sketch: "sk", ID: "s1"})
	applySketch(t, s, &SketchLabel{SketchOperation: sketchOp(4), Sketch: "sk", Label: "partial"})
	v = s.Sketches.Snapshot().Sketches[0]
	if v.Phase != "interrupted" || v.Label != "partial" || len(v.Visible) != 1 {
		t.Fatal("late metadata lost interruption")
	}
	applySketch(t, s, &AnnotationClear{Who: Author{ID: OperatorID}})
	refuseSketch(t, s, &SketchLabel{SketchOperation: sketchOp(5), Sketch: "sk", Label: "resurrect"})
}

func TestSketchPauseNavigationExpiryAndDriving(t *testing.T) {
	for _, reason := range []string{"pause", "navigation", "expiry", "end", "reset"} {
		t.Run(reason, func(t *testing.T) {
			s := sketchFixture(t)
			applySketch(t, s, sketchBegin())
			applySketch(t, s, sketchStroke(1, "s1"))
			applySketch(t, s, &SketchPoints{SketchOperation: sketchOp(2), Sketch: "sk", ID: "s1", Pts: []SketchPoint{{1, 2, .3}}})
			switch reason {
			case "pause":
				applySketch(t, s, &Stop{Why: StopPause})
			case "navigation":
				applySketch(t, s, &Seg{Seg: 2, Why: SegNavigate, Tab: Tab{ID: "tab1"}, VP: Viewport{W: 100, H: 100, DPR: 1}})
			case "expiry":
				applySketch(t, s, &SketchInterrupt{SketchOperation: sketchOp(3), Sketch: "sk", Reason: "expiry"})
			case "end":
				applySketch(t, s, &Stop{Why: StopEnd})
			case "reset":
				applySketch(t, s, &Ctl{Op: CtlReset, Who: Author{ID: OperatorID}})
			}
			v := s.Sketches.Snapshot().Sketches[0]
			if !v.Strokes[0].Sealed || !v.Strokes[0].Truncated || len(v.Visible) != 1 {
				t.Fatalf("partial lost %+v", v)
			}
			if reason == "pause" {
				if v.Phase != "drawing" {
					t.Fatal(v.Phase)
				}
				refuseSketch(t, s, sketchStroke(3, "s2"))
				applySketch(t, s, &Seg{Seg: 2, Why: SegResume, Tab: Tab{ID: "tab1"}, VP: Viewport{W: 100, H: 100, DPR: 1}})
				applySketch(t, s, sketchStroke(3, "s2"))
			} else if v.Phase != "interrupted" {
				t.Fatal(v.Phase)
			}
		})
	}
	s := sketchFixture(t)
	who := Author{ID: "driver", Color: "#00ff00"}
	applySketch(t, s, &Ctl{Op: CtlGrant, Who: who, Control: Control{Driver: &who}})
	refuseSketch(t, s, sketchBegin())
	b := sketchBegin()
	b.Author = who
	b.Owner = strings.Repeat("b", 64)
	applySketch(t, s, b)
}

func TestSketchBoundsAndCountedWork(t *testing.T) {
	for _, n := range []int{64, 128} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			s := sketchFixture(t)
			applySketch(t, s, sketchBegin())
			applySketch(t, s, sketchStroke(1, "s1"))
			var seq uint64 = 2
			for i := 0; i < n; i += 64 {
				pts := make([]SketchPoint, 64)
				for j := range pts {
					pts[j] = SketchPoint{float64(i + j), 1, 1}
				}
				applySketch(t, s, &SketchPoints{SketchOperation: sketchOp(seq), Sketch: "sk", ID: "s1", Chunk: uint64(i / 64), Pts: pts})
				seq++
			}
			if s.Sketches.work.points != uint64(n) {
				t.Fatalf("visited %d points for %d", s.Sketches.work.points, n)
			}
			applySketch(t, s, &SketchStrokeEnd{SketchOperation: sketchOp(seq), Sketch: "sk", ID: "s1"})
			seq++
			before := s.Sketches.work.points
			applySketch(t, s, &SketchUndo{SketchOperation: sketchOp(seq), Sketch: "sk", ID: "s1"})
			seq++
			applySketch(t, s, &SketchRedo{SketchOperation: sketchOp(seq), Sketch: "sk", ID: "s1"})
			if s.Sketches.work.points != before || s.Sketches.work.stacks != 2 {
				t.Fatal("history walked points")
			}
		})
	}
	s := sketchFixture(t)
	applySketch(t, s, sketchBegin())
	applySketch(t, s, sketchStroke(1, "s1"))
	seq := uint64(2)
	for i := 0; i < 4096; i += 64 {
		applySketch(t, s, &SketchPoints{SketchOperation: sketchOp(seq), Sketch: "sk", ID: "s1", Chunk: uint64(i / 64), Pts: make([]SketchPoint, 64)})
		seq++
	}
	refuseSketch(t, s, &SketchPoints{SketchOperation: sketchOp(seq), Sketch: "sk", ID: "s1", Chunk: 64, Pts: []SketchPoint{{1, 1, 1}}})
}

func TestSketchSignedZeroModelAccountingAndOwnedProjection(t *testing.T) {
	var states []*State
	for _, zero := range []float64{0, math.Copysign(0, -1)} {
		s := sketchFixture(t)
		applySketch(t, s, sketchBegin())
		stroke := sketchStroke(1, "a")
		stroke.At.Box[0] = zero
		applySketch(t, s, stroke)
		points := &SketchPoints{SketchOperation: sketchOp(2), Sketch: "sk", ID: "a", Pts: []SketchPoint{{zero, zero, zero}}}
		raw, err := Encode(points)
		if err != nil {
			t.Fatal(err)
		}
		if math.Signbit(zero) && !strings.Contains(string(raw), "-0") {
			t.Fatal("wire lost original signed zero")
		}
		applySketch(t, s, points)
		if math.Signbit(zero) && !math.Signbit(points.Pts[0][0]) {
			t.Fatal("admission mutated caller")
		}
		states = append(states, s)
	}
	a, b := states[0].Sketches.Snapshot(), states[1].Sketches.Snapshot()
	if a.HistoryBytes != b.HistoryBytes {
		t.Fatalf("signed zero model bytes %d != %d", a.HistoryBytes, b.HistoryBytes)
	}
	v := b.Sketches[0]
	if math.Signbit(v.Strokes[0].Begin.At.Box[0]) || math.Signbit(v.Strokes[0].Pts[0][0]) || math.Signbit(v.Strokes[0].Pts[0][2]) {
		t.Fatal("authoritative zero is not normalized")
	}
}

func TestSketchReceiptQueriesOwnedAndConstantWork(t *testing.T) {
	s := sketchFixture(t)
	applySketch(t, s, sketchBegin())
	applySketch(t, s, sketchStroke(1, "p"))
	applySketch(t, s, &SketchPoints{SketchOperation: sketchOp(2), Sketch: "sk", ID: "p", Pts: []SketchPoint{{1, 2, .5}}})
	before := s.Sketches.work
	for i := 0; i < 100; i++ {
		receipt, ok := s.Sketches.Receipt("sk", "op-2")
		if !ok || receipt.Sequence != 2 {
			t.Fatal(receipt)
		}
		meta, ok := s.Sketches.Metadata("sk")
		if !ok || meta.OpenStroke != "p" {
			t.Fatal(meta)
		}
		tops, ok := s.Sketches.StackTops("sk")
		if !ok || tops.VisibleCount != 0 {
			t.Fatal(tops)
		}
		stroke, ok := s.Sketches.StrokeMetadata("sk", "p")
		if !ok || stroke.PointCount != 1 {
			t.Fatal(stroke)
		}
		stroke.At.Box[0] = 999
		stroke.At.Path = append(stroke.At.Path, "owned")
	}
	if s.Sketches.work != before {
		t.Fatal("read queries visited history")
	}
	meta, _ := s.Sketches.StrokeMetadata("sk", "p")
	if meta.At.Box[0] != 0 {
		t.Fatal("query aliases anchor")
	}
	change, ok := s.Sketches.Change(&SketchPoints{Sketch: "sk", ID: "p", Pts: []SketchPoint{{1, 2, .5}}}, SketchBeforeChange{})
	if !ok || len(change.Points) != 1 {
		t.Fatal(change)
	}
	change.Points[0][0] = 999
	if s.Sketches.Snapshot().Sketches[0].Strokes[0].Pts[0][0] != 1 {
		t.Fatal("change aliases paths")
	}
}

func TestSketchReceiptQueriesContextChangeAndRestoredVisibility(t *testing.T) {
	s := sketchFixture(t)
	applySketch(t, s, sketchBegin())
	applySketch(t, s, sketchStroke(1, "p"))
	applySketch(t, s, &SketchPoints{SketchOperation: sketchOp(2), Sketch: "sk", ID: "p", Pts: []SketchPoint{{1, 2, .5}}})
	before := s.Sketches.BeforeChange()
	if _, ok := s.Sketches.Change(&Title{V: "unchanged"}, before); ok {
		t.Fatal("unrelated DOM event produced sketch change")
	}
	pause := &Stop{Why: StopPause}
	applySketch(t, s, pause)
	change, ok := s.Sketches.Change(pause, before)
	if !ok || change.Stroke == nil || !change.Stroke.Truncated || !change.Stroke.Visible {
		t.Fatal("contextual partial missing")
	}
	caught := sketchFixture(t)
	caught.allowSketchSnapshot = true
	if err := caught.Apply(s.SketchSnapshot()); err != nil {
		t.Fatal(err)
	}
	stroke, ok := caught.Sketches.StrokeMetadata("sk", "p")
	if !ok || !stroke.Visible {
		t.Fatal("snapshot lost O1 visible lookup")
	}
}

func TestSketchReceiptQueriesLateLabelDoesNotTouchNewActiveStroke(t *testing.T) {
	s := sketchFixture(t)
	applySketch(t, s, sketchBegin())
	applySketch(t, s, &SketchInterrupt{SketchOperation: sketchOp(1), Sketch: "sk", Reason: SketchInterruptExpiry})
	b := sketchBegin()
	b.ID = "next"
	applySketch(t, s, b)
	stroke := sketchStroke(1, "new-path")
	stroke.Sketch = "next"
	applySketch(t, s, stroke)
	before := s.Sketches.BeforeChange()
	label := &SketchLabel{SketchOperation: sketchOp(2), Sketch: "sk", Label: "Old intent"}
	applySketch(t, s, label)
	change, ok := s.Sketches.Change(label, before)
	if !ok || change.Sketch.ID != "sk" || change.Stroke != nil {
		t.Fatalf("late label patch collided with new drawer: %+v %v", change, ok)
	}
}
