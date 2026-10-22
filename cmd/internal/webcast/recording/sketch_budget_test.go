package recording

import (
	"strings"
	"testing"
)

func TestSketchBudgetExactAdmissionAndTerminalCapacity(t *testing.T) {
	var b SketchBudget
	admit := func(r Record, n int) {
		t.Helper()
		d, err := b.Next(r, n, false)
		if err != nil {
			t.Fatal(err)
		}
		b.Commit(d)
	}
	admit(&SketchBegin{ID: "s"}, 100)
	if b.Used() != 100 || b.Reserved() != SketchTerminalReserve {
		t.Fatalf("initial budget %d %d", b.Used(), b.Reserved())
	}
	admit(&SketchStrokeBegin{Sketch: "s", ID: "stroke"}, 100)
	admit(&SketchPoints{Sketch: "s", ID: "stroke"}, SketchHistoryLimit-b.Used()-b.Reserved())
	beforeUsed, beforeReserved := b.Used(), b.Reserved()
	if _, err := b.Next(&SketchPoints{Sketch: "s", ID: "stroke"}, 1, false); err == nil {
		t.Fatal("accepted over cap")
	}
	if b.Used() != beforeUsed || b.Reserved() != beforeReserved {
		t.Fatal("refusal mutated budget")
	}
	d, err := b.Next(&SketchPoints{Sketch: "s", ID: "stroke"}, 100, true)
	if err != nil {
		t.Fatal(err)
	}
	b.Commit(d)
	if b.Used() != beforeUsed || b.Reserved() != beforeReserved {
		t.Fatal("duplicate charged bytes")
	}
	admit(&SketchStrokeEnd{Sketch: "s", ID: "stroke"}, 100)
	// A repeated end cannot eat the capacity belonging to label or interrupt.
	if _, err := b.Next(&SketchStrokeEnd{Sketch: "s", ID: "stroke"}, SketchTerminalReserve, false); err == nil {
		t.Fatal("repeated end spent label capacity")
	}
	admit(&SketchInterrupt{Sketch: "s", Reason: "expiry"}, 100)
	if b.Reserved() < SketchLabelReserve {
		t.Fatal("interruption lost label capacity")
	}
	admit(&SketchLabel{Sketch: "s", Label: "Saved"}, 100)
	if b.Reserved() != 0 || b.Used() > SketchHistoryLimit {
		t.Fatalf("final budget %d %d", b.Used(), b.Reserved())
	}
}

func TestSketchBudgetStrokeAllowanceReplenishmentAndRetirement(t *testing.T) {
	var b SketchBudget
	commit := func(r Record, n int) {
		t.Helper()
		d, err := b.Next(r, n, false)
		if err != nil {
			t.Fatal(err)
		}
		b.Commit(d)
	}
	commit(&SketchBegin{ID: "s"}, 10)
	commit(&SketchStrokeBegin{Sketch: "s", ID: "a"}, 10)
	commit(&SketchStrokeEnd{Sketch: "s", ID: "a"}, 10)
	first := b.Reserved()
	commit(&SketchStrokeBegin{Sketch: "s", ID: "b"}, 10)
	if b.Reserved() <= first {
		t.Fatal("new stroke did not restore end allowance")
	}
	used := b.Used()
	commit(&AnnotationClear{}, 0)
	if b.Reserved() != 0 || b.Used() != used {
		t.Fatal("clear deleted used history or retained reservation")
	}
	commit(&SketchBegin{ID: "next"}, 10)
	used = b.Used()
	commit(&Stop{Why: StopEnd}, 0)
	if b.Reserved() != 0 || b.Used() != used {
		t.Fatal("end deleted used history or retained reservation")
	}
}

func TestSketchTerminalReserveCoversMaximumEncodedSequence(t *testing.T) {
	id := strings.Repeat("<", 128)
	op := SketchOperation{Operation: id, Sequence: 9007199254740991, Digest: strings.Repeat("a", 64)}
	var total int
	for _, tc := range []struct {
		r       Record
		reserve int
	}{
		{&SketchLabel{Header: Header{T: 9007199254740991}, SketchOperation: op, Sketch: id, Label: strings.Repeat("<", 1024)}, SketchLabelReserve},
		{&SketchStrokeEnd{Header: Header{T: 9007199254740991}, SketchOperation: op, Sketch: id, ID: id, Truncated: true}, SketchEndReserve},
		{&SketchInterrupt{Header: Header{T: 9007199254740991}, SketchOperation: op, Sketch: id, Reason: "navigation"}, SketchInterruptReserve},
	} {
		line, err := Encode(tc.r)
		if err != nil {
			t.Fatal(err)
		}
		n := len(line) + 1
		total += n
		if n > tc.reserve {
			t.Fatalf("%s maximum %d exceeds component %d", tc.r.Kind(), n, tc.reserve)
		}
	}
	if total > SketchTerminalReserve || SketchTerminalReserve < 8192 {
		t.Fatalf("maximum sequence %d reserve %d", total, SketchTerminalReserve)
	}
}

func TestRebuildSketchBudgetUsesAcceptedHistoryOnce(t *testing.T) {
	b := NewBuilder("budget rebuild")
	b.At(2).Page(SegLive, "http://x/")
	begin := &SketchBegin{Header: Header{T: 3}, SketchOperation: SketchOperation{Operation: "begin", Digest: strings.Repeat("a", 64)}, ID: "s", Author: Author{ID: OperatorID, Color: "#123456"}, Owner: OperatorID, Doc: 1, Tab: "tab1", VP: Viewport{W: 100, H: 100, DPR: 1}}
	label := &SketchLabel{Header: Header{T: 4}, SketchOperation: SketchOperation{Operation: "label", Sequence: 1, Digest: strings.Repeat("b", 64)}, Sketch: "s", Label: "Saved"}
	records := append(b.Recs, begin, label)
	budget, err := RebuildSketchBudget(records)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := Encode(begin)
	z, _ := Encode(label)
	if budget.Used() != len(a)+len(z)+2 || budget.Reserved() != 0 {
		t.Fatal("rebuild charged wrong history bytes")
	}
	retry := *label
	retry.SetTime(999)
	records = append(records, &retry)
	withRetry, err := RebuildSketchBudget(records)
	if err != nil {
		t.Fatal(err)
	}
	if withRetry.Used() != budget.Used() {
		t.Fatal("rebuild charged identical retry")
	}
	if _, err := RebuildSketchBudget(append(records, &Unknown{Header: Header{K: "sketch-snapshot"}})); err == nil {
		t.Fatal("rebuild accepted catch-up bootstrap")
	}
}
