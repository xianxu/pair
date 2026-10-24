package recording

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"testing"
)

func TestPageLoadIdentityAndStringRoot(t *testing.T) {
	a := &PageLoad{Target: "T", Loader: "L", Root: math.MaxInt64}
	raw, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"loader":"L","root":"9223372036854775807","target":"T"}` {
		t.Fatalf("wire %s", raw)
	}
	var b PageLoad
	if err = json.Unmarshal(raw, &b); err != nil {
		t.Fatal(err)
	}
	if !SamePageLoad(a, &b) || SamePageLoad(nil, nil) || SamePageLoad(a, nil) {
		t.Fatal("identity equality")
	}
	for _, other := range []*PageLoad{{Target: "t", Loader: "L", Root: a.Root}, {Target: "T", Loader: "l", Root: a.Root}, {Target: "T", Loader: "L", Root: 1}, {Target: "T", Loader: "L", Root: 0}} {
		if SamePageLoad(a, other) {
			t.Fatalf("matched %#v", other)
		}
	}
	// Opaque identifiers do not have sketch's 128-byte syntax or size policy.
	for i := 0; i < 300; i++ {
		b.Target += "x"
	}
	if b.Validate() != nil {
		t.Fatal("invented identifier limit")
	}
}

func TestPageLoadIndexAcceptedCreationMembershipIsOwned(t *testing.T) {
	b := NewBuilder("loads")
	node := b.El("button", nil)
	b.Page(SegLive, "https://x", node)
	load := &PageLoad{Target: "tab1", Loader: "A", Root: 1, unknown: map[string]json.RawMessage{"future": []byte(`{ "n": 1e9999 }`)}}
	b.Recs[1].(*Seg).PageLoad = load
	b.Add(&Note{ID: "note", Author: Author{ID: "operator"}, At: NoteAt{Node: node.ID}})
	s, err := Reduce(b.Recs, math.MaxInt64)
	if err != nil {
		t.Fatal(err)
	}
	if !SamePageLoad(s.CurrentPageLoad(), load) || !SamePageLoad(s.PageLoadAt(s.Notes[0].Seg), load) {
		t.Fatal("missing creation membership")
	}
	load.Loader = "changed"
	load.unknown["future"][0] = '['
	owned := s.PageLoadAt(1)
	if owned.Loader != "A" || string(owned.unknown["future"]) != `{ "n": 1e9999 }` {
		t.Fatal("accepted index aliases source")
	}
	owned.Target = "changed"
	owned.unknown["future"][0] = '['
	if got := s.PageLoadAt(1); got.Target != "tab1" || string(got.unknown["future"]) != `{ "n": 1e9999 }` {
		t.Fatal("lookup aliases index")
	}
	before := s.CurrentPageLoad()
	if err = s.Apply(&Seg{Header: Header{T: s.T}, Seg: 2, Why: SegTab, Tab: Tab{ID: "other"}, VP: Viewport{W: 1, H: 1}, PageLoad: &PageLoad{Target: "mismatch", Loader: "L", Root: 1}}); err == nil {
		t.Fatal("accepted target mismatch")
	}
	if !SamePageLoad(before, s.CurrentPageLoad()) || s.PageLoadAt(2) != nil {
		t.Fatal("refusal changed index")
	}
	if err = s.Apply(&Seg{Header: Header{T: s.T}, Seg: 2, Why: SegTab, Tab: Tab{ID: "other"}, VP: Viewport{W: 1, H: 1}}); err != nil {
		t.Fatal(err)
	}
	if s.CurrentPageLoad() != nil || s.PageLoadAt(1) == nil {
		t.Fatal("legacy absence borrowed or erased load")
	}
}

func TestPageLoadCatchUpEveryPrefixAndRepeatedCompaction(t *testing.T) {
	b, _, _ := twoDocs()
	for _, r := range b.Recs {
		if seg, ok := r.(*Seg); ok {
			loader := "A"
			if seg.Seg == 3 {
				loader = "B"
			}
			seg.PageLoad = &PageLoad{Target: seg.Tab.ID, Loader: loader, Root: int64(len(loader)) + int64(seg.Seg/3), unknown: map[string]json.RawMessage{"future": []byte(`{ "n": 1e9999 }`)}}
		}
	}
	// Add a tab return with a new activation segment and its old load identity.
	b.Add(&Seg{Seg: b.NextSeg(), Why: SegTab, Tab: Tab{ID: "tab1", URL: "http://x/a"}, VP: Viewport{W: 800, H: 600}, PageLoad: &PageLoad{Target: "tab1", Loader: "A", Root: 1}})
	b.Snapshot("http://x/a", b.El("html", nil))
	for cut := 1; cut <= len(b.Recs); cut++ {
		all, err := Reduce(b.Recs[:cut], math.MaxInt64)
		if err != nil {
			t.Fatal(err)
		}
		recs := b.Recs[:cut]
		for repeat := 0; repeat < 3; repeat++ {
			recs = CatchUp(recs)
			for i, record := range recs {
				if _, ok := record.(*Seg); !ok {
					continue
				}
				raw, err := Encode(record)
				if err != nil {
					t.Fatal(err)
				}
				recs[i], err = Decode(raw)
				if err != nil {
					t.Fatal(err)
				}
			}
			joined, err := ReduceCatchUp(recs, math.MaxInt64)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(all.pageLoads, joined.pageLoads) || !reflect.DeepEqual(all.CurrentPageLoad(), joined.CurrentPageLoad()) {
				t.Fatalf("cut %d repeat %d load facts differ", cut, repeat)
			}
			if err = CheckCatchUp(recs, math.MaxInt64); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestPageLoadSketchReceiptsUnchangedAcrossCompaction(t *testing.T) {
	corpus := makeSketchCorpus(t)
	for _, c := range corpus.Cases {
		var recs []Record
		s := NewState()
		for _, raw := range c.Records {
			r, err := Decode(raw)
			if err != nil {
				t.Fatal(err)
			}
			if seg, ok := r.(*Seg); ok {
				seg.PageLoad = &PageLoad{Target: seg.Tab.ID, Loader: "L", Root: math.MaxInt64}
			}
			if s.Apply(r) != nil {
				continue
			}
			recs = append(recs, r)
			compact := CatchUp(CatchUp(recs))
			joined, err := ReduceCatchUp(compact, math.MaxInt64)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(s.pageLoads, joined.pageLoads) || !reflect.DeepEqual(s.Sketches.Snapshot(), joined.Sketches.Snapshot()) {
				t.Fatalf("%s prefix %d differs", c.Name, len(recs))
			}
			for _, view := range joined.Sketches.Snapshot().Sketches {
				if !SamePageLoad(joined.PageLoadAt(view.Begin.Doc), s.PageLoadAt(view.Begin.Doc)) {
					t.Fatal("inactive sketch lost creation membership")
				}
			}
		}
	}
}

func TestPageLoadMembershipUsesAcceptedIndexWithoutCopying(t *testing.T) {
	s := NewState()
	s.pageLoads = map[int]*PageLoad{1: {Target: "T", Loader: "L", Root: 1, unknown: map[string]json.RawMessage{"future": []byte(`{ "n": 1e9999 }`)}}}
	membership, ok := any(s).(interface{ OnPageLoad(int, *PageLoad) bool })
	if !ok {
		t.Fatal("missing direct membership predicate")
	}
	known := &PageLoad{Target: "T", Loader: "L", Root: 1}
	if !membership.OnPageLoad(1, known) || membership.OnPageLoad(2, known) || membership.OnPageLoad(1, nil) || membership.OnPageLoad(2, nil) || membership.OnPageLoad(1, &PageLoad{Target: "T", Loader: "other", Root: 1}) {
		t.Fatal("incorrect page membership")
	}
	if allocs := testing.AllocsPerRun(100, func() {
		if !membership.OnPageLoad(1, known) {
			panic("membership changed")
		}
	}); allocs != 0 {
		t.Fatalf("membership allocated %v times", allocs)
	}
}

func TestPageLoadSketchProjectionVisitsEachCreationOnce(t *testing.T) {
	for _, n := range []int{32, 64} {
		s := NewState()
		build := NewBuilder("projection")
		for i := 0; i < n; i++ {
			why := SegNavigate
			if i == 0 {
				why = SegLive
			}
			start := len(build.Recs)
			build.Page(why, "http://same/")
			for _, r := range build.Recs[start:] {
				if seg, ok := r.(*Seg); ok {
					seg.PageLoad = &PageLoad{Target: seg.Tab.ID, Loader: fmt.Sprint("L", i%2), Root: 1}
				}
			}
		}
		// Populate through accepted records, interleaving one finished sketch in
		// every activation. The membership predicate sees only creation metadata.
		for _, r := range build.Recs {
			r.SetTime(s.T + 1)
			if err := s.Apply(r); err != nil {
				t.Fatal(err)
			}
			if _, ok := r.(*Doc); !ok {
				continue
			}
			begin := sketchBegin()
			begin.ID = fmt.Sprint("s", s.Seg)
			begin.Doc = s.DocSeg
			begin.Tab = s.Tab.ID
			applySketch(t, s, begin)
			applySketch(t, s, &SketchLabel{SketchOperation: sketchOp(1), Sketch: begin.ID, Label: "saved"})
		}
		load := s.PageLoadAt(1)
		visits := 0
		out := s.Sketches.snapshotForDocuments(func(doc int) bool { visits++; return s.OnPageLoad(doc, load) })
		if visits != n || len(out.Sketches) != n/2 {
			t.Fatalf("N=%d visits=%d selected=%d", n, visits, len(out.Sketches))
		}
		out.Sketches[0].Begin.ID = "mutated"
		if s.Sketches.Snapshot().Sketches[0].Begin.ID == "mutated" {
			t.Fatal("projection aliases history")
		}
		if got := s.SketchesOnPageLoad(nil); len(got.Sketches) != 0 {
			t.Fatal("unknown matched history")
		}
	}
}
