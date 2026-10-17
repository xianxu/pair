package terminal

import (
	"fmt"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"
)

// slowValidateCell preserves the pre-optimization algorithm and deliberately
// does not call production text/link helpers or recognize ASCII shortcuts.
func slowValidateCell(cells []Cell, i int) error {
	c := cells[i]
	safe := func(s string) bool {
		if !utf8.ValidString(s) {
			return false
		}
		for _, r := range s {
			if unicode.IsControl(r) {
				return false
			}
		}
		return true
	}
	if !safe(c.Content) || len(c.Content) > MaxClusterBytes || len(c.Link.URL) > MaxLinkBytes || len(c.Link.Params) > MaxLinkBytes || !safe(c.Link.URL) || !safe(c.Link.Params) {
		return fmt.Errorf("terminal: unsafe cell %d", i)
	}
	if c.Width < 0 || c.Width > 2 {
		return fmt.Errorf("terminal: unsupported cell width at %d", i)
	}
	if c.Content != "" {
		cluster, width := ansi.FirstGraphemeCluster(c.Content, ansi.GraphemeWidth)
		if len(cluster) != len(c.Content) || width != c.Width {
			return fmt.Errorf("terminal: content does not fit cell %d", i)
		}
	}
	if c.Width == 0 && c.Content != "" {
		return fmt.Errorf("terminal: content in continuation at %d", i)
	}
	if c.Width == 2 && (i == len(cells)-1 || cells[i+1].Width != 0 || cells[i+1].Content != "") {
		return fmt.Errorf("terminal: broken wide cell at %d", i)
	}
	return nil
}

