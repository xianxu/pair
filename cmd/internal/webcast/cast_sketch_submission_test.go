package webcast

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	core "github.com/xianxu/pair/cmd/internal/submission"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/notes"
	"github.com/xianxu/pair/cmd/internal/webcast/query"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
	"github.com/xianxu/pair/cmd/internal/webcast/sketchimage"
)

func castSketchSubmissionFixture(t *testing.T, n int) (*Cast, *sketchimage.FakePageRenderer, board.Selection) {
	t.Helper()
	b := recording.NewBuilder("shapes")
	node := b.El("button", []string{"id", "original", "data-action", "save"})
	b.Page(recording.SegLive, "https://original.test/settings", node)
	b.Recs[1].(*recording.Seg).PageLoad = &recording.PageLoad{Target: "tab1", Loader: "original-load", Root: 1}
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
	}
	bd := board.New(notes.Structural{}, nil)
	for _, r := range b.Recs {
		if err := bd.Put(r); err != nil {
			t.Fatal(err)
		}
	}
	// The press pins exact accepted witnesses before the browser leaves this
	// load. Evidence preparation may finish after a genuinely unrelated page.
	selected, err := bd.Selection()
	if err != nil || len(selected.Sketches) != n {
		t.Fatalf("original press: %+v, %v", selected, err)
	}
	previous := len(b.Recs)
	b.At(100).Page(recording.SegNavigate, "https://current.test/", b.El("div", []string{"id", "unrelated"}))
	b.Recs[previous].(*recording.Seg).PageLoad = &recording.PageLoad{Target: "tab1", Loader: "later-load", Root: 2}
	for _, r := range b.Recs[previous:] {
		if err := bd.Put(r); err != nil {
			t.Fatal(err)
		}
	}
	current, err := bd.Selection()
	if err != nil || len(current.Sketches) != 0 {
		t.Fatalf("later page selected original-load ink: %+v, %v", current, err)
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
	fake := sketchimage.NewFakePageRenderer()
	c := &Cast{board: bd, recDir: dir, done: make(chan struct{}), sketchPages: fake}
	return c, fake, selected
}

