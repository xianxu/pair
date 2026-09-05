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
		// From ABOVE is not enough: the loop so far only rejects a span that is
		// wrong, so losing one entirely passes. Assert from BELOW too -- every
		// actor whose label appears in the drawn row must HAVE a span.
		if len(got.Chips) > len(m.Actors) {
			t.Fatalf("width %d: %d spans for %d actors", width, len(got.Chips), len(m.Actors))
		}
		covered := map[couchcore.ThreadAddress]bool{}
		for _, chip := range got.Chips {
			covered[chip.Thread] = true
		}
		for thread, label := range labels {
			// A label the width dropped draws nothing and needs no span; one
			// that drew even a single character needs one.
			if strings.Contains(plain, label[:1]) && strings.Contains(plain, label) && !covered[thread] {
				t.Fatalf("width %d: %q is drawn in %q but has no span", width, label, plain)
			}
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

// A chip clipped to ONE column still has a span, and it is that one column.
//
// The width sweep above walks widths and asserts what it finds, so a rule that
// drops narrow spans passes it: nothing there requires a span to EXIST at a
// specific width. This constructs the case instead -- a width chosen so the
// second chip draws exactly one column, and an assertion that a click on that
// column lands on it.
func TestAChipClippedToOneColumnIsStillClickable(t *testing.T) {
	m := StatusModel{Actors: []StatusActor{
		{Label: "alpha", Thread: menuAddress("a")},
		{Label: "beta", Thread: menuAddress("b")},
	}}
	// "alpha" is 5, the separator 2, so width 8 leaves exactly one column of
	// "beta".
	row := RenderStatusRow(8, m)
	var narrow *ChipSpan
	for i := range row.Chips {
		if row.Chips[i].Thread == menuAddress("b") {
			narrow = &row.Chips[i]
		}
	}
	if narrow == nil {
		t.Fatalf("a chip that drew has no span: chips=%+v row=%q", row.Chips, row.Body)
	}
	if got := narrow.End - narrow.Start; got != 1 {
		t.Fatalf("clipped span covers %d columns, want exactly the 1 it drew: %+v", got, narrow)
	}
	if thread, ok := row.ColumnToActor(narrow.Start); !ok || thread != menuAddress("b") {
		t.Fatalf("the one drawn column of a clipped chip is not clickable: (%+v,%v)", thread, ok)
	}
}

// RenderedStatusRow's own contract, exercised by name rather than only through
// the values other tests happen to build. Its ColumnToActor is the whole reason
// the type exists: the Body and the Chips describe the SAME drawn row, so a
// column resolved against one must agree with the other.
func TestRenderedStatusRowIsSelfConsistent(t *testing.T) {
	var empty RenderedStatusRow
	if thread, ok := empty.ColumnToActor(0); ok {
		t.Errorf("an empty row resolved column 0 to %+v", thread)
	}
	row := RenderedStatusRow{
		Body: "alpha  beta",
		Chips: []ChipSpan{
			{Thread: menuAddress("a"), Start: 0, End: 5},
			{Thread: menuAddress("b"), Start: 7, End: 11},
		},
	}
	for column, want := range map[int]string{0: "a", 4: "a", 5: "", 6: "", 7: "b", 10: "b", 11: ""} {
		thread, ok := row.ColumnToActor(column)
		if want == "" {
			if ok {
				t.Errorf("column %d resolved to %+v, want nobody", column, thread)
			}
			continue
		}
		if !ok || thread != menuAddress(want) {
			t.Errorf("column %d = (%+v,%v), want %q", column, thread, ok, want)
		}
	}
}
