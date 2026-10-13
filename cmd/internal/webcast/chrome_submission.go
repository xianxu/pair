package webcast

import (
	"context"
	"github.com/xianxu/pair/cmd/internal/webcast/recorder"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// CropSource captures the cast tab through its existing recording connection.
type CropSource interface {
	CaptureCrops(context.Context, int, []recording.Box) ([]recorder.CapturedCrop, error)
}

func (c *Chrome) CaptureCrops(ctx context.Context, doc int, boxes []recording.Box) ([]recorder.CapturedCrop, error) {
	if c.rec == nil {
		return nil, ErrNotRecording
	}
	return c.rec.CaptureCrops(ctx, doc, boxes)
}
