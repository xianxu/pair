package webcast

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	core "github.com/xianxu/pair/cmd/internal/submission"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/notes"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
	"github.com/xianxu/pair/cmd/internal/webcast/sketchimage"
)

func castSketchSubmissionFixture(t *testing.T, n int) (*Cast, *sketchimage.FakeRenderer, board.Selection) {
	t.Helper()
	b := recording.NewBuilder("shapes")
	node := b.El("button", []string{"id", "original", "data-action", "save"})
	b.Page(recording.SegLive, "https://original.test/settings", node)
	selected := board.Selection{Document: 1, TabID: "tab1"}
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("s%d", i)
		op := func(seq uint64) recording.SketchOperation {
			return recording.SketchOperation{Operation: fmt.Sprintf("%s-op%d", id, seq), Sequence: seq, Digest: strings.Repeat("a", 64)}
		}
		at := int64(10 + i*10)
		b.At(at).Add(&recording.SketchBegin{SketchOperation: op(0), ID: id, Author: recording.Author{ID: "operator", Color: "#ff0000"}, Owner: "operator", Doc: 1, Tab: "tab1", VP: recording.Viewport{W: 1280, H: 800, DPR: 1}})
		b.Add(&recording.Attr{ID: node.ID, Name: "id", V: recording.Str("changed-after-begin")})
		b.At(at + 1).Add(&recording.SketchStrokeBegin{SketchOperation: op(1), Sketch: id, ID: "p", At: recording.NoteAt{Node: node.ID, Box: &recording.Box{0, 0, 100, 100}}, Width: 3})
		b.At(at + 2).Add(&recording.SketchPoints{SketchOperation: op(2), Sketch: id, ID: "p", Pts: []recording.SketchPoint{{20, 20, 0}, {40, 40, 1}}})
		b.At(at + 3).Add(&recording.SketchStrokeEnd{SketchOperation: op(3), Sketch: id, ID: "p"})
		b.At(at + 4).Add(&recording.SketchLabel{SketchOperation: op(4), Sketch: id, Label: "Move button"})
		selected.Sketches = append(selected.Sketches, board.SelectedSketch{ID: id, NextSequence: 5, Digest: strings.Repeat("a", 64)})
	}
	b.At(100).Page(recording.SegLive, "https://current.test/", b.El("div", []string{"id", "unrelated"}))
	bd := board.New(notes.Structural{}, nil)
	for _, r := range b.Recs {
		if err := bd.Put(r); err != nil {
			t.Fatal(err)
		}
	}
	dir := t.TempDir()
	for name, recs := range recording.Split(b.Recs) {
		var data []byte
		for _, r := range recs {
			line, err := recording.Encode(r)
			if err != nil {
				t.Fatal(err)
			}
			data = append(data, line...)
			data = append(data, '\n')
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	fake := sketchimage.NewFakeRenderer()
	c := &Cast{board: bd, recDir: dir, done: make(chan struct{}), sketchImages: fake}
	return c, fake, selected
}
func TestCastPackagesSketchOnlyHistoricalSubmission(t *testing.T) {
	c, fake, selected := castSketchSubmissionFixture(t, 2)
	sub, files, err := c.PrepareSubmission(context.Background(), "bundle", castSubmissionTarget(), selected, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(sub.Sketches) != 2 || len(files) != 2 || len(sub.Notes) != 0 {
		t.Fatalf("completed shapes absent from operator submission: sketches=%d files=%d notes=%d", len(sub.Sketches), len(files), len(sub.Notes))
	}
	for i, s := range sub.Sketches {
		if s.ID != fmt.Sprintf("s%d", i) || s.Source.URL != "https://original.test/settings" || s.Source.Provenance != "recorded-dom" || s.Image.Provenance != "replay-rendered" || s.Note != "Move button" || s.Privacy != core.Private || s.HistoryRecordIndex < s.Source.RecordIndex {
			t.Fatalf("wrong evidence: %+v", s)
		}
		if err := core.ValidateSketch(s); err != nil {
			t.Fatal(err)
		}
	}
	if sub.Sketches[0].Elements.Anchors[0].ID != "original" {
		t.Fatal("later same-T source mutation leaked")
	}
	state := fake.Snapshot()
	if state.Opens != 1 || state.Closes != 1 || state.Requests != 2 {
		t.Fatalf("rendered per image rather than batch: %+v", state)
	}
	sub.Delivery = core.Dispatched
	if err := c.ResolveSubmission(context.Background(), sub); err != nil {
		t.Fatalf("sketch dispatch requested note/annotation mutation: %v", err)
	}
}

// This stateful renderer consumes immutable batch inputs even when the original
// file disappears. It can also supply bounded-but-large valid PNGs to exercise
// the Cast's shared manifest/image budget rather than its own per-image bounds.
type submissionSketchRenderer struct {
	fake            *sketchimage.FakeRenderer
	batches         []sketchimage.Batch
	removeEvents    bool
	fillImageBudget bool
	corrupt         bool
}

func (r *submissionSketchRenderer) Render(ctx context.Context, b sketchimage.Batch) ([]sketchimage.Image, error) {
	r.batches = append(r.batches, b)
	if r.removeEvents {
		if err := os.Remove(filepath.Join(b.RecordingDir, recording.FileEvents)); err != nil {
			return nil, err
		}
	}
	images, err := r.fake.Render(ctx, b)
	if err != nil {
		return nil, err
	}
	for i := range images {
		if r.fillImageBudget {
			data := make([]byte, sketchimage.MaxImageBytes)
			copy(data, images[i].PNG)
			images[i].PNG = data
		}
	}
	if r.corrupt {
		images[0].Plan.Width++
	}
	return images, nil
}
func TestCastSketchPreparationReadsOnceAndUsesOriginalPrivacy(t *testing.T) {
	c, fake, selected := castSketchSubmissionFixture(t, 1)
	renderer := &submissionSketchRenderer{fake: fake, removeEvents: true}
	c.sketchImages = renderer
	policy, err := core.ParseOriginPolicy([]byte(`{"origins":[{"origin":"https://original.test","privacy":"shareable"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	reads := 0
	c.readOriginPolicy = func() (core.OriginPolicy, string) { reads++; return policy, "" }
	sub, files, err := c.PrepareSubmission(context.Background(), "once", castSubmissionTarget(), selected, "", nil)
	if err != nil || len(sub.Sketches) != 1 || len(files) != 1 || len(renderer.batches) != 1 || reads != 1 {
		t.Fatalf("one-read batch: %v %d %d %d", err, len(sub.Sketches), len(renderer.batches), reads)
	}
	if sub.Privacy != core.Shareable || sub.Sketches[0].PrivacyOrigin != "https://original.test" {
		t.Fatalf("privacy substituted current page: %+v", sub.Sketches[0].PrivacyDecision)
	}
	if len(sub.Sketches[0].Image.Limitations) == 0 {
		t.Fatal("renderer provenance limitations dropped")
	}
}
func TestCastSketchPreparationFailureRetainsInk(t *testing.T) {
	for _, kind := range []string{"render", "changed", "corrupt", "too-many", "canceled"} {
		t.Run(kind, func(t *testing.T) {
			count := 1
			if kind == "too-many" {
				count = 9
			}
			c, fake, selected := castSketchSubmissionFixture(t, count)
			ctx := context.Background()
			switch kind {
			case "render":
				fake.Configure(sketchimage.FakeControl{Stage: sketchimage.StageImage, Image: 0, Err: errors.New("controlled render failure")})
			case "changed":
				selected.Sketches[0].Digest = strings.Repeat("b", 64)
			case "corrupt":
				c.sketchImages = &submissionSketchRenderer{fake: fake, corrupt: true}
			case "canceled":
				next, cancel := context.WithCancel(ctx)
				cancel()
				ctx = next
			}
			if _, _, err := c.PrepareSubmission(ctx, "refusal", castSubmissionTarget(), selected, "", nil); err == nil {
				t.Fatal("failed evidence preparation was accepted")
			}
			state := fake.Snapshot()
			if state.Active != 0 || state.Opens != state.Closes {
				t.Fatalf("renderer cleanup incomplete: %+v", state)
			}
			after, err := c.board.Selection()
			if err != nil || len(after.Sketches) != count {
				t.Fatalf("preparation changed ink: %v %+v", err, after.Sketches)
			}
		})
	}
}
func TestCastSketchImagesAndManifestShareBundleBudget(t *testing.T) {
	c, fake, selected := castSketchSubmissionFixture(t, 8)
	c.sketchImages = &submissionSketchRenderer{fake: fake, fillImageBudget: true}
	_, _, err := c.PrepareSubmission(context.Background(), "oversized", castSubmissionTarget(), selected, "", nil)
	if err == nil || !strings.Contains(err.Error(), "64 MiB") {
		t.Fatalf("manifest plus bounded images exceeded bundle silently: %v", err)
	}
	state := fake.Snapshot()
	if state.Opens != 1 || state.Requests != 8 || state.Active != 0 {
		t.Fatalf("bad batch lifecycle: %+v", state)
	}
}
