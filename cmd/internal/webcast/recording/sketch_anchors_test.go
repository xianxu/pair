package recording

import "testing"

func TestSketchAnchorIndexAcceptedOnlyOwnedAndRestore(t *testing.T) {
	state := sketchFixture(t)
	if state.Sketches.AnchorRevision() != 0 {
		t.Fatal("empty index revision")
	}
	applySketch(t, state, sketchBegin())
	applySketch(t, state, sketchStroke(1, "a"))
	revision := state.Sketches.AnchorRevision()
	if revision == 0 || !state.Sketches.HasAnchor(1, 99) || state.Sketches.HasAnchor(2, 99) {
		t.Fatal("anchor missing or cross-document")
	}
	applySketch(t, state, &SketchPoints{SketchOperation: sketchOp(2), Sketch: "sk", ID: "a", Pts: []SketchPoint{{1, 2, 1}}})
	if state.Sketches.AnchorRevision() != revision {
		t.Fatal("point append rebuilt anchor index")
	}
	applySketch(t, state, &SketchStrokeEnd{SketchOperation: sketchOp(3), Sketch: "sk", ID: "a"})
	applySketch(t, state, sketchStroke(4, "b"))
	if state.Sketches.AnchorRevision() != revision {
		t.Fatal("duplicate anchor changed set revision")
	}
	applySketch(t, state, &SketchStrokeEnd{SketchOperation: sketchOp(5), Sketch: "sk", ID: "b"})
	next := sketchStroke(6, "c")
	next.At.Node = 100
	applySketch(t, state, next)
	bad := sketchStroke(9, "bad")
	bad.At.Node = 777
	refuseSketch(t, state, bad)
	if state.Sketches.HasAnchor(1, 777) || state.Sketches.AnchorRevision() != revision+1 {
		t.Fatal("refused operation changed index")
	}
	nodes := state.Sketches.AnchorNodes(1)
	if len(nodes) != 2 || nodes[0] != 99 || nodes[1] != 100 {
		t.Fatalf("nodes %v", nodes)
	}
	nodes[0] = 999
	if state.Sketches.HasAnchor(1, 999) || state.Sketches.AnchorNodes(1)[0] != 99 {
		t.Fatal("query aliases index")
	}
	restored, err := restoreSketchState(state.Sketches.Snapshot())
	if err != nil {
		t.Fatal(err)
	}
	if !restored.HasAnchor(1, 99) || !restored.HasAnchor(1, 100) || restored.HasAnchor(2, 99) {
		t.Fatal("restore lost source identity")
	}
	applySketch(t, state, &AnnotationClear{Who: Author{ID: OperatorID}})
	if state.Sketches.HasAnchor(1, 99) || len(state.Sketches.AnchorNodes(1)) != 0 || state.Sketches.AnchorRevision() != revision+2 {
		t.Fatal("clear retained original proof")
	}
	restored, err = restoreSketchState(state.Sketches.Snapshot())
	if err != nil {
		t.Fatal(err)
	}
	if restored.HasAnchor(1, 99) {
		t.Fatal("restore revived cleared proof")
	}
}
