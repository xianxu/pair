package webcast

import (
	"encoding/json"
	"github.com/xianxu/pair/cmd/internal/cdp"
	"testing"
)

func sketchBindingEvent(context int64) cdp.Event {
	body := json.RawMessage(`{"operation":"begin","sequence":0,"epoch":1,"seg":1,"color":0}`)
	envelope := operatorSketchEnvelope{Request: "r", Kind: "sketch-begin", Generation: "1", Epoch: 1, Body: body}
	payload, _ := json.Marshal(map[string]any{"t": "sketch", "request": envelope})
	params, _ := json.Marshal(map[string]any{"name": bindingName, "executionContextId": context, "payload": string(payload)})
	return cdp.Event{SessionID: "A", Method: "Runtime.bindingCalled", Params: params}
}
func TestOperatorSketchBindingUsesTrustedContextLifetime(t *testing.T) {
	runtime := newOperatorSketchRuntime(operatorSketchRuntimeOptions{})
	c := &Chrome{operatorSketch: runtime}
	s := &loopState{sessions: map[string]string{"A": "tab"}, contexts: map[string]int64{"A": 42}}
	if !c.handleSketchBinding(s, sketchBindingEvent(666)) {
		t.Fatal("sketch envelope not recognized")
	}
	if len(runtime.requests) != 0 {
		t.Fatal("page claimed isolated context")
	}
	if !c.handleSketchBinding(s, sketchBindingEvent(42)) {
		t.Fatal("trusted sketch not recognized")
	}
	q := <-runtime.requests
	if q.Scope.ParentContext != 42 || q.Scope.ParentSession != "A" || !operatorSketchAlive(q.Lifetime) {
		t.Fatal("lost trusted scope")
	}
	c.retireOperatorSketchScopes(s, "A")
	if operatorSketchAlive(q.Lifetime) {
		t.Fatal("retired origin survived")
	}
	c.handleSketchBinding(s, sketchBindingEvent(42))
	next := <-runtime.requests
	if next.Lifetime == q.Lifetime || !operatorSketchAlive(next.Lifetime) {
		t.Fatal("reused context ID resurrected original token")
	}
}
