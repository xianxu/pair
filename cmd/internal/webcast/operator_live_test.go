package webcast

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/submission"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/drive"
	"github.com/xianxu/pair/cmd/internal/webcast/notedraft"
	"github.com/xianxu/pair/cmd/internal/webcast/notes"
	"github.com/xianxu/pair/cmd/internal/webcast/query"
	"github.com/xianxu/pair/cmd/internal/webcast/recorder"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// One headless cast, real injected shared UI, Board and recorder. No desktop
// activation, viewer windows, external services or operator profile.
func TestOperatorLiveAnnotation(t *testing.T) {
	if os.Getenv("WEBCAST_LIVE_CHROME") == "" {
		t.Skip("set WEBCAST_LIVE_CHROME=1")
	}
	bin, err := FindChrome(os.Getenv, execLookPath, fileExists)
	if err != nil {
		t.Fatal(err)
	}
	bd := board.New(notes.Structural{}, nil)
	mem := &recording.MemSink{}
	dir := filepath.Join(t.TempDir(), "operator-recording")
	disk, err := recording.NewDirWriter(dir)
	if err != nil {
		t.Fatal(err)
	}
	hub := recording.NewHub(256)
	viewer, unsubscribe := hub.Subscribe()
	defer unsubscribe()

	var c *Chrome
	drafts, err := newCastNoteDrafts(dir, bd, func(context.Context) (Browser, error) { return c, nil }, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer drafts.Close()
	c, err = Launch(context.Background(), LaunchOptions{Binary: bin, Headless: true, ProfileDir: t.TempDir(), StartURL: "data:text/html,<title>Operator test</title><body><h1>Save</h1><input id=field><button>Save</button></body>", Record: &RecordOptions{ID: "operator", Operator: bd, NoteDrafts: drafts, ShotEvery: -1, Out: func() (recorder.Opened, error) { return recorder.Opened{Sink: recording.Tee(disk, hub, mem, bd)}, nil }}})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	bd.WithAnnotationCleanup(c.rec.ClearAnnotations)
	if err = c.SetLive(true); err != nil {
		t.Fatal(err)
	}
	var session string
	var id int64
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		done := make(chan struct{})
		c.do(func(s *loopState) {
			for ss, ctx := range s.contexts {
				session, id = ss, ctx
			}
			close(done)
		})
		<-done
		if id != 0 && bd.OperatorView().Doc > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if id == 0 {
		t.Fatal("no operator world")
	}
	eval := func(expr string) json.RawMessage {
		t.Helper()
		var res struct {
			Result struct {
				Value json.RawMessage `json:"value"`
			} `json:"result"`
			Exception any `json:"exceptionDetails"`
		}
		if err := c.call(session, "Runtime.evaluate", map[string]any{"expression": expr, "contextId": id, "returnByValue": true, "awaitPromise": true}, &res); err != nil {
			t.Fatal(err)
		}
		if res.Exception != nil {
			t.Fatalf("evaluation: %s: %+v", expr, res.Exception)
		}
		return res.Result.Value
	}
	// Resolve the closed shadow root through the debugger for test inspection.
	type node struct {
		Name     string            `json:"nodeName"`
		Backend  int               `json:"backendNodeId"`
		Children []json.RawMessage `json:"children"`
		Shadow   []json.RawMessage `json:"shadowRoots"`
	}
	var tree struct {
		Root json.RawMessage `json:"root"`
	}
	if err = c.call(session, "DOM.getDocument", map[string]any{"depth": -1, "pierce": true}, &tree); err != nil {
		t.Fatal(err)
	}
	var backend, viewerBackend int
	var visit func(json.RawMessage)
	visit = func(raw json.RawMessage) {
		var n node
		json.Unmarshal(raw, &n)
		if n.Name == "BUTTON" {
			viewerBackend = n.Backend
		}
		if n.Name == "COUCHCAST-CONTROL" && len(n.Shadow) > 0 {
			var shadow node
			json.Unmarshal(n.Shadow[0], &shadow)
			backend = shadow.Backend
		}
		for _, ch := range n.Children {
			visit(ch)
		}
	}
	visit(tree.Root)
	if backend == 0 {
		t.Fatal("no excluded host shadow root")
	}
	var resolved struct {
		Object struct {
			ID string `json:"objectId"`
		} `json:"object"`
	}
	if err = c.call(session, "DOM.resolveNode", map[string]any{"backendNodeId": backend, "executionContextId": id}, &resolved); err != nil {
		t.Fatal(err)
	}
	if err = c.call(session, "Runtime.callFunctionOn", map[string]any{"objectId": resolved.Object.ID, "functionDeclaration": "function(){globalThis.testRoot=this}"}, nil); err != nil {
		t.Fatal(err)
	}
	if string(eval(`!!testRoot.querySelector('.annotate-rail')`)) != "true" {
		t.Fatal("shared UI did not mount")
	}
	// Viewer authority starts OFF; the trusted owner has separate admission.
	if err = c.SetAnnotationMode("note"); err != nil {
		t.Fatalf("default-off owner Note: %v", err)
	}
	if string(eval(`testRoot.querySelector('.mode').disabled`)) != "false" {
		t.Fatal("default-off owner rail disabled")
	}
	if err = bd.SetAnnotationAllowed(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	waitPermission := func(allowed bool) {
		t.Helper()
		until := time.Now().Add(5 * time.Second)
		for time.Now().Before(until) {
			if string(eval(`!testRoot.querySelector('.mode').disabled`)) == strconv.FormatBool(allowed) && string(eval(`globalThis.__couchcastAnnotationDelivery.generation`)) == strconv.Quote(strconv.FormatUint(bd.AnnotationCapability().Generation, 10)) {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		t.Fatal("permission projection did not settle")
	}
	waitPermission(true)
	var deliveredGeneration string
	if err := json.Unmarshal(eval(`globalThis.__couchcastAnnotationDelivery.generation`), &deliveredGeneration); err != nil {
		t.Fatal(err)
	}
	if deliveredGeneration != strconv.FormatUint(bd.AnnotationCapability().Generation, 10) {
		t.Fatalf("operator received wrong authority generation: %s", deliveredGeneration)
	}
	if err = bd.SetAnnotationAllowed(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	waitPermission(true)
	if err = c.SetAnnotationMode("note"); err != nil {
		t.Fatalf("settled OFF owner Note: %v", err)
	}
	if err = bd.SetAnnotationAllowed(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	waitPermission(true)
	// Couch's mode command preserves page focus; Escape must still reach the
	// isolated overlay, rather than requiring a focused shadow-root control.
	for _, mode := range []string{"point", "note"} {
		eval(`document.querySelector('#field').focus()`)
		if err = c.SetAnnotationMode(mode); err != nil {
			t.Fatal(err)
		}
		for _, typ := range []string{"keyDown", "keyUp"} {
			if err = c.call(session, "Input.dispatchKeyEvent", map[string]any{"type": typ, "key": "Escape", "code": "Escape", "windowsVirtualKeyCode": 27}, nil); err != nil {
				t.Fatal(err)
			}
		}
		if string(eval(`[...testRoot.querySelectorAll('.mode')].every(b=>b.getAttribute('aria-pressed')==='false') && document.activeElement===document.querySelector('#field')`)) != "true" {
			t.Fatalf("page-focused Escape did not disable %s without moving focus", mode)
		}
	}
	if err = c.SetAnnotationMode("note"); err != nil {
		t.Fatal(err)
	}
	var xy []float64
	json.Unmarshal(eval(`(()=>{const r=document.querySelector('h1').getBoundingClientRect();return [r.x+40,r.y+10]})()`), &xy)
	for _, typ := range []string{"mousePressed", "mouseReleased"} {
		if err = c.call(session, "Input.dispatchMouseEvent", map[string]any{"type": typ, "button": "left", "clickCount": 1, "x": xy[0], "y": xy[1]}, nil); err != nil {
			t.Fatal(err)
		}
		if typ == "mousePressed" && string(eval(`testRoot.querySelector('.drag-region').hidden`)) != "true" {
			t.Fatal("note click drew the default box before release")
		}
	}
	for time.Now().Before(deadline) {
		if string(eval(`!testRoot.querySelector('.note-dialog').classList.contains('hidden')`)) == "true" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if string(eval(`!testRoot.querySelector('.note-dialog').classList.contains('hidden')`)) != "true" {
		t.Fatal("note gesture did not prepare draft")
	}

	if string(eval(`testRoot.querySelector('.drag-region').hidden`)) != "false" {
		t.Fatal("note release did not place the default box")
	}
	eval(`(()=>{const e=testRoot.querySelector('.note-input');e.value='Held operator draft';e.dispatchEvent(new Event('input',{bubbles:true}));})()`)
	// Capture starts at the gesture; typed text survives its asynchronous acknowledgment.
	deadline = time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if string(eval(`testRoot.querySelector('.note-msg').textContent.includes('Captured')`)) == "true" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if string(eval(`testRoot.querySelector('.note-msg').textContent.includes('Captured')`)) != "true" {
		t.Fatalf("capture did not become ready: %s", eval(`testRoot.querySelector('.note-msg').textContent`))
	}
	deadline = time.Now().Add(10 * time.Second)
	if path := os.Getenv("WEBCAST_OPERATOR_SHOT"); path != "" {
		var shot struct {
			Data string `json:"data"`
		}
		if err = c.call(session, "Page.captureScreenshot", map[string]any{"format": "png"}, &shot); err != nil {
			t.Fatal(err)
		}
		pixels, e := base64.StdEncoding.DecodeString(shot.Data)
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(path, pixels, 0600); e != nil {
			t.Fatal(e)
		}
	}

	ctrl := recording.Control{}
	changeControl := func(op string) {
		t.Helper()
		next, e := recording.NextControl(ctrl, op, recording.Author{ID: "v1", Name: "Ana"})
		if e != nil {
			t.Fatal(e)
		}
		ctrl = next
		if e = c.Emit(&recording.Ctl{Op: op, Who: recording.Author{ID: "v1", Name: "Ana"}, Control: next}, 0); e != nil {
			t.Fatal(e)
		}
	}
	changeControl(recording.CtlRequest)
	changeControl(recording.CtlGrant)
	c.SetDriving("Ana")
	for time.Now().Before(deadline) {
		if string(eval(`testRoot.querySelector('.note-dialog').hidden`)) == "true" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if string(eval(`testRoot.querySelector('.note-dialog').hidden && testRoot.querySelector('.note-input').value==='Held operator draft' && testRoot.activeElement!==testRoot.querySelector('.note-input')`)) != "true" {
		t.Fatal("driver did not hold/unfocus draft")
	}
	// Put a page field beneath the legacy authority button: remote hit-testing
	// and dispatch must reach the field, never toggle the cast.
	var remoteXY []float64
	json.Unmarshal(eval(`(()=>{const r=testRoot.querySelector('.cast-button').getBoundingClientRect();const f=document.querySelector('#field');Object.assign(f.style,{position:'fixed',left:r.x+'px',top:r.y+'px',width:r.width+'px',height:r.height+'px',boxSizing:'border-box'});return [r.x+r.width/2,r.y+r.height/2]})()`), &remoteXY)
	err = c.rec.WithTab(context.Background(), "v1", func(tab drive.Tab) error {
		node, e := tab.At(remoteXY[0], remoteXY[1])
		if e != nil {
			return e
		}
		if node == 0 {
			t.Error("remote hit test found no page node")
		}
		if e = tab.Dispatch(drive.Act{A: drive.ActClick, X: remoteXY[0], Y: remoteXY[1], Button: drive.ButtonLeft, Count: 1}); e != nil {
			return e
		}
		return tab.Dispatch(drive.Act{A: drive.ActText, V: "remote"})
	})
	if err != nil {
		t.Fatal(err)
	}
	if string(eval(`document.querySelector('#field').value`)) != `"remote"` || c.State().Phase != PhaseLive {
		t.Fatal("remote input reached Couch UI instead of page")
	}
	changeControl(recording.CtlRelease)
	c.SetDriving("")
	for time.Now().Before(deadline) {
		if string(eval(`!testRoot.querySelector('.note-dialog').hidden`)) == "true" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if string(eval(`testRoot.querySelector('.note-input').value==='Held operator draft' && [...testRoot.querySelectorAll('.mode')].every(b=>b.getAttribute('aria-pressed')==='false')`)) != "true" {
		t.Fatal("return did not restore held draft with pen off")
	}
	eval(`(()=>{const e=testRoot.querySelector('.note-input');e.value='Operator note';e.dispatchEvent(new Event('input',{bubbles:true}));testRoot.querySelector('.note-buttons').lastElementChild.click()})()`)
	for time.Now().Before(deadline) {
		if len(bd.OperatorView().Notes) == 1 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	v := bd.OperatorView()
	if len(v.Notes) != 1 {
		t.Fatalf("note not committed: UI %s; view %+v", eval(`testRoot.querySelector('.note-msg').textContent`), v)
	}
	if v.Notes[0].Capture == "" {
		t.Fatal("operator note omitted draw-time capture")
	}
	if v.Notes[0].Author.ID != "operator" {
		t.Fatal("operator identity lost")
	}
	// The same accepted note reaches a live subscriber and the still-open
	// recording consumed by replay/query; no end-of-cast flush is required.
	seenNote := false
	for !seenNote {
		select {
		case record, ok := <-viewer:
			if !ok {
				t.Fatal("viewer dropped before operator note")
			}
			if note, ok := record.(*recording.Note); ok {
				seenNote = note.Author.ID == "operator" && note.Text == "Operator note"
			}
		case <-time.After(time.Second):
			t.Fatal("operator note did not reach viewer")
		}
	}
	onDisk, err := recording.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	queried, err := query.New(onDisk.Records, notes.Structural{}).Notes(query.DefaultBefore)
	if err != nil || len(queried) != 1 || queried[0].Note.Author.ID != "operator" || queried[0].Note.Text != "Operator note" {
		t.Fatalf("live recording query lost operator note: %+v, %v", queried, err)
	}

	// Board publication precedes the asynchronous operator reply. Wait for
	// its acknowledgment to close the editor before the next independent gesture.
	saveDeadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(saveDeadline) {
		if string(eval(`testRoot.querySelector('.note-dialog').classList.contains('hidden')`)) == "true" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if string(eval(`testRoot.querySelector('.note-dialog').classList.contains('hidden')`)) != "true" {
		t.Fatalf("operator save acknowledgment did not close draft: %s", eval(`testRoot.querySelector('.note-msg').textContent`))
	}
	// Own operator note: open in watch, cancel without mutation, then save a
	// text revision through the real isolated command and ordered recorder.
	waitOverlay := func(expr, want string) {
		t.Helper()
		until := time.Now().Add(5 * time.Second)
		for time.Now().Before(until) {
			if string(eval(expr)) == want {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		t.Fatalf("overlay did not reach %s: %s", want, eval(expr))
	}
	waitOverlay(`!!testRoot.querySelector('.pin .chip')`, "true")
	eval(`testRoot.querySelector('.pin .chip').click()`)
	if string(eval(`testRoot.querySelector('.note-input').value`)) != `"Operator note"` {
		t.Fatal("own note did not open populated editor")
	}
	eval(`(()=>{const e=testRoot.querySelector('.note-input');e.value='cancelled edit';e.dispatchEvent(new Event('input',{bubbles:true}));[...testRoot.querySelectorAll('.note-buttons button')].find(b=>b.textContent==='Cancel').click()})()`)
	if bd.OperatorView().Notes[0].Text != "Operator note" {
		t.Fatal("cancel mutated operator note")
	}
	eval(`testRoot.querySelector('.pin .chip').click()`)
	eval(`(()=>{const e=testRoot.querySelector('.note-input');e.value='Operator note edited';e.dispatchEvent(new Event('input',{bubbles:true}));testRoot.querySelector('.note-buttons').lastElementChild.click()})()`)
	waitOverlay(`testRoot.querySelector('.note-dialog').classList.contains('hidden')`, "true")
	edited := bd.OperatorView().Notes
	if len(edited) != 1 || edited[0].ID != v.Notes[0].ID || edited[0].Text != "Operator note edited" || !edited[0].Edited || edited[0].Author != v.Notes[0].Author || edited[0].Capture != v.Notes[0].Capture || *edited[0].Region != *v.Notes[0].Region {
		t.Fatalf("edited note %+v", edited)
	}

	// Identity, not replacement scoring: common Save text has too little
	// replacement evidence. A large document must not hide this connected node,
	// and a viewer-authored note must get the same trusted debugger binding.
	waitOverlay(`!testRoot.querySelector('.pin').hidden`, "true")
	eval(`(()=>{const f=document.createDocumentFragment();for(let i=0;i<2100;i++)f.append(document.createElement('span'));document.body.append(f)})()`)
	waitOverlay(`!testRoot.querySelector('.pin').hidden`, "true")
	viewerNode, ok, e := c.rec.IDOf(context.Background(), session, viewerBackend)
	if e != nil || !ok {
		t.Fatalf("viewer anchor identity: %v %v", ok, e)
	}
	var viewerBox recording.Box
	json.Unmarshal(eval(`(()=>{const r=document.querySelector('button').getBoundingClientRect();return [r.x+scrollX,r.y+scrollY,r.width,r.height]})()`), &viewerBox)
	admission, e := bd.AdmitAnnotation()
	if e != nil {
		t.Fatal(e)
	}
	prepared, e := admission.PrepareNote(board.PrepareNoteIn{Seg: bd.OperatorView().Seg, Node: viewerNode, Box: viewerBox, Region: viewerBox}, false)
	if e != nil {
		t.Fatal(e)
	}
	owner := notedraft.Owner{Scope: "trusted-live-viewer", Identity: "trusted-live-viewer", Generation: bd.AnnotationCapability().Generation}
	status, e := drafts.Prepare(context.Background(), owner, "viewer-draw", admission, prepared, false)
	if e != nil {
		t.Fatal(e)
	}
	until := time.Now().Add(10 * time.Second)
	for status.State == "pending" && time.Now().Before(until) {
		time.Sleep(20 * time.Millisecond)
		status, e = drafts.Status(owner, "viewer-draw")
		if e != nil {
			t.Fatal(e)
		}
	}
	if status.State != "ready" {
		t.Fatalf("viewer draw: %+v", status)
	}
	status, e = drafts.Save(context.Background(), owner, "viewer-save", status.Draft, admission, board.Viewer{ID: "viewer", Name: "Viewer"}, 1, "Viewer common Save", nil)
	if e != nil {
		t.Fatal(e)
	}
	for status.State == "pending" && time.Now().Before(until) {
		time.Sleep(20 * time.Millisecond)
		status, e = drafts.Status(owner, "viewer-draw")
		if e != nil {
			t.Fatal(e)
		}
	}
	if status.State != "saved" {
		t.Fatalf("viewer save: %+v", status)
	}
	waitOverlay(`testRoot.querySelectorAll('.pin').length===2 && [...testRoot.querySelectorAll('.pin')].every(p=>!p.hidden)`, "true")
	if err = c.SetAnnotationMode("point"); err != nil {
		t.Fatal(err)
	}
	for i, typ := range []string{"mousePressed", "mouseMoved", "mouseMoved", "mouseReleased"} {
		if err = c.call(session, "Input.dispatchMouseEvent", map[string]any{"type": typ, "button": "left", "buttons": 1, "clickCount": 1, "x": xy[0] + float64(i*10), "y": xy[1]}, nil); err != nil {
			t.Fatal(err)
		}
	}
	strokeDeadline := time.Now().Add(time.Second)
	for time.Now().Before(strokeDeadline) {
		if len(bd.OperatorView().Strokes) > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if len(bd.OperatorView().Strokes) == 0 {
		t.Fatal("operator Pen produced no recorded stroke")
	}
	eval(`__couchcastOperatorMode('watch')`)
	// A bright rectangle in the excluded host gives a pixel-level proof that
	// screenshots omit the whole UI, rather than merely toggling a JS flag.
	eval(`(()=>{const e=document.createElement('div');e.style.cssText='position:fixed;left:0;top:0;width:20px;height:20px;background:rgb(255,0,255)';testRoot.append(e)})()`)
	sample := func(data string) (uint32, uint32, uint32) {
		t.Helper()
		b, e := base64.StdEncoding.DecodeString(data)
		if e != nil {
			t.Fatal(e)
		}
		im, e := png.Decode(bytes.NewReader(b))
		if e != nil {
			t.Fatal(e)
		}
		r, g, bv, _ := im.At(5, 5).RGBA()
		return r, g, bv
	}
	var rawShot struct {
		Data string `json:"data"`
	}
	if err = c.call(session, "Page.captureScreenshot", map[string]any{"format": "png"}, &rawShot); err != nil {
		t.Fatal(err)
	}
	rr, gg, bb := sample(rawShot.Data)
	if rr != 65535 || gg != 0 || bb != 65535 {
		t.Fatal("pixel witness not visible before suppression")
	}
	// Suppression covers the entire host and returns it on the exact document.
	var shot struct {
		Data string `json:"data"`
	}
	if err = recorder.CaptureScreenshot(context.Background(), c.conn, session, id, map[string]any{"format": "png"}, &shot); err != nil {
		t.Fatal(err)
	}
	rr, gg, bb = sample(shot.Data)
	if rr == 65535 && gg == 0 && bb == 65535 {
		t.Fatal("Couch UI pixels leaked into capture")
	}
	if shot.Data == "" {
		t.Fatal("no capture")
	}
	if string(eval(`document.querySelector('[data-couchcast-ui]').hidden`)) != "false" {
		t.Fatal("capture failed to restore host")
	}
	// #483 submission crops use the same exact-document suppression path.
	cropBox := recording.Box{0, 0, 40, 40}
	crops, err := c.CaptureCrops(context.Background(), bd.OperatorView().Doc, []submission.CaptureRegion{{Box: cropBox, Region: cropBox}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(crops) != 1 || len(crops[0].PNG) == 0 {
		t.Fatalf("missing submission crop: %+v", crops)
	}
	rr, gg, bb = sample(base64.StdEncoding.EncodeToString(crops[0].PNG))
	if rr == 65535 && gg == 0 && bb == 65535 {
		t.Fatal("Couch UI pixels leaked into submission crop")
	}
	if string(eval(`document.querySelector('[data-couchcast-ui]').hidden`)) != "false" {
		t.Fatal("crop failed to restore host")
	}
	if err = c.Close(); err != nil {
		t.Fatal(err)
	}
	for _, r := range mem.Records() {
		raw, _ := json.Marshal(r)
		if strings.Contains(string(raw), "couchcast-control") || strings.Contains(string(raw), "data-couchcast-ui") {
			t.Fatalf("Couch UI leaked into recording: %s", raw)
		}
	}
}
