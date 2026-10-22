package query

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/webcast/notes"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
	"io"
	"reflect"
	"strings"
	"testing"
)

func TestSketchesCLIArguments(t *testing.T) {
	a, err := parseArgs([]string{"recording", "sketches", "--sketch", "s", "--at", "50", "--json"})
	if err != nil {
		t.Fatalf("historical sketch query is unavailable: %v", err)
	}
	if a.verb != "sketches" || !a.atSet || a.at != 50 || !a.json {
		t.Fatalf("arguments: %+v", a)
	}
}

func sketchRecords(n int) ([]recording.Record, recording.NodeID) {
	b := recording.NewBuilder("query-sketch")
	anchor := b.El("button", []string{"id", "original", "data-action", "save"})
	parent := b.El("main", []string{"id", "settings"}, anchor)
	b.Page(recording.SegLive, "https://original.test/settings", parent)
	b.At(1).Add(&recording.Layout{Boxes: map[recording.NodeID]recording.Box{anchor.ID: {10, 20, 100, 40}}})
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("s%d", i)
		op := func(seq uint64) recording.SketchOperation {
			return recording.SketchOperation{Operation: fmt.Sprintf("%s-op%d", id, seq), Sequence: seq, Digest: strings.Repeat("a", 64)}
		}
		at := int64(10 + i*10)
		b.At(at).Add(&recording.SketchBegin{SketchOperation: op(0), ID: id, Author: recording.Author{ID: "operator", Color: "#ff0000"}, Owner: "operator", Doc: 1, Tab: "tab1", VP: recording.Viewport{W: 1280, H: 800, DPR: 1}})
		if i == 0 {
			b.Add(&recording.Attr{ID: anchor.ID, Name: "id", V: recording.Str("changed-after-begin")})
		}
		b.At(at + 1).Add(&recording.SketchStrokeBegin{SketchOperation: op(1), Sketch: id, ID: "p", At: recording.NoteAt{Node: anchor.ID, Box: &recording.Box{10, 20, 100, 40}}, Width: 3})
		b.At(at + 2).Add(&recording.SketchPoints{SketchOperation: op(2), Sketch: id, ID: "p", Pts: []recording.SketchPoint{{20, 30, 0}, {40, 35, 1}}})
		b.At(at + 3).Add(&recording.SketchStrokeEnd{SketchOperation: op(3), Sketch: id, ID: "p", Truncated: true})
		b.At(at + 4).Add(&recording.SketchLabel{SketchOperation: op(4), Sketch: id, Label: "Move control"})
	}
	b.At(int64(20 + n*10)).Add(&recording.AnnotationClear{Who: recording.Author{ID: "operator"}})
	return b.Recs, anchor.ID
}

func TestSketchesFreezeSourceAndExactHistory(t *testing.T) {
	recs, node := sketchRecords(1)
	q := New(recs, notes.Structural{})
	vs, err := q.SketchesAt(q.End(), SketchSelection{ID: "s0", NextSequence: 5, Digest: strings.Repeat("a", 64)})
	if err != nil || len(vs) != 1 {
		t.Fatalf("query: %v %+v", err, vs)
	}
	v := vs[0]
	if v.History.Phase != "complete" || v.Source.URL != "https://original.test/settings" || v.Source.T != 10 || v.Source.RecordIndex < 1 {
		t.Fatalf("wrong clocks/source: %+v", v)
	}
	if len(v.Elements.Anchors) != 1 || v.Elements.Anchors[0].Node != node || v.Elements.Anchors[0].ID != "original" || v.Elements.Anchors[0].Data["data-action"] != "save" {
		t.Fatalf("source facts consumed later same-time mutation: %+v", v.Elements)
	}
	if !v.History.Strokes[0].Truncated || v.History.Strokes[0].Pts[0][2] != 0 || v.History.Strokes[0].Pts[1][2] != 1 {
		t.Fatal("lost pressure/truncation")
	}
	if _, err = q.SketchesAt(q.End(), SketchSelection{ID: "s0", NextSequence: 5, Digest: strings.Repeat("b", 64)}); err == nil {
		t.Fatal("accepted wrong selected digest")
	}
	if _, err = q.SketchesAt(13, SketchSelection{ID: "s0", NextSequence: 5, Digest: strings.Repeat("a", 64)}); err == nil {
		t.Fatal("accepted missing prefix")
	}
	if _, err = q.SketchesAt(q.End(), SketchSelection{ID: "missing", NextSequence: 1, Digest: strings.Repeat("a", 64)}); err == nil {
		t.Fatal("accepted missing sketch")
	}
	latest, err := q.SketchesAt(q.End())
	if err != nil || latest[0].History.Phase != "cleared" {
		t.Fatalf("latest clear: %v %+v", err, latest)
	}
}

