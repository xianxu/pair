package app

import (
	"encoding/json"
	"example.com/mock-to-real/internal/contract"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestRealAppAndContract(t *testing.T) {
	h, e := New([]Screen{{"index", "/", "real"}, {"runs", "/runs", "real"}, {"detail", "/runs/detail", "mock"}})
	if e != nil {
		t.Fatal(e)
	}
	server := httptest.NewServer(h)
	defer server.Close()
	res, e := http.Get(server.URL + "/api/runs")
	if e != nil {
		t.Fatal(e)
	}
	defer res.Body.Close()
	var actual json.RawMessage
	if e := json.NewDecoder(res.Body).Decode(&actual); e != nil {
		t.Fatal(e)
	}
	fixture, e := os.ReadFile("../../mocks/fixtures.json")
	if e != nil {
		t.Fatal(e)
	}
	var fields map[string]json.RawMessage
	if e := json.Unmarshal(fixture, &fields); e != nil {
		t.Fatal(e)
	}
	s, e := contract.Infer(fields["runs"])
	if e != nil {
		t.Fatal(e)
	}
	if e := s.Match(actual); e != nil {
		t.Fatal(e)
	}
	if string(actual) == string(fields["runs"]) {
		t.Fatal("real API must own independent data")
	}
	if s.Match([]byte(`[{"id":false,"name":"bad","status":"Ready"}]`)) == nil {
		t.Fatal("negative control accepted")
	}
	for _, c := range []struct {
		p    string
		code int
		want string
	}{{"/", 200, "Not implemented"}, {"/runs", 200, "Not implemented"}, {"/runs/detail", 200, "Not implemented yet"}, {"/fixtures.json", 404, "404"}, {"/reload", 404, "404"}, {"/version", 404, "404"}, {"/mocks/mock-1/detail.html", 404, "404"}} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", c.p, nil))
		if w.Code != c.code || !strings.Contains(w.Body.String(), c.want) {
			t.Fatalf("%s: %d %s", c.p, w.Code, w.Body.String())
		}
	}
}
func TestMissingRealHandler(t *testing.T) {
	_, e := New([]Screen{{"detail", "/runs/detail", "real"}})
	if e == nil || !strings.Contains(e.Error(), "detail") {
		t.Fatalf("%v", e)
	}
}

func TestPreparedRoutesAndNavigationOwnSnapshot(t *testing.T) {
	screens := []Screen{{"index", "/", "real"}, {"detail<&", "/runs/detail", "mock"}}
	h, e := New(screens)
	if e != nil {
		t.Fatal(e)
	}
	screens[0] = Screen{"changed", "/rogue", "mock"}
	screens[1] = Screen{"changed", "/other", "real"}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/runs/detail", nil))
	body := w.Body.String()
	if w.Code != 200 || !strings.Contains(body, "Not implemented yet") || !strings.Contains(body, "detail&lt;&amp;") || !strings.Contains(body, `href="/"`) {
		t.Fatalf("lost prepared snapshot: %d %s", w.Code, body)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/rogue", nil))
	if w.Code != 404 {
		t.Fatalf("caller changed route table: %d", w.Code)
	}
}
