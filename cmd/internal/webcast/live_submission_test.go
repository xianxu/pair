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
	// BR-4: intersect notes, viewport and composed ancestor scrollports. Raw
	// DOM rectangles remain outside overflow clips, including assigned slots.
	call("Runtime.evaluate", map[string]any{"expression": `(async()=>{
 const fixture=document.createElement('div');fixture.id='clip-fixture';
 fixture.style.cssText='position:absolute;left:500px;top:1400px;width:250px;height:220px;font:10px/12px monospace';
 fixture.innerHTML='<div style="position:absolute;left:0;top:0;width:20px;height:20px;overflow:hidden"><span aria-label="Hidden overflow name" style="position:absolute;left:40px;top:0">HiddenOverflow</span></div>'+
'<div id="partial-clip" style="position:absolute;left:0;top:30px;width:24px;height:20px;overflow:hidden"><span id="partial-text">ABCDEFGHIJ</span></div>'+
'<div style="position:absolute;left:120px;top:30px;transform:scale(2);transform-origin:top left;width:12px;height:12px;border:2px solid black;overflow:hidden"><span id="scaled-text">123456</span></div>'+
'<div id="outer-scroll" style="position:absolute;left:0;top:60px;width:100px;height:12px;overflow-x:hidden;overflow-y:auto"><div style="position:relative;width:100px;height:60px;overflow-x:clip;overflow-y:visible"><span aria-label="Hidden nested name" style="position:absolute;top:0">HiddenNested</span><span style="position:absolute;top:30px">Nested visible</span></div></div>'+
'<div id="slot-host" style="position:absolute;left:0;top:100px;width:100px;height:20px"><span aria-label="Hidden slot name" style="position:absolute;left:40px;top:0">HiddenSlot</span></div>'+
'<div style="position:absolute;left:0;top:130px;opacity:0"><span aria-label="Hidden opacity name">HiddenOpacity</span></div>'+
'<div style="position:absolute;left:0;top:160px;clip-path:circle(5px)"><span aria-label="Unsupported shape name">UnsupportedShape</span></div>';
 document.body.append(fixture);
 document.getElementById('outer-scroll').scrollTop=30;
 document.getElementById('slot-host').attachShadow({mode:'open'}).innerHTML='<div style="position:relative;width:20px;height:20px;overflow:hidden"><slot></slot></div>';
 const wide=document.createElement('div');wide.setAttribute('aria-label','Wide offscreen ancestor');wide.style.cssText='position:absolute;left:0;top:1400px;width:1200px;height:200px;pointer-events:none';document.body.append(wide);
 await new Promise(resolve=>requestAnimationFrame(()=>requestAnimationFrame(resolve)));
 })()`, "awaitPromise": true}, nil)
	evidence, err := c.CaptureCrops(ctx, document, []core.CaptureRegion{
		{Box: recording.Box{500, 1400, 250, 220}, Region: recording.Box{500, 1400, 250, 220}},
		{Box: recording.Box{900, 1400, 10, 10}, Region: recording.Box{900, 1400, 10, 10}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var visible, names strings.Builder
	for _, entry := range evidence[0].BoxText.VisibleText {
		visible.WriteString(entry.Text)
	}
	for _, entry := range evidence[0].BoxText.AccessibleNames {
		names.WriteString(entry.Text)
	}
	if strings.Contains(visible.String(), "Hidden") || strings.Contains(names.String(), "Hidden") || strings.Contains(visible.String(), "UnsupportedShape") {
		t.Fatalf("clipped content claimed visible: %+v", evidence[0].BoxText)
	}
	if !strings.Contains(visible.String(), "Nested visible") {
		t.Fatalf("nested scroll lost visible text: %+v", evidence[0].BoxText)
	}
	foundPartial, foundScaled := false, false
	for _, entry := range evidence[0].BoxText.VisibleText {
		if entry.Selector == "#partial-text" {
			foundPartial = true
			if entry.Text != "ABCD" {
				t.Fatalf("partial characters = %q", entry.Text)
			}
		}
		if entry.Selector == "#scaled-text" {
			foundScaled = true
			if entry.Text != "12" {
				t.Fatalf("scaled scrollport characters = %q", entry.Text)
			}
		}
	}
	if !foundPartial || !foundScaled {
		t.Fatalf("missing clipped text: %+v", evidence[0].BoxText)
	}
	if !evidence[0].BoxText.Truncated || !strings.Contains(strings.Join(evidence[0].BoxText.Skipped, ","), "unsupported clipping") {
		t.Fatalf("unsupported clip not explicit: %+v", evidence[0].BoxText)
	}
	if len(evidence[1].BoxText.VisibleText)+len(evidence[1].BoxText.AccessibleNames) != 0 {
		t.Fatalf("offscreen region returned evidence: %+v", evidence[1].BoxText)
	}

}
