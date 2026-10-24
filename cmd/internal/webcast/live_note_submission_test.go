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
		wait("acknowledged preview: "+what, func() bool {
			var e error
			if host, _, e = cast.SubmissionPreviewHost(ctx); e != nil {
				return false
			}
			selection, _, e := visible(host)
			if e != nil || len(selection.Notes) != notes || len(selection.Items) != sketches {
				return false
			}
			state, _ = core.NextPreview(state, core.PreviewHostChanged{Host: host})
			state, _ = core.NextPreview(state, core.PreviewCandidatesChanged{Selection: selection})
			var effects []core.PreviewEffect
			state, effects = core.NextPreview(state, core.PreviewRefresh{})
			for _, effect := range effects {
				if p, ok := effect.(core.PreviewPublish); ok {
					if e = cast.PublishSubmissionPreview(ctx, p.Publication); e != nil {
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
}
