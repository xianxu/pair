package webcast

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// The real viewer page, end to end in a headless Chrome at a tablet-sized
// window: served by Server from a Hub that a fake recorder feeds the blog
// fixture into, joined mid-cast. It draws the reducer's page, scaled to fit
// without relayout, shows the URL, the operator's hover and pointer, and its
// boxOf (what a viewer's note carries) agrees with the boxes the recorder
// took in the operator's tab.
//
//	WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run LiveViewer -v
func TestLiveViewerPage(t *testing.T) {
	page := headlessPage(t)
	page.call(t, page.session, "Emulation.setDeviceMetricsOverride", map[string]any{"width": 1024, "height": 700, "deviceScaleFactor": 1, "mobile": false}, nil)
	rec, err := recording.ReadDir(recording.Fixture("blog"))
	if err != nil {
		t.Fatal(err)
	}
	const at = 10041 // the last layout before the stop; the scroll has been still since 8536
	var upto []recording.Record
	for _, r := range rec.Records {
		if r.Time() <= at {
			upto = append(upto, r)
		}
	}
	hub := recording.NewHub(4096)
	half := len(upto) / 2
	for _, r := range upto[:half] {
		hub.Put(r)
	}
	srv := viewerServer(t, rec, hub)
	t.Cleanup(func() { hub.Close() }) // before the server's Close: it ends the open stream

	page.call(t, page.session, "Page.navigate", map[string]any{"url": srv.URL + "/tok/?viewer=v1"}, nil)
	for _, r := range upto[half:] { // the rest arrives live, after the join
		hub.Put(r)
	}
	st, err := recording.Reduce(rec.Records, at)
	if err != nil {
		t.Fatal(err)
	}
	want := string(st.Canonical())
	waitEval(t, page, "the drawn page", fmt.Sprintf(`(async () => {
		const m = await import('./model.js');
		return window.couchViewer ? m.goJSON(window.couchViewer.canonical()) : '';
	})()`), want)

	// Scaled to fit 1024 wide: 1280×800 at 0.8, never relaid out.
	geo := page.evalString(t, `(() => {
		const f = document.getElementById('page'), b = document.getElementById('box').getBoundingClientRect();
		return JSON.stringify([f.style.width, f.style.height, Math.round(b.width), f.contentWindow.innerWidth]);
	})()`)
	if geo != `["1280px","800px",1024,1280]` {
		t.Errorf("frame geometry %s: want laid out at 1280×800, shown 1024 wide", geo)
	}
	if url := page.evalString(t, `document.getElementById('url').textContent`); url != st.URL {
		t.Errorf("URL shown %q, want %q", url, st.URL)
	}

	// boxOf from the viewer's own layout equals the operator tab's boxes.
	var boxes map[string][4]float64
	if err := json.Unmarshal(page.eval(t, `(async () => {
		await couchViewer.ready();
		// The tab's boxes were taken with its entry animations done; the
		// replay started them later. Finish them (an infinite one can't).
		const d = document.getElementById('page').contentDocument;
		for (const a of d.getAnimations()) { try { a.finish(); } catch {} }
		await new Promise((ok) => requestAnimationFrame(() => requestAnimationFrame(ok)));
		const out = {};
		for (const id of `+idsJSON(st.Boxes)+`) { const b = couchViewer.boxOf(id); if (b) out[id] = b; }
		return out;
	})()`), &boxes); err != nil {
		t.Fatal(err)
	}
	match, sized := 0, 0
	for id, b := range st.Boxes {
		if b[2] == 0 || b[3] == 0 || inSVGDefinition(st.Node(id)) {
			// An empty box has no place to agree on; the tab reports a
			// <symbol>'s content at its <use>, the DOM gives it none.
			continue
		}
		sized++
		got, ok := boxes[fmt.Sprint(id)]
		tol := 1.0
		if n := st.Node(id); n.NS == "svg" && n.N != "svg" {
			tol = 2 // the tab's box of an SVG shape includes half its stroke
		}
		if ok && near(got, b, tol) {
			match++
		} else if testing.Verbose() {
			t.Logf("node %d: viewer box %v, tab box %v", id, got, b)
		}
	}
	share := float64(match) / float64(sized)
	t.Logf("boxOf: %d of %d recorded boxes agree (%.1f%%)", match, sized, 100*share)
	if share < boxAgreement {
		t.Errorf("boxOf agrees with the tab on %.1f%% of boxes, want at least %.0f%%", 100*share, 100*boxAgreement)
	}

	// The operator's hover and pointer.
	target := hoverTarget(t, st)
	b := st.Boxes[target]
	hub.Put(decode(t, fmt.Sprintf(`{"t":%d,"k":"hover","id":%d}`, at+1, target)))
	hub.Put(decode(t, fmt.Sprintf(`{"t":%d,"k":"ptr","x":%g,"y":%g}`, at+1, b[0]+b[2]/2, b[1]+b[3]/2)))
	waitEval(t, page, "hover and pointer", fmt.Sprintf(`(() => {
		const d = document.getElementById('page').contentDocument, p = document.getElementById('pointer');
		const hovered = [...d.querySelectorAll('.__couch-hover')];
		if (!hovered.length || p.hidden) return 'none';
		const r = p.getBoundingClientRect();
		const [x, y] = couchViewer.toPage(r.left + r.width / 2, r.top + r.height / 2);
		const at = couchViewer.nodeAt(%g, %g);
		const deepest = JSON.stringify(couchViewer.boxOf(%d)) === JSON.stringify((() => { const e = hovered.at(-1).getBoundingClientRect(), w = d.defaultView; return [e.left + w.scrollX, e.top + w.scrollY, e.width, e.height]; })());
		return [hovered[0] === d.documentElement && deepest, Math.round(x) === %d && Math.round(y) === %d, at === %d].join(',');
	})()`, b[0]+b[2]/2, b[1]+b[3]/2, target, int(math.Round(b[0]+b[2]/2)), int(math.Round(b[1]+b[3]/2)), target), "true,true,true")
}

