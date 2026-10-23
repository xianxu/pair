package sketchimage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/png"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/webcast/query"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func rendererBatch(t *testing.T, n int) Batch {
	t.Helper()
	b := recording.NewBuilder("renderer")
	node := b.El("button", []string{"id", "source"})
	b.Page(recording.SegLive, "https://source.test/", node)
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
	views, err := query.New(b.Recs, nil).SketchesAt(b.T)
	if err != nil {
		t.Fatal(err)
	}
	return Batch{RecordingDir: "/recording", Records: b.Recs, Sketches: views}
}
func TestBatchRejectsWrongSourceBoundaryAndVersion(t *testing.T) {
	for _, change := range []func(*Batch){func(b *Batch) { b.Sketches[0].Source.RecordIndex++ }, func(b *Batch) { b.Sketches[0].Source.URL = "https://current.test/" }, func(b *Batch) { b.Sketches[0].History.Label = "forged" }, func(b *Batch) { b.Sketches[0].History.Strokes[0].Pts[0][2] = .4 }, func(b *Batch) { b.Sketches[0].History.Receipts[4].Digest = strings.Repeat("b", 64) }} {
		b := rendererBatch(t, 1)
		change(&b)
		if _, err := ValidateBatch(b); err == nil {
			t.Fatal("accepted source/version not witnessed by input recording")
		}
	}
}

func TestBatchKeepsContextualSealAtIndividualHistoryCutoff(t *testing.T) {
	b := rendererBatch(t, 1)
	b.Records = b.Records[:len(b.Records)-2]
	b.Records = append(b.Records, &recording.Stop{Header: recording.Header{T: 13}, Why: recording.StopPause})
	views, err := query.New(b.Records, nil).SketchesAt(13)
	if err != nil {
		t.Fatal(err)
	}
	b.Sketches = views
	if !views[0].History.Strokes[0].Truncated {
		t.Fatal("fixture did not seal open stroke")
	}
	plans, err := ValidateBatch(b)
	if err != nil || len(plans) != 1 || !plans[0].Paths[0].Truncated {
		t.Fatalf("lost contextual history: %v %+v", err, plans)
	}
}
func TestValidateImagesRejectsSubstitutedOrUnboundedOutput(t *testing.T) {
	b := rendererBatch(t, 1)
	fake := NewFakeRenderer()
	images, err := fake.Render(context.Background(), b)
	if err != nil {
		t.Fatal(err)
	}
	plans, err := ValidateBatch(b)
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*Image){func(im *Image) { im.SketchID = "other" }, func(im *Image) { im.Plan.Scale = 2 }, func(im *Image) { im.PNG = []byte("not png") }, func(im *Image) { im.PNG = make([]byte, MaxImageBytes+1) }, func(im *Image) {
		var out bytes.Buffer
		png.Encode(&out, image.NewRGBA(image.Rect(0, 0, 1, 1)))
		im.PNG = out.Bytes()
	}, func(im *Image) { im.Limitations = make([]string, 33) }} {
		bad := images[0]
		change(&bad)
		if err := ValidateImages(plans, []Image{bad}); err == nil {
			t.Fatal("accepted substituted/unbounded image")
		}
	}
}
func TestFakeRendererWholeBatchLifecycle(t *testing.T) {
	b := rendererBatch(t, 2)
	fake := NewFakeRenderer()
	RunRendererContract(t, fake, b)
	st := fake.Snapshot()
	if st.Opens != 1 || st.Closes != 1 || st.Active != 0 || st.MaxActive != 1 || st.Requests != 2 {
		t.Fatalf("session leaked or per-image opens: %+v", st)
	}
	images, err := fake.Render(context.Background(), b)
	if err != nil || len(images) != 2 {
		t.Fatalf("second session: %v", err)
	}
	st = fake.Snapshot()
	if st.Opens != 2 || st.Closes != 2 {
		t.Fatalf("nonpersistent fake state: %+v", st)
	}
}
func TestFakeRendererFailureAndCancellationJoinBeforeReturn(t *testing.T) {
	b := rendererBatch(t, 2)
	for _, stage := range []RenderStage{StageOpen, StageImage, StageClose} {
		t.Run(string(stage), func(t *testing.T) {
			fake := NewFakeRenderer()
			failure := errors.New("controlled failure")
			if err := fake.Configure(FakeControl{Stage: stage, Image: 1, Err: failure}); err != nil {
				t.Fatal(err)
			}
			images, err := fake.Render(context.Background(), b)
			if !errors.Is(err, failure) || images != nil {
				t.Fatalf("partial success: %v %+v", err, images)
			}
			s := fake.Snapshot()
			if s.Active != 0 || s.Opens != s.Closes {
				t.Fatalf("failure returned before close: %+v", s)
			}
		})
	}
	fake := NewFakeRenderer()
	entered := make(chan struct{}, 1)
	gate := make(chan struct{})
	fake.Configure(FakeControl{Stage: StageImage, Image: 0, Reached: entered, Gate: gate})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { _, err := fake.Render(ctx, b); done <- err }()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("renderer did not reach held image")
	}
	if _, err := fake.Render(context.Background(), b); err == nil {
		t.Fatal("concurrent second session admitted")
	}
	if err := fake.Configure(FakeControl{}); err == nil {
		t.Fatal("rewrote active fake operation")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancel: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("cancellation did not join")
	}
	st := fake.Snapshot()
	if st.Active != 0 || st.Opens != st.Closes || st.MaxActive != 1 {
		t.Fatalf("cancellation returned with active session: %+v", st)
	}
}

func TestRendererAcceptsExactHistoryBeforeSameTimeClear(t *testing.T) {
	b := rendererBatch(t, 1)
	b.Records = append(b.Records, &recording.AnnotationClear{Header: recording.Header{T: 14}, Who: recording.Author{ID: "operator"}})
	selected, err := query.New(b.Records, nil).SketchesAt(14, query.SketchSelection{ID: "s0", NextSequence: 5, Digest: strings.Repeat("a", 64)})
	if err != nil {
		t.Fatal(err)
	}
	b.Sketches = selected
	if _, err := ValidateBatch(b); err != nil {
		t.Fatalf("selected prefix was replaced by later same-time clear: %v", err)
	}
	latest, err := query.New(b.Records, nil).SketchesAt(14)
	if err != nil || latest[0].History.Phase != "cleared" {
		t.Fatalf("latest query lost clear: %v %+v", err, latest)
	}
}
