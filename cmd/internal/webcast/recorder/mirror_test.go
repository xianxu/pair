package recorder

import (
	"math/rand"
	"strings"
	"testing"

	rec "github.com/xianxu/pair/cmd/internal/webcast/recording"
)

const sess = "S1"

// start is a mirror with head, a segment and the fake page's snapshot.
func start(t *testing.T, p *fakePage) *mirror {
	t.Helper()
	m := newMirror()
	m.Emit(0, &rec.Head{Format: rec.Format, V: rec.Version, ID: "test", Started: "2026-10-12T00:00:00Z"})
	m.Session(sess, p.frameID, true)
	seg := &rec.Seg{Seg: 1, Why: rec.SegLive, Tab: rec.Tab{ID: "T", URL: p.url}, VP: rec.Viewport{W: 800, H: 600, DPR: 1}}
	if err := m.Seg(0, seg, sess, p.snapshot(), "L1"); err != nil {
		t.Fatal(err)
	}
	return m
}

func feed(t *testing.T, m *mirror, at int64, evs []fevent) {
	t.Helper()
	for _, e := range evs {
		if err := m.Event(at, sess, e.method, e.params); err != nil {
			t.Fatalf("%s: %v", e.method, err)
		}
	}
}

// The mirror's output, reduced, is the fake page's DOM, after every step of
// random mutation sequences (moves, shadow roots, password games included);
// the output validates; no planted password reaches it. A reverted mirror
// rule fails this.
func TestMirrorMatchesThePageUnderRandomMutations(t *testing.T) {
	for seed := int64(1); seed <= 40; seed++ {
		r := rand.New(rand.NewSource(seed))
		p := newFakePage("http://x/")
		m := start(t, p)
		var all []rec.Record
		all = append(all, m.take()...)
		for step := 0; step < 150; step++ {
			p.mutate(r)
			feed(t, m, int64(step), p.take())
			all = append(all, m.take()...)
			if err := sameTree(p.doc, m.State().Top, "doc"); err != nil {
				t.Fatalf("seed %d step %d: %v", seed, step, err)
			}
		}
		if len(m.Dropped) != 0 || m.Desync != 0 {
			t.Fatalf("seed %d: dropped %v desync %d: %v", seed, m.Dropped, m.Desync, m.Examples)
		}
		fs, _ := rec.Validate(files(t, all))
		if errs := rec.Errors(fs); len(errs) != 0 {
			t.Fatalf("seed %d: %v", seed, errs)
		}
		for name, b := range files(t, all) {
			if strings.Contains(string(b), "s3cret-planted") {
				t.Fatalf("seed %d: a password value reached %s", seed, name)
			}
		}
	}
}

// A moved node keeps its id; a re-snapshot of the same document keeps ids;
// a new document gets new ones, never reusing old.
func TestMirrorIDsAcrossMovesAndSnapshots(t *testing.T) {
	p := newFakePage("http://x/")
	div, span := p.el("div"), p.el("span")
	p.insert(p.body(), nil, div)
	p.insert(p.body(), div, span)
	p.take()
	m := start(t, p)
	id, ok := m.IDOf(sess, div.backend)
	if !ok {
		t.Fatal("no id for div")
	}
	p.remove(div)
	p.insert(p.body(), p.body().kids[0], div)
	feed(t, m, 1, p.take())
	if again, _ := m.IDOf(sess, div.backend); again != id {
		t.Fatalf("a move changed the id: %d -> %d", id, again)
	}
	if s, b, ok := m.NodeOf(id); !ok || s != sess || b != div.backend {
		t.Fatalf("NodeOf(%d) = %s %d %v", id, s, b, ok)
	}
	m.Resnapshot(2, sess, p.snapshot())
	if again, _ := m.IDOf(sess, div.backend); again != id {
		t.Fatal("a re-snapshot of the same document changed the id")
	}
	before := m.next
	// The new document's backend ids repeat the old one's (a cross-process
	// navigation restarts them): its key says it is new.
	q := newFakePage("http://x/2")
	if m.SameDocument(sess, "L2") || !m.SameDocument(sess, "L1") {
		t.Fatal("SameDocument goes by the key")
	}
	m.Emit(3, &rec.Stop{Why: rec.StopPause})
	if err := m.Seg(3, &rec.Seg{Seg: 2, Why: rec.SegNavigate, Tab: rec.Tab{ID: "T"}, VP: rec.Viewport{W: 1, H: 1, DPR: 1}}, sess, q.snapshot(), "L2"); err != nil {
		t.Fatal(err)
	}
	if top := m.State().Top.ID; top <= before {
		t.Fatalf("a new document reused id %d (counter was %d)", top, before)
	}
	if id2, ok := m.IDOf(sess, div.backend); ok && id2 == id {
		t.Fatal("the old document's ids must be dropped")
	}
}

