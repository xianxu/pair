package termcmd

import (
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/ansi"
	"github.com/xianxu/pair/cmd/internal/textwidth"
)

// EVERY test here drives at least TWO tabs with a non-active one present.
//
// That is a standing requirement, not a style note: M2's Critical (BR-35)
// shipped because every test in that milestone drove a single active tab, so
// the whole active/inactive distinction was unobserved by the suite.

func TestActiveTabIsDistinguishableWithoutColour(t *testing.T) {
	r := RenderStrip(40, StripModel{
		Tabs:   []TabChip{{Name: "one"}, {Name: "two"}, {Name: "three"}},
		Active: 1,
	})
	plain := string(ansi.Strip([]byte(r.Body)))
	if !strings.Contains(plain, "[two]") {
		t.Fatalf("the active tab is not marked in PLAIN text: %q", plain)
	}
	for _, inactive := range []string{"[one]", "[three]"} {
		if strings.Contains(plain, inactive) {
			t.Fatalf("an inactive tab is marked like the active one: %q", plain)
		}
	}
	// Colour may decorate, but must not be the only signal: a colour-blind
	// operator, a mono terminal, and `ansi.Strip`ped logs all lose it.
	if plain == r.Body && strings.Count(plain, "[") != 1 {
		t.Fatalf("active marking is ambiguous in plain text: %q", plain)
	}
}

// Carried from #172's BR-32: spans are DISPLAY COLUMNS. A rune count puts every
// span after a wide character one column left of what was drawn -- and an
// all-ASCII suite stays green while it does, which is why this fixture is not
// ASCII.
func TestSpansAreDisplayColumnsNotRuneCounts(t *testing.T) {
	r := RenderStrip(40, StripModel{
		Tabs:   []TabChip{{Name: "日本語"}, {Name: "build"}},
		Active: 0,
	})
	if len(r.Spans) != 2 {
		t.Fatalf("got %d spans for 2 tabs: %+v", len(r.Spans), r.Spans)
	}
	plain := string(ansi.Strip([]byte(r.Body)))
	at := strings.Index(plain, "build")
	if at < 0 {
		t.Fatalf("the second tab was not drawn: %q", plain)
	}
	wantCol := textwidth.Width(plain[:at])
	if r.Spans[1].Start != wantCol {
		t.Fatalf("span start %d is a rune count; the column is %d (%q)",
			r.Spans[1].Start, wantCol, plain)
	}
	// And the wide tab's own span must be as wide as it drew.
	if got := r.Spans[0].End - r.Spans[0].Start; got != textwidth.Width("[日本語]") {
		t.Fatalf("wide tab span is %d columns, drew %d", got, textwidth.Width("[日本語]"))
	}
}

// Tab names come from the operator. An escape in one must not become an escape
// in our row -- the same policy as couch's status row, via the same package.
func TestATabNameCannotInjectEscapesOrControls(t *testing.T) {
	r := RenderStrip(40, StripModel{
		Tabs:   []TabChip{{Name: "a\x1b[31mred"}, {Name: "b\x0egarble"}},
		Active: 0,
	})
	for _, bad := range []string{"\x1b", "\x0e"} {
		if strings.Contains(r.Body, bad) {
			t.Fatalf("control byte %q from a tab name reached the row: %q", bad, r.Body)
		}
	}
	if !strings.Contains(r.Body, "red") || !strings.Contains(r.Body, "garble") {
		t.Fatalf("sanitising mangled the readable text: %q", r.Body)
	}
}

// A narrow pane must not silently drop the tab the operator is looking at.
func TestANarrowPaneKeepsTheActiveTabVisible(t *testing.T) {
	m := StripModel{
		Tabs:   []TabChip{{Name: "alpha"}, {Name: "bravo"}, {Name: "charlie"}},
		Active: 2,
	}
	for _, width := range []int{40, 20, 12, 9} {
		r := RenderStrip(width, m)
		plain := string(ansi.Strip([]byte(r.Body)))
		if textwidth.Width(plain) > width {
			t.Fatalf("width %d: drew %d columns: %q", width, textwidth.Width(plain), plain)
		}
		if !strings.Contains(plain, "charlie") && !strings.Contains(plain, "charli") &&
			!strings.Contains(plain, "char") {
			t.Fatalf("width %d dropped the ACTIVE tab entirely: %q", width, plain)
		}
	}
}

// Degenerate widths must not panic or produce a row that wraps.
func TestDegenerateWidthsDrawNothingRatherThanWrapping(t *testing.T) {
	for _, width := range []int{0, -1} {
		r := RenderStrip(width, StripModel{Tabs: []TabChip{{Name: "a"}, {Name: "b"}}})
		if r.Body != "" || len(r.Spans) != 0 {
			t.Fatalf("width %d drew %q with %d spans", width, r.Body, len(r.Spans))
		}
	}
}

// No tabs is a real state (the last one just closed) and must not panic.
func TestNoTabsRendersNothing(t *testing.T) {
	if r := RenderStrip(40, StripModel{}); r.Body != "" {
		t.Fatalf("an empty model drew %q", r.Body)
	}
}

// An out-of-range Active index must not panic or mark the wrong tab: it comes
// from a mutable mux whose tabs can close between render and paint.
func TestAnOutOfRangeActiveIndexMarksNothing(t *testing.T) {
	for _, active := range []int{-1, 2, 99} {
		r := RenderStrip(40, StripModel{
			Tabs:   []TabChip{{Name: "one"}, {Name: "two"}},
			Active: active,
		})
		plain := string(ansi.Strip([]byte(r.Body)))
		if strings.Contains(plain, "[") {
			t.Fatalf("active=%d marked a tab: %q", active, plain)
		}
		if !strings.Contains(plain, "one") || !strings.Contains(plain, "two") {
			t.Fatalf("active=%d dropped tabs: %q", active, plain)
		}
	}
}
