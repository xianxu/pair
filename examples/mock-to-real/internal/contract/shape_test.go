package contract

import (
	"example.com/mock-to-real/internal/jsonvalue"
	"strings"
	"testing"
)

func TestRecursiveShape(t *testing.T) {
	s, e := Infer([]byte(`{"runs":[{"id":"fixture","count":1,"ready":true,"note":null}]}`))
	if e != nil {
		t.Fatal(e)
	}
	for _, a := range []string{`{"runs":[]}`, `{"runs":[{"id":"real","count":2,"ready":false,"note":null,"extra":1}],"extra":true}`} {
		if e := s.Match([]byte(a)); e != nil {
			t.Fatal(e)
		}
	}
	for _, a := range []string{`{"runs":[{"id":"real"}]}`, `{"runs":[{"id":1,"count":2,"ready":true,"note":null}]}`, `{"runs":[{"id":"real","count":2,"ready":true,"note":"x"}]}`, `{"runs":[{"id":"real","count":2,"ready":true,"note":null},false]}`} {
		if s.Match([]byte(a)) == nil {
			t.Fatalf("accepted mutation %s", a)
		}
	}
}
func TestBoundsAndAmbiguity(t *testing.T) {
	for _, a := range []string{`[]`, `[1,"x"]`, `[{"a":1},{"b":1}]`, `{"x":1,"x":2}`, `null null`, strings.Repeat("[", 65) + "0" + strings.Repeat("]", 65), `"` + strings.Repeat("x", 1<<20) + `"`} {
		if _, e := Infer([]byte(a)); e == nil {
			t.Fatalf("accepted %.80s", a)
		}
	}
	s, e := Infer([]byte(`[1,2]`))
	if e != nil {
		t.Fatal(e)
	}
	for _, a := range []string{`[true]`, `[null]`, `["2"]`, `[2] []`, strings.Repeat(" ", 1<<20) + `[]`} {
		if s.Match([]byte(a)) == nil {
			t.Fatalf("accepted %.80s", a)
		}
	}
}
func FuzzShape(f *testing.F) {
	for _, s := range []string{`{"runs":[{"id":"x"}]}`, `[]`, `null`, `[1,"x"]`, `{"a":1,"a":2}`} {
		f.Add([]byte(s), []byte(s))
	}
	f.Fuzz(func(t *testing.T, a, b []byte) {
		s, e := Infer(a)
		if e == nil {
			_ = s.Match(b)
		}
	})
}

// Count real traversal work rather than elapsed time. Each row has eight JSON
// nodes; the root object and runs array contribute two more. Shape comparisons
// during homogeneous-array inference count too, so repeated scans cannot hide
// behind the compatibility check.
func TestTraversalCountsScaleLinearly(t *testing.T) {
	var previousInfer, previousMatch int
	for _, n := range []int{128, 256} {
		rows := make([]string, n)
		for i := range rows {
			rows[i] = `{"id":"fixture","metadata":{"count":1,"ready":true},"tags":["a","b"]}`
		}
		raw := []byte(`{"runs":[` + strings.Join(rows, ",") + `]}`)
		value, e := jsonvalue.Decode(raw)
		if e != nil {
			t.Fatal(e)
		}
		inferVisits := 0
		s, e := inferWithVisit(value, func() { inferVisits++ })
		if e != nil {
			t.Fatal(e)
		}
		matchVisits := 0
		if e := s.matchWithVisit(value, "$", func() { matchVisits++ }); e != nil {
			t.Fatal(e)
		}
		nodes := 2 + 8*n
		if inferVisits < nodes || inferVisits > 2*nodes {
			t.Fatalf("N=%d: inference %d visits for %d nodes", n, inferVisits, nodes)
		}
		if matchVisits != nodes {
			t.Fatalf("N=%d: matching %d visits for %d nodes", n, matchVisits, nodes)
		}
		if previousInfer != 0 && (inferVisits*100 > previousInfer*205 || matchVisits*100 > previousMatch*205) {
			t.Fatalf("doubling N exceeded linear bound: infer %d -> %d, match %d -> %d", previousInfer, inferVisits, previousMatch, matchVisits)
		}
		t.Logf("N=%d nodes=%d infer=%d match=%d", n, nodes, inferVisits, matchVisits)
		previousInfer, previousMatch = inferVisits, matchVisits
	}
}
