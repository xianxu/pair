package mockserve

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

const frontCatalogue = `{"version":1,"screens":[{"id":"index","route":"/","mock":"index.html","state":"real"},{"id":"detail","route":"/detail","mock":"detail.html","state":"mock"}]}`

func frontProject(t *testing.T) (*Project, string) {
	t.Helper()
	p, dir := serverProject(t)
	for name, body := range map[string]string{"screens.json": frontCatalogue, "mock-2/detail.html": `<h1>Mock detail</h1><a href="/">Home</a>`} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return p, dir
}

func TestFrontRoutesAndErrorOwnership(t *testing.T) {
	p, _ := frontProject(t)
	var requests atomic.Int64
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.URL.Path == "/fail" {
			http.Error(w, "real failure", 500)
			return
		}
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, "http://"+r.Host+"/detail?from=real", 302)
			return
		}
		w.Header().Set("Content-Security-Policy", "default-src 'none'")
		w.Header().Set("Set-Cookie", "session=real; Path=/")
		body, _ := io.ReadAll(r.Body)
		_, _ = io.WriteString(w, "REAL "+r.Method+" "+r.URL.RequestURI()+" "+string(body))
	}))
	defer upstream.Close()
	s, err := newHybridServer(t.Context(), p, "front-test")
	if err != nil {
		t.Fatal(err)
	}
	f, err := newDevelopmentFront(t.Context(), s, upstream.URL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.Close() })
	for _, tc := range []struct {
		method, target, body string
		code                 int
		want                 string
	}{
		{"GET", "/", "", 200, "REAL GET /"},
		{"GET", "/detail?q=empty", "", 302, "/mock-2/detail.html?q=empty"},
		{"GET", "/mock-2/detail.html", "", 200, "MOCK"},
		{"GET", "/mock-1/index.html?q=x", "", 302, "/?q=x"},
		{"POST", "/api/save?q=x", "payload", 200, "REAL POST /api/save?q=x payload"},
		{"GET", "/framework/asset.js", "", 200, "REAL GET /framework/asset.js"},
		{"GET", "/fail", "", 500, "real failure"},
		{"GET", "/redirect", "", 302, "/detail?from=real"},
		{"POST", "/detail", "payload", 405, ""},
	} {
		t.Run(tc.method+tc.target, func(t *testing.T) {
			w := httptest.NewRecorder()
			f.ServeHTTP(w, httptest.NewRequest(tc.method, tc.target, strings.NewReader(tc.body)))
			if w.Code != tc.code || !strings.Contains(w.Body.String()+w.Header().Get("Location"), tc.want) {
				t.Fatalf("%d %s %s", w.Code, w.Header(), w.Body.String())
			}
			if strings.HasPrefix(w.Body.String(), "REAL") && strings.Contains(w.Body.String(), "MOCK") {
				t.Fatal("real response decorated")
			}
		})
	}
	before := requests.Load()
	for _, target := range []string{"/screens.json", "/design.md", "/.mock-current", "/__couch_mock/private", "/mock-bad/anything", "/mock-2/no.html", "/mock-2/%2fprivate", "/x/../secret"} {
		w := httptest.NewRecorder()
		f.ServeHTTP(w, httptest.NewRequest("GET", target, nil))
		if w.Code != 404 {
			t.Errorf("%s: %d", target, w.Code)
		}
	}
	if requests.Load() != before {
		t.Fatal("private or malformed mock path reached upstream")
	}
}

func TestHybridMissingDocumentRepairAndNoRequestScans(t *testing.T) {
	p, dir := frontProject(t)
	if err := os.Remove(filepath.Join(dir, "mock-2/detail.html")); err != nil {
		t.Fatal(err)
	}
	if _, err := newHybridServer(t.Context(), p, "test"); err == nil {
		t.Fatal("missing declared mock accepted at startup")
	}
	if err := os.WriteFile(filepath.Join(dir, "mock-2/detail.html"), []byte("detail restored"), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := newHybridServer(t.Context(), p, "test")
	if err != nil {
		t.Fatal(err)
	}
	f, err := newDevelopmentFront(t.Context(), s, "http://127.0.0.1:1")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := os.Remove(filepath.Join(dir, "mock-2/detail.html")); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	f.ServeHTTP(w, httptest.NewRequest("GET", "/mock-2/detail.html", nil))
	if w.Code != 503 || w.Header().Get("Location") != "" {
		t.Fatalf("missing document fell back: %d %s", w.Code, w.Header())
	}
	s.refresh(t.Context())
	if s.observation().Error == "" {
		t.Fatal("missing document not published as repair state")
	}
	if err := os.WriteFile(filepath.Join(dir, "mock-2/detail.html"), []byte("detail restored"), 0600); err != nil {
		t.Fatal(err)
	}
	s.refresh(t.Context())
	if s.observation().Error != "" {
		t.Fatal("repair did not recover")
	}
	s.scan = func(context.Context) (Snapshot, error) { t.Fatal("request scanned"); return Snapshot{}, nil }
	for _, n := range []int{8, 16} {
		for range n {
			f.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/mock-2/detail.html", nil))
		}
	}
}

func TestRealOriginValidation(t *testing.T) {
	canonical, err := parseRealOrigin("http://localhost:03000")
	if err != nil || canonical.Port() != "3000" {
		t.Fatalf("port must canonicalize for self-target comparison: %v %v", canonical, err)
	}
	for _, raw := range []string{"https://localhost:3000", "http://example.com:3000", "http://user@localhost:3000", "http://localhost:3000/app", "http://localhost:3000/?q=x", "http://localhost", "http://127.0.0.1:0", "http://localhost:3000#fragment"} {
		if _, err := parseRealOrigin(raw); err == nil {
			t.Errorf("accepted %q", raw)
		}
	}
	for _, raw := range []string{"http://localhost:3000", "http://127.0.0.1:3000", "http://[::1]:3000"} {
		if _, err := parseRealOrigin(raw); err != nil {
			t.Errorf("%s: %v", raw, err)
		}
	}
}
