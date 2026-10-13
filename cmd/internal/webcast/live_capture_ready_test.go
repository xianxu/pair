package webcast

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image/png"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	core "github.com/xianxu/pair/cmd/internal/submission"
	"github.com/xianxu/pair/cmd/internal/webcast/recorder"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// Real-browser oracle for #524: screenshots must not restart a completed
// responsive fade. Readiness labels alone cannot pass the pixel/timeline checks.
// One fresh headless profile, no operator tab/focus, one bounded test lifetime.
//
// WEBCAST_LIVE_CHROME=1 GOMAXPROCS=2 go test -p 1 ./cmd/internal/webcast -run '^TestLiveSubmissionCaptureReady$' -count=1 -timeout 30s -v
func TestLiveSubmissionCaptureReady(t *testing.T) {
	if os.Getenv("WEBCAST_LIVE_CHROME") == "" {
		t.Skip("set WEBCAST_LIVE_CHROME=1 to drive a real headless Chrome")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	page := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<!doctype html><html><head><title>Capture readiness oracle</title><style>
html,body{margin:0;width:100%;height:100%;overflow:hidden;background:black}
.card{position:absolute;width:220px;height:140px;background:white}
#fade{left:64px;top:80px;opacity:0;transform:translateY(18px)}
@keyframes enter{from{opacity:0;transform:translateY(18px)}to{opacity:1;transform:translateY(0)}}
@media(min-width:768px){#fade.seen{animation:enter 180ms linear both}}
#finite{left:64px;top:280px}
#shadow{position:absolute;left:320px;top:80px;width:180px;height:140px}
#unrelated{position:absolute;left:720px;top:280px;width:80px;height:80px;background:white}
#edge{position:absolute;left:820px;top:440px;width:140px;height:120px;background:white}
#overflow,#pseudo{position:absolute;left:-100px;top:-100px;width:0;height:0;overflow:visible}
#overflow>div{position:absolute;left:420px;top:400px;width:180px;height:100px;background:white}
#pseudo::before{content:'';position:absolute;left:420px;top:540px;width:180px;height:100px;background:white}
@keyframes overflowFade{0%,70%{opacity:0}100%{opacity:1}}
#pseudo.active::before{animation:overflowFade 900ms linear both}
#boxshadow,#dropshadow{position:absolute;left:-200px;width:180px;height:100px;background:white}
#boxshadow{top:600px;box-shadow:520px 0 0 white}
#dropshadow{top:740px;filter:drop-shadow(520px 0 0 white)}
</style></head><body><div id="fade" class="card"></div><div id="finite" class="card"></div>
<div id="shadow"></div><div id="unrelated"></div><div id="edge"></div>
<div id="overflow"><div></div></div><div id="pseudo"></div>
<div id="boxshadow"></div><div id="dropshadow"></div><script>
const fade=document.getElementById('fade');
new IntersectionObserver(entries=>{if(entries.some(e=>e.isIntersecting))fade.classList.add('seen')}).observe(fade);
const root=document.getElementById('shadow').attachShadow({mode:'open'});
root.innerHTML='<div id="inside" style="width:180px;height:140px;background:white"></div>';
window.unrelated=document.getElementById('unrelated').animate([{opacity:.6},{opacity:1}],{duration:300,iterations:Infinity});
window.snapshot=()=>{
 const animation=fade.getAnimations().find(a=>a.animationName==='enter');
 return {same:animation===window.baselineAnimation,start:animation?.startTime,current:animation?.currentTime,
  state:animation?.playState,width:innerWidth,height:innerHeight,dpr:devicePixelRatio,x:scrollX,y:scrollY};
};
</script></body></html>`)
	}))
	defer page.Close()
	dir := filepath.Join(t.TempDir(), "recording")
	c := liveRecordingChrome(t, dir, -1)
	stopDeadline := context.AfterFunc(ctx, c.proc.Kill)
	defer stopDeadline()
	session := (liveHarness{c}).session(t, "")
	call := func(method string, params, result any) {
		t.Helper()
		if err := c.conn.Call(ctx, session, method, params, result); err != nil {
			t.Fatalf("%s: %v", method, err)
		}
	}
	eval := func(expression string, out any) {
		t.Helper()
		var response struct {
			Result struct {
				Value json.RawMessage `json:"value"`
			} `json:"result"`
			Exception json.RawMessage `json:"exceptionDetails"`
		}
		call("Runtime.evaluate", map[string]any{"expression": expression, "returnByValue": true, "awaitPromise": true}, &response)
		if len(response.Exception) > 0 {
			t.Fatalf("evaluate %s: %s", expression, response.Exception)
		}
		if out != nil {
			if err := json.Unmarshal(response.Result.Value, out); err != nil {
				t.Fatalf("evaluate %s: %s: %v", expression, response.Result.Value, err)
			}
		}
	}
	tick := time.NewTicker(25 * time.Millisecond)
	defer tick.Stop()
	wait := func() {
		t.Helper()
		select {
		case <-ctx.Done():
			t.Fatal("capture readiness fixture: ", ctx.Err())
		case <-tick.C:
		}
	}
	call("Emulation.setDeviceMetricsOverride", map[string]any{"width": 864, "height": 997, "deviceScaleFactor": 2, "mobile": false}, nil)
	call("Page.navigate", map[string]any{"url": page.URL + "/"}, nil)
	for {
		var ready bool
		eval(`document.readyState==='complete' && typeof snapshot==='function' && document.getElementById('fade').getAnimations().some(a=>a.playState==='finished')`, &ready)
		if ready {
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
			if state.Top != nil && state.Live && state.URL == page.URL+"/" {
				document = state.DocSeg
			}
		}
		if document == 0 {
			wait()
		}
	}
	capture := func(box recording.Box) recorder.CapturedCrop {
		t.Helper()
		crops, err := c.CaptureCrops(ctx, document, []core.CaptureRegion{{Box: box, Region: box}}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(crops) != 1 {
			t.Fatalf("crops=%d, want one", len(crops))
		}
		return crops[0]
	}
	bright := func(crop recorder.CapturedCrop, box recording.Box) {
		t.Helper()
		if crop.Unavailable != "" || crop.Box != box || crop.Document != document {
			t.Fatalf("crop unavailable=%q box=%v document=%d", crop.Unavailable, crop.Box, crop.Document)
		}
		image, err := png.Decode(bytes.NewReader(crop.PNG))
		if err != nil {
			t.Fatal(err)
		}
		bounds := image.Bounds()
		if bounds.Dx() != int(box[2]*2) || bounds.Dy() != int(box[3]*2) {
			t.Fatalf("DPR2 crop dimensions=%v, box=%v", bounds, box)
		}
		white := 0
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				r, g, b, a := image.At(x, y).RGBA()
				if r >= 0xf000 && g >= 0xf000 && b >= 0xf000 && a == 0xffff {
					white++
				}
			}
		}
		if white*10 < bounds.Dx()*bounds.Dy()*9 {
			t.Fatalf("bright pixels=%d/%d: screenshot restarted the fade or used wrong geometry", white, bounds.Dx()*bounds.Dy())
		}
	}
	settled := func(crop recorder.CapturedCrop) {
		t.Helper()
		if crop.Settlement == nil || crop.Settlement.Status != "settled" || crop.Settlement.Reason != "" {
			t.Fatalf("expected settled evidence: %+v", crop.Settlement)
		}
	}
	type timeline struct {
		Same                     bool `json:"same"`
		Start, Current           *float64
		State                    string
		Width, Height, DPR, X, Y float64
	}
	var before, after timeline
	eval(`(()=>{window.baselineAnimation=document.getElementById('fade').getAnimations().find(a=>a.animationName==='enter');return snapshot()})()`, &before)
	if !before.Same || before.State != "finished" || before.Start == nil || before.Current == nil || before.Width != 864 || before.Height != 997 || before.DPR != 2 {
		t.Fatalf("baseline: %+v", before)
	}
	fadeBox := recording.Box{80, 100, 160, 70}
	crop := capture(fadeBox)
	// Observation covers all top-document and open-shadow animations, even
	// when an active target's rectangle is outside this crop.
	if crop.Settlement == nil || crop.Settlement.Status != "unsettled" || crop.Settlement.Reason != "infinite animation" || len(crop.PNG) == 0 {
		t.Fatalf("unrelated infinite animation evidence: %+v, PNG bytes=%d", crop.Settlement, len(crop.PNG))
	}
	eval(`unrelated.cancel()`, nil)
	crop = capture(fadeBox)
	eval(`snapshot()`, &after)
	if !after.Same || after.State != "finished" || after.Start == nil || after.Current == nil || math.Abs(*after.Start-*before.Start) > .01 || math.Abs(*after.Current-*before.Current) > .01 || after.Width != before.Width || after.Height != before.Height || after.DPR != before.DPR || after.X != before.X || after.Y != before.Y {
		if after.Start != nil && after.Current != nil {
			t.Logf("animation timing before start=%f current=%f; after start=%f current=%f", *before.Start, *before.Current, *after.Start, *after.Current)
		}
		t.Fatalf("screenshot restarted animation or changed viewport: before=%+v after=%+v", before, after)
	}
	settled(crop)
	bright(crop, fadeBox)

	// An active finite animation must finish naturally, including an open-shadow
	// target exposed by its ShadowRoot.getAnimations().
	for _, shadow := range []bool{false, true} {
		target, box := `document.getElementById('finite')`, (recording.Box{80, 300, 160, 70})
		if shadow {
			target, box = `document.getElementById('shadow').shadowRoot.getElementById('inside')`, (recording.Box{340, 100, 120, 70})
		}
		var active struct{ Running, Enumerated bool }
		eval(fmt.Sprintf(`(async()=>{const target=%s;window.finiteAnimation=target.animate([{opacity:0,transform:'translateY(12px)'},{opacity:1,transform:'translateY(0)'}],{duration:700,delay:100,fill:'both'});await finiteAnimation.ready;return {running:finiteAnimation.playState==='running',enumerated:target.getRootNode().getAnimations().includes(finiteAnimation)}})()`, target), &active)
		if !active.Running || !active.Enumerated {
			t.Fatalf("active finite fixture shadow=%v: %+v", shadow, active)
		}
		crop = capture(box)
		settled(crop)
		bright(crop, box)
		var finished bool
		eval(`finiteAnimation.playState==='finished' && finiteAnimation.effect.getComputedTiming().progress===1`, &finished)
		if !finished {
			var detail any
			eval(`({state:finiteAnimation.playState,time:finiteAnimation.currentTime,timing:finiteAnimation.effect.getComputedTiming(),unrelated:unrelated.playState})`, &detail)
			t.Fatalf("capture did not wait for natural completion shadow=%v: %+v", shadow, detail)
		}
	}
	// The animation target's own rectangle is not its painted footprint: a
	// zero-size offscreen parent changes its overflowing child's opacity, and
	// an overflowing pseudo-element paints beyond its zero-size host.
	for _, pseudo := range []bool{false, true} {
		box := recording.Box{340, 320, 120, 60}
		start := `const target=document.getElementById('overflow');window.overflowAnimation=target.animate([{opacity:0,offset:0},{opacity:0,offset:.7},{opacity:1}],{duration:900,fill:'both'});`
		if pseudo {
			box = recording.Box{340, 460, 120, 60}
			start = `const target=document.getElementById('pseudo');target.classList.add('active');window.overflowAnimation=document.getAnimations().find(a=>a.animationName==='overflowFade');`
		}
		var active bool
		eval(`(async()=>{`+start+`await overflowAnimation.ready;const r=target.getBoundingClientRect();return overflowAnimation.playState==='running' && r.width===0 && r.height===0 && r.right<0 && r.bottom<0})()`, &active)
		if !active {
			t.Fatalf("overflow animation fixture not active with offscreen zero-size target, pseudo=%v", pseudo)
		}
		crop = capture(box)
		var timing struct {
			State string
			Time  float64
		}
		eval(`({state:overflowAnimation.playState,time:overflowAnimation.currentTime})`, &timing)
		if timing.State != "finished" || timing.Time < 900 {
			t.Errorf("overflow capture preceded natural completion, pseudo=%v: %+v; settlement=%+v", pseudo, timing, crop.Settlement)
		}
		settled(crop)
		bright(crop, box)
	}
	// Leaf elements also paint outside their own geometry through box-shadow
	// and filters; opacity on either source must finish before the white crop.
	for _, fixture := range []struct {
		id  string
		box recording.Box
	}{
		{"boxshadow", recording.Box{340, 620, 120, 60}},
		{"dropshadow", recording.Box{340, 760, 120, 60}},
	} {
		var active bool
		eval(fmt.Sprintf(`(async()=>{const target=document.getElementById(%q);window.paintAnimation=target.animate([{opacity:0,offset:0},{opacity:0,offset:.7},{opacity:1}],{duration:900,fill:'both'});await paintAnimation.ready;return paintAnimation.playState==='running' && target.childNodes.length===0 && target.getBoundingClientRect().right<0})()`, fixture.id), &active)
		if !active {
			t.Fatalf("offscreen leaf animation fixture not active: %s", fixture.id)
		}
		crop = capture(fixture.box)
		var finished bool
		eval(`paintAnimation.playState==='finished' && paintAnimation.effect.getComputedTiming().progress===1`, &finished)
		if !finished {
			t.Errorf("capture preceded natural completion of %s; settlement=%+v", fixture.id, crop.Settlement)
		}
		settled(crop)
		bright(crop, fixture.box)
	}
	var running bool
	eval(`(async()=>{window.inBox=document.getElementById('fade').animate([{opacity:1},{opacity:.8}],{duration:300,iterations:Infinity});await inBox.ready;return inBox.playState==='running'})()`, &running)
	if !running {
		t.Fatal("in-box infinite animation did not start")
	}
	crop = capture(fadeBox)
	if crop.Settlement == nil || crop.Settlement.Status != "unsettled" || crop.Settlement.Reason != "infinite animation" || len(crop.PNG) == 0 {
		t.Fatalf("infinite animation evidence: %+v, PNG bytes=%d", crop.Settlement, len(crop.PNG))
	}
	eval(`inBox.cancel()`, nil)

	// Keep this fixture clear of the cast's bottom-right in-page control.
	crop = capture(recording.Box{840, 450, 80, 80})
	if !crop.ClippedToViewport {
		t.Fatal("partial crop omitted clipping evidence")
	}
	settled(crop)
	bright(crop, recording.Box{840, 450, 24, 80})
	crop = capture(recording.Box{900, 1050, 40, 40})
	if crop.Unavailable != "outside viewport" || len(crop.PNG) != 0 || crop.Box != (recording.Box{}) {
		t.Fatalf("wholly offscreen crop unavailable=%q box=%v PNG bytes=%d", crop.Unavailable, crop.Box, len(crop.PNG))
	}
}
