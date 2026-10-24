package webcast

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// TestCastNotesPanelLive uses native hit testing, pointer capture and wheel
// input against the shipped module/CSS. Its deadline also kills Chrome during
// cleanup, before go test's outer watchdog can bypass profile cleanup.
func TestCastNotesPanelLive(t *testing.T) {
	if os.Getenv("WEBCAST_LIVE_CHROME") == "" {
		t.Skip("set WEBCAST_LIVE_CHROME=1 to drive real headless Chrome")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 75*time.Second)
	defer cancel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.FileServer(http.FS(webFS)).ServeHTTP(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		// The standalone panel fixture excludes the launcher control, which
		// otherwise overlaps its controls at the deliberately tiny viewport.
		fmt.Fprint(w, `<!doctype html><meta charset="utf-8"><link rel="stylesheet" href="/web/annotate.css"><link rel="stylesheet" href="/web/sketch.css"><link rel="stylesheet" href="/web/feedback-panel.css">
<style>couchcast-control{display:none!important}html,body{margin:0;width:100%;height:100%;overflow:hidden}#page{position:fixed;width:110px;height:44px}</style>
<button id="page">Previously covered page</button><script type="module">
import {mountFeedbackPanel} from '/web/feedback-panel.js';
import {paintSketchPaths} from '/web/sketch-render.js';
import {sketchNoteEntries} from '/web/sketch-notes.js';
window.pageClicks=0;window.rowClicks=0;document.querySelector('#page').onclick=()=>pageClicks++;
window.fixtureNotes=count=>Array.from({length:count},(_,i)=>({id:'n'+i,text:'Note '+i+' '+('Long note text with wrapping. '.repeat(12))}));
window.noteActions={open:()=>rowClicks++};
window.resetPanel=()=>{window.panel?.dispose();pageClicks=0;rowClicks=0;
window.panel=mountFeedbackPanel(document,window,{parent:document.body});panel.setNotes(fixtureNotes(60),noteActions);window.ready=true;};resetPanel();
window.showSketchControls=(paths,targets)=>{
 window.panel?.dispose();document.querySelector('#sketch-fixture')?.remove();
 const root=document.createElement('div');root.id='sketch-fixture';root.style.cssText='position:fixed;inset:0;pointer-events:none';document.body.append(root);
 const canvas=document.createElement('canvas');canvas.width=innerWidth;canvas.height=innerHeight;root.append(canvas);
 const bounds=paths.map((p,i)=>paintSketchPaths(canvas.getContext('2d'),[p],{project:()=>targets[i]}));
 window.panel=mountFeedbackPanel(document,window,{parent:document.body});
 panel.setNotes([{id:paths[0].sketch,text:'Accepted annotation sharing a sketch identity'}],noteActions);
 panel.setSketches(sketchNoteEntries(paths,bounds,{metadata:key=>({phase:'complete',label:'Final visible sketch '+key}),editable:new Set(),viewport:{width:innerWidth,height:innerHeight}}),{open:()=>{throw Error('final labels cannot mutate')}});
};
</script>`)
	}))
	defer srv.Close()
	binary, err := FindChrome(os.Getenv, execLookPath, fileExists)
	if err != nil {
		t.Fatal(err)
	}
	c, err := Launch(ctx, LaunchOptions{Binary: binary, ProfileDir: t.TempDir(), Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	stopWatchdog := context.AfterFunc(ctx, c.proc.Kill)
	defer stopWatchdog()
	defer c.Close()
	session := (liveHarness{c}).session(t, "")
	call := func(method string, params, result any) {
		t.Helper()
		if err := c.conn.Call(ctx, session, method, params, result); err != nil {
			t.Fatalf("%s: %v", method, err)
		}
	}
	eval := func(expression string, out any) {
		t.Helper()
		var result struct {
			Result struct {
				Value json.RawMessage `json:"value"`
			} `json:"result"`
			Exception json.RawMessage `json:"exceptionDetails"`
		}
		call("Runtime.evaluate", map[string]any{"expression": expression, "returnByValue": true, "awaitPromise": true}, &result)
		if len(result.Exception) > 0 {
			t.Fatalf("JavaScript %s: %s", expression, result.Exception)
		}
		if out != nil {
			if err := json.Unmarshal(result.Result.Value, out); err != nil {
				t.Fatalf("decode %s: %s: %v", expression, result.Result.Value, err)
			}
		}
	}
	wait := func(expression string) {
		t.Helper()
		for {
			var ok bool
			eval(expression, &ok)
			if ok {
				return
			}
			select {
			case <-ctx.Done():
				t.Fatal("waiting for ", expression, ": ", ctx.Err())
			case <-time.After(20 * time.Millisecond):
			}
		}
	}
	require := func(expression string) {
		t.Helper()
		var ok bool
		eval(expression, &ok)
		if !ok {
			var details string
			eval(`JSON.stringify({root:panel.root.getBoundingClientRect().toJSON(),body:panel.body.getBoundingClientRect().toJSON(),scroll:panel.body.scrollTop,mode:panel.root.className,viewport:[innerWidth,innerHeight],pageClicks,rowClicks,controls:[...document.querySelectorAll('.floating-panel-header button')].map(e=>{const r=e.getBoundingClientRect();return {class:e.className,rect:r.toJSON(),hit:document.elementFromPoint(r.x+r.width/2,r.y+r.height/2)?.tagName};})})`, &details)
			t.Fatalf("native assertion %s: %s", expression, details)
		}
	}
	rect := func(selector string) map[string]float64 {
		t.Helper()
		var value map[string]float64
		eval(fmt.Sprintf(`document.querySelector(%q).getBoundingClientRect().toJSON()`, selector), &value)
		return value
	}
	mouse := func(typ string, x, y float64) {
		t.Helper()
		params := map[string]any{"type": typ, "x": x, "y": y, "button": "left", "clickCount": 1}
		if typ != "mouseReleased" {
			params["buttons"] = 1
		}
		call("Input.dispatchMouseEvent", params, nil)
	}
	click := func(selector string) {
		t.Helper()
		r := rect(selector)
		mouse("mousePressed", r["x"]+r["width"]/2, r["y"]+r["height"]/2)
		mouse("mouseReleased", r["x"]+r["width"]/2, r["y"]+r["height"]/2)
	}
	resize := func(width, height int) {
		t.Helper()
		call("Emulation.setDeviceMetricsOverride", map[string]any{"width": width, "height": height, "deviceScaleFactor": 1, "mobile": false}, nil)
		wait(fmt.Sprintf("innerWidth===%d && innerHeight===%d", width, height))
		eval(`new Promise(resolve=>requestAnimationFrame(()=>requestAnimationFrame(resolve)))`, nil)
	}
	call("Page.navigate", map[string]any{"url": srv.URL}, nil)
	wait("window.ready===true")
	for _, width := range []int{1280, 320} {
		t.Logf("native viewport width=%d", width)
		resize(width, 800)
		if width == 320 {
			resize(width, 480)
		}
		eval("resetPanel()", nil)
		expectedWidth := math.Min(300, float64(width)-24)
		r := rect(".floating-notes")
		if math.Abs(r["width"]-expectedWidth) > .1 || math.Abs(r["height"]-320) > .1 {
			t.Fatalf("panel dimensions: %+v", r)
		}
		require(`getComputedStyle(panel.root).position==='fixed' && getComputedStyle(panel.root).backgroundColor==='rgb(255, 245, 157)' && panel.body.scrollHeight>panel.body.clientHeight`)
		header := rect(".floating-panel-header")
		require(`document.documentElement.scrollWidth===innerWidth && document.documentElement.scrollHeight===innerHeight`)
		eval(`document.querySelector('.feedback-row .feedback-open').scrollIntoView({block:'center'})`, nil)
		click(".feedback-row .feedback-open")
		require("rowClicks===1")
		rowAnchor := rect(".floating-notes")
		if rowAnchor["left"] != r["left"] || rowAnchor["top"] != r["top"] {
			t.Fatalf("ordinary row click dragged panel: before=%v after=%v", r, rowAnchor)
		}
		eval(`panel.body.scrollTop=0`, nil)
		b := rect(".floating-panel-body")
		call("Input.dispatchMouseEvent", map[string]any{"type": "mouseWheel", "x": b["x"] + b["width"]/2, "y": b["y"] + b["height"]/2, "deltaX": 0, "deltaY": 500}, nil)
		wait("panel.body.scrollTop>0")
		after := rect(".floating-panel-header")
		if after["top"] != header["top"] {
			t.Fatalf("wheel moved fixed header: before=%v after=%v", header, after)
		}
		require("scrollY===0")
		// Drag the real header and require actual browser capture, not synthetic events.
		handle := rect(".floating-panel-drag")
		x, y := handle["x"]+20, handle["y"]+20
		mouse("mousePressed", x, y)
		mouse("mouseMoved", x, y-40)
		require(`document.querySelector('.floating-panel-header').hasPointerCapture(1)`)
		mouse("mouseReleased", x, y-40)
		moved := rect(".floating-notes")
		if math.Abs(moved["top"]-(r["top"]-40)) > .1 {
			t.Fatalf("header drag did not move anchor: before=%v after=%v", r, moved)
		}
		eval(`(()=>{const r=panel.root.getBoundingClientRect();document.querySelector('#page').style.left=(r.left+10)+'px';document.querySelector('#page').style.top=(r.top+80)+'px';})()`, nil)
		require(`(()=>{const r=document.querySelector('#page').getBoundingClientRect();return panel.root.contains(document.elementFromPoint(r.x+20,r.y+20));})()`)
		click(".floating-panel-minimize")
		require(`panel.root.classList.contains('minimized') && panel.body.hidden`)
		click("#page")
		require("pageClicks===1")
		small := rect(".floating-panel-restore")
		x, y = small["x"]+20, small["y"]+20
		dx := -120.0
		if width == 320 {
			dx = 100
		}
		mouse("mousePressed", x, y)
		mouse("mouseMoved", x+dx, y+25)
		require(`document.querySelector('.floating-panel-restore').hasPointerCapture(1)`)
		mouse("mouseReleased", x+dx, y+25)
		require(`panel.root.classList.contains('minimized')`)
		small = rect(".floating-notes")
		if math.Abs(small["left"]-moved["left"]) < 20 {
			t.Fatalf("minimized drag did not move: before=%v after=%v", moved, small)
		}
		eval(`(()=>{const b=document.querySelector('.floating-panel-restore');for(let i=0;i<200;i++)b.dispatchEvent(new KeyboardEvent('keydown',{key:'ArrowRight',shiftKey:true,bubbles:true}));panel.setNotes(fixtureNotes(100),noteActions);})()`, nil)
		require(`(()=>{const r=panel.root.getBoundingClientRect();return panel.root.classList.contains('minimized')&&r.right<=innerWidth&&r.left>=0})()`)
		click(".floating-panel-restore")
		require(`!panel.root.classList.contains('minimized') && !panel.body.hidden`)
		restored := rect(".floating-notes")
		if math.Abs(restored["top"]-small["top"]) > .1 {
			t.Fatalf("restore lost shared vertical anchor: %v %v", small, restored)
		}
		// A real lost-capture event must restore the pre-drag anchor.
		handle = rect(".floating-panel-drag")
		x, y = handle["x"]+20, handle["y"]+20
		mouse("mousePressed", x, y)
		mouse("mouseMoved", x, y+30)
		eval(`document.querySelector('.floating-panel-header').releasePointerCapture(1)`, nil)
		mouse("mouseMoved", x, y+31)
		mouse("mouseReleased", x, y+31)
		canceled := rect(".floating-notes")
		if math.Abs(canceled["top"]-restored["top"]) > .1 {
			t.Fatalf("lost capture retained tentative anchor: %v %v", restored, canceled)
		}
		eval(`document.querySelector('.floating-panel-drag').focus()`, nil)
		call("Input.dispatchKeyEvent", map[string]any{"type": "keyDown", "key": "ArrowDown", "code": "ArrowDown", "windowsVirtualKeyCode": 40}, nil)
		call("Input.dispatchKeyEvent", map[string]any{"type": "keyUp", "key": "ArrowDown", "code": "ArrowDown", "windowsVirtualKeyCode": 40}, nil)
		keyed := rect(".floating-notes")
		if math.Abs(keyed["top"]-(canceled["top"]+10)) > .1 {
			t.Fatalf("keyboard did not move anchor: %v %v", canceled, keyed)
		}
		// Resize in mid-gesture; both controls must remain hit-test reachable.
		handle = rect(".floating-panel-drag")
		x, y = handle["x"]+20, handle["y"]+20
		mouse("mousePressed", x, y)
		mouse("mouseMoved", x, y+20)
		resize(180, 130)
		mouse("mouseReleased", 30, 30)
		require(`(()=>{const r=panel.root.getBoundingClientRect();return r.left>=0 && r.top>=0 && r.right<=innerWidth && r.bottom<=innerHeight && !document.querySelector('.floating-panel-header').hasPointerCapture(1);})()`)
		require(`(()=>{const e=document.querySelector('.floating-panel-minimize'),r=e.getBoundingClientRect();return document.elementFromPoint(r.x+r.width/2,r.y+r.height/2)===e;})()`)
		click(".floating-panel-minimize")
		require(`(()=>{const e=document.querySelector('.floating-panel-restore'),r=e.getBoundingClientRect();return r.width>0 && r.height>0 && r.right<=innerWidth && r.bottom<=innerHeight && document.elementFromPoint(r.x+r.width/2,r.y+r.height/2)===e;})()`)
		eval("panel.dispose()", nil)
		require(`!document.querySelector('.floating-notes')`)
	}
	// Native scroll/multipath/edge proof uses bounds from the real painter.
	resize(320, 480)
	eval(`(()=>{const p={sketch:'s',id:'p',width:4,color:'#ff0000',at:{box:[0,0,100,800]},pts:[[30,100,.5],[60,700,.5]]};showSketchControls([p],[ [0,-500,100,800] ]);})()`, nil)
	reachable := `(()=>{const e=document.querySelector('.floating-panel-minimize'),r=e?.getBoundingClientRect();return !!r&&r.width>0&&r.height>0&&r.left>=0&&r.top>=0&&r.right<=innerWidth&&r.bottom<=innerHeight&&e===document.elementFromPoint(r.x+r.width/2,r.y+r.height/2)})()`
	require(reachable)
	click(`.feedback-row[data-key="sketch:s"] .feedback-open`)
	require(`document.querySelector('.feedback-readonly-text').textContent==='Final visible sketch s'&&!document.querySelector('.feedback-readonly').hidden`)
	click(".feedback-readonly-close")
	for _, corner := range [][2]int{{0, 0}, {319, 0}, {0, 479}, {319, 479}} {
		eval(fmt.Sprintf(`(()=>{const off={sketch:'s',id:'off',width:4,at:{box:[0,0,320,480]},pts:[[-100,-100,.5]]};const visible={...off,id:'visible',pts:[[%d,%d,.5]]};showSketchControls([off,visible],[[0,0,320,480],[0,0,320,480]]);})()`, corner[0], corner[1]), nil)
		require(reachable)
	}
	// Identical/partially overlapping/edge-clamped shapes share one Notes
	// panel with annotations; every latest identity remains selectable.
	eval(`(()=>{const paths=Array.from({length:24},(_,i)=>({sketch:'s'+i,id:'p',width:4,at:{box:[0,0,320,480]},pts:[[(i%3)*2,(i%4)*2,.5]]}));showSketchControls(paths,paths.map(()=>[0,0,320,480]));})()`, nil)
	require(`document.querySelectorAll('.feedback-panel').length===1&&document.querySelectorAll('.feedback-row').length===25&&document.querySelector('.floating-panel-drag').textContent==='Notes (25)'&&!!document.querySelector('.feedback-row[data-key="note:s0"]')&&!document.querySelector('.sketch-notes-summary,.sketch-note-control')`)
	require(reachable)
	for i := 0; i < 24; i++ {
		selector := fmt.Sprintf(`.feedback-row[data-key="sketch:s%d"] .feedback-open`, i)
		eval(fmt.Sprintf(`document.querySelector(%q).scrollIntoView({block:'center'})`, selector), nil)
		require(fmt.Sprintf(`(()=>{const e=document.querySelector(%q),r=e.getBoundingClientRect();return r.left>=0&&r.top>=0&&r.right<=innerWidth&&r.bottom<=innerHeight&&document.elementFromPoint(r.x+r.width/2,r.y+r.height/2)===e})()`, selector))
		click(selector)
		require(fmt.Sprintf(`document.querySelector('.feedback-readonly-text').textContent===%q&&!document.querySelector('.feedback-readonly').hidden`, fmt.Sprintf("Final visible sketch s%d", i)))
		click(".feedback-readonly-close")
	}
	eval(`panel.dispose();document.querySelector('#sketch-fixture').remove()`, nil)

}
