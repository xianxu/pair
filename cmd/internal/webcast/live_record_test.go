package webcast

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	castassets "github.com/xianxu/pair/cmd/internal/webcast/assets"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/notes"
	"github.com/xianxu/pair/cmd/internal/webcast/recorder"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// recPage is the live check's page: re-renders, open and closed shadow
// roots, a cross-origin iframe, insertRule and a CSSOM-only <style>, a
// <style> over CDP's 10k cut, form state set by script, a password, inner
// scroll, an image, and a link that navigates on click.
const recPage = `<!doctype html><html><head><title>rec</title>
<style id=big>%s</style>
<style id=cssom></style>
<link rel=stylesheet href="/a.css">
</head><body>
<h1 id=h>hello</h1>
<ul id=list></ul>
<div id=open></div><div id=closed></div>
<input id=name><input id=pw type=password><input id=chk type=checkbox>
<select id=sel><option>a<option>b</select>
<div id=scroller style="height:50px;overflow:auto"><div style="height:500px">tall</div></div>
<img src="/dot.png" width=20 height=20>
<iframe id=f src="%s/frame" width=300 height=100></iframe>
<a id=next href="/next" style="display:block;width:100px;height:30px;background:#ccc">next</a>
<script>
const list = document.getElementById('list');
let n = 0;
setInterval(() => { const li = document.createElement('li'); li.textContent = 'item ' + (n++); list.prepend(li); if (list.children.length > 20) list.lastChild.remove(); }, 50);
document.getElementById('open').attachShadow({mode:'open'}).innerHTML = '<b>open root</b>';
document.getElementById('closed').attachShadow({mode:'closed'}).innerHTML = '<i>closed root</i>';
document.getElementById('cssom').sheet.insertRule('#h { color: rgb(200, 0, 0) }');
document.getElementById('sel').selectedIndex = 1;
document.getElementById('chk').checked = true;
</script></body></html>`

func serveRec(t *testing.T) (main, other string) {
	t.Helper()
	big := ".x{color:red}\n" + strings.Repeat("/* padding to pass the 10k cut */\n", 400) + ".tail{color:blue}"
	png, err := os.ReadFile(filepath.Join("recording", "testdata", "fixtures", "rec-app", "frames", "0.png"))
	if err != nil {
		t.Fatal(err)
	}
	frameMux := http.NewServeMux()
	frameMux.HandleFunc("/frame", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `<!doctype html><body style="margin:0"><p id=in>in the frame</p><button id=fb>frame button</button></body>`)
	})
	fl, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	fs := &http.Server{Handler: frameMux}
	go fs.Serve(fl)
	t.Cleanup(func() { fs.Close() })
	// A different site (localhost vs 127.0.0.1): an out-of-process frame.
	other = "http://localhost:" + fmt.Sprint(fl.Addr().(*net.TCPAddr).Port)
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { fmt.Fprintf(w, recPage, big, other) })
	mux.HandleFunc("/a.css", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/css")
		fmt.Fprint(w, "body{font-family:sans-serif} h1{background:url(/dot.png)}")
	})
	mux.HandleFunc("/dot.png", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(png)
	})
	mux.HandleFunc("/next", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `<!doctype html><title>next</title><body><p id=arrived>arrived</p></body>`)
	})
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s := &http.Server{Handler: mux}
	go s.Serve(l)
	t.Cleanup(func() { s.Close() })
	return "http://" + l.Addr().String(), other
}

