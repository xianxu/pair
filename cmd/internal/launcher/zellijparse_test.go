package launcher

import (
	"reflect"
	"testing"
)

func TestSessionRowState(t *testing.T) {
	// Realistic --no-formatting rows (name + [created ...] + status suffix).
	raw := "pair-work [Created 1h ago]\n" +
		"pair-old (EXITED - attach to resurrect)\n" +
		"other-thing [Created 2h ago]\n"
	cases := []struct {
		session     string
		wantPresent bool
		wantExited  bool
	}{
		{"pair-work", true, false},     // running/detached → blocks
		{"pair-old", true, true},       // EXITED → clear + reusable
		{"pair-missing", false, false}, // absent → reuse free
		{"other-thing", true, false},
	}
	for _, tc := range cases {
		present, exited := sessionRowState(raw, tc.session)
		if present != tc.wantPresent || exited != tc.wantExited {
			t.Errorf("sessionRowState(%q) = (%v,%v), want (%v,%v)", tc.session, present, exited, tc.wantPresent, tc.wantExited)
		}
	}
	// A tag that is a prefix of a real row must NOT match on the first field.
	if present, _ := sessionRowState(raw, "pair"); present {
		t.Fatalf("prefix 'pair' must not match 'pair-work'")
	}
}

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
	if !sessionNameRejected("error: session name must be less than 0 characters") {
		t.Fatal("should detect the too-long rejection")
	}
	if sessionNameRejected("There is no active session!") {
		t.Fatal("a not-found error is not a name-length rejection")
	}
}
