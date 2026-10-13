package recorder

import (
	"bytes"
	"context"
	"encoding/base64"
	"image"
	"image/png"
	"strings"
	"testing"

	rec "github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func cropHarness(t *testing.T) (*harness, int) {
	h := newHarness(t)
	h.page("A", "http://app/", "L1")
	h.r.Send(Live{Tab: rec.Tab{ID: "tab"}, Session: "A"})
	h.settle()
	return h, h.mustValidate().DocSeg
}

func TestCropCaptureUsesPageCoordinatesAndOnePNGPerBox(t *testing.T) {
	h, doc := cropHarness(t)
	var pngbuf bytes.Buffer
	png.Encode(&pngbuf, image.NewRGBA(image.Rect(0, 0, 20, 10)))
	var clips []map[string]any
	h.fc.more = func(_ string, method string, p map[string]any, _ *fakePage) (any, error, bool) {
		if method != "Page.captureScreenshot" {
			return nil, nil, false
		}
		clips = append(clips, p["clip"].(map[string]any))
		if p["captureBeyondViewport"] != true || p["format"] != "png" {
			t.Errorf("capture params: %v", p)
		}
		return map[string]any{"data": base64.StdEncoding.EncodeToString(pngbuf.Bytes())}, nil, true
	}
	boxes := []rec.Box{{10, 1200, 20, 10}, {12, 1202, 20, 10}}
	got, err := h.r.CaptureCrops(context.Background(), doc, boxes)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || len(clips) != 2 || clips[0]["y"] != float64(1200) || clips[0]["scale"] != float64(1) {
		t.Fatalf("captures %v / %v", got, clips)
	}
	if got[0].Document != doc || got[0].CapturedAt.IsZero() || !bytes.Equal(got[0].PNG, pngbuf.Bytes()) {
		t.Fatalf("capture metadata %+v", got[0])
	}
}

func TestCropCaptureGoneBeforeAndChangedDuring(t *testing.T) {
	for _, during := range []bool{false, true} {
		t.Run(map[bool]string{false: "before", true: "during"}[during], func(t *testing.T) {
			h, doc := cropHarness(t)
			h.page("B", "http://app/", "L2")
			var buf bytes.Buffer
			png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 20, 10)))
			if !during {
				h.r.Send(Live{Tab: rec.Tab{ID: "other"}, Session: "B"})
				h.settle()
			}
			h.fc.more = func(_ string, method string, _ map[string]any, _ *fakePage) (any, error, bool) {
				if method != "Page.captureScreenshot" {
					return nil, nil, false
				}
				h.r.Send(Live{Tab: rec.Tab{ID: "other"}, Session: "B"})
				return map[string]any{"data": base64.StdEncoding.EncodeToString(buf.Bytes())}, nil, true
			}
			got, err := h.r.CaptureCrops(context.Background(), doc, []rec.Box{{0, 0, 20, 10}})
			if during {
				if err == nil || !strings.Contains(err.Error(), "changed") {
					t.Fatalf("during: %v", err)
				}
			} else if err != nil || len(got) != 1 || got[0].Unavailable != "page changed" {
				t.Fatalf("before: %+v %v", got, err)
			}
		})
	}
}

func TestCropCaptureRejectsOversizeBeforeCDP(t *testing.T) {
	h, doc := cropHarness(t)
	before := h.fc.countCalls("Page.captureScreenshot")
	if _, err := h.r.CaptureCrops(context.Background(), doc, []rec.Box{{0, 0, 1e9, 1e9}}); err == nil {
		t.Fatal("huge crop accepted")
	}
	if h.fc.countCalls("Page.captureScreenshot") != before {
		t.Fatal("capture ran for excessive crop")
	}
}

func TestCropCaptureSameURLReloadDuringScreenshot(t *testing.T) {
	h, doc := cropHarness(t)
	var buf bytes.Buffer
	png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 20, 10)))
	h.fc.more = func(_ string, method string, _ map[string]any, p *fakePage) (any, error, bool) {
		if method != "Page.captureScreenshot" {
			return nil, nil, false
		}
		h.fc.mu.Lock()
		h.fc.loader["A"] = "L2"
		h.fc.pages["A"] = newFakePage("http://app/")
		h.fc.pages["A"].frameID = p.frameID
		h.fc.mu.Unlock()
		h.r.Send(Event{Session: "A", Method: "DOM.documentUpdated", Seq: h.fc.next(), Params: raw(map[string]any{})})
		return map[string]any{"data": base64.StdEncoding.EncodeToString(buf.Bytes())}, nil, true
	}
	if _, err := h.r.CaptureCrops(context.Background(), doc, []rec.Box{{0, 0, 20, 10}}); err == nil || !strings.Contains(err.Error(), "changed") {
		t.Fatalf("same URL reload accepted: %v", err)
	}
}
func TestCropCaptureCancelledBeforeCDP(t *testing.T) {
	h, doc := cropHarness(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := h.r.CaptureCrops(ctx, doc, []rec.Box{{0, 0, 20, 10}}); err == nil {
		t.Fatal("cancel ignored")
	}
	if h.fc.countCalls("Page.captureScreenshot") != 0 {
		t.Fatal("cancelled capture called CDP")
	}
}

func TestCropCapturePausedIsNotPageChangedEvidence(t *testing.T) {
	h, doc := cropHarness(t)
	h.r.Send(Pause{})
	h.settle()
	if _, err := h.r.CaptureCrops(context.Background(), doc, []rec.Box{{0, 0, 20, 10}}); err == nil || !strings.Contains(err.Error(), "paused") {
		t.Fatalf("paused capture: %v", err)
	}
	if h.fc.countCalls("Page.captureScreenshot") != 0 {
		t.Fatal("paused capture called CDP")
	}
}
