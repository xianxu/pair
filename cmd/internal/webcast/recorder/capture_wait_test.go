package recorder

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	rec "github.com/xianxu/pair/cmd/internal/webcast/recording"
	"image"
	"image/png"
	"strings"
	"testing"
)

func readyReply(view rec.Box, pending bool, reason string) any {
	return map[string]any{"result": map[string]any{"value": map[string]any{"viewport": view, "dpr": 2, "pending": pending, "reason": reason}}}
}
func TestVisibleCrop(t *testing.T) {
	for _, tt := range []struct {
		box, view, want rec.Box
		ok              bool
	}{
		{rec.Box{10, 20, 30, 40}, rec.Box{0, 0, 100, 100}, rec.Box{10, 20, 30, 40}, true},
		{rec.Box{1.2, 2.4, 3.6, 4.8}, rec.Box{0, 0, 100, 100}, rec.Box{1.2, 2.4, 3.6, 4.8}, true},
		{rec.Box{10, 20, 30, 40}, rec.Box{20, 30, 100, 100}, rec.Box{20, 30, 20, 30}, true},
		{rec.Box{0, 0, 20, 20}, rec.Box{20, 0, 20, 20}, rec.Box{}, false},
		{rec.Box{0, 0, 20, 20}, rec.Box{30, 0, 20, 20}, rec.Box{}, false},
	} {
		got, ok := visibleCrop(tt.box, tt.view)
		if got != tt.want || ok != tt.ok {
			t.Fatalf("%v in %v = %v,%v", tt.box, tt.view, got, ok)
		}
	}
}
func TestCropWaitObservesProgressAndKeepsContext(t *testing.T) {
	h, _ := cropHarness(t)
	calls := 0
	view := rec.Box{0, 1100, 800, 600}
	h.fc.captureProbe = func(_ context.Context, p map[string]any) (any, error) {
		calls++
		if p["executionContextId"] != float64(42) {
			t.Errorf("context %v", p)
		}
		return readyReply(view, calls == 1, ""), nil
	}
	state, status, err := h.r.waitForCapture(context.Background(), captureIdentity{top: "A", contextID: 42}, rec.Box{10, 1200, 20, 10})
	if err != nil || calls != 2 || state.Viewport != view || status.Status != "settled" {
		t.Fatalf("got %+v %+v %v calls=%d", state, status, err, calls)
	}
}
func TestCropWaitNeverTreatsUnknownAsSettled(t *testing.T) {
	for _, reason := range []string{"paused animation", "infinite animation", "inspection limit", "inspection unavailable"} {
		t.Run(reason, func(t *testing.T) {
			h, _ := cropHarness(t)
			h.fc.captureProbe = func(context.Context, map[string]any) (any, error) {
				return readyReply(rec.Box{0, 0, 800, 600}, false, reason), nil
			}
			_, status, err := h.r.waitForCapture(context.Background(), captureIdentity{top: "A", contextID: 42}, rec.Box{1, 1, 20, 10})
			if err != nil || status.Status != "unsettled" || status.Reason != reason {
				t.Fatalf("%+v %v", status, err)
			}
		})
	}
}
func TestCropWaitDeadlineIsExplicitAndCancelledParentFails(t *testing.T) {
	for _, cancelParent := range []bool{false, true} {
		t.Run(map[bool]string{false: "deadline", true: "cancel"}[cancelParent], func(t *testing.T) {
			h, _ := cropHarness(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			calls := 0
			h.fc.captureProbe = func(callCtx context.Context, _ map[string]any) (any, error) {
				calls++
				if calls == 1 {
					return readyReply(rec.Box{0, 0, 800, 600}, true, ""), nil
				}
				if cancelParent {
					cancel()
				}
				<-callCtx.Done()
				return nil, callCtx.Err()
			}
			_, status, err := h.r.waitForCapture(ctx, captureIdentity{top: "A", contextID: 42}, rec.Box{1, 1, 20, 10})
			if cancelParent {
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("cancel: %v", err)
				}
			} else if err != nil || status.Status != "unsettled" || status.Reason != "timeout" {
				t.Fatalf("deadline: %+v %v", status, err)
			}
		})
	}
}
func TestCropWaitRefusesViewportDriftAndBadProbe(t *testing.T) {
	for _, kind := range []string{"viewport", "missing", "unknown reason"} {
		t.Run(kind, func(t *testing.T) {
			h, _ := cropHarness(t)
			calls := 0
			h.fc.captureProbe = func(context.Context, map[string]any) (any, error) {
				calls++
				if kind == "missing" {
					return map[string]any{}, nil
				}
				if kind == "unknown reason" {
					return readyReply(rec.Box{0, 0, 800, 600}, false, "bad"), nil
				}
				return readyReply(rec.Box{float64(calls), 0, 800, 600}, calls == 1, ""), nil
			}
			_, _, err := h.r.waitForCapture(context.Background(), captureIdentity{top: "A", contextID: 42}, rec.Box{1, 1, 20, 10})
			if err == nil {
				t.Fatal("invalid/drifting probe accepted")
			}
			if kind == "viewport" && !strings.Contains(err.Error(), "viewport") {
				t.Fatal(err)
			}
		})
	}
}

