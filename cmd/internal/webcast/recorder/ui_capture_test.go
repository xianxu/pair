package recorder

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

type uiCaller struct {
	confirm bool
	fail    bool
	calls   []string
}

func (f *uiCaller) CallSeq(_ context.Context, session, method string, params, result any) (uint64, error) {
	if session != "s" {
		panic("wrong document session")
	}
	f.calls = append(f.calls, method)
	if method == "Page.captureScreenshot" {
		if f.fail {
			return 1, errors.New("capture failed")
		}
		return 1, nil
	}
	p := params.(map[string]any)
	if p["contextId"] != int64(7) {
		panic("wrong context")
	}
	expression := p["expression"].(string)
	if strings.Contains(expression, "release") {
		f.calls[len(f.calls)-1] = "release"
	}
	b, _ := json.Marshal(map[string]any{"result": map[string]any{"value": f.confirm}})
	return 1, json.Unmarshal(b, result)
}
func TestCaptureScreenshotRequiresSuppressionAndRestores(t *testing.T) {
	for _, tc := range []struct {
		name          string
		confirm, fail bool
		want          []string
	}{
		{"missing", false, false, []string{"Runtime.evaluate", "release"}},
		{"failure", true, true, []string{"Runtime.evaluate", "Page.captureScreenshot", "release"}},
		{"success", true, false, []string{"Runtime.evaluate", "Page.captureScreenshot", "Runtime.evaluate", "release"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &uiCaller{confirm: tc.confirm, fail: tc.fail}
			err := CaptureScreenshot(context.Background(), f, "s", 7, map[string]any{"format": "png"}, nil)
			if (err != nil) != (tc.fail || !tc.confirm) {
				t.Fatalf("error %v", err)
			}
			if strings.Join(f.calls, ",") != strings.Join(tc.want, ",") {
				t.Fatalf("calls %v", f.calls)
			}
		})
	}
}

func TestBackgroundCaptureDefersBeforePixelsButKeepsSuppressionWhenAdmitted(t *testing.T) {
	for _, confirm := range []bool{false, true} {
		f := &uiCaller{confirm: confirm}
		err := CaptureBackgroundScreenshot(context.Background(), f, "s", 7, map[string]any{"format": "png"}, nil)
		if !confirm {
			if !errors.Is(err, ErrBackgroundCaptureDeferred) {
				t.Fatalf("deferred capture: %v", err)
			}
			if strings.Contains(strings.Join(f.calls, ","), "Page.captureScreenshot") {
				t.Fatal("deferred capture read pixels")
			}
		} else if err != nil || strings.Join(f.calls, ",") != "Runtime.evaluate,Page.captureScreenshot,Runtime.evaluate,release" {
			t.Fatalf("admitted capture lost privacy protocol: %v %v", err, f.calls)
		}
	}
}
