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
// index, and the most they may differ at their best time, so a gap guards
// what does work. One that comes within tolerance fails, so its entry goes.
var knownGaps = map[string]map[int]knownGap{
	// Unstyled, it is 24% off (#475 before the hold).
	"rec-stress": {0: {"photo.png's bytes are recorded at 152ms, after the screenshot at 84 had it, and an image does not block rendering (#475: non-blocking asset lag)", 0.04}},
}

type knownGap struct {
	why  string
	most float64
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
				best := 1.0
				try := func(offs []int64) bool {
					for _, at := range recording.FrameTimes(rec.Records, f.T, offs) {
						diff := page.renderDiff(t, rec, name, f, at)
						tried = append(tried, fmt.Sprintf("%.2f%% at %+dms", 100*diff, at-f.T))
						best = min(best, diff)
						if diff <= renderTolerance {
							return true
						}
					}
					return false
				}
				ok := try(frameOffsets(name))
				// The viewer holds the previous paint until a new document's
				// stylesheets bind (#475), so it shows the tab's page late
				// when their bytes were recorded after the screenshot
				// (rec-stress frame 0 at 84ms, stress.css at 103): the frame
				// may be where that hold releases. Asked only now: replaying
				// ahead warms the asset cache, which shifts transitions.
				if !ok {
					if held := page.evalInt(t, fmt.Sprintf("releaseAt(%q, %d)", name, f.T)) - f.T; held > 0 {
						ok = try([]int64{held})
					}
				}
				t.Logf("frame %d (t=%d): %s", i, f.T, strings.Join(tried, ", "))
				switch gap, known := knownGaps[name][i]; {
				case ok && known:
					t.Errorf("frame %d (t=%d) is within tolerance now: drop its known gap (%s)", i, f.T, gap.why)
				case known && best > gap.most:
					t.Errorf("frame %d (t=%d): %s of pixels differ, over its known gap's %.0f%% (%s)", i, f.T, strings.Join(tried, ", "), 100*gap.most, gap.why)
				case known:
					t.Logf("frame %d: known gap, %s", i, gap.why)
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

func (p *chromePage) evalInt(t *testing.T, expr string) int64 {
	t.Helper()
	var n float64
	if err := json.Unmarshal(p.eval(t, expr), &n); err != nil {
		t.Fatalf("%s: not a number: %v", expr, err)
	}
	return int64(n)
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

// A new document's paint is held until its stylesheets bind (#475): the
// previous paint stays on top while a linked stylesheet's bytes are still
// to come, a print-only one blocks nothing, and one that never arrives
// lets the page through after HOLD_MS. #page holds the new document all
// along.
func TestLiveRenderHoldsPaintUntilStylesheetsBind(t *testing.T) {
	page := headlessPage(t)
	dir := filepath.Join(t.TempDir(), "held")
	w, err := recording.NewDirWriter(dir)
	if err != nil {
		t.Fatal(err)
	}
	css := []byte("p{color:rgb(0,0,255)}")
	hash, err := w.PutAsset(css)
	if err != nil {
		t.Fatal(err)
	}
	b := recording.NewBuilder("held")
	b.Page(recording.SegLive, "http://x/a", b.El("p", nil, b.Txt("A")), b.El("img", []string{"alt", "pic", "srcset", "none.png 1x, none2.png 2x"}))
	doc := func(url, text string, links ...[]string) {
		b.Add(&recording.Seg{Seg: b.NextSeg(), Why: recording.SegNavigate, Tab: recording.Tab{ID: "tab1", URL: url}, VP: recording.Viewport{W: 1280, H: 800, DPR: 1}})
		var head []recording.Node
		for _, l := range links {
			head = append(head, b.El("link", l))
		}
		b.Snapshot(url, b.El("html", nil, b.El("head", nil, head...), b.El("body", nil, b.El("p", nil, b.Txt(text)))))
	}
	b.At(1000)
	doc("http://x/b", "B", []string{"rel", "stylesheet", "href", "b.css"}, []string{"rel", "stylesheet", "media", "print", "href", "p.css"})
	b.At(1100).Add(&recording.Asset{URL: "http://x/b.css", Hash: hash, MIME: "text/css", Size: int64(len(css))})
	b.At(2000)
	doc("http://x/c", "C", []string{"rel", "stylesheet", "href", "never.css"})
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
	page.eval(t, `load("held")`)
	// shown is the text on top at the page's corner; current, #page's.
	const shown = `(() => { const f = document.elementFromPoint(5, 5); return f.contentDocument.body ? f.contentDocument.body.textContent : ''; })()`
	const current = `document.getElementById('page').contentDocument.body.textContent`
	check := func(at, wantShown, wantCurrent string) {
		t.Helper()
		if got := page.evalString(t, shown); got != wantShown {
			t.Errorf("%s: shows %q, want %q", at, got, wantShown)
		}
		if got := page.evalString(t, current); got != wantCurrent {
			t.Errorf("%s: #page has %q, want %q", at, got, wantCurrent)
		}
	}
	page.evalString(t, "replayTo(0)")
	check("t=0", "A", "A")
	// An image none of whose candidates has arrived is one not loaded yet,
	// not a broken one: no srcset left to fail, so no alt text.
	if got := page.evalString(t, `(() => { const i = document.getElementById('page').contentDocument.querySelector('img'); return [...i.attributes].map((a) => a.name).join(' '); })()`); got != "" {
		t.Errorf("an image with no candidate yet has attributes %q, want none", got)
	}
	page.evalString(t, "replayTo(1000)")
	check("b.css not arrived", "A", "B")
	page.evalString(t, "replayTo(1100)")
	check("b.css bound", "B", "B")
	if got := page.evalString(t, `getComputedStyle(document.getElementById('page').contentDocument.querySelector('p')).color`); got != "rgb(0, 0, 255)" {
		t.Errorf("b.css bound: p is %s, want b.css's blue", got)
	}
	page.evalString(t, "replayTo(2000)")
	check("never.css pending", "B", "C")
	page.eval(t, "new Promise((ok) => setTimeout(ok, 700))") // HOLD_MS is 500
	check("past the deadline", "C", "C")
}

// A paint hold is one state on the frame pair (#475), handed between
// renderers: a renderer starting a hold while another's is in flight (a
// resync, a seek) takes it over, the previous paint stays, and the old
// deadline is cancelled; a new segment during a hold keeps the previous
// paint; the owner's deadline releases it; a stylesheet that fails to
// load stops blocking; and the held paint stays under what the shell
// draws after the frames (the pointer, marks, the Paused label). Timers
// are fake, so each ordering is driven, not waited for.
func TestLiveRenderHoldHandOff(t *testing.T) {
	page := headlessPage(t)
	srv := fixtureServer(t)
	page.open(t, srv.URL+"/harness.html")
	got := page.evalString(t, `(async () => {
		const { Model } = await import('/web/model.js');
		const { Renderer, currentFrame } = await import('/web/render.js');
		const box = document.createElement('div');
		box.style.cssText = 'position:absolute;left:0;top:0;width:200px;height:200px';
		const frame = document.createElement('iframe');
		frame.setAttribute('sandbox', 'allow-same-origin');
		frame.id = 'handoff';
		const overlay = document.createElement('div');
		overlay.style.cssText = 'position:absolute;left:150px;top:150px;width:50px;height:50px';
		box.append(frame, overlay);
		document.body.replaceChildren(box);
		const timers = [], cleared = new Set();
		const fake = { set: (fn) => timers.push(fn) - 1, clear: (id) => cleared.add(id) };
		const fire = (id) => { if (!cleared.has(id)) timers[id](); };
		const draw = (r, text, link) => {
			const head = link ? [{ id: 4, y: 'el', n: 'link', a: { rel: 'stylesheet', href: link } }] : [];
			for (const rec of [
				...(r.model.head ? [] : [{ t: 0, k: 'head', format: 'couch-webcast', v: 1, id: 'h', started: 's' }]),
				{ t: 0, k: 'seg', seg: (r.model.seg || 0) + 1, why: 'live', tab: { id: 'a', url: 'http://x/' }, vp: { w: 200, h: 200, dpr: 1 } },
				{ t: 0, k: 'doc', doc: 1, url: 'http://x/', tree: { id: 1, y: 'doc', c: [{ id: 2, y: 'el', n: 'html', c: [{ id: 3, y: 'el', n: 'head', c: head }, { id: 5, y: 'el', n: 'body', c: [{ id: 6, y: 'text', v: text }] }] }] } },
			]) { r.model.apply(rec); r.apply(rec); }
		};
		const renderer = () => new Renderer(currentFrame(frame), new Model(), () => '', fake);
		const at = (x, y) => { const e = document.elementFromPoint(x, y); return e === overlay ? 'overlay' : (e.contentDocument && e.contentDocument.body ? e.contentDocument.body.textContent : '?'); };
		const current = () => document.getElementById('handoff').contentDocument.body.textContent;
		const seen = [];
		const look = (what) => seen.push(what + ': shows ' + at(5, 5) + ', #page ' + current() + ', corner ' + at(175, 175));
		const r0 = renderer();
		draw(r0, 'P');
		look('P drawn, nothing blocks');
		const r1 = renderer();
		draw(r1, 'Q', 'http://x/q.css');
		look('Q waits for q.css');
		const r2 = renderer();
		draw(r2, 'R', 'http://x/r.css');
		look('R takes the hold over');
		seen.push('r1 holding ' + r1.holding + ', r2 holding ' + r2.holding);
		fire(1); // r1's deadline: cancelled
		look('r1 deadline');
		draw(r2, 'S', 'http://x/s.css');
		look('S during the hold');
		fire(timers.length - 1);
		look('r2 deadline');
		const r3 = renderer();
		draw(r3, 'T', 'http://x/t.css');
		const link = document.getElementById('handoff').contentDocument.querySelector('link');
		r3.setLinkCSS(4, link, 'p{}');
		link.dispatchEvent(new Event('error'));
		look('t.css failed');
		return seen.join('\n');
	})()`)
	want := strings.Join([]string{
		"P drawn, nothing blocks: shows P, #page P, corner overlay",
		"Q waits for q.css: shows P, #page Q, corner overlay",
		"R takes the hold over: shows P, #page R, corner overlay",
		"r1 holding false, r2 holding true",
		"r1 deadline: shows P, #page R, corner overlay",
		"S during the hold: shows P, #page S, corner overlay",
		"r2 deadline: shows S, #page S, corner overlay",
		"t.css failed: shows T, #page T, corner overlay",
	}, "\n")
	if got != want {
		t.Errorf("hand-off:\n%s\nwant:\n%s", got, want)
	}
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
