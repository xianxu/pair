package webcast

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	core "github.com/xianxu/pair/cmd/internal/submission"
	"github.com/xianxu/pair/cmd/internal/webcast/notedraft"
	"github.com/xianxu/pair/cmd/internal/webcast/recorder"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

type measuredDraftBrowser struct {
	*Chrome
	mu    sync.Mutex
	times []string
}

func (b *measuredDraftBrowser) CaptureCrops(ctx context.Context, doc int, regions []core.CaptureRegion, report func(core.SubmissionProgress)) ([]recorder.CapturedCrop, error) {
	start, last := time.Now(), time.Now()
	phase := "context"
	var spans []string
	out, err := b.Chrome.CaptureCrops(ctx, doc, regions, func(p core.SubmissionProgress) {
		now := time.Now()
		spans = append(spans, fmt.Sprintf("%s=%.1fms", phase, float64(now.Sub(last).Microseconds())/1000))
		last = now
		phase = p.Phase
		if report != nil {
			report(p)
		}
	})
	spans = append(spans, fmt.Sprintf("%s=%.1fms total=%.1fms", phase, float64(time.Since(last).Microseconds())/1000, float64(time.Since(start).Microseconds())/1000))
	b.mu.Lock()
	b.times = append(b.times, fmt.Sprint(spans))
	b.mu.Unlock()
	return out, err
}

