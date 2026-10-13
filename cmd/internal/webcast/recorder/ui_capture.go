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
	token := fmt.Sprintf("capture-%d", uiToken.Add(1))
	until := time.Now().Add(2 * callTimeout).UnixMilli()
	defer func() {
		restore, cancel := context.WithTimeout(context.Background(), callTimeout)
		defer cancel()
		uiCall(restore, call, session, contextID, "release", token, 0)
	}()
	callctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	if !uiCall(callctx, call, session, contextID, "acquire", token, until) {
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

func (r *Recorder) topUIContext() int64 {
	for id, frame := range r.contexts[r.top] {
		if frame == r.frameIDs[r.top] {
			return id
		}
	}
	return 0
}