// An event naming a node the mirror doesn't have is a desync, never applied.
func TestMirrorDesync(t *testing.T) {
	p := newFakePage("http://x/")
	m := start(t, p)
	m.take()
	err := m.Event(1, sess, "DOM.attributeModified", raw(map[string]any{"nodeId": 9999, "name": "a", "value": "b"}))
	if err != errDesync || m.Desync != 1 || len(m.take()) != 0 {
		t.Fatalf("err %v desync %d", err, m.Desync)
	}
}

// The cast's own control and user-agent shadow trees are not page content.
func TestMirrorLeavesOutTheControlAndUserAgentTrees(t *testing.T) {
	p := newFakePage("http://x/")
	ctl := p.el("couchcast-control", controlAttr, "")
	ctl.kids = []*fnode{p.txt("Go live")}
	ctl.kids[0].parent = ctl
	p.insert(p.doc.kids[1], p.body(), ctl) // appended to <html>
	in := p.el("input")
	p.insert(p.body(), nil, in)
	ua := p.node(11, "", "")
	ua.shadowType, ua.host = "user-agent", in
	in.shadow = ua
	p.take()
	m := start(t, p)
	all := m.take()
	// An insert after the control, and a change inside it, are both quiet.
	after := p.el("div")
	p.insert(p.doc.kids[1], ctl, after)
	p.setText(ctl.kids[0], "Pause")
	feed(t, m, 1, p.take())
	all = append(all, m.take()...)
	body := string(files(t, all)[rec.FileEvents])
	if strings.Contains(body, "couchcast") || strings.Contains(body, "Go live") || strings.Contains(body, "user-agent") {
		t.Fatalf("control or UA tree recorded:\n%s", body)
	}
	html := m.State().Node(m.State().Top.Kids[1].ID)
	if last := html.Kids[len(html.Kids)-1]; last.N != "div" {
		t.Fatalf("the div after the control is at the end: %+v", last)
	}
}

// Observer reports resolve by path against the mirror: a click, a form
// value (a password's only as secret), scroll on the document, hover, ptr.
func TestObserveResolvesPaths(t *testing.T) {
	p := newFakePage("http://x/")
	btn := p.el("button")
	pw := p.el("input", "type", "password")
	p.insert(p.body(), nil, btn)
	p.insert(p.body(), btn, pw)
	host := p.el("div")
	p.insert(p.body(), pw, host)
	sr := p.attachShadow(host, "open")
	inner := p.el("span")
	p.insert(sr, nil, inner)
	p.take()
	m := start(t, p)
	m.take()
	path := func(ix ...int) *[]int { return &ix }
	must := func(it Item) {
		t.Helper()
		if err := m.Observe(5, sess, p.frameID, it, [2]float64{0, 100}, ""); err != nil {
			t.Fatalf("%s: %v", it.K, err)
		}
	}
	// body is doc/1(html)/2(body); btn 0, pw 1, host 2
	must(Item{K: "click", P: path(1, 2, 0), X: 3, Y: 4})
	must(Item{K: "input", P: path(1, 2, 1), V: rec.Str("hunter2")})
	must(Item{K: "hover", P: path(1, 2, 2, shadowStep, 0)})
	must(Item{K: "hover", P: path(1, 2, 2, shadowStep, 0)}) // unchanged: no record
	must(Item{K: "ptr", X: 1, Y: 2})
	must(Item{K: "scroll", P: path(1), Y: 300})
	recs := m.take()
	kinds := []string{}
	for _, r := range recs {
		kinds = append(kinds, r.Kind())
	}
	if strings.Join(kinds, " ") != "click input hover ptr scroll" {
		t.Fatalf("kinds %v", kinds)
	}
	btnID, _ := m.IDOf(sess, btn.backend)
	innerID, _ := m.IDOf(sess, inner.backend)
	if c := recs[0].(*rec.Click); c.ID != btnID || c.Y != 104 {
		t.Errorf("click %+v", c)
	}
	if in := recs[1].(*rec.Input); !in.Secret || in.V != nil {
		t.Errorf("password input %+v", in)
	}
	if h := recs[2].(*rec.Hover); h.ID != innerID {
		t.Errorf("hover %d, want %d", h.ID, innerID)
	}
	if s := recs[4].(*rec.Scroll); s.ID != m.State().Top.ID {
		t.Errorf("the html element's scroll is the document's: %+v", s)
	}
	// Stale or malformed paths name nothing.
	for _, bad := range [][]int{{9}, {1, 2, 0, 0, 0}, {-2}, {1, 2, 0, shadowStep}} {
		if err := m.Observe(6, sess, p.frameID, Item{K: "click", P: &bad}, [2]float64{}, ""); err != errStale {
			t.Errorf("path %v: %v", bad, err)
		}
	}
}