func castSketchHistory(t *testing.T, c *Cast) []query.SketchView {
	t.Helper()
	read, err := recording.ReadDir(c.RecordingPath())
	if err != nil {
		t.Fatal(err)
	}
	views, err := query.New(read.Records, nil).SketchesAt(math.MaxInt64)
	if err != nil {
		t.Fatal(err)
	}
	return views
}
func TestCastPackagesSketchOnlyHistoricalSubmission(t *testing.T) {
	c, fake, selected := castSketchSubmissionFixture(t, 2)
	sub, files, err := c.PrepareSubmission(context.Background(), "bundle", castSubmissionTarget(), selected, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(sub.Sketches) != 2 || len(files) != 1 || len(sub.Notes) != 0 {
		t.Fatalf("completed shapes absent from operator submission: sketches=%d files=%d notes=%d", len(sub.Sketches), len(files), len(sub.Notes))
	}
	for i, s := range sub.Sketches {
		if s.ID != fmt.Sprintf("s%d", i) || s.Source.URL != "https://original.test/settings" || s.Source.Provenance != "recorded-dom" || s.Image != nil || s.Index != i+1 || len(s.Paths) == 0 || sub.SketchPage == nil || sub.SketchPage.Provenance != "replay-rendered" || s.Note != "Move button" || s.Privacy != core.Private || s.HistoryRecordIndex < s.Source.RecordIndex {
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
	if state.Opens != 1 || state.Closes != 1 || state.Requests != 1 {
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
	fake            *sketchimage.FakePageRenderer
	batches         []sketchimage.PageBatch
	removeEvents    bool
	fillImageBudget bool
	corrupt         bool
}

func (r *submissionSketchRenderer) RenderPage(ctx context.Context, b sketchimage.PageBatch) (sketchimage.PageImage, error) {
	r.batches = append(r.batches, b)
	if r.removeEvents {
		if err := os.Remove(filepath.Join(b.RecordingDir, recording.FileEvents)); err != nil {
			return sketchimage.PageImage{}, err
		}
	}
	image, err := r.fake.RenderPage(ctx, b)
	if err != nil {
		return image, err
	}
	if r.fillImageBudget {
		data := make([]byte, sketchimage.MaxImageBytes)
		copy(data, image.PNG)
		image.PNG = data
	}
	if r.corrupt {
		image.Plan.Width++
	}
	return image, nil
}

func TestCastSketchPreparationReadsOnceAndUsesOriginalPrivacy(t *testing.T) {
	c, fake, selected := castSketchSubmissionFixture(t, 1)
	renderer := &submissionSketchRenderer{fake: fake, removeEvents: true}
	c.sketchPages = renderer
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
	if len(sub.SketchPage.Limitations) == 0 {
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
			historyBefore := castSketchHistory(t, c)
			ctx := context.Background()
			switch kind {
			case "render":
				fake.Configure(sketchimage.FakeControl{Stage: sketchimage.StageImage, Image: 0, Err: errors.New("controlled render failure")})
			case "changed":
				selected.Sketches[0].Digest = strings.Repeat("b", 64)
			case "corrupt":
				c.sketchPages = &submissionSketchRenderer{fake: fake, corrupt: true}
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
			if err != nil || len(after.Sketches) != 0 || !reflect.DeepEqual(historyBefore, castSketchHistory(t, c)) || len(historyBefore) != count {
				t.Fatalf("preparation changed retained ink or selected another load: %v %+v", err, after.Sketches)
			}
		})
	}
}
func TestCastSketchPageSharesBundleBudgetWithOtherFeedback(t *testing.T) {
	c, fake, selected := castSketchSubmissionFixture(t, 8)
	c.sketchPages = &submissionSketchRenderer{fake: fake, fillImageBudget: true}
	rec, err := recording.ReadDir(c.RecordingPath())
	if err != nil {
		t.Fatal(err)
	}
	sub := core.Submission{Version: 3, Source: core.Source{Kind: "dom", RecordingID: "shapes"}}
	total := core.MaxBundleBytes - sketchimage.MaxImageBytes + 1
	err = c.appendSubmissionSketches(context.Background(), &sub, map[string][]byte{}, rec.Records, selected, core.OriginPolicy{}, &total, nil)
	if err == nil || !strings.Contains(err.Error(), "bundle budget") {
		t.Fatalf("combined feedback budget accepted: %v", err)
	}
	state := fake.Snapshot()
	if state.Opens != 1 || state.Requests != 1 || state.Active != 0 {
		t.Fatalf("bad one-page lifecycle: %+v", state)
	}
}

// Exercise the actual Cast evidence builder and durable store together. The
// current page has already navigated; a second target's merely prepared bundle
// must not suppress that shape, while ACKed versions suppress across targets.
func TestCastSketchSubmissionDurableLifecycleAcrossTargets(t *testing.T) {
	ctx := context.Background()
	c, _, before := castSketchSubmissionFixture(t, 3)
	currentBefore, err := c.SubmissionSelection()
	if err != nil || len(currentBefore.Sketches) != 0 {
		t.Fatalf("later page selection: %+v, %v", currentBefore, err)
	}
	historyBefore := castSketchHistory(t, c)
	inkBefore, err := c.board.OperatorSketchSince(board.SketchDeliveryCursor{})
	if err != nil || inkBefore.Bootstrap == nil {
		t.Fatalf("initial ink: %v", err)
	}
	selected := before
	selected.Sketches = before.Sketches[:2]
	target := castSubmissionTarget()
	sub, files, err := c.PrepareSubmission(ctx, "delivered-shapes", target, selected, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(t.TempDir(), "submissions")
	store, err := core.NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.Prepare(sub, files); err != nil {
		t.Fatal(err)
	}
	otherTarget := target
	otherTarget.Scope = "other-scope"
	otherTarget.Tag = "other-tag"
	otherTarget.Session = "other-session"
	otherTarget.Nonce = "other-nonce"
	pendingSelection := before
	pendingSelection.Sketches = before.Sketches[2:]
	pending, pendingFiles, err := c.PrepareSubmission(ctx, "other-target-prepared", otherTarget, pendingSelection, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.Prepare(pending, pendingFiles); err != nil {
		t.Fatal(err)
	}
	delivered, err := store.DeliveredSketchVersions(sub.Source.RecordingID)
	if err != nil || len(delivered) != 0 {
		t.Fatalf("Prepared suppressed sendable ink: %v %+v", err, delivered)
	}
	begun, err := store.Transition(sub.ID, target, core.Begin)
	if err != nil {
		t.Fatal(err)
	}
	if begun.Delivery == core.Dispatched || begun.Delivery == core.Resolved {
		t.Fatal("Begin falsely acknowledged delivery")
	}
	delivered, err = store.DeliveredSketchVersions(sub.Source.RecordingID)
	if err != nil || len(delivered) != 0 {
		t.Fatalf("Begin suppressed unacknowledged ink: %v %+v", err, delivered)
	}
	acknowledged, err := store.Transition(sub.ID, target, core.Ack)
	if err != nil {
		t.Fatal(err)
	}
	if acknowledged.Delivery != core.Dispatched {
		t.Fatalf("ACK disposition: %s", acknowledged.Delivery)
	}
	if err = c.ResolveSubmission(ctx, acknowledged); err != nil {
		t.Fatal(err)
	}
	completed, err := store.Transition(sub.ID, target, core.Complete)
	if err != nil || completed.Delivery != core.Resolved {
		t.Fatalf("Complete: %v %s", err, completed.Delivery)
	}
	// Reopening observes only durable manifests/files, not the original DTO or
	// in-memory delivery transaction.
	reopened, err := core.NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	read, err := reopened.Read(sub.ID)
	if err != nil {
		t.Fatal(err)
	}
	if read.Delivery != core.Resolved || read.Target != target || !reflect.DeepEqual(read.Sketches, sub.Sketches) {
		t.Fatalf("durable evidence/delivery changed: %+v", read)
	}
	if len(read.Sketches) != 2 || read.Sketches[0].Elements.Anchors[0].ID != "original" {
		t.Fatal("original declared element identity was replaced after navigation")
	}
	for _, sketch := range read.Sketches {
		if sketch.Source.URL != "https://original.test/settings" || sketch.Note != "Move button" {
			t.Fatalf("live page replaced historical facts: %+v", sketch)
		}
		points := sketch.Paths[0].Pts
		if len(points) != 2 || points[0][2] != 0 || points[1][2] != 1 {
			t.Fatalf("pressure lost during durable round trip: %+v", points)
		}
		path := filepath.Join(filepath.Dir(reopened.ManifestPath(sub.ID)), read.SketchPage.File)
		pngBytes, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(pngBytes, files[read.SketchPage.File]) {
			t.Fatal("durable PNG reference differs from prepared evidence")
		}
		config, err := png.DecodeConfig(bytes.NewReader(pngBytes))
		if err != nil || config.Width != read.SketchPage.Width || config.Height != read.SketchPage.Height {
			t.Fatalf("durable image transform: %v %+v", err, config)
		}
	}
	delivered, err = reopened.DeliveredSketchVersions(sub.Source.RecordingID)
	if err != nil || len(delivered) != 2 {
		t.Fatalf("reopened global delivery index: %v %+v", err, delivered)
	}
	for _, sketch := range read.Sketches {
		key := core.PreviewItem{RecordingID: sketch.RecordingID, SketchID: sketch.ID, Sequence: sketch.NextSequence, Digest: sketch.Digest}
		if _, ok := delivered[key]; !ok {
			t.Fatalf("exact dispatched version missing: %+v", key)
		}
	}
	pendingRead, err := reopened.Read(pending.ID)
	if err != nil || pendingRead.Target != otherTarget || pendingRead.Delivery != core.Prepared {
		t.Fatalf("other target prepared state: %v %+v", err, pendingRead)
	}
	after, err := c.SubmissionSelection()
	if err != nil {
		t.Fatal(err)
	}
	inkAfter, err := c.board.OperatorSketchSince(board.SketchDeliveryCursor{})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(currentBefore, after) || inkAfter.Bootstrap == nil || !reflect.DeepEqual(inkBefore.Bootstrap.Display, inkAfter.Bootstrap.Display) || !reflect.DeepEqual(historyBefore, castSketchHistory(t, c)) {
		t.Fatal("dispatch/resolve changed original ink or accepted versions")
	}
	// Delivery recovery evaluates candidates pinned by the original presses;
	// a new press on the unrelated current load still has no sketches.
	candidates := make([]core.PreviewItem, 0, len(before.Sketches))
	for _, sketch := range before.Sketches {
		candidates = append(candidates, core.PreviewItem{RecordingID: sub.Source.RecordingID, SketchID: sketch.ID, Sequence: sketch.NextSequence, Digest: sketch.Digest})
	}
	preview := core.SelectSketchPreview(candidates, delivered, core.PreviewStamp{})
	if len(preview.Items) != 1 || preview.Items[0].SketchID != pendingRead.Sketches[0].ID || preview.Remaining != 0 {
		t.Fatalf("target-independent policy resent delivered versions or hid merely Prepared evidence: %+v", preview)
	}
}

func TestCastSketchPageUsesFrozenSubmissionView(t *testing.T) {
	c, fake, selected := castSketchSubmissionFixture(t, 1)
	rec, err := recording.ReadDir(c.RecordingPath())
	if err != nil {
		t.Fatal(err)
	}
	scene := sketchimage.PageScene{RecordingID: "shapes", SketchSourceFacts: selected.Sketches[0].Finish}
	// A later source frame is the chord context, independently of sketch finish.
	recs := append([]recording.Record(nil), rec.Records[:scene.RecordIndex+1]...)
	recs = append(recs, &recording.Attr{Header: recording.Header{T: scene.T + 1}, ID: 2, Name: "title", V: recording.Str("current frame")})
	scene.T++
	scene.RecordIndex++
	box := recording.Box{0, 0, 100, 100}
	selected.CurrentPage = &scene
	selected.Current = &sketchimage.CurrentView{Projections: []sketchimage.PathProjection{{SketchID: selected.Sketches[0].ID, PathID: "p", At: recording.NoteAt{Node: 2, Box: &box}, Clip: recording.Box{0, 0, 1280, 800}}}}
	sub := core.Submission{Version: 3, Source: core.Source{Kind: "dom", RecordingID: scene.RecordingID}}
	total := 0
	err = c.appendSubmissionSketches(context.Background(), &sub, map[string][]byte{}, recs, selected, core.OriginPolicy{}, &total, nil)
	if err != nil {
		t.Fatal(err)
	}
	if sub.SketchPage.Source != scene || sub.SketchPage.Current == nil || sub.SketchPage.Provenance != "replay-rendered-current-view" {
		t.Fatalf("page=%+v", sub.SketchPage)
	}
	if sub.Sketches[0].Finish.SketchSourceFacts != selected.Sketches[0].Finish || fake.Snapshot().Requests != 1 {
		t.Fatal("history changed or image not shared")
	}
}

// pair#849: what a press submits carries no live view, so each sketch renders
// from the recording at its own recorded finish, whatever the page shows now.
func TestCastPressSelectionRendersFromTheRecording(t *testing.T) {
	c, _, selected := castSketchSubmissionFixture(t, 1)
	live := sketchimage.PageScene{RecordingID: "shapes", SketchSourceFacts: selected.Sketches[0].Finish}
	selected.CurrentPage = &live
	selected.Current = &sketchimage.CurrentView{}
	sub, _, err := c.PrepareSubmission(context.Background(), "press", castSubmissionTarget(), selected.PressSelection(core.DeliveredFeedback{}), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if sub.SketchPage == nil || sub.SketchPage.Provenance != "replay-rendered" || sub.SketchPage.Current != nil || sub.SketchPage.Source.SketchSourceFacts != selected.Sketches[0].Finish {
		t.Fatalf("page=%+v", sub.SketchPage)
	}
}
