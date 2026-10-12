package webcast

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/cdp"
	castassets "github.com/xianxu/pair/cmd/internal/webcast/assets"
	"github.com/xianxu/pair/cmd/internal/webcast/fidelity"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// The viewer's conformance (#460, atlas/webcast-format.md, "The JS viewer
// contract"): for every fixture at each pinned time, the page the viewer
// drew in a headless Chrome has the reducer's canonical state, and its
// render is within renderTolerance of the real tab's screenshot. Like the
// other live checks it needs a real Chrome outside the agent sandbox, and it
// runs headless, never taking focus:
//
//	WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run Replay -v

// A pixel differs when a channel is off by more than pixelThreshold (of
// 255): anti-aliasing and font hinting stay under it. renderTolerance is the
// share of differing pixels a render may have; see TestLiveReplayFixtures
// for the measured values it was set from.
const (
	pixelThreshold  = 48
	renderTolerance = 0.03
)

// observerLag is how late a fixture's scroll and input records can trail
// the tab: up to 150ms for #453's prototype, whose observer polls every
// 100ms and flushes every 50ms; 0 for #458's recorder, which reports
// synchronously (#472).
func observerLag(name string) int64 {
	if slices.Contains(recording.PrototypeFixtures, name) {
		return 150
	}
	return 0
}

// knownGaps are frames that differ for a filed reason, by fixture and frame
// index. One that comes within tolerance fails, so its entry goes.
var knownGaps = map[string]map[int]string{
	"rec-stress": {0: "pair#475: stress.css binds after the segment's first screenshot, which shows it applied"},
}

// frameOffsets are where a frame's pixels may be instead of its t: the
// product check's (its own, fidelity.CaptureSkew before), and observerLag
// after (the prototype's). recording.FrameTimes keeps those in the frame's
// segment.
func frameOffsets(name string) []int64 {
	offs := slices.Clone(fidelity.Offsets)
	if lag := observerLag(name); lag > 0 {
		offs = append(offs, lag)
	}
	return offs
}

func TestLiveReplayFixtures(t *testing.T) {
	page := headlessPage(t)
	srv := fixtureServer(t)
	page.open(t, srv.URL+"/harness.html")
	for _, name := range recording.Fixtures {
		t.Run(name, func(t *testing.T) {
			rec, err := recording.ReadDir(recording.Fixture(name))
			if err != nil {
				t.Fatal(err)
			}
			page.eval(t, fmt.Sprintf("load(%q)", name))
			for _, at := range recording.CanonicalTimes(rec) {
				got := page.evalString(t, fmt.Sprintf("replayTo(%d)", at))
				want := canonicalAt(t, rec, at)
				if got != want {
					t.Fatalf("t=%d: the drawn page's canonical differs from the reducer's\n%s", at, firstDiff(got, want))
				}
			}
			for i, f := range rec.Frames {
				var tried []string
				ok := false
				for _, at := range recording.FrameTimes(rec.Records, f.T, frameOffsets(name)) {
					diff := page.renderDiff(t, rec, name, f, at)
					tried = append(tried, fmt.Sprintf("%.2f%% at %+dms", 100*diff, at-f.T))
					if ok = diff <= renderTolerance; ok {
						break
					}
				}
				t.Logf("frame %d (t=%d): %s", i, f.T, strings.Join(tried, ", "))
				switch gap, known := knownGaps[name][i]; {
				case ok && known:
					t.Errorf("frame %d (t=%d) is within tolerance now: drop its known gap (%s)", i, f.T, gap)
				case !ok && known:
					t.Logf("frame %d: known gap, %s", i, gap)
				case !ok:
					t.Errorf("frame %d (t=%d): %s of pixels differ, over the %.0f%% tolerance", i, f.T, strings.Join(tried, ", "), 100*renderTolerance)
				}
			}
		})
	}
}

// A viewer joining at any pinned time draws what one that watched from the
// start draws, then and after the next changes.
func TestLiveReplayJoinMidCast(t *testing.T) {
	page := headlessPage(t)
	srv := fixtureServer(t)
	page.open(t, srv.URL+"/harness.html")
	for _, name := range recording.Fixtures {
		rec, err := recording.ReadDir(recording.Fixture(name))
		if err != nil {
			t.Fatal(err)
		}
		times := recording.CanonicalTimes(rec)
		for i, at := range times {
			until := at
			if i+1 < len(times) {
				until = times[i+1]
			}
			got := page.evalString(t, fmt.Sprintf("join(%q, %d, %d)", name, at, until))
			if want := canonicalAt(t, rec, until); got != want {
				t.Errorf("%s: joined at t=%d, at t=%d the canonical differs\n%s", name, at, until, firstDiff(got, want))
			}
		}
	}
}

