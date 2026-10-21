package webcast

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
	castassets "github.com/xianxu/pair/cmd/internal/webcast/assets"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
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
	wait("operator world", func() bool {
		done := make(chan struct{})
		chrome.do(func(s *loopState) { world = s.contexts[session]; close(done) })
		<-done
		return world != 0
	})
	wait("operator live while annotation is off", func() bool {
		return string(eval(world, `!!globalThis.__couchcastOperatorSketch?.state().live`)) == "true"
	})
	if got := string(eval(world, `(()=>{const ui=__couchcastOperatorSketch,root=ui.labelInput.getRootNode(),bar=root.querySelector('.sketch-toolbar'),buttons=[...bar.querySelectorAll('button')];return !ui.state().allowed&&!bar.hidden&&getComputedStyle(bar).display!=='none'&&buttons.find(b=>b.textContent==='Sketch').disabled&&buttons.find(b=>b.textContent==='Done').disabled&&/annotation.*👆.*Chrome/.test(root.querySelector('.sketch-status').textContent)})()`)); got != "true" {
		t.Fatalf("annotation-off toolbar lacks actionable guidance or enables Done: %s", got)
	}
	if err = cast.board.SetAnnotationAllowed(ctx, true); err != nil {
		t.Fatal(err)
	}
	wait("operator sketch permission", func() bool {
		return string(eval(world, `!!globalThis.__couchcastOperatorSketch?.state().allowed`)) == "true"
	})
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
	eval(world, `(async()=>{await __couchcastOperatorSketch.enter();return true})()`)
	wait("prepared console scene", func() bool { return string(eval(world, `__couchcastOperatorSketch.state().phase`)) == `"drawing"` })
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
	eval(world, `(async()=>{await __couchcastOperatorSketch.undo();return true})()`)
	waitKind("sketch-undo")
	eval(world, `(async()=>{await __couchcastOperatorSketch.redo();return true})()`)
	waitKind("sketch-redo")
	eval(world, `(async()=>{const s=__couchcastOperatorSketch;await s.leave('watch');s.labelInput.value='Move this console control';await s.saveLabel();return true})()`)
	waitKind("sketch-label")
	wait("held scene exit", func() bool { return string(eval(world, `!globalThis.__couchcastPrivateSceneFrame`)) == "true" })
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
	if len(projection.Sketches) != 1 || projection.Sketches[0].Label != "Move this console control" {
		t.Fatalf("console projection: %+v", projection)
	}
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
