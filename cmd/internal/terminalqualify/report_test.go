package terminalqualify

import (
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