// boxAgreement: the share of the recorder's layout boxes the viewer's
// boxOf must reproduce within 1px (2px for SVG shapes). Measured on blog:
// 215 of 217. Left out: Astro's dev toolbar, whose shadow content the
// fixture never recorded (a recorder gap, #458), and one SVG shape whose
// stroke adds 2.02px.
const boxAgreement = 0.98

func viewerServer(t *testing.T, rec *recording.Recording, live recording.Live) *httptest.Server {
	t.Helper()
	viewer := NewServer("tok", NewFakeBrowser(), time.Second, live, nil)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The asset proxy (#459) is mounted at <cast>/assets/; the fixture
		// stands in for it.
		if hash, ok := strings.CutPrefix(r.URL.Path, "/tok/assets/"); ok {
			w.Header().Set("Content-Security-Policy", contentSecurityPolicy)
			serveFixtureAsset(w, r, rec, "", hash)
			return
		}
		viewer.ServeHTTP(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func waitEval(t *testing.T, p *chromePage, what, expr, want string) {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	var got string
	for time.Now().Before(deadline) {
		if got = p.evalString(t, expr); got == want {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("%s: got %.300q\nwant %.300q", what, got, want)
}

func idsJSON(boxes map[recording.NodeID]recording.Box) string {
	ids := make([]string, 0, len(boxes))
	for id := range boxes {
		ids = append(ids, fmt.Sprint(id))
	}
	return "[" + strings.Join(ids, ",") + "]"
}

func near(a, b [4]float64, tol float64) bool {
	for i := range a {
		if math.Abs(a[i]-b[i]) > tol {
			return false
		}
	}
	return true
}

// hoverTarget is a link in view with a box: the stand-in for what the
// operator's mouse is over.
func hoverTarget(t *testing.T, st *recording.State) recording.NodeID {
	t.Helper()
	sy := st.Scrolls[st.Top.ID][1]
	for id, b := range st.Boxes {
		if n := st.Node(id); n != nil && n.N == "a" && b[2] > 20 && b[3] > 10 && b[1] > sy+100 && b[1]+b[3] < sy+700 {
			return id
		}
	}
	t.Fatal("no link in view")
	return 0
}

func decode(t *testing.T, line string) recording.Record {
	t.Helper()
	r, err := recording.Decode([]byte(line))
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func inSVGDefinition(n *recording.DOMNode) bool {
	for ; n != nil; n = n.Parent {
		if n.NS == "svg" && (n.N == "symbol" || n.N == "defs") {
			return true
		}
	}
	return false
}
