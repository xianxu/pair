package broadcast

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/terminal"
)

const testToken = "tok_abcdefghijklmnopqrstuvwxyz0123456789ABCD"

// The view link refuses every method but GET on every route, the pointer
// link's POST routes included, without reading a body.
var routes = []string{"", "viewer.js", "viewer.css", "xterm.js", "xterm.css", "addon-unicode11.js", "events", "point", "note", "resolve"}

func testServer(t *testing.T, opts HubOptions, ping time.Duration) (*Hub, http.Handler) {
	t.Helper()
	h := NewHub(opts)
	t.Cleanup(func() { h.Close(nil) })
	return h, NewServer(ServerOptions{Token: testToken, Hub: h, Ping: ping})
}

// noRead fails the test if the handler reads the request body.
type noRead struct{ t *testing.T }

func (r noRead) Read([]byte) (int, error) {
	r.t.Error("handler read the request body")
	return 0, io.EOF
}
func (noRead) Close() error { return nil }

func serve(t *testing.T, srv http.Handler, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	req.Body = noRead{t}
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	return rec
}

func assertSecurityHeaders(t *testing.T, path string, rec *httptest.ResponseRecorder) {
	t.Helper()
	want := map[string]string{
		"Content-Security-Policy": contentSecurityPolicy,
		"Cache-Control":           "no-store",
		"Referrer-Policy":         "no-referrer",
		"X-Content-Type-Options":  "nosniff",
	}
	for k, v := range want {
		if got := rec.Header().Get(k); got != v {
			t.Errorf("%s: %s = %q, want %q", path, k, got, v)
		}
	}
}

func TestServerRefusesWrongTokenAndPaths(t *testing.T) {
	_, srv := testServer(t, HubOptions{}, 0)
	for _, path := range []string{
		"/",
		"/x/",
		"/" + testToken,
		"/" + testToken[:10] + "/",
		"/" + testToken + "x/",
		"/x" + testToken + "/",
		"/" + strings.ToUpper(testToken) + "/",
		"/" + testToken + "/nope",
		"/" + testToken + "/vendor/xterm/xterm.js",
		"/" + testToken + "/../" + testToken + "/",
		"/" + testToken + "/./viewer.js",
		"/" + testToken + "//viewer.js",
		"/" + testToken + "/viewer.js/",
	} {
		rec := serve(t, srv, http.MethodGet, path)
		if rec.Code != http.StatusNotFound {
			t.Errorf("GET %s = %d, want 404", path, rec.Code)
		}
		if strings.Contains(rec.Body.String(), testToken) {
			t.Errorf("GET %s reflected the token", path)
		}
		assertSecurityHeaders(t, path, rec)
	}
}

func TestServerIsGetOnly(t *testing.T) {
	_, srv := testServer(t, HubOptions{}, 0)
	for _, route := range routes {
		path := "/" + testToken + "/" + route
		for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch, http.MethodHead, http.MethodOptions} {
			rec := serve(t, srv, method, path)
			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("%s %s = %d, want 405", method, path, rec.Code)
			}
			if rec.Header().Get("Allow") != http.MethodGet {
				t.Errorf("%s %s: Allow = %q", method, path, rec.Header().Get("Allow"))
			}
			assertSecurityHeaders(t, path, rec)
		}
	}
}

func TestServerServesAssetsWithHeaders(t *testing.T) {
	_, srv := testServer(t, HubOptions{}, 0)
	types := map[string]string{
		"":                   "text/html; charset=utf-8",
		"viewer.js":          "text/javascript; charset=utf-8",
		"notes.js":           "text/javascript; charset=utf-8",
		"viewer.css":         "text/css; charset=utf-8",
		"xterm.js":           "text/javascript; charset=utf-8",
		"xterm.css":          "text/css; charset=utf-8",
		"addon-unicode11.js": "text/javascript; charset=utf-8",
	}
	for route, ctype := range types {
		path := "/" + testToken + "/" + route
		rec := serve(t, srv, http.MethodGet, path)
		if rec.Code != http.StatusOK || rec.Body.Len() == 0 {
			t.Fatalf("GET %s = %d (%d bytes)", path, rec.Code, rec.Body.Len())
		}
		if got := rec.Header().Get("Content-Type"); got != ctype {
			t.Errorf("GET %s Content-Type %q, want %q", path, got, ctype)
		}
		assertSecurityHeaders(t, path, rec)
	}
}

