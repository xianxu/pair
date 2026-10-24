package webcast

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	core "github.com/xianxu/pair/cmd/internal/submission"
)

// pair#849: the operator's sequence through the real console. Notes are
// shown, then a sketch (first stroke undone) is labeled by click; the preview
// refreshes, the chord's prepare (fresh read, resolve, PreviewPrepare, freeze)
// admits it, and a preview that cannot refresh says it is out of date.
func TestOperatorFeedbackLivePrepareAndStalePanel(t *testing.T) {
	if os.Getenv("WEBCAST_LIVE_CHROME") == "" {
		t.Skip("set WEBCAST_LIVE_CHROME=1")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	page := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<!doctype html><title>feedback prepare</title><style>body{margin:0}section{position:absolute;left:80px;top:100px;width:400px;height:200px;background:#eee}form{position:absolute;left:80px;top:340px;width:400px;height:120px;background:#ddd}</style><section>Public transcript</section><form>Say</form>`)
	}))
	defer page.Close()
	bin, err := FindChrome(os.Getenv, execLookPath, fileExists)
	if err != nil {
		t.Fatal(err)
	}
	profile, root := t.TempDir(), filepath.Join(t.TempDir(), "recordings")
	var chrome *Chrome
	cast, err := StartCast(ctx, CastConfig{RecordingsDir: root, Host: broadcast.NewHost(broadcast.HostConfig{Tunnel: &broadcast.FakeTunnel{}}), Logf: t.Logf,
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
	mouse := func(kind string, x, y float64, buttons int) {
		button := "left"
		if kind == "mouseMoved" {
			button = "none"
		}
		call("Input.dispatchMouseEvent", map[string]any{"type": kind, "x": x, "y": y, "button": button, "buttons": buttons, "clickCount": 1}, nil)
	}
	// The operator's display: 864×997 at 2x.
	call("Emulation.setDeviceMetricsOverride", map[string]any{"width": 864, "height": 997, "deviceScaleFactor": 2, "mobile": false}, nil)
	wait("fixture document", func() bool {
		return string(eval(0, `document.readyState==='complete'&&!!document.querySelector('form')`)) == "true"
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
	wait("operator UI", func() bool {
		return string(eval(world, `!!globalThis.__couchcastOperatorSketch?.state().live&&!__couchcastOperatorSketch.labelInput.getRootNode().host.hidden`)) == "true"
	})
	clickRoot := func(selector, text string) {
		t.Helper()
		var point [2]float64
		wait(text+" control", func() bool {
			raw := eval(world, fmt.Sprintf(`(()=>{const root=__couchcastOperatorSketch.labelInput.getRootNode(),b=[...root.querySelectorAll(%q)].find(b=>b.textContent===%q),r=b?.getBoundingClientRect();if(!b||b.disabled||!r.width||!b.contains(root.elementFromPoint(r.x+r.width/2,r.y+r.height/2)))return null;return [r.x+r.width/2,r.y+r.height/2]})()`, selector, text))
			return string(raw) != "null" && json.Unmarshal(raw, &point) == nil
		})
		mouse("mousePressed", point[0], point[1], 1)
		mouse("mouseReleased", point[0], point[1], 0)
	}
	// This oracle measures the injected production DOM independently of the
	// layout calculator. Scrolling reaches the actual adopted control nodes.
	eval(world, `globalThis.__testWorkspaceGeometry=(editorSelector,minimized)=>{
 const root=__couchcastOperatorSketch.labelInput.getRootNode(),panel=root.querySelector('.feedback-panel'),toolbar=root.querySelector('.cast-tools'),body=panel.querySelector('.floating-panel-body'),header=panel.querySelector('.floating-panel-header'),status=panel.querySelector('.feedback-status');
 const rect=e=>e.getBoundingClientRect(),p=rect(panel),t=rect(toolbar),problems=[];
 const inside=(a,b)=>a.left>=b.left-.5&&a.top>=b.top-.5&&a.right<=b.right+.5&&a.bottom<=b.bottom+.5;
 const overlaps=(a,b)=>a.left<b.right&&a.right>b.left&&a.top<b.bottom&&a.bottom>b.top;
 const viewport={left:0,top:0,right:innerWidth,bottom:innerHeight};
 if(!(p.width>0&&p.height>0&&t.width>0&&t.height>0))problems.push('hidden workspace');
 if(!inside(p,viewport)||!inside(t,viewport))problems.push('workspace outside viewport');
 if(overlaps(p,t))problems.push('panel-toolbar overlap');
 if(panel.classList.contains('minimized')!==minimized)problems.push('wrong panel mode');
 const hit=element=>{
  element.scrollIntoView({block:'nearest',inline:'nearest'});
  const r=rect(element),clip=element.closest('.cast-tools')?rect(toolbar):element.closest('.floating-panel-body')?rect(body):rect(panel);
  const visible={left:Math.max(r.left,clip.left),top:Math.max(r.top,clip.top),right:Math.min(r.right,clip.right),bottom:Math.min(r.bottom,clip.bottom)};
  const target=root.elementFromPoint((visible.left+visible.right)/2,(visible.top+visible.bottom)/2);
  if(!(visible.left<visible.right&&visible.top<visible.bottom)||!element.contains(target))problems.push('control not hit-test reachable: '+element.className+' '+element.textContent);
 };
 for(const button of toolbar.querySelectorAll('button,select'))if(rect(button).height>0)hit(button);
 if(minimized){
  const restore=panel.querySelector('.floating-panel-restore');hit(restore);
  if(status.textContent&&!restore.textContent.includes('Status'))problems.push('minimized status missing');
  if(status.textContent&&!restore.title.includes(status.textContent)&&!restore.getAttribute('aria-label')?.includes(status.textContent))problems.push('minimized status inaccessible');
 }else{
  if(!inside(rect(header),p))problems.push('header outside panel');
  if(!inside(rect(body),p))problems.push('scroll area outside panel');
  if(status.textContent&&!inside(rect(status),p))problems.push('status outside panel');
  for(const button of header.querySelectorAll('button'))if(rect(button).height>0)hit(button);
  if(getComputedStyle(panel).overflow!=='hidden'||!['auto','scroll'].includes(getComputedStyle(body).overflowY))problems.push('panel body not clipped and scrollable');
  if(editorSelector){
   const editor=panel.querySelector(editorSelector),e=rect(editor),b=rect(body);
   if(editor.hidden||e.height<=0||e.left<b.left-.5||e.right>b.right+.5)problems.push('editor unavailable or outside body width');
   for(const control of editor.querySelectorAll('button,input,textarea,select'))if(rect(control).height>0)hit(control);
   // Content may exceed the scroll area, but the browser must clip its hits.
   const current=rect(editor),clip=rect(body),x=(Math.max(current.left,clip.left)+Math.min(current.right,clip.right))/2;
   for(const y of [clip.top-1,clip.bottom+1])if(y>=0&&y<innerHeight&&editor.contains(root.elementFromPoint(x,y)))problems.push('editor hit escapes scroll area');
  }
 }
 if(!inside(rect(panel),viewport)||!inside(rect(toolbar),viewport)||overlaps(rect(panel),rect(toolbar)))problems.push('workspace escaped after control scrolling');
 return {problems,panel:p.toJSON(),toolbar:t.toJSON(),body:rect(body).toJSON(),header:rect(header).toJSON(),statusRect:rect(status).toJSON(),editor:editorSelector?rect(panel.querySelector(editorSelector)).toJSON():null,status:status.textContent,viewport:[innerWidth,innerHeight]};
}`)
	resize := func(width, height int) {
		t.Helper()
		call("Emulation.setDeviceMetricsOverride", map[string]any{"width": width, "height": height, "deviceScaleFactor": 2, "mobile": false}, nil)
		eval(world, `new Promise(resolve=>requestAnimationFrame(()=>requestAnimationFrame(resolve)))`)
		wait("private workspace restored after resize", func() bool {
			return string(eval(world, `!__couchcastOperatorSketch.labelInput.getRootNode().host.hidden`)) == "true"
		})
	}
	checkWorkspace := func(stage, editor string, minimized bool) {
		t.Helper()
		for _, viewport := range [][2]int{{1280, 800}, {768, 800}, {320, 800}, {320, 480}} {
			resize(viewport[0], viewport[1])
			result := eval(world, fmt.Sprintf(`__testWorkspaceGeometry(%q,%t)`, editor, minimized))
			var measured struct {
				Problems []string `json:"problems"`
			}
			if err := json.Unmarshal(result, &measured); err != nil || len(measured.Problems) > 0 {
				t.Fatalf("production workspace %s at %dx%d: %s (%v)", stage, viewport[0], viewport[1], result, err)
			}
			t.Logf("production workspace %s at %dx%d: %s", stage, viewport[0], viewport[1], result)
		}
		resize(800, 600)
		// Narrow clamping preserves the user's anchor on resize. Move Notes
		// back to the right through its existing keyboard control before the
		// fixture's later page gestures at its original 100..400 coordinates.
		eval(world, fmt.Sprintf(`__couchcastOperatorSketch.labelInput.getRootNode().querySelector(%q).focus()`, map[bool]string{false: ".floating-panel-drag", true: ".floating-panel-restore"}[minimized]))
		for i := 0; i < 20; i++ {
			for _, kind := range []string{"keyDown", "keyUp"} {
				call("Input.dispatchKeyEvent", map[string]any{"type": kind, "key": "ArrowRight", "code": "ArrowRight", "modifiers": 8, "windowsVirtualKeyCode": 39}, nil)
			}
		}
		if string(eval(world, `__couchcastOperatorSketch.labelInput.getRootNode().querySelector('.feedback-panel').getBoundingClientRect().left>=400`)) != "true" {
			t.Fatalf("native Notes restoration did not clear the fixture drawing coordinates: %s", eval(world, `__testWorkspaceGeometry('',`+fmt.Sprint(minimized)+`)`))
		}
	}
	settled := func() {
		wait("confirmed stroke", func() bool {
			return string(eval(world, `__couchcastOperatorSketch.state().slots.every(s=>!s.inflight&&!s.queued)`)) == "true"
		})
	}
	stroke := func(points ...[2]float64) {
		pen := func(kind string, p [2]float64, buttons int) {
			button := "none"
			if kind != "mouseMoved" {
				button = "left"
			}
			call("Input.dispatchMouseEvent", map[string]any{"type": kind, "x": p[0], "y": p[1], "button": button, "buttons": buttons, "clickCount": 1, "pointerType": "pen", "force": .5}, nil)
		}
		pen("mousePressed", points[0], 1)
		for _, p := range points[1:] {
			pen("mouseMoved", p, 1)
		}
		pen("mouseReleased", points[len(points)-1], 0)
		settled()
	}
	notes := 0
	note := func(region [4]float64, text string) {
		if notes == 0 {
			if err := cast.board.SetAnnotationAllowed(ctx, true); err != nil {
				t.Fatal(err)
			}
		}
		if string(eval(world, `[...__couchcastOperatorSketch.labelInput.getRootNode().querySelectorAll('.cast-tools .mode')].find(b=>b.textContent==='▤ Note')?.getAttribute('aria-pressed')`)) != `"true"` {
			clickRoot(".cast-tools .mode", "▤ Note")
		}
		mouse("mousePressed", region[0], region[1], 1)
		mouse("mouseMoved", region[0]+region[2], region[1]+region[3], 1)
		mouse("mouseReleased", region[0]+region[2], region[1]+region[3], 0)
		quoted, _ := json.Marshal(text)
		// The editor resets when it opens: type and save once it is ready.
		wait("note saved by the editor", func() bool {
			return string(eval(world, `(()=>{const root=__couchcastOperatorSketch.labelInput.getRootNode(),d=root.querySelector('.note-dialog'),input=root.querySelector('.note-input');if(!d||d.hidden||d.classList.contains('hidden')||!d.getBoundingClientRect().height)return false;if(input.value!==`+string(quoted)+`){input.value=`+string(quoted)+`;input.dispatchEvent(new Event('input',{bubbles:true}));}const b=[...root.querySelectorAll('.note-buttons button')].find(b=>b.textContent==='Save');if(!b||b.disabled)return false;b.click();return true})()`)) == "true"
		})
		notes++
		wait("note on the board", func() bool {
			selected, e := cast.board.Selection()
			return e == nil && len(selected.Notes) == notes
		})
	}

	// The manager's worker: read, select, resolve and publish until shown.
	var state core.PreviewState
	delivered := core.DeliveredFeedback{RecordingID: chrome.opts.Record.ID}
	visible := func(host core.PreviewHost) (core.PreviewSelection, uint64, error) {
		selected, e := cast.board.Selection()
		if e != nil {
			return core.PreviewSelection{}, 0, e
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
			return core.PreviewSelection{}, 0, e
		}
		selection, e := core.SelectVisibleFeedback(candidates, notes, delivered, core.PreviewStamp{Board: selected.SketchGeneration}, view)
		if e != nil {
			return core.PreviewSelection{}, 0, e
		}
		selection, e = cast.ResolveSubmissionPreview(ctx, host, selection)
		return selection, selected.SketchGeneration, e
	}
	var host core.PreviewHost
	show := func(what string, sketches int) {
		t.Helper()
		lastProblem := ""
		problem := func(text string) {
			if text != lastProblem {
				t.Logf("preview %s waiting: %s", what, text)
				boardView, _ := json.Marshal(cast.board.OperatorView())
				t.Logf("preview readiness board=%s host=%+v browser=%s", boardView, host, eval(world, `(()=>{const root=__couchcastOperatorSketch.labelInput.getRootNode();let visibility;try{visibility=__couchcastFeedbackVisibility()}catch(e){visibility={error:e.message}}return {visibility,busy:__couchcastCaptureBusy(),hostHidden:root.host.hidden,status:root.querySelector('.feedback-status').textContent,viewport:[innerWidth,innerHeight],scroll:[scrollX,scrollY],page:[...document.querySelectorAll('section,form')].map(e=>({tag:e.tagName,rect:e.getBoundingClientRect().toJSON()}))}})()`))
				lastProblem = text
			}
		}
		wait("acknowledged preview: "+what, func() bool {
			var e error
			if host, _, e = cast.SubmissionPreviewHost(ctx); e != nil {
				problem(e.Error())
				return false
			}
			selection, _, e := visible(host)
			if e != nil || len(selection.Notes) != notes || len(selection.Items) != sketches {
				problem(fmt.Sprintf("visible %d sketches/%d notes, want %d/%d: %v", len(selection.Items), len(selection.Notes), sketches, notes, e))
				return false
			}
			state, _ = core.NextPreview(state, core.PreviewHostChanged{Host: host})
			state, _ = core.NextPreview(state, core.PreviewCandidatesChanged{Selection: selection})
			var effects []core.PreviewEffect
			state, effects = core.NextPreview(state, core.PreviewRefresh{})
			for _, effect := range effects {
				if p, ok := effect.(core.PreviewPublish); ok {
					if e = cast.PublishSubmissionPreview(ctx, p.Publication); e != nil {
						problem(e.Error())
						state, _ = core.NextPreview(state, core.PreviewPublishFailed{Publication: p.Publication})
						return false
					}
					state, _ = core.NextPreview(state, core.PreviewPublishAccepted{Publication: p.Publication})
				}
			}
			return state.Phase() == core.PreviewShown
		})
	}
	heading := func() string {
		var text string
		_ = json.Unmarshal(eval(world, `(()=>{const text=__couchcastOperatorSketch.labelInput.getRootNode().querySelector('.feedback-status').textContent,index=text.indexOf('Pending feedback');return index<0?text:text.slice(index)})()`), &text)
		return text
	}

	// Opening either existing editor must restore a minimized Notes panel.
	clickRoot(".floating-panel-header button", "Minimize")
	note([4]float64{100, 120, 300, 70}, "Operator: if the channel is public, why is this here?")
	note([4]float64{100, 350, 300, 90}, "Operator: three levels of audience control; why do we need 3?")
	show("two notes", 0)
	checkWorkspace("expanded Notes and status", "", false)
	// Deliberately overlap the real elements: the independent oracle must
	// reject the mutation, then accept the restored production styles.
	if got := string(eval(world, `(()=>{const root=__couchcastOperatorSketch.labelInput.getRootNode(),panel=root.querySelector('.feedback-panel'),toolbar=root.querySelector('.cast-tools'),before=panel.style.cssText,r=toolbar.getBoundingClientRect();try{panel.style.left=r.left+'px';panel.style.top=r.top+'px';return __testWorkspaceGeometry('',false).problems.includes('panel-toolbar overlap')}finally{panel.style.cssText=before}})()`)); got != "true" {
		t.Fatal("workspace rectangle oracle accepted deliberately overlapping production elements")
	}
	t.Log("workspace rectangle oracle rejected deliberately overlapping production panel and toolbar")
	clickRoot(".floating-panel-header button", "Minimize")
	checkWorkspace("minimized Notes and status", "", true)
	var restoreText string
	if err = json.Unmarshal(eval(world, `__couchcastOperatorSketch.labelInput.getRootNode().querySelector('.floating-panel-restore').textContent`), &restoreText); err != nil {
		t.Fatal(err)
	}
	clickRoot(".floating-panel-restore", restoreText)
	// The operator's sketch, drawn while that preview was shown.
	clickRoot(".cast-tools button", "✎ Pen")
	wait("drawing", func() bool { return string(eval(world, `__couchcastOperatorSketch.state().phase`)) == `"drawing"` })
	stroke([2]float64{100, 150}, [2]float64{180, 150}, [2]float64{260, 152})
	clickRoot(".cast-tools button", "Undo")
	wait("undo recorded", func() bool {
		return string(eval(world, `!__couchcastOperatorSketch.controls.redo.disabled`)) == "true"
	})
	stroke([2]float64{90, 110}, [2]float64{200, 106}, [2]float64{204, 130}, [2]float64{92, 132}, [2]float64{90, 110})
	clickRoot(".floating-panel-header button", "Minimize")
	clickRoot(".cast-tools button", "Done")
	wait("label editor", func() bool {
		return string(eval(world, `!__couchcastOperatorSketch.labelInput.parentElement.hidden&&__couchcastOperatorSketch.labelInput.getBoundingClientRect().height>0`)) == "true"
	})
	checkWorkspace("sketch label editor", ".sketch-label", false)
	eval(world, `(()=>{const input=__couchcastOperatorSketch.labelInput;input.value='make this a dropdown menu to select a different audience? or a "viewpoint"?';input.dispatchEvent(new Event('input',{bubbles:true}));return true})()`)
	clickRoot("button", "Save label")
	wait("sketch labeled", func() bool {
		selected, e := cast.board.Selection()
		return e == nil && len(selected.Sketches) == 1 && string(eval(world, `!globalThis.__couchcastPrivateSceneFrame`)) == "true"
	})
	show("two notes and the sketch", 1)
	if got := heading(); got != "Pending feedback: 1 sketches, 2 annotations" {
		t.Fatalf("panel heading %q", got)
	}

	// A preview that cannot refresh says so instead of showing old counts; the
	// next acknowledged publication restores them.
	if err = cast.MarkSubmissionPreviewStale(ctx, host, "the panel did not show it: test"); err != nil {
		t.Fatal(err)
	}
	if got := heading(); got != "Pending feedback is out of date: the panel did not show it: test" {
		t.Fatalf("stale heading %q", got)
	}
	// A failed refresh retires the host, as the manager's does; the refresh
	// after it is a new operation.
	state, _ = core.NextPreview(state, core.PreviewHostChanged{})
	show("after the failed refresh", 1)
	if got := heading(); got != "Pending feedback: 1 sketches, 2 annotations" {
		t.Fatalf("restored heading %q", got)
	}

	// The press (pair#849): the board's feedback, whatever the panel shows.
	selected, err := cast.board.Selection()
	if err != nil {
		t.Fatal(err)
	}
	if press := selected.PressSelection(delivered); len(press.Sketches) != 1 || len(press.Notes) != notes {
		t.Fatalf("press took %d sketches, %d notes", len(press.Sketches), len(press.Notes))
	}
	if got, err := cast.SubmissionUnsavedNote(ctx); err != nil || got != "" {
		t.Fatalf("nothing unsaved, yet: %q %v", got, err)
	}
	// A note still being written is the one thing a press refuses.
	if string(eval(world, `[...__couchcastOperatorSketch.labelInput.getRootNode().querySelectorAll('.cast-tools .mode')].find(b=>b.textContent==='▤ Note')?.getAttribute('aria-pressed')`)) != `"true"` {
		clickRoot(".cast-tools .mode", "▤ Note")
	}
	mouse("mousePressed", 100, 470, 1)
	mouse("mouseMoved", 400, 540, 1)
	mouse("mouseReleased", 400, 540, 0)
	wait("an unsaved draft reported", func() bool {
		got, err := cast.SubmissionUnsavedNote(ctx)
		return err == nil && got == "a new note is not saved yet"
	})
	// Finish the ordinary editor's private snapshot before changing viewport;
	// its Save readiness is the existing controller's capture-completion signal.
	eval(world, `(()=>{const input=__couchcastOperatorSketch.labelInput.getRootNode().querySelector('.note-input');input.value='Unsent workspace qualification draft';input.dispatchEvent(new Event('input',{bubbles:true}));return true})()`)
	wait("ordinary editor capture complete before geometry", func() bool {
		return string(eval(world, `(()=>{const root=__couchcastOperatorSketch.labelInput.getRootNode(),save=[...root.querySelectorAll('.note-buttons button')].find(b=>b.textContent==='Save');return save&&!save.disabled})()`)) == "true"
	})
	checkWorkspace("ordinary note editor", ".note-dialog", false)
	// Check the integrated host's real drag lifetime after all geometry stages.
	eval(world, `(()=>{
 const root=__couchcastOperatorSketch.labelInput.getRootNode();globalThis.__testWorkspaceDrag=[];
 const record=e=>__testWorkspaceDrag.push({type:e.type,attribute:e.attribute,oldDisplayNone:e.oldDisplayNone,x:e.clientX,y:e.clientY,pointer:e.pointerId,button:e.button,buttons:e.buttons,isPrimary:e.isPrimary,target:e.target?.className,captured:root.querySelector('.floating-panel-header').hasPointerCapture(e.pointerId||1),hidden:root.host.hidden,display:root.host.style.display,status:root.querySelector('.feedback-status').textContent,panel:root.querySelector('.feedback-panel').getBoundingClientRect().toJSON()});
 for(const type of ['pointerdown','pointermove','pointerup','pointercancel','gotpointercapture','lostpointercapture'])root.addEventListener(type,record,{capture:true});
 window.addEventListener('resize',record);
 new MutationObserver(records=>{for(const mutation of records)record({type:'host mutation',attribute:mutation.attributeName,oldDisplayNone:mutation.oldValue?.includes('display: none')})}).observe(root.host,{attributes:true,attributeFilter:['hidden','style'],attributeOldValue:true});
 new MutationObserver(()=>record({type:'status mutation'})).observe(root.querySelector('.feedback-status'),{childList:true,subtree:true,characterData:true});
 return true;
})()`)
	var drag [3]float64
	if err = json.Unmarshal(eval(world, `(()=>{const root=__couchcastOperatorSketch.labelInput.getRootNode(),p=root.querySelector('.feedback-panel').getBoundingClientRect(),h=root.querySelector('.floating-panel-drag').getBoundingClientRect();return [h.left+20,h.top+20,p.left]})()`), &drag); err != nil {
		t.Fatal(err)
	}
	mouse("mousePressed", drag[0], drag[1], 1)
	// Match the established standalone native fixture: identify the held left
	// button on mouseMoved as well. CDP button:none clears pending capture.
	call("Input.dispatchMouseEvent", map[string]any{"type": "mouseMoved", "x": drag[0] - 40, "y": drag[1], "button": "left", "buttons": 1, "clickCount": 1}, nil)
	eval(world, `new Promise(resolve=>requestAnimationFrame(()=>requestAnimationFrame(resolve)))`)
	mouse("mouseReleased", drag[0]-40, drag[1], 0)
	eval(world, `new Promise(resolve=>requestAnimationFrame(()=>requestAnimationFrame(resolve)))`)
	if string(eval(world, fmt.Sprintf(`Math.abs(__couchcastOperatorSketch.labelInput.getRootNode().querySelector('.feedback-panel').getBoundingClientRect().left-(%f-40))<1`, drag[2]))) != "true" {
		t.Fatalf("production Notes drag canceled or lost its anchor: %s", eval(world, `__testWorkspaceDrag`))
	}
	t.Logf("production Notes drag retained 40px move: %s", eval(world, `__testWorkspaceDrag`))
}
