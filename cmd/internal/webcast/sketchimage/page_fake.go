package sketchimage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
)

// FakePageRenderer reuses the stateful renderer's controlled lifecycle, with one
// image request per complete selected page rather than one per sketch.
type FakePageRenderer struct{ lifetime *FakeRenderer }

func NewFakePageRenderer() *FakePageRenderer              { return &FakePageRenderer{lifetime: NewFakeRenderer()} }
func (f *FakePageRenderer) Configure(c FakeControl) error { return f.lifetime.Configure(c) }
func (f *FakePageRenderer) Snapshot() FakeRenderState     { return f.lifetime.Snapshot() }
func (f *FakePageRenderer) RenderPage(ctx context.Context, b PageBatch) (out PageImage, err error) {
	if ctx == nil {
		return PageImage{}, fmt.Errorf("sketch page fake: nil context")
	}
	if err = ctx.Err(); err != nil {
		return out, err
	}
	plan, err := ValidatePageBatch(b)
	if err != nil {
		return out, err
	}
	owner := f.lifetime
	owner.mu.Lock()
	if owner.state.Active != 0 {
		owner.mu.Unlock()
		return out, fmt.Errorf("sketch page fake: session busy")
	}
	owner.state.Opens++
	session := owner.state.Opens
	owner.state.Active++
	owner.state.MaxActive = max(owner.state.MaxActive, owner.state.Active)
	control := owner.control
	owner.mu.Unlock()
	defer func() {
		closeErr := owner.step(ctx, control, session, StageClose, -1, "")
		owner.mu.Lock()
		owner.state.Closes++
		owner.state.Active--
		owner.mu.Unlock()
		err = errors.Join(err, closeErr)
		if err != nil {
			out = PageImage{}
		}
	}()
	if err = owner.step(ctx, control, session, StageOpen, -1, ""); err != nil {
		return out, err
	}
	if err = owner.step(ctx, control, session, StageImage, 0, "page"); err != nil {
		return out, err
	}
	pixels := image.NewRGBA(image.Rect(0, 0, plan.Width, plan.Height))
	draw.Draw(pixels, pixels.Bounds(), &image.Uniform{C: color.RGBA{240, 240, 240, 255}}, image.Point{}, draw.Src)
	var data bytes.Buffer
	if err = png.Encode(&data, pixels); err != nil {
		return out, err
	}
	out = PageImage{Plan: plan, PNG: data.Bytes(), Coverage: pageCoverage(b), Limitations: []string{"synthetic stateful-fake page; no recorded pixels"}}
	if err = validatePageResult(b, plan, out); err != nil {
		return PageImage{}, err
	}
	return out, nil
}
