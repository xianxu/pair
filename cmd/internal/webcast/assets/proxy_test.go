package assets

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func get(t *testing.T, p *Proxy, hash string) *http.Response {
	t.Helper()
	rr := httptest.NewRecorder()
	p.Serve(rr, httptest.NewRequest(http.MethodGet, "/tok/assets/"+hash, nil), hash)
	return rr.Result()
}

func body(t *testing.T, res *http.Response) string {
	t.Helper()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func put(t *testing.T, s Store, url, mime, b string) *recording.Asset {
	t.Helper()
	a, err := s.Put(context.Background(), Capture{URL: url, MIME: mime, Body: []byte(b)})
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestProxyUnbound(t *testing.T) {
	if res := get(t, NewProxy(), recording.HashOf([]byte("x"))); res.StatusCode != http.StatusNotFound {
		t.Errorf("an unbound proxy serves nothing: %d", res.StatusCode)
	}
}

func TestProxyServesByHash(t *testing.T) {
	s := NewRecorded(&memWriter{}, DefaultPolicy)
	p := NewProxy()
	p.Bind(s)
	font := put(t, s, "http://x/f.woff2", "font/woff2", "wOF2 body")
	res := get(t, p, font.Hash)
	if res.StatusCode != 200 || body(t, res) != "wOF2 body" {
		t.Fatalf("font: %d", res.StatusCode)
	}
	h := res.Header
	if h.Get("Content-Type") != "font/woff2" || h.Get("X-Content-Type-Options") != "nosniff" ||
		!strings.Contains(h.Get("Cache-Control"), "immutable") || !strings.Contains(h.Get("Content-Security-Policy"), "sandbox") {
		t.Errorf("headers %v", h)
	}
	for _, hash := range []string{"nothex", recording.HashOf([]byte("never")), ""} {
		if res := get(t, p, hash); res.StatusCode != http.StatusNotFound {
			t.Errorf("%q: %d", hash, res.StatusCode)
		}
	}
}

// TestProxyRefusesUnservable: a recorded page labels its own responses; a
// body typed as HTML or script is never served from the viewer's origin.
func TestProxyRefusesUnservable(t *testing.T) {
	s := NewRecorded(&memWriter{}, DefaultPolicy)
	p := NewProxy()
	p.Bind(s)
	for _, typ := range []string{"text/html", "text/javascript", "application/javascript", "garbage;;"} {
		a := put(t, s, "http://x/"+typ, typ, "<script>alert(1)</script>"+typ)
		if res := get(t, p, a.Hash); res.StatusCode != http.StatusNotFound {
			t.Errorf("%s served: %d", typ, res.StatusCode)
		}
	}
}

func TestProxyRewritesCSS(t *testing.T) {
	s := NewRecorded(&memWriter{}, DefaultPolicy)
	p := NewProxy()
	p.Bind(s)
	css := put(t, s, "http://x/css/site.css", "text/css", `@import "base.css"; body{background:url(../bg.png)} a{b:image-set("hi.png" 2x)}`)
	bg := put(t, s, "http://x/bg.png", "image/png", "png-ish")
	res := get(t, p, css.Hash)
	want := `@import "http://x/css/base.css"; body{background:url("` + bg.Hash + `")} a{b:image-set("http://x/css/hi.png" 2x)}`
	if got := body(t, res); got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
	if res.Header.Get("Cache-Control") != "no-store" {
		t.Errorf("a stylesheet's rewrite changes as assets arrive; it must not be cached")
	}
	// Assets arriving later show in the next response.
	base := put(t, s, "http://x/css/base.css", "text/css", "p{}")
	hi := put(t, s, "http://x/css/hi.png", "image/png", "hi")
	want = `@import "` + base.Hash + `"; body{background:url("` + bg.Hash + `")} a{b:image-set("` + hi.Hash + `" 2x)}`
	if got := body(t, get(t, p, css.Hash)); got != want {
		t.Errorf("after arrivals got\n%s", got)
	}
}