// FuzzResolve: no path panics, and a path resolves to nothing or to the
// node the fake reaches by the same steps.
func FuzzResolve(f *testing.F) {
	f.Add([]byte{1, 2, 0})
	f.Add([]byte{1, 2, 255, 0})
	f.Fuzz(func(t *testing.T, steps []byte) {
		p := newFakePage("http://x/")
		host := p.el("div")
		p.insert(p.body(), nil, host)
		sr := p.attachShadow(host, "open")
		p.insert(sr, nil, p.el("b"))
		root := p.snapshot()
		var path []int
		for _, s := range steps {
			path = append(path, int(int8(s))) // negative steps too
		}
		got := resolve(root, path)
		want := p.doc
		for _, s := range path {
			if want == nil {
				break
			}
			switch {
			case s == shadowStep && want.shadow != nil && want.shadow.shadowType == "open":
				want = want.shadow
			case s >= 0 && s < len(want.kids):
				want = want.kids[s]
			default:
				want = nil
			}
		}
		if (got == nil) != (want == nil) || (got != nil && got.BackendNodeID != want.backend) {
			t.Fatalf("path %v: got %+v want %+v", path, got, want)
		}
	})
}

// #460 saw astro-dev-toolbar's shadow root recorded empty: CDP pushed the
// root with a child count but no children. The mirror asks for them, and
// their arrival (setChildNodes) fills the root.
func TestMirrorExpandsNodesSentWithoutChildren(t *testing.T) {
	p := newFakePage("http://x/")
	host := p.el("astro-dev-toolbar")
	p.insert(p.body(), nil, host)
	p.take()
	m := start(t, p)
	m.take()
	feed(t, m, 1, []fevent{{"DOM.shadowRootPushed", raw(map[string]any{"hostId": host.nodeID, "root": map[string]any{
		"nodeId": 500, "backendNodeId": 500, "nodeType": 11, "nodeName": "#document-fragment", "shadowRootType": "open", "childNodeCount": 2}})}})
	if len(m.Expand) != 1 || m.Expand[0].NodeID != 500 || m.Expand[0].Session != sess {
		t.Fatalf("expand %v", m.Expand)
	}
	feed(t, m, 2, []fevent{{"DOM.setChildNodes", raw(map[string]any{"parentId": 500, "nodes": []any{
		map[string]any{"nodeId": 501, "backendNodeId": 501, "nodeType": 1, "localName": "div", "nodeName": "DIV"},
		map[string]any{"nodeId": 502, "backendNodeId": 502, "nodeType": 3, "nodeName": "#text", "nodeValue": "x"}}})}})
	hostID, _ := m.IDOf(sess, host.backend)
	sr := m.State().Node(hostID).Shadow
	if sr == nil || len(sr.Kids) != 2 || sr.Kids[0].N != "div" {
		t.Fatalf("shadow root %+v", sr)
	}
}

// pair#513: a node inserted empty is sent without a request for its
// children, so when a script fills it later (asciinema-player into the
// blog's cast embed, measured) CDP sends only childNodeCountUpdated, never
// childNodeInserted. The mirror asks for the children, and their arrival
// fills the node, so the recording holds the player.
func TestMirrorExpandsANodeFilledAfterItsInsert(t *testing.T) {
	p := newFakePage("http://x/")
	p.take()
	m := start(t, p)
	m.take()
	feed(t, m, 1, []fevent{{"DOM.childNodeInserted", raw(map[string]any{"parentNodeId": p.body().nodeID, "previousNodeId": 0,
		"node": map[string]any{"nodeId": 600, "backendNodeId": 600, "nodeType": 1, "localName": "div", "nodeName": "DIV", "childNodeCount": 0}})}})
	m.take()
	if len(m.Expand) != 0 {
		t.Fatalf("an empty insert asked for children: %v", m.Expand)
	}
	feed(t, m, 2, []fevent{{"DOM.childNodeCountUpdated", raw(map[string]any{"nodeId": 600, "childNodeCount": 1})}})
	// A script inserting several children sends a count per insert: one
	// request covers them.
	feed(t, m, 2, []fevent{{"DOM.childNodeCountUpdated", raw(map[string]any{"nodeId": 600, "childNodeCount": 2})}})
	if len(m.Expand) != 1 || m.Expand[0].NodeID != 600 || m.Expand[0].Session != sess {
		t.Fatalf("expand %v", m.Expand)
	}
	m.Expand = nil
	feed(t, m, 3, []fevent{{"DOM.setChildNodes", raw(map[string]any{"parentId": 600, "nodes": []any{
		map[string]any{"nodeId": 601, "backendNodeId": 601, "nodeType": 1, "localName": "div", "nodeName": "DIV",
			"attributes": []string{"class", "ap-wrapper"}, "childNodeCount": 0}}})}})
	id, _ := m.IDOf(sess, 600)
	n := m.State().Node(id)
	if n == nil || len(n.Kids) != 1 || n.Kids[0].A["class"] != "ap-wrapper" {
		t.Fatalf("the filled node %+v", n)
	}
	// A count that matches the children the mirror holds asks nothing.
	feed(t, m, 4, []fevent{{"DOM.childNodeCountUpdated", raw(map[string]any{"nodeId": 600, "childNodeCount": 1})}})
	if len(m.Expand) != 0 {
		t.Fatalf("a node whose children the mirror holds asked again: %v", m.Expand)
	}
}