func canonicalAt(t *testing.T, rec *recording.Recording, at int64) string {
	t.Helper()
	st, err := recording.Reduce(rec.Records, at)
	if err != nil {
		t.Fatal(err)
	}
	return string(st.Canonical())
}

func firstDiff(got, want string) string {
	g, w := strings.Split(got, "\n"), strings.Split(want, "\n")
	for i := 0; i < len(g) || i < len(w); i++ {
		var a, b string
		if i < len(g) {
			a = g[i]
		}
		if i < len(w) {
			b = w[i]
		}
		if a != b {
			return fmt.Sprintf("line %d:\n  drawn:   %s\n  reducer: %s", i+1, a, b)
		}
	}
	return "(equal)"
}

// fixtureServer serves the viewer's modules, the harness, and each
// fixture's records, catch-up and assets as the proxy (#459) will: CSS
// bodies with their url()s pointing at assets/<hash>. It sends the viewer's
// own CSP, so the harness draws under the production policy.
func fixtureServer(t *testing.T, extra ...*recording.Recording) *httptest.Server {
	t.Helper()
	recs := map[string]*recording.Recording{}
	for _, r := range extra {
		recs[filepath.Base(r.Dir)] = r
	}
	for _, name := range recording.Fixtures {
		r, err := recording.ReadDir(recording.Fixture(name))
		if err != nil {
			t.Fatal(err)
		}
		recs[name] = r
	}
	proxies := map[string]*castassets.Proxy{}
	for name, r := range recs {
		store, played := throughStore(t, r)
		recs[name] = played
		proxies[name] = castassets.NewProxy()
		proxies[name].Bind(store)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/web/", func(w http.ResponseWriter, r *http.Request) {
		body, err := webFS.ReadFile("web/" + strings.TrimPrefix(r.URL.Path, "/web/"))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/javascript")
		w.Write(body)
	})
	for _, f := range []string{"harness.html", "harness.js"} {
		mux.HandleFunc("/"+f, func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(f, ".js") {
				w.Header().Set("Content-Type", "text/javascript")
			}
			http.ServeFile(w, r, filepath.Join("testdata", "replay", f))
		})
	}
	mux.HandleFunc("/fx/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/fx/"), "/", 3)
		rec := recs[parts[0]]
		if rec == nil || len(parts) < 2 {
			http.NotFound(w, r)
			return
		}
		switch {
		case parts[1] == "records":
			writeRecords(w, rec.Records)
		case parts[1] == "catchup":
			at, _ := strconv.ParseInt(r.URL.Query().Get("t"), 10, 64)
			var upto []recording.Record
			for _, x := range rec.Records {
				if x.Time() <= at {
					upto = append(upto, x)
				}
			}
			writeRecords(w, recording.CatchUp(upto))
		case parts[1] == "assets" && len(parts) == 3:
			proxies[parts[0]].Serve(w, r, parts[2])
		default:
			http.NotFound(w, r)
		}
	})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", contentSecurityPolicy)
		mux.ServeHTTP(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func writeRecords(w http.ResponseWriter, recs []recording.Record) {
	for _, r := range recs {
		line, err := recording.Encode(r)
		if err != nil {
			panic(err)
		}
		w.Write(append(line, '\n'))
	}
}

// throughStore plays a fixture's assets through the real store, as the
// recorder does: each body is Put, and the store's asset record stands in
// the stream for the fixture's (an image over the budget comes back
// resampled, under another hash). The reducer's canonical state holds no
// asset table, so the fixture's pins still hold.
func throughStore(t *testing.T, rec *recording.Recording) (*castassets.Recorded, *recording.Recording) {
	t.Helper()
	s := castassets.NewRecorded(discardWriter{}, castassets.DefaultPolicy)
	out := *rec
	out.Records = slices.Clone(rec.Records)
	for i, x := range out.Records {
		a, ok := x.(*recording.Asset)
		if !ok {
			continue
		}
		body, err := rec.Asset(a.Hash)
		if err != nil {
			t.Fatal(err)
		}
		got, err := s.Put(context.Background(), castassets.Capture{URL: a.URL, MIME: a.MIME, Body: body})
		if err != nil {
			t.Fatal(err)
		}
		got.Header = a.Header
		out.Records[i] = got
	}
	return s, &out
}

// chromePage is one headless Chrome tab driven over CDP.
type chromePage struct {
	conn    *cdp.Conn
	session string

	mu        sync.Mutex
	requested []string // URLs from Network.requestWillBeSent, once Network is enabled
}

// requests is every URL the page has requested since Network.enable.
func (p *chromePage) requests() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return slices.Clone(p.requested)
}

