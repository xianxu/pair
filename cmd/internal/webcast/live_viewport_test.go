package webcast

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/testwait"
)

// resizeLiveViewport must join the page's resize handling before native input
// samples geometry. A metrics response alone only confirms the viewport size.
func resizeLiveViewport(call func(string, any, any) error, width, height int) (err error) {
	var receipt struct {
		Result struct {
			ObjectID string `json:"objectId"`
		}
	}
	expression := fmt.Sprintf(`(()=>{let finish;const promise=new Promise(resolve=>{finish=resolve});
const matches=()=>innerWidth===%d&&innerHeight===%d;
const resized=()=>{if(matches()){removeEventListener('resize',resized);finish(true)}};
if(matches())finish(true);else addEventListener('resize',resized);
return {promise,cancel(){removeEventListener('resize',resized);finish(false)}}})()`, width, height)
	if err = call("Runtime.evaluate", map[string]any{"expression": expression}, &receipt); err != nil {
		return err
	}
	object := receipt.Result.ObjectID
	if object == "" {
		return errors.New("viewport resize receipt unavailable")
	}
	defer func() {
		err = errors.Join(err,
			call("Runtime.callFunctionOn", map[string]any{"objectId": object, "functionDeclaration": "function(){this.cancel()}"}, nil),
			call("Runtime.releaseObject", map[string]any{"objectId": object}, nil))
	}()
	if err = call("Emulation.setDeviceMetricsOverride", map[string]any{"width": width, "height": height, "deviceScaleFactor": 1, "mobile": false}, nil); err != nil {
		return err
	}
	var result struct{ Result struct{ Value bool } }
	if err = call("Runtime.callFunctionOn", map[string]any{"objectId": object, "functionDeclaration": "function(){return this.promise}", "awaitPromise": true, "returnByValue": true}, &result); err == nil && !result.Result.Value {
		err = errors.New("viewport resize was not confirmed")
	}
	return err
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
	// Count actual browser listeners: a failed metrics or wait call must retire
	// the armed receipt, rather than leaving it attached to a later resize.
	eval(`globalThis.resizeListeners=new Set();
globalThis.savedAdd=addEventListener;globalThis.savedRemove=removeEventListener;
globalThis.addEventListener=function(type,fn,...rest){if(type==='resize')resizeListeners.add(fn);savedAdd.call(this,type,fn,...rest)};
globalThis.removeEventListener=function(type,fn,...rest){if(type==='resize')resizeListeners.delete(fn);savedRemove.call(this,type,fn,...rest)};true`)
	defer func() { eval(`globalThis.addEventListener=savedAdd;globalThis.removeEventListener=savedRemove;true`) }()
	for _, stage := range []string{"metrics", "wait"} {
		t.Run(stage+" error cleanup", func(t *testing.T) {
			want := errors.New("injected " + stage + " failure")
			err := resizeLiveViewport(func(method string, params, result any) error {
				if stage == "metrics" && method == "Emulation.setDeviceMetricsOverride" ||
					stage == "wait" && method == "Runtime.callFunctionOn" && params.(map[string]any)["awaitPromise"] == true {
					return want
				}
				return call(method, params, result)
			}, 901, 401)
			if !errors.Is(err, want) {
				t.Fatalf("resize error = %v", err)
			}
			if got := string(eval(`resizeListeners.size`)); got != "0" {
				t.Fatalf("resize receipt leaked %s listeners", got)
			}
		})
	}
}
