package textwidth_test

import (
	"testing"

	"github.com/xianxu/pair/cmd/internal/terminal"
	"github.com/xianxu/pair/cmd/internal/textwidth"
)

func TestGraphemeWidthMatchesChromeCells(t *testing.T) {
	for _, tc := range []struct {
		name, text string
		want       int
	}{
		{"sofa alias", "🛋️echo", 6},
		{"variation selector", "♥️", 2},
		{"joined emoji", "👩‍💻", 2},
		{"flag", "🇺🇸", 2},
		{"modifier", "👍🏽", 2},
		{"combining mark", "e\u0301", 1},
		{"leading zero width", "\u200d\ufe0f界", 2},
		{"only zero width", "\u200d\ufe0f\u0301", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cells, err := terminal.StyledRows(tc.text+"!", 32, 1)
			if err != nil {
				t.Fatal(err)
			}
			end := -1
			for x, cell := range cells {
				if cell.Content == "!" {
					end = x
				}
			}
			if end != tc.want {
				t.Fatalf("chrome marker at %d, want %d", end, tc.want)
			}
			if got := textwidth.Width(tc.text); got != end {
				t.Fatalf("Width(%q)=%d, physical chrome width=%d", tc.text, got, end)
			}
		})
	}
}

// Menu rows repeatedly fit labels that already fit. Those calls must keep a
// prefix of the original text rather than allocate a copy for every field.
func TestFitOrdinaryPrefixesDoNotAllocate(t *testing.T) {
	for _, tc := range []struct {
		text  string
		width int
		want  string
	}{
		{"ordinary label", 30, "ordinary label"},
		{"ordinary label", 8, "ordinary"},
		{"🛋️echo", 30, "🛋️echo"},
		{"👩‍💻work", 2, "👩‍💻"},
	} {
		var got string
		allocs := testing.AllocsPerRun(100, func() { got = textwidth.Fit(tc.text, tc.width) })
		if got != tc.want || allocs != 0 {
			t.Errorf("Fit(%q,%d)=%q with %g allocations; want %q without allocation", tc.text, tc.width, got, allocs, tc.want)
		}
	}
}
