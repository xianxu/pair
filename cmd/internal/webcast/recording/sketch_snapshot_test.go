package recording

import (
	"reflect"
	"testing"
)

func TestSketchSnapshotRoundTripAndAdmission(t *testing.T) {
	s := sketchFixture(t)
	applySketch(t, s, sketchBegin())
	applySketch(t, s, sketchStroke(1, "a"))
	applySketch(t, s, &SketchPoints{SketchOperation: sketchOp(2), Sketch: "sk", ID: "a", Pts: []SketchPoint{{1, 2, .5}}})
	frame := s.SketchSnapshot()
	raw, err := Encode(frame)
	if err != nil {
		t.Fatal(err)
	}
	// Snapshot wire never gains a disk registry entry or raw state authority.
	disk, err := Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := disk.(*Unknown); !ok {
		t.Fatal("bootstrap became persisted record kind")
	}
	if s.Apply(frame) == nil {
		t.Fatal("ordinary state admitted bootstrap")
	}
	caught := sketchFixture(t)
	caught.allowSketchSnapshot = true
	if err := caught.Apply(frame); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(s.Sketches.Snapshot(), caught.Sketches.Snapshot()) {
		t.Fatal("snapshot differs")
	}
	next := &SketchPoints{SketchOperation: sketchOp(3), Sketch: "sk", ID: "a", Chunk: 1, Pts: []SketchPoint{{3, 4, 1}}}
	applySketch(t, caught, next)
	if len(s.Sketches.Snapshot().Sketches[0].Strokes[0].Pts) != 1 {
		t.Fatal("snapshot restore aliased source")
	}
	// Stored receipts still suppress a retransmission of an accepted operation.
	retry := &SketchPoints{SketchOperation: sketchOp(2), Sketch: "sk", ID: "a", Pts: []SketchPoint{{1, 2, .5}}}
	before := sketchJSON(caught)
	if err := caught.Apply(retry); err != nil {
		t.Fatal(err)
	}
	if before != sketchJSON(caught) {
		t.Fatal("receipt lost across bootstrap")
	}
}
func TestSketchSnapshotRejectsMalformedAtomically(t *testing.T) {
	s := sketchFixture(t)
	applySketch(t, s, sketchBegin())
	applySketch(t, s, sketchStroke(1, "a"))
	applySketch(t, s, &SketchPoints{SketchOperation: sketchOp(2), Sketch: "sk", ID: "a", Pts: []SketchPoint{{1, 2, 1}}})
	mutations := []func(*SketchSnapshot){
		func(f *SketchSnapshot) { f.Sketches.Active = "missing" },
		func(f *SketchSnapshot) { f.Sketches.Sketches[0].Strokes[0].Pts[0][0] = 9 },
		func(f *SketchSnapshot) { f.Sketches.HistoryBytes = SketchHistoryLimit + 1 },
		func(f *SketchSnapshot) { f.Sketches.Sketches[0].NextSequence++ },
		func(f *SketchSnapshot) { f.Sketches.Sketches[0].Strokes[0].Pts[0][2] = 2 },
		func(f *SketchSnapshot) { f.Sketches.Sketches[0].Visible = []string{"a", "a"} },
		func(f *SketchSnapshot) { f.Sketches.Sketches[0].Receipts[1].Semantic = "forged" },
		func(f *SketchSnapshot) { f.Control = Control{Pending: []Author{{ID: "x"}, {ID: "x"}}} },
	}
	for i, mutate := range mutations {
		f := s.SketchSnapshot()
		mutate(f)
		target := sketchFixture(t)
		target.allowSketchSnapshot = true
		before := sketchJSON(target)
		oldT := target.T
		if err := target.Apply(f); err == nil {
			t.Fatalf("mutation %d accepted", i)
		}
		if before != sketchJSON(target) || oldT != target.T || target.Control.Driver != nil {
			t.Fatal("malformed snapshot mutated target")
		}
	}
}
func TestSketchCatchUpSnapshotsAtEveryPrefixAndCompaction(t *testing.T) {
	corpus := makeSketchCorpus(t)
	for _, c := range corpus.Cases {
		var recs []Record
		s := NewState()
		for _, raw := range c.Records {
			r, err := Decode(raw)
			if err != nil {
				t.Fatal(err)
			}
			if s.Apply(r) != nil {
				continue
			}
			recs = append(recs, r)
			if err := CheckCatchUp(recs, 1<<62); err != nil {
				t.Fatalf("%s prefix %d: %v", c.Name, len(recs), err)
			}
			joined, err := ReduceCatchUp(CatchUp(recs), 1<<62)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(s.Sketches.Snapshot(), joined.Sketches.Snapshot()) {
				t.Fatal("sketch projection differs")
			}
			compacted := CatchUp(CatchUp(recs))
			twice, err := ReduceCatchUp(compacted, 1<<62)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(joined.Sketches.Snapshot(), twice.Sketches.Snapshot()) {
				t.Fatal("recompaction differs")
			}
		}
	}
	var recs []Record
	b := NewBuilder("bounded")
	b.Page(SegLive, "http://x/")
	recs = append(recs, b.Recs...)
	begin := sketchBegin()
	begin.SetTime(1)
	recs = append(recs, begin)
	who := Author{ID: "viewer"}
	for i := 0; i < 1000; i++ {
		recs = append(recs, &Ctl{Header: Header{T: int64(i*2 + 2)}, Op: CtlRequest, Who: who, Control: Control{Pending: []Author{who}}}, &Ctl{Header: Header{T: int64(i*2 + 3)}, Op: CtlDeny, Who: who})
		recs = CatchUp(recs)
	}
	count := 0
	for _, r := range recs {
		if r.Kind() == "ctl" || r.Kind() == "sketch-snapshot" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("control chatter retained %d records", count)
	}
	encoded, _ := Encode(recs[len(recs)-1])
	if len(encoded) > 4096 || len(recs) > 4 {
		t.Fatal("unexpected fixture growth")
	}
}

func TestSketchHubCompactionReconnect(t *testing.T) {
	for _, c := range makeSketchCorpus(t).Cases {
		var recs []Record
		s := NewState()
		for _, raw := range c.Records {
			r, err := Decode(raw)
			if err != nil {
				t.Fatal(err)
			}
			if s.Apply(r) == nil {
				recs = append(recs, r)
			}
		}
		for cut := 1; cut <= len(recs); cut++ {
			hub := NewHub(len(recs) + 4)
			for _, r := range recs[:cut] {
				if err := hub.Put(r); err != nil {
					t.Fatal(err)
				}
			}
			stream, cancel := hub.Subscribe()
			for _, r := range recs[cut:] {
				if err := hub.Put(r); err != nil {
					t.Fatal(err)
				}
			}
			if err := hub.Close(); err != nil {
				t.Fatal(err)
			}
			var joined []Record
			for r := range stream {
				joined = append(joined, r)
			}
			cancel()
			state, err := ReduceCatchUp(joined, 1<<62)
			if err != nil {
				t.Fatalf("%s cut %d: %v", c.Name, cut, err)
			}
			if !reflect.DeepEqual(state.Sketches.Snapshot(), s.Sketches.Snapshot()) || !state.Control.Equal(s.Control) {
				t.Fatalf("%s cut %d differs", c.Name, cut)
			}
		}
	}
}
