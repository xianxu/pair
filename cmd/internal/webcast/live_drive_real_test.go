package webcast

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/webcast/drive"
	"github.com/xianxu/pair/cmd/internal/webcast/recorder"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// errSpy records what the real injector answers, so a failure shows its
// error, not the viewer's generic sentence.
type errSpy struct {
	mu      sync.Mutex
	errs    []string
	acts    []string
	arrived []arrival // every batch, as Couch's injector got it
	ok      int
	refused int
	next    drive.Injector
}

// arrival is one batch at the injector: when it came, how long it took,
// its wheel acts' summed dy, and the error it got.
type arrival struct {
	at, done time.Time
	wheels   int
	dy       float64
	err      error
}

func (s *errSpy) Inject(ctx context.Context, by string, b drive.Batch) error {
	a := arrival{at: time.Now()}
	for _, x := range b.Acts {
		if x.A == drive.ActWheel {
			a.wheels++
			a.dy += x.DY
		}
	}
	err := s.next.Inject(ctx, by, b)
	a.done, a.err = time.Now(), err
	s.mu.Lock()
	defer s.mu.Unlock()
	s.arrived = append(s.arrived, a)
	for _, x := range b.Acts {
		s.acts = append(s.acts, fmt.Sprintf("%s(%.0f,%.0f %s x%d)", x.A, x.X, x.Y, x.Button, x.Count))
	}
	switch {
	case errors.Is(err, drive.ErrRefused):
		s.refused++ // Confine's: a gesture the contract forbids
	case err != nil:
		s.errs = append(s.errs, fmt.Sprintf("%v (%T) for %+v", err, err, b.Acts))
	default:
		s.ok++
	}
	return err
}

func (s *errSpy) Release(ctx context.Context, by string) error {
	if r, ok := s.next.(drive.Releaser); ok {
		return r.Release(ctx, by)
	}
	return nil
}

func (s *errSpy) report() (int, []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ok, append([]string(nil), s.errs...)
}

const blogPage = `<!doctype html><title>blog</title><body style="margin:0;font:16px sans-serif">%s</body>`

// realCast is the production chain, live and driven: the cast Chrome with
// its recorder, a TabInjector behind a spy, the viewer routes, and a
// viewer's real drive.js in a second Chrome holding control.
type realCast struct {
	c    *Chrome
	h    liveHarness
	v    *chromePage
	spy  *errSpy
	ctrl *drive.MemController
	mem  *recording.MemSink
}

