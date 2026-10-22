package webcast

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	core "github.com/xianxu/pair/cmd/internal/submission"
	castassets "github.com/xianxu/pair/cmd/internal/webcast/assets"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/recorder"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// The actual console widget, trusted binding worker, Board and recording
// actor run together. The pen path must be durable before pointer-up.
func TestOperatorSketchLiveDrawing(t *testing.T) {
	if os.Getenv("WEBCAST_LIVE_CHROME") == "" {
		t.Skip("set WEBCAST_LIVE_CHROME=1")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	page := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/card.svg" {
			w.Header().Set("Content-Type", "image/svg+xml")
			fmt.Fprint(w, `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20"><rect width="20" height="20" fill="purple"/></svg>`)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<!doctype html><title>console sketch integration</title><style>body{margin:0}button{position:absolute;left:80px;top:100px;width:240px;height:160px;padding:0;border:0;background:#eee}</style><button>Save</button><img src="/card.svg" style="position:absolute;left:600px;top:500px">`)
	}))
	defer page.Close()
	bin, err := FindChrome(os.Getenv, execLookPath, fileExists)
	if err != nil {
		t.Fatal(err)
	}
	profile, root := t.TempDir(), filepath.Join(t.TempDir(), "recordings")
	var chrome *Chrome
	cast, err := StartCast(ctx, CastConfig{RecordingsDir: root, Host: broadcast.NewHost(broadcast.HostConfig{Tunnel: &broadcast.FakeTunnel{}}), Logf: t.Logf,
		NewAssets: func(w recording.AssetWriter) castassets.Served {
			return castassets.NewRecorded(w, castassets.DefaultPolicy)
		},
		Launch: func(ctx context.Context, opts *RecordOptions) (Browser, error) {
			opts.ShotEvery = -1
			var e error
			chrome, e = Launch(ctx, LaunchOptions{Binary: bin, Headless: true, ProfileDir: profile, StartURL: page.URL, Record: opts})
			return chrome, e
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer cast.End()
	stop := context.AfterFunc(ctx, chrome.proc.Kill)
	defer stop()
	session := (liveHarness{chrome}).session(t, "")
	call := func(method string, params, result any) {
		t.Helper()
		if e := chrome.conn.Call(ctx, session, method, params, result); e != nil {
			t.Fatal(method, e)
		}
	}
	eval := func(world int64, expression string) json.RawMessage {
		t.Helper()
		var res struct {
			Result struct {
				Value json.RawMessage `json:"value"`
			} `json:"result"`
			Exception json.RawMessage `json:"exceptionDetails"`
		}
		p := map[string]any{"expression": expression, "returnByValue": true, "awaitPromise": true}
		if world != 0 {
			p["contextId"] = world
		}
		call("Runtime.evaluate", p, &res)
		if len(res.Exception) > 0 {
			t.Fatalf("evaluate %s: %s", expression, res.Exception)
		}
		return res.Result.Value
	}
	wait := func(what string, condition func() bool) {
		t.Helper()
		for !condition() {
			select {
			case <-ctx.Done():
				t.Fatalf("waiting for %s: %v", what, ctx.Err())
			case <-time.After(25 * time.Millisecond):
			}
		}
	}
	call("Emulation.setDeviceMetricsOverride", map[string]any{"width": 800, "height": 600, "deviceScaleFactor": 1, "mobile": false}, nil)
	wait("fixture document", func() bool {
		return string(eval(0, `document.readyState==='complete'&&!!document.querySelector('button')&&document.querySelector('button').textContent==='Save'&&!document.querySelector('button').id`)) == "true"
	})
	if err = chrome.SetLive(true); err != nil {
		t.Fatal(err)
	}
	var world int64
	clickTool := func(name string) {
		t.Helper()
		var xy [2]float64
		if err := json.Unmarshal(eval(world, fmt.Sprintf(`(()=>{const root=__couchcastOperatorSketch.labelInput.getRootNode(),b=[...root.querySelectorAll('.sketch-toolbar button')].find(b=>b.textContent===%q),r=b.getBoundingClientRect();if(b.disabled||!b.contains(root.elementFromPoint(r.x+r.width/2,r.y+r.height/2)))throw Error('toolbar action unavailable');return [r.x+r.width/2,r.y+r.height/2]})()`, name)), &xy); err != nil {
			t.Fatal(err)
		}
		for _, typ := range []string{"mousePressed", "mouseReleased"} {
			call("Input.dispatchMouseEvent", map[string]any{"type": typ, "x": xy[0], "y": xy[1], "button": "left", "clickCount": 1}, nil)
		}
	}
	wait("operator world", func() bool {
		done := make(chan struct{})
		chrome.do(func(s *loopState) { world = s.contexts[session]; close(done) })
		<-done
		return world != 0
	})
	wait("operator sketch permission", func() bool {
		return string(eval(world, `!!globalThis.__couchcastOperatorSketch?.state().allowed&&__couchcastOperatorSketch.state().live`)) == "true"
	})
	if _, err = cast.board.AdmitAnnotation(); err == nil {
		t.Fatal("viewer annotation enabled by console drawing")
	}
	wait("operator UI unsuppressed", func() bool {
		return string(eval(world, `!__couchcastOperatorSketch.labelInput.getRootNode().host.hidden`)) == "true"
	})
	if got := string(eval(world, `(()=>{const root=__couchcastOperatorSketch.labelInput.getRootNode(),bar=root.querySelector('.sketch-toolbar'),buttons=[...bar.querySelectorAll('button')];return !bar.hidden&&getComputedStyle(bar).display!=='none'&&!buttons.find(b=>b.textContent==='Sketch').disabled&&buttons.find(b=>b.textContent==='Done').disabled})()`)); got != "true" {
		t.Fatalf("annotation-off console toolbar unavailable or idle Done enabled: %s; %s", got, eval(world, `JSON.stringify([...__couchcastOperatorSketch.labelInput.getRootNode().querySelectorAll('.sketch-toolbar button')].map(b=>({text:b.textContent,disabled:b.disabled,display:getComputedStyle(b).display})))`))
	}
	var records []recording.Record
	read := func() []recording.Record {
		t.Helper()
		rs, e := readSubmissionRecording(ctx, cast.RecordingPath())
		if e == nil {
			records = rs
		}
		return records
	}
	wait("recorded source asset", func() bool {
		for _, r := range read() {
			if r.Kind() == "asset" {
				return true
			}
		}
		return false
	})
	clickTool("Sketch")
	wait("prepared console scene", func() bool { return string(eval(world, `__couchcastOperatorSketch.state().phase`)) == `"drawing"` })
	if err := recorder.CaptureBackgroundScreenshot(ctx, chrome.conn, session, world, map[string]any{"format": "png"}, nil); !errors.Is(err, recorder.ErrBackgroundCaptureDeferred) {
		t.Fatalf("active drawing background capture: %v", err)
	}
	if got := string(eval(world, `(()=>{const root=__couchcastOperatorSketch.labelInput.getRootNode();return ['Sketch','Undo','Redo','Done'].every(name=>{const b=[...root.querySelectorAll('.sketch-toolbar button')].find(b=>b.textContent===name),r=b.getBoundingClientRect();return r.width>0&&r.height>0&&r.left>=0&&r.top>=0&&r.right<=innerWidth&&r.bottom<=innerHeight&&b.contains(root.elementFromPoint(r.x+r.width/2,r.y+r.height/2))})})()`)); got != "true" {
		t.Fatalf("active sketch toolbar buttons are hidden or covered: %s; %s", got, eval(world, `JSON.stringify([...__couchcastOperatorSketch.labelInput.getRootNode().querySelectorAll('.sketch-toolbar button')].map(b=>{const r=b.getBoundingClientRect(),s=getComputedStyle(b),hit=b.getRootNode().elementFromPoint(r.x+r.width/2,r.y+r.height/2);return {text:b.textContent,rect:[r.x,r.y,r.width,r.height],position:s.position,z:s.zIndex,hit:hit?.className}}))`))
	}
	if got := string(eval(0, `typeof globalThis.__couchcastOperatorSketch==='undefined'&&typeof globalThis.__couchcastPrivateSceneFrame==='undefined'`)); got != "true" {
		t.Fatalf("console hooks exposed in product world: %s", got)
	}
	if got := string(eval(world, `(()=>{const f=globalThis.__couchcastPrivateSceneFrame;if(!f)return false;try{return f.contentDocument===null&&!f.src.includes('Save')}catch{return true}})()`)); got != "true" {
		t.Fatalf("parent can read private scene DOM/assets: %s", got)
	}
	pen := func(kind string, x, pressure float64, buttons int) {
		t.Helper()
		button := "none"
		if kind != "mouseMoved" {
			button = "left"
		}
		call("Input.dispatchMouseEvent", map[string]any{"type": kind, "x": x, "y": 140, "button": button, "buttons": buttons, "clickCount": 1, "pointerType": "pen", "force": pressure}, nil)
	}
	pen("mousePressed", 100, .2, 1)
	pen("mouseMoved", 115, .5, 1)
	pen("mouseMoved", 130, .8, 1)
	var pts []recording.SketchPoint
	wait("durable pressure samples before pointer-up", func() bool {
		pts = nil
		for _, r := range read() {
			if p, ok := r.(*recording.SketchPoints); ok {
				pts = append(pts, p.Pts...)
			}
		}
		return len(pts) >= 3
	})
	if math.Abs(pts[0][2]-.2) > .02 || math.Abs(pts[len(pts)-1][2]-.8) > .02 {
		t.Fatalf("pen pressure lost: %v", pts)
	}
	for _, r := range read() {
		if r.Kind() == "sketch-stroke-end" {
			t.Fatal("stroke sealed before pointer-up")
		}
	}
	pen("mouseReleased", 130, 0, 0)
	waitKind := func(kind string) {
		t.Helper()
		wait(kind, func() bool {
			for _, r := range read() {
				if r.Kind() == kind {
					return true
				}
			}
			return false
		})
	}
	waitKind("sketch-stroke-end")
	wait("confirmed stroke", func() bool {
		return string(eval(world, `__couchcastOperatorSketch.state().slots.every(s=>!s.inflight&&!s.queued)`)) == "true"
	})
	clickTool("Undo")
	waitKind("sketch-undo")
	wait("Redo enabled", func() bool {
		return string(eval(world, `!Array.from(__couchcastOperatorSketch.labelInput.getRootNode().querySelectorAll('.sketch-toolbar button')).find(b=>b.textContent==='Redo').disabled`)) == "true"
	})
	clickTool("Redo")
	waitKind("sketch-redo")
	clickTool("Done")
	wait("required label", func() bool {
		return string(eval(world, `!__couchcastOperatorSketch.labelInput.parentElement.hidden`)) == "true"
	})
	eval(world, `(()=>{const input=__couchcastOperatorSketch.labelInput,host=input.getRootNode().host;input.value='AB';input.setSelectionRange(1,1);globalThis.captureBlur=0;globalThis.captureHidden=0;input.addEventListener('blur',()=>captureBlur++);new MutationObserver(()=>{if(host.hidden)captureHidden++}).observe(host,{attributes:true});return true})()`)
	var screenshot struct {
		Data string `json:"data"`
	}
	for i := 0; i < 3; i++ {
		if err := recorder.CaptureBackgroundScreenshot(ctx, chrome.conn, session, world, map[string]any{"format": "png"}, &screenshot); !errors.Is(err, recorder.ErrBackgroundCaptureDeferred) {
			t.Fatalf("open label background capture: %v", err)
		}
		eval(0, fmt.Sprintf(`document.querySelector('button').setAttribute('data-focus-check',%q)`, fmt.Sprint(i)))
	}
	if got := string(eval(world, `(()=>{const input=__couchcastOperatorSketch.labelInput;return input.getRootNode().activeElement===input&&input.selectionStart===1&&input.selectionEnd===1&&captureBlur===0&&captureHidden===0})()`)); got != "true" {
		t.Fatalf("background capture interrupted label focus/caret: %s", got)
	}
	call("Input.insertText", map[string]any{"text": "x"}, nil)
	if got := string(eval(world, `__couchcastOperatorSketch.labelInput.value`)); got != `"AxB"` {
		t.Fatalf("native typing after idle/remote capture opportunities: %s", got)
	}
	eval(world, `(async()=>{const s=__couchcastOperatorSketch;s.labelInput.value='Move this console control';await s.saveLabel();return true})()`)
	waitKind("sketch-label")
	wait("held scene exit", func() bool { return string(eval(world, `!globalThis.__couchcastPrivateSceneFrame`)) == "true" })
	// The same bridge used by the manager must expose the exact finished shape
	// before the chord can submit. No browser-side send action is added.
	var previewDelivered map[core.PreviewItem]struct{}
	var previewState core.PreviewState
	var preparedView core.PreviewSelection
	var preparedHost core.PreviewHost
	var previewDeliveryGeneration uint64
	publishPreview := func(want int, empty ...bool) {
		t.Helper()
		// An unrelated delivery transition may refresh the same selected shapes.
		// Exercise that new publication identity without replacing keyed cards.
		previewDeliveryGeneration++
		wait("acknowledged private submission preview", func() bool {
			host, _, e := cast.SubmissionPreviewHost(ctx)
			if e != nil {
				return false
			}
			selected, e := cast.board.Selection()
			if e != nil || len(selected.Sketches) != want {
				return false
			}
			candidates := make([]core.PreviewSketchCandidate, 0, len(selected.Sketches))
			for _, v := range selected.Sketches {
				candidates = append(candidates, core.PreviewSketchCandidate{Item: core.PreviewItem{RecordingID: chrome.opts.Record.ID, SketchID: v.ID, Sequence: v.NextSequence, Digest: v.Digest}, Source: v.Source, Finish: v.Finish})
			}
			notes := make([]core.PreviewNote, 0, len(selected.Notes))
			for _, v := range selected.Notes {
				notes = append(notes, core.PreviewNote{ID: v.ID, Revision: v.Revision})
			}
			view, e := cast.ReadSubmissionVisibility(ctx, host)
			if e != nil {
				return false
			}
			delivered := core.DeliveredFeedback{RecordingID: chrome.opts.Record.ID, Sketches: map[core.PreviewItem]struct{}{}}
			for item := range previewDelivered {
				delivered.Sketches[item] = struct{}{}
			}
			if len(empty) > 0 && empty[0] {
				for _, candidate := range candidates {
					delivered.Sketches[candidate.Item] = struct{}{}
				}
			}
			selection, e := core.SelectVisibleFeedback(candidates, notes, delivered, core.PreviewStamp{Board: selected.SketchGeneration, Delivered: previewDeliveryGeneration}, view)
			if e != nil {
				t.Fatalf("native visible feedback selector: %v", e)
			}
			selection, e = cast.ResolveSubmissionPreview(ctx, host, selection)
			if e != nil {
				return false
			}
			previewState, _ = core.NextPreview(previewState, core.PreviewHostChanged{Host: host})
			previewState, _ = core.NextPreview(previewState, core.PreviewCandidatesChanged{Selection: selection})
			var effects []core.PreviewEffect
			previewState, effects = core.NextPreview(previewState, core.PreviewRefresh{})
			for _, effect := range effects {
				if publication, ok := effect.(core.PreviewPublish); ok {
					if e = cast.PublishSubmissionPreview(ctx, publication.Publication); e != nil {
						previewState, _ = core.NextPreview(previewState, core.PreviewPublishFailed{Publication: publication.Publication})
						return false
					}
					previewState, _ = core.NextPreview(previewState, core.PreviewPublishAccepted{Publication: publication.Publication})
				}
			}
			if previewState.Phase() == core.PreviewShown {
				preparedView = selection
				preparedHost = host
				return true
			}
			return false
		})
	}
	publishPreview(1)
	if got := string(eval(world, `(()=>{const root=__couchcastOperatorSketch.labelInput.getRootNode(),preview=root.querySelector('.sketch-submission-preview'),cards=preview.querySelectorAll('.sketch-submission-card'),canvas=cards[0]?.querySelector('canvas');if(preview.hidden||cards.length!==1||preview.querySelectorAll('button').length!==0||!preview.textContent.includes('Pending feedback: 1 sketches, 0 annotations')||!preview.textContent.includes('Move this console control')||!canvas)return false;const pixels=canvas.getContext('2d').getImageData(0,0,canvas.width,canvas.height).data;let ink=0,min=canvas.width,max=-1;for(let y=0;y<canvas.height;y++)for(let x=0;x<canvas.width;x++)if(pixels[(y*canvas.width+x)*4+3]>20){ink++;min=Math.min(min,x);max=Math.max(max,x)}const thickness=x=>{let n=0;for(let y=0;y<canvas.height;y++)if(pixels[(y*canvas.width+x)*4+3]>20)n++;return n};const thin=thickness(Math.round(min+(max-min)*.25)),thick=thickness(Math.round(min+(max-min)*.75));globalThis.previewCard=cards[0];globalThis.previewCanvas=canvas;return ink>0&&thin>0&&thick>thin})()`)); got != "true" {
		t.Fatalf("native pressure/count/note preview unavailable: %s", got)
	}
	// Completed watch-mode cards are also private open UI: periodic capture
	// must defer before hiding their host, preserving both focus and card identity.
	eval(world, `globalThis.previewIdleFocus=__couchcastOperatorSketch.labelInput.getRootNode().activeElement`)
	for i := 0; i < 3; i++ {
		if e := recorder.CaptureBackgroundScreenshot(ctx, chrome.conn, session, world, map[string]any{"format": "png"}, nil); !errors.Is(e, recorder.ErrBackgroundCaptureDeferred) {
			t.Fatalf("completed preview background capture must defer: %v", e)
		}
	}
	if got := string(eval(world, `(()=>{const root=__couchcastOperatorSketch.labelInput.getRootNode();return root.activeElement===previewIdleFocus&&root.querySelector('.sketch-submission-card')===previewCard&&previewCard.querySelector('canvas')===previewCanvas})()`)); got != "true" {
		t.Fatalf("idle capture changed preview focus/identity: %s", got)
	}
	// Start an unfinished second sketch whose ink is inside the first stroke's
	// bounds, preserving the existing original-node placement checks below.
	clickTool("Sketch")
	wait("second preview focus sketch", func() bool { return string(eval(world, `__couchcastOperatorSketch.state().phase`)) == `"drawing"` })
	pen("mousePressed", 110, .2, 1)
	pen("mouseMoved", 120, .2, 1)
	pen("mouseReleased", 120, 0, 0)
	wait("second stroke settled", func() bool {
		return string(eval(world, `__couchcastOperatorSketch.state().slots.every(s=>!s.inflight&&!s.queued)`)) == "true"
	})
	clickTool("Done")
	wait("second required label focus", func() bool {
		return string(eval(world, `(()=>{const input=__couchcastOperatorSketch.labelInput;return !input.parentElement.hidden&&input.getRootNode().activeElement===input})()`)) == "true"
	})
	eval(world, `(()=>{const input=__couchcastOperatorSketch.labelInput;input.value='AB';input.setSelectionRange(1,1);globalThis.previewBlur=0;input.addEventListener('blur',()=>previewBlur++);return true})()`)
	for i := 0; i < 3; i++ {
		// Remote ordinary pointer traffic must neither replace the preview cards nor
		// interrupt the real sketch editor. The recorder remains the sole writer.
		selected, e := cast.board.Selection()
		if e != nil {
			t.Fatal(e)
		}
		if e = chrome.Emit(&recording.Stroke{ID: fmt.Sprintf("preview-remote-%d", i), Author: recording.Author{ID: "remote-preview", Color: "#00ff00"}, Color: "#00ff00", Pts: [][3]float64{{300, 300, 0}, {310, 310, 10}}}, selected.Document); e != nil {
			t.Fatal(e)
		}
		if e = recorder.CaptureBackgroundScreenshot(ctx, chrome.conn, session, world, map[string]any{"format": "png"}, nil); !errors.Is(e, recorder.ErrBackgroundCaptureDeferred) {
			t.Fatalf("preview label background capture: %v", e)
		}
		host, _, e := cast.SubmissionPreviewHost(ctx)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = cast.ReadSubmissionVisibility(ctx, host); e == nil {
			t.Fatal("active label editor admitted a completed visibility witness")
		}
	}
	if got := string(eval(world, `new Promise(resolve=>setTimeout(()=>{const input=__couchcastOperatorSketch.labelInput,root=input.getRootNode();resolve(root.activeElement===input&&input.selectionStart===1&&input.selectionEnd===1&&previewBlur===0&&root.querySelector('.sketch-submission-card')===previewCard&&previewCard.querySelector('canvas')===previewCanvas)},50))`)); got != "true" {
		t.Fatalf("preview refresh/remote traffic stole editor focus or replaced cards: %s", got)
	}
	call("Input.insertText", map[string]any{"text": "x"}, nil)
	if got := string(eval(world, `__couchcastOperatorSketch.labelInput.value`)); got != `"AxB"` {
		t.Fatalf("native typing after preview updates: %s", got)
	}
	eval(world, `(async()=>{const s=__couchcastOperatorSketch;s.labelInput.value='Preview focus draft';await s.saveLabel();return true})()`)
	wait("second sketch finished", func() bool { return string(eval(world, `!globalThis.__couchcastPrivateSceneFrame`)) == "true" })
	publishPreview(2)
	if got := string(eval(world, `(()=>{const p=__couchcastOperatorSketch.labelInput.getRootNode().querySelector('.sketch-submission-preview');return p.querySelectorAll('.sketch-submission-card').length===2&&p.textContent.includes('Preview focus draft')})()`)); got != "true" {
		t.Fatalf("second finished shape missing from preview: %s", got)
	}
	// After dispatch suppresses every exact version, no empty panel should
	// keep optional fidelity captures deferred; recorded ink remains unchanged.
	publishPreview(2, true)
	if got := string(eval(world, `(()=>{const root=__couchcastOperatorSketch.labelInput.getRootNode();return root.querySelectorAll('.sketch-submission-card').length===0&&!__couchcastCaptureBusy()})()`)); got != "true" {
		t.Fatalf("empty preview still holds capture suppression: %s", got)
	}
	if err := recorder.CaptureBackgroundScreenshot(ctx, chrome.conn, session, world, map[string]any{"format": "png"}, &screenshot); err != nil || screenshot.Data == "" {
		t.Fatalf("background capture did not resume after finished label: %v", err)
	}
	type inkBounds struct {
		Count int `json:"count"`
		MinX  int `json:"minX"`
		MinY  int `json:"minY"`
		MaxX  int `json:"maxX"`
		MaxY  int `json:"maxY"`
	}
	pixels := func() inkBounds {
		t.Helper()
		var bounds inkBounds
		raw := eval(world, `(()=>{const c=__couchcastOperatorSketch.layer.parentNode.querySelector('.sketch-observer');if(!c)return {count:0};const {data}=c.getContext('2d').getImageData(0,0,c.width,c.height);let count=0,minX=c.width,minY=c.height,maxX=-1,maxY=-1;for(let y=0;y<c.height;y++)for(let x=0;x<c.width;x++)if(data[(y*c.width+x)*4+3]>20){count++;minX=Math.min(minX,x);minY=Math.min(minY,y);maxX=Math.max(maxX,x);maxY=Math.max(maxY,y)}return {count,minX,minY,maxX,maxY}})()`)
		if err := json.Unmarshal(raw, &bounds); err != nil {
			t.Fatal(err)
		}
		return bounds
	}
	// Save has neither a stable ID nor meaningful replacement text. Seeing its
	// completed ink here therefore requires the real CDP original binder.
	var original inkBounds
	wait("completed live ink on original common Save", func() bool {
		original = pixels()
		return original.Count > 0 && original.MinX >= 96 && original.MinX <= 104 && original.MaxX >= 126 && original.MaxX <= 134 && original.MinY >= 136 && original.MinY <= 144
	})

	// Delivered feedback changes only the live painter. The second, shorter
	// shape overlaps the first, so both the selective and complete masks need
	// real pixel checks rather than just inspecting mask metadata.
	beforeMask, e := recording.Reduce(read(), math.MaxInt64)
	if e != nil {
		t.Fatal(e)
	}
	beforeProjection, e := json.Marshal(beforeMask.Sketches.Snapshot())
	if e != nil {
		t.Fatal(e)
	}
	selectedForMask, e := cast.SubmissionSelection()
	if e != nil || len(selectedForMask.Sketches) != 2 {
		t.Fatalf("mask selection: %+v %v", selectedForMask, e)
	}
	deliveredMask := core.DeliveredFeedback{RecordingID: filepath.Base(cast.RecordingPath()), Sketches: map[core.PreviewItem]struct{}{}, Notes: map[core.PreviewNote]struct{}{}}
	addMask := func(v board.SelectedSketch) {
		deliveredMask.Sketches[core.PreviewItem{RecordingID: deliveredMask.RecordingID, SketchID: v.ID, Sequence: v.NextSequence, Digest: v.Digest}] = struct{}{}
	}
	addMask(selectedForMask.Sketches[0])
	if e = cast.SetDeliveredFeedback(deliveredMask); e != nil {
		t.Fatal(e)
	}
	wait("submitted first sketch hidden, unsent second still painted", func() bool {
		ink := pixels()
		return ink.Count > 0 && ink.Count < original.Count && ink.MinX > original.MinX && ink.MaxX < original.MaxX
	})
	addMask(selectedForMask.Sketches[1])
	if e = cast.SetDeliveredFeedback(deliveredMask); e != nil {
		t.Fatal(e)
	}
	wait("all submitted console sketches hidden", func() bool { return pixels().Count == 0 })
	// Masked vectors stay in history, but are no longer eligible in the
	// current visible pending selection.
	publishPreview(2)
	if got := string(eval(world, `__couchcastOperatorSketch.labelInput.getRootNode().querySelectorAll('.sketch-submission-card').length`)); got != "0" {
		t.Fatalf("masked feedback still selected: %s", got)
	}
	publishPreview(2, true)
	afterMask, e := recording.Reduce(read(), math.MaxInt64)
	if e != nil {
		t.Fatal(e)
	}
	afterProjection, e := json.Marshal(afterMask.Sketches.Snapshot())
	if e != nil || !bytes.Equal(beforeProjection, afterProjection) {
		t.Fatalf("mask changed accepted recording projection: %v\nbefore %s\nafter %s", e, beforeProjection, afterProjection)
	}

	// Reproduce the operator's next action after submission: enter another
	// held scene and draw through real pointer input while both old versions are
	// still suppressed. Do not clear the mask to make drawing succeed.
	var postSubmitDiagnostic string
	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("post-submit drawing diagnostic: %s; mask=%+v; receiver stats=%+v", postSubmitDiagnostic, cast.board.FeedbackMask(), cast.board.OperatorSketchDeliveryStats())
		}
	})
	diagnosePostSubmit := func() {
		postSubmitDiagnostic = string(eval(world, `(()=>{const tool=__couchcastOperatorSketch,root=tool.labelInput.getRootNode(),frame=globalThis.__couchcastPrivateSceneFrame,canvases=[...tool.layer.parentNode.querySelectorAll('canvas')];return JSON.stringify({state:tool.state(),status:root.querySelector('.sketch-status')?.textContent,visibility:document.visibilityState,focused:document.hasFocus(),host:{connected:root.host.isConnected,hidden:root.host.hidden},frame:frame?{hidden:frame.hidden,rect:frame.getBoundingClientRect().toJSON(),ready:!!frame.contentWindow}:null,hit:root.elementFromPoint(110,140)?.className,layer:{hidden:tool.layer.hidden,pointerEvents:getComputedStyle(tool.layer).pointerEvents,zIndex:getComputedStyle(tool.layer).zIndex,rect:tool.layer.getBoundingClientRect().toJSON()},canvases:canvases.map(c=>({class:c.className,hidden:c.hidden,width:c.width,height:c.height,pointerEvents:getComputedStyle(c).pointerEvents,display:getComputedStyle(c).display}))})})()`))
	}

	// Match the operator's intervening box-note flow through the real shared
	// annotation module before returning to Sketch.
	if e = cast.board.SetAnnotationAllowed(ctx, true); e != nil {
		t.Fatal(e)
	}
	wait("annotation palette available before post-submit drawing", func() bool {
		return string(eval(world, `(()=>{const root=__couchcastOperatorSketch.labelInput.getRootNode(),b=[...root.querySelectorAll('.annotate-rail .mode')].find(b=>b.textContent==='💭');return !!b&&!b.disabled})()`)) == "true"
	})
	clickNativeRootButton := func(selector, text string) {
		t.Helper()
		var point [2]float64
		if e := json.Unmarshal(eval(world, fmt.Sprintf(`(()=>{const root=__couchcastOperatorSketch.labelInput.getRootNode(),b=[...root.querySelectorAll(%q)].find(b=>b.textContent===%q),r=b?.getBoundingClientRect();if(!b||b.disabled||!r.width||!b.contains(root.elementFromPoint(r.x+r.width/2,r.y+r.height/2)))throw Error('native control unavailable');return [r.x+r.width/2,r.y+r.height/2]})()`, selector, text)), &point); e != nil {
			t.Fatal(e)
		}
		for _, kind := range []string{"mousePressed", "mouseReleased"} {
			call("Input.dispatchMouseEvent", map[string]any{"type": kind, "x": point[0], "y": point[1], "button": "left", "clickCount": 1}, nil)
		}
	}
	clickNativeRootButton(".annotate-rail .mode", "💭")
	for _, event := range []struct {
		kind    string
		x, y    float64
		buttons int
	}{{"mousePressed", 180, 180, 1}, {"mouseMoved", 210, 210, 1}, {"mouseReleased", 210, 210, 0}} {
		button := "left"
		if event.kind == "mouseMoved" {
			button = "none"
		}
		call("Input.dispatchMouseEvent", map[string]any{"type": event.kind, "x": event.x, "y": event.y, "button": button, "buttons": event.buttons, "clickCount": 1}, nil)
	}
	wait("post-submit bounding-box note editor", func() bool {
		diagnosePostSubmit()
		return string(eval(world, `(()=>{const d=__couchcastOperatorSketch.labelInput.getRootNode().querySelector('.note-dialog');return d&&!d.hidden&&!d.classList.contains('hidden')})()`)) == "true"
	})
	eval(world, `(()=>{const input=__couchcastOperatorSketch.labelInput.getRootNode().querySelector('.note-input');input.value='Box note between sketch cycles';input.dispatchEvent(new Event('input',{bubbles:true}));return true})()`)
	wait("post-submit box capture ready to Save", func() bool {
		return string(eval(world, `(()=>{const b=[...__couchcastOperatorSketch.labelInput.getRootNode().querySelectorAll('.note-buttons button')].find(b=>b.textContent==='Save');return !!b&&!b.disabled})()`)) == "true"
	})
	clickNativeRootButton(".note-buttons button", "Save")
	wait("intervening bounding-box note saved", func() bool {
		diagnosePostSubmit()
		for _, r := range read() {
			if n, ok := r.(*recording.Note); ok && n.Text == "Box note between sketch cycles" {
				return true
			}
		}
		return false
	})
	t.Logf("intervening real box note saved; %s", postSubmitDiagnostic)
	clickTool("Sketch")
	wait("new post-submit held scene", func() bool {
		diagnosePostSubmit()
		return string(eval(world, `__couchcastOperatorSketch.state().phase`)) == `"drawing"`
	})
	pen("mousePressed", 110, .2, 1)
	pen("mouseMoved", 115, .5, 1)
	pen("mouseMoved", 120, .8, 1)
	wait("new accepted visible ink after submission mask", func() bool {
		diagnosePostSubmit()
		state, e := recording.Reduce(read(), math.MaxInt64)
		if e != nil {
			return false
		}
		projection := state.Sketches.Snapshot()
		return len(projection.Sketches) == 3 && len(projection.Sketches[2].Strokes) == 1 && len(projection.Sketches[2].Strokes[0].Pts) >= 3 && pixels().Count > 0
	})
	pen("mouseReleased", 120, 0, 0)
	wait("new post-submit stroke settled", func() bool {
		return string(eval(world, `__couchcastOperatorSketch.state().slots.every(s=>!s.inflight&&!s.queued)`)) == "true"
	})
	clickTool("Note")
	wait("new post-submit Note editor", func() bool {
		return string(eval(world, `!__couchcastOperatorSketch.labelInput.parentElement.hidden`)) == "true"
	})
	eval(world, `__couchcastOperatorSketch.labelInput.value='New after submitted feedback'`)
	var saveAfterSubmit [2]float64
	if e = json.Unmarshal(eval(world, `(()=>{const root=__couchcastOperatorSketch.labelInput.getRootNode(),button=[...root.querySelectorAll('button')].find(b=>b.textContent==='Save label'),r=button.getBoundingClientRect();if(button.disabled||!r.width||!button.contains(root.elementFromPoint(r.x+r.width/2,r.y+r.height/2)))throw Error('Save label unavailable');return [r.x+r.width/2,r.y+r.height/2]})()`), &saveAfterSubmit); e != nil {
		t.Fatal(e)
	}
	for _, kind := range []string{"mousePressed", "mouseReleased"} {
		call("Input.dispatchMouseEvent", map[string]any{"type": kind, "x": saveAfterSubmit[0], "y": saveAfterSubmit[1], "button": "left", "clickCount": 1}, nil)
	}
	wait("new post-submit sketch label accepted", func() bool {
		diagnosePostSubmit()
		state, e := recording.Reduce(read(), math.MaxInt64)
		return e == nil && len(state.Sketches.Snapshot().Sketches) == 3 && state.Sketches.Snapshot().Sketches[2].Label == "New after submitted feedback"
	})
	wait("new post-submit label reply settled and scene retired", func() bool {
		diagnosePostSubmit()
		return string(eval(world, `__couchcastOperatorSketch.state().slots.every(s=>!s.inflight&&!s.queued)&&!globalThis.__couchcastPrivateSceneFrame`)) == "true"
	})
	previewDelivered = deliveredMask.Sketches
	publishPreview(3)
	if got := string(eval(world, `(()=>{const p=__couchcastOperatorSketch.labelInput.getRootNode().querySelector('.sketch-submission-preview');return !p.hidden&&p.querySelectorAll('.sketch-submission-card').length===1&&p.textContent.includes('New after submitted feedback')})()`)); got != "true" {
		t.Fatalf("new post-submit sketch missing pending card: %s; %s", got, postSubmitDiagnostic)
	}
	if ink := pixels(); ink.Count == 0 {
		t.Fatalf("new completed post-submit sketch invisible: %s", postSubmitDiagnostic)
	}

	// Keep the third exact version UNSENT across the real tab round trip. A
	// pending thumbnail proves receiver vectors survived; its live ink must also
	// return to the same physical source element after reactivation.
	unsentStateBefore, e := recording.Reduce(read(), math.MaxInt64)
	if e != nil {
		t.Fatal(e)
	}
	unsentAnchor := unsentStateBefore.Sketches.Snapshot().Sketches[2].Strokes[0].Begin.At.Node
	unsentSelectionBefore, e := cast.SubmissionSelection()
	if e != nil {
		t.Fatal(e)
	}
	unsentSourceDoc := unsentSelectionBefore.Sketches[2].Source.Doc
	eval(0, `(()=>{globalThis.__testUnsentOriginalButton=document.querySelector('button');return true})()`)
	t.Logf("before unsent tab return: sourceDoc=%d at.node=%d originalHook=%s", unsentSourceDoc, unsentAnchor, eval(world, fmt.Sprintf(`__couchcastOperatorOriginal(%d,%d,document.querySelector('button'))`, unsentSourceDoc, unsentAnchor)))
	var unsentSourceTab string
	unsentTabDone := make(chan struct{})
	chrome.do(func(s *loopState) { unsentSourceTab = castTab(s); close(unsentTabDone) })
	<-unsentTabDone
	var unsentOtherTab struct {
		TargetID string `json:"targetId"`
	}
	if e = chrome.conn.Call(ctx, "", "Target.createTarget", map[string]any{"url": page.URL + "?tab=unsent-return"}, &unsentOtherTab); e != nil {
		t.Fatal(e)
	}
	if e = chrome.conn.Call(ctx, "", "Target.activateTarget", map[string]any{"targetId": unsentOtherTab.TargetID}, nil); e != nil {
		t.Fatal(e)
	}
	wait("unsent sketch other tab active", func() bool {
		done := make(chan struct{})
		active := ""
		chrome.do(func(s *loopState) { active = castTab(s); close(done) })
		<-done
		return active == unsentOtherTab.TargetID
	})
	if e = chrome.conn.Call(ctx, "", "Target.activateTarget", map[string]any{"targetId": unsentSourceTab}, nil); e != nil {
		t.Fatal(e)
	}
	wait("unsent sketch original tab active again", func() bool {
		done := make(chan struct{})
		active := ""
		chrome.do(func(s *loopState) { active = castTab(s); world = s.contexts[session]; close(done) })
		<-done
		return active == unsentSourceTab && world != 0
	})
	publishPreview(3)
	t.Logf("after unsent tab return: boardDoc=%d sourceDoc=%d at.node=%d sameConnectedDOM=%s sourceOriginalHook=%s", cast.board.OperatorView().Doc, unsentSourceDoc, unsentAnchor, eval(0, `globalThis.__testUnsentOriginalButton===document.querySelector('button')&&globalThis.__testUnsentOriginalButton.isConnected`), eval(world, fmt.Sprintf(`__couchcastOperatorOriginal(%d,%d,document.querySelector('button'))`, unsentSourceDoc, unsentAnchor)))
	// Reactivation can initially leave the old raster buffer intact. Exercise
	// the normal resize handler so success requires current placement proof.
	eval(world, `new Promise(resolve=>{window.dispatchEvent(new Event('resize'));requestAnimationFrame(()=>requestAnimationFrame(()=>resolve(true)))})`)
	var unsentRestored inkBounds
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); {
		diagnosePostSubmit()
		unsentRestored = pixels()
		if unsentRestored.Count > 0 {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-time.After(25 * time.Millisecond):
		}
	}
	if unsentRestored.Count == 0 {
		t.Logf("late unsent tab-return binding: boardDoc=%d sourceDoc=%d at.node=%d sameConnectedDOM=%s sourceOriginalHook=%s", cast.board.OperatorView().Doc, unsentSourceDoc, unsentAnchor, eval(0, `globalThis.__testUnsentOriginalButton===document.querySelector('button')&&globalThis.__testUnsentOriginalButton.isConnected`), eval(world, fmt.Sprintf(`__couchcastOperatorOriginal(%d,%d,document.querySelector('button'))`, unsentSourceDoc, unsentAnchor)))
		selected, e := cast.SubmissionSelection()
		t.Fatalf("UNSENT tab-return ink missing: pixels=%+v, pending=%s, drawing=%s, boardDoc=%d, mask=%+v, selection=%+v err=%v, receiver=%+v", unsentRestored, eval(world, `(()=>{const p=__couchcastOperatorSketch.labelInput.getRootNode().querySelector('.sketch-submission-preview');return {hidden:p.hidden,count:p.querySelectorAll('.sketch-submission-card').length,text:p.textContent}})()`), postSubmitDiagnostic, cast.board.OperatorView().Doc, cast.board.FeedbackMask(), selected, e, cast.board.OperatorSketchDeliveryStats())
	}
	t.Logf("unsent third sketch live pixels survived real A-to-B-to-A: %+v", unsentRestored)
	// Exercise the exact private visibility/recorder/Board/builder path after
	// reattaching the old ink to the new recorded document. The screenshot
	// freezes this current frame; original source/finish stay historical.
	publishPreview(3)
	frozen, e := cast.FreezeSubmissionPreview(preparedHost, preparedView)
	if e != nil {
		t.Fatal("freeze native current view", e)
	}
	bundle, files, e := cast.PrepareSubmission(ctx, "native-current-view", castSubmissionTarget(), frozen, "", nil)
	if e != nil {
		t.Fatal("render native current view", e)
	}
	if bundle.SketchPage == nil || bundle.SketchPage.Current == nil || bundle.SketchPage.Provenance != "replay-rendered-current-view" || bundle.SketchPage.Source.Doc == unsentSourceDoc || len(bundle.Sketches) != 1 || len(files[bundle.SketchPage.File]) == 0 {
		t.Fatalf("current shared evidence missing: %+v", bundle.SketchPage)
	}
	if bundle.Sketches[0].Source.Doc != unsentSourceDoc {
		t.Fatal("current projection rewrote original source")
	}

	t.Logf("mask-only post-submit real pointer/Note/save/card passed; %s", postSubmitDiagnostic)
	previewDelivered = nil
	publishPreview(3, true)
	if e = cast.SetDeliveredFeedback(core.DeliveredFeedback{RecordingID: deliveredMask.RecordingID}); e != nil {
		t.Fatal(e)
	}
	var restored inkBounds
	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("mask restore original=%+v restored=%+v", original, restored)
		}
	})
	wait("restored console display projection", func() bool {
		restored = pixels()
		return restored.Count > 0 && math.Abs(float64(restored.MinX-original.MinX)) <= 2 && math.Abs(float64(restored.MaxX-original.MaxX)) <= 2 && math.Abs(float64(restored.MinY-original.MinY)) <= 2 && math.Abs(float64(restored.MaxY-original.MaxY)) <= 2
	})
	t.Logf("mask restore original=%+v restored=%+v (outer geometry tolerance 2px)", original, restored)
	original = restored
	eval(0, `(()=>{const button=document.querySelector('button');button.style.left='280px';button.style.top='180px';return true})()`)
	var moved inkBounds
	wait("completed ink follows exact moved Save", func() bool {
		moved = pixels()
		return moved.Count > 0 && math.Abs(float64(moved.MinX-original.MinX-200)) <= 2 && math.Abs(float64(moved.MinY-original.MinY-80)) <= 2
	})
	eval(0, `(()=>{const original=document.querySelector('button');original.replaceWith(original.cloneNode(true));return true})()`)
	wait("identical replacement has no original ink proof", func() bool { return pixels().Count == 0 })
	// Keep watching across several worker retry turns: an identical new node
	// must not acquire the retired original's recording identity later.
	for deadline := time.Now().Add(750 * time.Millisecond); time.Now().Before(deadline); {
		if ink := pixels(); ink.Count != 0 {
			t.Fatalf("identical Save replacement acquired original proof: %+v", ink)
		}
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-time.After(50 * time.Millisecond):
		}
	}
	state, err := recording.Reduce(read(), math.MaxInt64)
	if err != nil {
		t.Fatal(err)
	}
	projection := state.Sketches.Snapshot()
	if len(projection.Sketches) != 3 || projection.Sketches[0].Label != "Move this console control" || projection.Sketches[1].Label != "Preview focus draft" || projection.Sketches[2].Label != "New after submitted feedback" {
		t.Fatalf("console projection: %+v", projection)
	}

	// A same-context SPA swaps its head/body and app root, without replacing
	// the Document or manually reattaching any private operator UI. The actual
	// recorder/control hooks must prepare a new physical drawing surface.
	selectedBeforeSPA, e := cast.SubmissionSelection()
	if e != nil {
		t.Fatal(e)
	}
	for _, v := range selectedBeforeSPA.Sketches {
		addMask(v)
	}
	if e = cast.SetDeliveredFeedback(deliveredMask); e != nil {
		t.Fatal(e)
	}
	eval(0, `(()=>{globalThis.__testSPADocument=document;history.pushState({screen:'next'},'', '/spa-next');const head=document.createElement('head');head.innerHTML='<title>SPA next screen</title><style>body{margin:0}button{position:absolute;left:80px;top:100px;width:240px;height:160px;padding:0;border:0;background:#eee}</style>';const body=document.createElement('body');body.innerHTML='<main id="spa-screen"><button>Continue</button><p>Next page content</p></main>';document.head.replaceWith(head);document.body.replaceWith(body);window.dispatchEvent(new PopStateEvent('popstate'));return document===globalThis.__testSPADocument})()`)
	wait("SPA source mutation recorded", func() bool {
		for _, r := range read() {
			data, e := recording.Encode(r)
			if e == nil && bytes.Contains(data, []byte("spa-screen")) {
				return true
			}
		}
		return false
	})
	if got := string(eval(0, `document===globalThis.__testSPADocument&&location.pathname==='/spa-next'&&document.querySelector('main button').textContent==='Continue'`)); got != "true" {
		t.Fatalf("SPA changed native document/context unexpectedly: %s", got)
	}
	diagnosePostSubmit()
	t.Logf("sanitized same-context head/body/root SPA before Sketch: %s", postSubmitDiagnostic)
	eval(world, `(()=>{const original=globalThis.__couchcastSketchReply;globalThis.__testSPAReplies=[];globalThis.__couchcastSketchReply=reply=>{__testSPAReplies.push({request:reply.request,ok:reply.ok,error:reply.error,outcome:reply.value?.outcome,status:reply.value?.status});if(__testSPAReplies.length>20)__testSPAReplies.shift();return original(reply)};return true})()`)
	clickTool("Sketch")
	wait("SPA held scene drawing", func() bool {
		diagnosePostSubmit()
		phase := string(eval(world, `__couchcastOperatorSketch.state().phase`))
		if phase == `"interrupted"` || string(eval(world, `__testSPAReplies.some(r=>r.ok===false)`)) == "true" {
			rs := read()
			for i, r := range rs {
				if i >= len(rs)-12 {
					switch v := r.(type) {
					case *recording.SketchBegin:
						t.Logf("SPA lastrecord[%d] %s t=%d id=%s sourceDoc=%d tab=%s", i, r.Kind(), r.Time(), v.ID, v.Doc, v.Tab)
					case *recording.SketchInterrupt:
						t.Logf("SPA lastrecord[%d] %s t=%d id=%s reason=%s", i, r.Kind(), r.Time(), v.Sketch, v.Reason)
					default:
						t.Logf("SPA lastrecord[%d] %s t=%d", i, r.Kind(), r.Time())
					}
				}
			}
			t.Fatalf("SPA scene preparation refused: transport=%s; drawing=%s; boardDoc=%d", eval(world, `__testSPAReplies`), postSubmitDiagnostic, cast.board.OperatorView().Doc)
		}
		return phase == `"drawing"`
	})
	t.Logf("SPA real pen interception before input: %s", postSubmitDiagnostic)
	pen("mousePressed", 110, .2, 1)
	pen("mouseMoved", 115, .5, 1)
	pen("mouseMoved", 120, .8, 1)
	wait("SPA new accepted visible pressure path", func() bool {
		diagnosePostSubmit()
		state, e := recording.Reduce(read(), math.MaxInt64)
		if e != nil {
			return false
		}
		sketches := state.Sketches.Snapshot().Sketches
		return len(sketches) == 4 && len(sketches[3].Strokes) == 1 && len(sketches[3].Strokes[0].Pts) >= 3 && pixels().Count > 0
	})
	pen("mouseReleased", 120, 0, 0)
	wait("SPA pressure path settled", func() bool {
		return string(eval(world, `__couchcastOperatorSketch.state().slots.every(s=>!s.inflight&&!s.queued)`)) == "true"
	})
	clickTool("Note")
	wait("SPA sketch Note editor", func() bool {
		return string(eval(world, `!__couchcastOperatorSketch.labelInput.parentElement.hidden`)) == "true"
	})
	eval(world, `__couchcastOperatorSketch.labelInput.value='Sketch after SPA replacement'`)
	clickNativeRootButton(".sketch-label button", "Save label")
	wait("SPA sketch label accepted and scene exited", func() bool {
		diagnosePostSubmit()
		state, e := recording.Reduce(read(), math.MaxInt64)
		return e == nil && len(state.Sketches.Snapshot().Sketches) == 4 && state.Sketches.Snapshot().Sketches[3].Label == "Sketch after SPA replacement" && string(eval(world, `__couchcastOperatorSketch.state().slots.every(s=>!s.inflight&&!s.queued)&&!globalThis.__couchcastPrivateSceneFrame`)) == "true"
	})
	previewDelivered = deliveredMask.Sketches
	publishPreview(4)
	if got := string(eval(world, `(()=>{const p=__couchcastOperatorSketch.labelInput.getRootNode().querySelector('.sketch-submission-preview');return !p.hidden&&p.querySelectorAll('.sketch-submission-card').length===1&&p.textContent.includes('Sketch after SPA replacement')})()`)); got != "true" {
		t.Fatalf("SPA new sketch pending card missing: %s; %s", got, postSubmitDiagnostic)
	}
	t.Logf("same-context SPA head/body/root replacement then real pen/Note/save/pending passed")
	for _, r := range records {
		b, e := recording.Encode(r)
		if e != nil {
			t.Fatal(e)
		}
		if strings.Contains(string(b), "blob:null/") || strings.Contains(string(b), "couch-sketch=1") {
			t.Fatalf("private source leaked into recording: %s", b)
		}
	}
}

