package recorder

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast/assets"
	rec "github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// Regression tests for the M2 review's findings: each fails when its fix is
// reverted.

// BR-19/BR-14: a same-document re-snapshot of a page larger than the
// graveyard keeps every live node's id (the Kids it emits would otherwise
// be refused), and removals keep the id map bounded.
func TestGraveyardKeepsLiveIDsAndBoundsTheMap(t *testing.T) {
	old := graveSize
	graveSize = 50
	t.Cleanup(func() { graveSize = old })
	p := newFakePage("http://x/")
	var divs []*fnode
	for i := 0; i < 80; i++ {
		d := p.el("div")
		p.insert(p.body(), nil, d)
		divs = append(divs, d)
	}
	p.take()
	m := start(t, p)
	ids := map[int]rec.NodeID{}
	for _, d := range divs {
		ids[d.backend], _ = m.IDOf(sess, d.backend)
	}
	for round := 0; round < 3; round++ {
		m.Resnapshot(int64(round), sess, p.snapshot())
		for _, d := range divs {
			if got, ok := m.IDOf(sess, d.backend); !ok || got != ids[d.backend] {
				t.Fatalf("round %d: div %d id %d -> %d", round, d.backend, ids[d.backend], got)
			}
		}
	}
	// setChildNodes with the same children keeps them too.
	kids := []any{}
	for _, k := range p.body().kids {
		kids = append(kids, p.cdp(k))
	}
	feed(t, m, 5, []fevent{{"DOM.setChildNodes", raw(map[string]any{"parentId": p.body().nodeID, "nodes": kids})}})
	if got, _ := m.IDOf(sess, divs[0].backend); got != ids[divs[0].backend] {
		t.Fatal("setChildNodes changed a kept child's id")
	}
	if len(m.Dropped) != 0 {
		t.Fatalf("dropped %v: %v", m.Dropped, m.Examples)
	}
	for _, d := range divs {
		p.remove(d)
	}
	feed(t, m, 6, p.take())
	if s := m.sessions[sess]; len(s.ids) > len(s.byBackend)+graveSize {
		t.Fatalf("id map %d entries for %d live nodes", len(s.ids), len(s.byBackend))
	}
}

// BR-12: an insert after a sibling the mirror doesn't have is a desync,
// never a guessed position.
func TestInsertAfterUnknownSiblingIsADesync(t *testing.T) {
	p := newFakePage("http://x/")
	m := start(t, p)
	m.take()
	err := m.Event(1, sess, "DOM.childNodeInserted", raw(map[string]any{"parentNodeId": p.body().nodeID, "previousNodeId": 9999,
		"node": map[string]any{"nodeId": 500, "backendNodeId": 500, "nodeType": 1, "localName": "p", "nodeName": "P"}}))
	if !errors.Is(err, errDesync) || len(m.take()) != 0 {
		t.Fatalf("err %v", err)
	}
}

