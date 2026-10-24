package jsonstrict

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestScanPolicies(t *testing.T) {
	for _, tc := range []struct {
		raw   string
		opts  Options
		valid bool
	}{
		{`{"a":1,"A":2}`, Options{}, true},
		{`{"a":1,"A":2}`, Options{FoldKeys: true}, false},
		{`{"s":1,"ſ":2}`, Options{FoldKeys: true}, false},
		{`{"K":1,"K":2}`, Options{FoldKeys: true}, false},
		{`{"a":1,"\u0061":2}`, Options{}, false},
		{`{"future":1e9999}`, Options{}, false},
		{`{"future":1e9999}`, Options{PreserveNumbers: true}, true},
		{`{} {}`, Options{}, false},
		{`{"x":[{"a":1,"a":2}]}`, Options{}, false},
		{strings.Repeat("[", 64) + "0" + strings.Repeat("]", 64), Options{MaxDepth: 64}, true},
		{strings.Repeat("[", 65) + "0" + strings.Repeat("]", 65), Options{MaxDepth: 64}, false},
	} {
		t.Run(fmt.Sprintf("%s/%+v", tc.raw, tc.opts), func(t *testing.T) {
			if err := Scan([]byte(tc.raw), tc.opts); (err == nil) != tc.valid {
				t.Fatalf("valid=%v error=%v", tc.valid, err)
			}
		})
	}
}
func TestScanVisitsNestedKeysAndPropagatesError(t *testing.T) {
	var got []string
	if err := Scan([]byte(`{"future":[{"x":1}],"z":2}`), Options{VisitKey: func(depth int, key string) error { got = append(got, fmt.Sprintf("%d:%s", depth, key)); return nil }}); err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, ",") != "0:future,2:x,0:z" {
		t.Fatal(got)
	}
	sentinel := errors.New("visitor stopped")
	if err := Scan([]byte(`{"x":1}`), Options{VisitKey: func(int, string) error { return sentinel }}); !errors.Is(err, sentinel) {
		t.Fatal(err)
	}
}
func TestScanWorkScalesWithBytesAndKeys(t *testing.T) {
	var previous work
	for _, n := range []int{128, 256} {
		rows := make([]string, n)
		for i := range rows {
			rows[i] = `{"escaped\u006bey":[{"ſ":1e9999}]}`
		}
		raw := []byte("[" + strings.Join(rows, ",") + "]")
		visits := 0
		var w work
		if err := scan(raw, Options{PreserveNumbers: true, FoldKeys: true, VisitKey: func(int, string) error { visits++; return nil }}, &w); err != nil {
			t.Fatal(err)
		}
		if visits != 2*n || w.keys != 2*n || w.bytes < len(raw) || w.bytes > 30*len(raw) {
			t.Fatalf("n=%d visits=%d work=%+v bytes=%d", n, visits, w, len(raw))
		}
		if previous.bytes != 0 && (w.bytes > 2*previous.bytes+30 || w.keys != 2*previous.keys) {
			t.Fatalf("nonlinear work: %+v -> %+v", previous, w)
		}
		previous = w
	}
}

func TestScanFoldedVisitorReceivesWireSpelling(t *testing.T) {
	var got string
	if err := Scan([]byte(`{"ſ":0}`), Options{FoldKeys: true, VisitKey: func(_ int, key string) error { got = key; return nil }}); err != nil {
		t.Fatal(err)
	}
	if got != "ſ" {
		t.Fatalf("visitor spelling=%q", got)
	}
}
