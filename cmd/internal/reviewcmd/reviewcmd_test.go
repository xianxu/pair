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

// The reap attributes a review embed by the file its tag's state names (#583),
// so the reader has to agree with the writer on every file the writer can
// record, and has to report "no file" rather than guess on anything else.
func TestTargetFileOf(t *testing.T) {
	t.Parallel()
	if got, ok := TargetFileOf(targetJSON(`/r/doc "q".md`, "ready", "sid1")); !ok || got != `/r/doc "q".md` {
		t.Fatalf("TargetFileOf(written) = %q, %v; want the file back", got, ok)
	}
	for name, content := range map[string]string{
		"no file":       targetJSON("", "proposed", "sid1"),
		"malformed":     `{"file":`,
		"not an object": `[]`,
		"empty":         "",
	} {
		if got, ok := TargetFileOf(content); ok || got != "" {
			t.Fatalf("%s: TargetFileOf = %q, %v; want no file", name, got, ok)
		}
	}
}
