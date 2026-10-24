package webcast

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	core "github.com/xianxu/pair/cmd/internal/submission"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/notes"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func TestCastSketchPageUsesLatestSelectedCompletion(t *testing.T) {
	c, fake, selected := castSketchSubmissionFixture(t, 3)
	unselected := selected.Sketches[2]
	selected.Sketches = selected.Sketches[:2]
	renderer := &submissionSketchRenderer{fake: fake}
	c.sketchPages = renderer
	sub, files, err := c.PrepareSubmission(context.Background(), "selected-page", castSubmissionTarget(), selected, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if sub.SketchPage == nil || len(files) != 1 || len(files[sub.SketchPage.File]) == 0 || len(sub.Sketches) != 2 {
		t.Fatalf("expected one page for two selected sketches: %+v files=%d", sub, len(files))
	}
	if len(renderer.batches) != 1 {
		t.Fatalf("render calls=%d", len(renderer.batches))
	}
	batch := renderer.batches[0]
	want := selected.Sketches[1].Finish
	if batch.Scene.SketchSourceFacts != want || sub.SketchPage.Source.SketchSourceFacts != want || batch.Scene.RecordIndex >= unselected.Finish.RecordIndex {
		t.Fatalf("later unselected feedback moved base scene: batch=%+v manifest=%+v unselected=%+v", batch.Scene, sub.SketchPage.Source, unselected.Finish)
	}
	if len(batch.Sketches) != 2 || len(batch.Finishes) != 2 {
		t.Fatalf("unselected sketch leaked into batch: sketches=%d finishes=%d", len(batch.Sketches), len(batch.Finishes))
	}
	for i, item := range sub.Sketches {
		original := selected.Sketches[i]
		if item.ID != original.ID || item.Index != i+1 || item.Source.SketchSourceFacts != original.Source || item.Finish == nil || item.Finish.SketchSourceFacts != original.Finish || batch.Sketches[i].History.Begin.ID != original.ID {
			t.Fatalf("selected original source/version/order replaced: %+v", item)
		}
	}
	if sub.Sketches[0].Elements.Anchors[0].ID != "original" {
		t.Fatal("completion context replaced original declared ID")
	}
	state := fake.Snapshot()
	if state.Opens != 1 || state.Requests != 1 || state.Closes != 1 || state.Active != 0 {
		t.Fatalf("expected one joined page session: %+v", state)
	}
}

func TestCastSketchPageRejectsForgedWitnessBeforeRendering(t *testing.T) {
	for _, kind := range []string{"source-url", "source-index", "finish-scroll", "finish-index"} {
		t.Run(kind, func(t *testing.T) {
			c, fake, selected := castSketchSubmissionFixture(t, 1)
			before, err := c.board.Selection()
			if err != nil {
				t.Fatal(err)
			}
			inkBefore, err := c.board.OperatorSketchSince(board.SketchDeliveryCursor{})
			if err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "source-url":
				selected.Sketches[0].Source.URL = "https://substituted.test/"
			case "source-index":
				selected.Sketches[0].Source.RecordIndex++
			case "finish-scroll":
				selected.Sketches[0].Finish.Scroll[1]++
			case "finish-index":
				selected.Sketches[0].Finish.RecordIndex++
			}
			if _, _, err := c.PrepareSubmission(context.Background(), "forged-page", castSubmissionTarget(), selected, "", nil); err == nil {
				t.Fatal("forged recorded witness accepted")
			}
			if state := fake.Snapshot(); state.Opens != 0 || state.Requests != 0 || state.Active != 0 {
				t.Fatalf("invalid witness started renderer: %+v", state)
			}
			after, err := c.board.Selection()
			if err != nil {
				t.Fatal(err)
			}
			inkAfter, err := c.board.OperatorSketchSince(board.SketchDeliveryCursor{})
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(before, after) || inkBefore.Bootstrap == nil || inkAfter.Bootstrap == nil || !reflect.DeepEqual(inkBefore.Bootstrap.Display, inkAfter.Bootstrap.Display) {
				t.Fatal("refused preparation changed Board versions or visible ink")
			}
		})
	}
}

func TestCastSketchPageCompositePrivacyIncludesPrivateOriginalSource(t *testing.T) {
	c, fake, selected := castSketchSubmissionFixture(t, 1)
	read, err := recording.ReadDir(c.RecordingPath())
	if err != nil {
		t.Fatal(err)
	}
	var records []recording.Record
	for _, r := range read.Records {
		if _, ok := r.(*recording.SketchLabel); ok {
			// Recorded same-document navigation changes the finish URL without
			// replacing the original BEGIN context or its declared element facts.
			nav := &recording.Nav{URL: "https://allowed.test/finish"}
			nav.SetTime(r.Time())
			records = append(records, nav)
		}
		records = append(records, r)
	}
	bd := board.New(notes.Structural{}, nil)
	var snapshot board.Selection
	for _, r := range records {
		bd.Put(r)
		if _, ok := r.(*recording.SketchLabel); ok {
			snapshot, err = bd.Selection()
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if len(snapshot.Sketches) != 1 {
		t.Fatalf("original press lost: %+v", snapshot)
	}
	current, err := bd.Selection()
	if err != nil || len(current.Sketches) != 0 {
		t.Fatalf("later page included original load: %+v, %v", current, err)
	}
	c.board = bd
	selected = snapshot
	for name, recs := range recording.Split(records) {
		var data bytes.Buffer
		for _, r := range recs {
			line, err := recording.Encode(r)
			if err != nil {
				t.Fatal(err)
			}
			data.Write(line)
			data.WriteByte('\n')
		}
		if err := os.WriteFile(filepath.Join(c.RecordingPath(), name), data.Bytes(), 0600); err != nil {
			t.Fatal(err)
		}
	}
	policy, err := core.ParseOriginPolicy([]byte(`{"origins":[{"origin":"https://allowed.test","privacy":"shareable"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	c.readOriginPolicy = func() (core.OriginPolicy, string) { return policy, "" }
	if policy.Decide(selected.Sketches[0].Finish.URL).Privacy != core.Shareable {
		t.Fatal("fixture finish must independently be shareable")
	}
	sub, files, err := c.PrepareSubmission(context.Background(), "composite-privacy", castSubmissionTarget(), selected, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if sub.SketchPage == nil || len(files) != 1 || len(sub.Sketches) != 1 {
		t.Fatal("missing shared page evidence")
	}
	item := sub.Sketches[0]
	if sub.Privacy != core.Private || sub.SketchPage.Privacy != core.Private || item.Privacy != core.Private {
		t.Fatalf("shareable base downgraded private contributor: aggregate=%s page=%s item=%s", sub.Privacy, sub.SketchPage.Privacy, item.Privacy)
	}
	if sub.SketchPage.Source.URL != "https://allowed.test/finish" || sub.SketchPage.PrivacyOrigin != "https://allowed.test" || sub.SketchPage.PrivacyRule != (core.PolicyEvidence{}) {
		t.Fatalf("composite misattributed shareable base grant: %+v", sub.SketchPage)
	}
	if item.Source.URL != "https://original.test/settings" || item.Source.SketchSourceFacts != selected.Sketches[0].Source || item.Elements.Anchors[0].ID != "original" || item.Finish == nil || item.Finish.URL != "https://allowed.test/finish" {
		t.Fatalf("privacy evaluation replaced original evidence: %+v", item)
	}
	if state := fake.Snapshot(); state.Opens != 1 || state.Requests != 1 || state.Active != 0 {
		t.Fatalf("bad composite session: %+v", state)
	}
}
