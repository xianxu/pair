package recorder

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	core "github.com/xianxu/pair/cmd/internal/submission"
	rec "github.com/xianxu/pair/cmd/internal/webcast/recording"
)

//go:embed capture_ready.js
var captureReadiness string

type captureObservation struct {
	Viewport rec.Box `json:"viewport"`
	DPR      float64 `json:"dpr"`
	Pending  bool    `json:"pending"`
	Reason   string  `json:"reason"`
}

func (o captureObservation) sameViewport(other captureObservation) bool {
	return o.Viewport == other.Viewport && o.DPR == other.DPR
}
func visibleCrop(box, viewport rec.Box) (rec.Box, bool) {
	x, y := math.Max(box[0], viewport[0]), math.Max(box[1], viewport[1])
	right, bottom := math.Min(box[0]+box[2], viewport[0]+viewport[2]), math.Min(box[1]+box[3], viewport[1]+viewport[3])
	if right <= x || bottom <= y {
		return rec.Box{}, false
	}
	width, height := right-x, bottom-y
	// Preserve unchanged axes exactly; subtraction can invent clipping for
	// fractional CSS boxes and invalidate the recorded request identity.
	if x == box[0] && right == box[0]+box[2] {
		width = box[2]
	}
	if y == box[1] && bottom == box[1]+box[3] {
		height = box[3]
	}
	return rec.Box{x, y, width, height}, true
}
func (r *Recorder) observeCapture(ctx context.Context, id captureIdentity, box rec.Box, inspect bool) (captureObservation, error) {
	var result struct {
		Result struct {
			Value json.RawMessage `json:"value"`
		} `json:"result"`
		Exception json.RawMessage `json:"exceptionDetails"`
	}
	_, err := r.opts.Call.CallSeq(ctx, id.top, "Runtime.callFunctionOn", map[string]any{
		"functionDeclaration": captureReadiness, "executionContextId": id.contextID,
		"arguments": []any{map[string]any{"value": box}, map[string]any{"value": inspect}}, "returnByValue": true}, &result)
	if err != nil {
		return captureObservation{}, err
	}
	if len(result.Exception) > 0 || len(result.Result.Value) == 0 {
		return captureObservation{}, errors.New("capture readiness context evaluation failed")
	}
	var o captureObservation
	if err = json.Unmarshal(result.Result.Value, &o); err != nil {
		return o, err
	}
	for i, v := range o.Viewport {
		if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 || v > 1e7 || (i >= 2 && v == 0) {
			return o, errors.New("invalid capture viewport")
		}
	}
	if math.IsNaN(o.DPR) || math.IsInf(o.DPR, 0) || o.DPR <= 0 || o.DPR > 100 {
		return o, errors.New("invalid capture pixel ratio")
	}
	if o.Reason != "" {
		if err = core.ValidateCaptureSettlement(&core.CaptureSettlement{Status: "unsettled", Reason: o.Reason}); err != nil {
			return o, err
		}
	}
	return o, nil
}

// waitForCapture observes natural progress only. Its timeout labels uncertainty;
// cancellation of the enclosing submission remains a hard failure.
func (r *Recorder) waitForCapture(parent context.Context, id captureIdentity, box rec.Box) (captureObservation, *core.CaptureSettlement, error) {
	ctx, cancel := context.WithTimeout(parent, 1500*time.Millisecond)
	defer cancel()
	var first, last captureObservation
	have := false
	for {
		o, err := r.observeCapture(ctx, id, box, true)
		if err != nil {
			if parent.Err() != nil {
				return last, nil, parent.Err()
			}
			if have && errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return last, &core.CaptureSettlement{Status: "unsettled", Reason: "timeout"}, nil
			}
			return last, nil, fmt.Errorf("capture readiness: %w", err)
		}
		if parent.Err() != nil {
			return o, nil, parent.Err()
		}
		if have && !o.sameViewport(first) {
			return o, nil, errors.New("viewport changed during crop capture; submit again")
		}
		if !have {
			first = o
			have = true
		}
		last = o
		if o.Reason != "" {
			return o, &core.CaptureSettlement{Status: "unsettled", Reason: o.Reason}, nil
		}
		if !o.Pending {
			return o, &core.CaptureSettlement{Status: "settled"}, nil
		}
		timer := time.NewTimer(50 * time.Millisecond)
		select {
		case <-parent.Done():
			timer.Stop()
			return last, nil, parent.Err()
		case <-ctx.Done():
			timer.Stop()
			return last, &core.CaptureSettlement{Status: "unsettled", Reason: "timeout"}, nil
		case <-timer.C:
		}
	}
}