func headlessPage(t *testing.T) *chromePage {
	t.Helper()
	if os.Getenv("WEBCAST_LIVE_CHROME") == "" {
		t.Skip("set WEBCAST_LIVE_CHROME=1 to drive a real Chrome")
	}
	bin, err := FindChrome(os.Getenv, execLookPath, fileExists)
	if err != nil {
		t.Fatal(err)
	}
	cmdR, cmdW, _ := os.Pipe()
	outR, outW, _ := os.Pipe()
	cmd := exec.Command(bin, "--remote-debugging-pipe", "--headless=new", "--user-data-dir="+t.TempDir(),
		"--no-first-run", "--no-default-browser-check", "--hide-scrollbars", "about:blank")
	cmd.ExtraFiles = []*os.File{cmdR, outW}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	cmdR.Close()
	outW.Close()
	conn := cdp.New(outR, cmdW)
	t.Cleanup(func() {
		conn.Close()
		cmd.Process.Kill()
		cmd.Wait()
	})
	p := &chromePage{conn: conn}
	var target struct{ TargetID string }
	p.call(t, "", "Target.createTarget", map[string]any{"url": "about:blank"}, &target)
	var att struct{ SessionID string }
	p.call(t, "", "Target.attachToTarget", map[string]any{"targetId": target.TargetID, "flatten": true}, &att)
	p.session = att.SessionID
	p.call(t, p.session, "Emulation.setDeviceMetricsOverride", map[string]any{"width": 1280, "height": 800, "deviceScaleFactor": 1, "mobile": false}, nil)
	go func() { // keep the pump flowing; note requests
		for ev := range conn.Events() {
			if ev.Method != "Network.requestWillBeSent" {
				continue
			}
			var r struct{ Request struct{ URL string } }
			if json.Unmarshal(ev.Params, &r) == nil {
				p.mu.Lock()
				p.requested = append(p.requested, r.Request.URL)
				p.mu.Unlock()
			}
		}
	}()
	return p
}

func (p *chromePage) call(t *testing.T, session, method string, params, result any) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err := p.conn.Call(ctx, session, method, params, result); err != nil {
		t.Fatalf("%s: %v", method, err)
	}
}

