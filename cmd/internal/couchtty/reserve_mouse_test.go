package couchtty

import (
	"strings"

	"github.com/xianxu/pair/cmd/internal/ansi"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/textwidth"
)

// The spans must come from the pass that CLIPS. A second derivation agrees at
// comfortable widths and disagrees at exactly the narrow ones where clipping
// happens -- which is where a mis-mapped click is least catchable by eye.
func TestChipSpansMatchTheDrawnRowAtEveryWidth(t *testing.T) {
	m := StatusModel{Actors: []StatusActor{
		{Label: "alpha", Thread: menuAddress("a")},
		{Label: "beta", Thread: menuAddress("b"), Active: true},
		{Label: "gamma-with-a-long-name", Thread: menuAddress("g")},
	}}
	labels := map[couchcore.ThreadAddress]string{menuAddress("a"): "alpha", menuAddress("b"): "beta", menuAddress("g"): "gamma-with-a-long-name"}
	for _, width := range []int{80, 40, 24, 16, 12, 6, 3, 1} {
		got := RenderStatusRow(width, m)
		plain := string(ansi.Strip([]byte(got.Body)))
		for _, chip := range got.Chips {
			if chip.Start < 0 || chip.End > width || chip.Start >= chip.End {
				t.Fatalf("width %d: span %+v outside [0,%d)", width, chip, width)
			}
			// The columns the span claims must actually hold that actor's label.
			runes := []rune(plain)
			if chip.End > len(runes) {
				t.Fatalf("width %d: span %+v past the drawn row %q", width, chip, plain)
			}
			// A CLIPPED chip draws a prefix of its label, so the assertion is
			// prefix-ness, not equality: at width 16 "gamma-with-a-long-name"
			// draws as "g", and a span covering "g" is correct rather than
			// broken. Equality here would have failed the code for being right.
			text := string(runes[chip.Start:chip.End])
			label := labels[chip.Thread]
			if !strings.HasPrefix(label, strings.TrimPrefix(strings.TrimSuffix(text, "]"), "[")) {
				t.Fatalf("width %d: span %+v covers %q, which is not a prefix of %q",
					width, chip, text, label)
			}
		}
		// A chip that did not render has NO span.
		if len(got.Chips) > len(m.Actors) {
			t.Fatalf("width %d: %d spans for %d actors", width, len(got.Chips), len(m.Actors))
		}
		if textwidth.Width(plain) > width {
			t.Fatalf("width %d: drew %d columns", width, textwidth.Width(plain))
		}
	}
}

// A column inside a chip is that actor; a gap, or past the end, is nobody.
// "Clicking bare row does nothing" is this returning false, not a branch at the
// call site.
func TestColumnToActorIsTotal(t *testing.T) {
	m := StatusModel{Actors: []StatusActor{
		{Label: "alpha", Thread: menuAddress("a")}, {Label: "beta", Thread: menuAddress("b")},
	}}
	row := RenderStatusRow(40, m)
	if len(row.Chips) != 2 {
		t.Fatalf("chips = %+v, want 2", row.Chips)
	}
	for _, tc := range []struct {
		name   string
		column int
		want   couchcore.ThreadAddress
		ok     bool
	}{
		{"first column of the first chip", row.Chips[0].Start, menuAddress("a"), true},
		{"last column of the first chip", row.Chips[0].End - 1, menuAddress("a"), true},
		{"the gap between chips", row.Chips[0].End, couchcore.ThreadAddress{}, false},
		{"first column of the second chip", row.Chips[1].Start, menuAddress("b"), true},
		{"past the last chip", row.Chips[1].End, couchcore.ThreadAddress{}, false},
		{"negative", -1, couchcore.ThreadAddress{}, false},
		{"far right", 39, couchcore.ThreadAddress{}, false},
	} {
		got, ok := row.ColumnToActor(tc.column)
		if got != tc.want || ok != tc.ok {
			t.Errorf("%s: ColumnToActor(%d) = (%+v,%v), want (%+v,%v)",
				tc.name, tc.column, got, ok, tc.want, tc.ok)
		}
	}
}
