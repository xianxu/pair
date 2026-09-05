package couchcore

import "testing"

func TestBuildThreadInventoryKeepsOneRowPerCompositeThreadAtSamePath(t *testing.T) {
	first := metadataThread("816fc349d3faebf8", "couch-0000000000000001", "/repo", "first")
	second := metadataThread("816fc349d3faebf8", "couch-0000000000000002", "/repo", "second")
	second.Incarnations = []ThreadIncarnation{{State: IncarnationLive}}

	rows := BuildThreadInventory(ThreadProjectionInput{Records: []ThreadRecord{second, first}})
	if len(rows) != 2 || rows[0].Address != first.Address || rows[1].Address != second.Address {
		t.Fatalf("inventory rows = %+v", rows)
	}
	if rows[0].Live() || !rows[1].Live() {
		t.Fatalf("inventory liveness = %v/%v", rows[0].Live(), rows[1].Live())
	}
}

// An unnamed row now falls back to its DIRECTORY, not its tag: a switcher of
// 16-hex addresses reads as noise. The tag remains the last resort, and
// DisambiguateLabels keeps two rows at one path distinguishable.
func TestThreadSummaryUsesNameFirstThenDirectory(t *testing.T) {
	named := metadataThread("816fc349d3faebf8", "couch-0000000000000001", "/repo/named", "compiler")
	unnamed := metadataThread("816fc349d3faebf8", "couch-0000000000000002", "/repo/unnamed", "")

	namedSummary := BuildThreadInventory(ThreadProjectionInput{Records: []ThreadRecord{named}})[0]
	if namedSummary.Label() != "compiler" || namedSummary.Label() == string(named.Address.Tag) {
		t.Fatalf("named label = %q", namedSummary.Label())
	}
	unnamedSummary := BuildThreadInventory(ThreadProjectionInput{Records: []ThreadRecord{unnamed}})[0]
	if unnamedSummary.Label() != "unnamed" {
		t.Fatalf("unnamed label = %q", unnamedSummary.Label())
	}
}

func TestThreadSummaryKeepsOperatorAndPublishedTextDistinct(t *testing.T) {
	record := metadataThread("816fc349d3faebf8", "couch-0000000000000001", "/repo", "compiler")
	row := BuildThreadInventory(ThreadProjectionInput{Records: []ThreadRecord{record}})[0]
	if row.Description != "operator description" || row.PublishedSummary != "agent summary" || row.DisplaySummary() != "agent summary" {
		t.Fatalf("summary metadata = %+v", row)
	}
}