func (p *chromePage) open(t *testing.T, url string) {
	t.Helper()
	p.call(t, p.session, "Page.navigate", map[string]any{"url": url}, nil)
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if p.evalString(t, "String(window.harnessReady === true)") == "true" {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("%s never became ready", url)
}

func (p *chromePage) eval(t *testing.T, expr string) json.RawMessage {
	t.Helper()
	var res struct {
		Result struct {
			Value json.RawMessage
		}
		ExceptionDetails *struct {
			Text      string
			Exception struct{ Description string }
		}
	}
	p.call(t, p.session, "Runtime.evaluate", map[string]any{"expression": expr, "awaitPromise": true, "returnByValue": true}, &res)
	if e := res.ExceptionDetails; e != nil {
		t.Fatalf("%s: %s %s", expr, e.Text, e.Exception.Description)
	}
	return res.Result.Value
}

func (p *chromePage) evalString(t *testing.T, expr string) string {
	t.Helper()
	var s string
	if err := json.Unmarshal(p.eval(t, expr), &s); err != nil {
		t.Fatalf("%s: not a string: %v", expr, err)
	}
	return s
}

// renderDiff replays a fixture to at, as the tab looked then, and compares
// the capture with the frame's screenshot.
func (p *chromePage) renderDiff(t *testing.T, rec *recording.Recording, name string, f *recording.Frame, at int64) float64 {
	t.Helper()
	p.eval(t, fmt.Sprintf("load(%q)", name))
	p.evalString(t, fmt.Sprintf("replayAt(%d, %d)", at, ReplayWindow))
	return p.compare(t, filepath.Join(rec.Dir, f.File), f)
}

// compare captures the frame at the screenshot's scale and returns the
// share of pixels that differ from it.
func (p *chromePage) compare(t *testing.T, file string, f *recording.Frame) float64 {
	t.Helper()
	var shot struct{ Data string }
	p.call(t, p.session, "Page.captureScreenshot", map[string]any{
		"format": "png",
		"clip":   map[string]any{"x": 0, "y": 0, "width": float64(f.W) / f.Scale, "height": float64(f.H) / f.Scale, "scale": f.Scale},
	}, &shot)
	raw, err := base64.StdEncoding.DecodeString(shot.Data)
	if err != nil {
		t.Fatal(err)
	}
	got, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	wantRaw, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	want, err := png.Decode(bytes.NewReader(wantRaw))
	if err != nil {
		t.Fatal(err)
	}
	if dir := os.Getenv("WEBCAST_REPLAY_SHOTS"); dir != "" {
		os.WriteFile(filepath.Join(dir, strings.ReplaceAll(strings.TrimPrefix(file, filepath.Dir(filepath.Dir(filepath.Dir(file)))+"/"), "/", "_")), raw, 0o644)
	}
	return pixelDiff(got, want)
}

// pixelDiff is the share of pixels where some channel differs by more than
// pixelThreshold; a size mismatch counts as entirely different.
func pixelDiff(a, b image.Image) float64 {
	ab, bb := a.Bounds(), b.Bounds()
	if ab.Dx() != bb.Dx() || ab.Dy() != bb.Dy() {
		return 1
	}
	differ := 0
	for y := 0; y < ab.Dy(); y++ {
		for x := 0; x < ab.Dx(); x++ {
			r1, g1, b1, _ := a.At(ab.Min.X+x, ab.Min.Y+y).RGBA()
			r2, g2, b2, _ := b.At(bb.Min.X+x, bb.Min.Y+y).RGBA()
			if far(r1, r2) || far(g1, g2) || far(b1, b2) {
				differ++
			}
		}
	}
	return float64(differ) / float64(ab.Dx()*ab.Dy())
}

func far(a, b uint32) bool {
	d := int(a>>8) - int(b>>8)
	return d > pixelThreshold || -d > pixelThreshold
}

// URLs resolve against the document that holds them, and SVG's mixed-case
// tags are matched: a <style> in a frame resolves its relative url()s
// against the frame's URL, and an <feImage>'s href goes through the asset
// table like any image.
func TestLiveRenderResolvesPerDocument(t *testing.T) {
	page := headlessPage(t)
	dir := filepath.Join(t.TempDir(), "synthetic")
	w, err := recording.NewDirWriter(dir)
	if err != nil {
		t.Fatal(err)
	}
	asset := func(url string, body []byte, mime string) *recording.Asset {
		hash, err := w.PutAsset(body)
		if err != nil {
			t.Fatal(err)
		}
		return &recording.Asset{URL: url, Hash: hash, MIME: mime, Size: int64(len(body))}
	}
	pic := asset("http://x/pic.png", []byte("top picture"), "image/png")
	img := asset("http://x/sub/img.png", []byte("frame picture"), "image/png")
	b := recording.NewBuilder("synthetic")
	b.Add(pic).Add(img)
	fe := b.El("feImage", []string{"href", "pic.png"})
	fe.NS = "svg"
	filter := b.El("filter", []string{"id", "f"}, fe)
	filter.NS = "svg"
	svg := b.El("svg", nil, filter)
	svg.NS = "svg"
	frame := b.El("iframe", nil)
	b.Page(recording.SegLive, "http://x/", svg, frame)
	style := b.El("style", nil, b.Txt("div{background:url(img.png)}"))
	inner := recording.Node{ID: b.ID(), Y: recording.YDoc, C: []recording.Node{b.El("html", nil, b.El("head", nil, style), b.El("body", nil))}}
	b.Add(&recording.Doc{Doc: inner.ID, Host: frame.ID, URL: "http://x/sub/f.html", Tree: inner})
	for _, r := range b.Recs {
		if err := w.Put(r); err != nil {
			t.Fatal(err)
		}
	}
	w.Close()
	rec, err := recording.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	srv := fixtureServer(t, rec)
	page.open(t, srv.URL+"/harness.html")
	page.eval(t, `load("synthetic")`)
	page.evalString(t, "replayTo(1e9)")
	got := page.evalString(t, `(() => {
		const top = document.getElementById('page').contentDocument;
		const fe = top.getElementsByTagNameNS('http://www.w3.org/2000/svg', 'feImage')[0];
		const style = top.querySelector('iframe').contentDocument.querySelector('style');
		return JSON.stringify([fe.getAttribute('href'), style.textContent]);
	})()`)
	want := fmt.Sprintf(`["/fx/synthetic/assets/%s","div{background:url(\"/fx/synthetic/assets/%s\")}"]`, pic.Hash, img.Hash)
	if got != want {
		t.Fatalf("drawn %s\nwant  %s", got, want)
	}
}
