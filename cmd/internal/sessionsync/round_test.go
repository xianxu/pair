package sessionsync

import (
	"reflect"
	"testing"
	"time"
)

func TestDeriveRoundSpans(t *testing.T) {
	cases := []struct {
		name, before, after string
		want                []Span
	}{
		{"word in a line", "The quick fox.\n", "The slow fox.\n",
			[]Span{{Start: [2]int{0, 4}, End: [2]int{0, 8}, Old: "quick", New: "slow"}}},
		{"multi-line hunk", "a\nb\nc\nd\n", "a\nB\nC\nd\n",
			[]Span{{Start: [2]int{1, 0}, End: [2]int{2, 1}, Old: "b\nc", New: "B\nC"}}},
		{"deletion", "keep\ndrop\nkeep2\n", "keep\nkeep2\n",
			[]Span{{Start: [2]int{1, 0}, End: [2]int{1, 0}, Old: "drop\n", New: ""}}},
		{"rune-safe narrowing", "🤖a\n", "🦾a\n",
			[]Span{{Start: [2]int{0, 0}, End: [2]int{0, 4}, Old: "🤖", New: "🦾"}}},
		{"no change", "same\n", "same\n", nil},
	}
	for _, c := range cases {
		got, ok := DeriveRound(c.before, c.after, nil)
		if ok != (c.want != nil) || !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: %+v ok=%v, want %+v", c.name, got, ok, c.want)
		}
	}
}

// The baseline is restamped exactly as the write restamps, so a write whose
// only difference is the stamp leaves no span.
func TestDeriveRoundExcludesRestamp(t *testing.T) {
	now := time.Date(2026, 10, 18, 9, 30, 0, 0, time.UTC)
	before := "# Doc\n\nUpdated 2026-10-01 08:00\n\nbody\n"
	after := Restamp(before, now)
	if _, ok := DeriveRound(Restamp(before, now), after, nil); ok {
		t.Fatal("restamp alone formed a round")
	}
	changed := Restamp("# Doc\n\nUpdated 2026-10-01 08:00\n\nbody two\n", now)
	spans, ok := DeriveRound(Restamp(before, now), changed, nil)
	if !ok || len(spans) != 1 || spans[0].New != " two" {
		t.Fatalf("spans=%+v ok=%v, want only the body change", spans, ok)
	}
}

// Adversarial strategy for attachReasons: edits that touch, overlap and miss
// a span, an insertion span touched only at its point, and the same reason
// twice. Each span must carry exactly the reasons of the edits it overlaps.
func TestDeriveRoundAttachesReasonsByOverlap(t *testing.T) {
	base := "one two three\nalpha\ngamma\n"
	records := []EditRecord{
		{Old: "two", Occurrence: 1, New: "2", Explain: "digit"},
		{Old: "three", Occurrence: 1, New: "3", Explain: "digit"},
		{Old: "alpha\n", Occurrence: 1, New: "alpha\nbeta\n", Explain: "insert beta"},
	}
	next, resolved, bad := ResolveEdits(base, records)
	if len(bad) != 0 {
		t.Fatal(bad)
	}
	spans, ok := DeriveRound(base, next, resolved)
	if !ok || len(spans) != 2 {
		t.Fatalf("spans=%+v", spans)
	}
	if spans[0].Explain != "digit" || spans[1].Explain != "insert beta" {
		t.Fatalf("reasons %q, %q", spans[0].Explain, spans[1].Explain)
	}
	// A write (no records) carries no reason.
	spans, _ = DeriveRound(base, next, nil)
	for _, s := range spans {
		if s.Explain != "" {
			t.Fatalf("write span carries a reason: %+v", s)
		}
	}
	// Two distinct reasons on one span join with "; ".
	next, resolved, _ = ResolveEdits("ab\n", []EditRecord{{Old: "a", Occurrence: 1, New: "A", Explain: "x"}, {Old: "b", Occurrence: 1, New: "B", Explain: "y"}})
	spans, _ = DeriveRound("ab\n", next, resolved)
	if len(spans) != 1 || spans[0].Explain != "x; y" {
		t.Fatalf("joined reasons %+v", spans)
	}
}

// BuildEdit is the engine both edit paths share: refusals are the apply
// refusals, and the round matches what Sync.Apply records for the same edit.
func TestBuildEditSharesTheApplyEngine(t *testing.T) {
	now := time.Date(2026, 10, 18, 9, 30, 0, 0, time.UTC)
	base := "# Doc\n\nThe quick fox.\n"
	e, err := BuildEdit(base, []EditRecord{{Old: "quick", Occurrence: 1, New: "slow", Explain: "pace"}}, now, false)
	if err != nil || e.Next != "# Doc\n\nThe slow fox.\n" || !e.Changed {
		t.Fatalf("edit=%+v err=%v", e, err)
	}
	if len(e.Round.Spans) != 1 || e.Round.Spans[0].Explain != "pace" || e.Round.ResultHash != Hash(e.Next) || e.Round.BaseHash != Hash(base) {
		t.Fatalf("round=%+v", e.Round)
	}
	_, err = BuildEdit(base, []EditRecord{{Old: "absent", Occurrence: 1, New: "x"}}, now, false)
	wantRefusal(t, err, ExitRefused, "edits-invalid")
	_, err = BuildEdit(base, nil, now, false)
	wantRefusal(t, err, ExitRefused, "refused")
}
