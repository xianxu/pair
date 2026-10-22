// Package sketchimage plans bounded historical sketch images without changing
// the original vectors. Browser rendering is an injected integration boundary.
package sketchimage

import (
	"fmt"
	"maps"
	"math"
	"slices"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

const maxViewport = 16384

type Limits struct {
	MaxPixels int
	Padding   float64
}

func DefaultLimits() Limits { return Limits{MaxPixels: 4 * 1024 * 1024, Padding: 16} }

// Path owns original page coordinates and pressure, independently of the crop.
type Path struct {
	ID        string                  `json:"id"`
	At        recording.NoteAt        `json:"at"`
	Width     float64                 `json:"width"`
	Pts       []recording.SketchPoint `json:"pts"`
	Truncated bool                    `json:"truncated"`
}

// RenderPlan maps original page coordinates to image pixels:
// (pagePoint - Region.origin) * Scale. DPR never alters stored pressure/width.
type RenderPlan struct {
	Sketch  string        `json:"sketch"`
	Region  recording.Box `json:"region"`
	Scale   float64       `json:"scale"`
	Width   int           `json:"width"`
	Height  int           `json:"height"`
	Clipped bool          `json:"clipped"`
	Paths   []Path        `json:"paths"`
}

func finite(n float64) bool { return !math.IsNaN(n) && !math.IsInf(n, 0) }

// Plan is linear in the selected history's strokes and visible points. It
// refuses missing/ambiguous paths instead of rendering a different history.
func Plan(h recording.SketchView, vp recording.Viewport, scroll [2]float64, limits Limits) (RenderPlan, error) {
	fail := func(why string) (RenderPlan, error) {
		return RenderPlan{}, fmt.Errorf("sketch image: %s", why)
	}
	if h.Begin == nil || h.Begin.ID == "" || h.Phase == "cleared" || len(h.Visible) == 0 || len(h.Visible) > recording.SketchMaxStrokes || len(h.Strokes) > recording.SketchMaxStrokes {
		return fail("no visible sketch")
	}
	if vp.W < 1 || vp.H < 1 || vp.W > maxViewport || vp.H > maxViewport || !finite(vp.DPR) || vp.DPR <= 0 ||
		!finite(scroll[0]) || !finite(scroll[1]) || math.Abs(scroll[0]) > recording.SketchMaxCoordinate || math.Abs(scroll[1]) > recording.SketchMaxCoordinate {
		return fail("invalid source viewport")
	}
	if limits.MaxPixels < 1 || limits.MaxPixels > DefaultLimits().MaxPixels || !finite(limits.Padding) || limits.Padding < 0 || limits.Padding > 1024 {
		return fail("invalid image limits")
	}
	strokes := make(map[string]recording.SketchStrokeView, len(h.Strokes))
	for _, s := range h.Strokes {
		if s.Begin == nil || s.Begin.ID == "" || s.Begin.Sketch != h.Begin.ID {
			return fail("invalid stroke identity")
		}
		if _, exists := strokes[s.Begin.ID]; exists {
			return fail("duplicate stroke identity")
		}
		strokes[s.Begin.ID] = s
	}
	p := RenderPlan{Sketch: h.Begin.ID}
	seen := make(map[string]bool, len(h.Visible))
	x0, y0, x1, y1 := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	points := 0
	for _, id := range h.Visible {
		s, ok := strokes[id]
		if !ok || seen[id] {
			return fail("missing or repeated visible stroke")
		}
		seen[id] = true
		b := s.Begin
		if len(s.Pts) == 0 || len(s.Pts) > recording.SketchMaxPointsPerStroke || !finite(b.Width) || b.Width <= 0 || b.Width > recording.SketchMaxCoordinate {
			return fail("invalid pressure path")
		}
		points += len(s.Pts)
		if points > recording.SketchMaxPointsPerSketch {
			return fail("too many pressure points")
		}
		at := b.At
		if at.Box == nil || at.Node <= 0 {
			return fail("missing original anchor")
		}
		box := *at.Box
		for _, n := range box {
			if !finite(n) || math.Abs(n) > recording.SketchMaxCoordinate {
				return fail("invalid original anchor")
			}
		}
		if box[2] <= 0 || box[3] <= 0 {
			return fail("invalid original anchor size")
		}
		at.Box = &box
		at.Classes = slices.Clone(at.Classes)
		at.Path = slices.Clone(at.Path)
		at.Stable = maps.Clone(at.Stable)
		path := Path{ID: id, At: at, Width: b.Width, Pts: append([]recording.SketchPoint(nil), s.Pts...), Truncated: s.Truncated}
		for _, point := range path.Pts {
			if !finite(point[0]) || !finite(point[1]) || math.Abs(point[0]) > recording.SketchMaxCoordinate || math.Abs(point[1]) > recording.SketchMaxCoordinate || !finite(point[2]) || point[2] < 0 || point[2] > 1 {
				return fail("invalid pressure point")
			}
			// Use the maximum pen radius for conservative containment of
			// interpolated widths, including the single-point dot.
			radius := b.Width / 2
			x0, y0 = math.Min(x0, point[0]-radius), math.Min(y0, point[1]-radius)
			x1, y1 = math.Max(x1, point[0]+radius), math.Max(y1, point[1]+radius)
		}
		p.Paths = append(p.Paths, path)
	}
	x0, y0, x1, y1 = x0-limits.Padding, y0-limits.Padding, x1+limits.Padding, y1+limits.Padding
	left, top := math.Max(x0, scroll[0]), math.Max(y0, scroll[1])
	right, bottom := math.Min(x1, scroll[0]+float64(vp.W)), math.Min(y1, scroll[1]+float64(vp.H))
	if right <= left || bottom <= top {
		return fail("sketch is outside the recorded viewport")
	}
	p.Clipped = left != x0 || top != y0 || right != x1 || bottom != y1
	p.Region = recording.Box{left, top, right - left, bottom - top}
	p.Scale = math.Min(1, math.Sqrt(float64(limits.MaxPixels)/(p.Region[2]*p.Region[3])))
	size := func(scale float64) (int, int) {
		return max(1, int(math.Ceil(p.Region[2]*scale))), max(1, int(math.Ceil(p.Region[3]*scale)))
	}
	p.Width, p.Height = size(p.Scale)
	if p.Width > limits.MaxPixels/p.Height {
		// Ceiling each side to a whole pixel can exceed the area estimate,
		// especially for narrow crops. Fit the shared scale, never merely
		// shrink dimensions while keeping an incompatible transform.
		low, high := 0.0, p.Scale
		for range 60 {
			mid := (low + high) / 2
			w, h := size(mid)
			if w <= limits.MaxPixels/h {
				low = mid
			} else {
				high = mid
			}
		}
		p.Scale = low
		p.Width, p.Height = size(low)
	}
	return p, nil
}
