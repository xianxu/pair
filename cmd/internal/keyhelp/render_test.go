package keyhelp

import (
	"strings"
	"testing"
)

func TestRenderAlignsWithinSection(t *testing.T) {
	got := Render([]Section{{
		Title: "Draft",
		Bindings: []Binding{
			{Key: "Alt+⏎", Desc: "send buffer + clear"},
			{Key: "Shift+Alt+⏎", Desc: "append, no send"},
		},
	}})
	want := "Draft\n" +
		"  Alt+⏎        send buffer + clear\n" +
		"  Shift+Alt+⏎  append, no send\n"
	if got != want {
		t.Fatalf("Render =\n%q\nwant\n%q", got, want)
	}
}

// Alignment is per-section, so a long key in one section must not indent another.
func TestRenderAlignsPerSectionNotGlobally(t *testing.T) {
	got := Render([]Section{
		{Title: "A", Bindings: []Binding{{Key: "Alt+h", Desc: "help"}}},
		{Title: "B", Bindings: []Binding{{Key: "Ctrl+Alt+n", Desc: "reload"}}},
	})
	want := "A\n  Alt+h  help\n\nB\n  Ctrl+Alt+n  reload\n"
	if got != want {
		t.Fatalf("Render =\n%q\nwant\n%q", got, want)
	}
}

// Width is measured in DISPLAY COLUMNS, not bytes: a wide glyph is one rune, two
// columns, and byte-length padding would visibly misalign the column (#130's bug,
// one layer up).
func TestRenderPadsByDisplayWidthNotBytes(t *testing.T) {
	got := Render([]Section{{Title: "T", Bindings: []Binding{
		{Key: "📁x", Desc: "wide"},
		{Key: "abc", Desc: "narrow"},
	}}})
	// "📁x" is 3 columns, "abc" is 3 — so both pad to the same column.
	want := "T\n  📁x  wide\n  abc  narrow\n"
	if got != want {
		t.Fatalf("Render =\n%q\nwant\n%q", got, want)
	}
}

// Centering is done in Go, not by bin/pair-help's awk (which measured BYTES and
// therefore skewed on glyph keys — #132 PQ-6).
func TestCenterIndentsByColumnsAndLeavesBlankLinesAlone(t *testing.T) {
	got := Center("ab\n\ncd\n", 10)
	// widest line is 2 columns → pad (10-2)/2 = 4
	want := "    ab\n\n    cd\n"
	if got != want {
		t.Fatalf("Center =\n%q\nwant\n%q", got, want)
	}
}

func TestCenterIsANoOpWhenContentIsWiderThanTheTerminal(t *testing.T) {
	in := "aaaaaaaaaa\n"
	if got := Center(in, 4); got != in {
		t.Fatalf("Center over-wide = %q, want unchanged", got)
	}
}

func TestCenterMeasuresWidestLineInColumns(t *testing.T) {
	// The wide glyph makes this line 6 columns, not 5 runes / 8 bytes; a byte-based
	// measure would compute a negative-or-smaller pad and mis-centre the block.
	got := Center("📁pair\n", 10)
	if want := "  📁pair\n"; got != want {
		t.Fatalf("Center = %q, want %q", got, want)
	}
	if strings.HasPrefix(got, "   ") {
		t.Error("over-padded — width was not measured in columns")
	}
}
