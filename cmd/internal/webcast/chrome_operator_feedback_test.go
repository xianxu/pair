package webcast

import (
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func TestInactiveOperatorViewPreservesConfirmedMask(t *testing.T) {
	mask := board.LiveFeedbackMask{Scope: "cast", Revision: 9, Sketches: []board.FeedbackSketchVersion{{ID: "s", NextSequence: 5, Digest: strings.Repeat("a", 64)}}, Notes: []board.FeedbackNoteVersion{{ID: "n", Revision: 2}}}
	active := board.OperatorView{Revision: 10, Seg: 1, Doc: 1, Notes: []*recording.Note{{ID: "n"}}, Strokes: []board.OperatorStroke{{Seq: 1}}, FeedbackMask: mask}
	inactive := inactiveOperatorView(active)
	if inactive.Revision != active.Revision || inactive.Doc != 0 || inactive.Seg != 0 || len(inactive.Notes) != 0 || len(inactive.Strokes) != 0 {
		t.Fatalf("inactive view=%+v", inactive)
	}
	if !reflect.DeepEqual(inactive.FeedbackMask, mask) {
		t.Fatalf("inactive producer lost confirmed mask: %+v", inactive.FeedbackMask)
	}
}
