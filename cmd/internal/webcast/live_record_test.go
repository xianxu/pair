package webcast

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	castassets "github.com/xianxu/pair/cmd/internal/webcast/assets"
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
	png, err := os.ReadFile(filepath.Join("recording", "testdata", "fixtures", "app", "frames", "0.png"))
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

// liveRecordingChrome launches a recording Chrome into dir.
func liveRecordingChrome(t *testing.T, dir string) *Chrome {
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
		Record: &RecordOptions{ID: "live", Out: out, ShotEvery: time.Second, Logf: t.Logf}})
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
	c := liveRecordingChrome(t, dir)
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
// then FIDELITY_DIR=DIR go test ./cmd/internal/webcast/recorder/internal/fidelity -run Dir -v.
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
	c := liveRecordingChrome(t, out)
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
	if os.Getenv("WEBCAST_RECORD_SCENARIO") == "browse" {
		wheel := func(dy float64) {
			if err := c.call(s, "Input.dispatchMouseEvent", map[string]any{"type": "mouseWheel", "x": 640, "y": 400, "deltaX": 0, "deltaY": dy}, nil); err != nil {
				t.Fatal(err)
			}
		}
		time.Sleep(1500 * time.Millisecond)
		for i := 0; i < 6; i++ {
			wheel(500)
			time.Sleep(900 * time.Millisecond)
		}
		wheel(-1500)
	}
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
