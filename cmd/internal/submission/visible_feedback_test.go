package submission

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
	"github.com/xianxu/pair/cmd/internal/webcast/sketchimage"
)

func visibleFixture() ([]PreviewSketchCandidate, []PreviewNote, FeedbackVisibility) {
	a, b := feedbackCandidate("a", 1, 10), feedbackCandidate("b", 2, 20)
	a.Item.Digest, b.Item.Digest = strings.Repeat("a", 64), strings.Repeat("b", 64)
	v := FeedbackVisibility{Revision: 1, Scene: sketchimage.PageScene{RecordingID: "recording", SketchSourceFacts: b.Finish}, MaskScope: "board", MaskRevision: 1,
		Sketches: []VisibleSketch{{ID: "b", Sequence: b.Item.Sequence, Digest: b.Item.Digest, Paths: []VisiblePath{{ID: "stroke", Locator: []int{0, 1}, Box: recording.Box{0, 0, 100, 100}, Clip: recording.Box{0, 0, 800, 600}}}}},
		Notes:    []PreviewNote{{ID: "visible", Revision: 2}}}
	return []PreviewSketchCandidate{a, b}, []PreviewNote{{ID: "hidden", Revision: 1}, {ID: "visible", Revision: 2}}, v
}

func TestSelectVisibleFeedbackUsesPaintedSetRatherThanOldestView(t *testing.T) {
	candidates, notes, view := visibleFixture()
	got, err := SelectVisibleFeedback(candidates, notes, DeliveredFeedback{}, PreviewStamp{Board: 7}, view)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Items, []PreviewItem{candidates[1].Item}) || !reflect.DeepEqual(got.Notes, notes[1:]) || got.Remaining != 1 || got.NoteRemaining != 1 || got.Page == nil || *got.Page != view.Scene {
		t.Fatalf("selection=%+v", got)
	}
	view.Sketches[0].Paths[0].Locator[0] = 99
	view.Notes[0].ID = "mutated"
	if got.Visibility.Sketches[0].Paths[0].Locator[0] != 0 || got.Visibility.Notes[0].ID != "visible" {
		t.Fatal("visibility aliases caller")
	}
}

func TestSelectVisibleFeedbackRefusesStaleOrInvalidPaintWitness(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*FeedbackVisibility)
	}{
		{"zero revision", func(v *FeedbackVisibility) { v.Revision = 0 }},
		{"unknown sketch", func(v *FeedbackVisibility) { v.Sketches[0].ID = "other" }},
		{"old sequence", func(v *FeedbackVisibility) { v.Sketches[0].Sequence-- }},
		{"wrong digest", func(v *FeedbackVisibility) { v.Sketches[0].Digest = strings.Repeat("c", 64) }},
		{"missing path", func(v *FeedbackVisibility) { v.Sketches[0].Paths = nil }},
		{"duplicate path", func(v *FeedbackVisibility) { v.Sketches[0].Paths = append(v.Sketches[0].Paths, v.Sketches[0].Paths[0]) }},
		{"missing locator", func(v *FeedbackVisibility) { v.Sketches[0].Paths[0].Locator = nil }},
		{"bad geometry", func(v *FeedbackVisibility) { v.Sketches[0].Paths[0].Box[2] = 0 }},
		{"stale note", func(v *FeedbackVisibility) { v.Notes[0].Revision++ }},
		{"duplicate note", func(v *FeedbackVisibility) { v.Notes = append(v.Notes, v.Notes[0]) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, n, v := visibleFixture()
			tc.change(&v)
			if _, err := SelectVisibleFeedback(c, n, DeliveredFeedback{}, PreviewStamp{}, v); err == nil {
				t.Fatal("invalid witness accepted")
			}
		})
	}
}

func TestSelectVisibleFeedbackCountsHiddenDeliveredAndOverflow(t *testing.T) {
	_, _, view := visibleFixture()
	view.Notes = nil
	view.Sketches = nil
	var candidates []PreviewSketchCandidate
	for i := 0; i < 11; i++ {
		c := feedbackCandidate(fmt.Sprint(i), 2, 20+i)
		c.Item.Digest = strings.Repeat("a", 64)
		candidates = append(candidates, c)
		if i < 10 {
			view.Sketches = append(view.Sketches, VisibleSketch{ID: c.Item.SketchID, Sequence: c.Item.Sequence, Digest: c.Item.Digest, Paths: []VisiblePath{{ID: "stroke", Locator: []int{0}, Box: recording.Box{0, 0, 10, 10}, Clip: recording.Box{0, 0, 800, 600}}}})
		}
	}
	delivered := DeliveredFeedback{Sketches: map[PreviewItem]struct{}{candidates[0].Item: {}}}
	got, visits, err := selectVisibleFeedback(candidates, nil, delivered, PreviewStamp{}, view)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 8 || got.Items[0] != candidates[1].Item || got.Remaining != 2 || visits > 4*(len(candidates)+len(view.Sketches)) {
		t.Fatalf("selection=%+v visits=%d", got, visits)
	}
	view.Sketches = nil
	got, err = SelectVisibleFeedback(candidates, nil, delivered, PreviewStamp{}, view)
	if err != nil || len(got.Items) != 0 || got.Remaining != 10 {
		t.Fatalf("empty visible selection=%+v err=%v", got, err)
	}
}

// The counted selection traversal must remain linear in authoritative and
// painted metadata; path geometry validation has its own fixed per-shape cap.
func TestVisibleSelectionMetadataTraversalScalesLinearly(t *testing.T) {
	counts := []int{}
	for _, size := range []int{32, 64} {
		_, _, view := visibleFixture()
		view.Sketches = nil
		view.Notes = nil
		var candidates []PreviewSketchCandidate
		var notes []PreviewNote
		for i := 0; i < size; i++ {
			c := feedbackCandidate(fmt.Sprintf("shape-%d", i), 1, i+20)
			c.Item.Digest = strings.Repeat("a", 64)
			candidates = append(candidates, c)
			notes = append(notes, PreviewNote{ID: fmt.Sprint(i)})
			if i%2 == 0 {
				view.Sketches = append(view.Sketches, VisibleSketch{ID: c.Item.SketchID, Sequence: c.Item.Sequence, Digest: c.Item.Digest, Paths: []VisiblePath{{ID: "stroke", Locator: []int{0}, Box: recording.Box{0, 0, 10, 10}, Clip: recording.Box{0, 0, 800, 600}}}})
				view.Notes = append(view.Notes, notes[i])
			}
		}
		got, visits, err := selectVisibleFeedback(candidates, notes, DeliveredFeedback{}, PreviewStamp{}, view)
		if err != nil {
			t.Fatal(err)
		}
		if len(got.Items) != 8 || got.Remaining != size-8 || len(got.Notes) != size/2 || got.NoteRemaining != size/2 {
			t.Fatalf("incorrect generated selection: %+v", got)
		}
		for i, item := range got.Items {
			if item != candidates[i*2].Item {
				t.Fatal("selection violates visible creation order")
			}
		}
		if visits != 3*size {
			t.Fatalf("visits=%d want%d", visits, 3*size)
		}
		counts = append(counts, visits)
	}
	if counts[1] != 2*counts[0] {
		t.Fatalf("nonlinear metadata work: %v", counts)
	}
}
