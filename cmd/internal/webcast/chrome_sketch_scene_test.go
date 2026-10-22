package webcast

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

type sketchSceneFake struct {
	oopif                 bool
	sandbox               string
	total                 int
	calls                 []string
	excluded              bool
	privateBeforeExcluded bool
	onCall                func()
}

func (f *sketchSceneFake) Call(_ context.Context, session, method string, params, result any) error {
	f.calls = append(f.calls, session+":"+method)
	if f.onCall != nil {
		f.onCall()
	}
	p, _ := params.(map[string]any)
	if p["contextId"] == int64(88) && !f.excluded {
		f.privateBeforeExcluded = true
	}
	var response any = map[string]any{}
	switch method {
	case "Runtime.evaluate":
		if p["expression"] == "scene.frame" {
			response = map[string]any{"result": map[string]any{"objectId": "frame-object"}}
		} else {
			response = map[string]any{"result": map[string]any{"value": true}}
			expression, _ := p["expression"].(string)
			if index := strings.Index(expression, "Transfer(..."); index >= 0 {
				var args []json.RawMessage
				raw := expression[index+len("Transfer(...") : len(expression)-1]
				if json.Unmarshal([]byte(raw), &args) == nil && len(args) >= 2 {
					if strings.Contains(expression, "beginTransfer") {
						json.Unmarshal(args[1], &f.total)
					} else {
						var offset int
						var encoded string
						json.Unmarshal(args[1], &offset)
						json.Unmarshal(args[2], &encoded)
						part, _ := base64.StdEncoding.DecodeString(encoded)
						response = map[string]any{"result": map[string]any{"value": map[string]any{"offset": offset + len(part), "total": f.total}}}
					}
				}
			}

		}
	case "Target.getTargets":
		response = map[string]any{"targetInfos": []any{map[string]any{"targetId": "child", "type": "iframe"}}}
	case "Target.attachToTarget":
		response = map[string]any{"sessionId": "oopif-session"}
	case "DOM.describeNode":
		response = map[string]any{"node": map[string]any{"frameId": "child", "localName": "iframe", "attributes": []string{"sandbox", "allow-same-origin"}}}
		if f.sandbox != "" {
			response = map[string]any{"node": map[string]any{"frameId": "child", "localName": "iframe", "attributes": []string{"sandbox", f.sandbox}}}
		}
	case "Page.createIsolatedWorld":
		if f.oopif && session != "oopif-session" {
			return errors.New("frame belongs to another target")
		}
		response = map[string]any{"executionContextId": 88}
	}
	if result != nil {
		raw, _ := json.Marshal(response)
		return json.Unmarshal(raw, result)
	}
	return nil
}
func TestSketchSceneSessionScopesAndExcludesBeforeUpload(t *testing.T) {
	fake := &sketchSceneFake{}
	live := true
	scope := SketchSceneScope{ParentSession: "parent", ParentContext: 7, Loader: "loader"}
	scene := NewSketchSceneSession(fake, scope, func(got SketchSceneScope) bool { return live && got == scope }, func(frame, session string) error {
		if frame != "child" || session != "parent" {
			t.Fatal("wrong exclusion scope")
		}
		fake.excluded = true
		return nil
	})
	if err := scene.Open(context.Background(), "scene.frame"); err != nil {
		t.Fatal(err)
	}
	if fake.privateBeforeExcluded {
		t.Fatal("private code uploaded before observer exclusion")
	}
	live = false
	n := len(fake.calls)
	if _, err := scene.HitTest(context.Background(), [2]float64{1, 2}); !errors.Is(err, ErrSketchSceneRetired) {
		t.Fatalf("retired hit test %v", err)
	}
	if len(fake.calls) != n {
		t.Fatal("retired scope still called Chrome")
	}
}
func TestSketchSceneSessionRejectsLateContextResponse(t *testing.T) {
	fake := &sketchSceneFake{}
	live := true
	scene := NewSketchSceneSession(fake, SketchSceneScope{ParentSession: "p", ParentContext: 7, Loader: "l"}, func(SketchSceneScope) bool { return live }, func(string, string) error { return nil })
	fake.onCall = func() { live = false }
	if err := scene.Open(context.Background(), "scene.frame"); !errors.Is(err, ErrSketchSceneRetired) {
		t.Fatalf("late response %v", err)
	}
	if len(fake.calls) != 1 {
		t.Fatal("retired Open recreated a context")
	}
}
func TestSketchSceneShellIsOpaqueWithoutScriptsOrPrivateData(t *testing.T) {
	shell := SketchSceneShellURL()
	if !strings.HasPrefix(shell, "data:text/html;couch-sketch=1,") || strings.Contains(strings.ToLower(shell), "script") || strings.Contains(shell, "blob:") {
		t.Fatal("shell carries active/private content")
	}
}