// Real Chrome, fresh profile, fake authenticated viewer context. The test uses
// the production HTTP/isolated-binding adapters; no operator browser is touched.
func TestLiveDrawCaptureProtocolsAndStoredSubmission(t *testing.T) {
	if os.Getenv("WEBCAST_LIVE_CHROME") == "" {
		t.Skip("set WEBCAST_LIVE_CHROME=1")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	page := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<!doctype html><title>draw conformance</title><style>body{margin:0;background:black}#card{position:absolute;left:80px;top:100px;width:240px;height:160px;background:white;color:black}.fade{animation:enter 600ms linear both}@keyframes enter{from{opacity:0}to{opacity:1}}</style><div id="card">original card</div>`)
	}))
	defer page.Close()
	bin, err := FindChrome(os.Getenv, execLookPath, fileExists)
	if err != nil {
		t.Fatal(err)
	}
	profile := t.TempDir()
	root := filepath.Join(t.TempDir(), "recordings")
	var browser *measuredDraftBrowser
	cast, err := StartCast(ctx, CastConfig{RecordingsDir: root, Host: broadcast.NewHost(broadcast.HostConfig{Tunnel: &broadcast.FakeTunnel{}}), Logf: t.Logf, Launch: func(ctx context.Context, opts *RecordOptions) (Browser, error) {
		opts.ShotEvery = -1
		chrome, err := Launch(ctx, LaunchOptions{Binary: bin, ProfileDir: profile, Headless: true, Record: opts})
		if err != nil {
			return nil, err
		}
		browser = &measuredDraftBrowser{Chrome: chrome}
		return browser, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer cast.End()
	stop := context.AfterFunc(ctx, browser.proc.Kill)
	defer stop()
	session := (liveHarness{browser.Chrome}).session(t, "")
	call := func(method string, params, result any) {
		t.Helper()
		if err := browser.conn.Call(ctx, session, method, params, result); err != nil {
			t.Fatal(method, err)
		}
	}
	evaluate := func(world int64, expression string, out any) {
		t.Helper()
		params := map[string]any{"expression": expression, "returnByValue": true, "awaitPromise": true}
		if world != 0 {
			params["contextId"] = world
		}
		var response struct {
			Result struct {
				Value json.RawMessage `json:"value"`
			} `json:"result"`
			Exception json.RawMessage `json:"exceptionDetails"`
		}
		call("Runtime.evaluate", params, &response)
		if len(response.Exception) > 0 {
			t.Fatalf("eval: %s", response.Exception)
		}
		if out != nil {
			if err := json.Unmarshal(response.Result.Value, out); err != nil {
				t.Fatal(err, string(response.Result.Value))
			}
		}
	}
	pause := func() {
		t.Helper()
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-time.After(25 * time.Millisecond):
		}
	}
	call("Emulation.setDeviceMetricsOverride", map[string]any{"width": 800, "height": 600, "deviceScaleFactor": 1, "mobile": false}, nil)
	call("Page.navigate", map[string]any{"url": page.URL + "/"}, nil)
	for {
		var ready bool
		evaluate(0, `document.readyState==='complete'&&!!document.getElementById('card')`, &ready)
		if ready {
			break
		}
		pause()
	}
	if err = browser.SetLive(true); err != nil {
		t.Fatal(err)
	}
	var node recording.NodeID
	var doc int
	for node == 0 {
		recs, e := readSubmissionRecording(ctx, cast.RecordingPath())
		if e == nil && len(recs) > 0 {
			state, e := recording.Reduce(recs, recs[len(recs)-1].Time())
			if e == nil && state.Live {
				doc = state.DocSeg
				var walk func(*recording.DOMNode)
				walk = func(n *recording.DOMNode) {
					if n == nil {
						return
					}
					if n.A["id"] == "card" {
						node = n.ID
					}
					for _, k := range n.Kids {
						walk(k)
					}
				}
				walk(state.Top)
			}
		}
		if node == 0 {
			pause()
		}
	}
	if err = cast.board.SetAnnotationAllowed(ctx, true); err != nil {
		t.Fatal(err)
	}
	generation := strconv.FormatUint(cast.board.AnnotationCapability().Generation, 10)
	type privateViewerKey struct{}
	server := NewServer("view", browser, time.Second, nil, nil).WithBoard(cast.board).WithPointer("annotate").WithNoteDrafts(cast.noteDrafts, func(r *http.Request) string { value, _ := r.Context().Value(privateViewerKey{}).(string); return value })
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		server.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), privateViewerKey{}, "private-cookie-proof")))
	}))
	defer httpServer.Close()
	viewerCall := func(route string, body any) notedraft.Status {
		t.Helper()
		data, _ := json.Marshal(body)
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, httpServer.URL+"/annotate/"+route, bytes.NewReader(data))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Couch-Annotation-Generation", generation)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			var reason bytes.Buffer
			reason.ReadFrom(resp.Body)
			t.Fatalf("%s %d: %s", route, resp.StatusCode, reason.String())
		}
		var status notedraft.Status
		if err = json.NewDecoder(resp.Body).Decode(&status); err != nil {
			t.Fatal(err)
		}
		return status
	}
	start := time.Now()
	status := viewerCall("prepare-note", map[string]any{"request": "viewer-prepare", "seg": doc, "node": node, "box": []int{80, 100, 240, 160}, "region": []int{90, 110, 100, 80}})
	ack := time.Since(start)
	for status.State == "pending" {
		time.Sleep(250 * time.Millisecond)
		status = viewerCall("draft-status", map[string]any{"request": "viewer-prepare", "draft": status.Draft})
	}
	if status.State != "ready" || status.Unavailable != "" {
		t.Fatal(status)
	}
	t.Logf("static viewer prepare_ack=%.1fms ready_including_store_and_poll=%.1fms", float64(ack.Microseconds())/1000, float64(time.Since(start).Microseconds())/1000)
	start = time.Now()
	status = viewerCall("save-note", map[string]any{"request": "viewer-save", "draft": status.Draft, "viewer": map[string]string{"id": "viewer", "name": "Viewer"}, "color": 0, "text": "viewer note"})
	for status.State == "pending" {
		time.Sleep(500 * time.Millisecond)
		status = viewerCall("draft-status", map[string]any{"request": "viewer-prepare", "draft": status.Draft})
	}
	if status.State != "saved" {
		t.Fatal(status)
	}
	viewerNoteID := status.NoteID
	t.Logf("viewer save_confirmation_including_500ms_poll=%.1fms", float64(time.Since(start).Microseconds())/1000)
	var world int64
	for world == 0 {
		done := make(chan struct{})
		browser.do(func(s *loopState) { world = s.contexts[session]; close(done) })
		<-done
		if world == 0 {
			pause()
		}
	}
	for {
		var ready bool
		evaluate(world, `typeof __couchcastOperatorReply==='function'&&globalThis.__couchcastAnnotationDelivery?.allowed===true`, &ready)
		if ready {
			break
		}
		pause()
	}
	evaluate(world, `globalThis.testReplies=[];globalThis.realReply=__couchcastOperatorReply;globalThis.__couchcastOperatorReply=r=>{testReplies.push(r);realReply(r)}`, nil)
	opSerial := 0
	operatorCall := func(op, request string, body any) recorder.OperatorReply {
		t.Helper()
		raw, _ := json.Marshal(map[string]any{"t": "operator", "request": map[string]any{"op": op, "request": request, "generation": generation, "body": body}})
		quoted, _ := json.Marshal(string(raw))
		evaluate(world, "__couchcast("+string(quoted)+")", nil)
		for {
			var reply *recorder.OperatorReply
			id, _ := json.Marshal(request)
			evaluate(world, "testReplies.find(r=>r.request=="+string(id)+")||null", &reply)
			if reply != nil {
				return *reply
			}
			pause()
		}
	}
	var path []int
	evaluate(world, `__couchcastPath(document.getElementById('card'))`, &path)
	evaluate(0, `document.getElementById('card').classList.add('fade')`, nil)
	start = time.Now()
	reply := operatorCall("prepare-note", "operator-prepare", map[string]any{"seg": doc, "path": path, "box": []int{80, 100, 240, 160}, "region": []int{90, 110, 100, 80}})
	ack = time.Since(start)
	for reply.State == "pending" {
		opSerial++
		reply = operatorCall("draft-status", fmt.Sprintf("poll-%d", opSerial), map[string]any{"draft": "operator-prepare"})
	}
	if reply.State != "ready" || reply.Unavailable != "" {
		t.Fatal(reply)
	}
	t.Logf("animated operator prepare_ack=%.1fms ready_including_store_and_poll=%.1fms", float64(ack.Microseconds())/1000, float64(time.Since(start).Microseconds())/1000)
	// Removing the original after its capture must not reanchor or refuse Save.
	evaluate(0, `globalThis.savedCard=document.getElementById('card');savedCard.remove()`, nil)
	if err = browser.conn.SyncEvents(ctx); err != nil {
		t.Fatal(err)
	}
	start = time.Now()
	reply = operatorCall("save-note", "operator-save", map[string]any{"draft": reply.Draft, "color": 1, "text": "operator note"})
	for reply.State == "pending" {
		opSerial++
		reply = operatorCall("draft-status", fmt.Sprintf("poll-%d", opSerial), map[string]any{"draft": "operator-prepare"})
	}
	if reply.State != "saved" {
		t.Fatal(reply)
	}
	t.Logf("operator save_after_element_removed=%.1fms", float64(time.Since(start).Microseconds())/1000)
	// Exercise real DOM placement, independent of the historical recording IDs.
	note, ok := cast.board.ObservedNote(viewerNoteID)
	if !ok {
		t.Fatal("viewer note missing")
	}
	noteJSON, _ := json.Marshal(note)
	anchorSource, _ := webFS.ReadFile("web/annotation-anchor.js")
	anchorModule, err := assembleOperatorModule(string(anchorSource), nil)
	if err != nil {
		t.Fatal(err)
	}
	placementSource, _ := webFS.ReadFile("web/annotation-placement.js")
	placementModule, err := assembleOperatorModule(string(placementSource), map[string]string{"./annotation-anchor.js": "anchor"})
	if err != nil {
		t.Fatal(err)
	}
	evaluate(0, "globalThis.place=(function(){const anchor="+anchorModule+";const placement="+placementModule+";return placement.createPlacement({document:()=>document,route:()=>location.href})})();globalThis.placedNote="+string(noteJSON), nil)
	placement := func(want bool) {
		t.Helper()
		var got bool
		evaluate(0, `!!place(placedNote)`, &got)
		if got != want {
			t.Fatalf("placement=%v want %v", got, want)
		}
	}
	placement(false)
	evaluate(0, `document.body.appendChild(savedCard);savedCard.classList.remove('fade')`, nil)
	placement(true)
	evaluate(0, `savedCard.style.display='none'`, nil)
	placement(false)
	evaluate(0, `savedCard.style.display=''`, nil)
	placement(true)
	evaluate(0, `globalThis.duplicate=savedCard.cloneNode(true);document.body.appendChild(duplicate)`, nil)
	placement(false)
	evaluate(0, `duplicate.remove();savedCard.style.background='rgb(255,0,0)';savedCard.textContent='mutated after draw'`, nil)
	selection, err := cast.SubmissionSelection()
	if err != nil || len(selection.Notes) != 2 {
		t.Fatal(selection, err)
	}
	start = time.Now()
	sub, crops, err := cast.PrepareSubmission(ctx, "live-stored", castSubmissionTarget(), selection, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("submit_read_claim_build=%.1fms", float64(time.Since(start).Microseconds())/1000)
	if len(crops) != 2 {
		t.Fatal(sub)
	}
	for _, n := range sub.Notes {
		if !strings.Contains(n.Element, "original card") || !strings.Contains(n.Context.Text, "original card") || strings.Contains(n.Element, "mutated after draw") {
			t.Fatalf("context not from draw moment: %+v", n)
		}
		img, err := png.Decode(bytes.NewReader(crops[n.Crop.File]))
		if err != nil {
			t.Fatal(err)
		}
		r, g, b, _ := img.At(img.Bounds().Dx()/2, img.Bounds().Dy()/2).RGBA()
		if r < 60000 || g < 60000 || b < 60000 {
			t.Fatalf("crop is changed/translucent: %d %d %d", r, g, b)
		}
	}
	store, err := core.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	start = time.Now()
	if _, err = store.Prepare(sub, crops); err != nil {
		t.Fatal(err)
	}
	t.Logf("bundle_publish=%.1fms", float64(time.Since(start).Microseconds())/1000)
	obs, err := store.ConfirmPrepared(sub, crops)
	if err != nil {
		t.Fatal(err)
	}
	start = time.Now()
	if err = cast.ReconcilePreparedSubmission(ctx, sub, obs); err != nil {
		t.Fatal(err)
	}
	t.Logf("bundle_rebind_cleanup=%.1fms", float64(time.Since(start).Microseconds())/1000)
	browser.mu.Lock()
	for i, span := range browser.times {
		t.Logf("draw_capture_%d %s", i+1, span)
	}
	if len(browser.times) != 2 {
		t.Error("submission recaptured", len(browser.times))
	}
	browser.mu.Unlock()
}
