package webcast

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/notes"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// Two viewers point and leave notes on the blog cast (#470), end to end in
// two headless Chromes: the real viewer page, served by Server with a board
// over the blog fixture, every record through one writer (as the recorder's
// queue is). Viewer A uses a mouse, viewer B touch (an iPad's input). A's
// stroke shows on B's canvas; A's note pins on its element in B within 2 px;
// B's dragged region shows in A; both stay while paused and clear on a
// navigation; the recording holds the notes and the stroke and reduces.
//
//	WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run AnnotateLive -v
func TestAnnotateLiveTwoViewers(t *testing.T) {
	a, b := headlessPage(t), headlessPage(t)
	for _, p := range []*chromePage{a, b} {
		p.call(t, p.session, "Emulation.setDeviceMetricsOverride", map[string]any{"width": 1024, "height": 700, "deviceScaleFactor": 1, "mobile": false}, nil)
	}
	b.call(t, b.session, "Emulation.setTouchEmulationEnabled", map[string]any{"enabled": true, "maxTouchPoints": 5}, nil)

	rec, err := recording.ReadDir(recording.Fixture("blog"))
	if err != nil {
		t.Fatal(err)
	}
	store, rec := throughStore(t, rec)
	const at = 10041 // as TestLiveViewerPage: the scroll is still, layout known
	hub, mem := recording.NewHub(4096), &recording.MemSink{}
	var w *board.Writer
	bd := board.New(notes.Structural{}, func(r recording.Record, doc int) error { return w.Emit(r, doc) })
	w = board.NewWriter(recording.Tee(mem, bd, hub), nil)
	for _, r := range rec.Records {
		if r.Time() <= at {
			if err := w.Put(r); err != nil {
				t.Fatal(err)
			}
		}
	}
	viewer := NewServer("tok", NewFakeBrowser(), time.Second, hub, nil).WithBoard(bd)
	viewer.Assets().Bind(store)
	srv := httptest.NewServer(viewer)
	t.Cleanup(func() { srv.CloseClientConnections(); srv.Close() })
	t.Cleanup(func() { hub.Close() })

	ready := `(() => window.couchViewer && document.querySelector('.annotate-rail') && !document.querySelector('.annotate-rail').hidden ? 'yes' : 'no')()`
	for i, p := range []*chromePage{a, b} {
		p.call(t, p.session, "Page.navigate", map[string]any{"url": fmt.Sprintf("%s/tok/?viewer=v%d", srv.URL, i+1)}, nil)
		waitEval(t, p, "the notes rail", ready, "yes")
		p.eval(t, `(async () => { await couchViewer.ready(); await new Promise((ok) => requestAnimationFrame(() => requestAnimationFrame(ok))); return ''; })()`)
	}
	st, err := recording.Reduce(mem.Records(), math.MaxInt64)
	if err != nil {
		t.Fatal(err)
	}
	target := hoverTarget(t, st) // a link in view
	tb := st.Boxes[target]

	// A points: ✏️, then a mouse drag across the target.
	clickButton(t, a, `.annotate-rail .mode:nth-of-type(1)`)
	from := clientOf(t, a, tb[0]+5, tb[1]+tb[3]/2)
	to := clientOf(t, a, tb[0]+tb[2]-5, tb[1]+tb[3]/2)
	mouse(t, a, "mousePressed", from)
	for i := 1; i <= 8; i++ {
		mouse(t, a, "mouseMoved", [2]float64{from[0] + (to[0]-from[0])*float64(i)/8, from[1]})
		time.Sleep(20 * time.Millisecond)
	}
	mouse(t, a, "mouseReleased", to)
	mid := clientOf(t, b, tb[0]+tb[2]/2, tb[1]+tb[3]/2)
	waitEval(t, b, "A's stroke on B's canvas", fmt.Sprintf(`(() => {
		const c = document.querySelector('.annotate .strokes'), r = c.getBoundingClientRect(), d = devicePixelRatio;
		const px = c.getContext('2d').getImageData(Math.round((%g - r.left) * d), Math.round((%g - r.top) * d), 1, 1).data;
		return px[3] > 0 ? 'drawn' : 'blank';
	})()`, mid[0], mid[1]), "drawn")
	// The posted stroke spans the whole drag: one stroke, every move in it
	// (the first echo must not end it).
	time.Sleep(200 * time.Millisecond)
	ids, maxX := map[string]bool{}, 0.0
	for _, r := range mem.Records() {
		if s, ok := r.(*recording.Stroke); ok && s.T > at-1 && s.Author.Color != "" {
			ids[s.ID] = true
			for _, p := range s.Pts {
				maxX = math.Max(maxX, p[0])
			}
		}
	}
	if len(ids) != 1 || maxX < tb[0]+tb[2]-15 {
		t.Errorf("A's stroke: %d stroke ids, reaching x=%.1f of %.1f", len(ids), maxX, tb[0]+tb[2]-5)
	}
	clickButton(t, a, `.annotate-rail .mode:nth-of-type(1)`) // back to watching

	// A notes the target: 💭, a click on it, a name and text, Save.
	clickButton(t, a, `.annotate-rail .mode:nth-of-type(2)`)
	c := clientOf(t, a, tb[0]+tb[2]/2, tb[1]+tb[3]/2)
	mouse(t, a, "mousePressed", c)
	mouse(t, a, "mouseReleased", c)
	fillAndSave(t, a, "Ana", "This link is hard to see.")
	var noted *recording.Note
	for _, r := range mem.Records() {
		if n, ok := r.(*recording.Note); ok && n.Author.Name == "Ana" {
			noted = n
		}
	}
	if noted == nil || !within(st.Node(noted.At.Node), target) {
		t.Fatalf("A's note %+v: want it on the link %d or inside it", noted, target)
	}
	nb, ok := st.Boxes[noted.At.Node] // the operator's layout of the element
	if !ok {
		nb = tb
	}
	wantPin := clientOf(t, b, nb[0], nb[1])
	waitEval(t, b, "A's note pinned on its element in B", fmt.Sprintf(`(() => {
		const p = document.querySelector('.pin[data-note=%q]');
		if (!p || p.hidden) return 'none';
		const r = p.getBoundingClientRect();
		const ok = Math.abs(r.left - %g) <= 2 && Math.abs(r.top - %g) <= 2 && Math.abs(r.width - %g) <= 2 && Math.abs(r.height - %g) <= 2;
		return ok ? 'pinned' : JSON.stringify([r.left, r.top, r.width, r.height]);
	})()`, noted.ID, wantPin[0], wantPin[1], nb[2]*0.8, nb[3]*0.8), "pinned")

	// B drags a region by touch: 💭, a touch drag, Save.
	clickButton(t, b, `.annotate-rail .mode:nth-of-type(2)`)
	r0 := clientOf(t, b, tb[0], tb[1]+tb[3]+20)
	r1 := clientOf(t, b, tb[0]+200, tb[1]+tb[3]+120)
	touch(t, b, "touchStart", r0)
	for i := 1; i <= 6; i++ {
		touch(t, b, "touchMove", [2]float64{r0[0] + (r1[0]-r0[0])*float64(i)/6, r0[1] + (r1[1]-r0[1])*float64(i)/6})
	}
	touch(t, b, "touchEnd", r1)
	fillAndSave(t, b, "Bo", "Too much space here.")
	waitEval(t, a, "B's region in A", `(() => document.querySelector('.pin.region:not([hidden])') ? 'shown' : 'none')()`, "shown")
	var region *recording.Note
	for _, r := range mem.Records() {
		if n, ok := r.(*recording.Note); ok && n.Author.Name == "Bo" {
			region = n
		}
	}

	if dir := os.Getenv("WEBCAST_REPLAY_SHOTS"); dir != "" {
		for name, p := range map[string]*chromePage{"annotate-a.png": a, "annotate-b.png": b} {
			var shot struct{ Data string }
			p.call(t, p.session, "Page.captureScreenshot", map[string]any{"format": "png"}, &shot)
			raw, _ := base64.StdEncoding.DecodeString(shot.Data)
			os.WriteFile(filepath.Join(dir, name), raw, 0o644)
		}
	}

	// Paused: the notes stay. A navigation: they clear in both.
	pins := fmt.Sprintf(`(() => String([%q, %q].filter((id) => { const p = document.querySelector('.pin[data-note="' + id + '"]'); return p && !p.hidden; }).length))()`, noted.ID, region.ID)
	w.Put(decode(t, fmt.Sprintf(`{"t":%d,"k":"stop","why":"pause"}`, at+100)))
	time.Sleep(300 * time.Millisecond)
	for _, p := range []*chromePage{a, b} {
		waitEval(t, p, "notes while paused", pins, "2")
	}
	// B, still in note mode, taps A's pin by touch and resolves it: the pin
	// goes in both viewers, and the recording says so.
	var chip [2]float64
	if err := json.Unmarshal(b.eval(t, fmt.Sprintf(`(() => { const r = document.querySelector('.pin[data-note=%q] .chip').getBoundingClientRect(); return [r.left + r.width / 2, r.top + r.height / 2]; })()`, noted.ID)), &chip); err != nil {
		t.Fatal(err)
	}
	touch(t, b, "touchStart", chip)
	touch(t, b, "touchEnd", chip)
	waitEval(t, b, "A's note open in B", `(() => { const d = document.querySelector('.note-dialog'); return d.classList.contains('view') && !d.classList.contains('hidden') ? d.querySelector('.note-text').textContent : 'closed'; })()`, "This link is hard to see.")
	b.eval(t, `(() => { [...document.querySelectorAll('.note-buttons button')].find((x) => x.textContent === 'Resolve').click(); return ''; })()`)
	for _, p := range []*chromePage{a, b} {
		waitEval(t, p, "the resolved note gone", pins, "1")
	}
	resolved := false
	for _, r := range mem.Records() {
		if ns, ok := r.(*recording.NoteState); ok && ns.ID == noted.ID && ns.State == recording.NoteResolved {
			resolved = true
		}
	}
	if !resolved {
		t.Errorf("no note-state resolving A's note in the recording")
	}

	seg := st.Seg + 1
	w.Put(decode(t, fmt.Sprintf(`{"t":%d,"k":"seg","seg":%d,"why":"navigate","tab":{"id":"tab1","url":"http://other/"},"vp":{"w":1280,"h":800,"dpr":1}}`, at+200, seg)))
	w.Put(decode(t, fmt.Sprintf(`{"t":%d,"k":"doc","doc":9000001,"url":"http://other/","tree":{"id":9000001,"y":"doc","c":[{"id":9000002,"y":"el","n":"html","c":[{"id":9000003,"y":"el","n":"head"},{"id":9000004,"y":"el","n":"body"}]}]}}`, at+200)))
	for _, p := range []*chromePage{a, b} {
		waitEval(t, p, "notes after a navigation", pins, "0")
		waitEval(t, p, "no pins after a navigation", `(() => String(document.querySelectorAll('.pin').length))()`, "0")
	}

	// The recording holds both viewers' notes and A's stroke, and reduces.
	final, err := recording.Reduce(mem.Records(), math.MaxInt64)
	if err != nil {
		t.Fatalf("the recording with the viewers' records: %v", err)
	}
	byName := map[string]*recording.NoteView{}
	for _, n := range final.Notes {
		byName[n.Author.Name] = n
	}
	ana, bo := byName["Ana"], byName["Bo"]
	switch {
	case ana == nil || bo == nil:
		t.Fatalf("notes by %v", byName)
	case ana.At.Sel == "" || ana.Region != nil:
		t.Errorf("A's note anchor %+v", ana.Note)
	case bo.Region == nil:
		t.Errorf("B's note has no region")
	case ana.Author.ID == bo.Author.ID:
		t.Errorf("both viewers wrote as %s", ana.Author.ID)
	}
	strokes := 0
	for _, s := range final.Strokes {
		if s.Author.ID == ana.Author.ID {
			strokes++
		}
	}
	if strokes == 0 {
		t.Errorf("no stroke chunks by A")
	}
}

