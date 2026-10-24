package webcast

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/webcast/notes"
	"github.com/xianxu/pair/cmd/internal/webcast/query"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// Disposable headless qualification of the recorder, Board and actual injected
// Notes UI. Reload has no subsequent page mutation to rescue missed navigation.
func TestPageLoadLiveFeedbackLifecycle(t *testing.T) {
	if os.Getenv("WEBCAST_LIVE_CHROME") == "" {
		t.Skip("set WEBCAST_LIVE_CHROME=1")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 110*time.Second)
	defer cancel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<!doctype html><title>page load feedback</title><style>body{margin:0;height:2400px}button{position:absolute;left:100px;top:120px;width:240px;height:120px}#app{position:absolute;left:100px;top:280px}</style><button id="anchor">Save</button><p id="app">Original app content</p>`)
	}))
	defer server.Close()
	pageURL := server.URL + "/"
	bin, err := FindChrome(os.Getenv, execLookPath, fileExists)
	if err != nil {
		t.Fatal(err)
	}
	var chrome *Chrome
	cast, err := StartCast(ctx, CastConfig{RecordingsDir: filepath.Join(t.TempDir(), "recordings"), Host: broadcast.NewHost(broadcast.HostConfig{Tunnel: &broadcast.FakeTunnel{}}), Logf: t.Logf,
		Launch: func(ctx context.Context, opts *RecordOptions) (Browser, error) {
			opts.ShotEvery = -1
			var err error
			chrome, err = Launch(ctx, LaunchOptions{Binary: bin, Headless: true, ProfileDir: t.TempDir(), StartURL: pageURL, Record: opts})
			return chrome, err
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	stop := context.AfterFunc(ctx, chrome.proc.Kill)
	// Keep the deadline watchdog armed until End has joined the browser,
	// recorder and cast workers. The go-test timeout must exceed this context.
	defer func() { cast.End(); stop() }()
	p := &pageLoadNative{t: t, ctx: ctx, chrome: chrome, cast: cast}
	if err = chrome.SetLive(true); err != nil {
		t.Fatal(err)
	}
	initial := p.load("initial accepted load", "")
	p.await("fixture document complete", func() bool {
		return string(p.tryEval(0, `document.readyState==='complete'&&!!document.querySelector('#anchor')`)) == "true"
	})
	initial = p.load("fixture accepted load", "")
	p.call(p.session, "Emulation.setDeviceMetricsOverride", map[string]any{"width": 1280, "height": 800, "deviceScaleFactor": 1, "mobile": false}, nil)
	p.ui(nil, nil)
	noteID := p.note("Ordinary note on original page load")
	p.ui([]string{noteID}, nil)
	sourceSession, sourceTarget := p.session, initial.Target
	sourceWorld := p.world
	p.mustEval(0, `globalThis.__pageLoadOriginalAnchor=document.querySelector('#anchor');window.scrollTo(0,600);true`)
	p.await("original anchor offscreen", func() bool {
		return string(p.tryEval(0, `document.querySelector('#anchor').getBoundingClientRect().bottom<0`)) == "true"
	})

	// No sketch exists yet: restoring this note cannot borrow the sketch
	// receiver's historical anchor retention and conceal note-only cache pruning.
	other := p.newTarget(pageURL + "other")
	p.activate(other)
	otherLoad := p.load("different target accepted", other, pageURL+"other")
	if recording.SamePageLoad(initial, otherLoad) {
		t.Fatal("different target reused original load")
	}
	p.ui(nil, nil)
	p.activate(sourceTarget)
	returned := p.load("note-only original target return", sourceTarget)
	if !recording.SamePageLoad(initial, returned) {
		t.Fatalf("same tab return changed native load: before=%+v after=%+v", initial, returned)
	}
	if p.session != sourceSession || p.world != sourceWorld {
		t.Fatalf("same-load return unexpectedly replaced original isolated world: session=%s/%s world=%d/%d", sourceSession, p.session, sourceWorld, p.world)
	}
	p.ui([]string{noteID}, nil)
	if string(p.mustEval(0, `__pageLoadOriginalAnchor===document.querySelector('#anchor')&&__pageLoadOriginalAnchor.isConnected`)) != "true" {
		t.Fatal("tab return replaced original page anchor")
	}
	key, _ := json.Marshal("note:" + noteID)
	p.mustEval(p.world, `(()=>{const row=[...__couchcastOperatorSketch.labelInput.getRootNode().querySelectorAll('.feedback-row')].find(r=>r.dataset.key===`+string(key)+`);row.querySelector('.feedback-reveal').click();return true})()`)
	p.await("note-only confirmed original reveal after tab return", func() bool {
		raw := p.tryEval(p.world, `(()=>{const r=document.querySelector('#anchor').getBoundingClientRect(),root=__couchcastOperatorSketch.labelInput.getRootNode();return r.top>=0&&r.bottom<=innerHeight&&!root.querySelector('.feedback-status').textContent.includes('anchor unavailable')})()`)
		return string(raw) == "true"
	})
	sketchID := p.sketch("Finished sketch on original page load")
	p.ui([]string{noteID}, []string{sketchID})

	if err = chrome.SetLive(false); err != nil {
		t.Fatal(err)
	}
	p.await("recorded pause", func() bool { return cast.board.OperatorView().Paused })
	if err = chrome.SetLive(true); err != nil {
		t.Fatal(err)
	}
	resumed := p.load("same-load resume", sourceTarget)
	if !recording.SamePageLoad(initial, resumed) {
		t.Fatalf("pause/resume changed load: %+v -> %+v", initial, resumed)
	}
	p.ui([]string{noteID}, []string{sketchID})

	p.mustEval(0, `document.querySelector('#app').textContent='Application-only refresh';true`)
	p.await("application mutation recorded", func() bool {
		recs, err := readSubmissionRecording(ctx, cast.RecordingPath())
		if err != nil {
			return false
		}
		text, err := query.New(recs, nil).DOMAt(math.MaxInt64, "")
		return err == nil && strings.Contains(text, "Application-only refresh")
	})
	if load := p.load("same document after app update", sourceTarget); !recording.SamePageLoad(initial, load) {
		t.Fatalf("app update changed load: %+v", load)
	}
	p.ui([]string{noteID}, []string{sketchID})

	sameURLTarget := p.newTarget(pageURL)
	p.activate(sameURLTarget)
	sameURLLoad := p.load("new target at exact same URL", sameURLTarget, pageURL)
	if sameURLLoad.Target == initial.Target || recording.SamePageLoad(initial, sameURLLoad) {
		t.Fatal("same URL became load authority")
	}
	p.ui(nil, nil)
	p.activate(sourceTarget)
	if load := p.load("original load restored after same-URL target", sourceTarget); !recording.SamePageLoad(initial, load) {
		t.Fatalf("same-URL target return changed original load: %+v", load)
	}
	p.ui([]string{noteID}, []string{sketchID})

	p.call(p.session, "Page.reload", map[string]any{"ignoreCache": true}, nil)
	// Read-only native/Board barriers; deliberately no application DOM edit.
	p.await("same-URL reload accepted without later DOM edit", func() bool {
		view := cast.board.OperatorView()
		return view.PageLoadReady && view.PageLoad != nil && view.PageLoad.Target == sourceTarget && view.PageLoad.Loader != initial.Loader
	})
	reloaded := p.load("reloaded native identity", sourceTarget, pageURL)
	if reloaded.Loader == initial.Loader || recording.SamePageLoad(initial, reloaded) {
		t.Fatalf("real reload retained prior load: %+v -> %+v", initial, reloaded)
	}
	p.ui(nil, nil)
	p.history(initial, noteID, sketchID)
}

type pageLoadNative struct {
	t       *testing.T
	ctx     context.Context
	chrome  *Chrome
	cast    *Cast
	session string
	world   int64
}

func (p *pageLoadNative) call(session, method string, params, result any) {
	p.t.Helper()
	if err := p.chrome.conn.Call(p.ctx, session, method, params, result); err != nil {
		p.t.Fatalf("%s: %v", method, err)
	}
}
func (p *pageLoadNative) tryEval(world int64, expr string) json.RawMessage {
	var result struct {
		Result struct {
			Value json.RawMessage `json:"value"`
		} `json:"result"`
		Exception json.RawMessage `json:"exceptionDetails"`
	}
	args := map[string]any{"expression": expr, "returnByValue": true, "awaitPromise": true}
	if world != 0 {
		args["contextId"] = world
	}
	if err := p.chrome.conn.Call(p.ctx, p.session, "Runtime.evaluate", args, &result); err != nil || len(result.Exception) > 0 {
		return nil
	}
	return result.Result.Value
}
func (p *pageLoadNative) mustEval(world int64, expr string) json.RawMessage {
	p.t.Helper()
	raw := p.tryEval(world, expr)
	if len(raw) == 0 {
		p.t.Fatalf("evaluation failed: %s", expr)
	}
	return raw
}
func (p *pageLoadNative) await(what string, check func() bool) {
	p.t.Helper()
	ctx, cancel := context.WithTimeout(p.ctx, 12*time.Second)
	defer cancel()
	// Poll only authoritative conditions. The clock never establishes ordering;
	// CDP responses, accepted Board scope and actual mounted DOM do.
	tick := time.NewTicker(20 * time.Millisecond)
	defer tick.Stop()
	for !check() {
		select {
		case <-ctx.Done():
			view, _ := json.Marshal(p.cast.board.OperatorView())
			p.t.Fatalf("waiting for %s: %v; Board=%s", what, ctx.Err(), view)
		case <-tick.C:
		}
	}
}
func (p *pageLoadNative) load(what, target string, urls ...string) *recording.PageLoad {
	p.t.Helper()
	var accepted *recording.PageLoad
	p.await(what, func() bool {
		view := p.cast.board.OperatorView()
		if !view.PageLoadReady || view.PageLoad == nil || (target != "" && view.PageLoad.Target != target) {
			return false
		}
		session, world := "", int64(0)
		done := make(chan struct{})
		if !p.chrome.do(func(s *loopState) {
			for ss, id := range s.sessions {
				if id == view.PageLoad.Target {
					session, world = ss, s.contexts[ss]
				}
			}
			close(done)
		}) {
			return false
		}
		select {
		case <-p.ctx.Done():
			return false
		case <-done:
		}
		if session == "" || world == 0 {
			return false
		}
		var tree struct {
			FrameTree struct {
				Frame struct {
					Loader string `json:"loaderId"`
				} `json:"frame"`
			} `json:"frameTree"`
		}
		if p.chrome.conn.Call(p.ctx, session, "Page.getFrameTree", nil, &tree) != nil {
			return false
		}
		root, ok := p.documentRoot(session)
		if !ok {
			return false
		}
		native := &recording.PageLoad{Target: view.PageLoad.Target, Loader: tree.FrameTree.Frame.Loader, Root: root}
		if !recording.SamePageLoad(native, view.PageLoad) {
			return false
		}
		p.session, p.world = session, world
		if len(urls) > 0 {
			if view.FeedbackContext.URL != urls[0] {
				return false
			}
			url, _ := json.Marshal(urls[0])
			if string(p.tryEval(0, `location.href===`+string(url)+`&&document.readyState==='complete'&&!!document.querySelector('#anchor')`)) != "true" {
				return false
			}
		}
		if string(p.tryEval(world, `!!globalThis.__couchcastOperatorSketch&&!!__couchcastOperatorSketch.labelInput.getRootNode().querySelector('.feedback-panel')`)) != "true" {
			return false
		}
		accepted = native
		return true
	})
	p.t.Logf("%s: target=%s loader=%s root=%d doc=%d", what, accepted.Target, accepted.Loader, accepted.Root, p.cast.board.OperatorView().Doc)
	return accepted
}

// Describe the actual document object without DOM.getDocument, which changes
// the recorder session's requested frontend subtree and mutation subscriptions.
func (p *pageLoadNative) documentRoot(session string) (int64, bool) {
	var object struct {
		Result struct {
			ID string `json:"objectId"`
		} `json:"result"`
	}
	if p.chrome.conn.Call(p.ctx, session, "Runtime.evaluate", map[string]any{"expression": "document", "returnByValue": false}, &object) != nil || object.Result.ID == "" {
		return 0, false
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(p.ctx), 3*time.Second)
		defer cancel()
		_ = p.chrome.conn.Call(ctx, session, "Runtime.releaseObject", map[string]any{"objectId": object.Result.ID}, nil)
	}()
	var described struct {
		Node struct {
			Backend int64 `json:"backendNodeId"`
		} `json:"node"`
	}
	if p.chrome.conn.Call(p.ctx, session, "DOM.describeNode", map[string]any{"objectId": object.Result.ID}, &described) != nil {
		return 0, false
	}
	return described.Node.Backend, described.Node.Backend > 0
}
func (p *pageLoadNative) ui(noteIDs, sketchIDs []string) {
	p.t.Helper()
	expected := []string{}
	for _, id := range noteIDs {
		expected = append(expected, "note:"+id)
	}
	for _, id := range sketchIDs {
		expected = append(expected, "sketch:"+id)
	}
	sort.Strings(expected)
	p.await("current Board selection and production Notes rows "+fmt.Sprint(expected), func() bool {
		selection, err := p.cast.board.Selection()
		if err != nil {
			return false
		}
		boardKeys := []string{}
		for _, n := range selection.Notes {
			boardKeys = append(boardKeys, "note:"+n.ID)
		}
		for _, s := range selection.Sketches {
			boardKeys = append(boardKeys, "sketch:"+s.ID)
		}
		sort.Strings(boardKeys)
		if !reflect.DeepEqual(boardKeys, expected) {
			return false
		}
		var keys []string
		raw := p.tryEval(p.world, `(()=>{const root=__couchcastOperatorSketch.labelInput.getRootNode();if(root.host.hidden||root.querySelectorAll('.feedback-panel').length!==1)return null;return [...root.querySelectorAll('.feedback-row')].map(r=>r.dataset.key).sort()})()`)
		return json.Unmarshal(raw, &keys) == nil && reflect.DeepEqual(keys, expected)
	})
}
func (p *pageLoadNative) mouse(kind string, x, y float64, buttons int, pen bool) {
	p.t.Helper()
	button := "left"
	if kind == "mouseMoved" {
		button = "none"
	}
	args := map[string]any{"type": kind, "x": x, "y": y, "button": button, "buttons": buttons, "clickCount": 1}
	if pen {
		args["pointerType"] = "pen"
		args["force"] = .5
	}
	p.call(p.session, "Input.dispatchMouseEvent", args, nil)
}
func (p *pageLoadNative) note(text string) string {
	p.t.Helper()
	if err := p.cast.board.SetAnnotationAllowed(p.ctx, true); err != nil {
		p.t.Fatal(err)
	}
	p.await("ordinary note permission", func() bool {
		return string(p.tryEval(p.world, `[...__couchcastOperatorSketch.labelInput.getRootNode().querySelectorAll('.mode')].some(b=>b.textContent.includes('Note')&&!b.disabled)`)) == "true"
	})
	if err := p.chrome.SetAnnotationMode("note"); err != nil {
		p.t.Fatal(err)
	}
	var point [2]float64
	json.Unmarshal(p.mustEval(0, `(()=>{const r=document.querySelector('#anchor').getBoundingClientRect();return [r.left+40,r.top+40]})()`), &point)
	p.mouse("mousePressed", point[0], point[1], 1, false)
	p.mouse("mouseReleased", point[0], point[1], 0, false)
	quoted, _ := json.Marshal(text)
	p.await("ordinary note editor ready and saved", func() bool {
		return string(p.tryEval(p.world, `(()=>{const root=__couchcastOperatorSketch.labelInput.getRootNode(),dialog=root.querySelector('.note-dialog'),input=root.querySelector('.note-input');if(!dialog||dialog.hidden||dialog.classList.contains('hidden'))return false;if(input.value!==`+string(quoted)+`){input.value=`+string(quoted)+`;input.dispatchEvent(new Event('input',{bubbles:true}));}const save=[...root.querySelectorAll('.note-buttons button')].find(b=>b.textContent==='Save');if(!save||save.disabled)return false;save.click();return true})()`)) == "true"
	})
	id := ""
	p.await("saved ordinary note on Board", func() bool {
		selected, err := p.cast.board.Selection()
		if err != nil || len(selected.Notes) != 1 {
			return false
		}
		id = selected.Notes[0].ID
		note, ok := p.cast.board.ObservedNote(id)
		return ok && note.Capture != ""
	})
	return id
}
func (p *pageLoadNative) sketch(label string) string {
	p.t.Helper()
	p.mustEval(p.world, `__couchcastOperatorSketch.controls.pen.click();true`)
	p.await("prepared sketch scene", func() bool {
		return string(p.tryEval(p.world, `__couchcastOperatorSketch.state().phase`)) == `"drawing"`
	})
	p.mouse("mousePressed", 140, 160, 1, true)
	p.mouse("mouseMoved", 180, 180, 1, true)
	p.mouse("mouseReleased", 180, 180, 0, true)
	p.await("accepted sketch contact", func() bool {
		return string(p.tryEval(p.world, `__couchcastOperatorSketch.state().slots.some(s=>s.nextSequence>=4&&!s.inflight&&s.queued===0&&!s.finishing)`)) == "true"
	})
	p.mustEval(p.world, `__couchcastOperatorSketch.controls.done.click();true`)
	p.await("required sketch label", func() bool {
		return string(p.tryEval(p.world, `!__couchcastOperatorSketch.labelInput.parentElement.hidden`)) == "true"
	})
	quoted, _ := json.Marshal(label)
	p.mustEval(p.world, `(async()=>{const tool=__couchcastOperatorSketch;tool.labelInput.value=`+string(quoted)+`;await tool.saveLabel();return true})()`)
	id := ""
	p.await("finished sketch on Board", func() bool {
		selected, err := p.cast.board.Selection()
		if err != nil || len(selected.Sketches) != 1 {
			return false
		}
		id = selected.Sketches[0].ID
		return true
	})
	return id
}
func (p *pageLoadNative) newTarget(url string) string {
	p.t.Helper()
	var result struct {
		ID string `json:"targetId"`
	}
	p.call("", "Target.createTarget", map[string]any{"url": url}, &result)
	return result.ID
}
func (p *pageLoadNative) activate(target string) {
	p.t.Helper()
	p.call("", "Target.activateTarget", map[string]any{"targetId": target}, nil)
}
func (p *pageLoadNative) history(load *recording.PageLoad, noteID, sketchID string) {
	p.t.Helper()
	p.await("independent retained note and sketch history on disk", func() bool {
		recs, err := readSubmissionRecording(p.ctx, p.cast.RecordingPath())
		if err != nil {
			return false
		}
		q := query.New(recs, notes.Structural{})
		notes, err := q.Notes(5000)
		if err != nil {
			return false
		}
		sketches, err := q.SketchesAt(math.MaxInt64)
		if err != nil || len(notes) != 1 || len(sketches) != 1 {
			return false
		}
		want := query.PageLoadFacts{Target: load.Target, Loader: load.Loader, Root: strconv.FormatInt(load.Root, 10)}
		return notes[0].Note.ID == noteID && notes[0].PageLoad == want && sketches[0].History.Begin.ID == sketchID && sketches[0].Source.PageLoad == want && sketches[0].History.Label != ""
	})
}