func TestOperatorSketchLiveScenePrivacy(t *testing.T) {
	if os.Getenv("WEBCAST_LIVE_CHROME") == "" {
		t.Skip("set WEBCAST_LIVE_CHROME=1")
	}
	binary, err := FindChrome(os.Getenv, execLookPath, fileExists)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<!doctype html><title>Attacker fixture</title><body><main>Public casting page</main></body>`))
	}))
	defer server.Close()
	c, err := Launch(context.Background(), LaunchOptions{Binary: binary, Headless: true, ProfileDir: t.TempDir(), StartURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	var session string
	for deadline := time.Now().Add(8 * time.Second); time.Now().Before(deadline); {
		done := make(chan struct{})
		c.do(func(s *loopState) { session = sessionOf(s, castTab(s)); close(done) })
		<-done
		if session != "" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if session == "" {
		t.Fatal("no cast session")
	}
	var tree struct {
		Tree struct {
			Frame struct {
				ID     string `json:"id"`
				URL    string `json:"url"`
				Loader string `json:"loaderId"`
			} `json:"frame"`
		} `json:"frameTree"`
	}
	// Launch acknowledges attachment before its initial navigation necessarily
	// commits. Capture the scope only after this fixture's document is ready.
	settled := false
	for deadline := time.Now().Add(8 * time.Second); time.Now().Before(deadline); {
		if err = c.conn.Call(ctx, session, "Page.getFrameTree", map[string]any{}, &tree); err != nil {
			t.Fatal(err)
		}
		var ready struct {
			Result struct {
				Value string `json:"value"`
			} `json:"result"`
		}
		if tree.Tree.Frame.URL == server.URL+"/" && tree.Tree.Frame.Loader != "" {
			if err = c.conn.Call(ctx, session, "Runtime.evaluate", map[string]any{"expression": "document.readyState", "returnByValue": true}, &ready); err == nil && ready.Result.Value == "complete" {
				settled = true
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !settled {
		t.Fatal("initial fixture document did not settle")
	}
	var world struct {
		ID int64 `json:"executionContextId"`
	}
	if err = c.conn.Call(ctx, session, "Page.createIsolatedWorld", map[string]any{"frameId": tree.Tree.Frame.ID, "worldName": "sketch-private-test-parent"}, &world); err != nil {
		t.Fatal(err)
	}
	scope := SketchSceneScope{ParentSession: session, ParentContext: world.ID, Loader: tree.Tree.Frame.Loader}
	initialTree, _ := json.Marshal(tree)
	defer func() {
		if t.Failed() {
			var current json.RawMessage
			_ = c.conn.Call(ctx, session, "Page.getFrameTree", map[string]any{}, &current)
			t.Logf("initial frame=%s current=%s", initialTree, current)
		}
	}()

	live := true
	excluded := false
	scene := NewSketchSceneSession(c.conn, scope, func(got SketchSceneScope) bool { return live && got == scope }, func(frame, session string) error { excluded = true; return nil })
	shell, _ := json.Marshal(SketchSceneShellURL())
	setup := `(async()=>{const host=document.createElement('div');document.body.appendChild(host);const root=host.attachShadow({mode:'closed'});const frame=document.createElement('iframe');frame.name='opaque-private-scene';frame.sandbox='allow-same-origin';frame.src=` + string(shell) + `;const loaded=new Promise((ok,bad)=>{frame.onload=ok;setTimeout(()=>bad(Error('CSP or shell readiness failure')),3000)});root.appendChild(frame);await loaded;globalThis.__couchcastPrivateSceneFrame=frame;return true;})()`
	if err = scene.evaluate(ctx, session, world.ID, setup, nil); err != nil {
		t.Fatal(err)
	}
	if err = scene.Open(ctx, "globalThis.__couchcastPrivateSceneFrame"); err != nil {
		initial := tree
		_ = c.conn.Call(ctx, session, "Page.getFrameTree", map[string]any{}, &tree)
		t.Logf("scope launch URL=%s loader=%s; current URL=%s loader=%s", initial.Tree.Frame.URL, initial.Tree.Frame.Loader, tree.Tree.Frame.URL, tree.Tree.Frame.Loader)
		t.Fatal(err)
	}
	if !excluded {
		t.Fatal("private frame was not excluded before upload")
	}
	defer scene.Dispose(context.Background())
	builder := recording.NewBuilder("scene-privacy")
	img := builder.El("img", []string{"src", "https://historical.test/private.svg"})
	private := builder.El("p", nil, builder.Txt("ACTUAL_RENDERER_PRIVATE_DOM"))
	nested := builder.El("iframe", []string{"name", "kept-recorded-frame"})
	style := builder.El("style", nil, builder.Txt(`iframe[name="kept-recorded-frame"]{width:123px;height:45px}`))
	builder.Page(recording.SegLive, "https://historical.test/", style, img, private, nested)
	frameDoc := builder.ID()
	frameTree := recording.Node{ID: frameDoc, Y: recording.YDoc, C: []recording.Node{builder.El("html", nil, builder.El("body", nil, builder.Txt("ACTUAL_NESTED_PRIVATE_DOM")))}}
	builder.Add(&recording.Doc{Doc: frameDoc, Host: nested.ID, URL: "https://historical.test/nested", Tree: frameTree})
	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="2" height="3"><rect width="2" height="3" fill="red"/></svg>`)
	sum := sha256.Sum256(svg)
	hash := hex.EncodeToString(sum[:])
	builder.Add(&recording.Asset{URL: "https://historical.test/private.svg", Hash: hash, MIME: "image/svg+xml", Size: int64(len(svg))})
	builder.At(1).Add(&recording.SketchBegin{SketchOperation: recording.SketchOperation{Operation: "begin", Digest: strings.Repeat("a", 64)}, ID: "private-sketch", Author: recording.Author{ID: recording.OperatorID, Color: "#123456"}, Owner: recording.OperatorID, Doc: 1, Tab: "tab1", VP: recording.Viewport{W: 1280, H: 800, DPR: 1}})
	var prefix bytes.Buffer
	for _, r := range builder.Recs {
		encoded, e := recording.Encode(r)
		if e != nil {
			t.Fatal(e)
		}
		prefix.Write(encoded)
		prefix.WriteByte('\n')
	}
	source := prefix.Bytes()
	descriptor := board.SketchSourceDescriptor{Handle: "private-source", Time: 1, Length: len(source), Doc: 1, Tab: "tab1", VP: recording.Viewport{W: 1280, H: 800, DPR: 1}}
	registeredURL := ""
	scene.SetAssetURLObserver(func(url string) error { registeredURL = url; return nil })
	assets, err := scene.Prepare(ctx, "private-sketch", descriptor, func(offset, length int) ([]byte, error) { return bytes.Clone(source[offset : offset+length]), nil })
	if err != nil {
		t.Fatal(err)
	}
	if len(assets) != 1 {
		t.Fatalf("source assets %+v", assets)
	}
	if err = scene.InstallAsset(ctx, "private-asset", assets[0], func(offset, length int) ([]byte, error) { return bytes.Clone(svg[offset : offset+length]), nil }); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(registeredURL, "blob:null/") {
		t.Fatal("private URL not registered before Render")
	}
	ready, err := scene.Render(ctx)
	if err != nil || !ready.Ready {
		t.Fatalf("actual renderer readiness %+v %v", ready, err)
	}
	var inspected []struct {
		Text, Name, Src, Nested        string
		Width, ImageWidth, ImageHeight float64
	}
	if err = scene.invoke(ctx, "inspect", []any{[]recording.NodeID{private.ID, img.ID, nested.ID}}, &inspected); err != nil {
		t.Fatal(err)
	}
	if len(inspected) != 3 || inspected[0].Text != "ACTUAL_RENDERER_PRIVATE_DOM" || inspected[1].ImageWidth != 2 || inspected[1].ImageHeight != 3 || inspected[2].Name != "kept-recorded-frame" || inspected[2].Nested != "ACTUAL_NESTED_PRIVATE_DOM" || inspected[2].Width != 127 {
		t.Fatalf("actual unchanged Renderer fidelity %+v", inspected)
	}
	blob, _ := json.Marshal(inspected[1].Src)
	attack := `(async()=>{const result={};for(const name of ['opaque-private-scene','kept-recorded-frame']){try{const target=window.open('',name);result[name]=target?target.document.body.textContent:'not-found';}catch(error){result[name]=error.name;}}try{result.blob=await(await fetch(` + string(blob) + `)).text();}catch(error){result.blob=error.name;}result.api=typeof globalThis.__couchcastSketchScene;return result;})()`
	var response struct {
		Result struct {
			Value map[string]string `json:"value"`
		} `json:"result"`
		Exception json.RawMessage `json:"exceptionDetails"`
	}
	if err = c.conn.Call(ctx, session, "Runtime.evaluate", map[string]any{"expression": attack, "returnByValue": true, "awaitPromise": true, "userGesture": true}, &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Exception) > 0 || response.Result.Value["opaque-private-scene"] != "SecurityError" || response.Result.Value["kept-recorded-frame"] != "SecurityError" || response.Result.Value["blob"] != "TypeError" || response.Result.Value["api"] != "undefined" {
		t.Fatalf("casting page reached private scene: %+v %s", response.Result.Value, response.Exception)
	}
	live = false
	if _, err = scene.HitTest(ctx, [2]float64{1, 2}); !errors.Is(err, ErrSketchSceneRetired) {
		t.Fatal("retired parent response admitted")
	}
}

