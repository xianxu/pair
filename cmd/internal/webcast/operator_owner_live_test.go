package webcast

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/webcast/capability"
)

// One disposable native cast proves the mounted owner controls work while a
// session-authenticated annotating viewer still has no grant, including at gen0.
func TestOperatorOwnerLiveAnnotationOff(t *testing.T) {
	if os.Getenv("WEBCAST_LIVE_CHROME") == "" {
		t.Skip("set WEBCAST_LIVE_CHROME=1")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	page := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<!doctype html><title>owner annotation OFF</title><style>body{margin:0}button{position:absolute;left:80px;top:100px;width:240px;height:160px;border:0;background:#eee}</style><button>Save</button>`)
	}))
	defer page.Close()
	binary, err := FindChrome(os.Getenv, execLookPath, fileExists)
	if err != nil {
		t.Fatal(err)
	}
	host := broadcast.NewHost(broadcast.HostConfig{Tunnel: &broadcast.FakeTunnel{}})
	var chrome *Chrome
	cast, err := StartCast(ctx, CastConfig{RecordingsDir: filepath.Join(t.TempDir(), "recordings"), Host: host, Logf: t.Logf,
		Launch: func(ctx context.Context, opts *RecordOptions) (Browser, error) {
			opts.ShotEvery = -1
			var e error
			chrome, e = Launch(ctx, LaunchOptions{Binary: binary, Headless: true, ProfileDir: t.TempDir(), StartURL: page.URL, Record: opts})
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
	diagnostic := func() string { return "operator UI not mounted" }
	wait := func(what string, condition func() bool) {
		t.Helper()
		for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); {
			if condition() {
				return
			}
			select {
			case <-ctx.Done():
				t.Fatalf("waiting for %s: %v", what, ctx.Err())
			case <-time.After(25 * time.Millisecond):
			}
		}
		t.Fatalf("waiting for %s; %s", what, diagnostic())
	}
	resize := func(height int) {
		t.Helper()
		call("Emulation.setDeviceMetricsOverride", map[string]any{"width": 1728, "height": height, "deviceScaleFactor": 1, "mobile": false}, nil)
	}
	resize(997)
	wait("fixture document", func() bool {
		return string(eval(0, `document.readyState==='complete'&&document.querySelector('button')?.textContent==='Save'`)) == "true"
	})
	if err = chrome.SetLive(true); err != nil {
		t.Fatal(err)
	}
	var world int64
	wait("operator world", func() bool {
		done := make(chan struct{})
		chrome.do(func(s *loopState) { world = s.contexts[session]; close(done) })
		<-done
		return world != 0 && cast.board.OperatorView().Doc > 0
	})
	wait("mounted operator tools", func() bool {
		return string(eval(world, `!!globalThis.__couchcastOperatorSketch?.labelInput&&!__couchcastOperatorSketch.labelInput.getRootNode().host.hidden`)) == "true"
	})
	// This test-only reference inspects real mounted controls; it does not mint
	// permission or send a synthetic owner request.
	eval(world, `globalThis.ownerTestRoot=__couchcastOperatorSketch.labelInput.getRootNode();true`)
	diagnostic = func() string {
		return string(eval(world, `JSON.stringify((()=>{const root=ownerTestRoot,bar=root.querySelector('.cast-tools');return {viewport:[innerWidth,innerHeight],delivery:globalThis.__couchcastAnnotationDelivery,sketch:__couchcastOperatorSketch.state(),scene:!!globalThis.__couchcastPrivateSceneFrame,hostHidden:root.host.hidden,status:root.querySelector('.feedback-status')?.textContent,labelVisible:!__couchcastOperatorSketch.labelInput.parentElement.hidden,toolbar:{rect:bar.getBoundingClientRect().toJSON(),scroll:[bar.scrollTop,bar.scrollHeight,bar.clientHeight]},buttons:[...bar.querySelectorAll('.mode')].map(b=>{const r=b.getBoundingClientRect();return {text:b.textContent,disabled:b.disabled,pressed:b.getAttribute('aria-pressed'),rect:r.toJSON(),hit:b.contains(root.elementFromPoint(r.x+r.width/2,r.y+r.height/2))}})}})())`))
	}
	assertOff := func() {
		t.Helper()
		if state := cast.board.AnnotationCapability(); state.Phase != capability.Off {
			t.Fatalf("owner action changed global viewer capability: %+v", state)
		}
		if _, err := cast.board.AdmitAnnotation(); err == nil {
			t.Fatal("ordinary viewer admitted while Annotation OFF")
		}
		if state := cast.server.viewerCapabilities(linkPointer).Annotation; state.Allowed || state.State != "off" {
			t.Fatalf("viewer projection claims annotation: %+v", state)
		}
	}
	assertOff()
	if state := cast.board.AnnotationCapability(); state.Generation != 0 {
		t.Fatalf("test must start at generation0: %+v", state)
	}
	wait("generation0 owner Note and Pointer enabled", func() bool {
		return string(eval(world, `globalThis.__couchcastAnnotationDelivery?.generation==='0'&&['.cast-tools-note .mode','.cast-tools-pointer .mode'].every(s=>{const b=ownerTestRoot.querySelector(s);return !!b&&!b.disabled})`)) == "true"
	})
	if err = chrome.SetAnnotationMode("note"); err != nil {
		t.Fatalf("generation0 owner mode command: %v", err)
	}
	if err = chrome.SetAnnotationMode("watch"); err != nil {
		t.Fatal(err)
	}

	var cookie *http.Cookie
	wait("authenticated viewer link", func() bool {
		r := httptest.NewRequest(http.MethodPost, "/"+cast.pointerToken+"/unlock", strings.NewReader(fmt.Sprintf(`{"code":%q}`, cast.locks.LockFor(cast.pointerToken).Code())))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		host.ServeHTTP(w, r)
		if w.Code != http.StatusOK || len(w.Result().Cookies()) == 0 {
			return false
		}
		cookie = w.Result().Cookies()[0]
		return true
	})
	refuseViewer := func() {
		t.Helper()
		for _, forged := range []bool{false, true} {
			body := `{"request":"viewer-prepare","seg":1,"node":1,"box":[80,100,240,160],"region":[100,120,120,80]}`
			if forged {
				body = `{"request":"claimed-owner","operator":true,"owner":"operator","viewer":{"id":"operator"},"seg":1,"node":1,"box":[80,100,240,160],"region":[100,120,120,80]}`
			}
			r := httptest.NewRequest(http.MethodPost, "/"+cast.pointerToken+"/prepare-note", strings.NewReader(body))
			r.AddCookie(cookie)
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("X-Couch-Annotation-Generation", strconv.FormatUint(cast.board.AnnotationCapability().Generation, 10))
			if forged {
				r.Header.Set(broadcast.NoteOwnerHeader, broadcast.OperatorNoteOwner)
			}
			w := httptest.NewRecorder()
			host.ServeHTTP(w, r)
			if w.Code != http.StatusForbidden || strings.TrimSpace(w.Body.String()) != "Annotation is off." {
				t.Fatalf("authenticated viewer forged=%v: %d %s", forged, w.Code, w.Body.String())
			}
		}
	}
	refuseViewer()
	// Native hit-testing and dispatch, rather than HTMLElement.click, exercises
	// the mounted listener and detects disabled, covered or unreachable buttons.
	click := func(selector string) {
		t.Helper()
		encoded, _ := json.Marshal(selector)
		var xy []float64
		wait("native hit target "+selector, func() bool {
			raw := eval(world, `(()=>{const b=ownerTestRoot.querySelector(`+string(encoded)+`);if(!b||b.disabled||ownerTestRoot.host.hidden)return null;const r=b.getBoundingClientRect();return r.width>0&&r.height>0&&r.left>=0&&r.top>=0&&r.right<=innerWidth&&r.bottom<=innerHeight&&b.contains(ownerTestRoot.elementFromPoint(r.x+r.width/2,r.y+r.height/2))?[r.x+r.width/2,r.y+r.height/2]:null})()`)
			return json.Unmarshal(raw, &xy) == nil && len(xy) == 2
		})
		for _, kind := range []string{"mousePressed", "mouseReleased"} {
			buttons := 0
			if kind == "mousePressed" {
				buttons = 1
			}
			call("Input.dispatchMouseEvent", map[string]any{"type": kind, "button": "left", "buttons": buttons, "clickCount": 1, "x": xy[0], "y": xy[1]}, nil)
		}
	}
	drag := func() {
		t.Helper()
		for i, kind := range []string{"mousePressed", "mouseMoved", "mouseMoved", "mouseReleased"} {
			buttons := 1
			if kind == "mouseReleased" {
				buttons = 0
			}
			call("Input.dispatchMouseEvent", map[string]any{"type": kind, "button": "left", "buttons": buttons, "clickCount": 1, "x": 100 + i*40, "y": 120 + i*20}, nil)
		}
	}
	click(".cast-tools-note .mode")
	wait("owner Note selected", func() bool {
		return string(eval(world, `ownerTestRoot.querySelector('.cast-tools-note .mode').getAttribute('aria-pressed')==='true'`)) == "true"
	})
	drag()
	wait("owner boxed capture ready", func() bool {
		return string(eval(world, `!ownerTestRoot.querySelector('.note-dialog').classList.contains('hidden')&&ownerTestRoot.querySelector('.note-msg').textContent.includes('Captured')`)) == "true"
	})
	eval(world, `(()=>{const input=ownerTestRoot.querySelector('.note-input');input.value='Owner note while viewers OFF';input.dispatchEvent(new Event('input',{bubbles:true}));return true})()`)
	click(".note-buttons button:last-child")
	wait("one durable captured owner note", func() bool { return len(cast.board.OperatorView().Notes) == 1 })
	note := cast.board.OperatorView().Notes[0]
	if note.Author.ID != "operator" || note.Text != "Owner note while viewers OFF" || note.Capture == "" || note.Region == nil || (*note.Region)[2] <= 0 || (*note.Region)[3] <= 0 {
		t.Fatalf("captured owner note: %+v", note)
	}
	wait("save acknowledged", func() bool {
		return string(eval(world, `ownerTestRoot.querySelector('.note-dialog').classList.contains('hidden')`)) == "true"
	})
	assertOff()
	refuseViewer()
	click(".cast-tools-pointer .mode")
	drag()
	wait("owner Pointer recorded", func() bool {
		for _, stroke := range cast.board.OperatorView().Strokes {
			if stroke.Author.ID == "operator" && len(stroke.Pts) > 0 {
				return true
			}
		}
		return false
	})
	assertOff()
	if err = chrome.SetAnnotationMode("watch"); err != nil {
		t.Fatal(err)
	}
	// Expand Pen in this same browser, then qualify Note reachability in the
	// short viewport through the toolbar's existing native scrolling surface.
	click(".cast-tools-pen button")
	wait("Pen expanded", func() bool {
		return string(eval(world, `__couchcastOperatorSketch.state().phase==='drawing'`)) == "true"
	})
	resize(400)
	wait("short viewport toolbar layout", func() bool {
		return string(eval(world, `innerHeight===400&&ownerTestRoot.querySelector('.cast-tools').getBoundingClientRect().bottom<=innerHeight`)) == "true"
	})
	eval(world, `ownerTestRoot.querySelector('.cast-tools-note .mode').scrollIntoView({block:'nearest'});true`)
	click(".cast-tools-note .mode")
	// Selecting another tool preserves the existing required sketch label.
	// Complete that mounted flow before expecting the requested Note mode.
	wait("expanded Pen requests its label before Note", func() bool {
		return string(eval(world, `__couchcastOperatorSketch.state().phase==='label-required'&&!__couchcastOperatorSketch.labelInput.parentElement.hidden`)) == "true"
	})
	eval(world, `__couchcastOperatorSketch.labelInput.value='Expanded Pen reachability';true`)
	click(".sketch-label button")
	wait("Note selected after expanded Pen", func() bool {
		return string(eval(world, `ownerTestRoot.querySelector('.cast-tools-note .mode').getAttribute('aria-pressed')==='true'`)) == "true"
	})
	resize(997)
	if err = cast.board.SetAnnotationAllowed(ctx, true); err != nil {
		t.Fatal(err)
	}
	wait("ON generation delivered", func() bool {
		return string(eval(world, `globalThis.__couchcastAnnotationDelivery.generation`)) == strconv.Quote(strconv.FormatUint(cast.board.AnnotationCapability().Generation, 10))
	})
	if err = cast.board.SetAnnotationAllowed(ctx, false); err != nil {
		t.Fatal(err)
	}
	assertOff()
	wait("settled OFF owner availability", func() bool {
		return string(eval(world, `globalThis.__couchcastAnnotationDelivery.generation===`+strconv.Quote(strconv.FormatUint(cast.board.AnnotationCapability().Generation, 10))+`&&globalThis.__couchcastAnnotationDelivery.allowed===true&&!ownerTestRoot.querySelector('.cast-tools-note .mode').disabled`)) == "true"
	})
	if err = chrome.SetAnnotationMode("note"); err != nil {
		t.Fatalf("owner mode unavailable after OFF cleanup: %v", err)
	}
	refuseViewer()
}
