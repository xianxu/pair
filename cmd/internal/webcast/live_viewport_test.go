package webcast

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/testwait"
)

// resizeLiveViewport must join the page's resize handling before native input
// samples geometry. A metrics response alone only confirms the viewport size.
func resizeLiveViewport(call func(string, any, any) error, width, height int) error {
	return call("Emulation.setDeviceMetricsOverride", map[string]any{"width": width, "height": height, "deviceScaleFactor": 1, "mobile": false}, nil)
}

func TestLiveViewportResizeJoinsPageHandlers(t *testing.T) {
	c := liveChrome(t)
	session := (liveHarness{c}).session(t, "")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	call := func(method string, params, result any) error {
		return c.conn.Call(ctx, session, method, params, result)
	}
	eval := func(expression string) json.RawMessage {
		t.Helper()
		var out struct {
			Result struct{ Value json.RawMessage }
		}
		if err := call("Runtime.evaluate", map[string]any{"expression": expression, "returnByValue": true}, &out); err != nil {
			t.Fatal(err)
		}
		return out.Result.Value
	}
	// Model the captured ordering using real Chrome events: metrics change first,
	// while the Notes-like panel's resize handler has not clamped its position.
	eval(`globalThis.viewportHeld=false;globalThis.panelTop=110;
globalThis.holdViewport=e=>{viewportHeld=true;e.stopImmediatePropagation()};
addEventListener('resize',holdViewport,true);
addEventListener('resize',()=>{panelTop=Math.min(110,innerHeight-12-320)});true`)
	release := func() {
		eval(`removeEventListener('resize',holdViewport,true);dispatchEvent(new Event('resize'));true`)
	}
	defer release()
	awaiting, done := make(chan struct{}), make(chan error, 1)
	go func() {
		done <- resizeLiveViewport(func(method string, params, result any) error {
			if method == "Runtime.callFunctionOn" && params.(map[string]any)["awaitPromise"] == true {
				close(awaiting)
			}
			return call(method, params, result)
		}, 900, 400)
	}()
	testwait.Until(t, "held native resize", func() bool { return string(eval(`viewportHeld&&innerHeight===400&&panelTop===110`)) == "true" })
	select {
	case err := <-done:
		t.Fatalf("resize completed before page handlers: %v", err)
	case <-awaiting:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	select {
	case err := <-done:
		t.Fatalf("resize receipt did not remain pending: %v", err)
	default:
	}
	release()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if got := string(eval(`panelTop`)); got != "68" {
		t.Fatalf("resize returned before panel clamped: %s", got)
	}
	// Chrome emits no resize when the requested geometry is already current.
	if err := resizeLiveViewport(call, 900, 400); err != nil {
		t.Fatal(err)
	}
}
