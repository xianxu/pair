package sketchimage

import (
	"math"
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func FuzzPlanBoundedTransform(f *testing.F) {
	f.Add(20.0, 20.0, 40.0, 40.0, .5, uint16(100), uint16(100), uint32(4096))
	f.Add(.5, 1.0, .5, 999.0, 1.0, uint16(1000), uint16(1000), uint32(0))
	f.Fuzz(func(t *testing.T, x0, y0, x1, y1, pressure float64, w, h uint16, budget uint32) {
		view := imageHistory()
		view.Strokes[0].Pts = []recording.SketchPoint{{x0, y0, pressure}, {x1, y1, pressure}}
		vp := recording.Viewport{W: 1 + int(w)%maxViewport, H: 1 + int(h)%maxViewport, DPR: 1}
		limits := Limits{MaxPixels: 1 + int(budget)%DefaultLimits().MaxPixels, Padding: 16}
		p, err := Plan(view, vp, [2]float64{}, limits)
		if err != nil {
			return
		}
		if p.Width < 1 || p.Height < 1 || p.Width > limits.MaxPixels/p.Height || !finite(p.Scale) || p.Scale <= 0 || p.Scale > 1 {
			t.Fatalf("unbounded output: %+v", p)
		}
		if p.Region[0] < 0 || p.Region[1] < 0 || p.Region[0]+p.Region[2] > float64(vp.W)+1e-8 || p.Region[1]+p.Region[3] > float64(vp.H)+1e-8 {
			t.Fatalf("output escaped source viewport: %+v", p)
		}
		if float64(p.Width)+1e-8 < p.Region[2]*p.Scale || float64(p.Height)+1e-8 < p.Region[3]*p.Scale {
			t.Fatalf("output clips its declared transform: %+v", p)
		}
		for _, n := range p.Region {
			if math.IsNaN(n) || math.IsInf(n, 0) {
				t.Fatal("nonfinite transform")
			}
		}
		if p.Paths[0].Pts[0] != (recording.SketchPoint{x0, y0, pressure}) || p.Paths[0].Pts[1] != (recording.SketchPoint{x1, y1, pressure}) {
			t.Fatal("plan changed original pressure vectors")
		}
	})
}
