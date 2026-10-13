package recorder

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/xianxu/pair/cmd/internal/submission"
	rec "github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// CapturedCrop is current-tab pixel evidence, distinct from note-time DOM.
type CapturedCrop struct {
	PNG         []byte
	Box         rec.Box
	Document    int
	CapturedAt  time.Time
	Unavailable string
}

type captureIdentity struct {
	doc              int
	top, loader, tab string
	live             bool
}

// Barrier is a CDP fence forwarded by Chrome after every preceding event.
type Barrier struct{ Done chan struct{} }

func (r *Recorder) captureBarrier(ctx context.Context) error {
	if r.opts.SyncEvents != nil {
		return r.opts.SyncEvents(ctx)
	}
	return r.ask(ctx, func() {}) // stateful doubles feed the recorder directly
}

func (r *Recorder) captureIdentity(ctx context.Context) (captureIdentity, error) {
	ch := make(chan captureIdentity, 1)
	err := r.ask(ctx, func() { ch <- captureIdentity{r.m.State().DocSeg, r.top, r.loader[r.top], r.tab.ID, r.live} })
	if err != nil {
		return captureIdentity{}, err
	}
	return <-ch, nil
}

// CaptureCrops captures sequential clips without blocking the recorder loop.
// Its CDP barrier includes events already read but not yet reduced, including
// same-URL navigation. No pixels enter the recording/viewer record stream.
func (r *Recorder) CaptureCrops(parent context.Context, doc int, boxes []rec.Box) ([]CapturedCrop, error) {
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	stop := context.AfterFunc(r.ctx, cancel)
	defer stop()
	for _, box := range boxes {
		if err := submission.ValidateBox(box); err != nil {
			return nil, err
		}
	}
	if err := r.captureBarrier(ctx); err != nil {
		return nil, err
	}
	start, err := r.captureIdentity(ctx)
	if err != nil {
		return nil, err
	}
	if start.doc != doc || !start.live || start.top == "" {
		out := make([]CapturedCrop, len(boxes))
		for i := range out {
			out[i] = CapturedCrop{Box: boxes[i], Unavailable: "page changed"}
		}
		return out, nil
	}
	var out []CapturedCrop
	for _, box := range boxes {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		var res struct {
			Data string `json:"data"`
		}
		callCtx, done := context.WithTimeout(ctx, 5*time.Second)
		_, err := r.opts.Call.CallSeq(callCtx, start.top, "Page.captureScreenshot", map[string]any{
			"format": "png", "captureBeyondViewport": true, "fromSurface": true,
			"clip": map[string]any{"x": box[0], "y": box[1], "width": box[2], "height": box[3], "scale": 1},
		}, &res)
		done()
		if err != nil {
			return nil, fmt.Errorf("capture crop: %w", err)
		}
		if len(res.Data) > base64.StdEncoding.EncodedLen(8<<20) {
			return nil, errors.New("crop PNG exceeds 8 MiB")
		}
		png, err := base64.StdEncoding.DecodeString(res.Data)
		if err != nil {
			return nil, err
		}
		if err = submission.ValidatePNG(png); err != nil {
			return nil, err
		}
		out = append(out, CapturedCrop{PNG: png, Box: box, Document: doc, CapturedAt: time.Now().UTC()})
		if err = r.captureBarrier(ctx); err != nil {
			return nil, err
		}
		current, err := r.captureIdentity(ctx)
		if err != nil {
			return nil, err
		}
		if current != start {
			return nil, errors.New("page changed during crop capture; submit again")
		}
	}
	return out, nil
}
