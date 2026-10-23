package sketchimage

import (
	"math"
	"reflect"
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func imageHistory() recording.SketchView {
	return recording.SketchView{
		Begin: &recording.SketchBegin{ID: "shape"}, Phase: "complete", Label: "Change this",
		Visible: []string{"ink"},
		Strokes: []recording.SketchStrokeView{{
			Begin: &recording.SketchStrokeBegin{Sketch: "shape", ID: "ink", Width: 3, At: recording.NoteAt{Node: 4, Box: &recording.Box{10, 10, 50, 50}}},
			Pts:   []recording.SketchPoint{{20, 20, 0}, {40, 40, 1}}, Sealed: true,
		}},
	}
}

func TestPlanKeepsOriginalPressureAndOwnsPaths(t *testing.T) {
	h := imageHistory()
	h.Strokes[0].Begin.At.Classes = []string{"source"}
	h.Strokes[0].Begin.At.Stable = map[string]string{"data-room": "one"}
	h.Strokes[0].Begin.At.Path = []string{"div", "p"}
	original := append([]recording.SketchPoint(nil), h.Strokes[0].Pts...)
	p, err := Plan(h, recording.Viewport{W: 100, H: 100, DPR: 2}, [2]float64{}, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if p.Sketch != "shape" || len(p.Paths) != 1 || !reflect.DeepEqual(p.Paths[0].Pts, original) {
		t.Fatalf("original path/pressure changed: %+v", p)
	}
	// Both endpoints, including their maximum pressure radius, fit in the
	// historical page crop, and the crop includes the requested context pad.
	if p.Region[0] > 4 || p.Region[1] > 4 || p.Region[0]+p.Region[2] < 56 || p.Region[1]+p.Region[3] < 56 {
		t.Fatalf("crop lost ink context: %v", p.Region)
	}
	if p.Width*p.Height > DefaultLimits().MaxPixels || p.Scale != 1 || p.Clipped {
		t.Fatalf("unexpected transform: %+v", p)
	}
	p.Paths[0].Pts[0][0] = 999
	p.Paths[0].At.Box[0] = 999
	p.Paths[0].At.Classes[0] = "changed"
	p.Paths[0].At.Path[0] = "changed"
	p.Paths[0].At.Stable["data-room"] = "changed"
	if h.Strokes[0].Pts[0][0] != 20 || (*h.Strokes[0].Begin.At.Box)[0] != 10 {
		t.Fatal("render plan aliases recording input")
	}
	if h.Strokes[0].Begin.At.Classes[0] != "source" || h.Strokes[0].Begin.At.Path[0] != "div" || h.Strokes[0].Begin.At.Stable["data-room"] != "one" {
		t.Fatal("render plan aliases original anchor metadata")
	}
}

func TestPlanPixelCapPreservesTransformForNarrowCrops(t *testing.T) {
	h := imageHistory()
	h.Strokes[0].Begin.Width = .000001
	h.Strokes[0].Pts = []recording.SketchPoint{{.5, 1, 0}, {.5, 999, 1}}
	p, err := Plan(h, recording.Viewport{W: 1000, H: 1000, DPR: 1}, [2]float64{}, Limits{MaxPixels: 1})
	if err != nil {
		t.Fatal(err)
	}
	if p.Width != 1 || p.Height != 1 || float64(p.Height)+1e-9 < p.Region[3]*p.Scale || float64(p.Width)+1e-9 < p.Region[2]*p.Scale {
		t.Fatalf("pixel cap changed dimensions without a matching transform: %+v", p)
	}
}

func TestPlanClipsOutputWithoutClippingStoredVectors(t *testing.T) {
	h := imageHistory()
	h.Strokes[0].Pts = []recording.SketchPoint{{-100, -100, .5}, {10000, 10000, 1}}
	p, err := Plan(h, recording.Viewport{W: 5000, H: 5000, DPR: 1}, [2]float64{10, 20}, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if !p.Clipped || p.Region != (recording.Box{10, 20, 5000, 5000}) || p.Width*p.Height > 4*1024*1024 || p.Scale >= 1 {
		t.Fatalf("unbounded or wrong clip/scale: %+v", p)
	}
	if !reflect.DeepEqual(p.Paths[0].Pts, h.Strokes[0].Pts) {
		t.Fatal("output clipping rewrote source vectors")
	}
}

func TestPlanRejectsUnrenderableEvidence(t *testing.T) {
	for _, mutate := range []func(*recording.SketchView){
		func(h *recording.SketchView) { h.Visible = nil },
		func(h *recording.SketchView) { h.Visible = []string{"absent"} },
		func(h *recording.SketchView) { h.Visible = []string{"ink", "ink"} },
		func(h *recording.SketchView) { h.Strokes[0].Pts = nil },
		func(h *recording.SketchView) { h.Strokes[0].Pts[0][0] = math.NaN() },
		func(h *recording.SketchView) { h.Strokes[0].Pts[0][2] = 2 },
		func(h *recording.SketchView) { h.Strokes[0].Begin.Width = math.Inf(1) },
		func(h *recording.SketchView) { h.Strokes[0].Begin.Sketch = "different" },
	} {
		h := imageHistory()
		mutate(&h)
		if _, err := Plan(h, recording.Viewport{W: 100, H: 100, DPR: 1}, [2]float64{}, DefaultLimits()); err == nil {
			t.Fatalf("unrenderable evidence admitted: %+v", h)
		}
	}
}