func startRealCast(t *testing.T, path string) *realCast {
	t.Helper()
	if os.Getenv("WEBCAST_LIVE_CHROME") == "" {
		t.Skip("set WEBCAST_LIVE_CHROME=1 to drive a real Chrome")
	}
	bin, err := FindChrome(os.Getenv, execLookPath, fileExists)
	if err != nil {
		t.Fatal(err)
	}
	var body strings.Builder
	for i := 0; i < 60; i++ {
		fmt.Fprintf(&body, `<p style="height:60px">Post %d: <a id=l%d href="/post/%d">read post %d</a></p>`, i, i, i, i)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/lines" { // dense text, as a blog's: many short rows under the pointer
			var b strings.Builder
			for i := 0; i < 300; i++ {
				fmt.Fprintf(&b, `<div style="height:18px;font-size:14px">line %d of a long article</div>`, i)
			}
			fmt.Fprintf(w, blogPage, b.String())
			return
		}
		if strings.HasPrefix(r.URL.Path, "/post/") {
			fmt.Fprintf(w, blogPage, `<h1 id=arrived>a post</h1>
<div id=hov style="width:300px;height:40px" onmouseover="window.hovered=1">hover</div>
<div id=dbl style="width:300px;height:40px" ondblclick="window.dblFlag=1">double</div>
<div id=drag style="width:300px;height:120px" onmouseup="window.upFlag=1" onmousemove="if (event.buttons) window.dragged=(window.dragged||0)+1">drag</div>
<input id=q style="width:300px">
<script>document.body.oncontextmenu = () => { window.ctx = 1; };</script>`)
			return
		}
		fmt.Fprintf(w, blogPage, body.String())
	})
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	site := &http.Server{Handler: mux}
	go site.Serve(l)
	t.Cleanup(func() { site.Close() })

	hub, mem := recording.NewHub(1<<16), &recording.MemSink{}
	out := func() (recorder.Opened, error) {
		w, err := recording.NewDirWriter(filepath.Join(t.TempDir(), "rec"))
		if err != nil {
			return recorder.Opened{}, err
		}
		return recorder.Opened{Sink: recording.Tee(w, mem, hub), Frames: w}, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// WEBCAST_DRIVE_WINDOWED=1 runs the cast Chrome in a window, as the
	// operator's is (it takes focus: with the operator's OK only).
	c, err := Launch(ctx, LaunchOptions{Binary: bin, ProfileDir: t.TempDir(), Headless: os.Getenv("WEBCAST_DRIVE_WINDOWED") == "",
		Record: &RecordOptions{ID: "live", Out: out, ShotEvery: -1, Logf: t.Logf}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	t.Cleanup(func() { hub.Close() })
	h := liveHarness{c}
	if dpr := os.Getenv("WEBCAST_DRIVE_DPR"); dpr != "" {
		var f float64
		fmt.Sscan(dpr, &f)
		if err := c.call(h.session(t, ""), "Emulation.setDeviceMetricsOverride", map[string]any{"width": 1200, "height": 900, "deviceScaleFactor": f, "mobile": false}, nil); err != nil {
			t.Fatal(err)
		}
	}
	if err := c.call(h.session(t, ""), "Page.navigate", map[string]any{"url": "http://" + l.Addr().String() + path}, nil); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second)
	if err := c.SetLive(true); err != nil {
		t.Fatal(err)
	}
	time.Sleep(1500 * time.Millisecond)

	spy := &errSpy{next: drive.NewTabInjector(c)}
	ctrl := drive.NewMemController(liveGate{c}, spy, func() int64 { return 0 }, time.Minute)
	t.Cleanup(func() { ctrl.Close() })
	go followDriving(ctrl, c)
	srv := httptest.NewServer(NewServer("tok", c, time.Second, hub, nil).WithDrive(ctrl, "drv"))
	t.Cleanup(func() { srv.CloseClientConnections(); srv.Close() })

	v := headlessPage(t)
	v.call(t, v.session, "Emulation.setDeviceMetricsOverride", map[string]any{"width": 1280, "height": 900, "deviceScaleFactor": 1, "mobile": false}, nil)
	v.call(t, v.session, "Page.navigate", map[string]any{"url": srv.URL + "/drv/"}, nil)
	button := `(() => { const b = document.querySelector('.drive-button'); return b ? b.textContent : ''; })()`
	waitEval(t, v, "the drive button", button, "Ask to drive")
	v.eval(t, `(async () => { await couchViewer.ready(); await new Promise((ok) => requestAnimationFrame(() => requestAnimationFrame(ok))); return ''; })()`)
	seat := v.evalString(t, `couchViewer.drive.seat.viewer`)
	clickButton(t, v, `.drive-button`)
	eventually(t, "the request", func() bool { return len(ctrl.Control().Pending) > 0 })
	if err := ctrl.Grant(seat); err != nil {
		t.Fatal(err)
	}
	waitEval(t, v, "driving", `(() => document.querySelector('.drive.on') ? 'on' : 'off')()`, "on")
	return &realCast{c: c, h: h, v: v, spy: spy, ctrl: ctrl, mem: mem}
}

// pair#505: remote driving through the whole production chain: the real
// cast Chrome and its recorder, the TabInjector over it (as StartCast wires
// it), the viewer routes, and a viewer's real drive.js in a second Chrome.
// The cast tab can run at a device-pixel ratio of 2 (WEBCAST_DRIVE_DPR), as
// the operator's retina window does. A wheel scrolls the page and a click
// follows a link; every batch must be injected.
//
//	WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run DriveRealCast -v
func TestDriveRealCast(t *testing.T) {
	cast := startRealCast(t, "/")
	h, v, spy := cast.h, cast.v, cast.spy
	// A wheel over the page scrolls the cast tab.
	mid := clientOf(t, v, 400, 300)
	v.call(t, v.session, "Input.dispatchMouseEvent", map[string]any{"type": "mouseWheel", "x": mid[0], "y": mid[1], "deltaX": 0, "deltaY": 400}, nil)
	var scrolled string
	eventually(t, "the cast tab scrolled", func() bool { h.eval(t, `String(window.scrollY)`, &scrolled); return scrolled != "0" && scrolled != "" })
	// And again once the page has scrolled (the operator's failing case,
	// pair#505): the wheel's hit test is at page px, past the scroll.
	time.Sleep(500 * time.Millisecond) // the viewer has the first scroll
	first := scrolled
	mid = clientOf(t, v, 400, 500)
	v.call(t, v.session, "Input.dispatchMouseEvent", map[string]any{"type": "mouseWheel", "x": mid[0], "y": mid[1], "deltaX": 0, "deltaY": 400}, nil)
	eventually(t, "the cast tab scrolled again", func() bool {
		var y, y0 float64
		h.eval(t, `String(window.scrollY)`, &scrolled)
		fmt.Sscan(scrolled, &y)
		fmt.Sscan(first, &y0)
		return y > y0
	})

	// A click on a link in view follows it.
	time.Sleep(500 * time.Millisecond) // the viewer has the scroll
	if ok, errs := spy.report(); len(errs) != 0 {
		t.Fatalf("after the wheel: %d batches injected, %d refused:\n%s", ok, len(errs), strings.Join(errs, "\n"))
	}
	// A link now in the middle of the cast tab's viewport, in page px.
	var lp map[string]float64
	h.eval(t, `(() => { const vh = innerHeight / 2; const a = [...document.querySelectorAll('a')].find((e) => { const r = e.getBoundingClientRect(); return r.top > vh - 100 && r.bottom < vh + 100; }); const r = a.getBoundingClientRect(); return {x: r.left + r.width / 2 + scrollX, y: r.top + r.height / 2 + scrollY}; })()`, &lp)
	at := clientOf(t, v, lp["x"], lp["y"])
	mouse(t, v, "mouseMoved", at)
	mouse(t, v, "mousePressed", at)
	mouse(t, v, "mouseReleased", at)
	var arrived string
	for deadline := time.Now().Add(10 * time.Second); arrived != "yes"; time.Sleep(100 * time.Millisecond) {
		h.eval(t, `document.getElementById('arrived') ? 'yes' : 'no'`, &arrived)
		if time.Now().After(deadline) {
			ok, errs := spy.report()
			t.Fatalf("the click never followed the link; %d batches injected, %d refused:\n%s\nviewer says: %s", ok, len(errs), strings.Join(errs, "\n"),
				v.evalString(t, `(document.querySelector('.annotate-msg, .drive-note') || {}).textContent || ''`))
		}
	}

	// The rest of #466's contract on the post page, through the viewer.
	waitEval(t, v, "the viewer on the post", `String((document.getElementById('url').textContent || '').includes('/post/'))`, "true")
	time.Sleep(500 * time.Millisecond)
	center := func(sel string) [2]float64 {
		var p map[string]float64
		h.eval(t, fmt.Sprintf(`(() => { const r = document.querySelector(%q).getBoundingClientRect(); return {x: r.left + 20 + scrollX, y: r.top + r.height / 2 + scrollY}; })()`, sel), &p)
		return clientOf(t, v, p["x"], p["y"])
	}
	castSays := func(what, expr, want string) {
		t.Helper()
		var got string
		for deadline := time.Now().Add(10 * time.Second); got != want; time.Sleep(100 * time.Millisecond) {
			h.eval(t, expr, &got)
			if time.Now().After(deadline) {
				_, errs := spy.report()
				spy.mu.Lock()
				acts := strings.Join(spy.acts[max(0, len(spy.acts)-12):], " ")
				spy.mu.Unlock()
				t.Fatalf("%s: got %q, want %q; injector errors: %s; last acts: %s", what, got, want, strings.Join(errs, "\n"), acts)
			}
		}
	}
	key := func(typ, k, code string, vk, mods int) {
		v.call(t, v.session, "Input.dispatchKeyEvent", map[string]any{"type": typ, "key": k, "code": code, "windowsVirtualKeyCode": vk, "modifiers": mods}, nil)
	}
	// Hover: a move with no button (the shared mouse helper holds one).
	hov := center("#hov")
	for i := 0; i < 3; i++ {
		v.call(t, v.session, "Input.dispatchMouseEvent", map[string]any{"type": "mouseMoved", "x": hov[0] + float64(i*5), "y": hov[1], "button": "none", "buttons": 0}, nil)
		time.Sleep(150 * time.Millisecond)
	}
	castSays("hover", `String(window.hovered || 0)`, "1")
	// Double click.
	dbl := center("#dbl")
	for _, n := range []int{1, 2} {
		for _, typ := range []string{"mousePressed", "mouseReleased"} {
			buttons := 1
			if typ == "mouseReleased" {
				buttons = 0
			}
			v.call(t, v.session, "Input.dispatchMouseEvent", map[string]any{"type": typ, "x": dbl[0], "y": dbl[1], "button": "left", "buttons": buttons, "clickCount": n}, nil)
		}
	}
	castSays("double click", `String(window.dblFlag || 0)`, "1")
	// Drag: press, move with the button held, release.
	d := center("#drag")
	mouse(t, v, "mouseMoved", d)
	mouse(t, v, "mousePressed", d)
	for i := 1; i <= 4; i++ {
		v.call(t, v.session, "Input.dispatchMouseEvent", map[string]any{"type": "mouseMoved", "x": d[0] + float64(20*i), "y": d[1], "button": "left", "buttons": 1}, nil)
		time.Sleep(60 * time.Millisecond)
	}
	mouse(t, v, "mouseReleased", [2]float64{d[0] + 80, d[1]})
	castSays("drag", `String((window.dragged || 0) > 0 && window.upFlag === 1)`, "true")
	// Typing into an input, then select-all, delete and undo.
	q := center("#q")
	mouse(t, v, "mousePressed", q)
	mouse(t, v, "mouseReleased", q)
	castSays("focus", `document.activeElement && document.activeElement.id`, "q")
	v.call(t, v.session, "Input.insertText", map[string]any{"text": "hello"}, nil)
	castSays("typing", `document.getElementById('q').value`, "hello")
	const meta = 4 // macOS: Command
	key("rawKeyDown", "a", "KeyA", 65, meta)
	key("keyUp", "a", "KeyA", 65, meta)
	key("rawKeyDown", "Backspace", "Backspace", 8, 0)
	key("keyUp", "Backspace", "Backspace", 8, 0)
	castSays("select-all and delete", `document.getElementById('q').value`, "")
	key("rawKeyDown", "z", "KeyZ", 90, meta)
	key("keyUp", "z", "KeyZ", 90, meta)
	castSays("undo", `document.getElementById('q').value`, "hello")
	// IME: a composition, then its commit, at the end (undo leaves the
	// restored text selected).
	key("rawKeyDown", "End", "End", 35, 0)
	key("keyUp", "End", "End", 35, 0)
	v.call(t, v.session, "Input.imeSetComposition", map[string]any{"text": "に", "selectionStart": 1, "selectionEnd": 1}, nil)
	v.call(t, v.session, "Input.insertText", map[string]any{"text": "に"}, nil)
	castSays("IME", `document.getElementById('q').value`, "helloに")
	// A refusal: the right button never reaches the cast tab.
	rc := center("#hov")
	v.call(t, v.session, "Input.dispatchMouseEvent", map[string]any{"type": "mousePressed", "x": rc[0], "y": rc[1], "button": "right", "buttons": 2, "clickCount": 1}, nil)
	v.call(t, v.session, "Input.dispatchMouseEvent", map[string]any{"type": "mouseReleased", "x": rc[0], "y": rc[1], "button": "right", "buttons": 0, "clickCount": 1}, nil)
	time.Sleep(time.Second)
	var menu string
	h.eval(t, `String(window.ctx || 0)`, &menu)
	if menu != "0" {
		t.Error("a right click reached the cast tab")
	}
	if ok, errs := spy.report(); len(errs) != 0 {
		t.Fatalf("%d batches injected, %d failed:\n%s", ok, len(errs), strings.Join(errs, "\n"))
	}
}

// pair#516: a fast trackpad flick (100 wheel events in about 1 s, decaying
// like macOS momentum) scrolls the cast tab without a refusal, and the
// measurement the coalescing decision rests on is logged: batches, arrival
// spacing at Couch, injection time, 503s, the total injected against the
// total sent, and how long injection ran past the gesture's end.
//
//	WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run DriveRealCastFlick -v
func TestDriveRealCastFlick(t *testing.T) {
	cast := startRealCast(t, "/lines")
	h, v, spy := cast.h, cast.v, cast.spy
	// The flick runs in the viewer page itself, so its timing is a
	// trackpad's (a CDP call per event costs ~26 ms): 100 wheel events, one
	// every 10 ms, decaying like macOS momentum, into drive.js's own layer.
	mid := clientOf(t, v, 400, 300)
	scale := clientOf(t, v, 400, 400)[1] - mid[1] // client px per 100 page px
	start := time.Now()
	sentClient := v.evalString(t, fmt.Sprintf(`new Promise((ok) => {
		const layer = document.querySelector('.drive.on');
		let i = 0, sum = 0;
		const tick = () => {
			if (i === 100) { ok(String(sum)); return; }
			const dy = Math.round(60 * Math.exp(-i / 30)) + 1;
			sum += dy; i++;
			layer.dispatchEvent(new WheelEvent('wheel', { deltaX: 0, deltaY: dy, clientX: %g, clientY: %g, bubbles: true, cancelable: true }));
			setTimeout(tick, 10);
		};
		tick();
	})`, mid[0], mid[1]))
	ended := time.Now()
	var sentC float64
	fmt.Sscan(sentClient, &sentC)
	sent := sentC * 100 / scale // page px
	time.Sleep(2 * time.Second)

	spy.mu.Lock()
	got := append([]arrival(nil), spy.arrived...)
	spy.mu.Unlock()
	var injected float64
	var refused, busy, wheels int
	var gaps, took []float64
	var lastDone time.Time
	for i, a := range got {
		switch {
		case a.err == nil:
			injected += a.dy
		case errors.Is(a.err, drive.ErrStale), errors.Is(a.err, drive.ErrNotLive):
			refused++
		case errors.Is(a.err, context.Canceled), errors.Is(a.err, context.DeadlineExceeded):
			busy++
		}
		wheels += a.wheels
		if i > 0 {
			gaps = append(gaps, float64(a.at.Sub(got[i-1].at).Milliseconds()))
		}
		took = append(took, float64(a.done.Sub(a.at).Milliseconds()))
		if a.done.After(lastDone) {
			lastDone = a.done
		}
	}
	var scrollY string
	h.eval(t, `String(Math.round(window.scrollY))`, &scrollY)
	t.Logf("flick: %d events over %s, %.0f px sent; %d batches (%d wheel acts), %.0f px injected, %d refused, %d busy; arrival gaps ms %s; injection ms %s; injection ran %s past the gesture; cast scrollY %s; batch dy %s",
		100, ended.Sub(start).Round(time.Millisecond), sent, len(got), wheels, injected, refused, busy, spread(gaps), spread(took), lastDone.Sub(ended).Round(time.Millisecond), scrollY, dys(got))
	if refused > 0 || busy > 0 {
		t.Errorf("%d refused and %d busy batches in a flick", refused, busy)
	}
	if injected < sent*0.95 {
		t.Errorf("injected %.0f px of the %.0f sent", injected, sent)
	}
}

// spread is min / median / max.
func spread(xs []float64) string {
	if len(xs) == 0 {
		return "-"
	}
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	return fmt.Sprintf("%.0f/%.0f/%.0f", s[0], s[len(s)/2], s[len(s)-1])
}

func dys(as []arrival) string {
	var out []string
	for _, a := range as {
		out = append(out, fmt.Sprintf("%.0f", a.dy))
	}
	return strings.Join(out, " ")
}
