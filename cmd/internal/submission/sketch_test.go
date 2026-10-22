package submission

import (
	"encoding/json"
	"github.com/xianxu/pair/cmd/internal/webcast/query"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
	"github.com/xianxu/pair/cmd/internal/webcast/sketchimage"
	"strings"
	"testing"
)

func sketchFixture(t *testing.T) Sketch {
	t.Helper()
	box := recording.Box{0, 0, 100, 100}
	history := recording.SketchView{Begin: &recording.SketchBegin{ID: "shape", Author: recording.Author{ID: "operator", Color: "#ffaf00"}}, Phase: "complete", Visible: []string{"stroke"}, Strokes: []recording.SketchStrokeView{{Begin: &recording.SketchStrokeBegin{Sketch: "shape", ID: "stroke", At: recording.NoteAt{Node: 1, Box: &box}, Width: 2}, Pts: []recording.SketchPoint{{20, 20, 0.2}, {40, 40, 0.8}}, Sealed: true}}}
	source := SketchSource{SketchSourceFacts: query.SketchSourceFacts{T: 10, RecordIndex: 2, Doc: 1, Tab: "tab", URL: "https://example.test/old", Title: "Original", VP: recording.Viewport{W: 100, H: 100, DPR: 1}}, Provenance: "recorded-dom"}
	plan, err := sketchimage.Plan(history, source.VP, source.Scroll, sketchimage.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	return Sketch{PrivacyDecision: (OriginPolicy{}).Decide(source.URL), ID: "shape", RecordingID: "rec", NextSequence: 5, Digest: strings.Repeat("a", 64), Author: history.Begin.Author, Note: "shape note", HistoryT: 20, HistoryRecordIndex: 3, Phase: "complete", Source: source, Elements: query.DeclaredElementFacts{Anchors: []query.ElementFact{{Node: 1, Tag: "div", ID: "old-id"}}, Bytes: 41}, Image: SketchImage{File: "sketch-0.png", Provenance: "replay-rendered", Plan: plan}}
}
func sketchSubmission(t *testing.T) Submission {
	s := fixture()
	s.Version = 3
	s.Privacy = Private
	s.Notes = nil
	s.Sketches = []Sketch{sketchFixture(t)}
	return s
}
func TestValidateSketchRefusesForgedOrChangedEvidence(t *testing.T) {
	if err := ValidateSketch(sketchFixture(t)); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*Sketch){
		func(s *Sketch) { s.Source.Provenance = "cdp" }, func(s *Sketch) { s.Image.Provenance = "cdp" }, func(s *Sketch) { s.Image.File = "../escape.png" },
		func(s *Sketch) { s.Digest = "bad" }, func(s *Sketch) { s.NextSequence = 0 }, func(s *Sketch) { s.Note = "" }, func(s *Sketch) { s.Phase = "drawing" },
		func(s *Sketch) { s.Image.Plan.Paths[0].Pts[0][2] = 1.1 }, func(s *Sketch) { s.Image.Plan.Width++ }, func(s *Sketch) { s.Image.Plan.Sketch = "other" },
		func(s *Sketch) { s.Elements.Anchors = nil }, func(s *Sketch) { s.Author.ID = "" }, func(s *Sketch) { s.Privacy = Shareable },
	} {
		s := sketchFixture(t)
		change(&s)
		if ValidateSketch(s) == nil {
			t.Fatalf("accepted forged sketch %#v", s)
		}
	}
}
func TestSketchManifestIsVersionThreeWithoutPrivateAuthority(t *testing.T) {
	s := sketchSubmission(t)
	if err := Validate(s); err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"sketches"`) || strings.Contains(string(b), `"owner"`) || strings.Contains(string(b), `"receipts"`) {
		t.Fatalf("manifest authority leak: %s", b)
	}
	s.Version = 2
	if Validate(s) == nil {
		t.Fatal("sketch silently accepted by legacy version")
	}
}
func TestEvidenceFilesEnumeratesBothKindsAndRejectsAliases(t *testing.T) {
	s := sketchSubmission(t)
	s.Notes = []Note{{Crop: Crop{File: "crop-2.png"}}}
	files, err := EvidenceFiles(s)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(files, ",") != "crop-2.png,sketch-0.png" {
		t.Fatalf("inventory=%v", files)
	}
	for _, name := range []string{"crop-2.png", "../sketch-0.png", "/sketch-0.png", "sketch-00.png", "sketch-1.png/extra"} {
		s.Sketches[0].Image.File = name
		if _, err = EvidenceFiles(s); err == nil {
			t.Fatalf("accepted alias %q", name)
		}
	}
}

func TestSketchImageLimitationsAreBoundedAndPreserved(t *testing.T) {
	sketch := sketchFixture(t)
	sketch.Image.Limitations = []string{"missing recorded image asset"}
	if err := ValidateSketch(sketch); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(sketch)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), "missing recorded image asset") {
		t.Fatal("limitation omitted")
	}
	for _, limits := range [][]string{make([]string, 33), {strings.Repeat("x", 257)}} {
		sketch.Image.Limitations = limits
		if ValidateSketch(sketch) == nil {
			t.Fatal("unbounded image limitations accepted")
		}
	}
}
func TestNoteElementFactsAreOptionalAndBounded(t *testing.T) {
	sub := sketchSubmission(t)
	sub.Sketches = nil
	sub.Notes = []Note{{PrivacyDecision: PrivacyDecision{Privacy: Private}, ID: "note", SourceDocument: 1, Crop: Crop{Unavailable: "recorded capture unavailable"}, At: recording.NoteAt{Node: 1}, Elements: &query.DeclaredElementFacts{Anchors: []query.ElementFact{{Node: 1, Tag: "div", ID: "old-id"}}, Bytes: 41}}}
	if err := Validate(sub); err != nil {
		t.Fatal(err)
	}
	sub.Notes[0].Elements.Anchors[0].Node = 2
	if Validate(sub) == nil {
		t.Fatal("declared anchor from another element accepted")
	}
	sub.Notes[0].Elements = nil
	if err := Validate(sub); err != nil {
		t.Fatal("legacy note unexpectedly requires declared facts", err)
	}
}