// clientOf is where a top-page point shows in p's window.
func clientOf(t *testing.T, p *chromePage, x, y float64) [2]float64 {
	t.Helper()
	var xy [2]float64
	if err := json.Unmarshal(p.eval(t, fmt.Sprintf(`couchViewer.fromPage(%g, %g)`, x, y)), &xy); err != nil {
		t.Fatal(err)
	}
	return xy
}

func clickButton(t *testing.T, p *chromePage, sel string) {
	t.Helper()
	var xy [2]float64
	if err := json.Unmarshal(p.eval(t, fmt.Sprintf(`(() => { const r = document.querySelector(%q).getBoundingClientRect(); return [r.left + r.width / 2, r.top + r.height / 2]; })()`, sel)), &xy); err != nil {
		t.Fatal(err)
	}
	mouse(t, p, "mousePressed", xy)
	mouse(t, p, "mouseReleased", xy)
}

func mouse(t *testing.T, p *chromePage, typ string, xy [2]float64) {
	t.Helper()
	ev := map[string]any{"type": typ, "x": xy[0], "y": xy[1], "button": "left", "buttons": 1, "clickCount": 1}
	if typ == "mouseMoved" {
		ev["button"] = "none"
	}
	if typ == "mouseReleased" {
		ev["buttons"] = 0
	}
	p.call(t, p.session, "Input.dispatchMouseEvent", ev, nil)
}

