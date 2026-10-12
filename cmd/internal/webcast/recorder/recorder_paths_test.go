package recorder

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/webcast/assets"
	rec "github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// The recorder's paths beyond the DOM events, each against the fake Chrome:
// out-of-process frames, stylesheets, truncated text, inline style, assets,
// screenshots and layout, detach, and snapshots that serve several reasons.

func (h *harness) ctxCreated(session, frameID string, id int) {
	h.r.Send(Event{Session: session, Method: "Runtime.executionContextCreated", Params: raw(map[string]any{
		"context": map[string]any{"id": id, "name": "couchcast", "auxData": map[string]any{"frameId": frameID}}})})
}

func (h *harness) send(session, method string, params any) {
	h.r.Send(Event{Session: session, Method: method, Params: raw(params), Seq: h.fc.next()})
}

func (h *harness) count(kind string) int {
	n := 0
	for _, r := range h.out.sink.Records() {
		if r.Kind() == kind {
			n++
		}
	}
	return n
}

func (h *harness) last(kind string) rec.Record {
	var out rec.Record
	for _, r := range h.out.sink.Records() {
		if r.Kind() == kind {
			out = r
		}
	}
	return out
}

func (h *harness) mustValidate() *rec.State {
	h.t.Helper()
	h.validate()
	s, err := rec.Reduce(h.out.sink.Records(), 1<<62)
	if err != nil {
		h.t.Fatal(err)
	}
	return s
}

func TestRecorderOutOfProcessFrame(t *testing.T) {
	h := newHarness(t)
	a := h.page("A", "http://a/", "L1")
	ifr := a.el("iframe")
	ifr.frame = "FR"
	a.insert(a.body(), nil, ifr)
	a.take()
	f := h.page("F", "http://other/", "LF")
	f.frameID = "FR"
	f.insert(f.body(), nil, f.el("p"))
	f.take()
	h.r.Send(FrameAttached{Session: "F", Parent: "A", FrameID: "FR"})
	h.r.Send(Live{Tab: rec.Tab{ID: "T-A"}, Session: "A"})
	h.settle()
	// A change inside the frame is recorded under its document.
	p := f.el("span")
	f.insert(f.body(), nil, p)
	h.events("F", f)
	h.settle()
	st := h.mustValidate()
	ifrID := h.id("A", ifr.backend)
	frame := st.Node(ifrID).Frame
	if frame == nil || frame.URL != "http://other/" {
		t.Fatalf("frame document %+v\n%s", frame, h.kinds())
	}
	spanID := h.id("F", p.backend)
	if st.Node(spanID) == nil {
		t.Fatal("the frame's change is not in the recording")
	}
	// The frame goes: its later events are ignored, nothing breaks.
	h.r.Send(Detached{Session: "F"})
	f.insert(f.body(), nil, f.el("b"))
	h.events("F", f)
	h.settle()
	h.mustValidate()
}

func TestRecorderStylesTextAndInlineStyle(t *testing.T) {
	h := newHarness(t)
	a := h.page("A", "http://a/", "L1")
	style := a.el("style")
	full := strings.Repeat("x", maxCDPText) + "END"
	long := a.txt(full[:maxCDPText]) // what CDP sends: cut at 10 000
	long.parent = style
	style.kids = []*fnode{long}
	div := a.el("div")
	a.insert(a.doc.kids[1].kids[0], nil, style)
	a.insert(a.body(), nil, div)
	a.take()
	h.fc.more = func(session, method string, pm map[string]any, p *fakePage) (any, error, bool) {
		switch method {
		case "CSS.getStyleSheetText":
			return map[string]any{"text": "c { color: red }"}, nil, true
		case "DOM.resolveNode":
			return map[string]any{"object": map[string]any{"objectId": fmt.Sprint(pm["backendNodeId"])}}, nil, true
		case "Runtime.callFunctionOn":
			b, _ := strconv.Atoi(fmt.Sprint(pm["objectId"]))
			fn := fmt.Sprint(pm["functionDeclaration"])
			switch {
			case strings.Contains(fn, "this.data") && b == long.backend:
				return map[string]any{"result": map[string]any{"value": full}}, nil, true
			case strings.Contains(fn, "cssRules") && b == style.backend:
				return map[string]any{"result": map[string]any{"value": "h1 { font-style: italic }"}}, nil, true
			}
		case "DOM.getAttributes":
			return map[string]any{"attributes": []string{"style", "color: blue"}}, nil, true
		}
		return nil, nil, false
	}
	h.ctxCreated("A", a.frameID, 7)
	h.r.Send(Live{Tab: rec.Tab{ID: a.frameID}, Session: "A"})
	h.settle()
	h.send("A", "CSS.styleSheetAdded", map[string]any{"header": map[string]any{"styleSheetId": "c1", "frameId": a.frameID, "isConstructed": true}})
	h.send("A", "DOM.adoptedStyleSheetsModified", map[string]any{"nodeId": a.doc.nodeID, "adoptedStyleSheets": []string{"c1"}})
	h.send("A", "CSS.styleSheetAdded", map[string]any{"header": map[string]any{"styleSheetId": "s1", "frameId": a.frameID, "ownerNode": style.backend}})
	h.send("A", "CSS.styleSheetChanged", map[string]any{"styleSheetId": "s1"})
	h.send("A", "DOM.inlineStyleInvalidated", map[string]any{"nodeIds": []int{div.nodeID}})
	h.settle()
	st := h.mustValidate()
	if len(st.Adopted[st.Top.ID]) != 1 || st.Sheets[st.Adopted[st.Top.ID][0]].Text != "c { color: red }" {
		t.Errorf("constructed sheet: %v", st.Adopted)
	}
	styleID := h.id("A", style.backend)
	owned := false
	for _, sh := range st.Sheets {
		owned = owned || (sh.Owner == styleID && sh.Text == "h1 { font-style: italic }")
	}
	if !owned {
		t.Error("the style's CSSOM rules are not recorded")
	}
	longID := h.id("A", long.backend)
	if st.Node(longID).V != full {
		t.Errorf("the text CDP cut is not read in full (%d chars)", len(st.Node(longID).V))
	}
	divID := h.id("A", div.backend)
	if st.Node(divID).A["style"] != "color: blue" {
		t.Errorf("inline style: %v", st.Node(divID).A)
	}
}

