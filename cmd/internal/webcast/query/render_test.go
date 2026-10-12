package query

import (
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast/notes"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func pageAt(t *testing.T, fixture string, at int64, sel string) string {
	t.Helper()
	r, err := recording.ReadDir(recording.Fixture(fixture))
	if err != nil {
		t.Fatal(err)
	}
	q := New(r.Records, notes.FakeModel{})
	q.PageLines = 100000
	page, err := q.DOMAt(at, sel)
	if err != nil {
		t.Fatal(err)
	}
	return page
}

// The page shows what the observer knows that attributes don't: typed
// values, focus and scroll, as of t.
func TestPageOverlaysObserverState(t *testing.T) {
	recs, _ := ContractRecording()
	q := New(recs, notes.FakeModel{})
	for at, want := range map[int64][]string{
		1200:  {`@1 <input id=name> {focused}`},
		1600:  {`@1 <input id=name> {value "Ada", focused}`},
		10000: {`#document http://app/form {scrolled 0,100}`, `@3 <button disabled id=save> Save`},
	} {
		page, err := q.DOMAt(at, "")
		if err != nil {
			t.Fatal(err)
		}
		for _, w := range want {
			if !strings.Contains(page, w) {
				t.Errorf("t=%d: want %q in\n%s", at, w, page)
			}
		}
	}
}

// Script and style bodies, the head, and SVG internals fold to one line;
// a closed shadow root and a cross-origin frame's document are shown.
func TestPageFoldsNoiseAndShowsHiddenTrees(t *testing.T) {
	page := pageAt(t, "stress", 3000, "")
	for _, want := range []string{"#shadow-root (closed)", "#document http", " elements inside)", "<head> (title "} {
		if !strings.Contains(page, want) {
			t.Errorf("want %q in the stress page", want)
		}
	}
	for _, line := range strings.Split(page, "\n") {
		if strings.Contains(line, "<script") && !strings.HasSuffix(line, "chars)") && strings.Contains(line, "{") {
			t.Errorf("a script body leaked: %s", line)
		}
		if len([]rune(line)) > 400 {
			t.Errorf("a line over 400 characters: %.80s…", line)
		}
	}
	if svg := pageAt(t, "stress", 3000, "svg"); !strings.Contains(svg, "<text") {
		t.Errorf("a selected svg is shown whole:\n%s", svg)
	}
}

func TestPageKeepsToItsBudget(t *testing.T) {
	r, err := recording.ReadDir(recording.Fixture("blog"))
	if err != nil {
		t.Fatal(err)
	}
	st, err := recording.Reduce(r.Records, 2500)
	if err != nil {
		t.Fatal(err)
	}
	page := Page(st, st.Top, 40)
	lines := strings.Split(strings.TrimSuffix(page, "\n"), "\n")
	if len(lines) != 41 || !strings.HasPrefix(lines[40], "… ") || !strings.HasSuffix(lines[40], "more lines; narrow it with a selector (--sel)") {
		t.Errorf("%d lines, last %q", len(lines), lines[len(lines)-1])
	}
}