// A report that follows a driver's injected input carries by (#467); one
// naming a viewer who doesn't drive is refused by the reducer and counted.
func TestObserveAttributesTheDriver(t *testing.T) {
	p := newFakePage("http://x/")
	btn := p.el("button")
	p.insert(p.body(), nil, btn)
	p.take()
	m := start(t, p)
	m.take()
	ana := rec.Author{ID: "v1", Name: "Ana"}
	for _, op := range []string{rec.CtlRequest, rec.CtlGrant} {
		next, err := rec.NextControl(m.State().Control, op, ana)
		if err != nil {
			t.Fatal(err)
		}
		m.Emit(5, &rec.Ctl{Op: op, Who: ana, Control: next})
	}
	m.take()
	click := Item{K: "click", P: &[]int{1, 2, 0}, X: 3, Y: 4}
	if err := m.Observe(6, sess, p.frameID, click, [2]float64{}, "v1"); err != nil {
		t.Fatal(err)
	}
	if err := m.Observe(7, sess, p.frameID, Item{K: "focus", P: &[]int{1, 2, 0}}, [2]float64{}, ""); err != nil {
		t.Fatal(err)
	}
	m.Observe(8, sess, p.frameID, click, [2]float64{}, "v2")
	recs := m.take()
	if len(recs) != 2 {
		t.Fatalf("records %d, want the driver's click and the operator's focus", len(recs))
	}
	if c := recs[0].(*rec.Click); c.By != "v1" {
		t.Errorf("the driver's click by %q", c.By)
	}
	if f := recs[1].(*rec.Focus); f.By != "" {
		t.Errorf("the operator's focus by %q", f.By)
	}
	if m.Dropped["click"] != 1 {
		t.Errorf("a non-driver's click: dropped %d, want 1", m.Dropped["click"])
	}
}

func TestMirrorSameLoadTabReturnKeepsOnlyTrustedBackendIDs(t *testing.T) {
	a := newFakePage("http://a/")
	kept, removed := a.el("button"), a.el("button")
	a.insert(a.body(), nil, kept)
	a.insert(a.body(), nil, removed)
	a.take()
	m := start(t, a)
	keptID, _ := m.IDOf(sess, kept.backend)
	removedID, _ := m.IDOf(sess, removed.backend)
	b := newFakePage("http://b/")
	m.Session("B", "T-B", true)
	if err := m.Seg(1, &rec.Seg{Seg: 2, Why: rec.SegTab, Tab: rec.Tab{ID: "T-B"}, VP: rec.Viewport{W: 800, H: 600, DPR: 1}}, "B", b.snapshot(), "LB"); err != nil {
		t.Fatal(err)
	}
	a.remove(removed)
	replacement := a.el("button")
	a.insert(a.body(), nil, replacement)
	a.take()
	m.Session(sess, a.frameID, true)
	if err := m.Seg(2, &rec.Seg{Seg: 3, Why: rec.SegTab, Tab: rec.Tab{ID: "T"}, VP: rec.Viewport{W: 800, H: 600, DPR: 1}}, sess, a.snapshot(), "L1"); err != nil {
		t.Fatal(err)
	}
	if got, _ := m.IDOf(sess, kept.backend); got != keptID {
		t.Fatalf("known backend lost stable identity: %d -> %d", keptID, got)
	}
	if _, ok := m.IDOf(sess, removed.backend); ok {
		t.Fatal("removed backend became live on return")
	}
	if got, _ := m.IDOf(sess, replacement.backend); got == removedID || got == keptID {
		t.Fatal("replacement borrowed an earlier node's identity")
	}
	if len(m.sessions[sess].grave) == 0 {
		t.Fatal("inactive removals bypassed bounded grave accounting")
	}
	before, _ := m.IDOf(sess, kept.backend)
	if err := m.Seg(3, &rec.Seg{Seg: 4, Why: rec.SegNavigate, Tab: rec.Tab{ID: "T"}, VP: rec.Viewport{W: 800, H: 600, DPR: 1}}, sess, a.snapshot(), "L2"); err != nil {
		t.Fatal(err)
	}
	if got, _ := m.IDOf(sess, kept.backend); got == before {
		t.Fatal("same backend in a different load retained identity")
	}
}
