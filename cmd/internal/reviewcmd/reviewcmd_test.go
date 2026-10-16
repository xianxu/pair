package reviewcmd

import (
	"encoding/json"
	"testing"
)

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"/repo/doc.md":        "doc",
		"Foo Bar.md":          "foo-bar",
		"x.test.md":           "x-test",
		"/a/--Weird__Name.md": "weird-name",
		"plain":               "plain",
	}
	for in, want := range cases {
		if got := slugify(in); got != want {
			t.Errorf("slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTargetJSON(t *testing.T) {
	s := targetJSON(`/r/doc "q".md`, "ready", "sid1")
	var d targetDoc
	if err := json.Unmarshal([]byte(s), &d); err != nil {
		t.Fatalf("not valid JSON: %v (%s)", err, s)
	}
	if d.File != `/r/doc "q".md` || d.Status != "ready" || d.Session != "sid1" {
		t.Fatalf("round-trip = %+v", d)
	}
}
