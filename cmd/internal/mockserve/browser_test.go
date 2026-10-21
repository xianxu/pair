package mockserve

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/cdp"
	"github.com/xianxu/pair/cmd/internal/webcast"
)

// Run with WEBCAST_LIVE_CHROME=1. Chrome has a private profile and debugging
// pipes; no visible browser or repository fixture is opened. For qualification
// negative controls, PAIR_MOCK_BROWSER_NEGATIVE_CONTROL=tags or script suppresses
// the corresponding delivered behavior and must make this test fail.
func TestMockBrowserReload(t *testing.T) {
	if os.Getenv("WEBCAST_LIVE_CHROME") != "1" {
		t.Skip("set WEBCAST_LIVE_CHROME=1 to drive headless Chrome")
	}
	t.Run("edits_navigation_and_repair", func(t *testing.T) {
		p, dir := mockBrowserProject(t)
		s, err := NewServer(t.Context(), p, "browser")
		if err != nil {
			t.Fatal(err)
		}
		srv := httptest.NewServer(mockBrowserHandler(s))
		t.Cleanup(srv.Close)
		b := newMockBrowser(t)
		b.navigate(t, srv.URL+"/mock-1/index.html?state=empty#selection")
		b.wait(t, "initial DOM", mockDOM("mock-1", "index")+` && document.querySelector('#fixture')?.textContent === 'shared' && window.mockScript === 'js-one'`)
		// This is the delivered script's pure reducer, executed by the real JS engine.
		b.wait(t, "reload decisions", `(() => { const f = window.couchMockReloadDecision; const a = {version:'mock-1',screen:'nested/detail',revision:'a'}; return f(a,{version:'mock-1',revision:'a'}).kind === 'none' && f(a,{version:'mock-1',revision:'b'}).kind === 'reload' && f(a,{version:'mock-2',revision:'b'}).target === '/mock-2/nested/detail.html' && f(a,{error:'repair'}).kind === 'error' && f(a,{}).kind === 'error'; })()`)
		edit := func(name, body string) { mockBrowserWrite(t, dir, name, body); s.refresh(t.Context()) }
		edit("mock-1/index.html", mockBrowserHTML("html-two"))
		b.wait(t, "HTML edit", `document.querySelector('#heading')?.textContent === 'html-two'`)
		edit("mock-1/app.css", "body { color: rgb(12, 34, 56); }")
		b.wait(t, "CSS edit", `getComputedStyle(document.body).color === 'rgb(12, 34, 56)'`)
		edit("mock-1/app.js", mockBrowserJS("js-two"))
		b.wait(t, "JS edit", `window.mockScript === 'js-two'`)
		edit("fixtures.json", `{"title":"fixture-two"}`)
		b.wait(t, "fixture edit", `document.querySelector('#fixture')?.textContent === 'fixture-two'`)
		b.evaluate(t, `document.querySelector('#detail').click()`)
		b.wait(t, "clickable detail screen", mockDOM("mock-1", "detail")+` && location.pathname === '/mock-1/detail.html' && document.querySelector('#fixture')?.textContent === 'fixture-two'`)
		// Detail does not exist in mock-2: the server must fall back to its index.
		b.evaluate(t, `history.replaceState(null,'','?state=empty#selection')`)
		if err := p.Use("mock-2"); err != nil {
			t.Fatal(err)
		}
		s.refresh(t.Context())
		b.wait(t, "switch forward with fallback", mockDOM("mock-2", "index")+` && location.pathname === '/mock-2/index.html' && location.search === '?state=empty' && location.hash === '#selection'`)
		if err := p.Use("mock-1"); err != nil {
			t.Fatal(err)
		}
		s.refresh(t.Context())
		b.wait(t, "switch backward", mockDOM("mock-1", "index")+` && location.pathname === '/mock-1/index.html' && location.search === '?state=empty' && location.hash === '#selection' && window.mockScript === 'js-two'`)
		edit("fixtures.json", "{")
		b.wait(t, "invalid JSON banner preserves document", `document.querySelector('#couch-mock-status')?.textContent.includes('Mock preview:') && document.querySelector('#heading')?.textContent === 'html-two'`)
		edit("fixtures.json", `{"title":"repaired"}`)
		b.wait(t, "repair clears banner and reloads fixture", `!document.querySelector('#couch-mock-status') && document.querySelector('#fixture')?.textContent === 'repaired'`)
	})
	for _, switchVersion := range []bool{false, true} {
		t.Run(fmt.Sprintf("change_before_first_poll/switch=%v", switchVersion), func(t *testing.T) {
			p, dir := mockBrowserProject(t)
			s, err := NewServer(t.Context(), p, "first-poll")
			if err != nil {
				t.Fatal(err)
			}
			held, release := make(chan struct{}), make(chan struct{})
			var first atomic.Bool
			var once sync.Once
			unblock := func() { once.Do(func() { close(release) }) }
			// Registered before the server cleanup so an early failure cannot strand it.
			handler := mockBrowserHandler(s)
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == statePath && first.CompareAndSwap(false, true) {
					close(held)
					select {
					case <-release:
					case <-r.Context().Done():
						return
					}
				}
				handler.ServeHTTP(w, r)
			}))
			t.Cleanup(func() { unblock(); srv.Close() })
			b := newMockBrowser(t)
			b.navigate(t, srv.URL+"/mock-1/index.html")
			b.wait(t, "original HTML delivered", mockDOM("mock-1", "index")+` && document.querySelector('#heading')?.textContent === 'html-one'`)
			ctx, cancel := context.WithTimeout(t.Context(), 8*time.Second)
			defer cancel()
			select {
			case <-held:
			case <-ctx.Done():
				t.Fatal("first state request never reached barrier")
			}
			wantVersion := "mock-1"
			if switchVersion {
				if err := p.Use("mock-2"); err != nil {
					t.Fatal(err)
				}
				wantVersion = "mock-2"
			} else {
				mockBrowserWrite(t, dir, "mock-1/index.html", mockBrowserHTML("intervening-edit"))
			}
			s.refresh(t.Context())
			unblock()
			predicate := mockDOM(wantVersion, "index")
			if switchVersion {
				predicate += ` && location.pathname === '/mock-2/index.html'`
			} else {
				predicate += ` && document.querySelector('#heading')?.textContent === 'intervening-edit'`
			}
			b.wait(t, "first observation compares delivered revision", predicate)
		})
	}
}

