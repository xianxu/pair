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
	mu   sync.Mutex
	errs    []string
	acts    []string
	ok      int
	refused int
	next drive.Injector
}

func (s *errSpy) Inject(ctx context.Context, by string, b drive.Batch) error {
	err := s.next.Inject(ctx, by, b)
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, a := range b.Acts {
		s.acts = append(s.acts, fmt.Sprintf("%s(%.0f,%.0f %s x%d)", a.A, a.X, a.Y, a.Button, a.Count))
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

// pair#505: remote driving through the whole production chain: the real
// cast Chrome and its recorder, the TabInjector over it (as StartCast wires
// it), the viewer routes, and a viewer's real drive.js in a second Chrome.
// The cast tab can run at a device-pixel ratio of 2 (WEBCAST_DRIVE_DPR), as
// the operator's retina window does. A wheel scrolls the page and a click
// follows a link; every batch must be injected.
//
//	WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run DriveRealCast -v
func TestDriveRealCast(t *testing.T) {
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
	c, err := Launch(ctx, LaunchOptions{Binary: bin, ProfileDir: t.TempDir(), Headless: true,
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
	if err := c.call(h.session(t, ""), "Page.navigate", map[string]any{"url": "http://" + l.Addr().String() + "/"}, nil); err != nil {
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

	// A wheel over the page scrolls the cast tab.
	mid := clientOf(t, v, 400, 300)
	v.call(t, v.session, "Input.dispatchMouseEvent", map[string]any{"type": "mouseWheel", "x": mid[0], "y": mid[1], "deltaX": 0, "deltaY": 400}, nil)
	var scrolled string
	eventually(t, "the cast tab scrolled", func() bool { h.eval(t, `String(window.scrollY)`, &scrolled); return scrolled != "0" && scrolled != "" })

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
