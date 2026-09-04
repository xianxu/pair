package couchcore

import "testing"

func TestBuildThreadInventoryKeepsOneRowPerCompositeThreadAtSamePath(t *testing.T) {
	first := metadataThread("816fc349d3faebf8", "couch-0000000000000001", "/repo", "first")
	second := metadataThread("816fc349d3faebf8", "couch-0000000000000002", "/repo", "second")
	second.Incarnations = []ThreadIncarnation{{State: IncarnationLive}}

	rows := BuildThreadInventory([]ThreadRecord{second, first}, nil)
	if len(rows) != 2 || rows[0].Address != first.Address || rows[1].Address != second.Address {
		t.Fatalf("inventory rows = %+v", rows)
	}
	if rows[0].Live() || !rows[1].Live() {
		t.Fatalf("inventory liveness = %v/%v", rows[0].Live(), rows[1].Live())
	}
}

func TestThreadSummaryUsesNameFirstAndOpaqueTagOnlyWhenUnnamed(t *testing.T) {
	named := metadataThread("816fc349d3faebf8", "couch-0000000000000001", "/repo/named", "compiler")
	unnamed := metadataThread("816fc349d3faebf8", "couch-0000000000000002", "/repo/unnamed", "")

	namedSummary := BuildThreadInventory([]ThreadRecord{named}, nil)[0]
	if namedSummary.Label() != "compiler" || namedSummary.Label() == string(named.Address.Tag) {
		t.Fatalf("named label = %q", namedSummary.Label())
	}
	unnamedSummary := BuildThreadInventory([]ThreadRecord{unnamed}, nil)[0]
	if unnamedSummary.Label() != string(unnamed.Address.Tag) {
		t.Fatalf("unnamed label = %q", unnamedSummary.Label())
	}
}

func TestThreadSummaryKeepsOperatorAndPublishedTextDistinct(t *testing.T) {
	record := metadataThread("816fc349d3faebf8", "couch-0000000000000001", "/repo", "compiler")
	row := BuildThreadInventory([]ThreadRecord{record}, nil)[0]
	if row.Description != "operator description" || row.PublishedSummary != "agent summary" || row.DisplaySummary() != "agent summary" {
		t.Fatalf("summary metadata = %+v", row)
	}
}
