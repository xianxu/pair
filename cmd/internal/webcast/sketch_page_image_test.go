package webcast

import (
	"bytes"
	"context"
	"github.com/xianxu/pair/cmd/internal/webcast/query"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
	"github.com/xianxu/pair/cmd/internal/webcast/sketchimage"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSketchPageRendererCanceledBeforeChrome(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	image, err := NewSketchPageImageRenderer("missing").RenderPage(ctx, sketchimage.PageBatch{})
	if err == nil || len(image.PNG) != 0 {
		t.Fatalf("canceled render: %v", err)
	}
}

func pageImageBatch(t *testing.T, n int) sketchimage.PageBatch {
	t.Helper()
	b := imageBatch(t, n)
	for _, r := range b.Records {
		if p, ok := r.(*recording.SketchPoints); ok {
			for j := range p.Pts {
				p.Pts[j][1] += float64((int(p.T)-12)/10) * 70
			}
		}
	}
	views, err := query.New(b.Records, nil).SketchesAt(1000)
	if err != nil {
		t.Fatal(err)
	}
	b.Sketches = views
	state := recording.NewState()
	finishes := make([]query.SketchSourceFacts, n)
	for index, r := range b.Records {
		if err := state.Apply(r); err != nil {
			t.Fatal(err)
		}
		for i, v := range views {
			if index == v.HistoryRecordIndex {
				f := query.SketchSourceFacts{T: r.Time(), RecordIndex: index, Doc: state.DocSeg, Tab: state.Tab.ID, URL: state.URL, Title: state.Title, VP: state.VP}
				if state.Top != nil {
					f.Scroll = state.Scrolls[state.Top.ID]
				}
				finishes[i] = f
			}
		}
	}
	return sketchimage.PageBatch{Batch: b, Scene: sketchimage.PageScene{RecordingID: "image", SketchSourceFacts: finishes[n-1]}, Finishes: finishes}
}
func TestSketchPageServerFreezesExactLatestFinish(t *testing.T) {
	b := pageImageBatch(t, 2)
	p, err := sketchimage.ValidatePageBatch(b)
	if err != nil {
		t.Fatal(err)
	}
	// A later same-time mutation cannot enter the scene or change its assets.
	b.Records = append(b.Records, &recording.Attr{Header: recording.Header{T: b.Scene.T}, ID: 1, Name: "id", V: recording.Str("after-finish")})
	s, err := serveSketchPage(b, p)
	if err != nil {
		t.Fatal(err)
	}
	defer s.close()
	res, err := http.Get(s.base + "/0/records")
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if bytes.Contains(raw, []byte("after-finish")) || !bytes.Contains(raw, []byte("Move this")) {
		t.Fatal("not exact finish prefix")
	}
	os.WriteFile(filepath.Join(b.RecordingDir, "events.jsonl"), []byte("replacement"), 0600)
	res, err = http.Get(s.base + "/0/records")
	if err != nil {
		t.Fatal(err)
	}
	again, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if !bytes.Equal(raw, again) {
		t.Fatal("reread events after server start")
	}
	res, err = http.Get(s.base + "/0/sketch-image.js")
	if err != nil {
		t.Fatal(err)
	}
	raw, _ = io.ReadAll(res.Body)
	res.Body.Close()
	if !bytes.Contains(raw, []byte("preparePageInk")) {
		t.Fatal("page entry not served")
	}
}
func TestLiveSketchPageImageHistoricalPressure(t *testing.T) {
	bin := liveFidelityChrome(t)
	b := pageImageBatch(t, 2)
	renderer := NewSketchPageImageRenderer(bin)
	sketchimage.RunPageRendererContract(t, renderer, b)
	out, err := renderer.RenderPage(context.Background(), b)
	if err != nil {
		t.Fatal(err)
	}
	im, err := png.Decode(bytes.NewReader(out.PNG))
	if err != nil {
		t.Fatal(err)
	}
	red := func(x, y int) bool { r, g, b, _ := im.At(x, y).RGBA(); return r > 50000 && g < 18000 && b < 18000 }
	if !red(148, 52) || red(48, 52) {
		t.Fatal("shared-page pressure was flattened or missing")
	}
	if !red(148, 122) {
		t.Fatal("second selected shape missing")
	}
}

func TestLiveSketchPageEightShapesAndHiddenPathRefusal(t *testing.T) {
	bin := liveFidelityChrome(t)
	b := pageImageBatch(t, 8)
	out, err := NewSketchPageImageRenderer(bin).RenderPage(context.Background(), b)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Coverage) != 8 {
		t.Fatal("selected badge coverage incomplete")
	}
	// Visibility is evaluated from the selected finish scene, not the begin.
	hidden := pageImageBatch(t, 1)
	for _, r := range hidden.Records {
		if a, ok := r.(*recording.Attr); ok {
			a.Name = "style"
			a.V = recording.Str("display:none")
		}
	}
	started := time.Now()
	out, err = NewSketchPageImageRenderer(bin).RenderPage(context.Background(), hidden)
	if err == nil || len(out.PNG) != 0 {
		t.Fatalf("hidden chosen path produced partialsuccess: %v", err)
	}
	if !strings.Contains(err.Error(), "index 1") || time.Since(started) > 5*time.Second {
		t.Fatalf("preflight did not fail directly with indexed diagnostic: %v after %s", err, time.Since(started))
	}
}

