package textwidth

import "testing"

// The three units this package exists to keep straight: "📁pair" is 5 runes and
// 8 bytes but 6 columns. Padding by either of the other two misaligns the row.
func TestWidthCountsColumnsNotRunesOrBytes(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"pair", 4},
		{"📁pair", 6},        // 5 runes, 8 bytes, 6 columns
		{"Alt+⏎", 5},         // ⏎ U+23CE is NOT in a wide block — narrow
		{"", 0},
		{"日本", 4},           // CJK: two runes, four columns
	}
	for _, c := range cases {
		if got := Width(c.in); got != c.want {
			t.Errorf("Width(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestPadFillsToColumnWidth(t *testing.T) {
	if got := Pad("📁pair", 10); got != "📁pair    " {
		t.Errorf("Pad = %q (len %d), want 4 trailing spaces", got, len(got))
	}
	// Over-wide input is returned unchanged rather than truncated — callers align,
	// they do not clip.
	if got := Pad("pair", 2); got != "pair" {
		t.Errorf("Pad over-wide = %q, want unchanged", got)
	}
}
