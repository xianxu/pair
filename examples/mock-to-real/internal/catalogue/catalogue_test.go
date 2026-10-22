package catalogue

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestRead(t *testing.T) {
	good := `{"version":1,"future":{},"screens":[{"id":"index","route":"/","mock":"index.html","state":"real","future":2}]}`
	got, e := Read(strings.NewReader(good))
	if e != nil || len(got) != 1 || got[0].ID != "index" {
		t.Fatalf("%v %v", got, e)
	}
	for _, s := range []string{`{"version":1,"version":1,"screens":[]}`, `{"version":null,"screens":[]}`, `{"version":1,"screens":null}`, `{"version":1,"screens":[{"id":"a","route":"/","mock":"a.html","state":"mock"},{"id":"a","route":"/b","mock":"b.html","state":"real"}]}`, `{"version":1,"screens":[{"id":"a","route":"/","mock":"../a.html","state":"real"}]}`, `{"version":1,"screens":[{"id":"a","route":"/","mock":"a.html","state":"typo"}]}`, `{"version":1,"screens":[{"id":"a","route":"/","mock":null,"state":"real"}]}`} {
		if _, e := Read(strings.NewReader(s)); e == nil {
			t.Fatalf("accepted %s", s)
		}
	}
}

func encoded(rows []map[string]any) string {
	b, _ := json.Marshal(map[string]any{"version": 1, "screens": rows})
	return string(b)
}
func rootRow() map[string]any {
	return map[string]any{"id": "index", "route": "/", "mock": "index.html", "state": "real"}
}
func detailRow() map[string]any {
	return map[string]any{"id": "detail", "route": "/runs/detail", "mock": "detail.html", "state": "mock"}
}
func TestHostContractParityClasses(t *testing.T) {
	for _, route := range []string{"/__couch_mock", "/__couch_mockish/x", "/mock-1", "/fixtures.json/x", "/screens.json", "/design.md", "/.hidden/x", "/%72uns", "/a b", "/a\tb", "/a\u00a0b", "/a:b", "/a*", "/a{id}", "/a?x", "/a#x", "/a\\b", "/a//b", "/a/../b", "/a/"} {
		t.Run("route_"+route, func(t *testing.T) {
			r := detailRow()
			r["route"] = route
			if _, e := Read(strings.NewReader(encoded([]map[string]any{rootRow(), r}))); e == nil {
				t.Fatalf("accepted route %q", route)
			}
		})
	}
	for _, mock := range []string{"detail.txt", ".html", "nested/.html", "../detail.html", "a/../detail.html", "/detail.html", "detail.html?x", "detail.html#x", "a\\detail.html", "a\x00.html"} {
		t.Run("mock_"+mock, func(t *testing.T) {
			r := detailRow()
			r["mock"] = mock
			if _, e := Read(strings.NewReader(encoded([]map[string]any{rootRow(), r}))); e == nil {
				t.Fatalf("accepted mock %q", mock)
			}
		})
	}
	for _, field := range []string{"id", "route", "mock"} {
		t.Run("duplicate_"+field, func(t *testing.T) {
			a, b := detailRow(), detailRow()
			b["id"] = "other"
			b["route"] = "/other"
			b["mock"] = "other.html"
			b[field] = a[field]
			if _, e := Read(strings.NewReader(encoded([]map[string]any{rootRow(), a, b}))); e == nil {
				t.Fatalf("accepted duplicate %s", field)
			}
		})
	}
	for _, id := range []string{"has space", "has\ncontrol", strings.Repeat("x", 129)} {
		r := detailRow()
		r["id"] = id
		if _, e := Read(strings.NewReader(encoded([]map[string]any{rootRow(), r}))); e == nil {
			t.Fatalf("accepted id %q", id)
		}
	}
	for _, rows := range [][]map[string]any{{}, {detailRow()}, {func() map[string]any { r := rootRow(); r["mock"] = "other.html"; return r }()}} {
		if _, e := Read(strings.NewReader(encoded(rows))); e == nil {
			t.Fatalf("accepted missing/invalid root: %v", rows)
		}
	}
	rows := []map[string]any{rootRow()}
	for i := 1; i <= 256; i++ {
		r := detailRow()
		r["id"] = fmt.Sprint("screen", i)
		r["route"] = fmt.Sprint("/screen", i)
		r["mock"] = fmt.Sprint("screen", i, ".html")
		rows = append(rows, r)
	}
	if _, e := Read(strings.NewReader(encoded(rows[:256]))); e != nil {
		t.Fatalf("256 screens: %v", e)
	}
	if _, e := Read(strings.NewReader(encoded(rows))); e == nil {
		t.Fatal("accepted 257 screens")
	}
	good := encoded([]map[string]any{rootRow(), detailRow()})
	exact := good + strings.Repeat(" ", (256<<10)-len(good))
	if _, e := Read(strings.NewReader(exact)); e != nil {
		t.Fatalf("exact byte bound: %v", e)
	}
	if _, e := Read(strings.NewReader(exact + " ")); e == nil {
		t.Fatal("accepted >256KiB")
	}
	r := detailRow()
	r["mock"] = "nested/detail.html"
	r["route"] = "/nested/detail-1_2.html"
	r["future"] = map[string]any{"arbitrary": []any{true, nil, "x"}}
	if _, e := Read(strings.NewReader(encoded([]map[string]any{rootRow(), r}))); e != nil {
		t.Fatalf("valid nested extension: %v", e)
	}
}