// An accepted open operator contact is sealed by the ordered handoff, not by
// a pointer-up. That authoritative seal must permit the old console's label
// even while the new holder starts an independent sketch.
func TestOperatorSketchLiveOpenContactHandoffLabel(t *testing.T) {
	if os.Getenv("WEBCAST_LIVE_CHROME") == "" {
		t.Skip("set WEBCAST_LIVE_CHROME=1")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	page := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<!doctype html><title>open-contact handoff</title><style>body{margin:0}button{position:absolute;left:80px;top:100px;width:240px;height:160px;padding:0;border:0}</style><button>Save</button>`)
	}))
	defer page.Close()
	binary, err := FindChrome(os.Getenv, execLookPath, fileExists)
	if err != nil {
		t.Fatal(err)
	}
	profile, root := t.TempDir(), filepath.Join(t.TempDir(), "recordings")
	var chrome *Chrome
	cast, err := StartCast(ctx, CastConfig{RecordingsDir: root, Host: broadcast.NewHost(broadcast.HostConfig{Tunnel: &broadcast.FakeTunnel{}}), Logf: t.Logf, Launch: func(ctx context.Context, opts *RecordOptions) (Browser, error) {
		opts.ShotEvery = -1
		var err error
		chrome, err = Launch(ctx, LaunchOptions{Binary: binary, Headless: true, ProfileDir: profile, StartURL: page.URL, Record: opts})
		return chrome, err
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer cast.End()
	stop := context.AfterFunc(ctx, chrome.proc.Kill)
	defer stop()
	session := (liveHarness{chrome}).session(t, "")
	call := func(method string, params, result any) {
		t.Helper()
		if err := chrome.conn.Call(ctx, session, method, params, result); err != nil {
			t.Fatal(method, err)
		}
	}
	eval := func(world int64, expression string) json.RawMessage {
		t.Helper()
		var result struct {
			Result struct {
				Value json.RawMessage `json:"value"`
			} `json:"result"`
			Exception json.RawMessage `json:"exceptionDetails"`
		}
		params := map[string]any{"expression": expression, "returnByValue": true, "awaitPromise": true}
		if world != 0 {
			params["contextId"] = world
		}
		call("Runtime.evaluate", params, &result)
		if len(result.Exception) > 0 {
			t.Fatalf("evaluate %s: %s", expression, result.Exception)
		}
		return result.Result.Value
	}
	wait := func(what string, condition func() bool) {
		t.Helper()
		for deadline := time.Now().Add(8 * time.Second); time.Now().Before(deadline); {
			if condition() {
				return
			}
			select {
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			case <-time.After(25 * time.Millisecond):
			}
		}
		t.Fatalf("waiting for %s", what)
	}
	call("Emulation.setDeviceMetricsOverride", map[string]any{"width": 800, "height": 600, "deviceScaleFactor": 1, "mobile": false}, nil)
	wait("fixture document", func() bool {
		return string(eval(0, `document.readyState==='complete'&&!!document.querySelector('button')`)) == "true"
	})
	if err = chrome.SetLive(true); err != nil {
		t.Fatal(err)
	}
	if err = cast.board.SetAnnotationAllowed(ctx, true); err != nil {
		t.Fatal(err)
	}
	controller := cast.Drive()
	if controller == nil {
		t.Fatal("real cast controller missing")
	}
	if err = controller.SetDriveAllowed(ctx, true); err != nil {
		t.Fatal(err)
	}
	var world int64
	wait("operator world", func() bool {
		done := make(chan struct{})
		chrome.do(func(state *loopState) { world = state.contexts[session]; close(done) })
		<-done
		return world != 0
	})
	wait("recorded live document", func() bool { return cast.board.OperatorView().Doc > 0 })
	wait("operator drawing authority", func() bool {
		return string(eval(world, `!!globalThis.__couchcastOperatorSketch?.state().allowed`)) == "true"
	})

	defer func() {
		if t.Failed() {
			t.Logf("operator handoff diagnostic: %s", eval(world, `JSON.stringify({state:__couchcastOperatorSketch.state(),ui:__couchcastOperatorSketch.labelInput.getRootNode().querySelector('.sketch-status')?.textContent})`))
		}
	}()
	eval(world, `(async()=>{await __couchcastOperatorSketch.enter();return true})()`)
	wait("prepared operator scene", func() bool { return string(eval(world, `__couchcastOperatorSketch.state().phase`)) == `"drawing"` })
	var oldID string
	if err = json.Unmarshal(eval(world, `__couchcastOperatorSketch.state().active`), &oldID); err != nil || oldID == "" {
		t.Fatalf("operator sketch id %q %v", oldID, err)
	}
	pen := func(kind string, x, pressure float64) {
		t.Helper()
		button := "none"
		if kind == "mousePressed" {
			button = "left"
		}
		call("Input.dispatchMouseEvent", map[string]any{"type": kind, "x": x, "y": 140, "button": button, "buttons": 1, "clickCount": 1, "pointerType": "pen", "force": pressure}, nil)
	}
	pen("mousePressed", 100, .2)
	pen("mouseMoved", 115, .5)
	pen("mouseMoved", 130, .8)
	var records []recording.Record
	read := func() *recording.State {
		t.Helper()
		rs, err := readSubmissionRecording(ctx, cast.RecordingPath())
		if err != nil {
			return nil
		}
		records = rs
		state, err := recording.Reduce(rs, math.MaxInt64)
		if err != nil {
			t.Fatal(err)
		}
		return state
	}
	wait("accepted open pressure contact", func() bool {
		state := read()
		if state == nil {
			return false
		}
		view := state.Sketches.Snapshot()
		return len(view.Sketches) == 1 && len(view.Sketches[0].Strokes) == 1 && len(view.Sketches[0].Strokes[0].Pts) >= 3 && !view.Sketches[0].Strokes[0].Sealed
	})
	wait("last open contact reply settled", func() bool {
		return string(eval(world, `__couchcastOperatorSketch.state().slots.every(s=>!s.inflight&&!s.queued)`)) == "true"
	})
	for _, record := range records {
		if record.Kind() == "sketch-stroke-end" {
			t.Fatal("fixture released pointer before handoff")
		}
	}
	seat, err := controller.Join("Next holder")
	if err != nil {
		t.Fatal(err)
	}
	detach, err := controller.Attach(seat.Viewer.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer detach()
	if err = controller.Request(seat.Key); err != nil {
		t.Fatal(err)
	}
	if err = controller.Grant(seat.Viewer.ID); err != nil {
		t.Fatal(err)
	}
	wait("authoritative handoff seals partial contact", func() bool {
		state := read()
		if state == nil {
			return false
		}
		view := state.Sketches.Snapshot()
		if len(view.Sketches) != 1 || len(view.Sketches[0].Strokes) != 1 {
			return false
		}
		sketch := view.Sketches[0]
		metadata, _ := state.Sketches.Metadata(oldID)
		return sketch.Begin.ID == oldID && sketch.Phase == "interrupted" && sketch.Reason == recording.SketchInterruptHandoff && metadata.OpenStroke == "" && sketch.Strokes[0].Sealed && sketch.Strokes[0].Truncated
	})
	authority := controller.DrawingAuthority()
	permit, err := controller.AdmitDrawing(authority.Epoch, seat.Key)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := controller.AdmitDrawingMetadata(seat.Key)
	if err != nil {
		t.Fatal(err)
	}
	annotation, err := cast.board.AdmitAnnotation()
	if err != nil {
		t.Fatal(err)
	}
	ownerHash := sha256.Sum256([]byte(seat.Key))
	nextAuthority := board.SketchAuthority{Author: permit.Principal(), Owner: hex.EncodeToString(ownerHash[:]), Epoch: authority.Epoch, Annotation: annotation, Check: permit.Check, MetadataCheck: identity.Check}
	next := cast.sketches.Execute(ctx, nextAuthority, board.SketchCommand{Kind: "sketch-begin", Operation: "next-holder-begin", Epoch: authority.Epoch, Seg: cast.board.OperatorView().Seg, Color: 1})
	if next.Outcome != board.SketchAccepted || next.Sketch == oldID {
		t.Fatalf("new holder blocked by old open contact: %+v", next)
	}
	wait("old console loses drawing authority", func() bool { return string(eval(world, `!__couchcastOperatorSketch.state().allowed`)) == "true" })
	// This is the ordinary label UI path; it must use journal-authoritative
	// sealing rather than retaining open_stroke from the last points reply.
	encodedID, _ := json.Marshal(oldID)
	eval(world, `(()=>{const tool=__couchcastOperatorSketch;tool.openLabel(`+string(encodedID)+`);tool.labelInput.value='Partial before handoff';return true})()`)
	// The journal is asynchronous. A temporary confirmation gate is valid;
	// repeatedly pressing the ordinary Save action must cease being blocked
	// once its authoritative seal arrives. Acceptance clears labelTarget, so
	// later polls cannot submit another label mutation.
	wait("old console label persisted during new-holder drawing", func() bool {
		eval(world, `(async()=>{await __couchcastOperatorSketch.saveLabel();return true})()`)
		state := read()
		if state == nil {
			return false
		}
		view := state.Sketches.Snapshot()
		if len(view.Sketches) != 2 {
			return false
		}
		return view.Sketches[0].Begin.ID == oldID && view.Sketches[0].Label == "Partial before handoff" && view.Active == next.Sketch && view.Sketches[1].Phase == "drawing"
	})
	if err = permit.Check(); err != nil {
		t.Fatalf("old metadata label revoked new holder: %v", err)
	}
	if current := controller.DrawingAuthority(); current.Holder != seat.Viewer.ID || current.Epoch != authority.Epoch {
		t.Fatalf("late label changed new holder authority: %+v", current)
	}
	// A valid permit must still reach the writer/source seam after the old
	// label: keeping the controller witness alone would miss a lost scene lease.
	current := read()
	if current == nil {
		t.Fatal("missing post-label recording")
	}
	oldAnchor := current.Sketches.Snapshot().Sketches[0].Strokes[0].Begin.At
	stroke := cast.sketches.Execute(ctx, nextAuthority, board.SketchCommand{Kind: "sketch-stroke-begin", Operation: "next-holder-contact", Sequence: 1, Epoch: authority.Epoch, Sketch: next.Sketch, ID: "next-contact", Node: oldAnchor.Node, Box: *oldAnchor.Box})
	if stroke.Outcome != board.SketchAccepted {
		t.Fatalf("old label blocked new holder source/writer: %+v", stroke)
	}
}