func TestSketchesMatchReducerAtEveryPrefix(t *testing.T) {
	recs, _ := sketchRecords(2)
	for i, r := range recs {
		prefix := recs[:i+1]
		st, err := recording.Reduce(prefix, r.Time())
		if err != nil {
			t.Fatalf("fixture %d: %v", i, err)
		}
		got, err := New(prefix, notes.Structural{}).SketchesAt(r.Time())
		if err != nil {
			t.Fatalf("prefix%d: %v", i, err)
		}
		want := st.Sketches.Snapshot().Sketches
		if len(got) != len(want) {
			t.Fatalf("prefix%d count", i)
		}
		for j := range got {
			if !reflect.DeepEqual(got[j].History, want[j]) {
				t.Fatalf("prefix%d projection differs", i)
			}
		}
	}
}
func TestSketchQuerySinglePassScaling(t *testing.T) {
	var work []sketchQueryStats
	for _, n := range []int{8, 16} {
		recs, _ := sketchRecords(n)
		q := New(recs, notes.Structural{})
		views, stats, err := q.sketchesAt(q.End(), nil)
		if err != nil || len(views) != n {
			t.Fatalf("n%d: %v", n, err)
		}
		if stats.Records != len(recs) {
			t.Fatal("recording scanned repeatedly")
		}
		work = append(work, stats)
	}
	if work[1].SketchEvents != work[0].SketchEvents*2 {
		t.Fatalf("nonlinear history visits: %+v", work)
	}
}
func TestDeclaredElementFactsBoundsAndAncestors(t *testing.T) {
	b := recording.NewBuilder("facts")
	leaf := b.El("button", []string{"data-action", "save", "data-long", strings.Repeat("x", 1025)})
	root := b.El("main", []string{"id", "root"}, leaf)
	b.Page(recording.SegLive, "https://x/", root)
	st, err := recording.Reduce(b.Recs, 0)
	if err != nil {
		t.Fatal(err)
	}
	f := DeclaredElementFactsAt(st, []recording.NodeID{leaf.ID}, nil)
	if len(f.Anchors) != 1 || f.Anchors[0].Data["data-action"] != "save" || !f.Truncated {
		t.Fatalf("bounded attributes: %+v", f)
	}
	if len(f.Ancestors) != 1 || f.Ancestors[0].ID != "root" {
		t.Fatalf("ancestor: %+v", f)
	}
	st.Node(leaf.ID).A["data-action"] = "changed"
	if f.Anchors[0].Data["data-action"] != "save" {
		t.Fatal("fact aliases DOM")
	}
	missing := DeclaredElementFactsAt(st, []recording.NodeID{999}, nil)
	if missing.Anchors[0].Unavailable == "" {
		t.Fatal("missing source anchor not explicit")
	}
}

