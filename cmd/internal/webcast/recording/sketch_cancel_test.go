package recording

import (
	"bytes"
	"encoding/json"
	"github.com/xianxu/pair/cmd/internal/strictjson"
	"strings"
	"testing"
)

func TestSketchCancelTruncationWireAndIdentity(t *testing.T) {
	plain := strings.Replace(sketchWireSamples()[3], `"id":"s"`, `"id":"s","truncated":true`, 1)
	r, err := Decode([]byte(plain))
	if err != nil {
		t.Fatal(err)
	}
	end := r.(*SketchStrokeEnd)
	if !end.Truncated {
		t.Fatal("truncation ignored")
	}
	encoded, err := Encode(end)
	if err != nil || !bytes.Contains(encoded, []byte(`"truncated":true`)) {
		t.Fatalf("truncation lost %s %v", encoded, err)
	}
	yes, _ := SketchDigest(end)
	end.Truncated = false
	no, _ := SketchDigest(end)
	if yes == no {
		t.Fatal("truncation missing from semantic identity")
	}
	absent, err := Decode([]byte(sketchWireSamples()[3]))
	if err != nil {
		t.Fatal(err)
	}
	want, _ := SketchDigest(absent)
	if want != no {
		t.Fatal("false changed old digest")
	}
	encoded, _ = Encode(end)
	if bytes.Contains(encoded, []byte("truncated")) {
		t.Fatal("false changed old wire")
	}
	for _, v := range []string{"null", "1", `"true"`} {
		if _, err := Decode([]byte(strings.Replace(plain, `"truncated":true`, `"truncated":`+v, 1))); err == nil {
			t.Errorf("accepted flag %s", v)
		}
	}
}
func TestSketchCancelContinueUndoRedoLabelAndSnapshot(t *testing.T) {
	s := sketchFixture(t)
	applySketch(t, s, sketchBegin())
	applySketch(t, s, sketchStroke(1, "p"))
	applySketch(t, s, &SketchPoints{SketchOperation: sketchOp(2), Sketch: "sk", ID: "p", Pts: []SketchPoint{{1, 2, .5}}})
	end := &SketchStrokeEnd{SketchOperation: sketchOp(3), Sketch: "sk", ID: "p", Truncated: true}
	applySketch(t, s, end)
	view := s.Sketches.Snapshot().Sketches[0]
	if view.Phase != "drawing" || !view.Strokes[0].Sealed || !view.Strokes[0].Truncated || len(view.Visible) != 1 {
		t.Fatalf("cancel changed sketch or lost partial %+v", view)
	}
	if err := s.Apply(end); err != nil {
		t.Fatal(err)
	}
	forged := *end
	forged.Truncated = false
	refuseSketch(t, s, &forged)
	caught := sketchFixture(t)
	caught.allowSketchSnapshot = true
	if err := caught.Apply(s.SketchSnapshot()); err != nil {
		t.Fatal(err)
	}
	applySketch(t, caught, &SketchUndo{SketchOperation: sketchOp(4), Sketch: "sk", ID: "p"})
	applySketch(t, caught, &SketchRedo{SketchOperation: sketchOp(5), Sketch: "sk", ID: "p"})
	applySketch(t, caught, sketchStroke(6, "q"))
	applySketch(t, caught, &SketchStrokeEnd{SketchOperation: sketchOp(7), Sketch: "sk", ID: "q"})
	applySketch(t, caught, &SketchLabel{SketchOperation: sketchOp(8), Sketch: "sk", Label: "Cancelled partial retained"})
	if !caught.Sketches.Snapshot().Sketches[0].Strokes[0].Truncated {
		t.Fatal("later events erased cancellation")
	}
}

// Freeze the pre-flag owned field set through the same preserving object seam
// used by the #751 codec, without assigning new effects to its old reader.
func TestSketchCancelLegacyOwnedFieldRoundTrip(t *testing.T) {
	raw := []byte(strings.Replace(sketchWireSamples()[3], `"id":"s"`, `"id":"s","truncated":true`, 1))
	known := map[string]bool{}
	for _, key := range []string{"t", "k", "operation", "sequence", "digest", "sketch", "id"} {
		known[key] = true
	}
	owned, unknown, err := strictjson.SplitObject(raw, known, nil)
	if err != nil {
		t.Fatal(err)
	}
	var old struct {
		Header
		SketchOperation
		Sketch string `json:"sketch"`
		ID     string `json:"id"`
	}
	clean, err := strictjson.EncodeObject(owned, nil, known)
	if err != nil {
		t.Fatal(err)
	}
	if strictjson.Decode(clean, &old) != nil {
		t.Fatal("old payload no longer decodes")
	}
	encoded, err := json.Marshal(old)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	json.Unmarshal(encoded, &fields)
	preserved, err := strictjson.EncodeObject(fields, unknown, known)
	if err != nil {
		t.Fatal(err)
	}
	r, err := Decode(preserved)
	if err != nil || !r.(*SketchStrokeEnd).Truncated {
		t.Fatalf("old reader erased optional fact: %s %v", preserved, err)
	}
}
