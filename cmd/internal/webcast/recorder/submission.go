package recorder

import (
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/xianxu/pair/cmd/internal/submission"
	rec "github.com/xianxu/pair/cmd/internal/webcast/recording"
)

//go:embed box_text.js
var boxTextCollector string

// CapturedCrop is current-tab pixel and text evidence, distinct from note-time DOM.
type CapturedCrop struct {
	Origin      string // admitted current page origin, fenced with the pixels and text
	BoxText     submission.BoxText
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
	contextID        int64
	origin           string
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
	err := r.ask(ctx, func() {
		id, _ := r.topContext(r.top)
		ch <- captureIdentity{r.m.State().DocSeg, r.top, r.loader[r.top], r.tab.ID, r.live, id, submission.OriginOf(r.m.State().URL)}
	})
	if err != nil {
		return captureIdentity{}, err
	}
	return <-ch, nil
}

// CaptureCrops captures sequential clips without blocking the recorder loop.
// Its CDP barrier includes events already read but not yet reduced, including
// same-URL navigation. No pixels enter the recording/viewer record stream.
func (r *Recorder) CaptureCrops(parent context.Context, doc int, boxes []submission.CaptureRegion) ([]CapturedCrop, error) {
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	stop := context.AfterFunc(r.ctx, cancel)
	defer stop()
	for _, box := range boxes {
		if err := submission.ValidateBox(box.Box); err != nil {
			return nil, err
		}
		if err := submission.ValidateBox(box.Region); err != nil {
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
	if start.doc != doc || start.top == "" {
		out := make([]CapturedCrop, len(boxes))
		for i := range out {
			out[i] = CapturedCrop{Box: boxes[i].Box, Unavailable: "page changed", BoxText: submission.BoxText{Unavailable: "page changed"}}
		}
		return out, nil
	}
	if !start.live {
		return nil, errors.New("cast is paused; resume before capturing note crops")
	}
	if start.contextID == 0 {
		return nil, errors.New("capture has no admitted top-frame context")
	}
	verifyIdentity := func() error {
		if err := r.captureBarrier(ctx); err != nil {
			return err
		}
		current, err := r.captureIdentity(ctx)
		if err != nil {
			return err
		}
		if current != start {
			return errors.New("page changed during crop capture; submit again")
		}
		return nil
	}

	total := 0
	var out []CapturedCrop
	for _, request := range boxes {
		box := request.Box
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		var textResult struct {
			Result struct {
				Value json.RawMessage `json:"value"`
			} `json:"result"`
			Exception json.RawMessage `json:"exceptionDetails"`
		}
		textCtx, textDone := context.WithTimeout(ctx, 5*time.Second)
		_, err := r.opts.Call.CallSeq(textCtx, start.top, "Runtime.callFunctionOn", map[string]any{"functionDeclaration": boxTextCollector, "executionContextId": start.contextID, "arguments": []any{map[string]any{"value": request.Region}}, "returnByValue": true}, &textResult)
		textDone()
		if err != nil {
			return nil, fmt.Errorf("capture box text: %w", err)
		}
		if len(textResult.Exception) > 0 || len(textResult.Result.Value) == 0 {
			return nil, errors.New("box text context evaluation failed")
		}
		var boxText submission.BoxText
		if err = json.Unmarshal(textResult.Result.Value, &boxText); err != nil {
			return nil, err
		}
		if boxText.Region != request.Region {
			return nil, errors.New("box text region changed")
		}
		if err = submission.ValidateBoxText(boxText); err != nil {
			return nil, err
		}
		if err = verifyIdentity(); err != nil {
			return nil, err
		}
		var res struct {
			Data string `json:"data"`
		}
		callCtx, done := context.WithTimeout(ctx, 5*time.Second)
		err = CaptureScreenshot(callCtx, r.opts.Call, start.top, start.contextID, map[string]any{
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
		if len(png) > submission.MaxBundleBytes-total {
			return nil, errors.New("submission crops exceed 64 MiB")
		}
		total += len(png)
		out = append(out, CapturedCrop{Origin: start.origin, PNG: png, Box: box, Document: doc, CapturedAt: time.Now().UTC(), BoxText: boxText})
		if err = verifyIdentity(); err != nil {
			return nil, err
		}
	}
	return out, nil
}
