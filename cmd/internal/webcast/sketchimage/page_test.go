package sketchimage

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/webcast/query"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func pageFixture(t *testing.T, n int) PageBatch {
	t.Helper()
	b := rendererBatch(t, n)
	state := recording.NewState()
	finishes := make([]query.SketchSourceFacts, n)
	for index, r := range b.Records {
		if err := state.Apply(r); err != nil {
			t.Fatal(err)
		}
		for i, v := range b.Sketches {
			if index == v.HistoryRecordIndex {
				facts := query.SketchSourceFacts{T: r.Time(), RecordIndex: index, Doc: state.DocSeg, Tab: state.Tab.ID, URL: state.URL, Title: state.Title, VP: state.VP}
				if state.Top != nil {
					facts.Scroll = state.Scrolls[state.Top.ID]
				}
				finishes[i] = facts
			}
		}
	}
	return PageBatch{Batch: b, Finishes: finishes, Scene: PageScene{RecordingID: "renderer", SketchSourceFacts: finishes[n-1]}}
}
func TestPagePlanUsesEntireFinishViewportAndBoundedScale(t *testing.T) {
	b := pageFixture(t, 2)
	plan, err := PlanPage(b.Scene)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Region != (recording.Box{0, 0, 1280, 800}) || plan.Width != 1280 || plan.Height != 800 || plan.Scale != 1 {
		t.Fatalf("cropped page: %+v", plan)
	}
	large := b.Scene
	large.VP = recording.Viewport{W: 16384, H: 16384, DPR: 2}
	plan, err = PlanPage(large)
	if err != nil || plan.Width*plan.Height > DefaultLimits().MaxPixels || plan.Scale >= 1 {
		t.Fatalf("unbounded viewport: %v %+v", err, plan)
	}
	other := b.Scene
	other.T++
	other.RecordIndex++
	other.Title = "Later title"
	if !SamePageView(b.Scene, other) {
		t.Fatal("time/index/title incorrectly split compatible view")
	}
	other.Scroll[1]++
	if SamePageView(b.Scene, other) {
		t.Fatal("different scroll entered same page batch")
	}
}
func TestPageBatchWitnessesExactLatestFinishAndOriginalHistory(t *testing.T) {
	b := pageFixture(t, 2)
	if _, err := ValidatePageBatch(b); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*PageBatch){func(b *PageBatch) { b.Scene.RecordIndex-- }, func(b *PageBatch) { b.Finishes[0].URL = "https://forged.test/" }, func(b *PageBatch) { b.Finishes[0].RecordIndex++ }, func(b *PageBatch) { b.Finishes[1].Scroll[1] = 10 }, func(b *PageBatch) { b.Sketches[0].History.Strokes[0].Pts[0][2] = .7 }, func(b *PageBatch) { b.Scene.RecordingID = "different" }} {
		b := pageFixture(t, 2)
		change(&b)
		if _, err := ValidatePageBatch(b); err == nil {
			t.Fatal("accepted forged finish/selection")
		}
	}
}
func TestFakePageRendererSharedContractAndSingleScreenshot(t *testing.T) {
	b := pageFixture(t, 8)
	fake := NewFakePageRenderer()
	RunPageRendererContract(t, fake, b)
	state := fake.Snapshot()
	if state.Opens != 1 || state.Closes != 1 || state.Requests != 1 || state.Active != 0 {
		t.Fatalf("not one joined page: %+v", state)
	}
	image, err := fake.RenderPage(context.Background(), b)
	if err != nil {
		t.Fatal(err)
	}
	if len(image.Coverage) != 8 || image.Coverage[7].Index != 8 || image.Coverage[0].PathIDs[0] != "p" {
		t.Fatalf("bad exact indexed coverage: %+v", image.Coverage)
	}
	for _, change := range []func(*PageImage){func(im *PageImage) { im.Coverage = im.Coverage[:7] }, func(im *PageImage) { im.Coverage[0].Index = 2 }, func(im *PageImage) { im.Coverage[0].Digest = "forged" }, func(im *PageImage) { im.Coverage[0].PathIDs = nil }, func(im *PageImage) { im.Plan.Width++ }, func(im *PageImage) { im.PNG = []byte("bad") }} {
		im, err := fake.RenderPage(context.Background(), b)
		if err != nil {
			t.Fatal(err)
		}
		change(&im)
		if err := ValidatePageImage(b, im); err == nil {
			t.Fatal("accepted incomplete/substituted page image")
		}
	}
}
func TestFakePageRendererCanceledWorkJoinsBeforeReplacement(t *testing.T) {
	b := pageFixture(t, 1)
	fake := NewFakePageRenderer()
	reached := make(chan struct{}, 1)
	gate := make(chan struct{})
	if err := fake.Configure(FakeControl{Stage: StageImage, Image: 0, Reached: reached, Gate: gate}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		im, err := fake.RenderPage(ctx, b)
		if !reflect.DeepEqual(im, PageImage{}) {
			done <- errors.New("partial page on cancellation")
			return
		}
		done <- err
	}()
	select {
	case <-reached:
	case <-time.After(time.Second):
		t.Fatal("page did not start")
	}
	if _, err := fake.RenderPage(context.Background(), b); err == nil {
		t.Fatal("concurrent page admitted")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("page cancellation did not join")
	}
	state := fake.Snapshot()
	if state.Active != 0 || state.Opens != state.Closes {
		t.Fatalf("retained page lifetime: %+v", state)
	}
}