func TestViewerPageLoadsOnlySameOrigin(t *testing.T) {
	page := string(mustAsset(t, "index.html"))
	refs := regexp.MustCompile(`(?i)\b(?:src|href)\s*=\s*"([^"]*)"`).FindAllStringSubmatch(page, -1)
	if len(refs) < 4 {
		t.Fatalf("found only %d asset references", len(refs))
	}
	for _, ref := range refs {
		u := ref[1]
		if strings.Contains(u, ":") || strings.HasPrefix(u, "/") || strings.Contains(u, "..") {
			t.Errorf("asset %q is not a relative same-origin path", u)
		}
	}
	if regexp.MustCompile(`(?is)<script\b[^>]*>\s*[^<\s]`).MatchString(page) {
		t.Error("page has an inline script body")
	}
	if regexp.MustCompile(`(?i)\sstyle\s*=|<style\b`).MatchString(page) {
		t.Error("page has inline style")
	}
	if regexp.MustCompile(`(?i)\bon[a-z]+\s*=`).MatchString(page) {
		t.Error("page has an inline event handler")
	}
	// No hint bar (#447 M3): the screen keeps the whole window; the rail's
	// controls explain themselves in tooltips.
	if strings.Contains(page, `id="hint"`) || strings.Contains(page, `id="controls"`) {
		t.Error("the page has a hint bar")
	}
	viewer := string(mustAsset(t, "viewer.js"))
	collab := string(mustAsset(t, "collab.js"))
	// The one thing a page keeps (#446): a pointer viewer's color pick, a
	// palette index under one key, read and written only by collab.js's
	// colorStore over the storage viewer.js hands it.
	if strings.Count(viewer, "localStorage") != 1 || strings.Contains(viewer, "getItem(") || strings.Contains(viewer, "setItem(") ||
		strings.Contains(collab, "localStorage") || strings.Count(collab, "getItem(") != 1 || strings.Count(collab, "setItem(") != 1 ||
		!strings.Contains(collab, "const COLOR_KEY = 'couch.pointerColor';") || !strings.Contains(collab, "store.get(COLOR_KEY)") || !strings.Contains(collab, "store.set(COLOR_KEY, String(i))") {
		t.Error("the viewer may keep only the pointer color pick in localStorage")
	}
	notes := string(mustAsset(t, "notes.js"))
	for name, src := range map[string]string{"viewer.js": viewer, "notes.js": notes, "collab.js": collab} {
		for _, api := range []string{"sessionStorage", "indexedDB", "caches", "serviceWorker", "document.cookie", "XMLHttpRequest", "WebSocket", "sendBeacon", "EventSource(\"http", "EventSource('http", "innerHTML", "outerHTML", "insertAdjacentHTML", "document.write"} {
			if strings.Contains(src, api) {
				t.Errorf("%s uses %s", name, api)
			}
		}
	}
	if strings.Contains(notes, "fetch(") || strings.Contains(notes, "localStorage") {
		t.Error("notes.js makes no request and keeps nothing; viewer.js's postJSON does the posting")
	}
	// Exactly two request functions besides the page's own assets and stream:
	// postJSON, collab.js's poster, to a relative route from a fixed list (the
	// pointer POST, #412, and the note routes, #447); and sessionLost's probe
	// of the gate's own `session` route (pair#495). Both send the link's
	// session cookie (same-origin credentials) and nothing else.
	if n := strings.Count(viewer, "fetch("); n != 1 || !strings.Contains(viewer, "const postJSON = poster((url, init) => fetch(url, init), POST_ROUTES);") ||
		!strings.Contains(viewer, "sessionLost: () => sessionLost(fetch),") {
		t.Errorf("viewer.js has %d fetch calls, want exactly postJSON's and the session probe", n)
	}
	if n := strings.Count(collab, "fetchFn("); n != 2 || strings.Count(collab, "fetch(") != 0 || !strings.Contains(collab, "await fetchFn('session', {") {
		t.Errorf("collab.js requests only through poster's fetchFn and the session probe")
	}
	if !strings.Contains(viewer, "const POST_ROUTES = ['point', 'note', 'resolve'];") || !strings.Contains(collab, "if (!routes.includes(route)) {") ||
		!strings.Contains(collab, "return fetchFn(route, {") || strings.Count(collab, "credentials: 'same-origin'") != 2 || strings.Contains(collab, "credentials: 'omit'") {
		t.Error("postJSON must target a relative route from POST_ROUTES with same-origin credentials")
	}
}

