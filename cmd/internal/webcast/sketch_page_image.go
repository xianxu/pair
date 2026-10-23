package webcast

import (
	"context"
	"encoding/base64"
	"errors"
	"github.com/xianxu/pair/cmd/internal/webcast/query"
	"github.com/xianxu/pair/cmd/internal/webcast/sketchimage"
	"sync"
)

// NewSketchPageImageRenderer reconstructs one exact finish scene for the whole
// selected group. It never connects to a live browser or rereads event files.
func NewSketchPageImageRenderer(chromeBin string) sketchimage.PageRenderer {
	return &sketchPageImageRenderer{chromeBin: chromeBin}
}

type sketchPageImageRenderer struct {
	chromeBin string
	active    sync.Mutex
}
type pageImageShape struct {
	Index  int                     `json:"index"`
	ID     string                  `json:"id"`
	Source query.SketchSourceFacts `json:"source"`
	Color  string                  `json:"color"`
	Paths  []sketchimage.Path      `json:"paths"`
}

func serveSketchPage(b sketchimage.PageBatch, p sketchimage.PagePlan) (*sketchImageServer, error) {
	shapes := make([]pageImageShape, len(b.Sketches))
	coverage := make([]sketchimage.PageCoverage, len(b.Sketches))
	for i, v := range b.Sketches {
		h := v.History
		// The same original path validator supplies vectors; its crop is unused.
		paths, err := sketchimage.Plan(h, v.Source.VP, v.Source.Scroll, sketchimage.DefaultLimits())
		if err != nil {
			return nil, err
		}
		shapes[i] = pageImageShape{i + 1, h.Begin.ID, v.Source, h.Begin.Author.Color, paths.Paths}
		coverage[i] = sketchimage.PageCoverage{Index: i + 1, SketchID: h.Begin.ID, NextSequence: h.NextSequence, Digest: h.Receipts[len(h.Receipts)-1].Digest, PathIDs: append([]string(nil), h.Visible...)}
	}
	payload := struct {
		Plan     sketchimage.PagePlan       `json:"plan"`
		Shapes   []pageImageShape           `json:"shapes"`
		Coverage []sketchimage.PageCoverage `json:"coverage"`
		Current  *sketchimage.CurrentView   `json:"current,omitempty"`
	}{p, shapes, coverage, b.Current}
	return serveHistoricalImageScenes(b.Batch, []historicalImageScene{{b.Scene.RecordIndex, payload, "sketch-page-image.js"}})
}
func (r *sketchPageImageRenderer) RenderPage(ctx context.Context, b sketchimage.PageBatch) (out sketchimage.PageImage, err error) {
	ctx, cancel := sketchImageContext(ctx)
	defer cancel()
	if err = ctx.Err(); err != nil {
		return out, err
	}
	if !r.active.TryLock() {
		return out, errors.New("sketch page: renderer already active")
	}
	defer r.active.Unlock()
	plan, err := sketchimage.ValidatePageBatch(b)
	if err != nil {
		return out, err
	}
	server, err := serveSketchPage(b, plan)
	if err != nil {
		return out, err
	}
	defer func() {
		if e := server.close(); e != nil {
			err = errors.Join(err, e)
			out = sketchimage.PageImage{}
		}
	}()
	browser, err := launchHeadless(ctx, r.chromeBin)
	if err != nil {
		return out, err
	}
	defer func() {
		if e := browser.closeJoined(); e != nil {
			err = errors.Join(err, e)
			out = sketchimage.PageImage{}
		}
	}()
	if err = browser.call(ctx, browser.session, "Emulation.setDeviceMetricsOverride", map[string]any{"width": plan.Width, "height": plan.Height, "deviceScaleFactor": 1, "mobile": false}, nil); err != nil {
		return out, err
	}
	if err = browser.openReady(ctx, server.base+"/0/sketch-image.html", "sketchPageImage"); err != nil {
		return out, err
	}
	var report struct {
		Error    string                     `json:"error"`
		Coverage []sketchimage.PageCoverage `json:"coverage"`
		Clipped  bool                       `json:"clipped"`
	}
	if err = browser.eval(ctx, "window.sketchPageImage.ready", &report); err != nil {
		return out, err
	}
	if report.Error != "" {
		return out, errors.New(report.Error)
	}
	var shot struct {
		Data string `json:"data"`
	}
	if err = browser.call(ctx, browser.session, "Page.captureScreenshot", map[string]any{"format": "png", "captureBeyondViewport": false, "clip": map[string]any{"x": 0, "y": 0, "width": plan.Width, "height": plan.Height, "scale": 1}}, &shot); err != nil {
		return out, err
	}
	if len(shot.Data) > base64.StdEncoding.EncodedLen(sketchimage.MaxImageBytes) {
		return out, errors.New("sketch page: PNG exceeds budget")
	}
	data, err := base64.StdEncoding.DecodeString(shot.Data)
	if err != nil {
		return out, err
	}
	limits := []string{"replay-rendered recorded DOM; uncaptured visual content and animation phase may differ"}
	if report.Clipped {
		limits = append(limits, "pressure fringe clipped to the recorded viewport or ancestor clip")
	}
	if server.cache.hasFailures() {
		limits = append(limits, "some recorded assets were unavailable or exceeded the bounded asset budget")
	}
	out = sketchimage.PageImage{Plan: plan, PNG: data, Limitations: limits, Clipped: report.Clipped, Coverage: report.Coverage}
	if err = sketchimage.ValidatePageImage(b, out); err != nil {
		return sketchimage.PageImage{}, err
	}
	return out, nil
}