func TestSketchesCLIJSONAndText(t *testing.T) {
	recs, _ := sketchRecords(1)
	q := New(recs, notes.Structural{})
	for _, jsonOutput := range []bool{false, true} {
		var output bytes.Buffer
		err := answer(q, args{verb: "sketches", sketch: "s0", at: 14, atSet: true, json: jsonOutput}, &output)
		if err != nil {
			t.Fatal(err)
		}
		if jsonOutput {
			var got []SketchView
			if err = json.Unmarshal(output.Bytes(), &got); err != nil || len(got) != 1 || got[0].History.Label != "Move control" {
				t.Fatalf("JSON: %v %s", err, output.String())
			}
		} else {
			for _, part := range []string{"source t=10", "history t=14", "https://original.test/settings", "anchor @", "original"} {
				if !strings.Contains(output.String(), part) {
					t.Fatalf("missing %q: %s", part, output.String())
				}
			}
		}
	}
	if err := answer(q, args{verb: "sketches", sketch: "missing", at: 14, atSet: true}, io.Discard); err == nil {
		t.Fatal("missing ID accepted")
	}
}
func TestSketchesUndoRedoInterruptedAndUnfinished(t *testing.T) {
	recs, node := sketchRecords(1)
	recs = recs[:len(recs)-2]
	op := func(n uint64) recording.SketchOperation {
		return recording.SketchOperation{Operation: fmt.Sprintf("later%d", n), Sequence: n, Digest: strings.Repeat("a", 64)}
	}
	recs = append(recs, &recording.SketchUndo{Header: recording.Header{T: 14}, SketchOperation: op(4), Sketch: "s0", ID: "p"}, &recording.SketchRedo{Header: recording.Header{T: 15}, SketchOperation: op(5), Sketch: "s0", ID: "p"}, &recording.Stop{Header: recording.Header{T: 16}, Why: recording.StopPause}, &recording.Seg{Header: recording.Header{T: 17}, Seg: 2, Why: recording.SegLive, Tab: recording.Tab{ID: "tab2", URL: "https://later.test/"}, VP: recording.Viewport{W: 100, H: 100, DPR: 1}}, &recording.SketchLabel{Header: recording.Header{T: 18}, SketchOperation: op(6), Sketch: "s0", Label: "After navigation"})
	for _, at := range []int64{12, 14, 15, 16, 17, 18} {
		st, err := recording.Reduce(recs, at)
		if err != nil {
			t.Fatalf("fixture: %v", err)
		}
		got, err := New(recs, notes.Structural{}).SketchesAt(at)
		if err != nil || len(got) != 1 || !reflect.DeepEqual(got[0].History, st.Sketches.Snapshot().Sketches[0]) {
			t.Fatalf("at%d: %v %+v", at, err, got)
		}
		if got[0].Source.URL != "https://original.test/settings" || got[0].Elements.Anchors[0].Node != node {
			t.Fatal("historical source changed")
		}
	}
}
func TestDeclaredFactsNearestIntersectionsAndLimits(t *testing.T) {
	b := recording.NewBuilder("facts-many")
	kids := []recording.Node{}
	for i := 0; i < 80; i++ {
		kids = append(kids, b.El("button", []string{"id", fmt.Sprintf("el%d", i)}))
	}
	b.Page(recording.SegLive, "https://x/", kids...)
	st, err := recording.Reduce(b.Recs, 0)
	if err != nil {
		t.Fatal(err)
	}
	for i, n := range kids {
		st.Boxes[n.ID] = recording.Box{float64(i), 0, 100, 100}
	}
	box := recording.Box{0, 0, 100, 100}
	f := DeclaredElementFactsAt(st, []recording.NodeID{kids[79].ID}, &box)
	if len(f.Anchors) != 1 || f.Anchors[0].ID != "el79" || len(f.Anchors)+len(f.Ancestors)+len(f.Intersections) > 64 || !f.Truncated {
		t.Fatalf("count/anchor priority: %+v", f)
	}
	if len(f.Intersections) == 0 || f.Intersections[0].ID != "el0" {
		t.Fatalf("nearest ordering: %+v", f.Intersections)
	}
	if f.Bytes > ElementFactsByteLimit {
		t.Fatal("byte bound exceeded")
	}
}

func TestSurfaceSupportsHistoricalSketches(t *testing.T) {
	type sketchSurface interface {
		SketchesAt(int64, ...SketchSelection) ([]SketchView, error)
	}
	for _, surface := range []Surface{New(nil, notes.Structural{}), FakeSurface{Model: notes.Structural{}}} {
		if _, ok := surface.(sketchSurface); !ok {
			t.Fatalf("%T lacks shared sketch surface", surface)
		}
	}
}
