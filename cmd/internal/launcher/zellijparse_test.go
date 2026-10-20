package launcher

import (
	"reflect"
	"testing"
)

func TestFamilyRows(t *testing.T) {
	raw := "pair-work [a]\npair-work-2 [b]\npair-workspace [c]\npair-other [d]\nnot-pair [e]\n"
	got := familyRows(raw, "pair-work")
	// Exact "pair-work" and "pair-work-2" match; "pair-workspace" does NOT
	// (no "-" boundary), nor "pair-other".
	want := []string{"pair-work [a]", "pair-work-2 [b]"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("familyRows = %v, want %v", got, want)
	}
	if rows := familyRows("", "pair-x"); rows != nil {
		t.Fatalf("empty input → nil, got %v", rows)
	}
}

func TestSessionNameRejected(t *testing.T) {
	for _, tc := range diagnosticMatrix() {
		t.Run(tc.name, func(t *testing.T) {
			if got := sessionNameRejected("pair-new", tc.status, tc.stdout, tc.stderr); got != tc.rejectsName {
				t.Fatalf("validator rejection=%v, want %v", got, tc.rejectsName)
			}
		})
	}
}

func TestSessionNameRejectedNumericBoundary(t *testing.T) {
	for _, tc := range []struct {
		limit string
		valid bool
	}{
		{"18446744073709551615", true},
		{"18446744073709551616", false},
	} {
		if got := sessionNameRejected("pair-new", 2, "", nativeNameValidator("pair-new", tc.limit)); got != tc.valid {
			t.Fatalf("native uint64 limit %q valid=%v, want %v", tc.limit, got, tc.valid)
		}
	}
}
