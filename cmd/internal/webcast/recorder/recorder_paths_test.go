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
	"sync/atomic"
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
	h.detach("F")
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
	h.confirmUI("A", a.frameID, 1)
	h.r.Send(Live{Tab: rec.Tab{ID: a.frameID}, Session: "A"})
	waitFor(t, func() bool { h.settle(); return h.count("layout") == 1 }, "a screenshot's layout")
	frames.mu.Lock()
	n := len(frames.frames)
	f0 := frames.frames[0]
	frames.mu.Unlock()
	if n != 1 || f0.W != 250 || f0.H != 175 || f0.Scale != 0.25 || f0.File != "frames/0.png" {
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
	h.settle() // Establish an active captured page before detaching its CDP session.
	h.detach("A")
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

// Assets Chrome no longer holds (a font, a memory-cached image) or never
// listed are loaded in the page's context, once each: a listed resource it
// can't read, an image the DOM references (src and srcset; never data:),
// and a FontFace font's src.
func TestRecorderLoadsAssetsChromeDoesntHold(t *testing.T) {
	store := assets.NewMemStore()
	h := &harness{t: t, fc: newFakeChrome(), out: &memOut{}}
	h.r = Start(Options{ID: "t", Call: h.fc, ShotEvery: -1,
		Out: func() (Opened, error) { return Opened{Sink: &h.out.sink, Assets: store}, nil }})
	t.Cleanup(func() { h.r.Close() })
	a := h.page("A", "http://a/post/", "L1")
	old := refSettle
	refSettle = 20 * time.Millisecond
	t.Cleanup(func() { refSettle = old })
	a.insert(a.body(), nil, a.el("img", "src", "../chosen.png", "srcset", "/x1.png 1x, /x2.png 2x"))
	pic := a.el("picture")
	a.insert(a.body(), nil, pic)
	a.insert(pic, nil, a.el("img", "src", "/in-picture.png"))
	a.insert(a.body(), nil, a.el("img", "src", "/below-the-fold.png", "loading", "lazy"))
	a.insert(a.body(), nil, a.el("img", "src", "data:image/png;base64,AAAA"))
	a.insert(a.body(), nil, a.el("img", "src", "/avatar.png")) // the same asset twice
	a.take()
	var mu sync.Mutex
	loads := map[string]int{}
	h.fc.more = func(session, method string, pm map[string]any, p *fakePage) (any, error, bool) {
		mu.Lock()
		defer mu.Unlock()
		switch method {
		case "Page.getResourceTree":
			return map[string]any{"frameTree": map[string]any{"frame": map[string]any{"id": a.frameID},
				"resources": []any{map[string]any{"url": "http://a/font.woff2", "type": "Font", "mimeType": "font/woff2"}}}}, nil, true
		case "Page.getResourceContent":
			return nil, fmt.Errorf("No resource with given URL found"), true
		case "Network.loadNetworkResource":
			u := fmt.Sprint(pm["url"])
			loads[u]++
			return map[string]any{"resource": map[string]any{"success": true, "httpStatusCode": 200, "stream": "s:" + u,
				"headers": map[string]string{"Content-Type": "image/png; charset=binary"}}}, nil, true
		case "IO.read":
			// Two chunks, the second base64.
			if loads[fmt.Sprint(pm["handle"])+"#"] == 0 {
				loads[fmt.Sprint(pm["handle"])+"#"]++
				return map[string]any{"data": "ab", "eof": false}, nil, true
			}
			return map[string]any{"data": base64.StdEncoding.EncodeToString([]byte("cd")), "base64Encoded": true, "eof": true}, nil, true
		}
		return nil, nil, false
	}
	h.r.Send(Live{Tab: rec.Tab{ID: a.frameID}, Session: "A"})
	h.send("A", "CSS.fontsUpdated", map[string]any{"font": map[string]any{"fontFamily": "Late", "src": "http://a/late.woff2"}})
	want := []string{"http://a/avatar.png", "http://a/font.woff2", "http://a/late.woff2"}
	waitFor(t, func() bool { h.settle(); return h.count("asset") == len(want) }, "the loaded assets")
	st := h.mustValidate()
	for _, u := range want {
		x := st.Assets[u]
		if x == nil {
			t.Fatalf("%s not captured: %v", u, st.Assets)
		}
		if body, err := store.Get(x.Hash); err != nil || string(body) != "abcd" || x.MIME != "image/png" {
			t.Errorf("%s: %q %s %v", u, body, x.MIME, err)
		}
	}
	mu.Lock()
	defer mu.Unlock()
	if loads["http://a/avatar.png"] != 1 {
		t.Errorf("avatar loaded %d times", loads["http://a/avatar.png"])
	}
	for u := range loads {
		if strings.HasPrefix(u, "data:") || strings.Contains(u, "x1.png") || strings.Contains(u, "below-the-fold") ||
			strings.Contains(u, "chosen") || strings.Contains(u, "in-picture") {
			t.Errorf("%s was loaded: data:, srcset, lazy and browser-chosen images never are", u)
		}
	}
}

// A reference and a Network response for one URL, in either order: the
// response's body is the one fetched, and the reference loads nothing.
func TestRecorderAssetOwnerIndependentOfOrder(t *testing.T) {
	old := refSettle
	refSettle = 200 * time.Millisecond
	t.Cleanup(func() { refSettle = old })
	for _, order := range []string{"ref first", "response first"} {
		t.Run(order, func(t *testing.T) {
			store := assets.NewMemStore()
			h := &harness{t: t, fc: newFakeChrome(), out: &memOut{}}
			h.r = Start(Options{ID: "t", Call: h.fc, ShotEvery: -1,
				Out: func() (Opened, error) { return Opened{Sink: &h.out.sink, Assets: store}, nil }})
			t.Cleanup(func() { h.r.Close() })
			a := h.page("A", "http://a/", "L1")
			var mu sync.Mutex
			calls := map[string]int{}
			h.fc.more = func(session, method string, pm map[string]any, p *fakePage) (any, error, bool) {
				mu.Lock()
				defer mu.Unlock()
				calls[method]++
				switch method {
				case "Network.getResponseBody":
					return map[string]any{"body": "real"}, nil, true
				case "Network.loadNetworkResource":
					return nil, fmt.Errorf("must not load"), true
				}
				return nil, nil, false
			}
			h.r.Send(Live{Tab: rec.Tab{ID: a.frameID}, Session: "A"})
			h.settle()
			response := func() {
				h.send("A", "Network.responseReceived", map[string]any{"requestId": "r", "type": "Image", "response": map[string]any{"url": "http://a/new.png", "mimeType": "image/png"}})
				h.send("A", "Network.loadingFinished", map[string]any{"requestId": "r"})
			}
			ref := func() {
				a.insert(a.body(), nil, a.el("img", "src", "/new.png"))
				h.events("A", a)
			}
			if order == "ref first" {
				ref()
				response()
			} else {
				response()
				ref()
			}
			time.Sleep(3 * refSettle)
			waitFor(t, func() bool { h.settle(); return h.count("asset") == 1 }, "the asset")
			mu.Lock()
			defer mu.Unlock()
			if calls["Network.loadNetworkResource"] != 0 || calls["Network.getResponseBody"] != 1 {
				t.Fatalf("calls %v", calls)
			}
		})
	}
}

// A Retina cast (DPR 2): the frame is half the viewport's CSS size, like
// any other, and its scale says image pixels per CSS pixel.
func TestRecorderScreenshotAtDPR2(t *testing.T) {
	frames := &frameRecorder{}
	h := &harness{t: t, fc: newFakeChrome(), out: &memOut{}}
	h.r = Start(Options{ID: "t", Call: h.fc, ShotEvery: time.Hour,
		Out: func() (Opened, error) { return Opened{Sink: &h.out.sink, Frames: frames}, nil }})
	t.Cleanup(func() { h.r.Close() })
	a := h.page("A", "http://a/", "L1")
	var img bytes.Buffer
	png.Encode(&img, image.NewRGBA(image.Rect(0, 0, 2000, 1400))) // 1000×700 CSS px at DPR 2
	h.fc.more = func(session, method string, pm map[string]any, p *fakePage) (any, error, bool) {
		switch method {
		case "Runtime.evaluate":
			if pm["expression"] == "devicePixelRatio" {
				return map[string]any{"result": map[string]any{"value": 2}}, nil, true
			}
		case "Page.captureScreenshot":
			return map[string]any{"data": base64.StdEncoding.EncodeToString(img.Bytes())}, nil, true
		}
		return nil, nil, false
	}
	h.confirmUI("A", a.frameID, 1)
	h.r.Send(Live{Tab: rec.Tab{ID: a.frameID}, Session: "A"})
	waitFor(t, func() bool { frames.mu.Lock(); defer frames.mu.Unlock(); return len(frames.frames) == 1 }, "a screenshot")
	frames.mu.Lock()
	f := frames.frames[0]
	frames.mu.Unlock()
	if f.W != 500 || f.H != 350 || f.Scale != 0.5 {
		t.Fatalf("frame %+v, want 500×350 at scale 0.5", *f)
	}
	h.mustValidate()
}

// A response that comes after a reference already loaded its URL fetches
// nothing: one fetch per URL per recording.
func TestRecorderLateResponseDoesntRefetch(t *testing.T) {
	old := refSettle
	refSettle = 10 * time.Millisecond
	t.Cleanup(func() { refSettle = old })
	store := assets.NewMemStore()
	h := &harness{t: t, fc: newFakeChrome(), out: &memOut{}}
	h.r = Start(Options{ID: "t", Call: h.fc, ShotEvery: -1,
		Out: func() (Opened, error) { return Opened{Sink: &h.out.sink, Assets: store}, nil }})
	t.Cleanup(func() { h.r.Close() })
	a := h.page("A", "http://a/", "L1")
	var mu sync.Mutex
	calls := map[string]int{}
	h.fc.more = func(session, method string, pm map[string]any, p *fakePage) (any, error, bool) {
		mu.Lock()
		defer mu.Unlock()
		calls[method]++
		switch method {
		case "Network.loadNetworkResource":
			return map[string]any{"resource": map[string]any{"success": true, "httpStatusCode": 200, "stream": "s"}}, nil, true
		case "IO.read":
			return map[string]any{"data": "x", "eof": true}, nil, true
		case "Network.getResponseBody":
			return map[string]any{"body": "x"}, nil, true
		}
		return nil, nil, false
	}
	h.r.Send(Live{Tab: rec.Tab{ID: a.frameID}, Session: "A"})
	h.settle()
	a.insert(a.body(), nil, a.el("img", "src", "/late.png"))
	h.events("A", a)
	waitFor(t, func() bool { h.settle(); return h.count("asset") == 1 }, "the referenced load")
	h.send("A", "Network.responseReceived", map[string]any{"requestId": "r", "type": "Image", "response": map[string]any{"url": "http://a/late.png"}})
	h.send("A", "Network.loadingFinished", map[string]any{"requestId": "r"})
	time.Sleep(50 * time.Millisecond)
	h.settle()
	mu.Lock()
	defer mu.Unlock()
	if calls["Network.getResponseBody"] != 0 || h.count("asset") != 1 {
		t.Fatalf("the late response fetched again: %v, %d assets", calls, h.count("asset"))
	}
}

// LowRes never divides by zero or reads past the image, whatever the DPR.
func TestLowResAnyDPR(t *testing.T) {
	var img bytes.Buffer
	png.Encode(&img, image.NewRGBA(image.Rect(0, 0, 101, 61)))
	for _, dpr := range []float64{0, 0.1, 0.25, 0.5, 1, 1.5, 2, 3} {
		out, w, h, err := LowRes(img.Bytes(), dpr)
		if err != nil || len(out) == 0 || w < 1 || h < 1 || w > 101 || h > 61 {
			t.Errorf("dpr %v: %dx%d %v", dpr, w, h, err)
		}
	}
}

// A frame is at its capture's midpoint even when records arrive while the
// layout is captured after it: only the layout record takes the stream's
// later time (#472, where the clamp put a busy page's frames up to ~90ms
// after their pixels).
func TestRecorderFrameKeepsItsCaptureTime(t *testing.T) {
	var ms atomic.Int64
	base := time.Unix(1000, 0)
	now := func() time.Time { return base.Add(time.Duration(ms.Load()) * time.Millisecond) }
	frames := &frameRecorder{}
	h := &harness{t: t, fc: newFakeChrome(), out: &memOut{}}
	h.r = Start(Options{ID: "t", Call: h.fc, ShotEvery: time.Hour, Now: now,
		Out: func() (Opened, error) { return Opened{Sink: &h.out.sink, Frames: frames}, nil }})
	t.Cleanup(func() { h.r.Close() })
	a := h.page("A", "http://a/", "L1")
	var img bytes.Buffer
	png.Encode(&img, image.NewRGBA(image.Rect(0, 0, 100, 80)))
	h.fc.more = func(session, method string, pm map[string]any, p *fakePage) (any, error, bool) {
		switch method {
		case "Page.captureScreenshot":
			// The recording opened at 0 and the call runs 0..120:
			// midpoint 60.
			defer ms.Store(120)
			return map[string]any{"data": base64.StdEncoding.EncodeToString(img.Bytes())}, nil, true
		case "DOMSnapshot.captureSnapshot":
			// The page changes while the layout is captured.
			ms.Store(300)
			a.insert(a.body(), nil, a.el("p"))
			h.events("A", a)
			return map[string]any{"strings": []string{}, "documents": []any{}}, nil, true
		}
		return nil, nil, false
	}
	h.confirmUI("A", a.frameID, 1)
	h.r.Send(Live{Tab: rec.Tab{ID: a.frameID}, Session: "A"})
	waitFor(t, func() bool { h.settle(); return h.count("layout") == 1 }, "a screenshot's layout")
	frames.mu.Lock()
	f := frames.frames[0]
	frames.mu.Unlock()
	if f.T != 60 {
		t.Errorf("frame at %d, want the capture's midpoint 60", f.T)
	}
	if l := h.last("layout"); l.Time() != 300 {
		t.Errorf("layout at %d, want the stream's 300", l.Time())
	}
	if h.count("ins") != 1 {
		t.Fatalf("the change during the capture is missing: %s", h.kinds())
	}
	h.mustValidate()
}

// A navigation or tab switch discards the old document's pending capture;
// the next capture belongs to the new document. A same-document resume may
// retain the capture, with its time clamped to the resumed segment (#472).
func TestRecorderFrameNeverBeforeItsSegment(t *testing.T) {
	for _, c := range []struct {
		name, want string
		straddle   func(h *harness, a *fakePage)
	}{
		{"navigate", "head seg:live doc seg:navigate doc", func(h *harness, a *fakePage) {
			h.fc.mu.Lock()
			h.fc.loader["A"] = "L2"
			h.fc.pages["A"] = newFakePage("http://a/next")
			h.fc.pages["A"].frameID = a.frameID
			h.fc.mu.Unlock()
			h.confirmUI("A", a.frameID, 2)
			h.r.Send(Event{Session: "A", Method: "DOM.documentUpdated", Seq: h.fc.next(), Params: raw(map[string]any{})})
		}},
		{"resume", "head seg:live doc stop:pause seg:resume doc", func(h *harness, a *fakePage) {
			h.r.Send(Pause{})
			h.r.Send(Live{Tab: rec.Tab{ID: a.frameID}, Session: "A"})
		}},
		{"tab", "head seg:live doc seg:tab doc", func(h *harness, a *fakePage) {
			h.r.Send(Live{Tab: rec.Tab{ID: "T-B"}, Session: "B"})
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			var ms atomic.Int64
			base := time.Unix(1000, 0)
			now := func() time.Time { return base.Add(time.Duration(ms.Load()) * time.Millisecond) }
			frames := &frameRecorder{}
			h := &harness{t: t, fc: newFakeChrome(), out: &memOut{}}
			h.r = Start(Options{ID: "t", Call: h.fc, ShotEvery: time.Hour, Now: now,
				Out: func() (Opened, error) { return Opened{Sink: &h.out.sink, Frames: frames}, nil }})
			t.Cleanup(func() { h.r.Close() })
			a := h.page("A", "http://a/", "L1")
			h.page("B", "http://b/", "L9")
			var img bytes.Buffer
			png.Encode(&img, image.NewRGBA(image.Rect(0, 0, 100, 80)))
			var replacement bytes.Buffer
			png.Encode(&replacement, image.NewRGBA(image.Rect(0, 0, 200, 160)))
			var straddled atomic.Bool
			h.fc.more = func(session, method string, pm map[string]any, p *fakePage) (any, error, bool) {
				switch method {
				case "Page.captureScreenshot":
					if !straddled.Swap(true) {
						// The call runs 0..200 (midpoint 100); the new
						// segment starts at 200, inside it.
						ms.Store(200)
						c.straddle(h, a)
						h.settle() // process replacement before the old pixels return
						return map[string]any{"data": base64.StdEncoding.EncodeToString(img.Bytes())}, nil, true
					}
					return map[string]any{"data": base64.StdEncoding.EncodeToString(replacement.Bytes())}, nil, true
				case "DOMSnapshot.captureSnapshot":
					return map[string]any{"strings": []string{}, "documents": []any{}}, nil, true
				}
				return nil, nil, false
			}
			h.confirmUI("A", a.frameID, 1)
			h.confirmUI("B", "T-B", 3)
			h.r.Send(Live{Tab: rec.Tab{ID: a.frameID}, Session: "A"})
			waitFor(t, func() bool { h.settle(); return h.count("layout") >= 1 }, "a screenshot's layout")
			if got := h.kinds(); !strings.HasPrefix(got, c.want) {
				t.Fatalf("records %s, want them to start %s", got, c.want)
			}
			frames.mu.Lock()
			f := frames.frames[0]
			frames.mu.Unlock()
			if seg := h.last("seg"); f.T != seg.Time() || f.T != 200 {
				t.Errorf("frame at %d, want its segment's start %d (not the straddling call's midpoint 100)", f.T, seg.Time())
			}
			wantWidth := 100
			if c.name == "resume" {
				wantWidth = 50
			}
			if f.W != wantWidth {
				t.Fatalf("frame width %d, want %d: wrong document capture retained", f.W, wantWidth)
			}
			h.mustValidate()
		})
	}
}