func mustAsset(t *testing.T, name string) []byte {
	t.Helper()
	b, err := webFS.ReadFile("web/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// sseReader reads Server-Sent Events from a live response.
type sseReader struct {
	t *testing.T
	r *bufio.Reader
}

type sseEvent struct {
	name, data string
	comment    bool
}

func (s *sseReader) next() sseEvent {
	s.t.Helper()
	var ev sseEvent
	for {
		line, err := s.r.ReadString('\n')
		if err != nil {
			s.t.Fatalf("stream ended: %v", err)
		}
		line = strings.TrimSuffix(line, "\n")
		switch {
		case line == "":
			if ev.name != "" || ev.data != "" || ev.comment {
				return ev
			}
		case strings.HasPrefix(line, ":"):
			ev.comment = true
		case strings.HasPrefix(line, "event: "):
			ev.name = strings.TrimPrefix(line, "event: ")
		case strings.HasPrefix(line, "data: "):
			ev.data += strings.TrimPrefix(line, "data: ")
		}
	}
}

// httpServer serves srv for one test. Teardown drops client connections
// first: an open SSE stream never finishes on its own, and Close waits for it.
func httpServer(t *testing.T, srv http.Handler) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(srv)
	t.Cleanup(func() {
		ts.CloseClientConnections()
		ts.Close()
	})
	return ts
}

func openEvents(t *testing.T, base string) (*http.Response, *sseReader) {
	t.Helper()
	resp, err := castClient.Get(base + "/" + testToken + "/events")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	return resp, &sseReader{t: t, r: bufio.NewReader(resp.Body)}
}

func TestServerSSEStreamsFrames(t *testing.T) {
	h, srv := testServer(t, HubOptions{}, 20*time.Millisecond)
	ts := httpServer(t, srv)
	a := live(t, "first frame")
	offer(h, a, terminal.FramePublic)
	resp, events := openEvents(t, ts.URL)
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("Content-Type %q", ct)
	}
	var v screen
	readFrame := func() {
		t.Helper()
		for {
			ev := events.next()
			if ev.comment {
				continue
			}
			if ev.name != "frame" {
				t.Fatalf("event %q", ev.name)
			}
			var m struct {
				Cols, Rows int
				B          string
			}
			if err := json.Unmarshal([]byte(ev.data), &m); err != nil {
				t.Fatal(err)
			}
			data, err := base64.StdEncoding.DecodeString(m.B)
			if err != nil {
				t.Fatal(err)
			}
			v.apply(t, Message{Cols: m.Cols, Rows: m.Rows, Data: data})
			return
		}
	}
	readFrame()
	if v.text() != frameText(t, a) {
		t.Fatalf("late joiner screen %q", v.text())
	}
	b := live(t, "second frame")
	offer(h, b, terminal.FramePublic)
	readFrame()
	if v.text() != frameText(t, b) {
		t.Fatalf("screen after diff %q", v.text())
	}
	if ev := events.next(); !ev.comment {
		t.Fatalf("expected a ping comment, got %+v", ev)
	}
	h.Close(ErrIndicatorHidden)
	for {
		ev := events.next()
		if ev.comment {
			continue
		}
		if ev.name != "end" || !strings.Contains(ev.data, "LIVE indicator") {
			t.Fatalf("end event %+v", ev)
		}
		break
	}
}

