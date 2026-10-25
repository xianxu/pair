package webcast

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// TestLiveControlReadiness exercises actual closed-shadow geometry, not a
// fake positive rectangle. Refusal must happen before any mouse input.
func TestLiveControlReadiness(t *testing.T) {
	c := liveChrome(t)
	h := liveHarness{c}
	session := h.session(t, "")
	waitState(t, c, "live", func(s State) bool { return s.HasTab })
	if err := c.SetLive(true); err != nil {
		t.Fatal(err)
	}
	waitState(t, c, "live", func(s State) bool { return s.Phase == PhaseLive })
	inputs, button := 0, 0
	call := func(method string, params, result any) error {
		if method == "Input.dispatchMouseEvent" {
			inputs++
		}
		err := c.call(session, method, params, result)
		if err == nil && method == "DOM.querySelector" && params.(map[string]any)["selector"] == ".cast-button" {
			raw, e := json.Marshal(result)
			if e != nil {
				t.Fatal(e)
			}
			var node struct {
				ID int `json:"nodeId"`
			}
			if e = json.Unmarshal(raw, &node); e != nil {
				t.Fatal(e)
			}
			if node.ID != 0 {
				button = node.ID
			}
		}
		return err
	}
	if err := clickLiveControl(call, time.Now().Add(10*time.Second)); err != nil {
		t.Fatal(err)
	}
	waitState(t, c, "visible control paused", func(s State) bool { return s.Phase == PhaseIdle })
	if inputs != 2 || button == 0 {
		t.Fatalf("visible inputs=%d button=%d", inputs, button)
	}
	for _, tc := range []struct{ name, style string }{
		{"hidden", "visibility:hidden!important"},
		{"transparent", "opacity:0!important"},
		{"offscreen", "transform:translateX(-2000px)!important"},
		{"clipped", "clip-path:inset(100%)!important"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := c.call(session, "DOM.setAttributeValue", map[string]any{"nodeId": button, "name": "style", "value": tc.style}, nil); err != nil {
				t.Fatal(err)
			}
			before := inputs
			if err := clickLiveControl(call, time.Now()); err == nil {
				t.Fatal("unusable control admitted")
			}
			if inputs != before {
				t.Fatalf("refusal sent %d mouse events", inputs-before)
			}
			if c.State().Phase != PhaseIdle {
				t.Fatal("refusal changed cast state")
			}
			if err := c.call(session, "DOM.removeAttribute", map[string]any{"nodeId": button, "name": "style"}, nil); err != nil {
				t.Fatal(err)
			}
		})
	}
	if err := clickLiveControl(call, time.Now()); err != nil {
		t.Fatal(err)
	}
	waitState(t, c, "restored control resumed", func(s State) bool { return s.Phase == PhaseLive })
	if err := c.call(session, "DOM.removeNode", map[string]any{"nodeId": button}, nil); err != nil {
		t.Fatal(err)
	}
	before := inputs
	if err := clickLiveControl(call, time.Now()); err == nil || !strings.Contains(err.Error(), "not mounted") {
		t.Fatalf("missing refusal: %v", err)
	}
	if inputs != before {
		t.Fatalf("missing control sent %d events", inputs-before)
	}
}
