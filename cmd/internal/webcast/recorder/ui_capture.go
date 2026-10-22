package recorder

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"
	"time"
)

var uiToken atomic.Uint64
var errUISuppression = errors.New("Couch UI suppression was not confirmed for this document")

// ErrBackgroundCaptureDeferred means optional pixels were not captured because
// UI suppression was unavailable or an operator sketch/editor was open.
var ErrBackgroundCaptureDeferred = errors.New("background capture deferred for operator UI")

func uiCall(ctx context.Context, call Caller, session string, contextID int64, op, token string, until int64) bool {
	if contextID == 0 {
		return false
	}
	args, _ := json.Marshal([]any{op, token, until})
	var res struct {
		Result struct {
			Value bool `json:"value"`
		} `json:"result"`
	}
	_, err := call.CallSeq(ctx, session, "Runtime.evaluate", map[string]any{
		"contextId": contextID, "returnByValue": true,
		"expression": "globalThis.__couchcastUI ? globalThis.__couchcastUI(..." + string(args) + ") : false",
	}, &res)
	return err == nil && res.Result.Value
}

// CaptureScreenshot is the only capture door for cast pixels. A missing control
// world, navigation, expired suppression or capture failure yields no usable
// pixels. Restore addresses the exact old context, never a replacement document.
func CaptureScreenshot(ctx context.Context, call Caller, session string, contextID int64, params, result any) error {
	return captureScreenshot(ctx, call, session, contextID, params, result, false)
}

// CaptureBackgroundScreenshot preserves an open operator UI without hiding it.
// Admitted captures retain the same fail-closed pixel privacy as explicit ones.
func CaptureBackgroundScreenshot(ctx context.Context, call Caller, session string, contextID int64, params, result any) error {
	return captureScreenshot(ctx, call, session, contextID, params, result, true)
}

func captureScreenshot(ctx context.Context, call Caller, session string, contextID int64, params, result any, background bool) error {
	token := fmt.Sprintf("capture-%d", uiToken.Add(1))
	until := time.Now().Add(2 * callTimeout).UnixMilli()
	defer func() {
		restore, cancel := context.WithTimeout(context.Background(), callTimeout)
		defer cancel()
		uiCall(restore, call, session, contextID, "release", token, 0)
	}()
	callctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	op := "acquire"
	if background {
		op = "acquire-background"
	}
	if !uiCall(callctx, call, session, contextID, op, token, until) {
		if background {
			return ErrBackgroundCaptureDeferred
		}
		return errUISuppression
	}
	if _, err := call.CallSeq(callctx, session, "Page.captureScreenshot", params, result); err != nil {
		return err
	}
	if time.Now().UnixMilli() >= until || !uiCall(callctx, call, session, contextID, "check", token, until) {
		return errUISuppression
	}
	return nil
}
