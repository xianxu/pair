package webcast

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
)

// Explicit measurement only: this publishes an owned synthetic fixture through
// an anonymous quick tunnel. It never reads a user's browser or named tunnel.
func TestSketchAnchorLatencyLive(t *testing.T) {
	if os.Getenv("WEBCAST_MEASURE_SKETCH_ANCHOR") != "1" {
		t.Skip("set WEBCAST_MEASURE_SKETCH_ANCHOR=1 and WEBCAST_LIVE_CHROME=1")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	samples := map[string][]float64{}
	base := os.TempDir()
	if st, e := os.Stat("/Volumes/TestScratch/tmp"); e == nil && st.IsDir() {
		base = "/Volumes/TestScratch/tmp"
	}
	out, err := os.MkdirTemp(base, "sketch-anchor-latency-")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		data, _ := json.MarshalIndent(samples, "", "  ")
		_ = os.WriteFile(filepath.Join(out, "samples.json"), data, 0600)
		t.Log("raw samples:", out)
		for name, raw := range samples {
			v := append([]float64(nil), raw...)
			sort.Float64s(v)
			n := len(v)
			if n == 0 {
				continue
			}
			over := 0
			for _, x := range v {
				if x > 150 {
					over++
				}
			}
			t.Logf("%s n=%d p50=%.3fms p95=%.3fms max=%.3fms >150ms=%d/%d", name, n, v[n/2], v[(n-1)*95/100], v[n-1], over, n)
		}
	}()
	f := newSketchHTTPFixture(t)
	defer f.hub.Close()
	epoch := f.grant(t, f.a)
	measureViewer := func(name, baseURL string) {
		p := headlessPage(t)
		defer func() {
			cleanup, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = p.conn.Call(cleanup, p.session, "Page.navigate", map[string]any{"url": "about:blank"}, nil)
			p.conn.Close()
		}()
		call := func(method string, params, result any) {
			t.Helper()
			if e := p.conn.Call(ctx, p.session, method, params, result); e != nil {
				t.Fatal(e)
			}
		}
		eval := func(expr string) json.RawMessage {
			t.Helper()
			var r struct {
				Result           struct{ Value json.RawMessage }
				ExceptionDetails json.RawMessage
			}
			call("Runtime.evaluate", map[string]any{"expression": expr, "awaitPromise": true, "returnByValue": true}, &r)
			if len(r.ExceptionDetails) > 0 {
				t.Fatalf("viewer %s: %s", name, r.ExceptionDetails)
			}
			return r.Result.Value
		}
		call("Page.navigate", map[string]any{"url": baseURL + "/driver/"}, nil)
		for string(eval("document.readyState==='complete'&&location.pathname==='/driver/'")) != "true" {
			select {
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			case <-time.After(20 * time.Millisecond):
			}
		}
		setup := fmt.Sprintf(`(async()=>{const {createSketchGesture}=await import('./sketch-policy.js');const {createSketchScene}=await import('./sketch-scene.js');const {createViewerSketchTransport}=await import('./viewer.js');const transport=createViewerSketchTransport((k,o)=>fetch(new URL(k,location.href),o),{key:%q},()=>%d);const epoch=%d;const b=await transport.send('sketch-begin',{operation:%q,sequence:0,epoch,seg:1,color:0});if(b.outcome!=='accepted')throw Error(JSON.stringify(b));const host=document.createElement('div');document.body.append(host);const scene=createSketchScene({container:host,readSource:transport.readSource});const start=performance.now();await scene.prepare(b.source,{sketch:b.sketch,epoch});globalThis.measure={transport,scene,b,epoch,createSketchGesture};return performance.now()-start})()`, f.a.Key, f.bd.AnnotationCapability().Generation, epoch, "measure-"+name)
		var prep float64
		if e := json.Unmarshal(eval(setup), &prep); e != nil {
			t.Fatal(e)
		}
		samples[name+" source preparation"] = []float64{prep}
		var hits []float64
		if e := json.Unmarshal(eval(`(async()=>{const a=[];for(let i=0;i<500;i++){const s=performance.now();const h=await measure.scene.hitTest([20,20]);if(!h.node)throw Error('missing anchor');a.push(performance.now()-s)}return a})()`), &hits); e != nil {
			t.Fatal(e)
		}
		samples[name+" ready viewer hit-test"] = hits
		var tokens []float64
		if e := json.Unmarshal(eval(`(async()=>{const a=[];const g=measure.createSketchGesture();for(let i=0;i<500;i++){const start=performance.now();g.start("token-"+i,[[20,20,.5]]);const p=Promise.resolve().then(()=>measure.scene.hitTest([20,20]));g.end();const effects=g.anchor(await p,"token-"+i);if(g.snapshot().active||effects.at(-1)?.type!=="stroke-end")throw Error("token did not drain");a.push(performance.now()-start)}return a})()`), &tokens); e != nil {
			t.Fatal(e)
		}
		samples[name+" viewer released-contact token drain"] = tokens
		var acks struct{ Begin, End []float64 }
		if e := json.Unmarshal(eval(`(async()=>{const {scene,transport,b,epoch}=measure;const begin=[],end=[];for(let i=0;i<30;i++){const h=await scene.hitTest([20,20]);let s=performance.now();let r=await transport.send('sketch-stroke-begin',{operation:'begin-'+i,sequence:1+2*i,epoch,sketch:b.sketch,id:'stroke-'+i,node:h.node,box:h.box});begin.push(performance.now()-s);if(r.outcome!=='accepted')throw Error(JSON.stringify(r));s=performance.now();r=await transport.send('sketch-stroke-end',{operation:'end-'+i,sequence:2+2*i,epoch,sketch:b.sketch,id:'stroke-'+i});end.push(performance.now()-s);if(r.outcome!=='accepted')throw Error(JSON.stringify(r))}return {begin,end}})()`), &acks); e != nil {
			t.Fatal(e)
		}
		samples[name+" stroke-begin ACK"] = acks.Begin
		samples[name+" terminal ACK"] = acks.End
		eval(`(async()=>{const {transport,b,epoch}=measure;const r=await transport.send("sketch-label",{operation:"measure-label",sequence:61,epoch,sketch:b.sketch,label:"latency fixture"});if(r.outcome!=="accepted")throw Error(JSON.stringify(r));measure.scene.dispose();return true})()`)
		t.Logf("%s viewer samples complete", name)
	}
	local := httptest.NewServer(f.s)
	defer local.Close()
	defer local.CloseClientConnections()
	measureViewer("local", local.URL)
	measureSketchConsoleLatency(t, ctx, samples)
	tunnel := broadcast.Cloudflared{RunDir: t.TempDir(), OpenTimeout: 25 * time.Second}
	listener, err := tunnel.Listen()
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	server := &http.Server{Handler: f.s}
	defer server.Close()
	go server.Serve(listener)
	handle, err := tunnel.Open(ctx, listener)
	if err != nil {
		t.Fatalf("anonymous tunnel unavailable (no fallback): %v", err)
	}
	defer handle.Close()
	stop := context.AfterFunc(ctx, func() { handle.Close() })
	defer stop()
	measureViewer("tunnel", handle.URL())
}

