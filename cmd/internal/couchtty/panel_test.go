package couchtty

import (
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func summaries() []couchcore.TreeSummary {
	return []couchcore.TreeSummary{
		{Tree: "/w/brain", Name: "brain", Desc: "the advisor"},
		{Tree: "/w/pair", Name: "pair", Desc: "couch tty switching",
			Actors: []couchcore.ActorView{{Live: true}}},
		{Tree: "/w/ariadne", Desc: "sdlc gates"},
	}
}

// Filter DELEGATES the match rule; it does not restate it. Decision 12: the
// same resolution serves the CLI, the panel and (in #148) the advisor, so a
// second copy here would drift from the one couchcore owns.
func TestPanelFilterUsesTheInjectedResolver(t *testing.T) {
	m := NewPanelModel(summaries())
	called := ""
	resolve := func(q string) []couchcore.Worktree {
		called = q
		return []couchcore.Worktree{"/w/ariadne"}
	}

	rows := m.Filter("anything", resolve)
	if called != "anything" {
		t.Fatalf("the resolver was not consulted (got %q)", called)
	}
	if len(rows) != 1 || rows[0].Tree != "/w/ariadne" {
		t.Fatalf("rows = %+v, want exactly what the resolver named", rows)
	}
}

// An empty query is not a resolution -- it is "show everything", and asking the
// resolver would make the panel's default view depend on a match rule.
func TestPanelFilterWithAnEmptyQueryShowsEverything(t *testing.T) {
	m := NewPanelModel(summaries())
	asked := false
	rows := m.Filter("", func(string) []couchcore.Worktree { asked = true; return nil })
	if asked {
		t.Fatal("an empty query consulted the resolver")
	}
	if len(rows) != 3 {
		t.Fatalf("rows = %d, want all 3", len(rows))
	}
}

// A parked tree -- named, no live actor -- is exactly the thread this project
// exists to stop losing, so it must be listed.
func TestPanelListsParkedTrees(t *testing.T) {
	m := NewPanelModel(summaries())
	rows := m.Filter("", nil)
	for _, r := range rows {
		if r.Tree == "/w/ariadne" {
			if r.Live {
				t.Fatal("a tree with no actors is marked live")
			}
			return
		}
	}
	t.Fatal("the parked tree was omitted")
}

// Numbered selection is only safe if the list does not reorder under the
// operator's fingers.
func TestPanelOrderingIsStable(t *testing.T) {
	m := NewPanelModel(summaries())
	first := m.Filter("", nil)
	for i := 0; i < 5; i++ {
		again := m.Filter("", nil)
		for j := range first {
			if first[j].Tree != again[j].Tree {
				t.Fatalf("row %d moved between refreshes: %q then %q", j, first[j].Tree, again[j].Tree)
			}
		}
	}
}

// Pick indexes the DISPLAYED rows. Picking from the underlying set after a
// filter is the classic off-by-list bug: the operator types 2 and lands on
// something that is not the second thing they can see.
func TestPickIndexesTheFilteredRows(t *testing.T) {
	m := NewPanelModel(summaries())
	rows := m.Filter("x", func(string) []couchcore.Worktree {
		return []couchcore.Worktree{"/w/pair", "/w/ariadne"}
	})
	if len(rows) != 2 {
		t.Fatalf("setup: rows = %d", len(rows))
	}

	got, ok := m.Pick(2)
	if !ok {
		t.Fatal("Pick(2) found nothing among 2 filtered rows")
	}
	if got.Tree != "/w/ariadne" {
		t.Fatalf("Pick(2) = %q, want the second FILTERED row", got.Tree)
	}
}

func TestPickRejectsOutOfRange(t *testing.T) {
	m := NewPanelModel(summaries())
	m.Filter("", nil)
	for _, n := range []int{0, -1, 4, 99} {
		if _, ok := m.Pick(n); ok {
			t.Fatalf("Pick(%d) succeeded against 3 rows", n)
		}
	}
}

// The label is what the operator reads; an unnamed tree must still be
// identifiable rather than showing an empty chip.
func TestPanelRowLabelFallsBackToTheRepo(t *testing.T) {
	m := NewPanelModel(summaries())
	rows := m.Filter("", nil)
	for _, r := range rows {
		if r.Tree == "/w/ariadne" && !strings.Contains(r.Label, "ariadne") {
			t.Fatalf("an unnamed tree rendered as %q", r.Label)
		}
	}
}

// The resolver is free to return matches in any order it likes -- it is a
// lookup, not a view. The panel must impose ITS order, or the numbers under the
// operator's fingers depend on a map iteration somewhere in couchcore.
//
// Found by a deletion check that failed to fire: filtering in the resolver's
// order left every ordering test green, because the fixtures happened to agree.
func TestPanelFilterKeepsTheModelsOrderNotTheResolvers(t *testing.T) {
	m := NewPanelModel(summaries())
	rows := m.Filter("x", func(string) []couchcore.Worktree {
		// Deliberately reversed relative to the model.
		return []couchcore.Worktree{"/w/ariadne", "/w/pair", "/w/brain"}
	})
	want := []couchcore.Worktree{"/w/brain", "/w/pair", "/w/ariadne"}
	for i := range want {
		if rows[i].Tree != want[i] {
			t.Fatalf("row %d = %q, want %q — the panel took the resolver's order",
				i, rows[i].Tree, want[i])
		}
	}
	// And the numbers follow the displayed order.
	if got, _ := m.Pick(1); got.Tree != "/w/brain" {
		t.Fatalf("Pick(1) = %q, want the first DISPLAYED row", got.Tree)
	}
}