func TestRecorderAssets(t *testing.T) {
	store := assets.NewMemStore()
	h := &harness{t: t, fc: newFakeChrome(), out: &memOut{}}
	h.r = Start(Options{ID: "t", Call: h.fc, ShotEvery: -1,
		Out: func() (Opened, error) { return Opened{Sink: &h.out.sink, Assets: store}, nil }})
	t.Cleanup(func() { h.r.Close() })
	a := h.page("A", "http://a/", "L1")
	var mu sync.Mutex
	fetched := map[string]int{}
	h.fc.more = func(session, method string, pm map[string]any, p *fakePage) (any, error, bool) {
		mu.Lock()
		defer mu.Unlock()
		switch method {
		case "Page.getResourceTree":
			return map[string]any{"frameTree": map[string]any{"frame": map[string]any{"id": a.frameID},
				"resources": []any{map[string]any{"url": "http://a/old.png", "type": "Image", "mimeType": "image/png"},
					map[string]any{"url": "http://a/app.js", "type": "Script", "mimeType": "text/javascript"}}}}, nil, true
		case "Page.getResourceContent":
			fetched[fmt.Sprint(pm["url"])]++
			return map[string]any{"content": "old", "base64Encoded": false}, nil, true
		case "Network.getResponseBody":
			fetched[fmt.Sprint(pm["requestId"])]++
			return map[string]any{"body": base64.StdEncoding.EncodeToString([]byte("new")), "base64Encoded": true}, nil, true
		}
		return nil, nil, false
	}
	h.r.Send(Live{Tab: rec.Tab{ID: a.frameID}, Session: "A"})
	h.settle()
	time.Sleep(50 * time.Millisecond)
	// A load while live, then one finishing as the cast pauses: both bind.
	h.send("A", "Network.responseReceived", map[string]any{"requestId": "r1", "type": "Image", "response": map[string]any{"url": "http://a/new.png", "mimeType": "image/png"}})
	h.send("A", "Network.loadingFinished", map[string]any{"requestId": "r1"})
	h.r.Send(Pause{})
	waitFor(t, func() bool { h.settle(); return h.count("asset") == 2 }, "two assets")
	// Going live again doesn't fetch what the recording already has.
	h.r.Send(Live{Tab: rec.Tab{ID: a.frameID}, Session: "A"})
	h.settle()
	time.Sleep(50 * time.Millisecond)
	h.settle()
	st := h.mustValidate()
	if st.Assets["http://a/old.png"] == nil || st.Assets["http://a/new.png"] == nil || st.Assets["http://a/app.js"] != nil {
		t.Fatalf("assets %v", st.Assets)
	}
	mu.Lock()
	defer mu.Unlock()
	if fetched["http://a/old.png"] != 1 {
		t.Errorf("an asset the recording has was fetched %d times", fetched["http://a/old.png"])
	}
}