func measureSketchConsoleLatency(t *testing.T, ctx context.Context, samples map[string][]float64) {
	t.Helper()
	page := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<!doctype html><title>owned latency fixture</title><style>body{margin:0}button{position:absolute;left:80px;top:100px;width:240px;height:160px}</style><button>Save</button>`)
	}))
	defer page.Close()
	bin, err := FindChrome(os.Getenv, execLookPath, fileExists)
	if err != nil {
		t.Fatal(err)
	}
	var chrome *Chrome
	cast, err := StartCast(ctx, CastConfig{RecordingsDir: filepath.Join(t.TempDir(), "recordings"), Host: broadcast.NewHost(broadcast.HostConfig{Tunnel: &broadcast.FakeTunnel{}}), Launch: func(ctx context.Context, opts *RecordOptions) (Browser, error) {
		opts.ShotEvery = -1
		var e error
		chrome, e = Launch(ctx, LaunchOptions{Binary: bin, Headless: true, ProfileDir: t.TempDir(), StartURL: page.URL, Record: opts})
		return chrome, e
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
		if e := chrome.conn.Call(ctx, session, method, params, result); e != nil {
			t.Fatal(e)
		}
	}
	var world int64
	eval := func(expr string) json.RawMessage {
		t.Helper()
		var r struct {
			Result           struct{ Value json.RawMessage }
			ExceptionDetails json.RawMessage
		}
		p := map[string]any{"expression": expr, "awaitPromise": true, "returnByValue": true}
		if world != 0 {
			p["contextId"] = world
		}
		call("Runtime.evaluate", p, &r)
		if len(r.ExceptionDetails) > 0 {
			t.Fatalf("console: %s", r.ExceptionDetails)
		}
		return r.Result.Value
	}
	wait := func(fn func() bool) {
		t.Helper()
		for !fn() {
			select {
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			case <-time.After(20 * time.Millisecond):
			}
		}
	}
	call("Emulation.setDeviceMetricsOverride", map[string]any{"width": 800, "height": 600, "deviceScaleFactor": 1, "mobile": false}, nil)
	wait(func() bool {
		return string(eval("document.readyState==='complete'&&!!document.querySelector('button')")) == "true"
	})
	if err = chrome.SetLive(true); err != nil {
		t.Fatal(err)
	}
	if err = cast.board.SetAnnotationAllowed(ctx, true); err != nil {
		t.Fatal(err)
	}
	wait(func() bool { return cast.board.OperatorView().Doc > 0 })
	wait(func() bool {
		done := make(chan struct{})
		chrome.do(func(s *loopState) { world = s.contexts[session]; close(done) })
		<-done
		return world != 0
	})
	wait(func() bool { return string(eval("!!globalThis.__couchcastOperatorSketch?.state().allowed")) == "true" })
	eval("__couchcastOperatorSketch.enter()")
	wait(func() bool { return string(eval("__couchcastOperatorSketch.state().phase")) == `"drawing"` })
	policySource, e := webFS.ReadFile("web/sketch-policy.js")
	if e != nil {
		t.Fatal(e)
	}
	policyModule, e := assembleOperatorModule(string(policySource), nil)
	if e != nil {
		t.Fatal(e)
	}
	script := fmt.Sprintf(`(async()=>{const g=(%s).createSketchGesture();const old=__couchcastSketchReply;const pending=new Map();globalThis.__couchcastSketchReply=r=>{const p=pending.get(r.request);if(p){pending.delete(r.request);r.ok&&r.value?.node?p.resolve({ms:performance.now()-p.start,at:r.value}):p.reject(Error(JSON.stringify(r)))}old(r)};try{const a=[],token=[];for(let i=0;i<300;i++){const request='latency-'+i;const started=performance.now();g.start(request,[[100,140,.5]]);const pendingHit=new Promise((resolve,reject)=>{pending.set(request,{resolve,reject,start:performance.now()});__couchcast(JSON.stringify({t:'sketch',request:{request,kind:'scene-hit',generation:%q,epoch:__couchcastOperatorSketch.state().epoch,body:{point:[100,140]}}}))});g.end();const hit=await pendingHit;const effects=g.anchor(hit.at,request);if(g.snapshot().active||effects.at(-1)?.type!=='stroke-end')throw Error('token not drained');token.push(performance.now()-started);a.push(hit.ms)}return {rpc:a,token}}finally{globalThis.__couchcastSketchReply=old}})()`, policyModule, fmt.Sprint(cast.board.AnnotationCapability().Generation))
	var idle struct{ RPC, Token []float64 }
	if err = json.Unmarshal(eval(script), &idle); err != nil {
		t.Fatal(err)
	}
	samples["local console parent-worker-CDP hit-test"] = idle.RPC
	samples["local console released-contact token drain"] = idle.Token
	// Real pressure input keeps normal point writes and delivery on this worker.
	pen := func(kind string, x float64, buttons int) {
		button := "none"
		if kind != "mouseMoved" {
			button = "left"
		}
		call("Input.dispatchMouseEvent", map[string]any{"type": kind, "x": x, "y": 140, "button": button, "buttons": buttons, "pointerType": "pen", "force": .5}, nil)
	}
	before := cast.board.OperatorSketchCursor()
	pen("mousePressed", 100, 1)
	done := make(chan struct{})
	joined := make(chan struct{})
	defer func() {
		select {
		case <-done:
		default:
			close(done)
		}
		<-joined
	}()
	go func() {
		defer close(joined)
		ticker := time.NewTicker(time.Second / 60)
		defer ticker.Stop()
		i := 0
		for {
			select {
			case <-done:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				i++
				_ = chrome.conn.Call(ctx, session, "Input.dispatchMouseEvent", map[string]any{"type": "mouseMoved", "x": 100 + i%30, "y": 140, "button": "none", "buttons": 1, "pointerType": "pen", "force": .5}, nil)
			}
		}
	}()
	var busy struct{ RPC, Token []float64 }
	raw := eval(script)
	close(done)
	<-joined
	pen("mouseReleased", 130, 0)
	if err = json.Unmarshal(raw, &busy); err != nil {
		t.Fatal(err)
	}
	samples["local console CDP hit-test with real journal traffic"] = busy.RPC
	samples["local console busy released-contact token drain"] = busy.Token
	t.Logf("console normal journal advances during real pressure input: %d", cast.board.OperatorSketchCursor().Seq-before.Seq)
}
