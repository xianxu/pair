package submission

import (
	"encoding/json"
	core "github.com/xianxu/pair/cmd/internal/submission"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
	"os"
	"path/filepath"
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
func TestSubmissionBuildRejectsMissingAndKeepsOriginalDocument(t *testing.T) {
	b, sel := buildFixture()
	sel.Notes = append(sel.Notes, board.SelectedNote{ID: "missing"})
	if _, _, err := Build(b.Recs, sel, "submission", target(), "/recording", nil); err == nil {
		t.Fatal("missing selected note omitted")
	}
	sel.Notes = sel.Notes[:2]
	sel.Document = 2
	sub, _, err := Build(b.Recs, sel, "submission", target(), "/recording", nil)
	if err != nil || sub.Notes[0].SourceDocument != 1 {
		t.Fatalf("historical document lost: %+v %v", sub, err)
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

func TestSubmissionBuildReadingGuideSurvivesStoreRoundTrip(t *testing.T) {
	b, selected := buildFixture()
	recordingPath := filepath.Join(t.TempDir(), "recordings with spaces", "rec")
	sub, _, err := Build(b.Recs, selected, "guide", target(), recordingPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := core.ReadingGuide(recordingPath)
	if sub.HowToRead != want {
		t.Fatalf("Build lost reading guide: got %q, want %q", sub.HowToRead, want)
	}
	for _, fragment := range []string{
		"submission.json", "crop-N.png", recordingPath,
		"couch --broadcast-web-query DIR notes|dom|events|link",
		"dom --at <t> --sel @N", "events --from <t-10000> --to <t>",
		"link --at <t>", "couch --skill",
	} {
		if !strings.Contains(sub.HowToRead, fragment) {
			t.Fatalf("guide missing %q: %s", fragment, sub.HowToRead)
		}
	}
	storeRoot := filepath.Join(t.TempDir(), "bundles")
	store, err := core.NewStore(storeRoot)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := store.Prepare(sub, nil)
	if err != nil {
		t.Fatal(err)
	}
	reopened, err := core.NewStore(storeRoot)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := reopened.Read(sub.ID)
	if err != nil {
		t.Fatal(err)
	}
	if restored.HowToRead != sub.HowToRead || restored.Source.RecordingPath != recordingPath {
		t.Fatalf("store changed reading instructions: %+v", restored)
	}
	data, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]json.RawMessage
	if err := json.Unmarshal(data, &wire); err != nil {
		t.Fatal(err)
	}
	var encodedGuide string
	if err := json.Unmarshal(wire["how_to_read"], &encodedGuide); err != nil {
		t.Fatal("reading guide missing from manifest JSON", err)
	}
	if encodedGuide != want {
		t.Fatalf("manifest guide changed: %q", encodedGuide)
	}
}

func TestSubmissionBuildDropsSelectorsWithoutMutatingRecording(t *testing.T) {
	b, selected := buildFixture()
	for _, r := range b.Recs {
		if n, ok := r.(*recording.Note); ok {
			n.At.Sel = "button:nth-of-type(1)"
		}
	}
	s, _, err := Build(b.Recs, selected, "selector-free", target(), "/recording", nil)
	if err != nil {
		t.Fatal(err)
	}
	if s.Version != 3 {
		t.Fatal("not current schema", s.Version)
	}
	for _, n := range s.Notes {
		if n.At.Sel != "" || n.At.Node == 0 || n.SourceDocument != selected.Document || !strings.Contains(n.Element, "button") {
			t.Fatal(n)
		}
	}
	bjson, _ := json.Marshal(s)
	if strings.Contains(string(bjson), `"sel":`) {
		t.Fatal(string(bjson))
	}
	for _, r := range b.Recs {
		if n, ok := r.(*recording.Note); ok && n.At.Sel == "" {
			t.Fatal("mutated recording")
		}
	}
}

func TestSubmissionBuildMixedDocumentsAndCaptureReferences(t *testing.T) {
	b, selected := buildFixture()
	for _, rec := range b.Recs {
		if n, ok := rec.(*recording.Note); ok && n.ID == "a" {
			n.Capture = "draw-a"
			n.CropUnavailable = "capture busy"
		}
	}
	el := b.El("button", nil, b.Txt("Later page"))
	b.At(30).Page(recording.SegNavigate, "https://other.test", el)
	box := recording.Box{20, 20, 100, 40}
	b.At(40).Add(&recording.Note{ID: "later", Text: "later note", Author: recording.Author{ID: "three"}, At: recording.NoteAt{Node: el.ID, Box: &box}, Region: &box})
	selected.Document = 2
	selected.Notes = append(selected.Notes, board.SelectedNote{ID: "later", Revision: 1})
	sub, _, err := Build(b.Recs, selected, "mixed", target(), "/recording", nil)
	if err != nil {
		t.Fatal(err)
	}
	if sub.Version != 3 || sub.Notes[1].CaptureID != "draw-a" || sub.Notes[1].Crop.Unavailable != "capture busy" || sub.Notes[1].SourceDocument != 1 || sub.Notes[2].SourceDocument != 2 || sub.Notes[2].Crop.Unavailable != "not captured when drawn" {
		t.Fatalf("wrong historical sources: %+v", sub)
	}
}