// TestSampleBrowser qualifies the real example's links, fixture-backed content,
// and scenario controls. It only reads the checked-in project.
func TestSampleBrowser(t *testing.T) {
	if os.Getenv("WEBCAST_LIVE_CHROME") != "1" {
		t.Skip("set WEBCAST_LIVE_CHROME=1 to drive headless Chrome")
	}
	p, err := OpenProject(filepath.Join("..", "..", "..", "examples", "couch-mocks"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { p.Close() })
	s, err := NewServer(t.Context(), p, "sample")
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(s)
	t.Cleanup(srv.Close)
	b := newMockBrowser(t)
	b.navigate(t, srv.URL+"/")
	ready := func(screen string) string {
		return mockDOM("mock-2", screen) + ` && document.documentElement.dataset.ready === 'true' && document.querySelector('#project')?.textContent === 'Echo control board'`
	}
	b.wait(t, "sample overview", ready("index")+` && document.querySelector('#view')?.textContent.includes('2 runs')`)
	b.evaluate(t, `document.querySelector('nav[aria-label="Screens"] a[href="runs.html"]').click()`)
	b.wait(t, "sample runs link and shared fixture", ready("runs")+` && document.querySelector('#view')?.textContent.includes('Morning checks') && document.querySelectorAll('#view a.run').length === 2`)
	b.evaluate(t, `document.querySelector('nav[aria-label="Screens"] a[href="detail.html"]').click()`)
	b.wait(t, "sample detail link and shared fixture", ready("detail")+` && document.querySelector('#view')?.textContent === 'Morning checks · Ready'`)
	b.evaluate(t, `document.querySelector('nav[aria-label="Screens"] a[href="index.html"]').click()`)
	b.wait(t, "sample overview link", ready("index"))
	b.evaluate(t, `document.querySelector('nav[aria-label="Screens"] a[href="runs.html"]').click()`)
	b.wait(t, "sample runs again", ready("runs"))
	for _, state := range []struct{ name, predicate string }{
		{"empty", `document.querySelector('#view')?.textContent.includes('No runs yet')`},
		{"error", `document.querySelector('#view')?.textContent.includes('Runs are temporarily unavailable')`},
		{"long-list", `document.querySelectorAll('#view a.run').length === 80`},
	} {
		b.evaluate(t, fmt.Sprintf(`document.querySelector('nav[aria-label="Scenarios"] a[href="runs.html?state=%s"]').click()`, state.name))
		b.wait(t, "sample "+state.name, ready("runs")+" && "+state.predicate)
	}
}

func mockDOM(version, screen string) string {
	return fmt.Sprintf(`document.documentElement.dataset.mockVersion === %q && document.documentElement.dataset.mockScreen === %q`, version, screen)
}
func mockBrowserHTML(heading string) string {
	return `<!doctype html><html><head><link rel="stylesheet" href="app.css"><script defer src="app.js"></script></head><body><h1 id="heading">` + heading + `</h1><span id="fixture"></span><a id="detail" href="detail.html">Detail</a></body></html>`
}
func mockBrowserJS(label string) string {
	return fmt.Sprintf(`window.mockScript = %q; fetch('/fixtures.json').then(r => r.json()).then(f => { document.querySelector('#fixture').textContent = f.title; });`, label)
}
func mockBrowserWrite(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(name)), []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
}
func mockBrowserProject(t *testing.T) (*Project, string) {
	t.Helper()
	p, dir := serverProject(t)
	for _, v := range []string{"mock-1", "mock-2"} {
		mockBrowserWrite(t, dir, v+"/index.html", mockBrowserHTML("html-one"))
		mockBrowserWrite(t, dir, v+"/app.css", "body { color: rgb(1, 2, 3); }")
		mockBrowserWrite(t, dir, v+"/app.js", mockBrowserJS("js-one"))
	}
	mockBrowserWrite(t, dir, "mock-1/detail.html", mockBrowserHTML("detail"))
	if err := p.Use("mock-1"); err != nil {
		t.Fatal(err)
	}
	return p, dir
}
func mockBrowserHandler(s *Server) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mode := os.Getenv("PAIR_MOCK_BROWSER_NEGATIVE_CONTROL")
		if mode == "script" && r.URL.Path == reloadPath {
			w.Header().Set("Content-Type", "text/javascript")
			return
		}
		if mode == "tags" && strings.HasSuffix(r.URL.Path, ".html") {
			rec := httptest.NewRecorder()
			s.ServeHTTP(rec, r)
			for key, values := range rec.Header() {
				if key != "Content-Length" {
					w.Header()[key] = values
				}
			}
			w.WriteHeader(rec.Code)
			body := strings.ReplaceAll(rec.Body.String(), "data-mock-version=", "data-suppressed-version=")
			_, _ = w.Write([]byte(body))
			return
		}
		s.ServeHTTP(w, r)
	})
}

