package submission

import (
	core "github.com/xianxu/pair/cmd/internal/submission"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
	"strings"
	"testing"
)

func buildFixture() (*recording.Builder, board.Selection) {
	b := recording.NewBuilder("recording")
	el := b.El("button", nil, b.Txt("Original save"))
	b.Page(recording.SegLive, "https://example.test", el)
	box := recording.Box{20, 20, 100, 40}
	b.At(10).Add(&recording.Note{ID: "a", Text: "first", Author: recording.Author{ID: "one"}, At: recording.NoteAt{Node: el.ID, Text: "Original save", Box: &box}, Region: &box})
	b.At(20).Add(&recording.Note{ID: "b", Text: "second", Author: recording.Author{ID: "two"}, At: recording.NoteAt{Node: el.ID, Text: "Original save", Box: &box}, Region: &box})
	return b, board.Selection{Document: 1, TabID: "tab1", Notes: []board.SelectedNote{{ID: "b", Revision: 3}, {ID: "a", Revision: 1}}}
}
func target() core.Target {
	return core.Target{Scope: "scope", Tag: "tag", Session: "session", Nonce: "nonce"}
}
func TestSubmissionBuildPreservesSelectionAndHistoricalContext(t *testing.T) {
	b, sel := buildFixture()
	b.At(30).Page(recording.SegNavigate, "https://other.test", b.El("p", nil, b.Txt("New page")))
	sub, boxes, err := Build(b.Recs, sel, "submission", target(), "/recording", func(t int64) string { return "replay" })
	if err != nil {
		t.Fatal(err)
	}
	if len(sub.Notes) != 2 || len(boxes) != 2 || boxes[0] != boxes[1] {
		t.Fatal(sub, boxes)
	}
	if sub.Notes[0].ID != "b" || sub.Notes[0].Revision != 3 || sub.Notes[1].ID != "a" {
		t.Fatal(sub.Notes)
	}
	for _, n := range sub.Notes {
		if !strings.Contains(n.Element, "Original save") || strings.Contains(n.Context.Text, "New page") || n.Replay != "replay" {
			t.Fatal(n)
		}
		if n.Crop.Unavailable == "" {
			t.Fatal("missing placeholder")
		}
	}
}
func TestSubmissionBuildRejectsMissingAndWrongDocument(t *testing.T) {
	b, sel := buildFixture()
	sel.Notes = append(sel.Notes, board.SelectedNote{ID: "missing"})
	if _, _, err := Build(b.Recs, sel, "submission", target(), "/recording", nil); err == nil {
		t.Fatal("missing selected note omitted")
	}
	sel.Notes = sel.Notes[:2]
	sel.Document = 2
	if _, _, err := Build(b.Recs, sel, "submission", target(), "/recording", nil); err == nil {
		t.Fatal("wrong document accepted")
	}
}

func TestSubmissionBuildMissingElementKeepsAnchorText(t *testing.T) {
	b, sel := buildFixture()
	for _, r := range b.Recs {
		if n, ok := r.(*recording.Note); ok {
			n.At.Node = 9999
			n.At.Sel = ""
		}
	}
	sub, _, err := Build(b.Recs, sel, "submission", target(), "/recording", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range sub.Notes {
		if !strings.Contains(n.Element, "unavailable") || n.Context.Text != "Original save" {
			t.Fatal(n)
		}
	}
}