func cellValidationError(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func TestValidateCellASCIIExhaustiveAgainstSlow(t *testing.T) {
	links := []struct{ URL, Params string }{{}, {URL: "https://example.test", Params: "id=ok"}, {URL: "bad\x1b"}, {Params: "bad\n"}, {URL: string([]byte{0xff})}, {URL: strings.Repeat("a", MaxLinkBytes)}, {Params: strings.Repeat("a", MaxLinkBytes+1)}}
	for value := 0; value < 256; value++ {
		for width := -1; width <= 3; width++ {
			for _, link := range links {
				cells := []Cell{{Content: string([]byte{byte(value)}), Width: width, Link: link}, {}}
				got, want := cellValidationError(validateCell(cells, 0)), cellValidationError(slowValidateCell(cells, 0))
				if got != want {
					t.Fatalf("byte=%02x width=%d link=%q/%q: got %q want %q", value, width, link.URL, link.Params, got, want)
				}
			}
		}
	}
}

func TestValidateCellUnicodeAndNeighborsAgainstSlow(t *testing.T) {
	contents := []string{"", "a", "ab", "e\u0301", "界", "👩\u200d💻", "🇺🇸", "\u0301", "\ufe0f", "a\ufe0f", "\x00", "\x7f", "\u0085", string([]byte{0xc3}), "a" + string([]byte{0xff}), strings.Repeat("\u0301", MaxClusterBytes), "a" + strings.Repeat("\u0301", MaxClusterBytes)}
	// Exercise the general Unicode path across scripts and Unicode boundaries.
	for r := rune(0x80); r <= utf8.MaxRune; r += 7919 {
		contents = append(contents, string(r), "a"+string(r))
	}
	neighbors := []Cell{{}, {Width: 1}, {Content: " ", Width: 1}, {Content: "x", Width: 0}, {Content: "界", Width: 2}}
	for _, content := range contents {
		for width := -1; width <= 3; width++ {
			for _, neighbor := range neighbors {
				for _, atEnd := range []bool{false, true} {
					cells := []Cell{{Content: content, Width: width}, neighbor}
					if atEnd {
						cells = cells[:1]
					}
					got, want := cellValidationError(validateCell(cells, 0)), cellValidationError(slowValidateCell(cells, 0))
					if got != want {
						t.Fatalf("content=%q width=%d neighbor=%+v end=%t: got %q want %q", content, width, neighbor, atEnd, got, want)
					}
				}
			}
		}
	}
}

func TestValidateCellExpectedOutcomesAndOracleMutants(t *testing.T) {
	cases := []struct {
		name  string
		cells []Cell
		valid bool
	}{
		{"space", []Cell{{Content: " ", Width: 1}}, true},
		{"tilde", []Cell{{Content: "~", Width: 1}}, true},
		{"empty", []Cell{{}}, true},
		{"combining", []Cell{{Content: "e\u0301", Width: 1}}, true},
		{"wide", []Cell{{Content: "界", Width: 2}, {}}, true},
		{"control", []Cell{{Content: "\x1b", Width: 1}}, false},
		{"delete", []Cell{{Content: "\x7f", Width: 1}}, false},
		{"wrong-width", []Cell{{Content: "x", Width: 2}, {}}, false},
		{"continuation-content", []Cell{{Content: "x", Width: 0}}, false},
		{"two-clusters", []Cell{{Content: "xy", Width: 1}}, false},
		{"broken-wide", []Cell{{Content: "界", Width: 2}, {Content: "x", Width: 1}}, false},
		{"unsafe-link", []Cell{{Content: "x", Width: 1, Link: struct{ URL, Params string }{URL: "\x1b]52"}}}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, validate := range []func([]Cell, int) error{validateCell, slowValidateCell} {
				if got := validate(tc.cells, 0) == nil; got != tc.valid {
					t.Fatalf("valid=%t want %t", got, tc.valid)
				}
			}
		})
	}
	// These intentionally incorrect test-only fast paths model three dangerous
	// shortcuts. The independent oracle must reject their false acceptances.
	mutants := []struct {
		name   string
		accept func(Cell) bool
		cell   Cell
	}{
		{"includes-control", func(c Cell) bool { return len(c.Content) == 1 && c.Width == 1 }, Cell{Content: "\x1b", Width: 1}},
		{"ignores-width", func(c Cell) bool { return len(c.Content) == 1 && c.Content[0] >= 0x20 && c.Content[0] <= 0x7e }, Cell{Content: "x", Width: 0}},
		{"skips-link", func(c Cell) bool {
			return len(c.Content) == 1 && c.Content[0] >= 0x20 && c.Content[0] <= 0x7e && c.Width == 1
		}, Cell{Content: "x", Width: 1, Link: struct{ URL, Params string }{URL: "\x1b"}}},
	}
	for _, m := range mutants {
		if !m.accept(m.cell) || slowValidateCell([]Cell{m.cell}, 0) == nil {
			t.Fatalf("oracle failed to distinguish mutant %s", m.name)
		}
	}
}

func BenchmarkFrameValidateASCIIShortcut(b *testing.B) {
	for _, g := range []Geometry{{80, 24}, {240, 80}} {
		for _, mixed := range []bool{false, true} {
			kind := "ascii"
			if mixed {
				kind = "mixed"
			}
			b.Run(fmt.Sprintf("%s/%dx%d", kind, g.Cols, g.Rows), func(b *testing.B) {
				f := testFrame(g.Cols, g.Rows)
				for i := range f.Cells {
					f.Cells[i] = Cell{Content: string(byte(0x20 + i%95)), Width: 1}
				}
				if mixed {
					for y := 0; y < g.Rows; y++ {
						start := y * g.Cols
						f.Cells[start+2] = Cell{Content: "界", Width: 2}
						f.Cells[start+3] = Cell{}
						f.Cells[start+5] = Cell{Content: "e\u0301", Width: 1}
						f.Cells[start+7] = Cell{Content: "👩\u200d💻", Width: 2}
						f.Cells[start+8] = Cell{}
					}
				}
				if err := f.Validate(); err != nil {
					b.Fatal(err)
				}
				b.ReportAllocs()
				b.ResetTimer()
				for n := 0; n < b.N; n++ {
					if err := f.Validate(); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}
