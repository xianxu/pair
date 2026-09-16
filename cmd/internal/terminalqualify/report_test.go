package terminalqualify

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestReportQualificationRequiresExactCoverage(t *testing.T) {
	for _, tc := range []struct {
		name             string
		results          []Result
		valid, qualified bool
	}{
		{"pass", []Result{{ID: "a", Status: Pass}}, true, true},
		{"fail", []Result{{ID: "a", Status: Fail}}, true, false},
		{"uncovered", []Result{{ID: "a", Status: NotCovered}}, true, false},
		{"missing", nil, false, false},
		{"duplicate", []Result{{ID: "a", Status: Pass}, {ID: "a", Status: Pass}}, false, false},
		{"foreign", []Result{{ID: "b", Status: Pass}}, false, false},
		{"unknown", []Result{{ID: "a", Status: "unknown"}}, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := Report{Required: []string{"a"}, Results: tc.results}
			if (r.Validate() == nil) != tc.valid {
				t.Fatalf("Validate=%v", r.Validate())
			}
			if r.Qualified() != tc.qualified {
				t.Fatalf("Qualified=%v", r.Qualified())
			}
		})
	}
}
func TestCompareDoesNotTruncatePredicate(t *testing.T) {
	expected := Observation{"cell:0,0": "hello", "cursor": "5,0"}
	if Compare(expected, Observation{"cell:0,0": "hello", "cursor": "5,0", "extra": "ok"}) != "" {
		t.Fatal("equal subset rejected")
	}
	if Compare(expected, Observation{"cell:0,0": "hello", "cursor": "4,0"}) == "" {
		t.Fatal("missed cursor mismatch")
	}
	if Compare(Observation{"empty": ""}, Observation{}) == "" {
		t.Fatal("missing key equals empty")
	}
}

func TestBoundedDetailRetainsFailureAfterLongPrefix(t *testing.T) {
	prefix := strings.Repeat("x", 5000)
	detail := Compare(Observation{"value": prefix + "a"}, Observation{"value": prefix + "b"})
	if detail == "" {
		t.Fatal("comparison truncated before difference")
	}
	if got := boundedDetail(detail); len(got) > 4096 || !utf8.ValidString(got) {
		t.Fatalf("invalid bounded detail length=%d", len(got))
	}
	if got := boundedDetail(strings.Repeat("界", 2000)); len(got) > 4096 || !utf8.ValidString(got) {
		t.Fatal("split UTF-8 evidence")
	}
}

func TestBoundedEvidenceKeepsSmallObservation(t *testing.T) {
	original := Observation{"cursor": "1,2", "replies": "\x1b[0n", "glyph": "界"}
	got, truncated := boundedEvidence(original)
	if truncated || !reflect.DeepEqual(got, original) {
		t.Fatalf("got=%v truncated=%t", got, truncated)
	}
	got["cursor"] = "changed"
	if original["cursor"] != "1,2" {
		t.Fatal("evidence aliases original")
	}
}

func TestBoundedEvidenceLimitsEncodedSizeAndEntries(t *testing.T) {
	for _, original := range []Observation{
		{"value": strings.Repeat("界\x00", 5000)},
		{strings.Repeat("k", 5000): "value", "short": "kept"},
		{"invalid": string([]byte{0xff, 0xfe}), string([]byte{0xff}): "key"},
		func() Observation {
			m := Observation{}
			for i := 0; i < 1000; i++ {
				m[fmt.Sprintf("%04d", i)] = "x"
			}
			return m
		}(),
	} {
		got, truncated := boundedEvidence(original)
		if !truncated {
			t.Fatal("missing truncation metadata")
		}
		raw, err := json.Marshal(got)
		if err != nil || len(raw) > 4096 || len(got) > 64 {
			t.Fatalf("size=%d entries=%d err=%v", len(raw), len(got), err)
		}
		for k, v := range got {
			if !utf8.ValidString(k) || !utf8.ValidString(v) {
				t.Fatal("invalid UTF-8 evidence")
			}
		}
		again, flag := boundedEvidence(original)
		if !flag || !reflect.DeepEqual(got, again) {
			t.Fatal("nondeterministic evidence")
		}
	}
}
