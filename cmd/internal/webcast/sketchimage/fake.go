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
	"slices"
	"sync"
)

type RenderStage string

const (
	StageOpen  RenderStage = "open"
	StageImage RenderStage = "image"
	StageClose RenderStage = "close"
)

// FakeControl holds or fails one deterministic stage. Image is zero-based and
// applies only to StageImage; configuration remains until explicitly replaced.
// Reached and Gate expose ordering without function-call mocks or sleeps.
type FakeControl struct {
	Stage   RenderStage
	Image   int
	Err     error
	Reached chan<- struct{}
	Gate    <-chan struct{}
}
type FakeRenderEvent struct {
	Session  int
	Stage    RenderStage
	Image    int
	SketchID string
}
type FakeRenderState struct {
	Opens, Closes, Active, MaxActive, Requests int
	Events                                     []FakeRenderEvent
}

// FakeRenderer models one serial replay session for a whole batch. All work is
// synchronous within Render's scope; cancellation settles the owned close before
// return. Its PNG is explicitly synthetic, with the real plan's dimensions.
type FakeRenderer struct {
	mu      sync.Mutex
	control FakeControl
	state   FakeRenderState
}

func NewFakeRenderer() *FakeRenderer { return &FakeRenderer{} }
func (f *FakeRenderer) Configure(c FakeControl) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.state.Active != 0 {
		return fmt.Errorf("sketch image fake: session active")
	}
	if c.Stage != "" && c.Stage != StageOpen && c.Stage != StageImage && c.Stage != StageClose {
		return fmt.Errorf("sketch image fake: invalid stage")
	}
	if c.Image < 0 {
		return fmt.Errorf("sketch image fake: invalid image index")
	}
	f.control = c
	return nil
}
func (f *FakeRenderer) Snapshot() FakeRenderState {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := f.state
	out.Events = slices.Clone(out.Events)
	return out
}
func (f *FakeRenderer) step(ctx context.Context, c FakeControl, session int, stage RenderStage, index int, id string) error {
	f.mu.Lock()
	f.state.Events = append(f.state.Events, FakeRenderEvent{session, stage, index, id})
	if stage == StageImage {
		f.state.Requests++
	}
	f.mu.Unlock()
	if c.Stage == stage && (stage != StageImage || c.Image == index) {
		if c.Reached != nil {
			select {
			case c.Reached <- struct{}{}:
			default:
			}
		}
		if c.Gate != nil {
			select {
			case <-c.Gate:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		if c.Err != nil {
			return c.Err
		}
	}
	return ctx.Err()
}
func (f *FakeRenderer) Render(ctx context.Context, b Batch) (images []Image, err error) {
	if ctx == nil {
		return nil, fmt.Errorf("sketch image fake: nil context")
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	plans, err := ValidateBatch(b)
	if err != nil {
		return nil, err
	}
	f.mu.Lock()
	if f.state.Active != 0 {
		f.mu.Unlock()
		return nil, fmt.Errorf("sketch image fake: session busy")
	}
	f.state.Opens++
	session := f.state.Opens
	f.state.Active++
	f.state.MaxActive = max(f.state.MaxActive, f.state.Active)
	control := f.control
	f.mu.Unlock()
	defer func() {
		closeErr := f.step(ctx, control, session, StageClose, -1, "")
		f.mu.Lock()
		f.state.Closes++
		f.state.Active--
		f.mu.Unlock()
		err = errors.Join(err, closeErr)
		if err != nil {
			images = nil
		}
	}()
	if err = f.step(ctx, control, session, StageOpen, -1, ""); err != nil {
		return nil, err
	}
	images = make([]Image, 0, len(plans))
	for index, plan := range plans {
		if err = f.step(ctx, control, session, StageImage, index, plan.Sketch); err != nil {
			return nil, err
		}
		pixels := image.NewRGBA(image.Rect(0, 0, plan.Width, plan.Height))
		draw.Draw(pixels, pixels.Bounds(), &image.Uniform{color.RGBA{240, 240, 240, 255}}, image.Point{}, draw.Src)
		var encoded bytes.Buffer
		if err = png.Encode(&encoded, pixels); err != nil {
			return nil, err
		}
		images = append(images, Image{SketchID: plan.Sketch, Plan: plan, PNG: encoded.Bytes(), Limitations: []string{"synthetic stateful-fake image; no recorded page pixels"}})
	}
	if err = ValidateImages(plans, images); err != nil {
		return nil, err
	}
	return images, nil
}
