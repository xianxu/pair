package query

import (
	"fmt"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func TestDeclaredFactsAnchorBudgetIgnoresUnselectedDeclarations(t *testing.T) {
	b := recording.NewBuilder("priority")
	var kids []recording.Node
	for i := 0; i < 70; i++ {
		kids = append(kids, b.El("div", []string{"data-unrelated", strings.Repeat("u", 1024)}))
	}
	anchor := b.El("button", []string{"id", strings.Repeat("a", 1024), "data-purpose", "original"})
	parent := b.El("section", []string{"id", "anchor-parent"}, anchor)
	kids = append(kids, parent)
	b.Page(recording.SegLive, "https://source.test/", kids...)
	st, err := recording.Reduce(b.Recs, 0)
	if err != nil {
		t.Fatal(err)
	}
	source := captureElementFacts(st)
	// Capturing source declarations must own maps; later DOM edits cannot alter it.
	st.Node(anchor.ID).A["id"] = "later"
	st.Node(anchor.ID).A["data-purpose"] = "later"
	facts := source.facts([]recording.NodeID{anchor.ID}, nil)
	if len(facts.Anchors) != 1 || facts.Anchors[0].ID != strings.Repeat("a", 1024) || facts.Anchors[0].Data["data-purpose"] != "original" {
		t.Fatalf("unselected attributes starved immutable mandatory anchor: %+v", facts.Anchors)
	}
	if len(facts.Ancestors) != 1 || facts.Ancestors[0].ID != "anchor-parent" || facts.Bytes > ElementFactsByteLimit {
		t.Fatalf("selected facts/budget: %+v", facts)
	}
}
func TestDeclaredFactsPreserveEveryLegalMandatoryHandleAtByteLimit(t *testing.T) {
	b := recording.NewBuilder("mandatory")
	var kids []recording.Node
	var anchors []recording.NodeID
	for i := 0; i < ElementFactsCountLimit; i++ {
		attrs := []string{"id", fmt.Sprintf("anchor-%02d", i)}
		for j := 0; j < 16; j++ {
			attrs = append(attrs, fmt.Sprintf("data-%02d", j), strings.Repeat("v", 1024))
		}
		node := b.El("button", attrs)
		kids = append(kids, node)
		anchors = append(anchors, node.ID)
	}
	b.Page(recording.SegLive, "https://source.test/", kids...)
	st, err := recording.Reduce(b.Recs, 0)
	if err != nil {
		t.Fatal(err)
	}
	facts := DeclaredElementFactsAt(st, anchors, nil)
	if len(facts.Anchors) != len(anchors) || facts.Bytes > ElementFactsByteLimit || !facts.Truncated {
		t.Fatalf("mandatory associations not bounded/preserved: count%d bytes%d truncated%v", len(facts.Anchors), facts.Bytes, facts.Truncated)
	}
	seen := map[recording.NodeID]bool{}
	for _, fact := range facts.Anchors {
		seen[fact.Node] = true
	}
	for _, id := range anchors {
		if !seen[id] {
			t.Fatalf("lost mandatory @%d", id)
		}
	}
}
func TestDeclaredFactsAbsentAnchorStillUsesRegion(t *testing.T) {
	b := recording.NewBuilder("region")
	node := b.El("button", []string{"id", "inside-region"})
	b.Page(recording.SegLive, "https://source.test/", node)
	st, err := recording.Reduce(b.Recs, 0)
	if err != nil {
		t.Fatal(err)
	}
	st.Boxes[node.ID] = recording.Box{10, 10, 20, 20}
	region := recording.Box{0, 0, 100, 100}
	facts := DeclaredElementFactsAt(st, []recording.NodeID{0}, &region)
	if len(facts.Anchors) != 0 || len(facts.Intersections) != 1 || facts.Intersections[0].Node != node.ID {
		t.Fatalf("absent anchor fabricated or region discarded: %+v", facts)
	}
}

func TestDeclaredFactCaptureCostAndHeaderBoundsScaleLinearly(t *testing.T) {
	var sources []elementFactSource
	var allocations []float64
	for _, n := range []int{64, 128} {
		b := recording.NewBuilder("fact-cost")
		var kids []recording.Node
		for i := 0; i < n; i++ {
			attrs := []string{"id", fmt.Sprintf("node-%d", i)}
			for j := 19; j >= 0; j-- {
				attrs = append(attrs, fmt.Sprintf("data-%02d", j), "literal")
			}
			kids = append(kids, b.El("div", attrs))
		}
		b.Page(recording.SegLive, "https://source.test/", kids...)
		st, err := recording.Reduce(b.Recs, 0)
		if err != nil {
			t.Fatal(err)
		}
		source := captureElementFacts(st)
		var measured elementFactSource
		count := testing.AllocsPerRun(3, func() { measured = captureElementFacts(st) })
		if measured.declarations != source.declarations {
			t.Fatal("allocation probe changed retained evidence")
		}
		allocations = append(allocations, count)
		t.Logf("nodes=%d inspected_attributes=%d retained_declarations=%d allocations=%.0f", n, source.attributes, source.declarations, count)
		sources = append(sources, source)
		if source.attributes != n*21 || source.declarations != n*17 {
			t.Fatalf("n%d inspection/retained header cost: %d/%d", n, source.attributes, source.declarations)
		}
		for _, node := range kids {
			fact := source.nodes[node.ID].fact
			if len(fact.Data) != 16 || fact.Data["data-00"] != "literal" || fact.Data["data-15"] != "literal" || !fact.Truncated {
				t.Fatalf("n%d nondeterministic or unbounded snapshot: %+v", n, fact)
			}
		}
	}
	if allocations[1] > 2*allocations[0]+32 {
		t.Fatalf("superlinear snapshot allocation growth: %v", allocations)
	}
	if sources[1].attributes != 2*sources[0].attributes || sources[1].declarations != 2*sources[0].declarations || len(sources[1].nodes)-len(sources[0].nodes) != 64 {
		t.Fatal("doubling source declarations did not double bounded snapshot work")
	}
}
func TestDeclaredFactSourceInspectionBoundRemainsExplicit(t *testing.T) {
	b := recording.NewBuilder("source-bound")
	kids := make([]recording.Node, 0, ElementFactsNodeLimit+1)
	for i := 0; i < ElementFactsNodeLimit+1; i++ {
		kids = append(kids, b.El("div", nil))
	}
	b.Page(recording.SegLive, "https://source.test/", kids...)
	st, err := recording.Reduce(b.Recs, 0)
	if err != nil {
		t.Fatal(err)
	}
	source := captureElementFacts(st)
	if source.inspected != ElementFactsNodeLimit || len(source.nodes) > ElementFactsNodeLimit || !source.truncated {
		t.Fatalf("unbounded source snapshot: inspected%d nodes%d truncated%v", source.inspected, len(source.nodes), source.truncated)
	}
	anchor := kids[len(kids)-1].ID
	facts := source.facts([]recording.NodeID{anchor}, nil)
	if len(facts.Anchors) != 1 || facts.Anchors[0].Node != anchor || facts.Anchors[0].Unavailable == "" || !facts.Truncated {
		t.Fatalf("bounded-out mandatory handle lost: %+v", facts)
	}
}