func TestCurrentViewCanPinSameFinishWithExplicitWitness(t *testing.T) {
	b := pageFixture(t, 1)
	b.Current = &CurrentView{Projections: []PathProjection{{SketchID: b.Sketches[0].History.Begin.ID, PathID: "p", At: b.Sketches[0].History.Strokes[0].Begin.At, Clip: recording.Box{0, 0, 1280, 800}}}}
	if _, err := ValidatePageBatch(b); err != nil {
		t.Fatal(err)
	}
}

func currentPageFixture(t *testing.T) PageBatch {
	t.Helper()
	b := pageFixture(t, 1)
	b.Records = append(b.Records, &recording.Seg{Header: recording.Header{T: 20}, Seg: 2, Why: recording.SegNavigate, Tab: recording.Tab{ID: "tab1", URL: "https://current.test/"}, VP: b.Scene.VP}, &recording.Doc{Header: recording.Header{T: 21}, Doc: 100, URL: "https://current.test/", Tree: recording.Node{ID: 100, Y: recording.YDoc, C: []recording.Node{{ID: 101, Y: recording.YEl, N: "button"}}}})
	state := recording.NewState()
	for _, r := range b.Records {
		if err := state.Apply(r); err != nil {
			t.Fatal(err)
		}
	}
	b.Scene = PageScene{RecordingID: "renderer", SketchSourceFacts: pageFinishFacts(len(b.Records)-1, state)}
	box := recording.Box{30, 40, 100, 100}
	b.Current = &CurrentView{Projections: []PathProjection{{SketchID: b.Sketches[0].History.Begin.ID, PathID: "p", At: recording.NoteAt{Node: 101, Tag: "button", Box: &box}, Clip: recording.Box{0, 0, 1280, 800}}}}
	return b
}
func TestCurrentPagePinsLaterExactSceneAndPreservesOriginalHistory(t *testing.T) {
	b := currentPageFixture(t)
	before := b.Sketches[0].Source
	if _, err := ValidatePageBatch(b); err != nil {
		t.Fatal(err)
	}
	if b.Scene.Doc == before.Doc {
		t.Fatal("fixture did not move document")
	}
	b.Records = append(b.Records, &recording.Attr{Header: recording.Header{T: b.Scene.T}, ID: 101, Name: "style", V: recording.Str("display:none")})
	if _, err := ValidatePageBatch(b); err != nil {
		t.Fatalf("same-time later event widened prefix: %v", err)
	}
	if b.Sketches[0].Source != before {
		t.Fatal("source rewritten")
	}
	for _, change := range []func(*PageBatch){
		func(b *PageBatch) { b.Scene.RecordIndex-- }, func(b *PageBatch) { b.Scene.Doc++ }, func(b *PageBatch) { b.Current.Projections = nil },
		func(b *PageBatch) { b.Current.Projections = append(b.Current.Projections, b.Current.Projections[0]) },
		func(b *PageBatch) { b.Current.Projections[0].PathID = "missing" }, func(b *PageBatch) { b.Current.Projections[0].At.Node = 999 },
		func(b *PageBatch) { b.Current.Projections[0].At.Tag = "div" }, func(b *PageBatch) { b.Current.Projections[0].At.Box = nil },
		func(b *PageBatch) { b.Current.Projections[0].Clip = recording.Box{0, 0, 1, 1} },
		func(b *PageBatch) { b.Current.Projections[0].Clip = recording.Box{-1, 0, 1280, 800} },
		func(b *PageBatch) { b.Sketches[0].History.Label = "forged" }, func(b *PageBatch) { b.Finishes[0].Doc++ },
	} {
		bad := currentPageFixture(t)
		change(&bad)
		if _, err := ValidatePageBatch(bad); err == nil {
			t.Fatal("accepted forged current projection/history")
		}
	}
}

