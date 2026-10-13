package webcast

import (
	"context"
	core "github.com/xianxu/pair/cmd/internal/submission"
	"github.com/xianxu/pair/cmd/internal/webcast/recorder"
)

// CropSource captures the cast tab through its existing recording connection.
type CropSource interface {
	CaptureCrops(context.Context, int, []core.CaptureRegion) ([]recorder.CapturedCrop, error)
}

func (c *Chrome) CaptureCrops(ctx context.Context, doc int, boxes []core.CaptureRegion) ([]recorder.CapturedCrop, error) {
	if c.rec == nil {
		return nil, ErrNotRecording
	}
	return c.rec.CaptureCrops(ctx, doc, boxes)
}
