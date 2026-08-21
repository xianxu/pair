package couchcore

import "testing"

func TestNamesAreFreeProse(t *testing.T) {
	// Deliberately NOT validated with launcher.NormalizeTag: that rejects
	// spaces and silently strips a "pair-" prefix, rules that exist for zellij
	// session names rather than human labels.
	n := NewNamingTable().SetName("/repo", "pair refactor thing")
	if got := n.Lookup("refactor"); len(got) != 1 {
		t.Fatalf("Lookup = %v; a name with spaces must be accepted and findable", got)
	}
	n2 := n.SetName("/other", "pair-prefixed")
	if got := n2.Lookup("pair-prefixed"); len(got) != 1 {
		t.Fatalf("a pair- prefix must survive intact, got %v", got)
	}
}

func TestLookupReturnsEveryCandidate(t *testing.T) {
	// Fuzzy in, exact out: duplicates are expected and the caller
	// disambiguates rather than the table guessing.
	n := NewNamingTable().SetName("/a", "pair thing").SetName("/b", "pair thing")
	if got := n.Lookup("pair thing"); len(got) != 2 {
		t.Fatalf("Lookup = %v, want both candidates", got)
	}
}

func TestLookupMatchesDescriptionNotJustName(t *testing.T) {
	n := NewNamingTable().SetDescription("/repo", "reworking the composer gate")
	if got := n.Lookup("composer"); len(got) != 1 {
		t.Fatalf("Lookup = %v; the agent-supplied description must be searchable", got)
	}
}

func TestSetNameDoesNotMutateReceiver(t *testing.T) {
	base := NewNamingTable().SetName("/repo", "one")
	_ = base.SetName("/repo", "two")
	if got := base.Entry("/repo").Name; got != "one" {
		t.Fatalf("receiver mutated to %q", got)
	}
}

func TestLookupIsCaseInsensitive(t *testing.T) {
	n := NewNamingTable().SetName("/repo", "Refactor")
	if got := n.Lookup("refactor"); len(got) != 1 {
		t.Fatalf("Lookup = %v", got)
	}
}

func TestLookupReturnsTheOriginalCasePath(t *testing.T) {
	// The table is keyed on the folded path, but a Worktree must always carry
	// the original case -- it is fed to launcher.ResolveRepoScope, which
	// hashes the raw string, and it is what gets displayed.
	w := Worktree("/Users/x/KBench")
	n := NewNamingTable().SetName(w, "kaggle")
	got := n.Lookup("kaggle")
	if len(got) != 1 {
		t.Fatalf("Lookup = %v", got)
	}
	if got[0] != w {
		t.Fatalf("Lookup returned %q, want the original-case path %q", got[0], w)
	}
}
