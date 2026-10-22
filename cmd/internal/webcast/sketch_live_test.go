package webcast

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// The real viewer/controller/Board/Writer/Hub seams, with two disposable
// headless Chrome profiles. The watch viewer sees the open pressure path
// before pointer-up, and the same accepted recording replays independently.
func TestSketchLiveTwoViewers(t *testing.T) {
	a, b := headlessPage(t), headlessPage(t)
	f := newSketchHTTPFixture(t)
	srv := httptest.NewServer(f.s)
	t.Cleanup(func() { f.hub.Close(); srv.CloseClientConnections(); srv.Close() })
	for i, p := range []*chromePage{a, b} {
		link := "driver"
		if i == 1 {
			link = "pointer"
		}
		// Preserve the real native EventSource so outage conformance invokes the
		// production onerror path without replacing the page, model or server.
		p.call(t, p.session, "Page.enable", nil, nil)
		p.call(t, p.session, "Page.addScriptToEvaluateOnNewDocument", map[string]any{"source": `(()=>{const NativeEventSource=window.EventSource;window.EventSource=class extends NativeEventSource{constructor(...args){super(...args);window.__testFeedbackSource=this;window.__testFeedbackFullMasks=0;this.addEventListener('feedback',()=>window.__testFeedbackFullMasks++);}};})()`}, nil)
		p.call(t, p.session, "Page.navigate", map[string]any{"url": srv.URL + "/" + link + "/"}, nil)
		waitEval(t, p, "shared sketch mount", `String(!!window.couchViewer?.sketch)`, "true")
		p.eval(t, `(async()=>{await couchViewer.ready();globalThis.__sketchSeen={};globalThis.__sketchProjection=null;couchViewer.onRecord((r,m)=>{if(r.k.startsWith('sketch-')){if(r.operation){const key=r.k+':'+r.operation;__sketchSeen[key]=(__sketchSeen[key]||0)+1;}globalThis.__sketchProjection=m.sketchProjection();}});return '';})()`)
	}
	seat := a.evalString(t, `couchViewer.drive.seat.viewer`)
	if seat == "" {
		t.Fatal("driver viewer has no seat")
	}
	a.eval(t, `(async()=>{await couchViewer.drive.control('request');return '';})()`)
	if err := f.ctrl.Grant(seat); err != nil {
		t.Fatal(err)
	}
	waitEval(t, a, "current holder drawing authority", `String(couchViewer.sketch.state().allowed)`, "true")
	if got := b.evalString(t, `String(couchViewer.sketch.state().allowed)`); got != "false" {
		t.Fatalf("pointer watcher may draw: %s", got)
	}
	b.eval(t, `(async()=>{await couchViewer.sketch.enter();return '';})()`)
	for _, r := range f.mem.Records() {
		if r.Kind() == "sketch-begin" {
			t.Fatal("pointer watcher emitted begin")
		}
	}
	a.eval(t, `(async()=>{await couchViewer.sketch.enter();return '';})()`)
	waitEval(t, a, "acknowledged held scene", `couchViewer.sketch.state().phase`, "drawing")
	var client [2]float64
	raw := a.eval(t, fmt.Sprintf(`(()=>{const r=couchViewer.boxOf(%d);return couchViewer.fromPage(r[0]+20,r[1]+r[3]/2);})()`, f.node))
	if err := json.Unmarshal(raw, &client); err != nil {
		t.Fatal(err)
	}
	pen := func(kind string, x, pressure float64, buttons int) {
		t.Helper()
		button := "none"
		if kind != "mouseMoved" {
			button = "left"
		}
		a.call(t, a.session, "Input.dispatchMouseEvent", map[string]any{"type": kind, "x": x, "y": client[1], "button": button, "buttons": buttons, "clickCount": 1, "pointerType": "pen", "force": pressure}, nil)
	}
	pen("mousePressed", client[0], .2, 1)
	pen("mouseMoved", client[0]+15, .5, 1)
	pen("mouseMoved", client[0]+30, .8, 1)
	waitEval(t, b, "incomplete streamed pressure path", `(()=>{const s=globalThis.__sketchProjection?.sketches[0]?.strokes[0];return s&&s.pts.length>=3&&!s.sealed?'open':'waiting';})()`, "open")
	var open recording.SketchProjection
	if err := json.Unmarshal(b.eval(t, `globalThis.__sketchProjection`), &open); err != nil {
		t.Fatal(err)
	}
	if len(open.Sketches) != 1 || len(open.Sketches[0].Strokes) != 1 {
		t.Fatalf("open projection %+v", open)
	}
	points := open.Sketches[0].Strokes[0].Pts
	if math.Abs(points[0][2]-.2) > .02 || math.Abs(points[len(points)-1][2]-.8) > .02 {
		t.Fatalf("pen pressures lost: %v", points)
	}
	for _, r := range f.mem.Records() {
		if r.Kind() == "sketch-stroke-end" {
			t.Fatal("pointerup occurred before open-path assertion")
		}
	}
	waitEval(t, b, "pressure canvas on watch viewer", `(()=>{const c=document.querySelector('.sketch-observer'),pixels=c.getContext('2d').getImageData(0,0,c.width,c.height).data;for(let i=3;i<pixels.length;i+=4)if(pixels[i])return 'painted';return 'blank';})()`, "painted")
	pen("mouseReleased", client[0]+30, 0, 0)
	waitEval(t, b, "sealed pressure path", `String(!!globalThis.__sketchProjection?.sketches[0]?.strokes[0]?.sealed)`, "true")
	if got := b.evalString(t, `String(Math.max(...Object.values(__sketchSeen)))`); got != "1" {
		t.Fatalf("echo duplicated accepted operations: %s", got)
	}
	waitEval(t, a, "confirmed stroke enables Undo", `String([...document.querySelectorAll('.sketch-toolbar button')].some(b=>b.textContent==='Undo'&&!b.disabled))`, "true")
	a.eval(t, `(async()=>{await couchViewer.sketch.undo();return '';})()`)
	waitEval(t, b, "undo streamed", `String(__sketchProjection.sketches[0].visible.length)`, "0")
	waitEval(t, b, "undo erases accepted ink", `(()=>{const c=document.querySelector('.sketch-observer'),pixels=c.getContext('2d').getImageData(0,0,c.width,c.height).data;for(let i=3;i<pixels.length;i+=4)if(pixels[i])return 'painted';return 'blank';})()`, "blank")
	waitEval(t, a, "confirmed Undo enables Redo", `String([...document.querySelectorAll('.sketch-toolbar button')].some(b=>b.textContent==='Redo'&&!b.disabled))`, "true")
	a.eval(t, `(async()=>{await couchViewer.sketch.redo();return '';})()`)
	waitEval(t, b, "redo streamed", `String(__sketchProjection.sketches[0].visible.length)`, "1")
	waitEval(t, b, "redo restores accepted ink", `(()=>{const c=document.querySelector('.sketch-observer'),pixels=c.getContext('2d').getImageData(0,0,c.width,c.height).data;for(let i=3;i<pixels.length;i+=4)if(pixels[i])return 'painted';return 'blank';})()`, "painted")
	waitEval(t, a, "confirmed Redo before labeling", `String(couchViewer.sketch.state().slots.every(s=>!s.inflight&&!s.queued))`, "true")
	a.eval(t, `(async()=>{await couchViewer.sketch.leave('watch');couchViewer.sketch.labelInput.value='Move this control';await couchViewer.sketch.saveLabel();return '';})()`)
	waitEval(t, b, "required label streamed", `__sketchProjection.sketches[0].label`, "Move this control")
	records := f.mem.Records()
	reduced, err := recording.Reduce(records, math.MaxInt64)
	if err != nil {
		t.Fatal(err)
	}
	expected, err := json.Marshal(reduced.Sketches.Snapshot())
	if err != nil {
		t.Fatal(err)
	}
	compare := func(what string, got []byte) {
		t.Helper()
		var wantValue, gotValue any
		if err := json.Unmarshal(expected, &wantValue); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(got, &gotValue); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(wantValue, gotValue) {
			t.Fatalf("%s differs from recording reducer\nwant %s\ngot %s", what, expected, got)
		}
	}
	compare("watch viewer", b.eval(t, `__sketchProjection`))

	// The same Board display mask travels over real SSE to both viewers. It
	// changes neither their reducer projections nor the accepted recording.
	selectedForMask, err := f.bd.Selection()
	if err != nil || len(selectedForMask.Sketches) != 1 {
		t.Fatalf("finished mask selection: %+v %v", selectedForMask, err)
	}
	if err = f.bd.SetDeliveredFeedback(selectedForMask.Sketches, nil); err != nil {
		t.Fatal(err)
	}
	canvasPaint := `(()=>{const c=document.querySelector('.sketch-observer'),pixels=c.getContext('2d').getImageData(0,0,c.width,c.height).data;for(let i=3;i<pixels.length;i+=4)if(pixels[i])return 'painted';return 'blank';})()`
	for i, p := range []*chromePage{a, b} {
		waitEval(t, p, fmt.Sprintf("submitted ink hidden on viewer %d", i), canvasPaint, "blank")
	}
	compare("masked driving viewer", a.eval(t, `__sketchProjection`))
	compare("masked watching viewer", b.eval(t, `__sketchProjection`))
	afterMask, err := recording.Reduce(f.mem.Records(), math.MaxInt64)
	if err != nil {
		t.Fatal(err)
	}
	afterProjection, err := json.Marshal(afterMask.Sketches.Snapshot())
	if err != nil {
		t.Fatal(err)
	}
	compare("recording after live mask", afterProjection)
	// A view-only link also observes historical paths but cannot start one.
	b.call(t, b.session, "Page.navigate", map[string]any{"url": srv.URL + "/view/"}, nil)
	waitEval(t, b, "view-only sketch observer", `String(!!window.couchViewer?.sketch)`, "true")
	if got := b.evalString(t, `String(couchViewer.sketch.state().allowed)`); got != "false" {
		t.Fatalf("view-only watcher may draw: %s", got)
	}
	b.eval(t, `(async()=>{await couchViewer.sketch.enter();return '';})()`)
	count := 0
	for _, r := range f.mem.Records() {
		if r.Kind() == "sketch-begin" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("view-only watcher changed sketch count: %d", count)
	}
	waitEval(t, b, "submitted historical path hidden after view-only reconnect", `(()=>{const c=document.querySelector('.sketch-observer'),pixels=c.getContext('2d').getImageData(0,0,c.width,c.height).data;for(let i=3;i<pixels.length;i+=4)if(pixels[i])return 'painted';return 'blank';})()`, "blank")

	// An EventSource outage retains this page and reducer. Rotate stream
	// ownership twice through its actual error handler; before any new full mask
	// arrives, the confirmed ink must stay absent on both native canvases.
	for i, p := range []*chromePage{a, b} {
		got := p.evalString(t, `(async()=>{const source=window.__testFeedbackSource,modelPage=document.documentElement,fullMasks=window.__testFeedbackFullMasks;if(!source||source.readyState!==EventSource.OPEN||typeof source.onerror!=='function'||fullMasks<1)throw Error('native feedback source unavailable '+JSON.stringify({source:!!source,state:source?.readyState,open:EventSource.OPEN,error:typeof source?.onerror,fullMasks}));source.dispatchEvent(new Event('error'));source.dispatchEvent(new Event('error'));await new Promise(resolve=>requestAnimationFrame(()=>requestAnimationFrame(resolve)));return String(window.__testFeedbackSource===source&&document.documentElement===modelPage&&window.__testFeedbackFullMasks===fullMasks);})()`)
		if got != "true" {
			t.Fatalf("viewer %d replaced page/stream or received a new full mask during outage check: %s", i, got)
		}
		waitEval(t, p, fmt.Sprintf("submitted ink stays hidden during viewer %d outage", i), canvasPaint, "blank")
	}
	compare("retained driving viewer projection during outage", a.eval(t, `__sketchProjection`))
	afterOutage, err := recording.Reduce(f.mem.Records(), math.MaxInt64)
	if err != nil {
		t.Fatal(err)
	}
	afterOutageProjection, err := json.Marshal(afterOutage.Sketches.Snapshot())
	if err != nil {
		t.Fatal(err)
	}
	compare("recording after retained-page outage", afterOutageProjection)
	// The reconnect mask is live display state, so replay below must still paint.
	// Persist exactly the accepted wire, then let the real replay viewer reduce
	// and display it in a fresh page. No source browser or live Board participates.
	root := t.TempDir()
	dir := filepath.Join(root, "sketch")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	var events bytes.Buffer
	for _, r := range records {
		line, e := recording.Encode(r)
		if e != nil {
			t.Fatal(e)
		}
		events.Write(line)
		events.WriteByte('\n')
	}
	if err := os.WriteFile(filepath.Join(dir, recording.FileEvents), events.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	replay := httptest.NewServer(NewReplayServer("replay", root))
	t.Cleanup(replay.Close)
	b.call(t, b.session, "Page.navigate", map[string]any{"url": replay.URL + "/replay/r/sketch/"}, nil)
	waitEval(t, b, "recording replay", `String(!!window.couchViewer?.sketch)`, "true")
	// Replay's public seek API drives the same mounted Model/Renderer path;
	// observe its accepted records after a restart from the beginning.
	b.eval(t, `(async()=>{await couchViewer.ready();globalThis.__replayProjection=null;couchViewer.onRecord((r,m)=>{if(r.k.startsWith('sketch-'))globalThis.__replayProjection=m.sketchProjection();});location.hash='t=1000000000000';return '';})()`)
	waitEval(t, b, "replayed final label", `globalThis.__replayProjection?.sketches[0]?.label||''`, "Move this control")
	compare("recording replay", b.eval(t, `__replayProjection`))
	waitEval(t, b, "replay paints historical accepted ink", `(()=>{const c=document.querySelector('.sketch-observer'),pixels=c.getContext('2d').getImageData(0,0,c.width,c.height).data;for(let i=3;i<pixels.length;i+=4)if(pixels[i])return 'painted';return 'blank';})()`, "painted")
	t.Logf("open pressure points=%d; one echo; undo/redo/label and replay projections agree", len(points))
}