func TestSketchSceneTransferStepVisitsOneChunk(t *testing.T) {
	fake := &sketchSceneFake{excluded: true}
	scene := NewSketchSceneSession(fake, SketchSceneScope{ParentSession: "p", ParentContext: 7, Loader: "l"}, func(SketchSceneScope) bool { return true }, func(string, string) error { return nil })
	scene.session = "p"
	scene.world = 88
	reads := 0
	transfer, err := scene.BeginTransfer(context.Background(), "h", 65536, func(offset, length int) ([]byte, error) { reads++; return make([]byte, length), nil })
	if err != nil {
		t.Fatal(err)
	}
	done, err := transfer.Step(context.Background())
	if err != nil || done || reads != 1 {
		t.Fatalf("first step done=%v reads=%d err=%v", done, reads, err)
	}
	done, err = transfer.Step(context.Background())
	if err != nil || !done || reads != 2 {
		t.Fatalf("second step done=%v reads=%d err=%v", done, reads, err)
	}
	n := len(fake.calls)
	if done, err = transfer.Step(context.Background()); err != nil || !done || len(fake.calls) != n {
		t.Fatal("complete transfer performed another step")
	}
}

func TestSketchSceneSessionOOPIFAndUnsafeSandbox(t *testing.T) {
	fake := &sketchSceneFake{oopif: true}
	var exclusions []string
	scene := NewSketchSceneSession(fake, SketchSceneScope{ParentSession: "parent", ParentContext: 7, Loader: "l"}, func(SketchSceneScope) bool { return true }, func(frame, session string) error {
		exclusions = append(exclusions, session)
		fake.excluded = true
		return nil
	})
	if err := scene.Open(context.Background(), "scene.frame"); err != nil {
		t.Fatal(err)
	}
	if len(exclusions) != 2 || exclusions[0] != "parent" || exclusions[1] != "oopif-session" || scene.session != "oopif-session" {
		t.Fatalf("wrong OOPIF source session %v", exclusions)
	}
	if err := scene.Dispose(context.Background()); err != nil {
		t.Fatal(err)
	}
	if fake.calls[len(fake.calls)-1] != ":Target.detachFromTarget" {
		t.Fatal("owned OOPIF attachment not released")
	}
	bad := &sketchSceneFake{sandbox: "allow-same-origin allow-scripts"}
	scene = NewSketchSceneSession(bad, SketchSceneScope{ParentSession: "parent", ParentContext: 7, Loader: "l"}, func(SketchSceneScope) bool { return true }, func(string, string) error { t.Fatal("unsafe sandbox reached private admission"); return nil })
	if err := scene.Open(context.Background(), "scene.frame"); err == nil {
		t.Fatal("script-enabled frame admitted")
	}
}