func waitFor(t *testing.T, ok func() bool, what string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for !ok() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// frameRecorder is the FrameWriter double: it keeps screenshots and puts
// their records into the same sink, as DirWriter does into frames/index.
type frameRecorder struct {
	mu     sync.Mutex
	frames []*rec.Frame
}

func (f *frameRecorder) PutFrame(fr *rec.Frame, png []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(png) == 0 {
		return fmt.Errorf("empty png")
	}
	f.frames = append(f.frames, fr)
	return nil
}

func TestRecorderScreenshotsAndLayout(t *testing.T) {
	frames := &frameRecorder{}
	h := &harness{t: t, fc: newFakeChrome(), out: &memOut{}}
	h.r = Start(Options{ID: "t", Call: h.fc, ShotEvery: time.Hour,
		Out: func() (Opened, error) { return Opened{Sink: &h.out.sink, Frames: frames}, nil }})
	t.Cleanup(func() { h.r.Close() })
	a := h.page("A", "http://a/", "L1")
	div := a.el("div")
	a.insert(a.body(), nil, div)
	a.take()
	var img bytes.Buffer
	png.Encode(&img, image.NewRGBA(image.Rect(0, 0, 500, 350)))
	h.fc.more = func(session, method string, pm map[string]any, p *fakePage) (any, error, bool) {
		switch method {
		case "Page.captureScreenshot":
			return map[string]any{"data": base64.StdEncoding.EncodeToString(img.Bytes())}, nil, true
		case "DOMSnapshot.captureSnapshot":
			return map[string]any{"strings": []string{a.frameID}, "documents": []any{map[string]any{
				"frameId": 0, "scrollOffsetX": 0, "scrollOffsetY": 100,
				"nodes":  map[string]any{"backendNodeId": []int{div.backend, 99999}},
				"layout": map[string]any{"nodeIndex": []int{0, 1}, "bounds": [][]float64{{10, 120, 50, 20}, {0, 0, 1, 1}}}}}}, nil, true
		}
		return nil, nil, false
	}
	h.r.Send(Live{Tab: rec.Tab{ID: a.frameID}, Session: "A"})
	waitFor(t, func() bool { h.settle(); return h.count("layout") == 1 }, "a screenshot's layout")
	frames.mu.Lock()
	n := len(frames.frames)
	f0 := frames.frames[0]
	frames.mu.Unlock()
	if n != 1 || f0.W != 500 || f0.H != 350 || f0.Scale != shotScale || f0.File != "frames/0.png" {
		t.Fatalf("frames %d %+v", n, f0)
	}
	l := h.last("layout").(*rec.Layout)
	divID := h.id("A", div.backend)
	// The top document's boxes are page coordinates already, whatever the
	// mirror's last scroll; the unknown backend id is left out.
	if len(l.Boxes) != 1 || l.Boxes[divID] != (rec.Box{10, 120, 50, 20}) {
		t.Fatalf("layout %v", l.Boxes)
	}
	h.mustValidate()
}

// The cast page's session goes (its tab closed) and the tracker moves the
// cast to the next tab: a tab segment, never a gap.
func TestRecorderActivePageDetach(t *testing.T) {
	h := newHarness(t)
	h.page("A", "http://a/", "L1")
	h.page("B", "http://b/", "L2")
	h.r.Send(Live{Tab: rec.Tab{ID: "T-A"}, Session: "A"})
	h.r.Send(Detached{Session: "A"})
	h.r.Send(Live{Tab: rec.Tab{ID: "T-B"}, Session: "B"})
	h.settle()
	h.r.Close()
	if got := h.kinds(); got != "head seg:live doc seg:tab doc stop:end" {
		t.Fatalf("records %s", got)
	}
	h.mustValidate()
}

// A desync and a document update together cost one snapshot.
func TestRecorderOneSnapshotServesEveryReason(t *testing.T) {
	h := newHarness(t)
	a := h.page("A", "http://a/", "L1")
	gc := &gatedCaller{fakeChrome: h.fc, gate: make(chan struct{})}
	close(gc.gate)
	h.r.Close()
	h.out = &memOut{}
	h.r = Start(Options{ID: "t", Call: gc, ShotEvery: -1, Out: func() (Opened, error) { return Opened{Sink: &h.out.sink}, nil }})
	h.r.Send(Live{Tab: rec.Tab{ID: "T-A"}, Session: "A"})
	h.settle()
	before := h.fc.countCalls("A DOM.getDocument")
	gc.hold()
	// Busy: the recorder is inside a call while both reasons arrive.
	h.send("A", "DOM.inlineStyleInvalidated", map[string]any{"nodeIds": []int{a.body().nodeID}})
	gc.waitEntered(t)
	a.insert(a.body(), nil, a.el("p"))
	h.send("A", "DOM.attributeModified", map[string]any{"nodeId": 424242, "name": "x", "value": "y"}) // desync
	h.send("A", "DOM.documentUpdated", map[string]any{})
	gc.release()
	h.settle()
	h.settle()
	if got := h.fc.countCalls("A DOM.getDocument") - before; got != 1 {
		t.Fatalf("%d snapshots, want 1", got)
	}
	st := h.mustValidate()
	if err := sameTree(a.doc, st.Top, "doc"); err != nil {
		t.Fatal(err)
	}
}

func (f *fakeChrome) countCalls(call string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, c := range f.calls {
		if c == call {
			n++
		}
	}
	return n
}

// id is a node's recording id; the test fails when it has none.
func (h *harness) id(session string, backend int) rec.NodeID {
	h.t.Helper()
	id, ok, err := h.r.IDOf(context.Background(), session, backend)
	if err != nil || !ok {
		h.t.Fatalf("no id for %s/%d: %v", session, backend, err)
	}
	return id
}