func TestLiveSketchPageRefusesMatchingReplacementOfRecordedAnchor(t *testing.T) {
	bin := liveFidelityChrome(t)
	b := pageImageBatch(t, 1)
	const text = "Distinct recorded anchor heading"
	var body recording.NodeID
	for _, r := range b.Records {
		if doc, ok := r.(*recording.Doc); ok {
			body = doc.Tree.C[1].C[1].ID
			original := &doc.Tree.C[1].C[1].C[0]
			original.C = []recording.Node{{ID: 10001, Y: recording.YText, V: recording.Str(text)}}
		}
		if stroke, ok := r.(*recording.SketchStrokeBegin); ok {
			stroke.At.Tag = "div"
			stroke.At.Text = text
		}
	}
	replacement := recording.Node{ID: 10002, Y: recording.YEl, N: "div", A: map[string]string{"style": "background:white;width:600px;height:400px"}, C: []recording.Node{{ID: 10003, Y: recording.YText, V: recording.Str(text)}}}
	// Replace the recorded node between the stroke and its completed label. The
	// different node has deliberately matching text/tag and visible geometry.
	last := len(b.Records) - 1
	records := append([]recording.Record(nil), b.Records[:last]...)
	records = append(records, &recording.Kids{Header: recording.Header{T: b.Scene.T - 1}, P: body, C: []recording.Node{replacement}}, b.Records[last])
	b.Records = records
	views, err := query.New(records, nil).SketchesAt(b.Scene.T)
	if err != nil {
		t.Fatal(err)
	}
	b.Sketches = views
	b.Finishes[0].RecordIndex = views[0].HistoryRecordIndex
	b.Scene.SketchSourceFacts = b.Finishes[0]
	if _, err := sketchimage.ValidatePageBatch(b); err != nil {
		t.Fatal(err)
	}
	out, err := NewSketchPageImageRenderer(bin).RenderPage(context.Background(), b)
	if err == nil || len(out.PNG) != 0 || !strings.Contains(err.Error(), "index 1") || !strings.Contains(err.Error(), "path p unavailable") {
		t.Fatalf("replacement silently moved historical ink: bytes=%d err=%v", len(out.PNG), err)
	}
}

func currentRenderBatch(t *testing.T) sketchimage.PageBatch {
	t.Helper()
	b := pageImageBatch(t, 1)
	current := recording.Node{ID: 1001, Y: recording.YEl, N: "div", A: map[string]string{"style": "position:absolute;left:120px;top:80px;width:600px;height:400px;background:white"}}
	b.Records = append(b.Records,
		&recording.Seg{Header: recording.Header{T: 18}, Seg: 2, Why: recording.SegTab, Tab: recording.Tab{ID: "other", URL: "https://away.test/"}, VP: b.Scene.VP},
		&recording.Doc{Header: recording.Header{T: 19}, Doc: 900, URL: "https://away.test/", Tree: recording.Node{ID: 900, Y: recording.YDoc, C: []recording.Node{{ID: 901, Y: recording.YEl, N: "body"}}}},
		&recording.Seg{Header: recording.Header{T: 20}, Seg: 3, Why: recording.SegTab, Tab: recording.Tab{ID: "tab1", URL: "https://source.test/"}, VP: b.Scene.VP}, &recording.Doc{Header: recording.Header{T: 21}, Doc: 1000, URL: "https://source.test/", Tree: recording.Node{ID: 1000, Y: recording.YDoc, C: []recording.Node{{ID: 1002, Y: recording.YEl, N: "html", C: []recording.Node{{ID: 1003, Y: recording.YEl, N: "head"}, {ID: 1004, Y: recording.YEl, N: "body", C: []recording.Node{current}}}}}}})
	state := recording.NewState()
	for _, r := range b.Records {
		if err := state.Apply(r); err != nil {
			t.Fatal(err)
		}
	}
	facts := query.SketchSourceFacts{T: state.T, RecordIndex: len(b.Records) - 1, Doc: state.DocSeg, Tab: state.Tab.ID, URL: state.URL, Title: state.Title, VP: state.VP}
	if state.Top != nil {
		facts.Scroll = state.Scrolls[state.Top.ID]
	}
	b.Scene = sketchimage.PageScene{RecordingID: "image", SketchSourceFacts: facts}
	box := recording.Box{120, 80, 600, 400}
	b.Current = &sketchimage.CurrentView{Projections: []sketchimage.PathProjection{{SketchID: b.Sketches[0].History.Begin.ID, PathID: "p", At: recording.NoteAt{Node: 1001, Tag: "div", Box: &box}, Clip: recording.Box{0, 0, 1280, 800}}}}
	return b
}
func TestLiveCurrentSketchPageReplaysPinnedViewAndFrozenProjection(t *testing.T) {
	bin := liveFidelityChrome(t)
	b := currentRenderBatch(t)
	before := b.Sketches[0].Source
	b.Records = append(b.Records, &recording.Attr{Header: recording.Header{T: b.Scene.T}, ID: 1001, Name: "style", V: recording.Str("display:none")})
	renderer := NewSketchPageImageRenderer(bin)
	sketchimage.RunPageRendererContract(t, renderer, b)
	out, err := renderer.RenderPage(context.Background(), b)
	if err != nil {
		t.Fatal(err)
	}
	im, err := png.Decode(bytes.NewReader(out.PNG))
	if err != nil {
		t.Fatal(err)
	}
	red := func(x, y int) bool { r, g, b, _ := im.At(x, y).RGBA(); return r > 50000 && g < 18000 && b < 18000 }
	if !red(260, 124) || red(160, 124) {
		t.Fatal("current-view pressure projection missing or flattened")
	}
	if b.Sketches[0].Source != before || b.Scene.Doc == before.Doc {
		t.Fatal("source was rewritten instead of projected")
	}
	bad := currentRenderBatch(t)
	bad.Current.Projections[0].At.Box[0] += 3
	out, err = renderer.RenderPage(context.Background(), bad)
	if err == nil || len(out.PNG) != 0 || !strings.Contains(err.Error(), "current geometry changed") {
		t.Fatalf("frozen geometry disagreement: %v", err)
	}
}