func TestServerTooManyViewers(t *testing.T) {
	_, srv := testServer(t, HubOptions{MaxViewers: 1}, time.Hour)
	ts := httpServer(t, srv)
	openEvents(t, ts.URL)
	resp, err := castClient.Get(ts.URL + "/" + testToken + "/events")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("second viewer = %d, want 503", resp.StatusCode)
	}
}

func TestServerEventsAfterEndIsGone(t *testing.T) {
	h, srv := testServer(t, HubOptions{}, time.Hour)
	h.Close(errors.New("stopped"))
	<-h.Done()
	if rec := serve(t, srv, http.MethodGet, "/"+testToken+"/events"); rec.Code != http.StatusGone {
		t.Fatalf("events after end = %d, want 410", rec.Code)
	}
}

// A viewer that disconnects frees its slot without waiting for a frame.
func TestServerReleasesSlotOnDisconnect(t *testing.T) {
	h, srv := testServer(t, HubOptions{MaxViewers: 1}, 10*time.Millisecond)
	ts := httpServer(t, srv)
	resp, _ := openEvents(t, ts.URL)
	resp.Body.Close()
	deadline := time.Now().Add(5 * time.Second)
	for {
		var n int
		h.do(func() { n = len(h.subs) })
		if n == 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("disconnected viewer still holds its slot")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// BR-9: the end reason crosses to remote viewers, so it is a closed
// vocabulary. No error text from the OS or a wrapped cause gets through.
func TestEndReasonIsAClosedVocabulary(t *testing.T) {
	secret := "/Users/someone/private/couch-broadcast-1234/s: accept tcp 127.0.0.1:53211"
	cases := []struct {
		err  error
		want string
	}{
		{nil, "the operator stopped broadcasting"},
		{ErrHubClosed, "the operator stopped broadcasting"},
		{ErrIndicatorHidden, "the LIVE indicator was not visible on the operator's screen"},
		{ErrTunnelExited, "the tunnel closed"},
		{fmt.Errorf("%w: %s", ErrServerFailed, secret), "the broadcast server stopped"},
		{fmt.Errorf("%w: %s", ErrTunnelExited, secret), "the tunnel closed"},
		{errors.New(secret), "the broadcast ended"},
		{fmt.Errorf("wrapped: %w", errors.New(secret)), "the broadcast ended"},
	}
	for _, c := range cases {
		got := EndReason(c.err)
		if got != c.want {
			t.Errorf("EndReason(%v) = %q, want %q", c.err, got, c.want)
		}
		if strings.Contains(got, "/") || strings.Contains(got, "127.0.0.1") {
			t.Errorf("EndReason(%v) leaks local detail: %q", c.err, got)
		}
	}
}

// A viewer that stops reading is dropped once a write can't finish, freeing
// its slot; it can't hold the stream and a viewer slot forever.
func TestServerDropsStalledViewer(t *testing.T) {
	h, srv := testServer(t, HubOptions{MaxViewers: 1, QueueDepth: 1000}, 20*time.Millisecond)
	ts := httpServer(t, srv)
	conn, err := net.Dial("tcp", strings.TrimPrefix(ts.URL, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if tcp, ok := conn.(*net.TCPConn); ok {
		tcp.SetReadBuffer(4096)
	}
	fmt.Fprintf(conn, "GET /%s/events HTTP/1.1\r\nHost: x\r\n\r\n", testToken)
	// Never read; push frames until the server's writes back up.
	deadline := time.Now().Add(15 * time.Second)
	for i := 0; ; i++ {
		offer(h, live(t, fmt.Sprintf("%d %s", i, strings.Repeat("x", 30))), terminal.FramePublic)
		var n int
		h.do(func() { n = len(h.subs) })
		if n == 0 && i > 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("a viewer that stopped reading still holds its slot after %d frames", i)
		}
		time.Sleep(time.Millisecond)
	}
}
