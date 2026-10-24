package recording

import (
	"reflect"
	"testing"
)

func requestedAnchorFixture(n int) (*State, *PageLoad) {
	load := &PageLoad{Target: "A", Loader: "LA", Root: 1}
	state := NewState()
	state.pageLoads = map[int]*PageLoad{1: load, 2: {Target: "B", Loader: "LB", Root: 2}, 3: load}
	// These are the existing accepted source-document index buckets. A second
	// activation shares the load; unrelated-load anchors must confer no proof.
	for i := 0; i < n; i++ {
		state.Sketches.addAnchor(1, NodeID(1000+i))
		state.Sketches.addAnchor(2, NodeID(2000+i))
		state.Sketches.addAnchor(3, NodeID(3000+i))
	}
	state.Sketches.addAnchor(1, 10)
	state.Sketches.addAnchor(3, 11)
	state.Sketches.addAnchor(2, 12)
	return state, load
}
func TestRequestedSketchAnchorsOnPageLoadBoundedOwnedAndUnknown(t *testing.T) {
	wanted := map[NodeID]bool{10: true, 11: true, 12: true, 99: true, 1000: false}
	for _, n := range []int{32, 64} {
		state, load := requestedAnchorFixture(n)
		got := state.SketchAnchorsOnPageLoad(load, wanted)
		if !reflect.DeepEqual(got, map[NodeID]bool{10: true, 11: true}) {
			t.Fatalf("history %d: current requested proof %v", n, got)
		}
		delete(got, 10)
		got[99] = true
		if again := state.SketchAnchorsOnPageLoad(load, wanted); !again[10] || again[99] {
			t.Fatal("result aliases accepted index or requested input")
		}
		if got := state.SketchAnchorsOnPageLoad(nil, wanted); len(got) != 0 {
			t.Fatal("unknown load matched historical anchors")
		}
		if got := state.SketchAnchorsOnPageLoad(load, nil); len(got) != 0 {
			t.Fatal("empty batch scanned into unrequested result")
		}
	}
}
func TestRequestedSketchAnchorAllocationsDoNotGrowWithHistory(t *testing.T) {
	wanted := map[NodeID]bool{10: true, 11: true, 12: true, 99: true}
	var previous float64
	for _, n := range []int{32, 64} {
		state, load := requestedAnchorFixture(n)
		var got map[NodeID]bool
		allocs := testing.AllocsPerRun(100, func() { got = state.SketchAnchorsOnPageLoad(load, wanted) })
		if len(got) != 2 || len(got) > len(wanted) {
			t.Fatal("result grew beyond requested matching anchors")
		}
		if n == 64 && allocs != previous {
			t.Fatalf("fixed batch allocation grows with anchor history: N=%v 2N=%v", previous, allocs)
		}
		t.Logf("history=%d per matching/unrelated bucket, requested=%d, result=%d, allocations=%g", n, len(wanted), len(got), allocs)
		previous = allocs
	}
}