func touch(t *testing.T, p *chromePage, typ string, xy [2]float64) {
	t.Helper()
	pts := []map[string]any{{"x": xy[0], "y": xy[1], "id": 1}}
	if typ == "touchEnd" {
		pts = []map[string]any{}
	}
	p.call(t, p.session, "Input.dispatchTouchEvent", map[string]any{"type": typ, "touchPoints": pts}, nil)
}

// fillAndSave types a name (when asked) and the text into the open draft,
// and saves it.
func fillAndSave(t *testing.T, p *chromePage, name, text string) {
	t.Helper()
	waitEval(t, p, "the draft", `(() => { const d = document.querySelector('.note-dialog'); return d && !d.classList.contains('hidden') && d.classList.contains('draft') ? 'open' : 'closed'; })()`, "open")
	p.eval(t, fmt.Sprintf(`(() => {
		const set = (el, v) => { el.value = v; el.dispatchEvent(new Event('input', { bubbles: true })); };
		const n = document.querySelector('.note-name');
		if (!n.hidden) set(n, %q);
		set(document.querySelector('.note-input'), %q);
		[...document.querySelectorAll('.note-buttons button')].find((b) => b.textContent === 'Save').click();
		return '';
	})()`, name, text))
	waitEval(t, p, "the draft saved", `(() => { const d = document.querySelector('.note-dialog'); return d.classList.contains('hidden') ? 'saved' : d.querySelector('.note-msg').textContent || 'open'; })()`, "saved")
}

// within reports whether n is the element id or inside it.
func within(n *recording.DOMNode, id recording.NodeID) bool {
	for ; n != nil; n = n.Parent {
		if n.ID == id {
			return true
		}
	}
	return false
}
