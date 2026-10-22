package mockserve

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

const testCatalogue = `{"version":1,"screens":[{"id":"home","route":"/","mock":"index.html","state":"real"},{"id":"runs","route":"/runs","mock":"runs.html","state":"mock"},{"id":"detail","route":"/runs/detail","mock":"nested/detail.html","state":"mock"}]}`

func TestCatalogue(t *testing.T) {
	c, err := ParseCatalogue([]byte(testCatalogue))
	if err != nil {
		t.Fatal(err)
	}
	if real, total := c.Progress(); real != 1 || total != 3 {
		t.Fatal(real, total)
	}
	if got := c.Report(); got != "Screen progress: 1/3 real\nMissing: runs /runs\nMissing: detail /runs/detail\n" {
		t.Fatal(got)
	}
	for _, s := range c.Screens() {
		if got, ok := c.ResolveRoute(s.Route); !ok || got != s {
			t.Fatal(got, ok)
		}
		if got, ok := c.ResolveMock(s.Mock); !ok || got != s {
			t.Fatal(got, ok)
		}
	}
	for _, p := range []string{"", "/absent", "/runs?query", "/mock-1/runs.html"} {
		if _, ok := c.ResolveRoute(p); ok {
			t.Fatal(p)
		}
	}
	if _, ok := c.ResolveMock("/runs.html"); ok {
		t.Fatal("accepted absolute alias")
	}
	c.Screens()[0].ID = "changed"
	c.Missing()[0].ID = "changed"
	if got, _ := c.ResolveRoute("/"); got.ID != "home" {
		t.Fatal("mutable screens")
	}
	if c.Missing()[0].ID != "runs" {
		t.Fatal("mutable missing screens")
	}
}

func catalogueRows(rows ...map[string]any) []byte {
	b, _ := json.Marshal(map[string]any{"version": 1, "screens": rows})
	return b
}
func catalogueRow(id, route, mock, state string) map[string]any {
	return map[string]any{"id": id, "route": route, "mock": mock, "state": state}
}

func TestCatalogueRejectsMalformed(t *testing.T) {
	bad := []string{"", "null", "[]", `{}`, `{"version":null,"screens":[]}`, `{"version":2,"screens":[]}`, `{"version":1.0,"screens":[]}`, `{"version":1,"screens":null}`, `{"version":1,"screens":[]}`, `{"version":1,"version":1,"screens":[]}`, testCatalogue + `{}`, strings.Replace(testCatalogue, `"id":"home"`, `"id":"home","id":"other"`, 1), strings.Replace(testCatalogue, `"version":1`, `"future":{"a":1,"a":2},"version":1`, 1)}
	for _, raw := range bad {
		if _, err := ParseCatalogue([]byte(raw)); err == nil {
			t.Errorf("accepted %q", raw)
		}
	}
	for _, key := range []string{"id", "route", "mock", "state"} {
		for _, value := range []any{nil, 1, true, []any{}, map[string]any{}, ""} {
			r := catalogueRow("home", "/", "index.html", "real")
			r[key] = value
			if _, err := ParseCatalogue(catalogueRows(r)); err == nil {
				t.Errorf("accepted %s=%v", key, value)
			}
		}
		r := catalogueRow("home", "/", "index.html", "real")
		delete(r, key)
		if _, err := ParseCatalogue(catalogueRows(r)); err == nil {
			t.Errorf("accepted absent %s", key)
		}
	}
	for _, route := range []string{"runs", "//runs", "/runs/", "/./runs", "/../runs", "/a/../runs", "/a//runs", "/runs?x", "/runs#x", "/a\\b", "/%2fruns", "/%2Eruns", "/runs/:id", "/runs/*", "/runs/{id}", "/__couch_mock", "/__couch_mockish", "/mock-other/x", "/fixtures.json", "/screens.json", "/.secret", "/design.md", "/a\x00b", "/a\nb", "/a b", strings.Repeat("/a", 1025)} {
		rows := []map[string]any{catalogueRow("home", "/", "index.html", "real"), catalogueRow("other", route, "other.html", "mock")}
		if _, err := ParseCatalogue(catalogueRows(rows...)); err == nil {
			t.Errorf("accepted route %q", route)
		}
	}
	for _, mock := range []string{"/x.html", "../x.html", "a/../x.html", "x", "x.html?x", "a\\b.html"} {
		if _, err := ParseCatalogue(catalogueRows(catalogueRow("home", "/", "index.html", "real"), catalogueRow("other", "/other", mock, "mock"))); err == nil {
			t.Errorf("accepted mock %q", mock)
		}
	}
	for _, rows := range [][]map[string]any{
		{catalogueRow("home", "/other", "other.html", "real")},
		{catalogueRow("home", "/", "other.html", "real")},
		{catalogueRow("home", "/", "index.html", "unknown")},
		{catalogueRow(strings.Repeat("x", 129), "/", "index.html", "real")},
		{catalogueRow("home", "/", "index.html", "real"), catalogueRow("home", "/other", "other.html", "mock")},
		{catalogueRow("home", "/", "index.html", "real"), catalogueRow("other", "/", "other.html", "mock")},
		{catalogueRow("home", "/", "index.html", "real"), catalogueRow("other", "/other", "index.html", "mock")},
	} {
		if _, err := ParseCatalogue(catalogueRows(rows...)); err == nil {
			t.Errorf("accepted rows %v", rows)
		}
	}
	if _, err := ParseCatalogue([]byte(strings.Repeat(" ", 256*1024) + testCatalogue)); err == nil {
		t.Fatal("unbounded bytes")
	}
	rows := []map[string]any{catalogueRow("home", "/", "index.html", "real")}
	for i := 1; i <= 256; i++ {
		rows = append(rows, catalogueRow(fmt.Sprint(i), fmt.Sprint("/s", i), fmt.Sprint(i, ".html"), "mock"))
	}
	if _, err := ParseCatalogue(catalogueRows(rows...)); err == nil {
		t.Fatal("unbounded screens")
	}
}

