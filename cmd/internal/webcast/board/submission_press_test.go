package board

import (
	"fmt"
	"testing"

	core "github.com/xianxu/pair/cmd/internal/submission"
)

// pair#849: a press takes every open note and finished sketch not already
// delivered, the oldest eight sketches first, and counts the rest.
func TestPressSelectionLeavesOutDeliveredAndKeepsOldestEight(t *testing.T) {
	s := Selection{Document: 2, TabID: "tab", SketchGeneration: 7, Notes: []SelectedNote{{"a", 1}, {"b", 2}, {"c", 1}}}
	for i := 0; i < 11; i++ {
		s.Sketches = append(s.Sketches, SelectedSketch{ID: fmt.Sprintf("s%02d", i), NextSequence: 3, Digest: "d"})
	}
	delivered := core.DeliveredFeedback{RecordingID: "rec",
		Sketches: map[core.PreviewItem]struct{}{{RecordingID: "rec", SketchID: "s01", Sequence: 3, Digest: "d"}: {}},
		Notes:    map[core.PreviewNote]struct{}{{ID: "b", Revision: 2}: {}, {ID: "c", Revision: 9}: {}}}
	got := s.PressSelection(delivered)
	if len(got.Notes) != 2 || got.Notes[0].ID != "a" || got.Notes[1] != (SelectedNote{"c", 1}) {
		t.Fatalf("notes %+v", got.Notes)
	}
	if len(got.Sketches) != 8 || got.Sketches[0].ID != "s00" || got.Sketches[1].ID != "s02" || got.Sketches[7].ID != "s08" || got.SketchRemaining != 2 {
		t.Fatalf("sketches %+v remaining %d", got.Sketches, got.SketchRemaining)
	}
	if got.Document != 2 || got.TabID != "tab" || got.SketchGeneration != 7 || got.CurrentPage != nil || got.Current != nil {
		t.Fatalf("identity %+v", got)
	}
}