// liveRecordingChrome launches a recording Chrome into dir, screenshotting
// every shotEvery.
func liveRecordingChrome(t *testing.T, dir string, shotEvery time.Duration) *Chrome {
	t.Helper()
	if os.Getenv("WEBCAST_LIVE_CHROME") == "" {
		t.Skip("set WEBCAST_LIVE_CHROME=1 to drive a real Chrome")
	}
	bin, err := FindChrome(os.Getenv, execLookPath, fileExists)
	if err != nil {
		t.Fatal(err)
	}
	out := func() (recorder.Opened, error) {
		w, err := recording.NewDirWriter(dir)
		if err != nil {
			return recorder.Opened{}, err
		}
		return recorder.Opened{Sink: w, Frames: w, Assets: castassets.NewRecorded(w, castassets.DefaultPolicy)}, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	c, err := Launch(ctx, LaunchOptions{Binary: bin, ProfileDir: t.TempDir(), Headless: true,
		Record: &RecordOptions{ID: "live", Out: out, ShotEvery: shotEvery, Logf: t.Logf}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	return c
}

// evalValue evaluates an expression in the page's main world (the test's
// stand-in for the page itself).
func (h liveHarness) eval(t *testing.T, expr string, out any) {
	t.Helper()
	var r struct {
		Result struct {
			Value any `json:"value"`
		} `json:"result"`
	}
	if err := h.c.call(h.session(t, ""), "Runtime.evaluate", map[string]any{"expression": expr, "returnByValue": true}, &r); err != nil {
		t.Fatal(err)
	}
	if p, ok := out.(*map[string]float64); ok {
		obj, ok := r.Result.Value.(map[string]any)
		if !ok {
			t.Fatalf("%s: %v is not an object", expr, r.Result.Value)
		}
		*p = map[string]float64{}
		for k, v := range obj {
			f, ok := v.(float64)
			if !ok {
				t.Fatalf("%s: %s is %v", expr, k, v)
			}
			(*p)[k] = f
		}
	}
	if p, ok := out.(*string); ok {
		*p, _ = r.Result.Value.(string)
	}
}

// center scrolls an element into view (headless Chrome's window is 800×600)
// and gives its center in viewport coordinates.
func (h liveHarness) center(t *testing.T, sel string) (float64, float64) {
	t.Helper()
	var m map[string]float64
	h.eval(t, fmt.Sprintf(`(() => { const e = document.querySelector(%q); e.scrollIntoView({block: 'center'}); const r = e.getBoundingClientRect(); return {x: r.left + r.width/2, y: r.top + r.height/2}; })()`, sel), &m)
	return m["x"], m["y"]
}

func (h liveHarness) mouse(t *testing.T, typ string, x, y float64) {
	t.Helper()
	p := map[string]any{"type": typ, "x": x, "y": y}
	if typ != "mouseMoved" {
		p["button"], p["clickCount"] = "left", 1
	}
	if err := h.c.call(h.session(t, ""), "Input.dispatchMouseEvent", p, nil); err != nil {
		t.Fatal(err)
	}
}

func (h liveHarness) typeText(t *testing.T, sel, text string) {
	t.Helper()
	x, y := h.center(t, sel)
	h.mouse(t, "mousePressed", x, y)
	h.mouse(t, "mouseReleased", x, y)
	if err := h.c.call(h.session(t, ""), "Input.insertText", map[string]any{"text": text}, nil); err != nil {
		t.Fatal(err)
	}
}

// TestLiveRecording drives the recorder on a real headless Chrome and holds
// its output to #457's validator and reducer:
//
//	WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run LiveRecording -v
func TestLiveRecording(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "rec")
	c := liveRecordingChrome(t, dir, time.Second)
	h := liveHarness{c}
	mainURL, _ := serveRec(t)
	if err := c.call(h.session(t, ""), "Page.navigate", map[string]any{"url": mainURL + "/"}, nil); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second)
	if err := c.SetLive(true); err != nil {
		t.Fatal(err)
	}
	time.Sleep(1500 * time.Millisecond)
	h.typeText(t, "#name", "typed by the operator")
	h.typeText(t, "#pw", "hunter2-never-recorded")
	x, y := h.center(t, "#h")
	h.mouse(t, "mouseMoved", x, y)
	h.eval(t, `document.getElementById('scroller').scrollTop = 120`, nil)
	h.eval(t, `document.getElementById('cssom').sheet.insertRule('#h { font-style: italic }', 1)`, nil)
	time.Sleep(1200 * time.Millisecond)
	if err := c.SetLive(false); err != nil {
		t.Fatal(err)
	}
	h.eval(t, `document.getElementById('h').textContent = 'changed while paused'`, nil)
	time.Sleep(300 * time.Millisecond)
	c.SetLive(true)
	time.Sleep(800 * time.Millisecond)
	// A click that navigates: recorded before the new segment.
	x, y = h.center(t, "#next")
	h.mouse(t, "mouseMoved", x, y)
	h.mouse(t, "mousePressed", x, y)
	h.mouse(t, "mouseReleased", x, y)
	time.Sleep(2 * time.Second)
	stats, err := c.rec.Stats(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("recorder: dropped %v, desync %d, overflow %d %v", stats.Dropped, stats.Desync, stats.Overflow, stats.Examples)
	if len(stats.Dropped) != 0 || stats.Overflow != 0 {
		t.Errorf("the recorder lost records")
	}
	c.Close()

	fs, st, err := recording.ValidateDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fs {
		t.Log(f)
	}
	if errs := recording.Errors(fs); len(errs) != 0 {
		t.Fatalf("%d errors", len(errs))
	}
	r, err := recording.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	all := ""
	for _, name := range []string{recording.FileEvents, recording.FileFrames} {
		all += string(r.Files[name])
	}
	if strings.Contains(all, "hunter2") {
		t.Fatal("a password value was recorded")
	}
	kinds := map[string]int{}
	var order []string
	for _, x := range r.Records {
		kinds[x.Kind()]++
		switch v := x.(type) {
		case *recording.Seg:
			order = append(order, "seg:"+v.Why)
		case *recording.Stop:
			order = append(order, "stop:"+v.Why)
		case *recording.Click:
			order = append(order, "click")
		}
	}
	t.Logf("kinds %v", kinds)
	t.Logf("order %v", order)
	t.Logf("final url %s, %d frames", st.URL, len(r.Frames))
	for _, k := range []string{"seg", "doc", "ins", "rm", "input", "scroll", "sheet", "asset", "layout", "hover", "ptr", "click"} {
		if kinds[k] == 0 {
			t.Errorf("no %s record", k)
		}
	}
	if got := strings.Join(order, " "); got != "seg:live click click stop:pause seg:resume click seg:navigate stop:end" {
		t.Errorf("segments: %s", got)
	}
	if len(r.Frames) == 0 {
		t.Error("no screenshots")
	}
	// The closed shadow root, the iframe document, and the big style's tail
	// are in the recording.
	events := string(r.Files[recording.FileEvents])
	for _, want := range []string{`"mode":"closed"`, `closed root`, `in the frame`, `.tail{color:blue}`, `font-style: italic`, `"secret":true`, `typed by the operator`} {
		if !strings.Contains(events, want) {
			t.Errorf("events lack %s", want)
		}
	}
}

// TestLiveRecordURL records a page with the real recorder for the fidelity
// runs, in #453's scenarios at its 1280×800 viewport:
//
//	WEBCAST_LIVE_CHROME=1 WEBCAST_RECORD_URL=stress WEBCAST_RECORD_OUT=DIR \
//	  WEBCAST_RECORD_SCENARIO=idle WEBCAST_RECORD_SECS=7 go test ./cmd/internal/webcast -run LiveRecordURL -v
//
// then couch --broadcast-web-fidelity DIR (or FIDELITY_DIR=DIR WEBCAST_LIVE_CHROME=1 go test
// ./cmd/internal/webcast -run LiveFidelityFixtures -v).
// The URL "stress" serves #453's stress page from testdata/stress, its
// cross-origin frame from a second origin.
func TestLiveRecordURL(t *testing.T) {
	url, out := os.Getenv("WEBCAST_RECORD_URL"), os.Getenv("WEBCAST_RECORD_OUT")
	if url == "" || out == "" {
		t.Skip("set WEBCAST_RECORD_URL and WEBCAST_RECORD_OUT")
	}
	secs, _ := time.ParseDuration(os.Getenv("WEBCAST_RECORD_SECS") + "s")
	if secs <= 0 {
		secs = 10 * time.Second
	}
	if url == "stress" {
		url = serveStress(t)
	}
	scenario := scenarios[os.Getenv("WEBCAST_RECORD_SCENARIO")]
	if scenario == nil {
		scenario = scenarios["idle"]
	}
	recordScenario(t, url, out, secs, time.Second, scenario)
}

// A scenario drives the cast tab (#453's scenarios.mjs, plus segments).
type scenario func(t *testing.T, h liveHarness, s string)

var scenarios = map[string]scenario{
	"idle": func(*testing.T, liveHarness, string) {},
	"browse": func(t *testing.T, h liveHarness, s string) {
		time.Sleep(1500 * time.Millisecond)
		for i := 0; i < 6; i++ {
			h.wheel(t, s, 500)
			time.Sleep(900 * time.Millisecond)
		}
		h.wheel(t, s, -1500)
	},
	"app": func(t *testing.T, h liveHarness, s string) {
		pause := func(ms int) { time.Sleep(time.Duration(ms) * time.Millisecond) }
		pause(2000)
		h.click(t, s, "#nav-tasks")
		pause(800)
		h.click(t, s, "#draft")
		h.typeSlowly(t, s, "Review PR 42")
		h.key(t, s, "Enter")
		pause(1200)
		h.click(t, s, "#search")
		h.typeSlowly(t, s, "fix")
		pause(1500)
		for i := 0; i < 3; i++ {
			h.key(t, s, "Backspace")
			pause(100)
		}
		h.click(t, s, "li input[type=checkbox]")
		pause(800)
		h.click(t, s, "#nav-settings")
		pause(800)
		h.wheel(t, s, 400)
		pause(800)
		h.click(t, s, "#open-modal")
		pause(1500)
		h.click(t, s, "#close-modal")
		pause(800)
		h.click(t, s, "#nav-dashboard")
		pause(2500)
		h.click(t, s, "#next-doc")
		pause(2500)
	},
	// segments: a same-document navigation, a pause with a change while
	// paused, a resume, a new tab (the cast follows it), and a
	// cross-document navigation there.
	"segments": func(t *testing.T, h liveHarness, s string) {
		time.Sleep(time.Second)
		h.eval(t, `history.pushState({}, '', '#week-3')`, nil)
		time.Sleep(time.Second)
		if err := h.c.SetLive(false); err != nil {
			t.Fatal(err)
		}
		h.eval(t, `document.querySelector('h1').textContent = 'Reports (changed while paused)'`, nil)
		time.Sleep(700 * time.Millisecond)
		if err := h.c.SetLive(true); err != nil {
			t.Fatal(err)
		}
		time.Sleep(time.Second)
		var r struct {
			TargetID string `json:"targetId"`
		}
		var u string
		h.evalInto(t, s, `location.href.replace(/#.*/, '')`, &u)
		if err := h.c.call("", "Target.createTarget", map[string]any{"url": u + "?tab=2"}, &r); err != nil {
			t.Fatal(err)
		}
		time.Sleep(1500 * time.Millisecond)
		s2 := h.session(t, r.TargetID)
		if err := h.c.call(s2, "Page.navigate", map[string]any{"url": strings.Replace(u, "app2.html", "app.html", 1)}, nil); err != nil {
			t.Fatal(err)
		}
		time.Sleep(2 * time.Second)
	},
}

// recordScenario records url at #453's 1280×800 into out for secs while
// the scenario runs, and holds the result to the validator.
func recordScenario(t *testing.T, url, out string, secs, shotEvery time.Duration, run scenario) {
	t.Helper()
	c := liveRecordingChrome(t, out, shotEvery)
	h := liveHarness{c}
	s := h.session(t, "")
	if err := c.call(s, "Emulation.setDeviceMetricsOverride", map[string]any{"width": 1280, "height": 800, "deviceScaleFactor": 1, "mobile": false}, nil); err != nil {
		t.Fatal(err)
	}
	if err := c.call(s, "Page.navigate", map[string]any{"url": url}, nil); err != nil {
		t.Fatal(err)
	}
	time.Sleep(1500 * time.Millisecond)
	if err := c.SetLive(true); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	run(t, h, s)
	time.Sleep(secs - time.Since(start))
	stats, err := c.rec.Stats(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("recorder: %+v", stats)
	if loss := stats.Losses(); loss != "" || stats.Desync != 0 {
		t.Errorf("the recording lost %q (desync %d)", loss, stats.Desync)
	}
	c.Close()
	fs, _, err := recording.ValidateDir(out)
	if err != nil {
		t.Fatal(err)
	}
	if errs := recording.Errors(fs); len(errs) != 0 {
		t.Fatalf("invalid: %v", errs)
	}
}

func (h liveHarness) evalInto(t *testing.T, s, expr string, out any) {
	t.Helper()
	var r struct {
		Result struct {
			Value json.RawMessage `json:"value"`
		} `json:"result"`
	}
	if err := h.c.call(s, "Runtime.evaluate", map[string]any{"expression": expr, "returnByValue": true}, &r); err != nil {
		t.Fatal(err)
	}
	if out != nil {
		if err := json.Unmarshal(r.Result.Value, out); err != nil {
			t.Fatalf("%s: %v", expr, err)
		}
	}
}

func (h liveHarness) input(t *testing.T, s, method string, p map[string]any) {
	t.Helper()
	if err := h.c.call(s, method, p, nil); err != nil {
		t.Fatal(err)
	}
}

func (h liveHarness) wheel(t *testing.T, s string, dy float64) {
	h.input(t, s, "Input.dispatchMouseEvent", map[string]any{"type": "mouseWheel", "x": 640, "y": 400, "deltaX": 0, "deltaY": dy})
}

// click scrolls the element into view as #453's scenarios do and clicks
// its center with real mouse input.
func (h liveHarness) click(t *testing.T, s, sel string) {
	t.Helper()
	var p []float64
	h.evalInto(t, s, fmt.Sprintf(`(() => { const e = document.querySelector(%q); if (!e) return null; e.scrollIntoView({block:'nearest'}); const b = e.getBoundingClientRect(); return [b.x + b.width/2, b.y + b.height/2]; })()`, sel), &p)
	if len(p) != 2 {
		t.Fatalf("no element %s", sel)
	}
	h.input(t, s, "Input.dispatchMouseEvent", map[string]any{"type": "mouseMoved", "x": p[0], "y": p[1]})
	for _, typ := range []string{"mousePressed", "mouseReleased"} {
		h.input(t, s, "Input.dispatchMouseEvent", map[string]any{"type": typ, "x": p[0], "y": p[1], "button": "left", "clickCount": 1})
	}
}

func (h liveHarness) typeSlowly(t *testing.T, s, text string) {
	for _, ch := range text {
		h.input(t, s, "Input.insertText", map[string]any{"text": string(ch)})
		time.Sleep(60 * time.Millisecond)
	}
}

func (h liveHarness) key(t *testing.T, s, k string) {
	code := map[string]int{"Enter": 13, "Backspace": 8}[k]
	for _, typ := range []string{"keyDown", "keyUp"} {
		h.input(t, s, "Input.dispatchKeyEvent", map[string]any{"type": typ, "key": k, "code": k, "windowsVirtualKeyCode": code})
	}
}

// TestRecordFixtures records the raw runs behind the real-recorder
// fixtures (rec-*, #472) into WEBCAST_FIXTURE_RUNS; fixturegen -recorded
// turns each into a fixture (see the fixtures' README):
//
//	WEBCAST_LIVE_CHROME=1 WEBCAST_FIXTURE_RUNS=DIR WEBCAST_APP_PAGES=PAGES \
//	  WEBCAST_BLOG_URL=http://localhost:4321/2026/10/ariadne-1-building-blocks-of-ai-coding \
//	  go test ./cmd/internal/webcast -run RecordFixtures -v
//
// PAGES is #453's prototypes/broadcast-web-recorder/pages with its assets
// set up. A run whose source isn't given is skipped.
func TestRecordFixtures(t *testing.T) {
	runs := os.Getenv("WEBCAST_FIXTURE_RUNS")
	if runs == "" {
		t.Skip("set WEBCAST_FIXTURE_RUNS")
	}
	if err := os.MkdirAll(runs, 0o755); err != nil {
		t.Fatal(err)
	}
	app := ""
	if pages := os.Getenv("WEBCAST_APP_PAGES"); pages != "" {
		app = serveDir(t, pages)
	}
	for _, r := range []struct {
		name, url, scenario string
		secs                time.Duration
	}{
		{"stress", "stress", "idle", 5 * time.Second},
		{"app", app + "/app.html", "app", 22 * time.Second},
		{"blog", os.Getenv("WEBCAST_BLOG_URL"), "browse", 12 * time.Second},
		{"segments", app + "/app2.html", "segments", 9 * time.Second},
		{"driven", "driven", "driven", 16 * time.Second},
	} {
		t.Run(r.name, func(t *testing.T) {
			url := r.url
			switch {
			case url == "stress":
				url = serveStress(t)
			case url == "driven":
				url = serveDriven(t)
			case !strings.HasPrefix(url, "http"):
				t.Skip("no source for this run")
			}
			out := filepath.Join(runs, r.name)
			if err := os.RemoveAll(out); err != nil {
				t.Fatal(err)
			}
			// Every 2 s: half the frames of a fidelity run keeps the
			// fixtures small and still compares each page and moment.
			recordScenario(t, url, out, r.secs, 2*time.Second, scenarios[r.scenario])
		})
	}
}

// serveDir serves a directory on 127.0.0.1 without caching.
func serveDir(t *testing.T, dir string) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	fs := http.FileServer(http.Dir(dir))
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		fs.ServeHTTP(w, r)
	})}
	go srv.Serve(l)
	t.Cleanup(func() { srv.Close() })
	return "http://" + l.Addr().String()
}