func TestCatalogueAdditiveFields(t *testing.T) {
	raw := strings.Replace(testCatalogue, `"version":1`, `"future":{"nested":[1,null]},"version":1`, 1)
	raw = strings.Replace(raw, `"id":"home"`, `"future":[{"x":true}],"id":"home"`, 1)
	if _, err := ParseCatalogue([]byte(raw)); err != nil {
		t.Fatal(err)
	}
}

func TestCatalogueLookupWork(t *testing.T) {
	for _, size := range []int{64, 128} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			rows := []map[string]any{catalogueRow("home", "/", "index.html", "real")}
			for i := 1; i < size; i++ {
				rows = append(rows, catalogueRow(fmt.Sprint(i), fmt.Sprint("/s", i), fmt.Sprint(i, ".html"), "real"))
			}
			c, err := ParseCatalogue(catalogueRows(rows...))
			if err != nil {
				t.Fatal(err)
			}
			screens := c.Screens()
			// Remove the ordered iteration sources from this private test copy.
			// Every lookup must still succeed; a request-time screen/missing scan
			// has no entries to inspect and therefore cannot produce the answer.
			lookup := *c
			lookup.screens = nil
			lookup.missing = nil
			for _, requests := range []int{size, 2 * size} {
				lookups := 0
				for i := 0; i < requests; i++ {
					s := screens[i%size]
					if got, ok := lookup.ResolveRoute(s.Route); !ok || got != s {
						t.Fatal("route lookup depended on scan", s)
					}
					lookups++
					if got, ok := lookup.ResolveMock(s.Mock); !ok || got != s {
						t.Fatal("mock lookup depended on scan", s)
					}
					lookups++
					if _, ok := lookup.ResolveRoute("/absent"); ok {
						t.Fatal("unknown route")
					}
					lookups++
					if _, ok := lookup.ResolveMock("absent.html"); ok {
						t.Fatal("unknown mock")
					}
					lookups++
				}
				if lookups != 4*requests {
					t.Fatalf("lookups %d for %d requests", lookups, requests)
				}
			}
		})
	}
}

func TestCatalogueCompleteAndLimit(t *testing.T) {
	rows := []map[string]any{catalogueRow("home", "/", "index.html", "real")}
	for i := 1; i < 256; i++ {
		rows = append(rows, catalogueRow(fmt.Sprint(i), fmt.Sprint("/s", i), fmt.Sprint(i, ".html"), "real"))
	}
	c, err := ParseCatalogue(catalogueRows(rows...))
	if err != nil {
		t.Fatal(err)
	}
	if real, total := c.Progress(); real != 256 || total != 256 {
		t.Fatal(real, total)
	}
	if len(c.Missing()) != 0 || c.Report() != "Screen progress: 256/256 real\n" {
		t.Fatal(c.Report())
	}
}

func FuzzParseCatalogue(f *testing.F) {
	for _, s := range []string{testCatalogue, "null", `{"version":1,"screens":[null]}`, strings.Replace(testCatalogue, "/runs", "/%2f", 1), strings.Replace(testCatalogue, `"version":1`, `"version":1,"version":2`, 1)} {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		c, err := ParseCatalogue(b)
		if err != nil {
			return
		}
		real, total := c.Progress()
		if real < 0 || real > total || total != len(c.Screens()) {
			t.Fatal("invalid progress")
		}
		for _, s := range c.Screens() {
			if got, ok := c.ResolveRoute(s.Route); !ok || got != s {
				t.Fatal("route mismatch")
			}
			if got, ok := c.ResolveMock(s.Mock); !ok || got != s {
				t.Fatal("alias mismatch")
			}
		}
	})
}