type mockBrowser struct {
	conn    *cdp.Conn
	session string
	ctx     context.Context
}

func newMockBrowser(t *testing.T) *mockBrowser {
	t.Helper()
	bin, err := webcast.FindChrome(os.Getenv, exec.LookPath, func(path string) bool { info, err := os.Stat(path); return err == nil && !info.IsDir() })
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
	t.Cleanup(cancel)
	cmdR, cmdW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	outR, outW, err := os.Pipe()
	if err != nil {
		cmdR.Close()
		cmdW.Close()
		t.Fatal(err)
	}
	cmd := exec.CommandContext(ctx, bin, "--remote-debugging-pipe", "--headless=new", "--user-data-dir="+t.TempDir(), "--no-first-run", "--no-default-browser-check", "about:blank")
	cmd.ExtraFiles = []*os.File{cmdR, outW}
	if err := cmd.Start(); err != nil {
		cmdR.Close()
		cmdW.Close()
		outR.Close()
		outW.Close()
		t.Fatal(err)
	}
	cmdR.Close()
	outW.Close()
	conn := cdp.New(outR, cmdW)
	pumped := make(chan struct{})
	go func() {
		defer close(pumped)
		for ev := range conn.Events() {
			if ev.Barrier != nil {
				close(ev.Barrier)
			}
		}
	}()
	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()
	t.Cleanup(func() {
		conn.Close()
		cancel()
		outR.Close()
		timer := time.NewTimer(5 * time.Second)
		defer timer.Stop()
		select {
		case <-exited:
		case <-timer.C:
			t.Error("Chrome process did not join")
		}
		select {
		case <-pumped:
		case <-time.After(5 * time.Second):
			t.Error("Chrome event pump did not join")
		}
	})
	b := &mockBrowser{conn: conn, ctx: ctx}
	var target struct{ TargetID string }
	b.call(t, "", "Target.createTarget", map[string]any{"url": "about:blank"}, &target)
	var attached struct{ SessionID string }
	b.call(t, "", "Target.attachToTarget", map[string]any{"targetId": target.TargetID, "flatten": true}, &attached)
	b.session = attached.SessionID
	return b
}
func (b *mockBrowser) call(t *testing.T, session, method string, params, result any) {
	t.Helper()
	ctx, cancel := context.WithTimeout(b.ctx, 8*time.Second)
	defer cancel()
	if err := b.conn.Call(ctx, session, method, params, result); err != nil {
		t.Fatalf("%s: %v", method, err)
	}
}
func (b *mockBrowser) navigate(t *testing.T, url string) {
	t.Helper()
	b.call(t, b.session, "Page.navigate", map[string]any{"url": url}, nil)
}
func (b *mockBrowser) evaluate(t *testing.T, expression string) {
	t.Helper()
	b.call(t, b.session, "Runtime.evaluate", map[string]any{"expression": expression}, nil)
}
func (b *mockBrowser) wait(t *testing.T, label, expression string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(b.ctx, 8*time.Second)
	defer cancel()
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	var last error
	for {
		var result struct {
			Result           struct{ Value json.RawMessage }
			ExceptionDetails json.RawMessage
		}
		last = b.conn.Call(ctx, b.session, "Runtime.evaluate", map[string]any{"expression": "Boolean(" + expression + ")", "returnByValue": true}, &result)
		if last == nil && string(result.Result.Value) == "true" {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("%s: DOM predicate did not become true: %s (last error %v, result %s)", label, expression, last, result.Result.Value)
		case <-ticker.C:
		}
	}
}