func TestCaptureViewportAndSettlementReachEvidence(t *testing.T) {
	for _, tt := range []struct {
		name        string
		box         rec.Box
		want        rec.Box
		reason      string
		unavailable string
	}{
		{"clip", rec.Box{790, 10, 20, 10}, rec.Box{790, 10, 10, 10}, "", ""},
		{"unsettled", rec.Box{10, 10, 20, 10}, rec.Box{10, 10, 20, 10}, "infinite animation", ""},
		{"outside", rec.Box{900, 10, 20, 10}, rec.Box{}, "", "outside viewport"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			h, doc := cropHarness(t)
			h.fc.captureProbe = func(context.Context, map[string]any) (any, error) {
				return readyReply(rec.Box{0, 0, 800, 600}, false, tt.reason), nil
			}
			var buf bytes.Buffer
			png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 20, 20)))
			shots := 0
			h.fc.more = func(_ string, method string, p map[string]any, _ *fakePage) (any, error, bool) {
				if method == "Runtime.callFunctionOn" {
					return map[string]any{"result": map[string]any{"value": map[string]any{"region": textRequestRegion(p)}}}, nil, true
				}
				if method == "Page.captureScreenshot" {
					shots++
					return map[string]any{"data": base64.StdEncoding.EncodeToString(buf.Bytes())}, nil, true
				}
				return nil, nil, false
			}
			got, err := h.r.CaptureCrops(context.Background(), doc, captureRequests([]rec.Box{tt.box}), nil)
			if err != nil || len(got) != 1 {
				t.Fatalf("%+v %v", got, err)
			}
			c := got[0]
			if c.Box != tt.want || c.Unavailable != tt.unavailable {
				t.Fatalf("crop %+v", c)
			}
			if tt.unavailable != "" {
				if shots != 0 || c.Settlement != nil || len(c.PNG) != 0 || c.BoxText.Unavailable != tt.unavailable {
					t.Fatalf("offscreen %+v shots%d", c, shots)
				}
				return
			}
			if shots != 1 || c.ClippedToViewport != (tt.box != tt.want) || c.Settlement == nil || c.Settlement.Reason != tt.reason {
				t.Fatalf("metadata %+v", c)
			}
		})
	}
}
func TestCaptureRefusesViewportChangeBeforeOrDuringScreenshot(t *testing.T) {
	for _, during := range []bool{false, true} {
		t.Run(map[bool]string{false: "before", true: "during"}[during], func(t *testing.T) {
			h, doc := cropHarness(t)
			probes, shots := 0, 0
			h.fc.captureProbe = func(context.Context, map[string]any) (any, error) {
				probes++
				x := 0.0
				if (!during && probes > 1) || (during && shots > 0) {
					x = 1
				}
				return readyReply(rec.Box{x, 0, 800, 600}, false, ""), nil
			}
			var buf bytes.Buffer
			png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 20, 10)))
			h.fc.more = func(_ string, method string, p map[string]any, _ *fakePage) (any, error, bool) {
				if method == "Runtime.callFunctionOn" {
					return map[string]any{"result": map[string]any{"value": map[string]any{"region": textRequestRegion(p)}}}, nil, true
				}
				if method == "Page.captureScreenshot" {
					shots++
					return map[string]any{"data": base64.StdEncoding.EncodeToString(buf.Bytes())}, nil, true
				}
				return nil, nil, false
			}
			got, err := h.r.CaptureCrops(context.Background(), doc, captureRequests([]rec.Box{{10, 10, 20, 10}}), nil)
			if err == nil || !strings.Contains(err.Error(), "viewport") || got != nil {
				t.Fatalf("cropaccepted %+v %v", got, err)
			}
			if !during && shots != 0 {
				t.Fatal("screenshot ran after viewport drift")
			}
		})
	}
}