func TestSketchSceneAssetTransferUsesExistingCaptureEnvelope(t *testing.T) {
	fake := &sketchSceneFake{excluded: true}
	scene := NewSketchSceneSession(fake, SketchSceneScope{ParentSession: "p", ParentContext: 7, Loader: "l"}, func(SketchSceneScope) bool { return true }, func(string, string) error { return nil })
	scene.session = "p"
	scene.world = 88
	read := func(offset, length int) ([]byte, error) { return make([]byte, length), nil }
	if _, err := scene.BeginAssetTransfer(context.Background(), "asset", (16<<20)+1, read); err != nil {
		t.Fatalf("legal captured asset narrowed to source cap: %v", err)
	}
	if _, err := scene.BeginTransfer(context.Background(), "source", (16<<20)+1, read); err == nil {
		t.Fatal("source prefix exceeded16MiB")
	}
	if _, err := scene.BeginAssetTransfer(context.Background(), "asset", (32<<20)+1, read); err == nil {
		t.Fatal("asset exceeded capture envelope")
	}
}

type sketchSceneLostContext struct{ calls int }

func (f *sketchSceneLostContext) Call(context.Context, string, string, any, any) error {
	f.calls++
	return errors.New("cdp: Cannot find context with specified id (-32000)")
}
func TestSketchSceneContextLossPermanentlyRetires(t *testing.T) {
	caller := &sketchSceneLostContext{}
	scene := NewSketchSceneSession(caller, SketchSceneScope{ParentSession: "top", ParentContext: 1, Loader: "l"}, func(SketchSceneScope) bool { return true }, nil)
	scene.world = 88
	scene.session = "child"
	if _, err := scene.HitTest(context.Background(), [2]float64{1, 2}); err == nil {
		t.Fatal("lost context accepted")
	}
	if _, err := scene.HitTest(context.Background(), [2]float64{1, 2}); !errors.Is(err, ErrSketchSceneRetired) {
		t.Fatalf("scene revived: %v", err)
	}
	if caller.calls != 1 {
		t.Fatal("retired scene touched CDP")
	}
}