func FuzzCurrentViewProjectionBounds(f *testing.F) {
	f.Add(10.0, 20.0, 100.0, 100.0, 0.0, 0.0, 1280.0, 800.0)
	f.Add(-100.0, 20.0, 1000.0, 100.0, 0.0, 0.0, 1280.0, 800.0)
	f.Fuzz(func(t *testing.T, x, y, w, h, cx, cy, cw, ch float64) {
		b := pageFixture(t, 1)
		original, err := Plan(b.Sketches[0].History, b.Sketches[0].Source.VP, b.Sketches[0].Source.Scroll, DefaultLimits())
		if err != nil {
			t.Fatal(err)
		}
		box := recording.Box{x, y, w, h}
		current := CurrentView{Projections: []PathProjection{{SketchID: b.Sketches[0].History.Begin.ID, PathID: "p", At: recording.NoteAt{Node: 1, Box: &box}, Clip: recording.Box{cx, cy, cw, ch}}}}
		if ValidateCurrentView(b.Scene, current, map[string][]Path{b.Sketches[0].History.Begin.ID: original.Paths}) != nil {
			return
		}
		for _, point := range original.Paths[0].Pts {
			from := *original.Paths[0].At.Box
			px := x + (point[0]-from[0])/from[2]*w
			py := y + (point[1]-from[1])/from[3]*h
			if !finite(px) || !finite(py) || px < 0 || py < 0 || px > 1280 || py > 800 || px < cx || py < cy || px > cx+cw || py > cy+ch {
				t.Fatal("accepted unpaintable projected point")
			}
		}
	})
}

func TestCurrentPageRefusesHistoryClearedBeforePinnedScene(t *testing.T) {
	b := currentPageFixture(t)
	b.Records = append(b.Records, &recording.AnnotationClear{Header: recording.Header{T: 22}, Who: recording.Author{ID: "operator"}})
	state := recording.NewState()
	for _, r := range b.Records {
		if err := state.Apply(r); err != nil {
			t.Fatal(err)
		}
	}
	b.Scene.SketchSourceFacts = pageFinishFacts(len(b.Records)-1, state)
	if _, err := ValidatePageBatch(b); err == nil {
		t.Fatal("rendered history that was cleared before current visibility boundary")
	}
}

func TestCurrentFakeUsesSamePageRendererContract(t *testing.T) {
	batch := currentPageFixture(t)
	renderer := NewFakePageRenderer()
	RunPageRendererContract(t, renderer, batch)
	state := renderer.Snapshot()
	if state.Opens != 1 || state.Closes != 1 || state.Requests != 1 || state.Active != 0 {
		t.Fatalf("current context changed one-frame owned lifetime: %+v", state)
	}
}
