package webcast

import (
	"encoding/json"
	"fmt"
	"math"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// replacedPage is the sweep's page (pair#513): every replaced element kind,
// plus the blog's cast embed shape: an element inserted empty and filled by
// script a moment later (asciinema-player does this lazily).
const replacedPage = `<!doctype html><title>replaced</title>
<style>body{margin:0;font:16px sans-serif} .box{margin:8px 0}</style>
<body>
<video id=video class=box width=320 height=180 poster="/poster.png"></video>
<iframe id=iframe class=box src="/inner" width=320 height=120></iframe>
<canvas id=canvas class=box width=320 height=80></canvas>
<picture id=picture class=box><source srcset="/poster.png 1x"><img src="/poster.png" width=160 height=90></picture>
<object id=object class=box data="/poster.png" type="image/png" width=160 height=90></object>
<div id=embed class=box></div>
<script>
document.getElementById('canvas').getContext('2d').fillRect(0, 0, 320, 80);
const inner = document.createElement('div');
document.getElementById('embed').appendChild(inner); // inserted empty
setTimeout(() => {
  const term = document.createElement('div');
  term.className = 'ap-term';
  term.style.cssText = 'width:600px;height:300px;background:#333';
  term.innerHTML = '<pre class="ap-term-text">$ parley</pre>';
  inner.appendChild(term); // filled later: CDP sends only childNodeCountUpdated
}, 1500);
</script>`

// A 1x1 PNG.
var onePixelPNG = []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4, 0x89, 0x00, 0x00, 0x00,
	0x0d, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9c, 0x63, 0xf8, 0xcf, 0xc0, 0xf0, 0x1f, 0x00, 0x05, 0x00, 0x01, 0xff, 0x89, 0x99,
	0x3d, 0x1d, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82}

// pair#513: what a cast shows, the viewer shows at the same place and size:
// an element a script fills after its insert (the blog's asciinema embed,
// which the viewer collapsed to nothing), and each replaced element kind.
// A canvas's pixels are not recorded (they are never in the DOM): the
// viewer shows the canvas at its size, empty.
//
//	WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run LiveReplaced -v
func TestLiveReplacedElements(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, replacedPage) })
	mux.HandleFunc("/inner", func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, `<p>inside the frame</p>`) })
	mux.HandleFunc("/poster.png", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(onePixelPNG)
	})
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	site := &http.Server{Handler: mux}
	go site.Serve(l)
	t.Cleanup(func() { site.Close() })

	dir := filepath.Join(t.TempDir(), "rec")
	c := liveRecordingChrome(t, dir, time.Second) // shots carry the layout boxes
	h := liveHarness{c}
	if err := c.call(h.session(t, ""), "Page.navigate", map[string]any{"url": "http://" + l.Addr().String() + "/"}, nil); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second)
	if err := c.SetLive(true); err != nil {
		t.Fatal(err)
	}
	time.Sleep(3 * time.Second) // past the embed's fill
	// The operator's canvas changes size: the viewer follows the live box,
	// not the attributes (320x80).
	h.eval(t, `(() => { const c = document.getElementById('canvas'); c.style.width = '400px'; c.style.height = '100px'; return ''; })()`, nil)
	time.Sleep(2500 * time.Millisecond) // a shot or two, with their layout
	ids := []string{"video", "iframe", "canvas", "picture", "object", "embed"}
	sizes := func(eval func(string) string) map[string][2]float64 {
		var out map[string][2]float64
		js := fmt.Sprintf(`(() => { const d = (document.querySelector('iframe#page') || {}).contentDocument || document; const o = {}; for (const id of %s) { const e = d.getElementById(id); if (e) { const r = e.getBoundingClientRect(); o[id] = [r.width, r.height]; } } return JSON.stringify(o); })()`, mustJSON(ids))
		if err := json.Unmarshal([]byte(eval(js)), &out); err != nil {
			t.Fatal(err)
		}
		return out
	}
	cast := sizes(func(js string) string { var s string; h.eval(t, js, &s); return s })

	rec, err := recording.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	store, rec := throughStore(t, rec)
	hub := recording.NewHub(1 << 16)
	for _, r := range rec.Records {
		hub.Put(r)
	}
	t.Cleanup(func() { hub.Close() })
	viewer := NewServer("tok", NewFakeBrowser(), time.Second, hub, nil)
	viewer.Assets().Bind(store)
	srv := httptest.NewServer(viewer)
	t.Cleanup(func() { srv.CloseClientConnections(); srv.Close() })
	v := headlessPage(t)
	v.call(t, v.session, "Page.navigate", map[string]any{"url": srv.URL + "/tok/"}, nil)
	v.eval(t, `(async () => { for (let i = 0; i < 100 && !window.couchViewer; i++) await new Promise((ok) => setTimeout(ok, 50)); await couchViewer.ready(); return ''; })()`)
	shown := sizes(func(js string) string { return v.evalString(t, js) })
	// The viewer may scale the page to fit; compare shapes, and each
	// element's width relative to the video's.
	scale := shown["video"][0] / cast["video"][0]
	for _, id := range ids {
		c, s := cast[id], shown[id]
		if s[0] == 0 || s[1] == 0 {
			t.Errorf("%s: shown %v, cast %v: collapsed in the viewer", id, s, c)
			continue
		}
		if math.Abs(s[0]-c[0]*scale) > 2 || math.Abs(s[1]-c[1]*scale) > 2 {
			t.Errorf("%s: shown %v, cast %v at scale %.3f", id, s, c, scale)
		}
	}
	// Every element whose content the recording can't hold is marked;
	// the recorded frame and the late-filled embed are not.
	time.Sleep(500 * time.Millisecond)
	if got := v.evalString(t, `[...document.querySelectorAll('.uncaptured-mark span')].map((s) => s.textContent).sort().join(', ')`); got != "canvas: not captured, object: not captured, video: not captured" {
		t.Errorf("the viewer's marks: %q", got)
	}
	if got := v.evalString(t, `(() => { const f = document.querySelector('iframe#page'); const d = f ? f.contentDocument : document; const p = d.querySelector('#embed .ap-term-text'); return p ? p.textContent : 'none'; })()`); got != "$ parley" {
		t.Errorf("the embed's late content in the viewer: %q", got)
	}
}

func mustJSON(v any) string { b, _ := json.Marshal(v); return string(b) }
