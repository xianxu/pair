package submission

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast/query"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func feedbackCandidate(id string, doc, index int) PreviewSketchCandidate {
	facts := query.SketchSourceFacts{T: int64(index), RecordIndex: index, Doc: doc, Tab: "tab", URL: fmt.Sprintf("https://example.test/%d", doc), Title: "Page", VP: recording.Viewport{W: 800, H: 600, DPR: 1}}
	source := facts
	source.RecordIndex--
	source.T--
	return PreviewSketchCandidate{Item: PreviewItem{RecordingID: "recording", SketchID: id, Sequence: 5, Digest: "accepted"}, Source: source, Finish: facts}
}

func TestFeedbackPreviewOldestCompatibleViewAndLatestFinish(t *testing.T) {
	a := feedbackCandidate("a", 1, 10)
	sent := feedbackCandidate("sent", 2, 11)
	b := feedbackCandidate("b", 2, 12)
	c := feedbackCandidate("c", 1, 20)
	c.Finish.Title = "Latest title"
	notes := []PreviewNote{{ID: "n", Revision: 2}}
	got, err := SelectFeedbackPreview([]PreviewSketchCandidate{sent, a, b, c}, notes, map[PreviewItem]struct{}{sent.Item: {}}, PreviewStamp{Board: 7, Delivered: 9})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Items, []PreviewItem{a.Item, c.Item}) || got.Remaining != 1 || got.Page == nil || got.Page.RecordIndex != 20 || got.Page.Title != "Latest title" || !reflect.DeepEqual(got.Notes, notes) {
		t.Fatalf("selection=%+v page=%+v", got, got.Page)
	}
	notes[0].ID = "mutated"
	if got.Notes[0].ID != "n" {
		t.Fatal("borrowed notes")
	}
}

func TestFeedbackPreviewCapIsSketchesOnlyAndCountsEveryDeferredView(t *testing.T) {
	var all []PreviewSketchCandidate
	for i := 0; i < 12; i++ {
		all = append(all, feedbackCandidate(fmt.Sprint(i), 1, 10+i))
	}
	all = append(all, feedbackCandidate("other", 2, 30))
	notes := make([]PreviewNote, 20)
	for i := range notes {
		notes[i] = PreviewNote{ID: fmt.Sprint(i), Revision: uint64(i)}
	}
	got, err := SelectFeedbackPreview(all, notes, nil, PreviewStamp{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 8 || len(got.Notes) != 20 || got.Remaining != 5 || got.Page.RecordIndex != 17 {
		t.Fatalf("selection=%+v", got)
	}
	for i := range got.Items {
		if got.Items[i] != all[i].Item {
			t.Fatal("creation order changed")
		}
	}
}

func TestFeedbackPreviewUsesExactVersionsAndViewGeometry(t *testing.T) {
	a := feedbackCandidate("a", 1, 10)
	b := feedbackCandidate("b", 1, 20)
	b.Finish.Scroll[1] = 100
	c := feedbackCandidate("c", 1, 30)
	c.Finish.VP.W++
	d := feedbackCandidate("d", 1, 40)
	old := a.Item
	old.Sequence--
	got, err := SelectFeedbackPreview([]PreviewSketchCandidate{a, b, c, d}, nil, map[PreviewItem]struct{}{old: {}}, PreviewStamp{})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Items, []PreviewItem{a.Item, d.Item}) || got.Remaining != 2 {
		t.Fatalf("selection=%+v", got)
	}
}

func TestFeedbackPreviewNoteOnlyAndMissingContext(t *testing.T) {
	got, err := SelectFeedbackPreview(nil, []PreviewNote{{ID: "n", Revision: 1}}, nil, PreviewStamp{})
	if err != nil || got.Page != nil || len(got.Items) != 0 || len(got.Notes) != 1 {
		t.Fatalf("note-only=%+v err=%v", got, err)
	}
	bad := feedbackCandidate("bad", 1, 10)
	bad.Finish.Doc = 0
	if _, err = SelectFeedbackPreview([]PreviewSketchCandidate{bad}, nil, nil, PreviewStamp{}); err == nil {
		t.Fatal("missing finish silently substituted")
	}
}

func TestFeedbackPreviewPublicationOwnsAllWitnesses(t *testing.T) {
	a := feedbackCandidate("a", 1, 10)
	selected, err := SelectFeedbackPreview([]PreviewSketchCandidate{a}, []PreviewNote{{ID: "n", Revision: 1}}, nil, PreviewStamp{Board: 1})
	if err != nil {
		t.Fatal(err)
	}
	state, _ := NextPreview(PreviewState{}, PreviewHostChanged{Host: PreviewHost{ID: "host", Generation: 1}})
	state, _ = NextPreview(state, PreviewCandidatesChanged{Selection: selected})
	state, effects := NextPreview(state, PreviewRefresh{})
	pub := effects[0].(PreviewPublish).Publication
	selected.Notes[0].Revision = 99
	selected.Candidates[0].Finish.Doc = 99
	selected.Page.Doc = 99
	frozen := pub.Selection()
	if frozen.Notes[0].Revision != 1 || frozen.Candidates[0].Finish.Doc != 1 || frozen.Page.Doc != 1 {
		t.Fatal("publication borrowed witnesses")
	}
	frozen.Notes[0].Revision = 88
	frozen.Page.Doc = 88
	if again := pub.Selection(); again.Notes[0].Revision != 1 || again.Page.Doc != 1 {
		t.Fatal("publication getter leaked witnesses")
	}
	changed := pub.Selection()
	changed.Notes[0].Revision++
	state, _ = NextPreview(state, PreviewCandidatesChanged{Selection: changed})
	state, _ = NextPreview(state, PreviewPublishAccepted{Publication: pub})
	if state.Phase() == PreviewShown {
		t.Fatal("note edit accepted stale publication")
	}
}

func TestFeedbackPreviewSelectionCountsOneVisitPerCandidate(t *testing.T) {
	for _, n := range []int{32, 64} {
		all := make([]PreviewSketchCandidate, n)
		delivered := map[PreviewItem]struct{}{}
		for i := range all {
			all[i] = feedbackCandidate(fmt.Sprint(i), 1+i%3, 10+i)
			if i%5 == 0 {
				delivered[all[i].Item] = struct{}{}
			}
		}
		got, visits, err := selectFeedbackPreview(all, nil, delivered, PreviewStamp{})
		if err != nil {
			t.Fatal(err)
		}
		if visits != n || len(got.Items) > 8 {
			t.Fatalf("n=%d visits=%d selection=%+v", n, visits, got)
		}
	}
}
