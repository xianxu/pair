package mockserve

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
)

// WEBCAST_LIVE_CHROME=1 qualifies the actual HTTP front in a private headless
// profile. PAIR_FRONT_BROWSER_NEGATIVE_CONTROL=progress removes the delivered
// progress identity and must fail the first mock-document assertion.
func TestDevelopmentFrontBrowser(t *testing.T) {
	if os.Getenv("WEBCAST_LIVE_CHROME") != "1" {
		t.Skip("set WEBCAST_LIVE_CHROME=1 to drive headless Chrome")
	}
	p, dir := serverProject(t)
	catalogue := func(detailState string) string {
		return fmt.Sprintf(`{"version":1,"screens":[{"id":"index","route":"/","mock":"index.html","state":"real"},{"id":"runs","route":"/runs","mock":"runs.html","state":"real"},{"id":"detail","route":"/runs/detail","mock":"detail.html","state":%q}]}`, detailState)
	}
	mockBrowserWrite(t, dir, "screens.json", catalogue("mock"))
	for _, version := range []string{"mock-1", "mock-2"} {
		mockBrowserWrite(t, dir, version+"/runs.html", `<h1>Mock runs</h1>`)
		mockBrowserWrite(t, dir, version+"/detail.html", `<!doctype html><html><head><title>Mock detail</title></head><body><h1 id="mock-detail">`+version+` detail</h1><a id="home" href="/">Home</a><a id="runs" href="/runs">Runs</a></body></html>`)
	}
	if err := p.Use("mock-1"); err != nil {
		t.Fatal(err)
	}
	var detailReal atomic.Bool
	realHTML := func(route string) string {
		return `<!doctype html><html><head><title>Real app</title></head><body><h1 id="real-screen">` + route + `</h1><a id="home" href="/">Home</a><a id="runs" href="/runs">Runs</a><a id="detail" href="/runs/detail?state=empty#selection">Detail</a></body></html>`
	}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" && r.URL.Path != "/runs" && !(r.URL.Path == "/runs/detail" && detailReal.Load()) {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, realHTML(r.URL.Path))
	}))
	t.Cleanup(upstream.Close)
	s, err := newHybridServer(t.Context(), p, "front-browser")
	if err != nil {
		t.Fatal(err)
	}
	f, err := newDevelopmentFront(t.Context(), s, upstream.URL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.Close() })
	front := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if os.Getenv("PAIR_FRONT_BROWSER_NEGATIVE_CONTROL") == "progress" && strings.HasSuffix(r.URL.Path, ".html") {
			rec := httptest.NewRecorder()
			f.ServeHTTP(rec, r)
			for key, values := range rec.Header() {
				if key != "Content-Length" {
					w.Header()[key] = values
				}
			}
			w.WriteHeader(rec.Code)
			_, _ = io.WriteString(w, strings.ReplaceAll(rec.Body.String(), "couch-mock-progress", "suppressed-progress"))
			return
		}
		f.ServeHTTP(w, r)
	}))
	t.Cleanup(front.Close)
	assertRealBytes := func(route string) {
		t.Helper()
		res, err := http.Get(front.URL + route)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		body, err := io.ReadAll(res.Body)
		if err != nil || res.StatusCode != http.StatusOK || string(body) != realHTML(route) {
			t.Fatalf("real %s changed: status=%d body=%q err=%v", route, res.StatusCode, body, err)
		}
	}
	assertRealBytes("/")
	assertRealBytes("/runs")
	b := newMockBrowser(t)
	realDOM := func(route string) string {
		return fmt.Sprintf(`location.pathname === %q && document.querySelector('#real-screen')?.textContent === %q && !document.querySelector('#couch-mock-progress, #couch-mock-bootstrap, #couch-mock-status, script[src="/__couch_mock/reload.js"]') && !document.documentElement.hasAttribute('data-mock-version') && window.frontNavigationSentinel === undefined`, route, route)
	}
	click := func(id string) {
		t.Helper()
		b.evaluate(t, fmt.Sprintf(`window.frontNavigationSentinel = 'old document'; document.querySelector(%q).click()`, "#"+id))
	}
	mockDetail := func(version string) string {
		return mockDOM(version, "detail") + fmt.Sprintf(` && location.pathname === %q && location.search === '?state=empty' && location.hash === '#selection' && document.querySelector('#mock-detail')?.textContent === %q && window.frontNavigationSentinel === undefined && (() => { const badge = document.querySelector('#couch-mock-progress'); return badge && badge.getClientRects().length > 0 && getComputedStyle(badge).visibility !== 'hidden' && badge.textContent.includes('MOCK') && badge.textContent.includes('2 of 3 screens real'); })()`, "/"+version+"/detail.html", version+" detail")
	}
	b.navigate(t, front.URL+"/")
	b.wait(t, "real home without mock chrome", realDOM("/"))
	click("runs")
	b.wait(t, "real runs full navigation", realDOM("/runs"))
	click("detail")
	b.wait(t, "real to mock detail with authoritative visible progress", mockDetail("mock-1"))
	click("home")
	b.wait(t, "mock to real home full navigation", realDOM("/"))
	click("detail")
	b.wait(t, "reopen mock detail", mockDetail("mock-1"))
	click("runs")
	b.wait(t, "mock to real runs full navigation", realDOM("/runs"))
	click("detail")
	b.wait(t, "mock detail before selection change", mockDetail("mock-1"))
	if err := p.Use("mock-2"); err != nil {
		t.Fatal(err)
	}
	s.refresh(t.Context())
	b.wait(t, "selection change preserves query and hash", mockDetail("mock-2"))
	detailReal.Store(true)
	mockBrowserWrite(t, dir, "screens.json", catalogue("real"))
	s.refresh(t.Context())
	b.wait(t, "open mock promotes to canonical real detail", realDOM("/runs/detail")+` && location.search === '?state=empty' && location.hash === '#selection'`)
	assertRealBytes("/runs/detail")
}
