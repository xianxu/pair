package mockserve

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func serverProject(t *testing.T) (*Project, string) {
	t.Helper()
	dir := t.TempDir()
	for name, body := range map[string]string{"fixtures.json": `{"title":"shared"}`, "design.md": "private", "mock-1/index.html": "<h1>one</h1>", "mock-1/detail.html": "<h1>detail</h1>", "mock-2/index.html": "<h1>two</h1>", "mock-2/app.css": "body{color:red}"} {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	p, err := OpenProject(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { p.Close() })
	return p, dir
}

func TestHTTPProjectRoutes(t *testing.T) {
	p, dir := serverProject(t)
	s, err := NewServer(t.Context(), p, "test")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		path string
		code int
		body string
	}{{"/", 302, "/mock-2/index.html"}, {"/mock-2/index.html", 200, `data-mock-version="mock-2"`}, {"/fixtures.json", 200, `"shared"`}, {"/mock-2/app.css", 200, "color:red"}, {"/design.md", 404, ""}, {"/.mock-current", 404, ""}, {"/mock-2/../../design.md", 404, ""}, {"/mock-2/%2fetc", 404, ""}, {"/mock-2/no.html", 302, "/mock-2/index.html"}} {
		t.Run(tc.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			s.ServeHTTP(w, httptest.NewRequest("GET", tc.path, nil))
			if w.Code != tc.code || !strings.Contains(w.Body.String()+w.Header().Get("Location"), tc.body) {
				t.Fatalf("%s: %d %s %s", tc.path, w.Code, w.Header(), w.Body.String())
			}
		})
	}
	if err := p.Use("mock-1"); err != nil {
		t.Fatal(err)
	}
	s.refresh(t.Context())
	w := httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest("GET", "/mock-2/index.html?state=empty", nil))
	if w.Header().Get("Location") != "/mock-1/index.html?state=empty" {
		t.Fatal(w.Header())
	}
	if err := os.WriteFile(filepath.Join(dir, "fixtures.json"), []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
	s.refresh(t.Context())
	w = httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest("GET", "/__couch_mock/state", nil))
	if w.Code != 503 || !strings.Contains(w.Body.String(), "error") {
		t.Fatalf("error state: %d %s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest("POST", "/fixtures.json", nil))
	if w.Code != 405 {
		t.Fatal(w.Code)
	}
}

func TestHTTPStateNeverScans(t *testing.T) {
	p, _ := serverProject(t)
	s, err := NewServer(t.Context(), p, "test")
	if err != nil {
		t.Fatal(err)
	}
	s.scan = func(context.Context) (Snapshot, error) {
		t.Fatal("cached HTTP read scanned project")
		return Snapshot{}, nil
	}
	for range 16 {
		s.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/__couch_mock/state", nil))
	}
}

func TestServeLifecycle(t *testing.T) {
	_, dir := serverProject(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	ready := make(chan string, 1)
	out := &urlWriter{ready: ready}
	done := make(chan error, 1)
	go func() { done <- Serve(ctx, dir, 0, out) }()
	var url string
	select {
	case url = <-ready:
	case <-time.After(10 * time.Second):
		t.Fatal("no URL")
	}
	res, err := http.Get(strings.TrimSpace(url))
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, res.Body)
	res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatal(res.Status)
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("server failed to join")
	}
	if res, err := http.Get(strings.TrimSpace(url)); err == nil {
		res.Body.Close()
		t.Fatal("listener survives Serve")
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	var buf bytes.Buffer
	if err := Serve(t.Context(), dir, ln.Addr().(*net.TCPAddr).Port, &buf); err == nil || buf.Len() != 0 {
		t.Fatalf("bind refusal %v output %q", err, &buf)
	}
}

type urlWriter struct{ ready chan string }

func (w *urlWriter) Write(p []byte) (int, error) {
	select {
	case w.ready <- string(p):
	default:
	}
	return len(p), nil
}

func TestHTTPRejectsUntaggedHTMLAliases(t *testing.T) {
	p, dir := serverProject(t)
	for _, name := range []string{"raw.HTML", "raw.htm"} {
		if err := os.WriteFile(filepath.Join(dir, "mock-2", name), []byte("<html><body>alias</body></html>"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	s, err := NewServer(t.Context(), p, "test")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"raw.HTML", "raw.htm"} {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, httptest.NewRequest("GET", "/mock-2/"+name, nil))
		if w.Code != http.StatusUnsupportedMediaType {
			t.Fatalf("untagged HTML %s: %d", name, w.Code)
		}
	}
}