// serveStress serves testdata/stress on 127.0.0.1 and its frame on
// localhost (another site: an out-of-process frame), returning the page's URL.
func serveStress(t *testing.T) string {
	t.Helper()
	dir := filepath.Join("testdata", "stress")
	listen := func(h http.Handler) string {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		srv := &http.Server{Handler: h}
		go srv.Serve(l)
		t.Cleanup(func() { srv.Close() })
		return fmt.Sprint(l.Addr().(*net.TCPAddr).Port)
	}
	frame := "http://localhost:" + listen(http.FileServer(http.Dir(dir)))
	page, err := os.ReadFile(filepath.Join(dir, "stress.html"))
	if err != nil {
		t.Fatal(err)
	}
	page = []byte(strings.ReplaceAll(string(page), "{{FRAME}}", frame))
	mux := http.NewServeMux()
	mux.Handle("/assets/", http.FileServer(http.Dir(dir)))
	mux.HandleFunc("/stress.html", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write(page)
	})
	return "http://127.0.0.1:" + listen(mux) + "/stress.html"
}

// pair#493, through the real recorder: a viewer's stroke during a LIVE segment
// goes Board → Chrome.Emit → the recorder (its document check) → the
// recording's Tee, and is in pointer.jsonl while the cast is still running.
// Wired as cast.go's recordingOutput wires a cast: the DirWriter, then the
// board, in one Tee.
//
//	WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run LiveStroke -v
func TestLiveStrokeReachesDiskWhileLive(t *testing.T) {
	if os.Getenv("WEBCAST_LIVE_CHROME") == "" {
		t.Skip("set WEBCAST_LIVE_CHROME=1 to drive a real Chrome")
	}
	bin, err := FindChrome(os.Getenv, execLookPath, fileExists)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(t.TempDir(), "rec")
	var c *Chrome
	bd := board.New(notes.Structural{}, func(r recording.Record, doc int) error { return c.Emit(r, doc) })
	out := func() (recorder.Opened, error) {
		w, err := recording.NewDirWriter(dir)
		if err != nil {
			return recorder.Opened{}, err
		}
		return recorder.Opened{Sink: recording.Tee(w, bd), Frames: w}, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	c, err = Launch(ctx, LaunchOptions{Binary: bin, ProfileDir: t.TempDir(), Headless: true,
		Record: &RecordOptions{ID: "live", Out: out, ShotEvery: time.Second, Logf: t.Logf}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	h := liveHarness{c}
	mainURL, _ := serveRec(t)
	if err := c.call(h.session(t, ""), "Page.navigate", map[string]any{"url": mainURL + "/"}, nil); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second)
	if err := c.SetLive(true); err != nil {
		t.Fatal(err)
	}
	in, err := board.ParseStroke([]byte(`{"viewer":{"id":"ana","name":"Ana"},"seg":1,"id":"s1","color":1,"pts":[[10,20,0],[30,40,16]]}`))
	if err != nil {
		t.Fatal(err)
	}
	// The board learns segment 1 from the recording's own records; until
	// then a stroke is refused as not yet on the page, as a viewer's is.
	deadline := time.Now().Add(10 * time.Second)
	for err = bd.Stroke(in); err != nil && time.Now().Before(deadline); err = bd.Stroke(in) {
		time.Sleep(100 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("the board refused a live stroke: %v", err)
	}
	for {
		body, _ := os.ReadFile(filepath.Join(dir, recording.FilePointer))
		if strings.Contains(string(body), `"k":"stroke"`) {
			break
		}
		if time.Now().After(deadline) {
			st, _ := c.rec.Stats(context.Background())
			t.Fatalf("no stroke in pointer.jsonl while live (%q); recorder dropped %v %v", body, st.Dropped, st.Examples)
		}
		time.Sleep(100 * time.Millisecond)
	}
}