// BR-6: a report from a frame whose context is unknown names nothing (it
// isn't applied to the top document), and an overflowing burst of DOM
// events never drops contexts or observer reports.
func TestUnknownFrameIsStaleAndOverflowKeepsReports(t *testing.T) {
	h := &harness{t: t, fc: newFakeChrome(), out: &memOut{}}
	gc := &gatedCaller{fakeChrome: h.fc, gate: make(chan struct{})}
	close(gc.gate)
	h.r = Start(Options{ID: "t", Call: gc, ShotEvery: -1, MaxQueue: 3, Out: func() (Opened, error) { return Opened{Sink: &h.out.sink}, nil }})
	t.Cleanup(func() { h.r.Close() })
	p := h.page("A", "http://a/", "L1")
	btn := p.el("button")
	p.insert(p.body(), nil, btn)
	p.take()
	h.r.Send(Live{Tab: rec.Tab{ID: p.frameID}, Session: "A"})
	h.settle()
	click := func(ctx int) {
		payload, _ := json.Marshal(map[string]any{"t": "obs", "items": []any{map[string]any{"k": "click", "p": []int{1, 2, 0}}}})
		h.send("A", "Runtime.bindingCalled", map[string]any{"name": "__couchcast", "payload": string(payload), "executionContextId": ctx})
	}
	click(99) // no such context
	h.settle()
	if n := h.count("click"); n != 0 {
		t.Fatalf("a report from an unknown frame was applied (%d clicks)", n)
	}
	gc.hold()
	h.send("A", "DOM.inlineStyleInvalidated", map[string]any{"nodeIds": []int{p.body().nodeID}})
	gc.waitEntered(t)
	for i := 0; i < 10; i++ {
		p.insert(p.body(), nil, p.el("i"))
	}
	h.events("A", p) // over the bound
	h.ctxCreated("A", p.frameID, 7)
	click(7)
	gc.release()
	h.settle()
	st, err := h.r.Stats(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if st.Lost == 0 {
		t.Fatal("the burst should have overflowed")
	}
	if n := h.count("click"); n != 1 {
		t.Fatalf("the context or the report was dropped with the DOM events: %d clicks\n%s", n, h.kinds())
	}
	h.mustValidate()
}

// BR-4: what a recording lost is in Close's error.
func TestCloseReportsLosses(t *testing.T) {
	h := &harness{t: t, fc: newFakeChrome(), out: &memOut{}}
	h.r = Start(Options{ID: "rec1", Call: h.fc, ShotEvery: -1,
		Out: func() (Opened, error) { return Opened{Sink: &h.out.sink, Assets: failingStore{assets.NewMemStore()}}, nil }})
	h.page("A", "http://a/", "L1")
	h.r.Send(Live{Tab: rec.Tab{ID: "T-A"}, Session: "A"})
	h.send("A", "Network.responseReceived", map[string]any{"requestId": "r", "type": "Image", "response": map[string]any{"url": "http://a/x.png"}})
	h.send("A", "Network.loadingFinished", map[string]any{"requestId": "r"})
	waitFor(t, func() bool { st, _ := h.r.Stats(context.Background()); return st.Assets == 1 }, "the failed asset")
	err := h.r.Close()
	if err == nil || !strings.Contains(err.Error(), "rec1 lost: 1 assets not captured") {
		t.Fatalf("Close: %v", err)
	}
}

type failingStore struct{ *assets.MemStore }

func (failingStore) Put(context.Context, assets.Capture) (*rec.Asset, error) {
	return nil, errors.New("disk full")
}

// BR-13: switching to a tab that can't be read ends the segment that was
// open; nothing is left live without a document.
func TestUnreadableTabEndsTheSegment(t *testing.T) {
	h := newHarness(t)
	h.page("A", "http://a/", "L1")
	h.r.Send(Live{Tab: rec.Tab{ID: "T-A"}, Session: "A"})
	h.r.Send(Live{Tab: rec.Tab{ID: "T-B"}, Session: "B"}) // B has no page: getDocument fails
	h.settle()
	h.r.Close()
	if got := h.kinds(); got != "head seg:live doc stop:pause" {
		t.Fatalf("records %s", got)
	}
	h.mustValidate()
}

// BR-17: a new segment emits an out-of-process frame's document once.
func TestOneFrameDocPerSegment(t *testing.T) {
	h := newHarness(t)
	a := h.page("A", "http://a/", "L1")
	ifr := a.el("iframe")
	ifr.frame = "FR"
	a.insert(a.body(), nil, ifr)
	a.take()
	f := h.page("F", "http://other/", "LF")
	f.frameID = "FR"
	h.r.Send(FrameAttached{Session: "F", Parent: "A", FrameID: "FR"})
	h.r.Send(Live{Tab: rec.Tab{ID: "T-A"}, Session: "A"})
	h.r.Send(Pause{})
	h.r.Send(Live{Tab: rec.Tab{ID: "T-A"}, Session: "A"})
	h.settle()
	frameDocs := 0
	for _, r := range h.out.sink.Records() {
		if d, ok := r.(*rec.Doc); ok && d.Host != 0 {
			frameDocs++
		}
	}
	if frameDocs != 2 {
		t.Fatalf("%d frame docs over two segments, want 2\n%s", frameDocs, h.kinds())
	}
	h.mustValidate()
}

// BR-7/BR-11: click and pointer positions in frames map into the top page:
// an out-of-process frame through its owner's box in the parent session, a
// nested in-process frame through its owner's box in its session's viewport.
func TestFrameOffsets(t *testing.T) {
	h := newHarness(t)
	a := h.page("A", "http://a/", "L1")
	// An in-process iframe, inside another in-process iframe.
	outer := a.el("iframe")
	outer.frame = "OUT"
	outer.content = a.node(9, "", "")
	outer.content.frame = "OUT"
	inner := a.el("iframe")
	inner.frame = "IN"
	inner.content = a.node(9, "", "")
	inner.content.frame = "IN"
	ib := a.el("b")
	inner.content.kids = []*fnode{ib}
	ib.parent = inner.content
	outer.content.kids = []*fnode{inner}
	inner.parent = outer.content
	oop := a.el("iframe")
	oop.frame = "FR"
	a.insert(a.body(), nil, outer)
	a.insert(a.body(), outer, oop)
	a.take()
	f := h.page("F", "http://other/", "LF")
	f.frameID = "FR"
	fb := f.el("button")
	f.insert(f.body(), nil, fb)
	f.take()
	boxes := map[string][2]float64{"A/" + fmt.Sprint(inner.backend): {300, 400}, "A/" + fmt.Sprint(oop.backend): {100, 200}}
	owners := map[string]int{"A/IN": inner.backend, "A/FR": oop.backend}
	h.fc.more = func(session, method string, pm map[string]any, p *fakePage) (any, error, bool) {
		switch method {
		case "DOM.getFrameOwner":
			b, ok := owners[session+"/"+fmt.Sprint(pm["frameId"])]
			if !ok {
				return nil, fmt.Errorf("no owner"), true
			}
			return map[string]any{"backendNodeId": b}, nil, true
		case "DOM.getBoxModel":
			xy, ok := boxes[session+"/"+fmt.Sprint(pm["backendNodeId"])]
			if !ok {
				return nil, fmt.Errorf("no box"), true
			}
			return map[string]any{"model": map[string]any{"content": []float64{xy[0], xy[1], 0, 0, 0, 0, 0, 0}}}, nil, true
		}
		return nil, nil, false
	}
	h.ctxCreated("A", a.frameID, 1)
	h.ctxCreated("A", "IN", 2)
	h.ctxCreated("F", "FR", 3)
	h.r.Send(FrameAttached{Session: "F", Parent: "A", FrameID: "FR"})
	h.r.Send(Live{Tab: rec.Tab{ID: a.frameID}, Session: "A"})
	h.settle()
	// The page is scrolled by 50: a top-page point is a viewport point + 50.
	report := func(session string, ctx int, items ...any) {
		payload, _ := json.Marshal(map[string]any{"t": "obs", "items": items})
		h.send(session, "Runtime.bindingCalled", map[string]any{"name": "__couchcast", "payload": string(payload), "executionContextId": ctx})
	}
	report("A", 1, map[string]any{"k": "scroll", "p": []int{}, "x": 0, "y": 50})
	report("A", 2, map[string]any{"k": "click", "p": []int{0}, "x": 5, "y": 6})
	report("F", 3, map[string]any{"k": "click", "p": []int{1, 2, 0}, "x": 7, "y": 8}, map[string]any{"k": "ptr", "x": 1, "y": 1})
	h.settle()
	var clicks []*rec.Click
	var ptr *rec.Ptr
	for _, r := range h.out.sink.Records() {
		switch v := r.(type) {
		case *rec.Click:
			clicks = append(clicks, v)
		case *rec.Ptr:
			ptr = v
		}
	}
	if len(clicks) != 2 || ptr == nil {
		t.Fatalf("clicks %v ptr %v\n%s", clicks, ptr, h.kinds())
	}
	if c := clicks[0]; c.ID != h.id("A", ib.backend) || c.X != 305 || c.Y != 456 {
		t.Errorf("nested in-process click %+v, want (305, 456)", *c)
	}
	if c := clicks[1]; c.ID != h.id("F", fb.backend) || c.X != 107 || c.Y != 258 {
		t.Errorf("out-of-process click %+v, want (107, 258)", *c)
	}
	if ptr.X != 101 || ptr.Y != 251 {
		t.Errorf("ptr %+v, want (101, 251)", *ptr)
	}
	h.mustValidate()
}
