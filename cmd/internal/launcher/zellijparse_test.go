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
	if !sessionNameRejected("error: session name must be less than 0 characters") {
		t.Fatal("should detect the too-long rejection")
	}
	if sessionNameRejected("There is no active session!") {
		t.Fatal("a not-found error is not a name-length rejection")
	}
}

func TestPairSessionNames(t *testing.T) {
	// --short output: clean names, one per line, unsorted + a dup + non-pair rows.
	short := "pair-work\nother\npair-brain\npair-work\n\nscratch\n"
	got := pairSessionNames(short)
	want := []string{"pair-brain", "pair-work"} // pair-* only, sorted, deduped
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("pairSessionNames = %v, want %v", got, want)
	}
	if names := pairSessionNames(""); names != nil {
		t.Fatalf("empty input → nil, got %v", names)
	}
}

func TestClientCount(t *testing.T) {
	cases := []struct {
		raw  string
		want int
	}{
		{"", 0},                                  // query failed → detached
		{"CLIENT_ID ...\n", 0},                   // header only → 0 clients
		{"CLIENT_ID ...\nc1 ...\n", 1},           // one client
		{"CLIENT_ID ...\nc1 ...\nc2 ...\n", 2},   // two clients
		{"CLIENT_ID ...\nc1 ...\n\nc2 ...\n", 2}, // blank lines don't count
	}
	for _, c := range cases {
		if got := parseClientCount(c.raw); got != c.want {
			t.Errorf("parseClientCount(%q) = %d, want %d", c.raw, got, c.want)
		}
	}
}
