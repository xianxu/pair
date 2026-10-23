package sketchimage

import (
	"bytes"
	"context"
	"fmt"
	"image/png"
	"math"
	"reflect"
	"slices"

	"github.com/xianxu/pair/cmd/internal/webcast/query"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// PageScene is an exact recorded finish boundary, never the browser's current
// page. Ordinals distinguish later changes with an identical timestamp.
type PageScene struct {
	RecordingID string `json:"recording_id"`
	query.SketchSourceFacts
}

// SamePageView deliberately excludes title/time/ordinal, which may change while
// the selected shapes still describe the same recorded visible viewport.
func SamePageView(a, b PageScene) bool {
	return a.RecordingID == b.RecordingID && a.Doc == b.Doc && a.Tab == b.Tab && a.URL == b.URL && a.VP == b.VP && a.Scroll == b.Scroll
}

type PagePlan struct {
	Scene  PageScene     `json:"scene"`
	Region recording.Box `json:"region"`
	Scale  float64       `json:"scale"`
	Width  int           `json:"width"`
	Height int           `json:"height"`
}

// PathProjection binds one original accepted path to its trusted current
// recorded anchor and effective visible clip; it never rewrites source vectors.
type PathProjection struct {
	SketchID string           `json:"sketch_id"`
	PathID   string           `json:"path_id"`
	At       recording.NoteAt `json:"at"`
	Clip     recording.Box    `json:"clip"`
}
type CurrentView struct {
	Projections []PathProjection `json:"projections"`
}
type PageBatch struct {
	Batch
	Scene    PageScene
	Finishes []query.SketchSourceFacts
	Current  *CurrentView
}
type PageCoverage struct {
	Index        int      `json:"index"`
	SketchID     string   `json:"sketch_id"`
	NextSequence uint64   `json:"next_sequence"`
	Digest       string   `json:"digest"`
	PathIDs      []string `json:"path_ids"`
}
type PageImage struct {
	Plan        PagePlan
	PNG         []byte
	Limitations []string
	Clipped     bool
	Coverage    []PageCoverage
}
type PageRenderer interface {
	RenderPage(context.Context, PageBatch) (PageImage, error)
}

// PlanPage is pure viewport geometry. Original pressure paths live only in the
// selected histories/items; this plan does not duplicate or rewrite them.
func PlanPage(scene PageScene) (PagePlan, error) {
	if scene.RecordingID == "" || scene.Doc < 1 || scene.Tab == "" || scene.T < 0 || uint64(scene.T) > recording.MaxSketchInteger || scene.RecordIndex < 0 || scene.VP.W < 1 || scene.VP.H < 1 || scene.VP.W > maxViewport || scene.VP.H > maxViewport || !finite(scene.VP.DPR) || scene.VP.DPR <= 0 || !finite(scene.Scroll[0]) || !finite(scene.Scroll[1]) || math.Abs(scene.Scroll[0]) > recording.SketchMaxCoordinate || math.Abs(scene.Scroll[1]) > recording.SketchMaxCoordinate {
		return PagePlan{}, fmt.Errorf("sketch page: invalid finish viewport")
	}
	p := PagePlan{Scene: scene, Region: recording.Box{scene.Scroll[0], scene.Scroll[1], float64(scene.VP.W), float64(scene.VP.H)}}
	p.Scale, p.Width, p.Height = fitPagePixels(p.Region[2], p.Region[3], DefaultLimits().MaxPixels)
	return p, nil
}
func fitPagePixels(width, height float64, maxPixels int) (scale float64, w, h int) {
	scale = math.Min(1, math.Sqrt(float64(maxPixels)/(width*height)))
	size := func(s float64) (int, int) { return max(1, int(math.Ceil(width*s))), max(1, int(math.Ceil(height*s))) }
	w, h = size(scale)
	if w > maxPixels/h {
		low, high := 0.0, scale
		for range 60 {
			mid := (low + high) / 2
			mw, mh := size(mid)
			if mw <= maxPixels/mh {
				low = mid
			} else {
				high = mid
			}
		}
		scale = low
		w, h = size(scale)
	}
	return
}
func pageFinishFacts(index int, state *recording.State) query.SketchSourceFacts {
	f := query.SketchSourceFacts{T: state.T, RecordIndex: index, Doc: state.DocSeg, Tab: state.Tab.ID, URL: state.URL, Title: state.Title, VP: state.VP}
	if state.Top != nil {
		f.Scroll = state.Scrolls[state.Top.ID]
	}
	return f
}

// ValidatePageBatch shares the legacy ordered witness pass. All supplied BEGIN,
// selected history and completion facts are verified against the same reducer.
func ValidatePageBatch(b PageBatch) (PagePlan, error) {
	plan, err := PlanPage(b.Scene)
	if err != nil {
		return PagePlan{}, err
	}
	if len(b.Finishes) != len(b.Sketches) || len(b.Sketches) == 0 {
		return PagePlan{}, fmt.Errorf("sketch page: missing finish witnesses")
	}
	byIndex := map[int][]int{}
	latest := -1
	for i, v := range b.Sketches {
		f := b.Finishes[i]
		if v.History.Begin == nil || v.History.Label == "" || len(v.History.Visible) == 0 || (v.History.Phase != "complete" && v.History.Phase != "interrupted") || f.RecordIndex != v.HistoryRecordIndex || f.T != v.HistoryT || f.Doc != v.Source.Doc || f.Tab != v.Source.Tab {
			return PagePlan{}, fmt.Errorf("sketch page: invalid completion witness at index %d", i+1)
		}
		if b.Current == nil && !SamePageView(b.Scene, PageScene{RecordingID: b.Scene.RecordingID, SketchSourceFacts: f}) {
			return PagePlan{}, fmt.Errorf("sketch page: incompatible finish view at index %d", i+1)
		}
		byIndex[f.RecordIndex] = append(byIndex[f.RecordIndex], i)
		latest = max(latest, f.RecordIndex)
	}
	if b.Current == nil && latest != b.Scene.RecordIndex {
		return PagePlan{}, fmt.Errorf("sketch page: scene is not latest selected finish")
	}
	if b.Current != nil && b.Scene.RecordIndex < latest {
		return PagePlan{}, fmt.Errorf("sketch page: current scene precedes selected history")
	}
	seen := 0
	sceneSeen := false
	var through []int
	if b.Current != nil {
		through = []int{b.Scene.RecordIndex}
	}
	err = validateBatchInput(b.Batch, func(index int, state *recording.State) error {
		if index == 0 && (state.Head == nil || state.Head.ID != b.Scene.RecordingID) {
			return fmt.Errorf("sketch page: recording identity differs")
		}
		for _, i := range byIndex[index] {
			if state.Top == nil || pageFinishFacts(index, state) != b.Finishes[i] {
				return fmt.Errorf("sketch page: finish boundary differs at index %d", i+1)
			}
			if b.Current == nil && index == latest && b.Scene.SketchSourceFacts != b.Finishes[i] {
				return fmt.Errorf("sketch page: latest scene facts differ")
			}
			seen++
		}
		if b.Current != nil && index == b.Scene.RecordIndex {
			if state.Top == nil || pageFinishFacts(index, state) != b.Scene.SketchSourceFacts {
				return fmt.Errorf("sketch page: current scene facts differ")
			}
			for _, selected := range b.Sketches {
				history := selected.History
				metadata, ok := state.Sketches.Metadata(history.Begin.ID)
				if !ok || metadata.NextSequence != history.NextSequence || metadata.Digest != history.Receipts[len(history.Receipts)-1].Digest || metadata.Phase != history.Phase || metadata.Label != history.Label || metadata.Reason != history.Reason {
					return fmt.Errorf("sketch page: selected version changed before current scene")
				}
			}
			for _, p := range b.Current.Projections {
				node := state.Node(p.At.Node)
				if node == nil || node.Y != recording.YEl || p.At.Tag != "" && p.At.Tag != node.N {
					return fmt.Errorf("sketch page: current anchor unavailable")
				}
				for key, value := range p.At.Stable {
					if node.A[key] != value {
						return fmt.Errorf("sketch page: current anchor conflicts")
					}
				}
				for depth := 0; ; depth++ {
					if node == state.Top {
						break
					}
					if depth >= 256 || node.Y == recording.YDoc {
						return fmt.Errorf("sketch page: unsupported current frame scope")
					}
					if node.Parent != nil {
						node = node.Parent
					} else if node.Y == recording.YShadow && node.Host != nil {
						node = node.Host
					} else {
						return fmt.Errorf("sketch page: detached current anchor")
					}
				}
			}
			sceneSeen = true
		}
		return nil
	}, through...)
	if err != nil {
		return PagePlan{}, err
	}
	if seen != len(b.Finishes) {
		return PagePlan{}, fmt.Errorf("sketch page: finish boundary not witnessed")
	}
	if b.Current != nil {
		if !sceneSeen {
			return PagePlan{}, fmt.Errorf("sketch page: current boundary not witnessed")
		}
		originals := map[string][]Path{}
		for _, v := range b.Sketches {
			paths, err := Plan(v.History, v.Source.VP, v.Source.Scroll, DefaultLimits())
			if err != nil {
				return PagePlan{}, err
			}
			originals[v.History.Begin.ID] = paths.Paths
		}
		if err := ValidateCurrentView(b.Scene, *b.Current, originals); err != nil {
			return PagePlan{}, err
		}
	}
	return plan, nil
}

// ValidateCurrentView validates complete current placement coverage and projected
// centerlines. Browser-measured box/clip agreement is checked by the adapter.
func ValidateCurrentView(scene PageScene, current CurrentView, originals map[string][]Path) error {
	plan, err := PlanPage(scene)
	if err != nil {
		return err
	}
	expected := map[string]Path{}
	for sketch, paths := range originals {
		for _, p := range paths {
			key := sketch + "\x00" + p.ID
			if _, exists := expected[key]; exists {
				return fmt.Errorf("sketch page: repeated original path")
			}
			expected[key] = p
		}
	}
	if len(expected) == 0 || len(expected) > MaxBatchImages*recording.SketchMaxStrokes || len(current.Projections) != len(expected) {
		return fmt.Errorf("sketch page: incomplete current projection coverage")
	}
	seen := map[string]bool{}
	validBox := func(box recording.Box) bool {
		for _, v := range box {
			if !finite(v) || math.Abs(v) > recording.SketchMaxCoordinate {
				return false
			}
		}
		return box[2] > 0 && box[3] > 0
	}
	inside := func(x, y float64, b recording.Box) bool {
		return x >= b[0] && y >= b[1] && x <= b[0]+b[2] && y <= b[1]+b[3]
	}
	for _, p := range current.Projections {
		key := p.SketchID + "\x00" + p.PathID
		path, ok := expected[key]
		if !ok || seen[key] || p.At.Node <= 0 || uint64(p.At.Node) > recording.MaxSketchInteger || p.At.Box == nil || !validBox(*p.At.Box) || !validBox(p.Clip) || p.At.Kind != "" && p.At.Kind != recording.AnchorElement || p.At.Route != "" && p.At.Route != scene.URL {
			return fmt.Errorf("sketch page: invalid current projection %q/%q", p.SketchID, p.PathID)
		}
		if !inside(p.Clip[0], p.Clip[1], plan.Region) || !inside(p.Clip[0]+p.Clip[2], p.Clip[1]+p.Clip[3], plan.Region) {
			return fmt.Errorf("sketch page: current clip outside viewport")
		}
		if path.At.Box == nil || !validBox(*path.At.Box) {
			return fmt.Errorf("sketch page: invalid original anchor")
		}
		from, to := *path.At.Box, *p.At.Box
		for _, point := range path.Pts {
			x := to[0] + (point[0]-from[0])/from[2]*to[2]
			y := to[1] + (point[1]-from[1])/from[3]*to[3]
			if !finite(x) || !finite(y) || !inside(x, y, p.Clip) || !inside(x, y, plan.Region) {
				return fmt.Errorf("sketch page: projected path outside current clip")
			}
		}
		seen[key] = true
	}
	return nil
}
func pageCoverage(b PageBatch) []PageCoverage {
	out := make([]PageCoverage, len(b.Sketches))
	for i, v := range b.Sketches {
		h := v.History
		out[i] = PageCoverage{Index: i + 1, SketchID: h.Begin.ID, NextSequence: h.NextSequence, Digest: h.Receipts[len(h.Receipts)-1].Digest, PathIDs: slices.Clone(h.Visible)}
	}
	return out
}
func ValidatePageImage(b PageBatch, im PageImage) error {
	plan, err := ValidatePageBatch(b)
	if err != nil {
		return err
	}
	return validatePageResult(b, plan, im)
}
func validatePageResult(b PageBatch, plan PagePlan, im PageImage) error {
	if !reflect.DeepEqual(plan, im.Plan) || !reflect.DeepEqual(pageCoverage(b), im.Coverage) {
		return fmt.Errorf("sketch page: incomplete or substituted indexed coverage/scene")
	}
	if len(im.PNG) == 0 || len(im.PNG) > MaxImageBytes {
		return fmt.Errorf("sketch page: PNG exceeds image byte budget")
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(im.PNG))
	if err != nil || cfg.Width != plan.Width || cfg.Height != plan.Height {
		return fmt.Errorf("sketch page: PNG dimensions differ from viewport plan")
	}
	if _, err := png.Decode(bytes.NewReader(im.PNG)); err != nil {
		return fmt.Errorf("sketch page: malformed PNG: %w", err)
	}
	if len(im.Limitations) > 32 {
		return fmt.Errorf("sketch page: too many limitations")
	}
	for _, limitation := range im.Limitations {
		if len(limitation) > 256 {
			return fmt.Errorf("sketch page: limitation exceeds bound")
		}
	}
	if im.Clipped && len(im.Limitations) == 0 {
		return fmt.Errorf("sketch page: natural fringe clipping must be explicit")
	}
	return nil
}
func RunPageRendererContract(t recording.TB, r PageRenderer, b PageBatch) {
	t.Helper()
	im, err := r.RenderPage(context.Background(), b)
	if err != nil {
		t.Fatalf("page renderer: %v", err)
	}
	if err := ValidatePageImage(b, im); err != nil {
		t.Fatalf("page renderer output: %v", err)
	}
	bad := b
	bad.Scene.RecordIndex++
	if _, err = r.RenderPage(context.Background(), bad); err == nil {
		t.Fatalf("page renderer accepted another finish boundary")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err = r.RenderPage(ctx, b); err == nil {
		t.Fatalf("page renderer ignored cancellation")
	}
}
