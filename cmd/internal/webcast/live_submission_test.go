package webcast

import (
	"bytes"
	"context"
	"fmt"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	core "github.com/xianxu/pair/cmd/internal/submission"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// Opt-in conformance for current-tab screenshots: page CSS coordinates remain
// correct after scrolling, two overlapping notes retain two images, and DPR
// scales pixels once. A blue recorded rectangle turns red before capture, so
// replayed/old pixels cannot satisfy this test.
//
// WEBCAST_LIVE_CHROME=1 GOMAXPROCS=2 go test -p 1 ./cmd/internal/webcast -run '^TestLiveSubmissionCrop$' -count=1 -timeout 30s -v
func TestLiveSubmissionCrop(t *testing.T) {
	if os.Getenv("WEBCAST_LIVE_CHROME") == "" {
		t.Skip("set WEBCAST_LIVE_CHROME=1 to drive a real headless Chrome")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	page := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<!doctype html><html><head><title>Submission crop coordinates</title><style>
 html,body { margin:0; width:800px; height:2400px; background:white; }
 #region { position:absolute; left:80px; top:1400px; width:400px; height:250px; background:rgb(0,0,255); }
  #current { position:absolute;left:170px;top:1460px;font-size:10px;line-height:12px; }
 </style></head><body><div id="region">Crop reference</div>
 <button id="current" aria-label="Exact current name" title="Current tip"><span>Old</span></button>
 <span style="position:absolute;left:110px;top:1430px">Padding only</span>
 <span aria-hidden="true" style="position:absolute;left:170px;top:1475px">Hidden</span>
 </body></html>`)
	}))
	defer page.Close()
	pageURL := page.URL + "/"
	dir := filepath.Join(t.TempDir(), "recording")
	// The shared helper owns a fresh temporary profile and always Headless:true.
	// Disable periodic shots: every asserted PNG must come from CaptureCrops.
	c := liveRecordingChrome(t, dir, -1)
	h := liveHarness{c}
	session := h.session(t, "")
	// Bound any outstanding CDP or recorder wait to this test's one deadline.
	stopDeadline := context.AfterFunc(ctx, c.proc.Kill)
	defer stopDeadline()
	call := func(method string, params, result any) {
		t.Helper()
		if err := c.conn.Call(ctx, session, method, params, result); err != nil {
			t.Fatalf("%s: %v", method, err)
		}
	}
	call("Emulation.setDeviceMetricsOverride", map[string]any{"width": 800, "height": 600, "deviceScaleFactor": 2, "mobile": false}, nil)
	call("Page.navigate", map[string]any{"url": pageURL}, nil)
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	wait := func() {
		t.Helper()
		select {
		case <-ctx.Done():
			t.Fatal("submission crop readiness: ", ctx.Err())
		case <-ticker.C:
		}
	}
	for {
		var result struct {
			Result struct {
				Value bool `json:"value"`
			} `json:"result"`
		}
		call("Runtime.evaluate", map[string]any{"expression": "document.readyState === 'complete' && !!document.getElementById('region')", "returnByValue": true}, &result)
		if result.Result.Value {
			break
		}
		wait()
	}
	if err := c.SetLive(true); err != nil {
		t.Fatal(err)
	}
	document := 0
	for document == 0 {
		recs, err := readSubmissionRecording(ctx, dir)
		if err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
		if len(recs) > 0 {
			state, err := recording.Reduce(recs, recs[len(recs)-1].Time())
			if err != nil {
				t.Fatal(err)
			}
			if state.Top != nil && state.Live && state.URL == pageURL {
				document = state.DocSeg
			}
		}
		if document == 0 {
			wait()
		}
	}
	var changed struct {
		Result struct {
			Value struct {
				DPR    float64 `json:"dpr"`
				Scroll float64 `json:"scroll"`
				Top    float64 `json:"top"`
			} `json:"value"`
		} `json:"result"`
	}
	call("Runtime.evaluate", map[string]any{
		"expression": `(async () => {
 const region = document.getElementById('region');
 region.style.background = 'rgb(255, 0, 0)';
 document.querySelector('#current span').textContent = 'New exact';
 window.scrollTo(0, 1100);
 await new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve)));
 return {dpr: devicePixelRatio, scroll: scrollY, top: region.getBoundingClientRect().top + scrollY};
 })()`, "awaitPromise": true, "returnByValue": true}, &changed)
	if got := changed.Result.Value; got.DPR != 2 || got.Scroll != 1100 || got.Top != 1400 {
		t.Fatalf("unexpected page geometry: %+v", got)
	}
	boxes := []recording.Box{{100, 1420, 120, 80}, {160, 1450, 120, 80}}
	began := time.Now()
	crops, err := c.CaptureCrops(ctx, document, []core.CaptureRegion{{Box: boxes[0], Region: recording.Box{160, 1450, 80, 40}}, {Box: boxes[1], Region: recording.Box{160, 1450, 80, 40}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(crops) != len(boxes) {
		t.Fatalf("got %d crops, want two overlapping note crops", len(crops))
	}
	for i, crop := range crops {
		if crop.Unavailable != "" || crop.Document != document || crop.Box != boxes[i] || crop.CapturedAt.Before(began) {
			t.Fatalf("crop %d metadata: %+v", i, crop)
		}
		var visible, names strings.Builder
		for _, entry := range crop.BoxText.VisibleText {
			visible.WriteString(entry.Text)
		}
		for _, entry := range crop.BoxText.AccessibleNames {
			names.WriteString(entry.Text)
		}
		if !strings.Contains(visible.String(), "New exact") || strings.Contains(visible.String(), "Old") || strings.Contains(visible.String(), "Padding") || strings.Contains(visible.String(), "Hidden") || !strings.Contains(names.String(), "Exact current name") {
			t.Fatalf("crop %d text: %+v", i, crop.BoxText)
		}
		img, err := png.Decode(bytes.NewReader(crop.PNG))
		if err != nil {
			t.Fatalf("crop %d: %v", i, err)
		}
		bounds := img.Bounds()
		wantW, wantH := int(boxes[i][2]*2), int(boxes[i][3]*2)
		if bounds.Dx() != wantW || bounds.Dy() != wantH {
			t.Fatalf("crop %d dimensions=%v; want %dx%d at DPR 2", i, bounds, wantW, wantH)
		}
		red, green, blue, alpha := img.At(bounds.Min.X+bounds.Dx()/2, bounds.Min.Y+bounds.Dy()/2).RGBA()
		if red != 0xffff || green != 0 || blue != 0 || alpha != 0xffff {
			t.Fatalf("crop %d center RGBA=(%d,%d,%d,%d); want current red pixels at page CSS coordinates", i, red, green, blue, alpha)
		}
	}
}
