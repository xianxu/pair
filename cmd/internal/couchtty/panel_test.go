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

func TestPanelTargetJoinKeepsParkedRowsAndAddsRoutingSeparately(t *testing.T) {
	m := NewPanelModel(summaries())
	m.BindTargets([]PanelTarget{
		{Tree: "/w/brain", Target: "child-brain"},
		{Tree: "/w/pair", Target: "child-pair", Bell: true},
	})

	rows := m.Rows()
	if len(rows) != 3 {
		t.Fatalf("rows = %d, want all three summaries", len(rows))
	}
	for _, row := range rows {
		switch row.Tree {
		case "/w/ariadne":
			if row.Target != "" || row.Live {
				t.Fatalf("parked row gained a live target: %+v", row)
			}
		case "/w/pair":
			if row.Target != "child-pair" || !row.Bell {
				t.Fatalf("live target join = %+v", row)
			}
		}
	}
}

// The panel may not grow a private verb. Every action it offers must be one
// couch already declares, so the operator's surface and the advisor's cannot
// drift -- the same audit the CLI has.
func TestPanelActionsAreDeclaredOperations(t *testing.T) {
	declared := map[string]bool{}
	for _, n := range couchcore.OperationNames() {
		declared[n] = true
	}
	for _, a := range PanelActions() {
		if !declared[a] {
			t.Errorf("the panel offers %q, which couch does not declare as an operation", a)
		}
	}
}

// And the panel must actually offer the actions the operator needs from it --
// an empty set would pass the audit above vacuously.
func TestPanelOffersTheOperatorActions(t *testing.T) {
	got := map[string]bool{}
	for _, a := range PanelActions() {
		got[a] = true
	}
	for _, want := range []string{"start", "stop", "name", "describe"} {
		if !got[want] {
			t.Errorf("the panel does not offer %q", want)
		}
	}
}

// Every declared action must be REACHABLE from a keystroke.
//
// A subset check is satisfied by a list that does nothing -- which is exactly
// what shipped: four action names with no dispatch behind them, so the operator
// had no way to start a second child and the audit passed anyway.
func TestEveryPanelActionHasAKey(t *testing.T) {
	keys := PanelActionKeys()
	for _, a := range PanelActions() {
		k, ok := keys[a]
		if !ok {
			t.Errorf("action %q has no key; it is declared but unreachable", a)
			continue
		}
		if len(k) != 2 || k[0] != ':' || k[1] < 0x20 || k[1] >= 0x7f {
			t.Errorf("action %q is not in the ':' command namespace: %q", a, k)
		}
	}
	// And no key may be claimed by two actions.
	seen := map[string]string{}
	for a, k := range keys {
		if prev, dup := seen[k]; dup {
			t.Errorf("key %q is claimed by both %q and %q", k, prev, a)
		}
		seen[k] = a
	}
}
