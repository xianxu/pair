package sketchimage

import (
	"bytes"
	"context"
	"fmt"
	"image/png"
	"reflect"
	"slices"

	"github.com/xianxu/pair/cmd/internal/webcast/query"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

const (
	MaxBatchImages     = 8
	MaxImageBytes      = 8 << 20
	MaxBatchImageBytes = 64 << 20
)

// Batch owns the read-complete recording and selected histories. Renderers must
// not reread RecordingDir for events or consult the current browser document.
// Records are read-only; each source prefix ends at Source.RecordIndex inclusive.
type Batch struct {
	RecordingDir string
	Records      []recording.Record
	Sketches     []query.SketchView
}
type Image struct {
	SketchID    string
	Plan        RenderPlan
	PNG         []byte
	Limitations []string
}
type ImageRenderer interface {
	Render(context.Context, Batch) ([]Image, error)
}

// ValidateBatch uses one ordered reducer pass for all requested source and
// history indices. Contextual pause/interrupt effects do not advance sketch
// sequence, so the whole witnessed history is compared at its exact HistoryRecordIndex (timestamps can coincide).
func ValidateBatch(b Batch) ([]RenderPlan, error) {
	if err := validateBatchInput(b, nil); err != nil {
		return nil, err
	}
	plans := make([]RenderPlan, len(b.Sketches))
	for i, v := range b.Sketches {
		p, err := Plan(v.History, v.Source.VP, v.Source.Scroll, DefaultLimits())
		if err != nil {
			return nil, err
		}
		plans[i] = p
	}
	return plans, nil
}

func validateBatchInput(b Batch, observe func(int, *recording.State) error) error {
	if len(b.Sketches) == 0 || len(b.Sketches) > MaxBatchImages || len(b.Records) == 0 {
		return fmt.Errorf("sketch image: batch requires 1..%d sketches and recorded input", MaxBatchImages)
	}
	byID := map[string]int{}
	cutoffs := []int{}
	byIndex := map[int][]int{}
	maxIndex := 0
	for i, v := range b.Sketches {
		if v.History.Begin == nil || v.History.Begin.ID == "" || v.History.NextSequence == 0 || len(v.History.Receipts) == 0 || v.Source.RecordIndex < 0 || v.Source.RecordIndex >= len(b.Records) || v.Source.T < 0 || v.HistoryT < v.Source.T || v.HistoryRecordIndex < v.Source.RecordIndex || v.HistoryRecordIndex >= len(b.Records) {
			return fmt.Errorf("sketch image: invalid selected source/history")
		}
		id := v.History.Begin.ID
		if _, exists := byID[id]; exists {
			return fmt.Errorf("sketch image: duplicate selected sketch")
		}
		byID[id] = i
		if _, exists := byIndex[v.HistoryRecordIndex]; !exists {
			cutoffs = append(cutoffs, v.HistoryRecordIndex)
		}
		byIndex[v.HistoryRecordIndex] = append(byIndex[v.HistoryRecordIndex], i)
		maxIndex = max(maxIndex, v.HistoryRecordIndex)
	}
	slices.Sort(cutoffs)
	state := recording.NewState()
	next := 0
	sourceSeen := map[string]bool{}
	begins := map[string]bool{}
	checkHistory := func(at int) error {
		histories := state.Sketches.Snapshot().Sketches
		actual := make(map[string]recording.SketchView, len(histories))
		for _, h := range histories {
			actual[h.Begin.ID] = h
		}
		for _, i := range byIndex[at] {
			v := b.Sketches[i]
			h, exists := actual[v.History.Begin.ID]
			if !exists || !reflect.DeepEqual(h, v.History) || b.Records[at].Time() != v.HistoryT {
				return fmt.Errorf("sketch image: selected history differs from recording for %q", v.History.Begin.ID)
			}
		}
		return nil
	}
	last := -1
	for index, r := range b.Records {
		if r == nil {
			return fmt.Errorf("sketch image: nil recording event")
		}
		if index > maxIndex {
			break
		}
		for next < len(cutoffs) && cutoffs[next] < index {
			if err := checkHistory(cutoffs[next]); err != nil {
				return err
			}
			next++
		}
		if err := state.Apply(r); err != nil {
			return fmt.Errorf("sketch image: recording event %d: %w", index, err)
		}
		if observe != nil {
			if err := observe(index, state); err != nil {
				return err
			}
		}
		last = index
		if begin, ok := r.(*recording.SketchBegin); ok && !begins[begin.ID] {
			begins[begin.ID] = true
			if i, selected := byID[begin.ID]; selected {
				v := b.Sketches[i]
				source := query.SketchSourceFacts{T: begin.T, RecordIndex: index, Doc: begin.Doc, Tab: begin.Tab, URL: state.URL, Title: state.Title, VP: begin.VP}
				if state.Top != nil {
					source.Scroll = state.Scrolls[state.Top.ID]
				}
				if source != v.Source {
					return fmt.Errorf("sketch image: source boundary differs for %q", begin.ID)
				}
				sourceSeen[begin.ID] = true
			}
		}
	}
	if maxIndex > last {
		return fmt.Errorf("sketch image: history boundary exceeds witnessed input")
	}
	for next < len(cutoffs) {
		if err := checkHistory(cutoffs[next]); err != nil {
			return err
		}
		next++
	}
	if len(sourceSeen) != len(b.Sketches) {
		return fmt.Errorf("sketch image: source begin missing")
	}
	return nil
}

// ValidateImages is the common output contract. Call with plans returned by
// ValidateBatch; identity/order, immutable geometry, full PNG decoding and all
// image-size budgets are checked before any result is published.
func ValidateImages(plans []RenderPlan, images []Image) error {
	if len(plans) == 0 || len(plans) > MaxBatchImages || len(images) != len(plans) {
		return fmt.Errorf("sketch image: output count differs")
	}
	total := 0
	for i, im := range images {
		p := plans[i]
		if im.SketchID != p.Sketch || !reflect.DeepEqual(im.Plan, p) {
			return fmt.Errorf("sketch image: output identity or transform differs")
		}
		if p.Width < 1 || p.Height < 1 || p.Width > DefaultLimits().MaxPixels/p.Height {
			return fmt.Errorf("sketch image: output dimensions exceed pixel limit")
		}
		if len(im.PNG) == 0 || len(im.PNG) > MaxImageBytes || len(im.PNG) > MaxBatchImageBytes-total {
			return fmt.Errorf("sketch image: PNG byte budget exceeded")
		}
		total += len(im.PNG)
		cfg, err := png.DecodeConfig(bytes.NewReader(im.PNG))
		if err != nil || cfg.Width != p.Width || cfg.Height != p.Height {
			return fmt.Errorf("sketch image: PNG dimensions differ from transform")
		}
		decoded, err := png.Decode(bytes.NewReader(im.PNG))
		if err != nil || decoded.Bounds().Dx() != p.Width || decoded.Bounds().Dy() != p.Height {
			return fmt.Errorf("sketch image: malformed PNG")
		}
		if len(im.Limitations) > 32 {
			return fmt.Errorf("sketch image: too many limitations")
		}
		for _, s := range im.Limitations {
			if len(s) > 256 {
				return fmt.Errorf("sketch image: limitation exceeds bound")
			}
		}
	}
	return nil
}

// RunRendererContract is shared by the stateful fake and real integration.
// Backends must reject substituted source and cancellation, and return only
// complete bounded images; backend-specific tests additionally observe cleanup.
func RunRendererContract(t recording.TB, renderer ImageRenderer, b Batch) {
	t.Helper()
	plans, err := ValidateBatch(b)
	if err != nil {
		t.Fatalf("contract fixture: %v", err)
	}
	images, err := renderer.Render(context.Background(), b)
	if err != nil {
		t.Fatalf("renderer: %v", err)
	}
	if err = ValidateImages(plans, images); err != nil {
		t.Fatalf("renderer output: %v", err)
	}
	bad := b
	bad.Sketches = append([]query.SketchView(nil), b.Sketches...)
	bad.Sketches[0].Source.RecordIndex++
	if _, err = renderer.Render(context.Background(), bad); err == nil {
		t.Fatalf("renderer accepted a different source boundary")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err = renderer.Render(ctx, b); err == nil {
		t.Fatalf("renderer ignored canceled lifetime")
	}
}
